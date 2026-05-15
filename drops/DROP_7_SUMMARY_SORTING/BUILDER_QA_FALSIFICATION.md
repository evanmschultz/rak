# DROP_7_SUMMARY_SORTING — BUILDER_QA_FALSIFICATION

Build-QA falsification rounds for Drop 7. Each round appends; no overwrites.

---

## Unit 7.1 — Round 1

**Date:** 2026-05-15
**Verdict:** PASS (no CONFIRMED counterexamples)
**Files reviewed:**
- `internal/summary/summary.go` (new, 53 LOC)
- `internal/summary/sort.go` (new, 93 LOC)
- `internal/summary/summary_test.go` (new, 199 LOC)
**Commit:** `4717207 feat(summary): add directory + summary + sortkey + sortdirs`
**Evidence:** `git diff HEAD~1 -- internal/summary/`, direct Read of all three files, `mage ci` (green), `go doc slices.SortFunc`, Hylla `LangCounts` lookup.

### Attack pass

| # | Attack surface | Outcome | Notes |
|---|---|---|---|
| 1 | F43 field order (`Path`, `Counts`, `ByLang`, `Files`) for Drop 7.2 bare struct conversion | REFUTED | `summary.go` declares fields exactly in spec order. Confirmed via direct Read. |
| 2 | `SortDirs` stability + tiebreakers (`slices.SortFunc` vs `SortStableFunc`) | REFUTED — spec-authorized | `go doc slices.SortFunc` confirms "not guaranteed to be stable". PLAN.md F38 / unit 7.1 acceptance explicitly mandates `slices.SortFunc` (not `SortStableFunc`). Builder honored spec. Tie-stability is an open design question logged in Unknowns, not a unit 7.1 defect. |
| 3 | `effectiveAsc` inversion for `SortPath` vs numeric keys | REFUTED | Walked four cases manually: `effectiveAsc(SortPath, false)=true → ascending (a,b,c)`; `effectiveAsc(SortPath, true)=false → descending (c,b,a)`; `effectiveAsc(SortLines, false)=false → descending (20,10,5)`; `effectiveAsc(SortLines, true)=true → ascending (5,10,20)`. All four match the test expectations in `TestSortDirs_Path_{Default,Asc}` and `TestSortDirs_Lines_{Default,Asc}`. |
| 4 | Unknown-key panic message format | REFUTED | Panic emits `"summary: SortDirs called with unrecognized SortKey %q"` via `fmt.Sprintf`. Unit 7.1 acceptance only says "descriptive message" — no canonical wording mandated at this layer. The CLI-layer canonical message (`"X is not a valid sort key; valid keys: lines, files, bytes, path"`) is unit 7.3's responsibility. |
| 5 | No external deps (only stdlib + internal/counting + internal/lang) | REFUTED | `sort.go`: `cmp`, `fmt`, `slices`, `strings` (all stdlib). `summary.go`: `internal/counting`, `internal/lang` (internal only). `summary_test.go`: `testing`, `internal/counting`. Zero external imports. |
| 6 | `Files int64` consistency vs `int` | REFUTED | `counting.Counts.Bytes/Lines/Words/Chars` are all `int64`. PLAN.md F35 spec-mandates `int64`. `cmp.Compare[int64]` works correctly in `SortDirs`. Consistent across the domain. |
| 7 | `mage ci` re-verification | REFUTED | Re-ran `mage ci`: golangci-lint `0 issues`, `go test -race ./...` all packages `ok` (most cached). |
| 8 | F26 RelPath invariant | REFUTED | `summary.Directory.Path` is a raw `string` field with no normalization. F26 (forward-slash) is upstream (`fileset`); summary stores whatever caller provides — correct boundary. |

### Go-quality sweep (additional attack families)

