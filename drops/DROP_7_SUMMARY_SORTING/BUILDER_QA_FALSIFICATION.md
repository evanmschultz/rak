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

---

## Unit 7.3 — Round 1

**Date:** 2026-05-15
**Verdict:** PASS (no CONFIRMED counterexamples)
**Files reviewed:**
- `cmd/rak/root.go` (+40/-4)
- `cmd/rak/root_test.go` (+238/-4)
**Commit:** `8f69db4 feat(cmd): add --sort and --sort-asc flags with key-specific defaults`
**Evidence:** `git diff HEAD~1 -- cmd/rak/`, direct `Read` of `root_test.go` sort-block (lines 800–940), `git diff HEAD~1 -- ':!cmd/rak/' ':!drops/'` (empty), `mage ci` (green).

### Attack pass

| # | Attack surface | Outcome | Notes |
|---|---|---|---|
| 1 | F41 canonical error text — exact match between source and test assertion | REFUTED | Source: `fmt.Errorf("%q is not a valid sort key; valid keys: lines, files, bytes, path", flags.sort)` in `root.go` PersistentPreRunE. Test assertion in `TestRootCmd_SortTokens_Errors` (line 872): wants `` `"tokens" is not a valid sort key; valid keys: lines, files, bytes, path` ``. `%q` on `"tokens"` produces `"tokens"` with surrounding double quotes — character-for-character match with the test's `want` string. Single canonical form. |
| 2 | `PersistentPreRunE` actually fires before `RunE` | REFUTED | Cobra's documented execution order is PersistentPreRunE → PreRunE → RunE (cobra `Command.Execute` chain). `TestRootCmd_SortTokens_Errors` drives `cmd.Execute()` with `--sort tokens .` and `cmd.SetIn(strings.NewReader(""))`. If the validator did not fire before `runRoot`, the test would either succeed silently (no error returned), fail with a different error (e.g. walk error on `.`), or hang on stdin. None occur — the test expects and gets the canonical validator error. The path arg `.` ensures `RunE` would otherwise reach `runDirectory`, so the validator is the only thing that can short-circuit. |
| 3 | F44 e2e depth: ordering AND JSON content both asserted | REFUTED | `TestRootCmd_SortFiles_NonDegenerate` (lines 888–940) asserts BOTH (a) `firstPath == "myroot/sub"` after `--sort files` default desc (ordering on `Files` field, proving SortDirs uses real Files data, not a degenerate field where 0==0), AND (b) `filesByPath["myroot"] == 2` and `filesByPath["myroot/sub"] == 3` (JSON envelope carries non-zero Files through labelDirectories reconstruction). Both halves use unmarshalled JSON, not stub values. F44 is a real end-to-end smoke. |
| 4 | `--sort path --sort-asc` flip behavior (the C1 trap) | REFUTED | `TestRootCmd_Sort_Path_AscFlipped` (lines 839–850) explicitly asserts `res.Directories[0].Path == "sub"` (not `"."`) under `&rootFlags{sort: "path", sortAsc: true}`. Because path's key-specific default is asc, `--sort-asc` flips to descending; `"sub"` > `"."` lexicographically, so `"sub"` lands first. Test directly exercises the inversion-vs-numeric-keys asymmetry; the C1 trap is caught. Companion `TestRootCmd_Sort_Path_Default` (lines 824–835) confirms the unflipped path-asc default ordering. |
| 5 | `runDirectory` call ordering invariant: sort AFTER labelDirectories (F39 / Decision 3.3) | REFUTED | Diff at `runDirectory` body: `labelDirectories(...)` returns `labeled`, then comment "Apply user-controlled sort AFTER labelDirectories so SortDirs operates on the final user-facing paths (Decision 3.3, F39)", then `summary.SortDirs(labeled, summary.SortKey(sortKey), sortAsc)`, then `renderer.RenderTree(w, labeled, ...)`. Order: labelDirectories → SortDirs → RenderTree. Matches F39 / Decision 3.3 verbatim. |
| 6 | No leftover `sort.Slice` in `walkAndCount` (would double-sort with path tiebreaker) | REFUTED | Diff in `walkAndCount`: `-sort.Slice(dirs, func(i, j int) bool { return dirs[i].Path < dirs[j].Path })` (removed). Also `-"sort"` in the import block (the std `sort` package is no longer imported by `root.go`). Confirms no leftover pre-sort that would change tiebreaker behavior for `slices.SortFunc` (which is not stable). |
| 7 | Drop 5/6 surface preservation (cmd/rak-only touch) | REFUTED | `git diff HEAD~1 -- ':!cmd/rak/' ':!drops/'` returns empty. Only `cmd/rak/root.go` (+40/-4) and `cmd/rak/root_test.go` (+238/-4) changed. No leakage into `internal/lang/`, `internal/summary/`, `internal/render/`, `internal/fileset/`, `internal/lister/`, or any other package. |
| 8 | `mage ci` re-verify | REFUTED | Re-ran `mage ci`: golangci-lint `0 issues`; `go test -race ./... ` all 9 packages `ok` (most cached, `cmd/rak` re-run with new tests). |
| 9 | Shipped-but-not-wired (validator without consumer) | REFUTED | F41 has schema (`validSortKeys` map), resolver (PersistentPreRunE closure), AND integration test (`TestRootCmd_SortTokens_Errors` drives full `cmd.Execute()`). F44 has schema (`Files` field in summary.Directory, threaded since unit 7.2), resolver (`summary.SortDirs` with `SortFiles` key), AND integration test (`TestRootCmd_SortFiles_NonDegenerate` decoding real JSON). All four steps (schema/resolver/consumer/integration test) present for both load-bearing claims. |
| 10 | Error wrapping (`%w` vs `%q`) at validator | REFUTED | Validator uses `fmt.Errorf("%q is not a valid sort key; ...", flags.sort)` with no `%w`. CLAUDE.md § "Errors" mandates `%w` "at every boundary that adds information" to an existing error. Here there is no underlying error to wrap — the validator constructs a fresh user-facing message from a flag value. No sentinel callers `errors.Is` against this either (the only consumer is cobra surfacing the error to the user). Not a CLAUDE.md violation. |
| 11 | Concurrency / shared mutable state | REFUTED | `validSortKeys` is a package-level `map[string]struct{}` populated at package init via a literal, never mutated thereafter. Map reads from multiple goroutines are safe when no writer is active. No other shared state introduced by the unit. |
| 12 | Hidden init / package-level side effects | REFUTED | No `init()` added. `validSortKeys` is a literal var declaration; no ordering coupling. |
| 13 | YAGNI on `validSortKeys` (4-entry map vs slice + linear scan) | EXHAUSTED, no counterexample found | A 4-entry slice with linear scan would work. But map-set is the idiomatic Go choice for membership tests and v0.2 will add `tokens` per Decision 30. The map costs ~32B more than a slice — not a defect. |
| 14 | Empty-`--sort` value handling | REFUTED | Cobra default `"lines"` from `cmd.Flags().StringVar(&flags.sort, "sort", "lines", ...)`. If a user explicitly passes `--sort ""`, the validator rejects (`""` not in `validSortKeys`) and returns the canonical error with empty quotes — `"" is not a valid sort key; ...`. Consistent rejection, no panic. |
| 15 | `runTreeFS` default-injection vs. cobra default parity | REFUTED | `runTreeFS` injects `sortKey = "lines"` when `flags.sort == ""`; cobra also defaults `flags.sort` to `"lines"` via `StringVar`. Production path and test path use identical defaults — existing test assertions in unrelated tests (e.g. `TestRootCmd_PerLangRollup`) remain valid because lines-desc preserves the pre-Drop-7 default path-asc-ish order only by coincidence on a single-dir fixture (one dir, no tiebreak needed). Confirmed by green `mage ci`. |

