# DROP_3 — DIRECTORY_WALK_GITIGNORE_DEPTH

**State:** planning
**Blocked by:** DROP_2
**Paths (expected):** `main/internal/fileset/` (new package — `File` type, `Walker`), `main/internal/ignore/` (new package — gitignore + include/exclude globs), `main/cmd/rak/root.go` (wire `len(args)==1` path case into walker), `main/cmd/rak/root_test.go` (extend) or `main/cmd/rak/integration_test.go` (extend fixture tree), `main/cmd/rak/testdata/` (may grow a real directory fixture), plus per-package `*_test.go` files
**Packages (expected):** `github.com/evanmschultz/rak/internal/fileset` (new), `github.com/evanmschultz/rak/internal/ignore` (new), `github.com/evanmschultz/rak/cmd/rak`
**PLAN.md ref:** main/PLAN.md → `DROP_3_DIRECTORY_WALK_GITIGNORE_DEPTH` row
**Workflow:** main/drops/WORKFLOW.md
**Started:** 2026-04-21
**Closed:** —

## Scope

Land the directory-walking spine behind `rak [path]`: `internal/fileset` exposes the `File` type (with `Open() (io.ReadCloser, error)` + `Peek(n int) ([]byte, error)` per decision 25) and a `Walker` that emits `iter.Seq2[*File, error]` over a tree. `internal/ignore` unifies `.gitignore` parsing and `--include` / `--exclude` glob matchers behind a single `Matcher` interface. Binary file detection via `File.Peek(512)` skips binaries by default (adds `--binary` escape hatch). Root command's `len(args)==1` path case — which Unit 2.3 rejected with a "walker lands in Drop 3" error — now walks the directory, counts every text file, and renders per-directory aggregates through the existing `internal/render` boundary. **No language detection, no code-aware splits, no stdin changes, no token counting, no parallelism yet** — all deferred to later drops. Expected decomposition: 4 units (3.1 fileset / 3.2 ignore / 3.3 binary detection / 3.4 root wiring + per-dir aggregation).

## Planner

<Filled by go-planning-agent in Phase 1. Atomic units of work below. Each unit's state is mutated in place by the builder during Phase 4. See main/drops/WORKFLOW.md § "Phase 1 — Plan" for deliverable rules.>

## Notes

<Planner fills.>