- **Concurrency / race:** no goroutines, no shared state, no channels. `SortDirs` mutates the caller's slice in place and documents that explicitly. `mage test` runs with `-race`; no race detected. EXHAUSTED.
- **Interface misuse:** no interfaces in this unit. N/A.
- **Error swallowing:** no error-returning paths in this unit (struct defs + sort func, no I/O). N/A.
- **Raw go commands:** no shell-outs in the diff. Verified via `mage ci`, not `go test`. REFUTED.
- **`mage install` invocation:** no reference anywhere in the diff. REFUTED.
- **YAGNI pressure:** `effectiveAsc` helper is justified by the `SortPath` inversion logic (vs numeric keys) — extracting it makes the comparator readable and the inversion auditable in one place. All four `SortKey` constants have documented purposes. No premature abstraction. REFUTED.
- **Hidden dependencies / `init()`:** no `init()` functions, no package-level state, no test-order coupling. REFUTED.
- **File/package gating:** all three new files live under `internal/summary/` (within declared `paths`). No edits outside scope. REFUTED.
- **Doc comment coverage (project rule 11):** `SortKey`, `SortLines`, `SortFiles`, `SortBytes`, `SortPath`, `effectiveAsc` (internal but documented), `SortDirs`, `Directory`, `Summary`, plus every exported field — all carry doc comments. REFUTED.
- **Test coverage:** 8 sort-key cases (4 keys × 2 directions) + 2 edge cases (empty, single) + 1 panic case. Matches acceptance criteria exactly. EXHAUSTED.

### Counterexamples

None. Zero CONFIRMED counterexamples after 8 named attack surfaces and 10 Go-quality attack families.

### Unknowns

- **Tie-stability for `SortDirs`:** spec mandates `slices.SortFunc` (not stable). Two `Directory` entries with equal primary key (e.g., equal `Files`) have non-deterministic relative order. Output reproducibility (snapshot tests, diff-friendly emission) may want a secondary tiebreaker (most natural: `Path` ascending). This is NOT a unit 7.1 defect — the spec explicitly chose `SortFunc` — but it is worth surfacing to the dev as a downstream design question for Drop 7.2/7.3 (snapshot tests) or a later drop. Routed: this round's verdict; orchestrator may surface to dev or open a separate drop.
- **Panic vs error-return for unknown SortKey:** unit 7.1 panics; unit 7.3 will validate upstream at the CLI layer. The panic is correct under the unit-7.1-only contract ("CLI validates first"), but means library callers outside `cmd/rak` who skip validation would crash the binary. Acceptable for v0.1.0 with no external API; flag if rak ever exposes a stable package API.

### Hylla Feedback

None — Hylla answered everything needed (`LangCounts` lookup, `Counts` struct cross-check). The summary package was queried directly via `Read` since the diff was small and local.

### Falsification certificate

- **Premises:** unit 7.1 must ship `Directory`, `Summary`, `SortKey` constants (Lines/Files/Bytes/Path), and `SortDirs` with documented in-place mutation, key-specific direction defaults (numeric desc, path asc), and panic on unknown key. F43 field order must match the future `directoryJSON` exactly.
- **Evidence:** `git diff HEAD~1 -- internal/summary/`, full Read of all three files, `mage ci` green, `go doc slices.SortFunc` for stability claim, Hylla for `lang.LangCounts` confirmation.
- **Trace or cases:** 8 attack surfaces named in the spawn prompt + 10 Go-quality attack families. All REFUTED or EXHAUSTED with reason.
- **Conclusion:** PASS. No CONFIRMED counterexample. Unit 7.1's claim that the summary package satisfies F35/F36/F38/F41/F42/F43 is upheld.
- **Unknowns:** Tie-stability (spec-authorized but worth dev visibility); panic-vs-error for library callers (acceptable for v0.1.0).

---

## Unit 7.2 — Round 1

**Date:** 2026-05-15
**Verdict:** PASS (no CONFIRMED counterexamples)
**Files reviewed:**
- `internal/render/render.go` (Directory deleted, RenderTree signature retargeted)
- `internal/render/json.go` (`directoryJSON.Files int64`, `filterUnknown` retargeted, `RenderTree` retargeted)
- `internal/render/human.go` (signature only)
- `internal/render/toon.go` (signature only)
- `internal/render/render_test.go` (14 sites migrated, import added)
- `cmd/rak/root.go` (walkAndCount, labelDirectories retargeted; byDirFiles accumulator added)
- `cmd/rak/root_test.go` (new `TestRootCmd_FilesField_SurvivesLabelDirectories`)
**Commit:** `b492a6e refactor: migrate render.directory to summary, propagate files`
**Evidence:** `git diff HEAD~1` per-file, direct Read of `internal/summary/summary.go` for F43 field-order cross-check, `mage ci` (green; gofumpt 0 issues, golangci-lint 0 issues, all packages `ok`).

### Attack pass

