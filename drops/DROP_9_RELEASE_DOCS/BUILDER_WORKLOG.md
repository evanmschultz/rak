# DROP_9 — Builder Worklog

## Unit 9.0 — Round 1

- **Builder:** go-builder-agent
- **Started:** 2026-05-15
- **Files touched:**
  - `internal/summary/summary.go` (add `TotalByLang` field to `Summary`)
  - `internal/render/render.go` (Option A: `RenderTree` signature → `summary.Summary`)
  - `internal/render/toon.go` (add `TotalByLang []toonLangRow` + emit block)
  - `internal/render/human.go` (emit `total by language` section)
  - `internal/render/json.go` (add `TotalByLang` to `treeJSON`, filter LangUnknown)
  - `internal/render/render_test.go` (update call sites + 4 new tests)
  - `cmd/rak/root.go` (aggregate `totalByLang` in `walkAndCount`, construct `summary.Summary`)
  - `cmd/rak/root_test.go` (add `TestRootCmd_TotalByLang_EndToEnd`, extend `treeResultWithLang`)
- **Option chosen:** Option A — `RenderTree(w io.Writer, s summary.Summary, errs []error) error`. Rationale: collapses three separate params (`dirs`, `total`, new `TotalByLang`) into one coherent value type. No external callers under `internal/` so F25/F32 authorize the change. Avoids a 5-param signature. Simpler call sites.
- **`walkAndCount` return extension:** extended from 4-tuple to 5-tuple adding `map[lang.Language]lang.LangCounts` for top-level per-lang totals. `runDirectory` constructs `summary.Summary{Dirs: labeled, Total: total, TotalByLang: totalByLang}` after `labelDirectories` + `SortDirs`.
- **F33 LangUnknown suppression:** applied uniformly — `sortedKnownLangs` (existing helper) used for TOON and human; JSON filters via new `filterTotalByLangUnknown` helper.
- **Mage targets run:** mage build (pass), mage test (pass), mage format (auto-fixed toon.go, json.go, root_test.go), mage ci (pass — 87.8% coverage, floor 70%)

## Unit 9.3 — Round 1

- **Builder:** go-builder-agent
- **Started:** 2026-05-15
- **Files touched:**
  - `main/magefile.go` — added `const coverageFloor = 70.0`, updated `Coverage()` to parse `go tool cover -func` output and enforce the 70% floor, added `parseCoverageTotal()` helper, added `strconv` import, updated `CI()` to include `Coverage` in `mg.SerialDeps`.
  - `main/.github/workflows/ci.yml` — no change needed; workflow already runs `mage ci` as its sole step.
- **Mage targets run:**
  - `mage build` — pass.
  - `mage coverage` — pass (87.3% >= 70.0% floor, gate enforced).
  - `mage ci` — pass green, coverage gate fires and passes within the chain.
- **Coverage observed:** 87.3% on `-coverpkg=./internal/...` scope. 17.3 percentage points above the floor. No scope adjustment needed (resolves U1).
- **Design choices:**
  - `parseCoverageTotal` is an unexported helper in `magefile.go` (package `main`). The `go tool cover -func` output format `total:\t(statements)\t87.3%` is stable across Go versions; `strings.Fields` + `strings.TrimSuffix(..., "%")` + `strconv.ParseFloat` is the simplest correct parse.
  - Used `sh.Output` (captures stdout) for the `go tool cover -func` step then `fmt.Println` to echo the full report, giving the same UX as the previous `sh.RunV` while also enabling parsing.
  - `mg.SerialDeps(gofumptClean, Lint, Test, Coverage)` — appended `Coverage` at the end of the existing serial chain. Fast-fail order preserved: format → lint → test → coverage. Coverage runs last (most expensive; only worth doing if tests are already green).
  - Floor wired as a package-level constant (`coverageFloor = 70.0`) rather than a magic literal — auditable and easy to find.

## Hylla Feedback

N/A — task touched non-Go source files only (`magefile.go` is build automation, not an indexed Go package; `ci.yml` is YAML). Hylla is Go package-indexed; no Hylla queries were needed or made.
