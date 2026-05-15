# DROP_5 — LANGUAGE_DETECTION_CODE_SPLITS

**State:** planning
**Tier:** A
**Blocked by:** DROP_4
**Paths (expected):** `main/internal/lang/` (new package — `Language` type, `Detect`, blank/comment/code splitter + tests), `main/internal/render/render.go` (per-type rollup data shape; F25-aware — interface may grow), `main/internal/render/toon.go` / `main/internal/render/human.go` / `main/internal/render/json.go` (extend to render per-type aggregation), `main/internal/render/render_test.go` (extend snapshot/contains tests), `main/cmd/rak/root.go` (wire language detection into per-file counting + per-type aggregation + add `--lang` walk-filter flag), `main/cmd/rak/root_test.go` (flag-parsing + per-type tests), `main/cmd/rak/integration_test.go` (extend fixture or expectations for per-type rollup)
**Packages (expected):** `github.com/evanmschultz/rak/internal/lang` (new), `github.com/evanmschultz/rak/internal/render`, `github.com/evanmschultz/rak/cmd/rak`
**PLAN.md ref:** main/PLAN.md → `DROP_5_LANGUAGE_DETECTION_CODE_SPLITS` row
**Workflow:** main/drops/WORKFLOW.md
**Started:** 2026-05-15
**Closed:** —

## Scope

Add language awareness to rak's counting. Detect each file's language via (a) extension lookup, (b) shebang sniff using the existing `fileset.File.Peek(512)` contract (F4 from Drop 3), and (c) a small content-heuristic fallback for files whose extension is ambiguous or absent. Per detected language, split each file's lines into three categories — **blank**, **comment**, **code** — using language-specific comment grammar (line-comment markers, block-comment delimiters). Aggregate per-type counts in addition to the existing per-directory rollup; surface both in all three renderers (TOON, human, JSON). Add the `--lang go,rs` walk-filter flag (per main/PLAN.md decision 24) so users can scope counting to one or more detected languages.

Drop 4's spine is preserved: `internal/lister`, `internal/fileset`, `internal/ignore`, `internal/counting`, `internal/render`'s `Renderer` interface (subject to a possible additive growth — planner decides), and the `--human` / `--json` / `--toon` flag surface all remain. Drop 5's new code is additive. Expected decomposition: 4 atomic units (5.1 internal/lang detection / 5.2 code-aware splits / 5.3 per-type aggregation in render / 5.4 `--lang` walk filter). Per the `feedback_parallelize_aggressively` memory rule, 5.2 and 5.4 are eligible to run in parallel after 5.1 closes (both consume `Language` but neither blocks the other).

`--as <lang>` (stream-type assertion for stdin) is cut per decision 30; only `--lang` (walk filter) is added in Drop 5.

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
