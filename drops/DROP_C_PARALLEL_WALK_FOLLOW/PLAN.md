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

<Filled by go-planning-agent in Phase 1.>

## Notes

**Cross-stream coordination**: Streams B, C, D all add new flags to `cmd/rak/root.go`. The planner should make the cmd/rak flag-wiring unit explicit so the orchestrator can serialize it against B and D at build time. Internal-package work (`internal/fileset/*`) is parallel-safe with the other streams.

**Correctness over speed**: race detector failures are blockers, not "fix later." Tests must include `t.Parallel()` cases that exercise the worker pool under contention; `mage test` runs `-race` unconditionally and CI fails on any race detection. Cycle detection for `--follow` must be tested with both a self-loop (`a → a`) and an A → B → A loop fixture.
