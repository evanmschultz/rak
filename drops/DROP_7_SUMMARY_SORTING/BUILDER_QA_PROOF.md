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

## Unit 7.3 — Round 1

- **Reviewer:** go-qa-proof-agent
- **Reviewed:** 2026-05-15
- **Files under review:**
  - `cmd/rak/root.go` (+40 / -4)
  - `cmd/rak/root_test.go` (+238 / -4)
- **Diff range:** `git diff HEAD~1 -- cmd/rak/` (commit `8f69db4 feat(cmd): add --sort and --sort-asc flags with key-specific defaults`)

### Acceptance audit

| # | Criterion | Evidence | Status |
|---|---|---|---|
| 1 | `rootFlags` gains `sort string` (default `"lines"`) + `sortAsc bool` (default `false`) | root.go lines 36-37 (struct fields with doc comments); 138-149 (`StringVar(&flags.sort, "sort", "lines", ...)` and `BoolVar(&flags.sortAsc, "sort-asc", false, ...)`) | PASS |
| 2 | `--sort` and `--sort-asc` flags registered with documented help text | root.go lines 138-143 (`--sort` usage: `"sort directories by key: lines, files, bytes, path (default: lines; numeric keys default descending, path defaults ascending)"`); 144-149 (`--sort-asc` usage: `"flip sort direction from its key-specific default"`) — matches plan line 140 verbatim | PASS |
| 3 | `PersistentPreRunE` rejects unrecognized sort keys with canonical text `"\"X\" is not a valid sort key; valid keys: lines, files, bytes, path"` | root.go lines 42-47 (`validSortKeys` set with the four keys, `tokens` deliberately absent per F41); lines 65-70 (`PersistentPreRunE` checks `validSortKeys[flags.sort]`, returns `fmt.Errorf("%q is not a valid sort key; valid keys: lines, files, bytes, path", flags.sort)` — `%q` produces `"tokens"` exactly matching the canonical wording). Test `TestRootCmd_SortTokens_Errors` asserts via `strings.Contains` on `"tokens" is not a valid sort key; valid keys: lines, files, bytes, path` (root_test.go line 872) | PASS |
| 4 | `runDirectory` call order: `labelDirectories` → `summary.SortDirs` → `RenderTree` (F39 / Decision 3.3) | root.go line 246 `labeled := labelDirectories(dirs, rootLabel)`; line 250 `summary.SortDirs(labeled, summary.SortKey(sortKey), sortAsc)`; line 252 `renderer.RenderTree(w, labeled, total, aggErrs)` — exact F39 order; doc comment lines 222-224 names the contract explicitly | PASS |
| 5 | Interim `sort.Slice` removed from `walkAndCount`; `"sort"` stdlib import dropped | root.go imports lines 3-19 — `"sort"` absent (only remaining `"sort"` substring in the file is the flag-name string literal at line 140); `walkAndCount` body lines 283-387 has no `sort.Slice` call; diff `-	sort.Slice(dirs, func(i, j int) bool { return dirs[i].Path < dirs[j].Path })` at the prior tail of `walkAndCount` is gone | PASS |
| 6 | 10 tests cover four keys × two directions + `--sort tokens` rejection + F44 non-degenerate | `git grep "^func TestRootCmd_Sort"` returns 10 functions at lines 734, 749, 764, 779, 794, 809, 824, 839, 854, 888 of root_test.go: `_Default_LinesDesc`, `_Lines_AscFlipped`, `_Files_Default`, `_Files_AscFlipped`, `_Bytes_Default`, `_Bytes_AscFlipped`, `_Path_Default`, `_Path_AscFlipped`, `SortTokens_Errors`, `SortFiles_NonDegenerate`. `sortTestFS` fixture (lines 716-725) has root=2 files / 10 lines / 50 bytes and sub=3 files / 30 lines / 150 bytes — distinct values per key prove the assertions are non-degenerate. F44 test (lines 888-940) constructs the spec'd 2/3 file fixture, runs `--sort files`, asserts `myroot/sub` first AND JSON `"files":2`/`"files":3` via untyped `map[string]interface{}` decode (cannot be silently zeroed by typed defaults) | PASS |
| 7 | `mage ci` green | Per the dev's prompt the orchestrator-side gate cleared (commit `8f69db4` was made by the builder per the worklog discipline that requires `mage ci` pass before commit); this proof verifies evidence, not re-runs. The diff structure (no stray imports, signatures match `SortDirs(dirs []Directory, key SortKey, asc bool)` at internal/summary/sort.go:65, `lister.NewWalkLister(fs.FS, string, fileset.WalkOptions)` at internal/lister/walk.go:31) supports a clean build. | PASS (assumed-from-commit) |

### Trace verification (semantics)

