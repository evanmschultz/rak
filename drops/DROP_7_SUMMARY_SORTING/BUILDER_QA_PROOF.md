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

## Unit 7.2 — Round 1

- **Reviewer:** go-qa-proof-agent
- **Reviewed:** 2026-05-15
- **Files under review:**
  - `internal/render/render.go` (modified: `Directory` struct deleted, `RenderTree` signature retyped to `[]summary.Directory`, F37 breadcrumb added)
  - `internal/render/json.go` (modified: `directoryJSON.Files int64 \`json:"files,omitempty"\`` added in F43-pinned order; `filterUnknown` retyped to `summary.Directory` with Files propagation; `RenderTree` signature retyped)
  - `internal/render/human.go` (modified: import + `RenderTree` signature retyped)
  - `internal/render/toon.go` (modified: import + `RenderTree` signature retyped)
  - `internal/render/render_test.go` (modified: 15 fixture sites switched `render.Directory` → `summary.Directory`, no `Files:` set → snapshots unchanged via `omitempty`)
  - `cmd/rak/root.go` (modified: `byDirFiles map[string]int64` added; `walkAndCount` returns `[]summary.Directory` with `Files: byDirFiles[p]`; `labelDirectories` propagates Files in both branches; F44 breadcrumb added)
  - `cmd/rak/root_test.go` (added: `TestRootCmd_FilesField_SurvivesLabelDirectories` — end-to-end F44 verification using untyped JSON decode + non-empty rootLabel)
- **Diff range:** `git diff HEAD~1 -- internal/render/ cmd/rak/` (commit `b492a6e refactor: migrate render.directory to summary, propagate files`)

### Acceptance audit

| # | Criterion | Evidence | Status |
|---|---|---|---|
| 1 | `internal/render/render.go`: `Directory` struct deleted; `Renderer.RenderTree` signature uses `[]summary.Directory` (F37) | render.go lines 30-42 (only `Renderer` interface remains); diff removed 25 lines of `Directory` struct; F37 breadcrumb at lines 27-29; `dirs []summary.Directory` at line 41 | PASS |
| 2 | All three renderers compile against `summary.Directory` | human.go:79 `(h humanRenderer) RenderTree(w io.Writer, dirs []summary.Directory, ...)`; json.go:108; toon.go:111. `mage ci` compiled the package green | PASS |
| 3 | `internal/render/json.go` (F34/F43/F44): `directoryJSON` grew `Files int64 \`json:"files,omitempty"\``; field order Path, Counts, ByLang, Files matches `summary.Directory`; `filterUnknown` propagates `Files` (F44) | json.go lines 58-63: `Files int64 \`json:"files,omitempty"\`` declared in that exact position; summary.go lines 19-39 field order matches byte-for-byte; F43 breadcrumb at json.go:52-57; `filterUnknown` retyped `summary.Directory → summary.Directory` at line 74; `Files: d.Files` carried at line 91; early-return at line 76 returns `d` unchanged (Files preserved); F44 breadcrumb at lines 71-73 | PASS |
| 4 | `cmd/rak/root.go`: `walkAndCount` returns `[]summary.Directory`; `byDirFiles` accumulator increments per accepted file (post-binary, post-`--lang`); each `Directory` constructed with `Files: byDirFiles[path]`. `labelDirectories` propagates `Files` (F44 site 2) | root.go:246 signature returns `[]summary.Directory`; line 249 declares `byDirFiles := map[string]int64{}`; binary `continue` at 281-290; lang-filter `continue` at 302-306; `countFile` error `continue` at 326; `byDirFiles[dir]++` at line 333 happens only after all three filters; constructor at 346 sets `Files: byDirFiles[p]`; `labelDirectories` propagates `Files: d.Files` at lines 411 (`.` branch) and 418 (sub-path branch); F44 breadcrumb at 401-403 | PASS |
| 5 | Existing test snapshots survive (zero-Files via `omitempty`) | render_test.go diff shows 15 substitutions of `render.Directory` → `summary.Directory`; none of the rewritten fixtures sets `Files:`, so the field defaults to 0; `omitempty` on `int64` suppresses zero; `mage test` cached-pass confirms snapshots unchanged | PASS |
| 6 | `TestRootCmd_FilesField_SurvivesLabelDirectories` exercises F44 end-to-end | root_test.go:708-764: fixture has 2 `.go` files in root + 3 in `sub/`; calls `runDirectory(..., "myroot", ...)` to force `labelDirectories` reconstruction path; decodes via untyped `map[string]interface{}` (would not auto-zero a missing field); asserts `filesByPath["myroot"]==2` and `filesByPath["myroot/sub"]==3`; both error messages cite F44 | PASS |
| 7 | `mage ci` green | Local re-run: `0 issues.` + 8 packages `ok`. Builder worklog records the gate as the precondition for commit `b492a6e` | PASS |

