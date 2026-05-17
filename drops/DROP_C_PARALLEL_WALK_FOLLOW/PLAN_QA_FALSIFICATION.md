# DROP_C — Plan QA Falsification — Round 1

**Date:** 2026-05-16
**Reviewer:** go-qa-falsification-agent
**Verdict:** FAIL — multiple CONFIRMED counterexamples; revisions required before build.

Adversarial review of the Phase 1 Planner section. Findings ordered by severity.

---

## CONFIRMED counterexamples (planner MUST address)

### F1 — Order non-determinism: current sort is UNSTABLE, parallel input order is RANDOMIZED

**Severity:** blocker. Direct refutation of the "Order stability" claim in PLAN.md § Notes.

**Evidence:**

- `internal/summary/sort.go:75` uses `slices.SortFunc`, which is the **unstable** stdlib sort. The stable variant is `slices.SortStableFunc`. Per Context7 / `pkg.go.dev/slices`, only `Sorted` + `SortedStableFunc` carry the "Stable" guarantee; `SortFunc` does not.
- PLAN.md § Notes "Order stability" claims: *"--sort runs after all parallel results are collected in walkAndCount. Output is deterministic regardless of worker execution order."* False under the proposed design.
- `walkAndCount` accumulates into `byDir map[string]counting.Counts` (`cmd/rak/root.go:349`) and then builds the `dirs` slice by ranging that map (`cmd/rak/root.go:461-464`). **Go map iteration order is intentionally randomized.** Today this is masked because per-directory counts converge to a single value before the slice is built, and ties are rare.
- Under workers≥2 the inputs to the unstable sort can differ in input ORDER even when input VALUES are identical: two directories `a/` and `b/` with identical `Lines/Files/Bytes` ("ties") can end up at different positions across runs.

**Repro construction:**

```
fixture/
├── a/x.go   (10 lines)
└── b/y.go   (10 lines)
```

`rak --sort lines --workers 8 fixture/` vs `rak --sort lines --workers 1 fixture/`. Both runs produce two `summary.Directory` entries with `Counts.Lines == 10`. Map iteration randomizes which goes first into the slice; `slices.SortFunc` does not break the tie deterministically. Output bytes differ across runs at the same worker count, and certainly between workers=1 and workers=8.

The proposed `TestOutput_WorkersIdempotent` (C.3 acceptance) would intermittently flake — and would have flaked equally under the current serial design if the test fixture had any ties (serial walk also feeds an unstable sort over a randomized-map-iteration slice).

**Required revisions:**

1. Switch `summary.SortDirs` to `slices.SortStableFunc` OR add a secondary tie-break on `Path` to every comparator. (Stable sort is the smaller, more defensible change.)
2. Make the `dirs` slice construction in `walkAndCount` deterministic: range the map, then `sort.Strings` the keys, then build `dirs` in key order. (This alone fixes the test's order-stability claim under any sort, even the existing unstable one, **as long as ties don't exist** — but the stable sort + sorted keys belt-and-suspenders is the only honest fix.)
3. Add the ties-fixture explicitly to `TestOutput_WorkersIdempotent` to prevent regression. Currently the test as specified is silent on whether the fixture has ties — without ties it is a no-op.

---

### F2 — Channel-feed send is NOT ctx-aware; producer can block forever on cancel

**Severity:** blocker. Concurrency contract.

**Evidence:**

- PLAN.md § Unit C.2 design point 3 says: *"the main goroutine (still serial) iterates `source.List(ctx)`, applies the binary-check filter, and pushes accepted `*fileset.File` values onto a buffered channel."*
- No mention of `select { case ch <- f: case <-derivedCtx.Done(): }`. A bare `ch <- f` is the natural reading.
- Context7 / `pkg.go.dev/errgroup`: `WithContext(ctx)` derives a context that is cancelled on first worker error.

**Counterexample trace** (workers=8, channel buffer=8):

