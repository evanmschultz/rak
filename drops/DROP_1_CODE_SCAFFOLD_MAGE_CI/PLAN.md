# DROP_1 — CODE_SCAFFOLD_MAGE_CI

**State:** planning
**Blocked by:** —
**Paths (expected):** `main/go.mod`, `main/go.sum`, `main/cmd/rak/main.go`, `main/cmd/rak/root.go`, `main/magefile.go`, `main/.github/workflows/ci.yml`, `/tmp/rak-stash/*` (source for move)
**Packages (expected):** `github.com/evanmschultz/rak/cmd/rak` (only package with Go code after Drop 1; `internal/*` packages land from Drop 2 onward)
**PLAN.md ref:** main/PLAN.md → `DROP_1_CODE_SCAFFOLD_MAGE_CI` row
**Workflow:** main/drops/WORKFLOW.md
**Started:** 2026-04-18
**Closed:** —

## Scope

Move the stashed `fwc` prototype at `/tmp/rak-stash/` into the rak layout under `main/`, rewrite the module path to `github.com/evanmschultz/rak`, split the flat `main.go` into `cmd/rak/main.go` (fang entry) + `cmd/rak/root.go` (cobra root), rewrite the root command for rak's shape (`rak [path]`, `MaximumNArgs(1)`, drop wc-style flags) with fang signal-to-context wiring, add `github.com/magefile/mage` dep, land `magefile.go` with the 9 canonical targets, and ship `.github/workflows/ci.yml` running `mage ci`. **No `internal/*` packages yet — `count(io.Reader)` stays unexported in `cmd/rak/root.go` for Drop 2.1 to lift into `internal/counting`.** Expected decomposition: 6 units (1.1–1.6) per main/PLAN.md.

## Planner

<Filled by go-planning-agent in Phase 1. Atomic units of work below. Each unit's state is mutated in place by the builder during Phase 4. See main/drops/WORKFLOW.md § "Phase 1 — Plan" for deliverable rules.>

## Notes

<Optional. Cross-unit decisions, library choices made during planning, deferrals to later drops.>
