# DROP_7 — SUMMARY_SORTING

**State:** planning
**Tier:** A
**Blocked by:** DROP_6
**Paths (expected):** `main/internal/summary/` (new package — `Summary` struct + sort functions + tests), `main/internal/render/render.go` (migrate provisional `render.Directory` to `summary.Summary` or have render consume `summary.Summary` directly), `main/internal/render/{human,json,toon}.go` (update consumers), `main/internal/render/render_test.go` (extend), `main/cmd/rak/root.go` (add `--sort` and `--sort-asc` flags; apply sort to the directories slice before rendering), `main/cmd/rak/root_test.go` (sort behavior tests)
**Packages (expected):** `github.com/evanmschultz/rak/internal/summary` (new), `github.com/evanmschultz/rak/internal/render`, `github.com/evanmschultz/rak/cmd/rak`
**PLAN.md ref:** main/PLAN.md → `DROP_7_SUMMARY_SORTING` row
**Workflow:** main/drops/WORKFLOW.md
**Started:** 2026-05-15
**Closed:** —

## Scope

Land `internal/summary` as the canonical home for rollup data and add `--sort` to surface the per-directory listing in the user's preferred order. The Drop 3 `render.Directory` was provisional per planner pin C8 (carried as a v0.1.0 stand-in); Drop 7 migrates it into `summary.Summary` (or `summary.Directory`) and updates all three renderers + `cmd/rak`'s `walkAndCount` accumulator to produce the new shape. The migration is mechanical for the Drop 4/5 spine (TOON/human/JSON renderers, per-language ByLang map) — all the existing shape stays; only the type's location changes.

Sort surface: `--sort {lines,files,bytes,name}` selects the key; `--sort-asc` flips direction; default is **`lines desc`** per decision 19. The sort applies to the directories slice ONLY (per-language rollup inside each directory remains alphabetically sorted by language string per F33 / Drop 5 deterministic-order convention). `tokens` is NOT a sort key in v0.1.0 — decision 30 cut tokens to v0.2.

Drop 5 spine preserved: `internal/lang` Detect + Split + LangCounts unchanged; F26 RelPath invariant; F33 LangUnknown suppression; cobra `--human` / `--json` / `--toon` / `--lang` / `--include` / `--exclude` / `--depth` / `--hidden` / `--no-gitignore` / `--binary` all unchanged. Renderer interface (F25/F32) may grow if necessary; planner decides per same dep-edge reasoning as Drop 5.

## Planner

<Filled by go-planning-agent in Phase 1. Atomic units of work below. Each unit's state is mutated in place by the builder during Phase 4.>

### Unit N.1 — <title>

- **State:** todo
- **Paths:**
- **Packages:**
- **Acceptance:**
- **Blocked by:** —

## Notes

<Optional.>