### Prompt-injection family

DORMANT — pre-team-feature. Rak does not expose contributor-authored content to agent spawn prompts. No active attack vectors.

### Cascade-vocabulary attacks

N/A — rak does not use the Tillsyn cascade vocabulary.

### Plan-level attacks

N/A — this is `build-qa-falsification`, not `plan-qa-falsification`.

### Hylla Feedback

N/A — Unit 7.3 touched only `cmd/rak/root.go` and `cmd/rak/root_test.go`. All evidence gathered via `git diff` + direct `Read` + `mage ci`. No Hylla queries attempted, no fallbacks recorded. (Hylla is Go-aware but the diff-based audit is the natural primary evidence for a 2-file, single-commit unit; Hylla would only be consulted for cross-package reference confirmation, which was not needed.)

### Falsification certificate

- **Premises:** Unit 7.3 must (a) introduce `--sort` (default `lines`) + `--sort-asc` (default `false`) on the root cobra command; (b) reject any `--sort` value not in `{lines, files, bytes, path}` with the exact F41 canonical text in a `PersistentPreRunE`; (c) thread `sortKey` and `sortAsc` into `runDirectory`; (d) call `summary.SortDirs(labeled, ...)` AFTER `labelDirectories` and BEFORE `renderer.RenderTree` (F39 / Decision 3.3); (e) remove the interim `sort.Slice(dirs, ...)` from `walkAndCount`; (f) preserve key-specific defaults (numeric keys desc; path asc) and the `--sort-asc` inversion (the C1 trap); (g) ship an F44 non-degenerate end-to-end test asserting both ordering on `Files` and non-zero JSON Files values through labelDirectories; (h) confine the change to `cmd/rak/`; (i) `mage ci` green.
- **Evidence:** `git diff HEAD~1 -- cmd/rak/root.go` (validator + flag wiring + runDirectory threading + sort.Slice removal); `Read` of `cmd/rak/root_test.go` lines 800–940 (all 6 sort-key/direction tests, error test, F44 e2e test); `git diff HEAD~1 -- ':!cmd/rak/' ':!drops/'` (empty — no cross-package leakage); `mage ci` re-run (golangci-lint 0 issues, all 9 packages `ok`).
- **Trace or cases:** 15 attack surfaces (8 from spawn prompt + 7 Go-quality / discipline). 13 REFUTED with reproducible evidence, 2 EXHAUSTED with stated reason (YAGNI + Hylla N/A). No CONFIRMED counterexamples.
- **Conclusion:** PASS. Unit 7.3's claim — `--sort` / `--sort-asc` wired with F41 validation, F39 ordering, F44 end-to-end coverage, no Drop 5/6 spillover, mage ci green — is upheld under active adversarial attack.
- **Unknowns:** None routed. The cobra-execution-order claim (PersistentPreRunE before RunE) is asserted from cobra documentation rather than Hylla-grounded source; an LSP verification would close that residual but the test's observable behavior (`cmd.Execute()` returns the validator error before any walk runs) is itself sufficient evidence.

