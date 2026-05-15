# DROP_7 — Builder Worklog

Append a `## Unit N.M — Round K` section per build attempt. See `main/drops/WORKFLOW.md` § "Phase 4 — Build (per unit)" for what each section should contain.

## Unit 7.1 — Round 1

- **Builder:** go-builder-agent
- **Started:** 2026-05-15
- **Files touched:**
  - `internal/summary/summary.go` (new, ~50 LOC)
  - `internal/summary/sort.go` (new, ~80 LOC)
  - `internal/summary/summary_test.go` (new, ~130 LOC)
- **Mage targets run:** `mage build` (pass), `mage test` (pass, all 8 packages green), `mage ci` (pass, gofumpt clean + lint clean + test -race green)
- **Notes:**
  - `effectiveAsc` helper resolves key-specific direction: numeric keys pass `asc` straight through (default false → descending); `SortPath` returns `!asc` (default false → effective ascending, matching `ls` convention).
  - Used `slices.SortFunc` + `cmp.Compare` + `strings.Compare` from stdlib (Go 1.21+; project on 1.26.1).
  - Unknown `SortKey` panics via a `switch default` branch with `fmt.Sprintf` message; the CLI layer validates before calling (Unit 7.3).
  - `Directory` field order (Path, Counts, ByLang, Files) matches F43 requirement for bare struct conversion in `directoryJSON` (Unit 7.2 scope).
  - `mage test` does not accept a package-path argument — target runs `go test -race ./...`; summary package green in the full run.