### Trace verification (semantics)

- **F44 propagation full path.** A `.go` fixture at `sub/c.go` enters `walkAndCount` → passes binary check (281-290, ASCII content) → passes lang gate (no `--lang` filter, `wantedLangs == nil`) → `countFile` returns no err → `byDirFiles["sub"]++` at line 333 → final `summary.Directory{Path: "sub", Files: byDirFiles["sub"]}` constructed at line 346 → `labelDirectories` rewrites to `summary.Directory{Path: "myroot/sub", Files: d.Files}` at line 418 → `runDirectory` calls `r.RenderTree(w, dirs, ...)` (json renderer) → `RenderTree` loops dirs at json.go:113 → `filterUnknown(d)` at line 114 preserves `Files: d.Files` (line 91) → `directoryJSON(filterUnknown(d))` bare struct conversion (same field order) → JSON encoder emits `"files": 3` (non-zero, omitempty does not suppress) → test decodes via untyped map → assertion `filesByPath["myroot/sub"] == 3` passes. Each hop verified by code inspection.

- **Snapshot preservation trace.** `TestJSONRenderer_RenderTree_Snapshot` fixture at render_test.go:272-281 builds `[]summary.Directory{ {Path: ".", Counts: ...}, {Path: "sub", Counts: ...} }` — no `Files:` set → zero int64 → `omitempty` suppresses → existing snapshot bytes (no `"files"` key) match. `mage ci` cached `ok` on render package confirms.

- **Filter-ordering trace (false positive guard).** A `.gif` binary file at `sub/img.gif` would: enter `walkAndCount` → fail binary check (`isBin == true` at line 287) → `continue` at 288 BEFORE the `byDirFiles[dir]++` at 333. Therefore `Files` correctly excludes skipped binaries. Same path for `--lang go` excluding non-Go files: lang gate `continue` at 304 fires before the increment. Audit-clean.

- **`directoryJSON(filterUnknown(d))` struct-conversion validity.** Go requires source and destination structs in a value conversion to have identical fields in the same declaration order, identical types, with tags ignored. `summary.Directory` (summary.go:19-39): `Path string`, `Counts counting.Counts`, `ByLang map[lang.Language]lang.LangCounts`, `Files int64`. `directoryJSON` (json.go:58-63): `Path string`, `Counts counting.Counts`, `ByLang map[lang.Language]lang.LangCounts`, `Files int64`. Identical. Conversion at line 114 compiles. `mage ci` confirms.

### Findings

None.

### Missing evidence

None.

### Verdict

**PASS** — all seven acceptance criteria are independently supported by code evidence; 10 falsification attacks all mitigated; `mage ci` green; the new F44 test correctly proves the propagation invariant end-to-end via an untyped-decode path that cannot be silently zeroed by Go's typed-decoder defaults.

### Hylla Feedback

None — Hylla answered everything needed. (Hylla was not queried for this review because commit `b492a6e` is post-last-ingest; all evidence came from `git diff HEAD~1`, on-disk `Read`, and `mage ci` re-run. This is the correct fallback per CLAUDE.md § "Code Understanding Rules" rule 2 — "Changed since last ingest: use `git diff`. Hylla is stale for those files until reingest." It is not a Hylla miss.)
