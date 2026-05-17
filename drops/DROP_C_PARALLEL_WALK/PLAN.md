# DROP_C — PARALLEL_WALK

**State:** planning
**Tier:** A
**Blocked by:** —
**Paths (expected):** internal/summary/sort.go, internal/summary/sort_test.go, cmd/rak/root.go, cmd/rak/root_test.go, go.mod, go.sum, README.md, docs/tapes/parallel.tape (new), docs/parallel.gif (new)
**Packages (expected):** internal/summary, cmd/rak
**PLAN.md ref:** — (top-level PLAN.md removed at v0.1.0 ship; see memory `session_handoff_2026_05_16_v020_planning.md`)
**Workflow:** main/drops/WORKFLOW.md
**Started:** 2026-05-16
**Closed:** —

## Scope

**Bounded parallel counting + language detection.** Today rak walks serially and reads each accepted file in order. Switch the per-file `Open() → Peek → Count → Detect` work into a worker pool via `golang.org/x/sync/errgroup` with `Group.SetLimit(N)` for bounded concurrency. **Default `N = runtime.NumCPU()`**; expose `--workers <int>` flag for override (`0 = runtime.NumCPU()`). Target 2–5× speedup on large repos with cold cache. Race detector stays on for tests (`mage test` already runs `-race`).

**`--follow` symlink traversal is OUT of v0.2.0 scope** (cut 2026-05-16 per memory `feedback_yagni_strict.md`). Stream D's `--files-from -` covers the symlink case via Unix composition: `find -L . -type f | rak --files-from -`. If a real user asks for native `--follow` support, it can land in v0.2.1+ as a dedicated drop with proper Windows support.

**Feature trio (mandatory per memory `feedback_rak_docs_and_gifs_before_pr.md`):**

1. VHS demo: `main/docs/tapes/parallel.tape` + `main/docs/parallel.gif` — showing speedup on a sizable fixture (use a big public fixture so the gif is honest about wall-clock improvement).
2. README example: `rak --workers 8 .` in "Common invocations" + a "Performance" narrative section.
3. Cobra `Example:` entry for `--workers` in `cmd/rak/root.go`.

## Planner

### Unit C.1 — Promote errgroup to direct dependency

**Paths:** `go.mod`, `go.sum`
**Packages:** (module-level; no Go package edit)
**Blocked by:** —
**State:** todo

`golang.org/x/sync` v0.20.0 is already present in `go.mod` as an `// indirect` dependency (pulled transitively by an existing dep). Importing `golang.org/x/sync/errgroup` directly in `cmd/rak/root.go` (Unit C.2) requires promoting it to a direct entry by running `mage addDep golang.org/x/sync`. This removes the `// indirect` comment and records the import in the `require` block.

**Acceptance:**
- `go.mod` `require` block contains `golang.org/x/sync` without `// indirect`.
- `mage build` passes.
- No other source files are modified by this unit.

---

### Unit C.2 — Parallel per-file counting via errgroup worker pool in walkAndCount

**Paths:** `cmd/rak/root.go`, `cmd/rak/root_test.go`
**Packages:** `cmd/rak` (main)
**Blocked by:** C.1
**State:** todo

Refactor `walkAndCount` in `cmd/rak/root.go` (currently ~120 LOC, lines 348–467) to process accepted files in parallel using `golang.org/x/sync/errgroup` with a bounded worker pool.

**Design decisions (builder must follow; do not relitigate):**

1. **Worker pool lives in `walkAndCount`, not in the walker.** The walker (`internal/fileset/walker.go`) stays a pure `iter.Seq2[*File, error]` serial iterator. Parallelism is a consumption-layer concern. `WalkLister.List` and `GitLister.List` are unchanged. `internal/fileset/walker.go` is NOT touched in this drop.

2. **Accumulation via mutex.** Shared mutable state (`byDir`, `byDirLang`, `byDirFiles`, `totalByLang`, `total`, `aggErrs`, `acceptedFiles`) is protected by a `sync.Mutex` embedded in the refactored function. The worker goroutine acquires the mutex only during the accumulation step (after the file is fully read, counted, and split), not during I/O. This keeps critical sections short and avoids holding the lock during slow disk operations.

