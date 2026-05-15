# DROP_4 — DEFAULT_BEHAVIOR_TRACKED_TOON

**State:** planning
**Tier:** A
**Blocked by:** DROP_3
**Paths (expected):** `main/go.mod`, `main/go.sum` (dep add), `main/internal/lister/` (new package — `FileLister` interface + `GitLister` + `WalkLister` + `Detect` factory + tests), `main/internal/render/render.go` (interface unchanged), `main/internal/render/toon.go` (new), `main/internal/render/render_test.go` (extend snapshot tests), `main/cmd/rak/root.go` (rewire `runDirectory` from direct `fileset.Walker` to `lister.Detect`; replace `--format` flag with bool `--human` / `--json` / `--toon`), `main/cmd/rak/root_test.go` (update flag-parsing cases), `main/cmd/rak/integration_test.go` (extend for tracked-only default behavior + TOON output snapshot)
**Packages (expected):** `github.com/evanmschultz/rak/internal/lister` (new), `github.com/evanmschultz/rak/internal/render`, `github.com/evanmschultz/rak/cmd/rak`
**PLAN.md ref:** main/PLAN.md → `DROP_4_DEFAULT_BEHAVIOR_TRACKED_TOON` row
**Workflow:** main/drops/WORKFLOW.md
**Started:** 2026-05-15
**Closed:** —

## Scope

Refit rak's default behavior to its v0.1.0 product positioning: **wc++ for LLM-first consumption** (per `main/PLAN.md` decisions 30, 32, 33). Two surface changes ship in this drop. First, the **default file source** becomes git-tracked-only — when `git rev-parse --is-inside-work-tree` succeeds at the walk root, enumerate via `git ls-files --full-name -z` (NUL-delimited, paths relative to `git rev-parse --show-toplevel`); when not in a git repo, fall back to the existing `internal/fileset.Walker` + `.gitignore` filter from Drop 3 (unchanged). Second, the **default renderer** becomes TOON via `github.com/toon-format/toon-go`; the Drop 3.5 `--format auto|human|json` flag is replaced by mutually exclusive boolean flags `--human` / `--json` / `--toon`, with TOON as the default regardless of TTY (LLM audience). Drop 3's spine work (`Walker`, `ignore.Matcher`, `File`, binary detection, per-dir aggregation) is preserved — the `WalkLister` is a thin adapter over the existing `Walker`, and `humanRenderer` / `jsonRenderer` keep their Drop 2/3 contracts. Expected decomposition: 6 atomic units (4.0 dep add / 4.1 lister interface / 4.2 GitLister / 4.3 WalkLister adapter / 4.4 cmd/rak rewire + flag-surface reshape / 4.5 TOON renderer + snapshot tests).

Lockfiles (`go.sum`, `package-lock.json`, etc.) are counted by default per decision 34 — whatever git tracks, rak counts. No lockfile denylist in v0.1.0.

## Planner

<Filled by go-planning-agent in Phase 1. Atomic units of work below. Each unit's state is mutated in place by the builder during Phase 4. See main/drops/WORKFLOW.md § "Phase 1 — Plan" for deliverable rules.>

### Unit N.1 — <title>

- **State:** todo
- **Paths:** <file-level footprint>
- **Packages:** <Go package footprint>
- **Acceptance:** <yes/no-verifiable criteria a QA subagent can call>
- **Blocked by:** —

### Unit N.2 — <title>

- **State:** todo
- **Paths:**
- **Packages:**
- **Acceptance:**
- **Blocked by:** N.1

<…repeat per unit…>

## Notes

<Optional. Cross-unit decisions, library choices made during planning, deferrals to later drops.>
