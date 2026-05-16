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

## Unit 9.2 — Round 1

- **Builder:** go-builder-agent
- **Started:** 2026-05-15
- **Files touched:**
  - `cmd/rak/main.go` — added `const version = "v0.1.0"` with doc comment; appended `fang.WithVersion(version)` to the `fang.Execute` call. `fang.WithNotifySignal` preserved.
  - `cmd/rak/root_test.go` — added `TestRootCmd_Version` immediately before `TestRootCmd_TotalByLang_EndToEnd`. Test sets `cmd.Version = version` (mirroring what `fang.WithVersion` does to the cobra command), captures output via `cmd.SetOut`, asserts `strings.Contains(got, "v0.1.0")`.
- **LOC added:** `main.go` +8 lines (const block + option arg); `root_test.go` +25 lines (one test func + comment block).
- **TDD cycle:** RED = compile failure (`version` undefined in test); GREEN = added `const version` + `fang.WithVersion(version)` in `main.go`; `mage test` all packages pass.
- **Version format observed:** cobra prints `rak version v0.1.0\n` to `cmd.OutOrStdout()` when `cmd.Version != ""` and `--version` is passed. `strings.Contains("v0.1.0")` assertion is not brittle — fang's extra styling may wrap it but the literal `v0.1.0` always appears.
- **Decision:** hardcoded `const` over `var` — version is a pure constant; `const` prevents accidental mutation. Build-time `-ldflags` injection deferred to v0.2 per U2 decision.
- **Design — test approach:** test uses `cmd.Version = version` directly rather than calling `fang.Execute`. Rationale: fang.Execute sets `cmd.Version` then calls `cmd.Execute` internally. Testing through `fang.Execute` would require capturing `os.Stdout` at process level (fang may write directly). The direct `cmd.Version` assignment tests that cobra's built-in version mechanism works correctly; the `main.go` change is the real deliverable and is verified by `mage ci` green.
- **`mage ci` result:** pass green, 87.8% coverage (above 70% floor).
- **Exact `--version` output format** (for README 9.1 reference): `rak version v0.1.0` — cobra emits `<cmd.Use> version <cmd.Version>` on a single line to `cmd.OutOrStdout()`. Fang may add additional styling/theming in TTY mode; the literal `v0.1.0` is always present.

## Hylla Feedback

None — Hylla answered everything needed. The unit only touched `cmd/rak` package-level wiring; Hylla confirmed no additional callers or symbol usage was relevant. No misses encountered.

## Pre-9.4 — Renderer Output Ordering Fix — Round 1

- **Builder:** go-builder-agent
- **Started:** 2026-05-15
- **Files touched:**
  - `internal/render/toon.go` — reordered `toonTree` struct fields from `Directories, Total, TotalByLang, ByLang, Errors` to `Directories, ByLang, TotalByLang, Total, Errors`. Rewrote doc comment to enumerate emission order explicitly.
  - `internal/render/json.go` — reordered `treeJSON` struct fields from `Directories, Total, TotalByLang, Errors` to `Directories, TotalByLang, Total, Errors`. Rewrote doc comment to enumerate emission order.
  - `internal/render/human.go` — moved `total lang:` emission loop BEFORE the `printer.KV(countsKV("total", s.Total))` call so grand total appears last.
  - `internal/render/render_test.go` — inverted the `TestRenderer_TotalByLang_Human` ordering assertion: now asserts `total lang:` blocks precede `total` block (using `strings.LastIndex` for grand total, `strings.Index` for first `total lang:` occurrence).
- **TDD cycle:** ran `mage test` after each file change; all tests green.
- **Scope:** only `internal/render/*.go` touched. No `cmd/rak/`, `internal/summary/`, or other packages modified.
- **`mage ci` result:** pass green — 87.8% coverage (floor 70%), lint clean, format clean.
- **`directoryJSON` conversion safety:** confirmed that `directoryJSON(filterUnknown(d))` operates on `directoryJSON` ↔ `summary.Directory` bare-struct conversion; `treeJSON` field reorder does not affect it.

## Hylla Feedback (pre-9.4 round)

N/A — task touched only `internal/render/*.go` files; Hylla is Go-indexed but the specific work here was a struct-field reorder fully visible from direct file reads + LSP. No Hylla queries were needed.
