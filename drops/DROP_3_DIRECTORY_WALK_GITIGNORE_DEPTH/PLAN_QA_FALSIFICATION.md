# DROP_3 — Plan QA Falsification — Round 1

**Agent:** go-qa-falsification-agent
**Target:** `main/drops/DROP_3_DIRECTORY_WALK_GITIGNORE_DEPTH/PLAN.md`
**Round:** 1
**Verdict:** FAIL — Round 2 revise required

## Premises

- Falsification attacks the plan with counterexamples, hidden dependencies, contract mismatches, YAGNI pressure, and memory-rule conflicts.
- An unmitigated counterexample is a blocker; a surface finding is a polish item that should still land before sign-off.
- Mitigations must be evidence-backed (Hylla / `go doc` / Context7 / source), not hand-waved.

## Evidence

- `go doc iter.Seq2` + `go doc iter.Pull2` — yield-function semantics.
- `go doc io/fs.WalkDirFunc` + `go doc io/fs.SkipAll` + `go doc io/fs.SkipDir` — WalkDir control-flow.
- `go doc io/fs.WalkDir` — symlink behavior, per-entry error propagation.
- `go doc testing/fstest.MapFile` — symlink fixture support.
- `main/drops/DROP_3_DIRECTORY_WALK_GITIGNORE_DEPTH/PLAN.md` — the plan under attack.

## Confirmed counterexamples (blockers)

### C1 — `iter.Seq2` yield-false without `fs.SkipAll` panics the walker

**Attack:** PLAN.md:80–87 describes the Walker as `iter.Seq2[*File, error]` wrapping `fs.WalkDir`. The plan does not specify how the `WalkDirFunc` must react when the caller's `yield(...)` returns `false` (i.e. caller did `break` or `return` out of the range).

**Per `go doc iter` ("range-over-func" semantics):** *"Yield panics if called after it returns false."* So a `WalkDirFunc` that does not short-circuit after the first `yield(...) == false` will panic on the very next `yield` call.

**Mechanism required:** once `yield(...)` returns false, the `WalkDirFunc` must return `fs.SkipAll` (or a sentinel the outer iterator recognizes and converts to `SkipAll`) so `fs.WalkDir` stops descending. Returning `nil` is a panic; returning `fs.SkipDir` only skips the current dir.

**Why this matters:** the naive builder reading PLAN.md:80–87 writes `return nil` after yielding and the test `TestWalker_RangeBreak` (PLAN.md:100) panics with "yield called after false". That test is the canary; without the pin, the builder might write the iterator one way, the test another, and the failure mode is a crash rather than a wrong value.

**Mitigation required in Round 2:** Add a new F-pin (e.g. F14) to Unit 3.3 that says verbatim: *"When `yield(...)` returns false, the `WalkDirFunc` MUST return `fs.SkipAll` so `fs.WalkDir` terminates cleanly. Returning `nil` after a false yield invokes `yield` again and panics per `go doc iter`."* Plus a positive test `TestWalker_RangeBreak` with a 3-file tree where `break` happens after the first file, asserting exactly one file was yielded and no panic.

### C2 — `WalkOptions.GitignoreEnabled bool` zero-value contradicts default

**Attack:** PLAN.md:78 defines `WalkOptions.GitignoreEnabled bool` with `false` → gitignore disabled. But PLAN.md:45 + `main/PLAN.md` DROP_3 decision 10 say gitignore is **ON by default**. Any caller writing `WalkOptions{}` silently gets the wrong behavior (gitignore OFF instead of ON).

**Mechanism:** Go struct zero-values are mandatory; there's no "default-to-true" for a bare `bool` field. Users / tests constructing `WalkOptions{Depth: 1}` get `GitignoreEnabled: false` for free, which is the opposite of intent.

**Why this matters:** The walker tests at PLAN.md:89–100 construct `WalkOptions` literals. If half the tests forget to set `GitignoreEnabled: true`, the tests silently exercise the `--no-gitignore` path while claiming to test the gitignore-on path. QA proof cannot catch this — the test passes either way, it's just testing the wrong thing.

**Mitigation required in Round 2:** Flip the field to `DisableGitignore bool` (zero-value false → gitignore ENABLED, matching the default) OR require construction via `NewWalkOptions()` / mandate `NewWalker` accept individual flags. Update PLAN.md § "Unit 3.3" + § "Cross-unit pins" accordingly.

### C3 — `IsHidden` naming drift: package helper vs. method

**Attack:** PLAN.md:61 declares `IsHidden(name string) bool` as a **package-level helper**. PLAN.md:78 says `File.IsHidden` method (via "skips hidden files/dirs via `File.IsHidden`"). PLAN.md:83 says "skip hidden entries when `!IncludeHidden` (via `File.IsHidden`)" — method form again. PLAN.md:94 says `TestWalker_SkipsHidden` exercises hidden-file exclusion but doesn't disambiguate.

**Mechanism:** builder can't implement both forms without duplication, and the walker at PLAN.md:83 needs the call site nailed down. If it's a package helper, the walker calls `fileset.IsHidden(name)` with a `DirEntry.Name()`. If it's a method, the walker has to construct the `*File` first (which implies allocating before the hidden check — the opposite of the performance-conscious approach).