3. **Channel-feed pattern with context-aware send:** the main goroutine (still serial) iterates `source.List(ctx)`, applies the binary-check filter, and pushes accepted `*fileset.File` values onto a buffered channel using a select that guards against the derived errgroup context being cancelled:

   ```go
   select {
   case ch <- f:
   case <-egCtx.Done():
       return egCtx.Err()
   }
   ```

   Worker goroutines in the errgroup drain the channel, perform `lang.Detect → lang.Split → countFile`, then acquire the mutex to accumulate.

   **Close-then-Wait ordering:** the producer goroutine MUST use `defer close(ch)` as its first statement (fires when the goroutine body returns, i.e. after all sends are done or after early return). Workers consume via `for f := range ch` until the channel is closed. `eg.Wait()` is called after the producer goroutine is launched, collecting any worker errors. This ordering prevents the producer from blocking forever if workers exit early due to context cancellation.

   Channel buffer size = `workers` (one slot per worker so the iterator stays slightly ahead). Pinned here to avoid drift with Notes.

4. **Context propagation and two-context discipline.** Use `egCtx, eg := errgroup.WithContext(ctx)` so the errgroup-derived context (`egCtx`) is passed to the workers. The first worker error cancels `egCtx`, which unblocks the producer's select and propagates to worker I/O paths. The main goroutine's `source.List(ctx)` uses the ORIGINAL `ctx` (not `egCtx`) so a user-level cancel from `cmd.Context()` is still respected independently. Original cancel → egCtx cancel (via Go context chain). egCtx cancel (worker error) does NOT cancel the original ctx (no upstream effect).

5. **`--max-files` logic.** The accepted-file counter is written by the main goroutine only (the goroutine that calls `source.List` and pushes onto the channel). Workers do not write `acceptedFiles`. The main goroutine checks the limit before pushing, so no mutex is needed for `acceptedFiles` specifically.

6. **`--workers` flag is added in Unit C.3.** This unit hardcodes `runtime.NumCPU()` as the worker count so the pool is functional. Unit C.3 wires the flag and threads the value through `runDirectoryOpts → walkAndCount`.

**Worker count resolution (to be replaced by C.3):**
```go
n := runtime.NumCPU()
if n < 1 {
    n = 1
}
eg.SetLimit(n)
```

**Context7 evidence:** `errgroup.WithContext(ctx)` confirmed; `Group.SetLimit(n)` confirmed; `Group.Go(f func() error)` blocks when pool is full; `Group.Wait()` returns first non-nil error. Source: pkg.go.dev/golang.org/x/sync/errgroup.

**Acceptance:**
- `mage build` passes.
- `mage test` (with `-race`) passes. No data races detected.
- Existing `root_test.go` tests (arg/flag parsing, path-arg behavior) continue to pass without modification.
- `TestWalkAndCount_ParallelIdempotent` (new, in `root_test.go`): runs `walkAndCount` over a deterministic fixture tree with worker count 1, 2, 4, 8 and verifies `dirs`, `total`, and `totalByLang` are identical across all runs. Uses `t.Parallel()` on the outer test; subtests run in parallel.
- `TestWalkAndCount_RaceDetector` (new): uses `testing/fstest.MapFS` with 20+ files, workers=4, runs under `-race`; passes.
- `TestWalkAndCount_CancelMidStream` (new): fixture with 100 files, workers=4; inject a worker error after file 10 (via an errored `*fileset.File` or by cancelling a context from a custom source); assert `eg.Wait()` returns within 5 seconds via `select { case <-done: case <-time.After(5*time.Second): t.Fatal("deadlock") }`. This is the direct falsification test for the cancel-deadlock scenario. A hang here means the select-with-ctx guard is absent or broken.
- `TestWalkAndCount_MaxFilesUnderWorkers` (new): fixture with 200 source + 50 binary files, workers=8, `--max-files 100`; assert exactly 100 accepted files counted and the function returns the `ErrMaxFilesExceeded` sentinel (or equivalent early-exit signal).
- `TestWalkAndCount_MultipleWorkerErrors` (new): inject 3 simultaneous worker errors; assert all 3 errors are collected in `aggErrs`, no panic, no lost errors. Verifies the mutex-protected accumulation of worker errors.
- The `context.Canceled` early-return path is preserved: when `source.List(ctx)` yields `ctx.Err()`, the main goroutine closes the channel and returns the error — workers drain or short-circuit via the errgroup context.

---

### Unit C.2.5 — Stable sort fix in internal/summary

**Paths:** `internal/summary/sort.go`, `internal/summary/sort_test.go`
**Packages:** `internal/summary`
**Blocked by:** C.1
**State:** todo

