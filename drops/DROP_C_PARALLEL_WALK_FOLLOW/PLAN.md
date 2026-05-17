# DROP_C — PARALLEL_WALK_FOLLOW

**State:** planning
**Tier:** A
**Blocked by:** —
**Paths (expected):** internal/fileset/walker.go, internal/fileset/walker_test.go, internal/fileset/file.go (possibly), cmd/rak/root.go, magefile.go (mage addDep for errgroup), README.md
**Packages (expected):** internal/fileset, cmd/rak
**PLAN.md ref:** — (top-level PLAN.md removed at v0.1.0 ship; see memory `session_handoff_2026_05_16_v020_planning.md`)
**Workflow:** main/drops/WORKFLOW.md
**Started:** 2026-05-16
**Closed:** —

## Scope

Two related features bundled into one drop because they both modify walk semantics:

1. **Bounded parallel counting + language detection.** Today rak walks serially and reads each accepted file in order. Switch the per-file `Open() → Peek → Count → Detect` work into a worker pool via `golang.org/x/sync/errgroup` with `Group.SetLimit(N)` for bounded concurrency. **Default `N = runtime.NumCPU()`**; expose `--workers <int>` flag for override (0 = default). Target 2–5× speedup on large repos with cold cache. Race detector stays on for tests (`mage test` already runs `-race`).
2. **`--follow` symlink traversal during walk.** Today rak does NOT follow symlinks during walk (the walker's `fs.WalkDirFunc` callback skips symlinks by default). `--follow` opts into traversal, with cycle detection via `filepath.EvalSymlinks` + a visited-inode set keyed by `(dev, inode)` from `syscall.Stat_t` to short-circuit symlink loops. **macOS + Linux only** for v0.2.0; document Windows non-support if any platform-specific behavior surfaces in cross-platform CI.

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

3. **Channel-feed pattern:** the main goroutine (still serial) iterates
   `source.List(ctx)`, applies the binary-check filter, and pushes accepted
   `*fileset.File` values onto a buffered channel. Worker goroutines in the
   errgroup drain the channel, perform `lang.Detect` → `lang.Split` → `countFile`,
   then acquire the mutex to accumulate. When iteration ends, close the channel;
   errgroup.Wait() collects any worker error. Channel buffer size = `workers`
   (one slot per worker to let the iterator stay ahead).

4. **Context propagation.** Use `errgroup.WithContext(ctx)` so the errgroup-derived
   context is passed to the workers. The first worker error cancels the derived
   context, which propagates to all other workers via their file I/O paths (file
   opens check ctx if the FS supports it) and to the iterator via `ctx.Done()`.

5. **`--max-files` logic.** The accepted-file counter must be thread-safe. The
   main goroutine (which pushes onto the channel) is the only writer of
   `acceptedFiles`; workers do not write it. The main goroutine checks the limit
   before pushing, so no mutex is needed for `acceptedFiles` specifically.

6. **`--workers` flag is added in Unit C.3.** This unit hardcodes `runtime.NumCPU()`
   as the worker count so the pool is functional. Unit C.3 wires the flag and
   threads the value through `runDirectoryOpts → walkAndCount`.

**Worker count passed to SetLimit:**
```
n := runtime.NumCPU()  // replaced by flags.workers in C.3
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
- `TestWalkAndCount_ParallelIdempotent` (new, to be written in `root_test.go`):
  runs `walkAndCount` over a deterministic fixture tree with worker count 1, 2, 4,
  8 and verifies `dirs`, `total`, and `totalByLang` are identical across all runs.
  Uses `t.Parallel()` on the outer test; subtests run in parallel.
- `TestWalkAndCount_RaceDetector` (new): uses `testing/fstest.MapFS` with 20+
  files, workers=4, runs under `-race`; passes.
- The `context.Canceled` early-return path is preserved: when `source.List(ctx)`
  yields `ctx.Err()`, the main goroutine closes the channel and returns the error
  wrapping — workers drain or short-circuit as the errgroup context is cancelled.

---

### Unit C.3 — `--workers` flag in cmd/rak

**Paths:** `cmd/rak/root.go`, `cmd/rak/root_test.go`
**Packages:** `cmd/rak` (main)
**Blocked by:** C.2
**State:** todo

Wire `--workers <int>` flag through `rootFlags → runDirectoryOpts → walkAndCount`.

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
  all parse without error; `--workers -1` triggers the PersistentPreRunE validation
  error.
- `TestOutput_WorkersIdempotent` (new): runs `runDirectory` over the same fixture
  tree with workers=1 and workers=8, captures output via `bytes.Buffer`, asserts
  the two outputs are byte-for-byte identical. This is the falsifiable order-stability
  test — `--sort` runs after all parallel results are collected, so output must
  be deterministic regardless of worker count.

---

### Unit C.4 — `--follow` symlink traversal in internal/fileset/walker.go

**Paths:** `internal/fileset/walker.go`, `internal/fileset/walker_test.go`, `internal/fileset/follow_unix.go` (new), `internal/fileset/follow_windows.go` (new)
**Packages:** `internal/fileset`
**Blocked by:** —
**State:** todo

Add opt-in symlink traversal with cycle detection to `Walker`.

**Design decision (flagged for dev):** `fs.WalkDir` cannot follow symlinks — the
`io/fs` contract does not include symlink traversal. To follow symlinks, the
walker must switch to `filepath.WalkDir` (real OS paths, not `io/fs` paths) when
`Follow` is true. This requires `Walker` to know the real OS root path, not just
the `fs.FS` handle. The recommended implementation:

- Add `Follow bool` to `WalkOptions`.
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
follow). If rejected, fallback is a `NewWalkerFollow(rootPath string, opts)` factory.

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

// follow_windows.go
//go:build windows

type visitedKey struct{ Path string } // fallback: path-based dedup only

func sysStat(path string) (visitedKey, error) {
    return visitedKey{Path: filepath.Clean(path)}, nil
}
```

The visited set is populated per `Walk()` invocation, inside the closure,
not on the `Walker` struct — so concurrent `Walk()` calls are independent.

**`--follow` and GitLister:** when `lister.Detect` returns a `GitLister`, the
`--follow` flag is silently a no-op. Git already enumerates all tracked files
including those reachable through symlinks. The `WalkOptions.Follow` field is
only consulted by `WalkLister`; `GitLister` ignores it. Document this in
`WalkOptions.Follow`'s doc comment.

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

**Changes:**
- Add `follow bool` to `rootFlags`.
- Add the cobra flag: `cmd.Flags().BoolVar(&flags.follow, "follow", false, "follow symbolic links during walk (cycle detection enabled; macOS + Linux only in v0.2.0)"`.
- Pass `flags.follow` into `listerOpts` → `fileset.WalkOptions{Follow: flags.follow, ...}`.
- Pass the real OS root path to `newWalkLister` when `flags.follow` is true:
  `lister.newWalkLister(os.DirFS(absRoot), ".", opts)` becomes
  `lister.newWalkListerFollow(os.DirFS(absRoot), absRoot, ".", opts)` (or via an
  updated `NewWalker` signature per C.4's decision).
- The `Follow` field on `WalkOptions` is already wired through `listerOpts` by this
  point; the only cmd-level change is adding the flag and threading `absRoot` to
  the WalkLister constructor.

**Acceptance:**
- `mage build` passes.
- `mage test` passes with `-race`.
- `TestFlags_Follow` (new): verifies `--follow` parses without error; default is
  false.
- Integration: `TestRunDirectory_FollowSymlink` (new, uses real `t.TempDir()` tree):
  creates a dir with a file + a symlink to a sibling dir containing another file;
  runs `runDirectory` with `--follow`; asserts both files appear in output.
- `TestRunDirectory_FollowDisabled` (new): same fixture, `--follow` false; asserts
  symlink target files do NOT appear (current behavior preserved).

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
  Windows non-support in v0.2.0, embed `docs/follow.gif`.
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
- `rak --help` output includes both new Example lines (verify via `mage run -- --help`
  or equivalent).
- `mage build` and `mage ci` pass (no Go logic changes in this unit).

---

## Notes

**Cross-stream coordination**: Streams B, C, D all add new flags to `cmd/rak/root.go`
and touch `cmd/rak` package. Units C.2, C.3, C.5 are in `cmd/rak`; the orchestrator
must serialize these against any B/D units that also modify `root.go` or
`rootFlags`. Internal-package work (C.4 in `internal/fileset`) is parallel-safe
with all other streams.

**Correctness over speed**: race detector failures are blockers, not "fix later."
Tests must include `t.Parallel()` cases that exercise the worker pool under
contention; `mage test` runs `-race` unconditionally and CI fails on any race
detection. Cycle detection for `--follow` must be tested with both a self-loop
(`a → a`) and an A → B → A loop fixture.

**Order stability**: `--sort` runs after all parallel results are collected in
`walkAndCount`. Output is deterministic regardless of worker execution order.
`TestOutput_WorkersIdempotent` in C.3 is the falsifiable test for this claim.

**Cancellation**: `errgroup.WithContext(ctx)` derives a child context. The first
worker error cancels the derived context. Workers should use the derived context
for their file I/O where possible. The main goroutine's `source.List(ctx)` uses
the original `ctx` (so a user-level cancel from `cmd.Context()` is still
respected). The two contexts are compatible: original cancel → derived cancel;
derived cancel (worker error) does NOT cancel original ctx (no upstream effect).

**`--follow` and GitLister**: when inside a git repo, `lister.Detect` returns a
`GitLister`. `--follow` is a no-op in that mode — git already enumerates all
tracked files including symlink-reachable ones. Document in `WalkOptions.Follow`
doc comment. No error or warning is emitted; the behavior is correct and silent.

**Windows non-support for `--follow`**: inode-based cycle detection (`syscall.Stat_t`
`Dev`+`Ino`) is macOS/Linux only. The `follow_windows.go` stub falls back to
path-based dedup, which catches most cycles but not hard-link loops. v0.2.0
documents this limitation in the `--follow` flag usage string and README. A future
drop can add proper Windows inode support via `windows.GetFileInformationByHandle`.

**Dev decision needed (C.4)**: confirm the `NewWalker` signature change
(`rootPath string` as fourth arg) is acceptable. If not, the fallback is a
separate `NewWalkerFollow(rootPath string, opts WalkOptions) *Walker` factory that
callers opt into. This affects `lister.newWalkLister` and `lister.NewWalkLister`.
Please confirm before C.4 build starts.

**`errgroup` already in go.mod**: `golang.org/x/sync v0.20.0` appears as
`// indirect` in the current `go.mod`. Unit C.1 promotes it to a direct dep.
No fresh network fetch needed if the module cache already has it.
