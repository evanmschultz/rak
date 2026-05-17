# DROP_C — PARALLEL_WALK_FOLLOW

**State:** planning
**Tier:** A
**Blocked by:** —
**Paths (expected):** internal/fileset/walker.go, internal/fileset/walker_test.go, internal/fileset/follow_unix.go (new), internal/summary/sort.go, internal/summary/sort_test.go, cmd/rak/root.go, cmd/rak/root_test.go, go.mod, go.sum, README.md
**Packages (expected):** internal/fileset, internal/summary, cmd/rak
**PLAN.md ref:** — (top-level PLAN.md removed at v0.1.0 ship; see memory `session_handoff_2026_05_16_v020_planning.md`)
**Workflow:** main/drops/WORKFLOW.md
**Started:** 2026-05-16
**Closed:** —

## Scope

Two related features bundled into one drop because they both modify walk semantics:

1. **Bounded parallel counting + language detection.** Today rak walks serially and reads each accepted file in order. Switch the per-file `Open() → Peek → Count → Detect` work into a worker pool via `golang.org/x/sync/errgroup` with `Group.SetLimit(N)` for bounded concurrency. **Default `N = runtime.NumCPU()`**; expose `--workers <int>` flag for override (0 = default). Target 2–5× speedup on large repos with cold cache. Race detector stays on for tests (`mage test` already runs `-race`).
2. **`--follow` symlink traversal during walk.** Today rak does NOT follow symlinks during walk (the walker's `fs.WalkDirFunc` callback skips symlinks by default). `--follow` opts into traversal, with cycle detection via `filepath.EvalSymlinks` + a visited-inode set keyed by `(dev, inode)` from `syscall.Stat_t` to short-circuit symlink loops. **macOS + Linux only** for v0.2.0; `--follow true` on Windows is a **hard error** returned from `PersistentPreRunE`.

**Feature trio (mandatory per memory `feedback_rak_docs_and_gifs_before_pr.md`):**

1. VHS demos:
   - `main/docs/tapes/parallel.tape` + `main/docs/parallel.gif` — showing speedup on a sizable fixture (use a big public fixture so the gif is honest about wall-clock improvement).
   - `main/docs/tapes/follow.tape` + `main/docs/follow.gif` — showing `rak --follow .` traversing a symlink into a sibling dir.
2. README examples: `rak --workers 8 .` + `rak --follow .` in "Common invocations" + a "Performance" narrative section.
3. Cobra `Example:` entries for both flags in `cmd/rak/root.go`.

## Planner

### Unit C.1 — Promote errgroup to direct dependency

**Paths:** `go.mod`, `go.sum`
**Packages:** (module-level; no Go package edit)
**Blocked by:** —
**State:** todo

`golang.org/x/sync` v0.20.0 is already present in `go.mod` as an `// indirect`
dependency (pulled transitively by an existing dep). Importing
`golang.org/x/sync/errgroup` directly in `cmd/rak/root.go` (Unit C.2) requires
promoting it to a direct entry by running `mage addDep golang.org/x/sync`. This
removes the `// indirect` comment and records the import in the `require` block.

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

Refactor `walkAndCount` in `cmd/rak/root.go` (currently ~120 LOC, lines 348–467)
to process accepted files in parallel using `golang.org/x/sync/errgroup` with a
bounded worker pool.

**Design decisions (builder must follow; do not relitigate):**

1. **Worker pool lives in `walkAndCount`, not in the walker.** The walker
   (`internal/fileset/walker.go`) stays a pure `iter.Seq2[*File, error]` serial
   iterator. Parallelism is a consumption-layer concern. `WalkLister.List` and
   `GitLister.List` are unchanged.

2. **Accumulation via mutex.** Shared mutable state (`byDir`, `byDirLang`,
   `byDirFiles`, `totalByLang`, `total`, `aggErrs`, `acceptedFiles`) is protected
   by a `sync.Mutex` embedded in the refactored function. The worker goroutine
   acquires the mutex only during the accumulation step (after the file is fully
   read, counted, and split), not during I/O. This keeps critical sections short
   and avoids holding the lock during slow disk operations.

3. **Channel-feed pattern with context-aware send:** the main goroutine (still
   serial) iterates `source.List(ctx)`, applies the binary-check filter, and
   pushes accepted `*fileset.File` values onto a buffered channel using a select
   that guards against the derived errgroup context being cancelled:

   ```go
   select {
   case ch <- f:
   case <-egCtx.Done():
       return egCtx.Err()
   }
   ```

   Worker goroutines in the errgroup drain the channel, perform
   `lang.Detect → lang.Split → countFile`, then acquire the mutex to accumulate.

   **Close-then-Wait ordering:** the producer goroutine MUST use
   `defer close(ch)` as its first statement (fires when the goroutine body
   returns, i.e. after all sends are done or after early return). Workers
   consume via `for f := range ch` until the channel is closed. `eg.Wait()`
   is called after the producer goroutine is launched, collecting any worker
   errors. This ordering prevents the producer from blocking forever if workers
   exit early due to context cancellation.

   Channel buffer size = `workers` (one slot per worker so the iterator stays
   slightly ahead). Pinned here to avoid drift with Notes.

4. **Context propagation and two-context discipline.** Use
   `egCtx, eg := errgroup.WithContext(ctx)` so the errgroup-derived context
   (`egCtx`) is passed to the workers. The first worker error cancels `egCtx`,
   which unblocks the producer's select and propagates to worker I/O paths.
   The main goroutine's `source.List(ctx)` uses the ORIGINAL `ctx` (not `egCtx`)
   so a user-level cancel from `cmd.Context()` is still respected independently.
   Original cancel → egCtx cancel (via Go context chain).
   egCtx cancel (worker error) does NOT cancel the original ctx (no upstream effect).

5. **`--max-files` logic.** The accepted-file counter is written by the main
   goroutine only (the goroutine that calls `source.List` and pushes onto the
   channel). Workers do not write `acceptedFiles`. The main goroutine checks the
   limit before pushing, so no mutex is needed for `acceptedFiles` specifically.

6. **`--workers` flag is added in Unit C.3.** This unit hardcodes
   `runtime.NumCPU()` as the worker count so the pool is functional. Unit C.3
   wires the flag and threads the value through `runDirectoryOpts → walkAndCount`.

**Worker count resolution (to be replaced by C.3):**
```go
n := runtime.NumCPU()
if n < 1 {
    n = 1
}
eg.SetLimit(n)
```

**Context7 evidence:** `errgroup.WithContext(ctx)` confirmed; `Group.SetLimit(n)`
confirmed; `Group.Go(f func() error)` blocks when pool is full; `Group.Wait()`
returns first non-nil error. Source: pkg.go.dev/golang.org/x/sync/errgroup.

**Acceptance:**
- `mage build` passes.
- `mage test` (with `-race`) passes. No data races detected.
- Existing `root_test.go` tests (arg/flag parsing, path-arg behavior) continue to
  pass without modification.
- `TestWalkAndCount_ParallelIdempotent` (new, in `root_test.go`): runs
  `walkAndCount` over a deterministic fixture tree with worker count 1, 2, 4, 8
  and verifies `dirs`, `total`, and `totalByLang` are identical across all runs.
  Uses `t.Parallel()` on the outer test; subtests run in parallel.
- `TestWalkAndCount_RaceDetector` (new): uses `testing/fstest.MapFS` with 20+
  files, workers=4, runs under `-race`; passes.
- `TestWalkAndCount_CancelMidStream` (new): fixture with 100 files, workers=4;
  inject a worker error after file 10 (via an errored `*fileset.File` or by
  cancelling a context from a custom source); assert `eg.Wait()` returns within
  5 seconds via `select { case <-done: case <-time.After(5*time.Second): t.Fatal("deadlock") }`.
  This is the direct falsification test for the F2 cancel-deadlock scenario.
  A hang here means the select-with-ctx guard is absent or broken.
- `TestWalkAndCount_MaxFilesUnderWorkers` (new): fixture with 200 source + 50
  binary files, workers=8, `--max-files 100`; assert exactly 100 accepted files
  counted and the function returns the `ErrMaxFilesExceeded` sentinel (or
  equivalent early-exit signal).
- `TestWalkAndCount_MultipleWorkerErrors` (new): inject 3 simultaneous worker
  errors; assert all 3 errors are collected in `aggErrs`, no panic, no lost errors.
  Verifies the mutex-protected accumulation of worker errors.
- The `context.Canceled` early-return path is preserved: when `source.List(ctx)`
  yields `ctx.Err()`, the main goroutine closes the channel and returns the
  error — workers drain or short-circuit via the errgroup context.

---

### Unit C.2.5 — Stable sort fix in internal/summary

**Paths:** `internal/summary/sort.go`, `internal/summary/sort_test.go`
**Packages:** `internal/summary`
**Blocked by:** C.1
**State:** todo

`SortDirs` in `internal/summary/sort.go` currently uses `slices.SortFunc` (line 75),
which is UNSTABLE per Go stdlib docs. Map iteration order over the `dirs` accumulator
in `walkAndCount` is random, and an unstable sort produces different byte output
for the same input whenever two directory entries tie on the sort key. This will
surface as non-deterministic output across `--workers` counts on any fixture with
sort-key ties.

**Change (one line):** replace `slices.SortFunc` with `slices.SortStableFunc` in
`SortDirs`. No API change; the function signature and behavior are unchanged except
that equal elements preserve their original relative order. `slices.SortStableFunc`
is available since Go 1.21; rak requires Go 1.26+.

Also update the doc comment on `SortDirs` to say `slices.SortStableFunc` instead
of `slices.SortFunc`.

**Test (add to `sort_test.go`):** `TestSortDirs_StableTieBreak` — a tie-rich
fixture with at least 3 `Directory` values that share identical `Lines`, `Files`,
AND `Bytes` counts (e.g. three dirs each with Lines=100, Files=5, Bytes=2048).
Sort by each of the four keys in turn; assert the output order is exactly the
input order for tied elements (i.e. stable). This test would silently pass on a
tie-free fixture; the tie-richness is mandatory.

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

Wire `--workers <int>` flag through `rootFlags → runDirectoryOpts → walkAndCount`.
C.3 is blocked by both C.2 (needs the pool implementation) AND C.2.5 (needs the
stable sort fix so `TestOutput_WorkersIdempotent` is a meaningful test rather than
a false pass on a tie-free fixture).

**Flag spec:**
- Name: `--workers`
- Type: `int`
- Default: `0` (meaning `runtime.NumCPU()` at runtime, resolved inside `walkAndCount`)
- Usage: `"number of parallel file-counting workers (0 = runtime.NumCPU())"`
- Validation: reject negative values in `PersistentPreRunE` with a user-facing
  error: `"--workers must be 0 or positive"`.

**Changes:**
- Add `workers int` to `rootFlags`.
- Add `workers int` to `runDirectoryOpts`.
- Wire `flags.workers → opts.workers` in `runRoot`.
- Pass `opts.workers` into `walkAndCount`; resolve `0 → runtime.NumCPU()` there.
- No changes to `internal/fileset`, `internal/lister`, or other packages.

**Acceptance:**
- `mage build` passes.
- `mage test` passes with `-race`.
- `TestFlags_Workers` (new): verifies `--workers 0`, `--workers 1`, `--workers 8`
  all parse without error; `--workers -1` triggers the `PersistentPreRunE` validation
  error.
- `TestOutput_WorkersIdempotent` (new): runs `runDirectory` over the same fixture
  tree with workers=1 and workers=8, captures output via `bytes.Buffer`, asserts
  the two outputs are byte-for-byte identical. **The fixture MUST be intentionally
  tie-rich** — at least 3 dirs with identical `lines`, `files`, AND `bytes` counts —
  so this test actually catches unstable-sort regression. A tie-free fixture would
  make this test silently vacuous; the QA agents will check this.

---

### Unit C.4 — `--follow` symlink traversal in internal/fileset/walker.go

**Paths:** `internal/fileset/walker.go`, `internal/fileset/walker_test.go`, `internal/fileset/follow_unix.go` (new)
**Packages:** `internal/fileset`
**Blocked by:** —
**State:** todo

Add opt-in symlink traversal with cycle detection to `Walker`. Windows support is
NOT included in this unit — `--follow` on Windows is a hard error enforced in
`PersistentPreRunE` (Unit C.5). This unit ships `follow_unix.go` only; no
`follow_windows.go`.

**Design decision (flagged for dev):** `fs.WalkDir` cannot follow symlinks — the
`io/fs` contract does not include symlink traversal. To follow symlinks, the
walker must switch to `filepath.WalkDir` (real OS paths, not `io/fs` paths) when
`Follow` is true. This requires `Walker` to know the real OS root path, not just
the `fs.FS` handle. The recommended implementation:

- Add `Follow bool` to `WalkOptions`. (New field; not yet in tree.)
- Add `rootPath string` to `Walker` struct (the real OS path, populated only when
  `Follow` is true and the FS is an `os.DirFS`-backed tree).
- Modify `NewWalker` to accept `rootPath string` as a fourth argument (empty string
  = no follow capability, preserving backward compatibility for `fstest.MapFS`
  callers in tests).
- When `Walk` is called and `opts.Follow` is true AND `rootPath != ""`: use
  `filepath.WalkDir(w.rootPath, ...)` with inode cycle detection (see below).
- When `opts.Follow` is false or `rootPath == ""`: existing `fs.WalkDir` path,
  unchanged.

**Alternative considered:** implement follow logic entirely in `cmd/rak/root.go`
as a pre-pass. Rejected: the walker is the correct abstraction owner for walk
semantics; placing follow logic in `cmd` breaks the layering (cmd should not know
about `filepath.WalkDir`).

**Dev decision needed:** confirm the `rootPath string` addition to `NewWalker`
signature is acceptable. Callers: `lister.newWalkLister` (passes `absRoot`),
`lister.NewWalkLister` (passes empty string — test callers using MapFS don't
follow). If rejected, fallback is a `NewWalkerFollow(rootPath string, opts WalkOptions) *Walker`
factory. This decision gates C.4 build start.

**Inode cycle detection (macOS + Linux only):**

```go
// follow_unix.go
//go:build !windows

type visitedKey struct{ Dev, Ino uint64 }

func sysStat(path string) (visitedKey, error) {
    var s syscall.Stat_t
    if err := syscall.Stat(path, &s); err != nil {
        return visitedKey{}, err
    }
    return visitedKey{Dev: uint64(s.Dev), Ino: uint64(s.Ino)}, nil
}
```

The visited set is populated per `Walk()` invocation, inside the closure, not on
the `Walker` struct — so concurrent `Walk()` calls are independent.

**`--follow` and GitLister:** when `lister.Detect` returns a `GitLister`, the
`--follow` flag is silently a no-op. Git already enumerates all tracked files
including those reachable through symlinks. The `WalkOptions.Follow` field is
only consulted by `WalkLister`; `GitLister` ignores it. Document this in
`WalkOptions.Follow`'s doc comment.

**`.gitignore` + `--follow`:** gitignore matching applies to the resolved real
path after symlink resolution. Document in `WalkOptions.Follow`'s doc comment.

**Broken symlinks:** yield the broken symlink as an error (same policy as existing
broken-symlink-as-file behavior in v0.1.4). Do not silently skip.

**Filter replication (critical correctness requirement):** when `opts.Follow` is
true and the `filepath.WalkDir` path is active, ALL existing filter semantics must
be preserved identically: hidden-file pruning, depth limit, gitignore matching,
include/exclude glob application, and yield-false handling (the `fs.SkipAll` carry-
over from Drop 3 F14). The two code paths (`fs.WalkDir` and `filepath.WalkDir`)
must apply the same filter logic; no silent filter omissions in the follow path.

**Acceptance:**
- `mage build` passes.
- `mage test` passes with `-race`.
- `TestWalker_Follow_SelfLoop` (new, real OS fixtures via `t.TempDir()`): creates
  a self-referential symlink (`dir/link → dir/`); verifies walk terminates and
  files within `dir/` are yielded exactly once.
- `TestWalker_Follow_ABCycle` (new): creates `a/link → b/`, `b/link → a/`; verifies
  walk terminates and files in both directories are yielded exactly once each.
- `TestWalker_Follow_ReachableFiles` (new): creates a regular dir + a symlink to a
  sibling dir with real files; verifies files in the symlink target are yielded
  (i.e., traversal occurred).
- `TestWalker_Follow_BrokenSymlink` (new): creates a symlink pointing to a
  non-existent target; verifies walk yields an error (not a silent skip).
- `TestWalker_Follow_FilterReplication` (new): parallel table that runs every
  existing walker filter scenario (hidden, depth, gitignore, include, exclude,
  yield-false) under both `Follow: false` (current) and `Follow: true` (new) and
  asserts identical accepted-file sets. This is the falsification test for filter-
  replication correctness.
- `TestWalker_SymlinkYielded` (existing) continues to pass — that test uses MapFS
  with `Follow: false` (default), unchanged behavior.
- No data races under `-race`.

---

### Unit C.5 — Wire `--follow` flag into cmd/rak

**Paths:** `cmd/rak/root.go`, `cmd/rak/root_test.go`
**Packages:** `cmd/rak` (main)
**Blocked by:** C.3, C.4
**State:** todo

Wire `--follow` flag through `rootFlags → WalkOptions → listerOpts → lister.Detect`.

**Windows hard error:** on Windows, `--follow true` must return a hard error from
`PersistentPreRunE` with the message:
`"--follow is not supported on windows in v0.2.0"`. The `follow_unix.go` inode-
detection file is `!windows`-tagged; there is no `follow_windows.go` stub. The
`PersistentPreRunE` guard is the sole Windows gating point.

**Changes:**
- Add `follow bool` to `rootFlags`.
- Add the cobra flag: `cmd.Flags().BoolVar(&flags.follow, "follow", false, "follow symbolic links during walk (cycle detection enabled; macOS + Linux only in v0.2.0)")`.
- Add `PersistentPreRunE` check: if `flags.follow && runtime.GOOS == "windows"`,
  return `fmt.Errorf("--follow is not supported on windows in v0.2.0")`.
- Pass `flags.follow` into `listerOpts` → `fileset.WalkOptions{Follow: flags.follow, ...}`.
- Pass the real OS root path to the WalkLister constructor when `flags.follow` is
  true (per the `NewWalker` signature decision from C.4).
- `--follow` when lister is `SingleFileLister`: a single file has no directory to
  follow into; the flag is a no-op (no error, no warning). `SingleFileLister` is
  selected by `lister.Detect` when the walk root resolves to a regular file.

**Acceptance:**
- `mage build` passes.
- `mage test` passes with `-race`.
- `TestFlags_Follow` (new): verifies `--follow` parses without error; default is
  false.
- `TestFlags_FollowWindowsError` (new): unit test that calls the `PersistentPreRunE`
  validation logic directly with `flags.follow=true` and `runtime.GOOS` mocked or
  replaced by a `currentGOOS` variable. The test uses a `runtime.GOOS == "windows"`
  guard at the TEST level so it compiles and runs on all platforms; on non-Windows
  CI it verifies the error message returned when GOOS is forced to "windows" via
  the injectable variable. This avoids build-tag-conditional tests that CI (Linux)
  would never exercise. Exact implementation: extract the Windows check into a
  small helper `checkFollowPlatform(goos string) error` that takes GOOS as an
  argument; `TestFlags_FollowWindowsError` calls `checkFollowPlatform("windows")`
  and asserts the error is non-nil with the expected message.
- `TestRunDirectory_FollowSymlink` (new, uses real `t.TempDir()` tree): creates a
  dir with a file + a symlink to a sibling dir containing another file; runs
  `runDirectory` with `--follow`; asserts both files appear in output.
- `TestRunDirectory_FollowDisabled` (new): same fixture, `--follow` false; asserts
  symlink target files do NOT appear (current behavior preserved).
- `TestRunDirectory_FollowOnSingleFile` (new): passes a single regular file path
  as the walk root with `--follow true`; asserts normal output (no error, no crash,
  file counts as expected). Verifies that `SingleFileLister` + `--follow` is a
  no-op.

---

### Unit C.6 — Feature trio: gif + README + cobra Examples for --workers and --follow

**Paths:** `docs/tapes/parallel.tape` (new), `docs/parallel.gif` (new), `docs/tapes/follow.tape` (new), `docs/follow.gif` (new), `README.md`, `cmd/rak/root.go`
**Packages:** (docs only; root.go cobra Example field only — no logic change)
**Blocked by:** C.5
**State:** todo

Ship the mandatory feature trio per memory `feedback_rak_docs_and_gifs_before_pr.md`.

**VHS tapes:**
- `docs/tapes/parallel.tape`: demonstrates `rak --workers 8 <large-dir>` vs
  `rak <large-dir>` side-by-side or sequentially, showing wall-clock speedup.
  Use `main/` itself or a real-world fixture large enough for the gif to be honest.
- `docs/tapes/follow.tape`: demonstrates `rak --follow .` traversing through a
  symlink into a sibling directory. The symlink setup is scripted in the tape
  (mkdir / ln -s / rak --follow).

**README additions (two new subsections under a `## Performance` heading and
`## Symlink traversal` heading):**
- `## Performance` — brief narrative explaining `--workers`, default behavior
  (`runtime.NumCPU()`), `--workers 1` for serial equivalent, embed
  `docs/parallel.gif`.
- `## Symlink traversal` — brief narrative explaining `--follow`, cycle detection,
  Windows hard error in v0.2.0 (not "path-based fallback" — a hard error), embed
  `docs/follow.gif`.
- Add both flags to `## Common invocations` with one-liner examples:
  `rak --workers 8 .` and `rak --follow .`.

**Cobra `Example:` entries:** append to the existing `Example:` string in `newRootCmd`:
```
  # Parallel counting with 8 workers
  rak --workers 8 .

  # Follow symbolic links (macOS + Linux; cycle detection enabled)
  rak --follow .
```

**Acceptance:**
- `docs/parallel.gif` and `docs/follow.gif` exist and render in a browser/markdown viewer.
- `docs/tapes/parallel.tape` and `docs/tapes/follow.tape` exist and are syntactically
  valid VHS tapes (builder verifies by running `vhs --dry-run docs/tapes/parallel.tape`
  and `vhs --dry-run docs/tapes/follow.tape`).
- README contains both new sections with embedded gifs.
- README `## Symlink traversal` section accurately states Windows returns a hard
  error (not "path-based dedup stub").
- `rak --help` output includes both new Example lines (verify via `mage run -- --help`
  or equivalent).
- `mage build` and `mage ci` pass (no Go logic changes in this unit).

---

## Notes

**Cross-stream coordination**: Streams B, C, D all add new flags to `cmd/rak/root.go`
and touch `cmd/rak` package. Units C.2, C.3, C.5 are in `cmd/rak`; the orchestrator
must serialize these against any B/D units that also modify `root.go` or
`rootFlags`. Internal-package work (C.2.5 in `internal/summary`, C.4 in
`internal/fileset`) is parallel-safe with all other streams.

**Stream ordering (dev confirmed, B → C → D)**: Stream C's `cmd/rak/root.go` units
(C.3 `--workers` and C.5 `--follow`) land AFTER Stream B's B.5. Stream D's D.2
rebases against this. The flag-registration block ordering in `root.go`:
`--tokens` / `--tokens-encoding` (B) → `--workers` / `--follow` (C) →
`--files-from` (D). `PersistentPreRunE` checks chained in this order.

**Correctness over speed**: race detector failures are blockers, not "fix later."
Tests must include `t.Parallel()` cases that exercise the worker pool under
contention; `mage test` runs `-race` unconditionally and CI fails on any race
detection. Cycle detection for `--follow` must be tested with both a self-loop
(`a → a`) and an A → B → A loop fixture.

**Order stability and stable sort**: `--sort` runs after all parallel results are
collected in `walkAndCount`. Output is deterministic regardless of worker execution
order, BUT only because `SortDirs` uses `slices.SortStableFunc` (Unit C.2.5).
Without stable sort, map randomization in the `dirs` accumulator + `slices.SortFunc`
= non-deterministic output on fixtures with sort-key ties. `TestOutput_WorkersIdempotent`
in C.3 is the falsifiable test for this claim; its fixture must be tie-rich.

**Cancellation and the two-context model**: `errgroup.WithContext(ctx)` derives
`egCtx`. Workers use `egCtx`; the first worker error cancels `egCtx`, unblocking
the producer's `select { case ch <- f: case <-egCtx.Done(): }`. The producer's
`source.List` uses the original `ctx`, so user-level cancellation (via
`cmd.Context()`) also terminates iteration independently. The two cancellation
signals are compatible: original cancel propagates to `egCtx` (via Go context
parent chain); `egCtx` cancel does NOT propagate upstream. This prevents a single
worker error from corrupting the user's context.

**`--follow` and GitLister**: when inside a git repo, `lister.Detect` returns a
`GitLister`. `--follow` is a no-op in that mode — git already enumerates all
tracked files including symlink-reachable ones. Document in `WalkOptions.Follow`
doc comment. No error or warning is emitted; the behavior is correct and silent.

**Windows non-support for `--follow`**: `--follow true` on Windows returns a hard
error from `PersistentPreRunE`:
`"--follow is not supported on windows in v0.2.0"`.
There is no path-based dedup fallback; the flag is blocked entirely. The check
is extracted into `checkFollowPlatform(goos string) error` for testability. A
future drop can add Windows inode support via `windows.GetFileInformationByHandle`.

**`SingleFileLister` and `--follow`**: `--follow` is a no-op when `lister.Detect`
selects `SingleFileLister` (root is a regular file — nothing to walk into). No
error, no warning. Covered by `TestRunDirectory_FollowOnSingleFile` in C.5.

**`--workers 0` UX**: cobra `--help` will show `(default 0)`. The flag description
string says `"(0 = runtime.NumCPU())"` explicitly to avoid xargs-style "unlimited"
confusion. Users who want the serial equivalent use `--workers 1`.

**`--workers 1` overhead**: single-worker execution stays on the parallel code
path (goroutine + channel + mutex). The minor overhead is accepted for code-path
uniformity. No separate serial fast-path.

**`--follow` + `--no-gitignore`**: orthogonal flags. `--no-gitignore` controls
gitignore filtering; `--follow` controls symlink traversal. Both can be set
simultaneously without conflict.

**Dual walker divergence**: the follow path uses `filepath.WalkDir` while the
default path uses `fs.WalkDir`. The ~300 LOC of filter duplication between the
two paths is intentional for v0.2.0 — filter logic must be replicated into the
follow path rather than sharing an abstraction. v0.2.1 could collapse the two
paths via an FS adapter over `filepath.WalkDir`; that is out of scope now.

**Broken symlinks under `--follow`**: yield with an error (same as existing
behavior for other per-entry errors). Do not silently skip. Covered by
`TestWalker_Follow_BrokenSymlink` in C.4.

**Channel buffer cap**: pinned at `workers` (one slot per worker so the iterator
stays slightly ahead of consumption). This is specified in the C.2 design text
and must not be changed without updating both locations.

**`-race` adequacy**: the race detector is necessary but not sufficient. Tests must
also explicitly exercise the constructed deadlock and race traces (cancel-midstream,
multiple-worker-errors, parallel-idempotent). `-race` catches data races on shared
memory; the cancel-deadlock is a liveness bug that `-race` will not catch. The
explicit tests are the deadlock detector.

**NumCPU in containers**: `runtime.NumCPU()` reports the host CPU count, not the
cgroup quota. CGroup-aware NumCPU is `go.uber.org/automaxprocs` territory — deferred
to v0.2.1+. Not in scope for this drop.

**`errgroup` already in go.mod**: `golang.org/x/sync v0.20.0` appears as
`// indirect` in the current `go.mod`. Unit C.1 promotes it to a direct dep.
No fresh network fetch needed if the module cache already has it.

**Dev decision needed (C.4)**: confirm the `NewWalker` signature change
(`rootPath string` as fourth arg) is acceptable. If not, the fallback is a
separate `NewWalkerFollow(rootPath string, opts WalkOptions) *Walker` factory that
callers opt into. This affects `lister.newWalkLister` and `lister.NewWalkLister`.
**Builder must not start C.4 without this confirmation.**