`SortDirs` in `internal/summary/sort.go` currently uses `slices.SortFunc` (line 75), which is UNSTABLE per Go stdlib docs. Map iteration order over the `dirs` accumulator in `walkAndCount` is random, and an unstable sort produces different byte output for the same input whenever two directory entries tie on the sort key. This will surface as non-deterministic output across `--workers` counts on any fixture with sort-key ties.

**Change (one line):** replace `slices.SortFunc` with `slices.SortStableFunc` in `SortDirs`. No API change; the function signature and behavior are unchanged except that equal elements preserve their original relative order. `slices.SortStableFunc` is available since Go 1.21; rak requires Go 1.26+.

Also update the doc comment on `SortDirs` to say `slices.SortStableFunc` instead of `slices.SortFunc`.

**Test (add to `sort_test.go`):** `TestSortDirs_StableTieBreak` — a tie-rich fixture with at least 3 `Directory` values that share identical `Lines`, `Files`, AND `Bytes` counts (e.g. three dirs each with Lines=100, Files=5, Bytes=2048). Sort by each of the four keys in turn; assert the output order is exactly the input order for tied elements (i.e. stable). This test would silently pass on a tie-free fixture; the tie-richness is mandatory.

**Acceptance:**
- `slices.SortFunc` does NOT appear in `SortDirs` after this unit; `slices.SortStableFunc` does.
- `TestSortDirs_StableTieBreak` passes under `mage test`.
- All existing `sort_test.go` tests continue to pass.
- `mage build` passes.

---

### Unit C.3 — `--workers` flag in cmd/rak

**Paths:** `cmd/rak/root.go`, `cmd/rak/root_test.go`
**Packages:** `cmd/rak` (main)
**Blocked by:** C.2, C.2.5
**State:** todo

Wire `--workers <int>` flag through `rootFlags → runDirectoryOpts → walkAndCount`. C.3 is blocked by both C.2 (needs the pool implementation) AND C.2.5 (needs the stable sort fix so `TestOutput_WorkersIdempotent` is a meaningful test rather than a false pass on a tie-free fixture).

**Flag spec:**
- Name: `--workers`
- Type: `int`
- Default: `0` (meaning `runtime.NumCPU()` at runtime, resolved inside `walkAndCount`)
- Usage: `"number of parallel file-counting workers (0 = runtime.NumCPU())"`
- Validation: reject negative values in `PersistentPreRunE` with a user-facing error: `"--workers must be 0 or positive"`.

**Changes:**
- Add `workers int` to `rootFlags`.
- Add `workers int` to `runDirectoryOpts`.
- Wire `flags.workers → opts.workers` in `runRoot`.
- Pass `opts.workers` into `walkAndCount`; resolve `0 → runtime.NumCPU()` there.
- No changes to `internal/fileset`, `internal/lister`, or other packages.

**Acceptance:**
- `mage build` passes.
- `mage test` passes with `-race`.
- `TestFlags_Workers` (new): verifies `--workers 0`, `--workers 1`, `--workers 8` all parse without error; `--workers -1` triggers the `PersistentPreRunE` validation error.
- `TestOutput_WorkersIdempotent` (new): runs `runDirectory` over the same fixture tree with workers=1 and workers=8, captures output via `bytes.Buffer`, asserts the two outputs are byte-for-byte identical. **The fixture MUST be intentionally tie-rich** — at least 3 dirs with identical `lines`, `files`, AND `bytes` counts — so this test actually catches unstable-sort regression. A tie-free fixture would make this test silently vacuous; the QA agents will check this.

---

### Unit C.4 — Feature trio: gif + README + cobra Example for --workers

**Paths:** `docs/tapes/parallel.tape` (new), `docs/parallel.gif` (new), `README.md`, `cmd/rak/root.go`
**Packages:** (docs only; root.go cobra Example field only — no logic change)
**Blocked by:** C.3
**State:** todo

Ship the mandatory feature trio per memory `feedback_rak_docs_and_gifs_before_pr.md`.

**VHS tape:**
- `docs/tapes/parallel.tape`: demonstrates `rak --workers 8 <large-dir>` vs `rak <large-dir>` side-by-side or sequentially, showing wall-clock speedup. Use `main/` itself or a real-world fixture large enough for the gif to be honest.

**README additions (one new subsection under a `## Performance` heading):**
- `## Performance` — brief narrative explaining `--workers`, default behavior (`runtime.NumCPU()`), `--workers 1` for serial equivalent, embed `docs/parallel.gif`.
- Add `--workers` to `## Common invocations` with a one-liner example: `rak --workers 8 .`.