1. Producer fills the 8-slot channel buffer with 8 files.
2. All 8 workers pick up files and start slow I/O (e.g. multi-MB files on cold disk).
3. Producer iterates the 9th file from `source.List(ctx)` and reaches the channel send.
4. Worker 1 hits a `f.Open()` error → returns error → errgroup cancels derived ctx → `derivedCtx.Done()` fires.
5. Workers 2-8 are still in mid-I/O and have not yet returned to the channel-receive loop. They will return (the read completes), then they will check `derivedCtx.Done()` (if they check) and exit.
6. **Producer is blocked on `ch <- file_9`. None of the 8 buffer slots have been drained yet** (workers are still finishing slow I/O for file_1..file_8).
7. When workers 2-8 finally finish their current file and check `derivedCtx`, they exit. Now the channel has 8 undrained slots and no receivers. Producer blocks **forever** on `ch <- file_9` even though ctx is cancelled.
8. `errgroup.Wait()` blocks waiting for the producer (if the producer is also wrapped in `eg.Go`) OR (if the producer is on the main goroutine) the program hangs.

The "force-quit on Ctrl-C" path of fang's `WithNotifySignal` mitigates the user-visible symptom (process eventually dies), but the contract violation — "first error cancels everything cleanly" — is broken.

**Required revision:** the producer's channel send MUST be `select { case ch <- f: case <-derivedCtx.Done(): return derivedCtx.Err() }`. The plan must spell this out explicitly so the builder cannot fall through to the bare-send reading.

---

### F3 — Worker leak on producer panic / mid-iteration crash

**Severity:** high. Plan does not address.

**Evidence:**

- If `source.List(ctx)` panics mid-iteration (e.g. a defensive bug in walker.go, an `os.DirFS` Stat panic on a malformed path, a third-party library panicking), the panic unwinds the producer goroutine.
- The channel is never closed. The 8 worker goroutines block on `<-ch` forever.
- `errgroup.Wait()` blocks forever waiting for workers that will never exit.
- Process hangs until SIGTERM / SIGKILL.

PLAN.md does not mention `defer close(ch)` or `defer recover()` on the producer. Without one or the other, panics turn into hangs.

**Required revision:** explicit design point — *"channel close is deferred at the top of the producer goroutine so that any termination path (normal exit, ctx cancel, panic) closes the channel exactly once. Workers exit cleanly on the closed-channel-empty receive."* If the producer runs on the main goroutine, the defer pattern still applies; the test should include a panic-injection or fault-injection case if practical.

---

### F4 — `--max-files` race: limit can be silently overshot

**Severity:** medium. Counterexample to design point 5.

**Evidence:**