| # | Attack surface | Outcome | Notes |
|---|---|---|---|
| 1 | F43 field order (`Path`, `Counts`, `ByLang`, `Files`) byte-for-byte between `summary.Directory` and `directoryJSON` | REFUTED | `summary/summary.go` L19-39: `Path, Counts, ByLang, Files`. `render/json.go` L65-75: `Path, Counts, ByLang, Files`. Identical order. Bare conversion `directoryJSON(filterUnknown(d))` compiles only when types match exactly. |
| 2 | F44 reconstruction sites — each `summary.Directory{...}` literal must carry `Files` | REFUTED | Four reconstruction sites, all carry `Files`: (a) `root.go:346` `summary.Directory{Path: p, Counts: c, ByLang: byDirLang[p], Files: byDirFiles[p]}`; (b) `root.go:411` (Path="." arm) `summary.Directory{..., Files: d.Files}`; (c) `root.go:414-419` (nested arm) `Files: d.Files`; (d) `json.go:84-89` (`filterUnknown` rebuild) `Files: d.Files`. `filterUnknown` short-circuit `if len(d.ByLang)==0 { return d }` returns by value with Files intact. |
| 3 | `byDirFiles[dir]++` accumulator placement vs binary-skip + `--lang` filter | REFUTED | `root.go:333` increment fires AFTER: binary-skip continue (L281-290), `--lang` filter continue (L302-306), and `countFile` error continue (L324-328). Matches `summary.Directory.Files` doc ("accepted: passed binary-skip, --lang, --include, --exclude"). No off-by-one. Map initialized at L249 (`map[string]int64{}`); zero-value ++ is safe. |
| 4 | `labelDirectories` both arms (root="." vs nested) carry Files | REFUTED | `root.go:411` (Path="." → rootLabel arm) carries `Files: d.Files`. `root.go:414-419` (nested arm `rootLabel + "/" + d.Path`) carries `Files: d.Files`. Both arms explicit. |
| 5 | Integration test snapshot resilience — does new `"files"` field break existing `TestRootCmd_Integration_PathArg_JSONFormat`? | REFUTED | `cmd/rak/integration_test.go` decodes into `treeResult{Directories []dirResult, Total counting.Counts, Errors []string}` where `dirResult{Path string, Counts counting.Counts}` (root_test.go:221-230) — only `path` + `counts` consumed. encoding/json silently ignores unknown fields by default, so `"files":1` in the emitted JSON is dropped on decode. Assertions on `rootDir.Counts.Bytes == 12` and `subDir.Counts.Bytes == 8` unaffected. |
| 6 | `render_test.go` migration completeness (14 sites) | REFUTED | `git diff HEAD~1 -- internal/render/render_test.go`: 14 `[]Directory{` → `[]summary.Directory{` migrations + 1 import added. JSON snapshot `TestJSONRenderer_RenderTree_Snapshot` (L270-291) sets no `Files` → zero → `omitempty` suppresses → existing snapshot literal unchanged. |
| 7 | Drop 5 spine preservation (lang/lister/fileset/ignore/counting untouched) | REFUTED | `git diff HEAD~1 -- internal/lang/ internal/lister/ internal/fileset/ internal/ignore/ internal/counting/` returns empty. Spine clean. |
| 8 | `mage ci` re-verification | REFUTED | `mage ci`: gofumpt 0 issues, golangci-lint 0 issues, `go test -race ./...` all 8 packages `ok` (mostly cached on second invocation). Build artifact reproducible. |

### Go-quality sweep (additional attack families)