**Cobra `Example:` entry:** append to the existing `Example:` string in `newRootCmd`:
```
  # Parallel counting with 8 workers
  rak --workers 8 .
```

**Acceptance:**
- `docs/parallel.gif` exists and renders in a browser/markdown viewer.
- `docs/tapes/parallel.tape` exists and is a syntactically valid VHS tape (builder verifies by running `vhs --dry-run docs/tapes/parallel.tape`).
- README contains the new `## Performance` section with embedded gif.
- `rak --help` output includes the new Example line (verify via `mage run -- --help`).
- `mage build` and `mage ci` pass (no Go logic changes in this unit).

---

## Notes

**Symlink traversal (cut from v0.2.0)**: `--follow` was scoped in early Round 1/2 planning and cut on 2026-05-16 per memory `feedback_yagni_strict.md`. Reason: Stream D's `--files-from -` covers the symlink use case via Unix composition (`find -L . -type f | rak --files-from -`), making a native `--follow` flag redundant ~300 LOC. If a real user asks for native `--follow` support, it can land in v0.2.1+ as a dedicated drop with proper Windows support (current rak has no Windows-friendly cycle-detection mechanism).

**Cross-stream coordination**: Streams B, C, D all add new flags to `cmd/rak/root.go` and touch `cmd/rak` package. Units C.2, C.3 are in `cmd/rak`; the orchestrator must serialize these against any B/D units that also modify `root.go` or `rootFlags`. Internal-package work (C.2.5 in `internal/summary`) is parallel-safe with all other streams.

**Stream ordering (dev confirmed, B → C → D)**: Stream C's `cmd/rak/root.go` units (C.2, C.3) land AFTER Stream B's B.5. Stream D's D.2 rebases against this. The flag-registration block ordering in `root.go`: `--tokens` / `--tokens-encoding` (B) → `--workers` (C) → `--files-from` (D). `PersistentPreRunE` checks chained in this order.

**Correctness over speed**: race detector failures are blockers, not "fix later." Tests must include `t.Parallel()` cases that exercise the worker pool under contention; `mage test` runs `-race` unconditionally and CI fails on any race detection.

**Order stability and stable sort**: `--sort` runs after all parallel results are collected in `walkAndCount`. Output is deterministic regardless of worker execution order, BUT only because `SortDirs` uses `slices.SortStableFunc` (Unit C.2.5). Without stable sort, map randomization in the `dirs` accumulator + `slices.SortFunc` = non-deterministic output on fixtures with sort-key ties. `TestOutput_WorkersIdempotent` in C.3 is the falsifiable test for this claim; its fixture must be tie-rich.

**Cancellation and the two-context model**: `errgroup.WithContext(ctx)` derives `egCtx`. Workers use `egCtx`; the first worker error cancels `egCtx`, unblocking the producer's `select { case ch <- f: case <-egCtx.Done(): }`. The producer's `source.List` uses the original `ctx`, so user-level cancellation (via `cmd.Context()`) also terminates iteration independently. The two cancellation signals are compatible: original cancel propagates to `egCtx` (via Go context parent chain); `egCtx` cancel does NOT propagate upstream. This prevents a single worker error from corrupting the user's context.

**`--workers 0` UX**: cobra `--help` will show `(default 0)`. The flag description string says `"(0 = runtime.NumCPU())"` explicitly to avoid xargs-style "unlimited" confusion. Users who want the serial equivalent use `--workers 1`.

**`--workers 1` overhead**: single-worker execution stays on the parallel code path (goroutine + channel + mutex). The minor overhead is accepted for code-path uniformity. No separate serial fast-path.

**Channel buffer cap**: pinned at `workers` (one slot per worker so the iterator stays slightly ahead of consumption). This is specified in the C.2 design text and must not be changed without updating both locations.

**`-race` adequacy**: the race detector is necessary but not sufficient. Tests must also explicitly exercise the constructed deadlock and race traces (cancel-midstream, multiple-worker-errors, parallel-idempotent). `-race` catches data races on shared memory; the cancel-deadlock is a liveness bug that `-race` will not catch. The explicit tests are the deadlock detector.

**NumCPU in containers**: `runtime.NumCPU()` reports the host CPU count, not the cgroup quota. CGroup-aware NumCPU is `go.uber.org/automaxprocs` territory — deferred to v0.2.1+. Not in scope for this drop.

**`errgroup` already in go.mod**: `golang.org/x/sync v0.20.0` appears as `// indirect` in the current `go.mod`. Unit C.1 promotes it to a direct dep. No fresh network fetch needed if the module cache already has it.