- PLAN.md § Unit C.2 design point 5: *"The main goroutine (which pushes onto the channel) is the only writer of `acceptedFiles`; workers do not write it. The main goroutine checks the limit before pushing, so no mutex is needed for `acceptedFiles` specifically."*
- But `acceptedFiles` is the wrong gate. The current code (`cmd/rak/root.go:438-445`) increments `acceptedFiles` AFTER `countFile` succeeds, AFTER binary-check, AFTER lang-filter. The "accepted" file is one that survived all filters AND the count succeeded.
- Under the parallel design, the producer cannot know whether a file will be accepted until the worker finishes counting it. If the producer counts at push-time, the count overshoots by the number of skipped/errored files. If the producer counts at post-success-time (i.e., it doesn't, the worker does), then `acceptedFiles` is worker-written and the "no mutex needed" claim is wrong.

**Counterexample:** user runs `rak --workers 8 --max-files 100 .` on a tree with 50 binary files mixed into 200 source files. If the producer increments at push-time, it stops pushing after the 100th file pushed — but ~25 of those may be binary and get dropped by workers, so accepted count is ~75, well under the limit, and the user gets fewer results than they asked for. If the producer counts at "I just pushed a non-binary file" time, the binary check is being done on the producer (defeats parallelism — binary check is I/O). If the producer doesn't count at all and workers count, the gating moves to workers, which means workers must check the shared counter under mutex and signal the producer to stop, which is exactly the design the plan tried to avoid.

**Required revision:** the plan must explicitly state WHICH stage (push, count-success, both) increments `acceptedFiles`, justify the choice against the overshoot/undershoot tradeoff, and either (a) accept the semantic drift from the serial behavior, (b) move the binary check into the producer (parallelism cost), or (c) make `acceptedFiles` a worker-written atomic the producer reads on every iteration.

---

### F5 — `runtime.NumCPU()` resolution inside `walkAndCount` makes `--workers 0` user-hostile

**Severity:** low-medium. UX.

**Evidence:**

- PLAN.md § Unit C.3 flag spec: *"Default: `0` (meaning `runtime.NumCPU()` at runtime, resolved inside `walkAndCount`)"*.
- The cobra Example added in C.6 is `rak --workers 8 .`. But the flag default in `--help` reads as `0`. A user reading `--help` cannot tell from the help text alone what `0` means without reading the usage string. The usage string says `"number of parallel file-counting workers (0 = runtime.NumCPU())"` — adequate, but the conventional Go-cli idiom is to print the actual default in help (`(default 8)` on an 8-core machine).
- More important: a user who reads "default 0 = no parallelism" (a reasonable but wrong guess) and writes a script that pins `--workers 0` to "be safe" against an unknown environment will silently get `NumCPU` workers. This is a semantic mismatch against the convention in tools like `xargs -P 0` ("as many as possible") and `make -j` (no arg = unlimited).

**Required revision:** either (a) resolve `0 → NumCPU()` in `runRoot` BEFORE help text is rendered, so the help line reads `(default <NumCPU>)` cleanly, or (b) tighten the usage string to reduce ambiguity, or (c) accept the UX drift and explicitly note in the cobra Example that `--workers 0` means "auto" (or rename the magic value to `-1` for "auto" and reserve `0` for "synchronous").

The plan should at least surface the decision; right now it silently picks (a-without-the-NumCPU-in-help-string).

---

### F6 — `--workers 1` pays full parallel-overhead cost for no benefit

**Severity:** low. YAGNI / perf hygiene.

**Evidence:**

- PLAN.md § Unit C.3 acceptance asserts workers=1 produces identical output to workers=8. The implementation path is the worker-pool code path with `SetLimit(1)`.
- This means workers=1 still pays: 1 goroutine spawn per file via `eg.Go(...)`, 1 channel send + receive per file, mutex acquire/release per file.
- For a small repo (< 100 files) the overhead is invisible. For a directory with 100k files (which `--max-files` defaulting to 0 = no limit implicitly allows), the overhead becomes measurable: ~1µs per file × 100k = 100ms baseline of pure synchronization cost.

**Suggested mitigation:** add a special-case branch `if workers == 1 { return walkAndCountSerial(...) }` that keeps the existing serial code path. Less elegant but faster, and provides a clean A/B for benchmark validation. Surface as a decision; do not silently bake.

---

### F7 — C.4 introduces TWO walker code paths that will diverge

**Severity:** medium. Maintenance hazard.

**Evidence:**

- PLAN.md § Unit C.4: *"When `Walk` is called and `opts.Follow` is true AND `rootPath != ""`: use `filepath.WalkDir(w.rootPath, ...)` with inode cycle detection. When `opts.Follow` is false or `rootPath == ""`: existing `fs.WalkDir` path, unchanged."*
- `fs.WalkDir` (io/fs) and `filepath.WalkDir` (filepath) are sibling APIs but their `WalkDirFunc` callbacks see different things: `fs.WalkDir` passes forward-slash io/fs paths, `filepath.WalkDir` passes OS-native paths. The existing walker is ~300 LOC of carefully-tuned `fs.WalkDir` logic (gitignore handling, hidden filter, depth math, F14 yield-guard).
- The plan proposes duplicating all of that into a `filepath.WalkDir`-based code path. The duplication will drift: a fix to gitignore handling in the `fs.WalkDir` path will not land in the `filepath.WalkDir` path. A change to the depth math in one path will not land in the other.

**Alternatives the plan should consider and explicitly reject:**

1. **Wrap `filepath.WalkDir` in an `fs.FS` adapter** so the existing `fs.WalkDir` body is reused unchanged. (Cost: ~50 LOC adapter, possible perf overhead.)
2. **Defer `--follow` to v0.2.1.** Symlink traversal is a low-frequency feature; the dual-walker maintenance cost may exceed the benefit for v0.2.0. The plan should at minimum surface this as a deferral candidate.
3. **Implement follow as a pre-pass that builds a list of symlink-targets to merge into the walk**, keeping the walker itself unchanged. Brittle but contained.

**Required revision:** name the chosen approach AND name the rejected alternatives with one-line rationale per. Right now the plan presents the divergent-walker approach as the only option, which it is not.

---

### F8 — Plan does not address `--follow` interaction with broken symlinks

**Severity:** medium. Behavior gap.

**Evidence:**

- Current behavior (per `walker.go:113-115` doc comment): *"A broken symlink manifests as a File whose Open call returns an error that unwraps to fs.ErrNotExist."* Walker yields the entry as a regular file; downstream surfaces the error.
- With `--follow`, what does the walker do when it encounters a symlink that points to a non-existent target? `filepath.EvalSymlinks` returns an error on a broken symlink. The plan's pseudocode (`sysStat`) bails with the error. Does the entry get yielded with the error? Silently skipped? Treated as a non-symlink regular entry?
- The plan is silent. The cycle-detection pseudocode does `syscall.Stat(path)` which would fail with `ENOENT` on a broken symlink. The plan does not say what to do with that error.

**Required revision:** explicit `--follow` policy for broken symlinks. Suggested default: yield with the wrapped Stat error so the renderer's error summary surfaces it (consistent with F6 "walker continues past per-entry errors"). Add a `TestWalker_Follow_BrokenSymlink` acceptance test.

---

### F9 — Plan does not address `--follow` interaction with `.gitignore`

**Severity:** medium. Behavior gap.

**Evidence:**

- `internal/fileset/walker.go:234-249` reads `.gitignore` files as the walker descends and applies them via `ignore.Matcher.Match`.
- With `--follow`, a symlink `link → ../other-repo/` might point into a directory whose `.gitignore` is unrelated to the current walk's `.gitignore` chain. Does the walker read `other-repo/.gitignore`? Apply the current walk's accumulated `roots` even though we crossed a tree boundary? Reset gitignore state at the symlink boundary?
- The plan does not say. Each option has different correctness implications: applying the wrong `.gitignore` would surface user-visible file-inclusion / -exclusion bugs.

**Required revision:** explicit `--follow` + `.gitignore` policy. Simplest defensible default: continue accumulating gitignore roots verbatim, treating the symlink target's directory as just another directory in the walk. Add a `TestWalker_Follow_GitignoreCrossesSymlink` acceptance test.

---

### F10 — Plan does not address `--follow` + `--no-gitignore` combination

**Severity:** low. Spec gap.

**Evidence:**

- `--no-gitignore` outside a git repo is the trigger for `lister.Detect` to return `WalkLister` (see `lister/lister.go:50-99`). Inside a git repo, `--no-gitignore` returns `ErrNoGitignoreInRepo`.
- `--follow` is a no-op under `GitLister` (PLAN.md § Unit C.4). It only applies to `WalkLister`.
- Combination matrix:
  - In git repo + `--no-gitignore` + `--follow` → `ErrNoGitignoreInRepo` (no-gitignore rejected before follow even matters; consistent).
  - In git repo + `--follow` (no `--no-gitignore`) → `GitLister`, `--follow` silently no-op (consistent with plan).
  - Outside git repo + `--follow` → `WalkLister` with `Follow: true` (works).
  - Outside git repo + `--follow` + `--no-gitignore` → `WalkLister` with `Follow: true, DisableGitignore: true` — does this actually work? The plan does not address whether the gitignore-accumulation code (walker.go:237) is skipped under `DisableGitignore: true`. Reading walker.go: yes, the `if isDir && !w.opts.DisableGitignore` guard skips ingestion, so this combination should work, but the plan should explicitly confirm.

**Required revision:** add the combination matrix to PLAN.md § Unit C.5 acceptance. Add `TestRunDirectory_FollowNoGitignore` to verify.

---

### F11 — Cycle-detection on Windows path-based dedup may miss directory junctions

**Severity:** low. v0.2.0 documents Windows as unsupported anyway.

**Evidence:**

- PLAN.md § Notes "Windows non-support": *"`follow_windows.go` stub falls back to path-based dedup, which catches most cycles but not hard-link loops."*
- Windows directory junctions (`mklink /J`) and Windows symlinks (`mklink /D`) both create new path names. A loop like `C:\foo\bar` (junction) → `C:\foo` would produce visited keys `C:\foo`, `C:\foo\bar` (visited), then descending `C:\foo\bar` resolves to `C:\foo` — but the visited-key for that descent is `C:\foo\bar\<entry>` (a fresh path), NOT `C:\foo\<entry>`. So path-based dedup does NOT catch this loop. Walker recurses infinitely.
- Plan says "macOS + Linux only for v0.2.0; document Windows non-support". OK — but the `follow_windows.go` STUB is being shipped as part of the build, and if anyone runs `rak --follow .` on Windows it WILL infinite-loop. That's worse than refusing to compile.

**Required revision:** `follow_windows.go` should explicitly reject `Follow: true` with a clear error like `"--follow is not supported on Windows in v0.2.0"` rather than silently fall back to broken path-dedup. CI's cross-platform matrix (if any) should test this rejection.

---

### F12 — `TestOutput_WorkersIdempotent` will produce false-positive PASS if fixture has no ties

**Severity:** medium. Test as specified is silent on the failure mode it must catch.

**Evidence:**

- The whole point of the test (per PLAN.md § Unit C.3 acceptance) is to catch order non-determinism under workers≥2. The order non-determinism specifically arises when the sort produces ties.
- The acceptance text says: *"runs `runDirectory` over the same fixture tree with workers=1 and workers=8, captures output via `bytes.Buffer`, asserts the two outputs are byte-for-byte identical."*
- It does NOT specify the fixture. A fixture with all unique `Lines` counts (the most likely first-draft fixture: 5 files with 1, 2, 3, 4, 5 lines respectively) will sort uniquely under both workers=1 and workers=8 even with an unstable sort, and the test will pass — falsely confirming the order-stability claim. Then a real-world repo with ties breaks the invariant.

**Required revision:** the test MUST use a fixture with intentional ties on every sort key (`--sort lines`: 2+ directories with identical line totals; `--sort files`: 2+ dirs with same file count; `--sort bytes`: 2+ with identical bytes). Run the test under each `--sort` value. The acceptance text should spell this out.

---

### F13 — Cross-stream sequencing in `cmd/rak/root.go` is hand-waved

**Severity:** medium. Coordination risk.

**Evidence:**

- PLAN.md § Notes "Cross-stream coordination": *"Units C.2, C.3, C.5 are in `cmd/rak`; the orchestrator must serialize these against any B/D units that also modify `root.go` or `rootFlags`."*
- This is a statement of intent, not a mechanism. C.2 + C.3 + C.5 internally all touch `rootFlags`, `runDirectoryOpts`, `walkAndCount`. They are already serialized within Stream C (`blocked_by` chain C.1 → C.2 → C.3 → C.5).
- The hand-wave is the inter-stream serialization with B and D. If Stream B adds a flag (say `--files-from`) at the same time, the actual rebase pressure depends on whether B and C touch overlapping lines. The plan does not name which lines / which `rootFlags` field positions are at risk.

**Required revision:** out of scope to enumerate exhaustively across drops, but the plan should at minimum specify a single "register flags in this section of root.go" convention so that B and C and D each add their flag in a self-contained block, minimizing rebase conflict surface. Right now flags are added in a flat sequence with no internal grouping.

---

## Findings (lower-severity, surface only)

### F14 — `errgroup` already in go.mod; C.1 is a 30-second unit

**Severity:** none, just calling out. C.1 is a `mage addDep` invocation that removes one `// indirect` comment. Confirmed via `go.mod:47` (`golang.org/x/sync v0.20.0 // indirect`). The unit is correctly atomic. No counterexample.

### F15 — `runtime.NumCPU()` in containers reports host CPU count

**Severity:** low. v0.2.0 not-a-blocker.

**Evidence:** in a Docker container with `--cpus=2`, `runtime.NumCPU()` returns the host's count (e.g. 8 on a typical dev machine). The CFS-aware variant requires `runtime.GOMAXPROCS` in Go 1.25+ or a third-party lib like `uber-go/automaxprocs`. Not a v0.2.0 concern — surface as a future-improvement candidate, possibly via a `--workers` doc note: *"in container environments, set `--workers` explicitly to your cpu quota; rak does not detect cgroup limits"*.

### F16 — `-race` is necessary but not sufficient

**Severity:** none, hygiene reminder.

**Evidence:** the race detector catches active races during a test run, not absence of synchronization in untested code paths. The proposed `TestWalkAndCount_RaceDetector` (20+ files, workers=4) is a good baseline. Add: a longer-running stress test with random fault injection on a few files would catch more, but is out of scope for v0.2.0.

### F17 — YAGNI candidate: `--follow` for v0.2.0

**Severity:** low. Scope-pressure surfacer.

**Evidence:** PLAN.md memory `session_handoff_2026_05_16_v020_planning.md` lists v0.2.0 scope as "lang expansion + tokens + parallel walk + --files-from piping + smaller items". `--follow` is part of "smaller items" presumably. The cost as drafted: ~150 LOC walker fork + cycle-detection + two new files + two new tests + ~50 LOC cmd wiring + VHS tape + README section + dev-decision sign-off on `NewWalker` signature change. That is not "smaller". If scope pressure mounts late in v0.2.0, `--follow` is the cleanest defer target — it has zero dependencies on the rest of v0.2.0 work, and v0.2.1 is a natural landing zone.

**Recommendation:** orch flag for dev: if v0.2.0 starts running long, defer `--follow` to v0.2.1 and ship `--workers` alone in v0.2.0. The parallel-walk feature stands on its own.

### F18 — Cobra `Example:` formatting consistency

**Severity:** trivial. PLAN.md § Unit C.6 says: *"append to the existing `Example:` string in `newRootCmd`"*. The existing Example block (`cmd/rak/root.go:71-93`) ends with `cat README.md | rak` (no trailing newline shown). Appended Examples must preserve the leading-2-space indent and the blank-line-between-examples convention. Builder should verify.

---

## EXHAUSTED attack families (no counterexample found)

- **Goroutine leak on normal completion** — addressed by errgroup.Wait(), assuming F2 and F3 are fixed.
- **Mutex correctness during accumulation** — design point 2 is sound; critical section is short and well-scoped. No counterexample.
- **`errgroup.WithContext` correctness** — Context7-confirmed; cancel-propagation semantics match the plan's claim.
- **`mage addDep` mechanics** — C.1 is correctly characterized; `mage addDep` deliberately skips `go mod tidy` per main/CLAUDE.md § Dependencies, so the promotion-from-indirect-to-direct sequence is correct.
- **F14 yield-guard interaction with workers** — the F14 yield-guard lives in walker.go, not in walkAndCount. Workers iterate the channel, not the walker directly. F14 invariant is preserved by the producer-only `for f := range source.List(ctx)` iteration; workers never call yield. No counterexample.
- **C.5 `--follow` GitLister no-op** — correct (git already enumerates symlink-reachable tracked files); plan's "silently no-op" is the right call. No counterexample.

---

## Summary

13 CONFIRMED counterexamples (F1–F13) requiring planner revision; 4 lower-severity findings (F14–F18) for awareness. The plan's core architecture (channel-feed errgroup pool, dual walker for `--follow`) is workable, but several load-bearing claims — order stability, cancellation-safety of the producer, `--max-files` semantics under parallelism, broken-symlink handling, gitignore-across-symlink semantics — are either wrong (F1, F2, F12), under-specified (F3, F4, F8, F9, F10), or hand-waved (F5, F6, F7, F11, F13).

**Verdict:** FAIL. Route to planner for Round 2 revision.

## Hylla Feedback

N/A — Hylla is Go-only and the falsification surface was source files plus stdlib doc cross-checks via Context7. No Hylla queries were issued; no fallback was forced.
