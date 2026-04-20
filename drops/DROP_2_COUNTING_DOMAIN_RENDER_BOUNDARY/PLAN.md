# DROP_2 — COUNTING_DOMAIN_RENDER_BOUNDARY

**State:** planning
**Blocked by:** —
**Paths (expected):** `main/cmd/rak/root.go` (lift `count` out), `main/internal/counting/` (new package), `main/internal/render/` (new package), `main/cmd/rak/testdata/` (integration fixture), plus per-package `*_test.go` files
**Packages (expected):** `github.com/evanmschultz/rak/cmd/rak`, `github.com/evanmschultz/rak/internal/counting` (new), `github.com/evanmschultz/rak/internal/render` (new)
**PLAN.md ref:** main/PLAN.md → `DROP_2_COUNTING_DOMAIN_RENDER_BOUNDARY` row
**Workflow:** main/drops/WORKFLOW.md
**Started:** 2026-04-19
**Closed:** —

## Scope

Lift the `count(io.Reader) (Counts, error)` primitive out of `cmd/rak/root.go` into a first-class `internal/counting` package with an exported `Count` function and `Counts` struct (bytes/lines/words/chars). Land `internal/render` as the laslig-backed rendering boundary with `NewHumanRenderer` / `NewJSONRenderer` constructors (no Format enum factory per decision 27(d)) and `Format{Human,JSON}` plumbing. Wire the root command to the new counting + render layer and auto-select renderer via laslig's TTY-vs-pipe detection. Ship counting table tests and render snapshot tests. **No walker, no language detection, no tokens, no summary rollup yet** — all deferred to later drops. Expected decomposition: ~5 units (2.1 counting / 2.2 render / 2.3 wire-up / 2.4 TTY-auto / 2.5 tests) per main/PLAN.md § "Expected Decomposition" lines 107–113.

## Planner

<Filled by go-planning-agent in Phase 1. Atomic units of work below. Each unit's state is mutated in place by the builder during Phase 4. See main/drops/WORKFLOW.md § "Phase 1 — Plan" for deliverable rules.>

## Notes

<Optional. Cross-unit decisions, library choices made during planning, deferrals to later drops.>