**Why this matters:** without pinning, the builder picks one form and the walker test expects the other; QA proof rejects the inconsistency on Round 1 build-QA.

**Mitigation required in Round 2:** Pick one. Recommendation: **package-level helper** — the hidden check wants to fire on a `DirEntry.Name()` string inside the `WalkDirFunc` before any `*File` allocation. Update PLAN.md:78 + PLAN.md:83 to say `fileset.IsHidden(entry.Name())`. Drop the method form everywhere.

### C4 — F7 symlink behavior is untested (and MapFS DOES support symlinks)

**Attack:** PLAN.md:88 pins F7 as "symlinks yielded, not followed; broken-target errors surface via `Open`". There is no corresponding test in the `TestWalker_*` list (PLAN.md:90–100). An unpinned behavior is an untested behavior.

**MapFS supports symlinks.** Per `go doc testing/fstest.MapFile`: `Mode fs.FileMode` — you set `fs.ModeSymlink` and put the target path in `Data`. Example construction: `fstest.MapFS{"link": {Mode: fs.ModeSymlink, Data: []byte("target.txt")}, "target.txt": {Data: []byte("...")}}`. So the fixture IS buildable in unit tests; there's no excuse to leave F7 untested.

**Why this matters:** F7 is a load-bearing claim that will show up in Drop 8.5's `--follow` work. If Drop 3 doesn't actually test symlink yielding, Drop 8.5 discovers the walker was doing something else all along and we retrofit.

**Mitigation required in Round 2:** Add `TestWalker_SymlinkYielded` to PLAN.md:89–100's test list: MapFS with one regular file + one symlink pointing at it + one symlink to a missing target. Assert the symlinks are yielded (`DirEntry.Type()&fs.ModeSymlink != 0`), and that `File.Open` on the broken one returns a wrapped `fs.ErrNotExist` error.

## Surface findings (non-blocking polish for Round 2)

- **C5** — PLAN.md:112 cites git's `xdiff-interface.c` and ripgrep's `searcher/src/searcher/core.rs` by name. Verify the file paths still exist upstream or reword as "git + ripgrep behavior" without dangling source citations.
- **C6** — Unit 3.1's `Matcher.Match(relPath, isDir) bool` — pin whether `relPath` is separator-agnostic (stdlib `io/fs` uses forward slashes). If Windows ever compiles rak, gitignore patterns MUST match against forward-slash paths.
- **C7** — Unit 3.3's "Enforce `Depth` — count edges from the walk root" — pin that `Depth` counts **directory edges**, not all path segments (so `root/file.txt` is depth 0, `root/sub/file.txt` is depth 1). Off-by-one here will cost a QA round.
- **C8** — Unit 3.5's new `render.Directory` struct conflicts conceptually with the Drop 6.1 `summary.Summary` structure. U3 already notes this, but the plan should explicitly flag the `render.Directory` type as **provisional, migrates to `summary.Summary` in Drop 6.1** so QA proof at Drop 6.1 has a bread-crumb.
- **C9** — Unit 3.5's `RenderTree(w, dirs, total)` adds a method to the existing `Renderer` interface. Go interfaces are implemented implicitly — adding a method breaks any external implementer. Drop 3 has no external implementers (all `internal/`), so this is safe today, but the plan should pin F-level: *"Renderer interface growth is acceptable within `internal/` scope; external implementers do not exist pre-v1.0."*
- **C10** — `TestRootCmd_PathArg_SkipsBinary` (PLAN.md:140) needs an explicit assertion that the error path — `IsBinary()` returning an error — does not silently swallow the file. Spec whether the file is skipped + error logged, or counted + warning, or abort.

## Trace or cases

- C1 trace: caller writes `for f, err := range w.Walk(ctx) { if something { break } }`. First `yield` returns false. `WalkDirFunc` next call is `fn(path, d, err)` → calls `yield(...)` again → panic.
- C2 trace: test writes `NewWalker(mapfs, "root", WalkOptions{Depth: 1})` expecting gitignore ON. Field defaults false. Test passes because there's no `.gitignore` in the fixture, but it was silently testing `--no-gitignore`.
- C3 trace: walker calls `File.IsHidden` → forced to construct `*File` → allocation per entry. Or walker calls `fileset.IsHidden(name)` → cheap string check. Plan allows both; builder picks one; QA rejects.
- C4 trace: Drop 8.5 lands `--follow`. Assumes walker yields symlinks. Walker tested under Drop 3 never exercised symlink path. Bug surfaces at Drop 8.5 QA.

## Conclusion

**Verdict: FAIL (Round 2 revise required).** Four CONFIRMED counterexamples (C1, C2, C3, C4) + six surface findings (C5–C10). C1 is the most load-bearing — an unmitigated C1 makes the walker panic on `break` / early-return from the caller. C2 and C3 cause silent test misbehavior. C4 leaves a pinned contract untested.

## Unknowns

- None on the falsification side — all four blockers are mechanically specifiable. The Round 2 planner revise addresses each with a small surgical edit to the existing unit descriptions and F-pins.
