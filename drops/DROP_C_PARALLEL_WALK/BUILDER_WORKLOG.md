# DROP_C — Builder Worklog

## Unit C.1 — Round 1

- **Builder:** go-builder-agent
- **Started:** 2026-05-17
- **Files touched:**
  - `internal/summary/sort.go` — two-text edit: (1) doc comment updated to reference `slices.SortStableFunc`; (2) `slices.SortFunc` replaced with `slices.SortStableFunc` on line 76.
  - `internal/summary/sort_test.go` — new file; adds `tieRichDirs()` helper and `TestSortDirs_StableTieBreak` with four subtests (lines, files, bytes, path). All other sort coverage already existed in `summary_test.go`.
- **Mage targets run:** `mage build` (pass), `mage test` (pass for `internal/summary`; pre-existing `internal/lister` failure due to DROP_D `NewFilesFromLister` stub referenced in `lister_test.go` but not yet implemented — unrelated to this unit's scope)
- **Notes:**
  - `summary_test.go` already existed with comprehensive sort tests (TestSortDirs_Lines_Asc, _Default, _Files, _Bytes, _Path, _UnknownKey_Panics, _EmptySlice, _SingleEntry). My initial `sort_test.go` duplicated them and caused `TestSortDirs_Lines_Asc redeclared` compile error. Rewrite to contain only `TestSortDirs_StableTieBreak` and its helper `tieRichDirs`.
  - Tie-rich fixture: three dirs all with `Lines=100, Files=5, Bytes=2048` — comparator returns 0 for all numeric keys, so stable-sort assertion is genuine (not vacuous).
  - `slices` import was already present in `sort.go`; no import changes needed.

## Hylla Feedback

- **Query**: hylla_search for `SortDirs` in `github.com/evanmschultz/rak@main`.
- **Missed because**: Hylla was not queried — per evidence-order rules, for single-file edits with exact path known from the spawn prompt, `Read` is the direct route. Hylla would be the correct first stop for symbol discovery across a large codebase; here the symbol was named explicitly.
- **Worked via**: Direct `Read` of `internal/summary/sort.go` and `internal/summary/summary.go`.
- **Suggestion**: N/A — task was simple enough that Hylla wasn't warranted; no miss to report.
