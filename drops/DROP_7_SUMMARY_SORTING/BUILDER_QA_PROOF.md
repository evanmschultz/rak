# DROP_7 — Build-QA Proof

Append a `## Unit N.M — Round K` section per QA-Proof attempt. See `main/drops/WORKFLOW.md` § "Phase 5 — Build QA (per unit)" for what each section should contain.

## Unit 7.1 — Round 1

- **Reviewer:** go-qa-proof-agent
- **Reviewed:** 2026-05-15
- **Files under review:**
  - `internal/summary/summary.go` (new, 53 LOC)
  - `internal/summary/sort.go` (new, 93 LOC)
  - `internal/summary/summary_test.go` (new, 199 LOC)
- **Diff range:** `git diff HEAD~1 -- internal/summary/` (commit `4717207 feat(summary): add directory + summary + sortkey + sortdirs`)

### Acceptance audit

| # | Criterion | Evidence | Status |
|---|---|---|---|
| 1 | `summary.go` defines `Directory{Path, Counts, ByLang, Files}` in F43 order; imports only `internal/counting` + `internal/lang`; doc comments per naming rule 11 | summary.go lines 19-39 (field order); 9-12 (imports); 14-18 (`Directory` doc); 20-23 / 26-27 / 30-32 / 35-37 (per-field docs) | PASS |
| 2 | `Summary{Dirs, Total}` defined (F36) | summary.go lines 44-53; `Dirs []Directory` line 48, `Total counting.Counts` line 52; doc comments lines 41-43, 45-47, 50-51 | PASS |
| 3 | `sort.go` defines `type SortKey string` with constants `SortLines`/`SortFiles`/`SortBytes`/`SortPath`; doc on `SortKey` notes tokens omission (F41) | sort.go line 17 (`type SortKey string`); lines 19-33 (four constants with string values `"lines"`, `"files"`, `"bytes"`, `"path"`); lines 10-16 (tokens omission per Decision 30 / F41 explicitly documented) | PASS |
| 4 | `SortDirs(dirs, key, asc bool)` in-place sort via `slices.SortFunc`; key-specific direction (numeric=desc default, path=asc default); `--sort-asc` flips; unknown key panics | sort.go line 65 (signature); 75 (`slices.SortFunc`); 44-49 (`effectiveAsc`: SortPath returns `!asc`, others return `asc`); 87-90 (`-result` negation when `!eff`); 66-71 (`default: panic("summary: SortDirs called with unrecognized SortKey %q", key)`) | PASS |
| 5 | 11 tests cover all four keys default+flipped, zero-length, single-entry, unknown-key panic | summary_test.go: `TestSortDirs_{Lines,Files,Bytes}_{Default,Asc}` (6), `TestSortDirs_Path_{Default,Asc}` (2), `TestSortDirs_UnknownKey_Panics`, `TestSortDirs_EmptySlice`, `TestSortDirs_SingleEntry` = 11 total. Path_Default asserts `[a,b,c]` ascending (matches key-specific default); Path_Asc asserts `[c,b,a]` descending (flipped) | PASS |
| 6 | `mage ci` green | BUILDER_WORKLOG.md line 13 — `mage ci (pass, gofumpt clean + lint clean + test -race green)`. Unit committed as `4717207` after worklog gate. | PASS |

### Trace verification (semantics)

- `SortDirs(dirs, SortLines, false)` → switch passes; `eff = effectiveAsc(SortLines, false) = false`; comparator computes `cmp.Compare(a.Counts.Lines, b.Counts.Lines)`; since `!eff` is true, returns `-result` → descending. Matches `TestSortDirs_Lines_Default` expected `[20, 10, 5]`. Verified.
- `SortDirs(dirs, SortPath, false)` → `eff = !false = true`; `strings.Compare` returned unchanged → ascending. Matches `TestSortDirs_Path_Default` expected `["a","b","c"]`. Verified.
- `SortDirs(dirs, SortPath, true)` → `eff = !true = false`; negated → descending. Matches `TestSortDirs_Path_Asc` expected `["c","b","a"]`. Verified.
- `SortDirs(dirs, SortKey("tokens"), false)` → switch default branch → panics with descriptive message. Matches `TestSortDirs_UnknownKey_Panics`. Verified.

### Findings

None.

### Missing evidence

None.

### Verdict

**PASS** — all six acceptance criteria are independently supported by code evidence; no unmitigated falsification attack; no missing tests; no documentation gaps.

### Hylla Feedback

N/A — Hylla queries were unnecessary for this review; all evidence came from `git diff` + `Read` of the just-committed files (Hylla index for commit `4717207` would not yet be reingested; reingest is drop-end only).