- **Concurrency / race:** Unit 7.2 introduces no goroutines, no channels, no shared mutable state. `byDirFiles` map is local to `walkAndCount`, never escapes. `mage test -race` clean. EXHAUSTED.
- **Interface misuse:** `Renderer` interface signature changed (`RenderTree`'s `dirs` param type from `[]Directory` to `[]summary.Directory`). All three implementations (`humanRenderer`, `jsonRenderer`, `toonRenderer`) updated symmetrically. Compile-time `_ render.Renderer = ...` assertions in `root_test.go:22-26` and `render_test.go:313-317` would catch any drop. REFUTED.
- **Error swallowing:** Unit 7.2 adds no new error paths. The `walkAndCount` short-circuits (binary-skip, lang-filter, countFile-error) are pre-existing; only the `byDirFiles[dir]++` line is new and has no failure mode. REFUTED.
- **Raw go commands:** no shell-outs in the diff. Verified via `mage ci`, not raw `go test`. REFUTED.
- **`mage install` invocation:** zero references anywhere in the diff. REFUTED.
- **YAGNI pressure:** `Files int64` field exists because unit 7.3 introduces `--sort files`; the F42 chain documents the dependency. No premature generalization. The `omitempty` JSON tag is a deliberate snapshot-compat choice (documented in `directoryJSON` doc comment). REFUTED.
- **Hidden dependencies / `init()`:** no `init()` introduced. `summary` package import is the only graph-level addition to `render` and `cmd/rak`. Import DAG (CLAUDE.md § "Import DAG") preserved: `render → summary`, `cmd/rak → render + summary`. No cycles. REFUTED.
- **File/package gating:** edits restricted to `internal/render/*` + `cmd/rak/*`. No leakage into `internal/summary/` (correct — that was unit 7.1's territory), `internal/lang/`, `internal/lister/`, `internal/fileset/`, `internal/ignore/`, `internal/counting/`. Diff-stat confirms. REFUTED.
- **Doc comment coverage:** `Renderer.RenderTree` doc updated (mentions the summary.Directory transition + F37 breadcrumb). `directoryJSON` doc updated (field-order pin to F43 + new `Files` rationale). `filterUnknown` doc updated (Files propagation rationale + F44 reference). `labelDirectories` doc updated (Files propagation rationale + F44 reference). All exported identifiers in touched packages retain doc comments per project rule 11. REFUTED.
- **Test coverage:** New `TestRootCmd_FilesField_SurvivesLabelDirectories` exercises both `labelDirectories` arms (root → `myroot`, nested → `myroot/sub`) AND the JSON pipeline through `filterUnknown`. Asserts `files=2` and `files=3` post-reconstruction. The negative case (without F44 fix, both would be zero/absent) is the test's load-bearing assertion. EXHAUSTED.

### Counterexamples

None. Zero CONFIRMED counterexamples across 8 named attack surfaces and 10 Go-quality attack families.

### Unknowns

- **JSON `"files":0` invisibility:** `omitempty` on `Files int64` suppresses the field when zero. In current code paths this is impossible (a directory bucket exists in `dirs[]` only if at least one accepted file fired `byDirFiles[dir]++`), but if a future caller constructs a `summary.Directory` literal with `Files=0` and ships it through `jsonRenderer.RenderTree`, the JSON consumer cannot distinguish "no files reported" from "field absent." Acceptable today; flag if a non-walker caller ever appears.
- **Hylla snapshot staleness:** Hylla `artifact_ref=github.com/evanmschultz/rak@main` is at snapshot 5 (pre-Unit-7.1 ingest). `hylla_search_keyword` for `summary.Directory` returned zero results — symbol not yet in the index. Fallback to `git diff` + `Read` was sufficient. Documented in Hylla Feedback below.

### Hylla Feedback

- **Query:** `hylla_search_keyword(query="summary.Directory", artifact_ref="github.com/evanmschultz/rak@main", fields=["content"])`.
- **Missed because:** Hylla snapshot 5 predates Unit 7.1's commit `4717207` (which introduced `internal/summary/`). The artifact ref `@main` resolves to the most recent ingest, not the current commit on disk. Drop-end reingest closes the gap (per CLAUDE.md § "Hylla Baseline").
- **Worked via:** Direct `Read` of `internal/summary/summary.go` + `git diff HEAD~1` per file.
- **Suggestion:** None — this is the expected steady-state ingest cadence. Surfaced only to document the fallback path the build-QA agent took.

### Falsification certificate

- **Premises:** Unit 7.2 must (a) delete `render.Directory`, (b) retarget every `RenderTree` impl signature to `summary.Directory`, (c) match `directoryJSON` field declaration order to `summary.Directory` exactly (F43), (d) propagate `Files` through every reconstruction site (F44: `filterUnknown`, `labelDirectories` both arms), (e) accumulate `byDirFiles` only over accepted files, (f) preserve Drop 5 spine packages, (g) `mage ci` green.
- **Evidence:** `git diff HEAD~1` per touched file, direct Read cross-check of `summary.Directory` against `directoryJSON`, `mage ci` green output, integration-test decoder type inspection, manual trace of `walkAndCount`'s skip-and-continue control flow against `byDirFiles[dir]++` position.
- **Trace or cases:** 8 attack surfaces from the spawn prompt + 10 Go-quality attack families. All REFUTED with reproducible evidence or EXHAUSTED with reason.
- **Conclusion:** PASS. No CONFIRMED counterexample. Unit 7.2's claim that `render.Directory` is fully migrated to `summary.Directory` with F43 field-order parity and F44 Files-propagation across all reconstruction sites is upheld.
- **Unknowns:** `omitempty`-on-zero edge for non-walker callers (cosmetic, not a unit-7.2 defect); Hylla snapshot staleness (expected, fallback to `git diff` + `Read` covered the gap).