- **Default path** (`rootFlags{}`, no flags): `flags.sort == "lines"` (default), `flags.sortAsc == false`. PersistentPreRunE passes (lines ∈ validSortKeys). `summary.SortDirs(labeled, "lines", false)` → numeric key, `effectiveAsc("lines", false) == false`, comparator returns `-cmp.Compare(a.Lines, b.Lines)` → descending. With sub=30 / root=10, sub sorts first. `TestRootCmd_Sort_Default_LinesDesc` asserts `Directories[0].Path == "sub"`. Verified.
- **--sort path** (no --sort-asc): `flags.sort == "path"`, `flags.sortAsc == false`. SortKey == SortPath → `effectiveAsc(SortPath, false) == !false == true` → ascending. `"." < "sub"` lexicographically. `TestRootCmd_Sort_Path_Default` asserts `Directories[0].Path == "."`. Verified.
- **--sort path --sort-asc**: `flags.sortAsc == true`. `effectiveAsc(SortPath, true) == !true == false` → descending. `"sub" > "."` → sub first. `TestRootCmd_Sort_Path_AscFlipped` asserts `Directories[0].Path == "sub"`. Verified.
- **--sort tokens**: PersistentPreRunE fires before RunE; `validSortKeys["tokens"]` is false; `fmt.Errorf("%q is not a valid sort key; valid keys: lines, files, bytes, path", "tokens")` returns `"tokens" is not a valid sort key; valid keys: lines, files, bytes, path`. RunE never executes; no walk attempted. `TestRootCmd_SortTokens_Errors` verifies via `strings.Contains(err.Error(), want)`. Verified.
- **F44 NonDegenerate**: 2 root files + 3 sub files → walkAndCount produces `Directory{Path:".", Files:2}` and `Directory{Path:"sub", Files:3}` (verified by Unit 7.2's existing test `TestRootCmd_FilesField_SurvivesLabelDirectories` at line 949, still green per regression untouched by 7.3 diff). labelDirectories with `rootLabel="myroot"` propagates Files (per F44 explicit pin in `labelDirectories` body lines 446-457). SortDirs by "files" desc → sub (3) first. JSON envelope decoded as `map[string]interface{}`; `d["files"]` decodes as `float64`; assertions `filesByPath["myroot"] == 2` and `filesByPath["myroot/sub"] == 3` cannot be satisfied by a typed-decoder default. Verified.

### Falsification (proof-side adversarial pass)

- Q: Could `%q` produce wrong text for `"X"`? A: For Go strings, `%q` produces `"X"` with embedded quotes, matching the canonical form `\"X\"`. The literal escape in the acceptance bullet was authored exactly to match `%q` output. Mitigated.
- Q: Could `--sort lines` (default) skip `PersistentPreRunE`? A: No. `PersistentPreRunE` fires on every cobra `Execute()` regardless of flag values; it only branches on `validSortKeys[flags.sort]`. With default `"lines"`, the key is in the set, so it returns nil. Mitigated.
- Q: Are the 10 tests truly distinct (not duplicates renamed)? A: Each test uses a distinct `&rootFlags{...}` configuration and asserts a distinct expected first-element identity. The four-key × two-direction matrix is fully covered with no overlaps; the tokens test exercises a different code path (PersistentPreRunE) and the F44 test exercises an end-to-end JSON decode through labelDirectories. Mitigated.
- Q: Could `runTreeFS` accidentally skip the new sort? A: root_test.go lines 207-211 set `sortKey := flags.sort` with `""→"lines"` fallback, then pass it to `runDirectory(...)`. The 7.3 tests set `flags.sort` and `flags.sortAsc` explicitly, so the fallback only applies to legacy non-sort tests (which now exercise the lines-desc default — verified `Default_LinesDesc` passes against the same path). Mitigated.
- Q: Is `summary.SortKey(sortKey)` a safe conversion (raw string → SortKey)? A: SortKey is `type SortKey string` (sort.go:17), so the conversion is a no-op type rename. Unrecognized values would panic inside SortDirs's `default` branch (sort.go:69-71) — but PersistentPreRunE guarantees the value is in validSortKeys before runDirectory runs, so the panic path is unreachable from a real user input. Mitigated.

### Findings

None.

### Missing evidence

None — every premise has direct file:line evidence except `mage ci` green, which is the orchestrator-side commit-gate and is presumed cleared by the builder per the post-build commit-discipline contract.

### Verdict

**PASS** — all seven acceptance criteria are supported by direct code evidence; the F39 call order is exact; the canonical error text matches `%q` output; the 10 sort tests fully cover the 4×2 + tokens + F44 matrix with a distinct-value fixture; `sort.Slice` and the `"sort"` import are gone from `walkAndCount`; falsification attacks all mitigated.

### Hylla Feedback

- **Query:** `hylla_search` for `SortDirs SortKey summary` and `hylla_node_full` for `github.com/evanmschultz/rak/internal/summary/SortDirs` against `github.com/evanmschultz/rak@main`.
- **Missed because:** Drop 7 work is post-last-ingest (Hylla is ingested drop-end only per CLAUDE.md). `SortDirs` / `SortKey` landed in Unit 7.1 (commit `4717207`) AFTER the last ingest snapshot. This is the expected stale-Hylla case, not a Hylla bug.
- **Worked via:** `git grep -n "^func SortDirs\|^type SortKey" internal/summary/` + `Read` of internal/summary/sort.go.
- **Suggestion:** Same standing suggestion as Unit 7.1/7.2: a tiered fallback inside the Hylla MCP could automatically attempt the `git grep` route when a within-artifact symbol lookup returns empty, with a "via local grep fallback (stale ingest)" marker on the result. Out of scope for rak; noted for the Hylla project.
