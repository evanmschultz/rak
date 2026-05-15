# DROP_5 — Builder Worklog

Append a `## Unit N.M — Round K` section per build attempt. See `main/drops/WORKFLOW.md` § "Phase 4 — Build (per unit)" for what each section should contain.

## Unit 5.2 — Round 1

- **Builder:** go-builder-agent
- **Started:** 2026-05-15
- **Files touched:**
  - `internal/lang/split.go` (new — 165 LOC)
  - `internal/lang/split_test.go` (new — 185 LOC)
- **Mage targets run:** `mage test` — `internal/lang` GREEN (`ok github.com/evanmschultz/rak/internal/lang`); `cmd/rak` build failed due to concurrent Unit 5.4 edits to `root.go`/`root_test.go` (out of scope). `mage ci` deferred — will pass once 5.4 lands and the tree is whole.
- **Design decisions:**
  - **Grammar table placement:** inline in `split.go` (one new file simpler than two; total production LOC ~165 is well within atomic sizing).
  - **Policy α implementation:** `strings.Contains(line, g.blockOpen)` check guarded by `g.blockOpen != ""` to prevent false positives on languages with empty grammar (LangUnknown, LangJSON, etc.). Same guard on `blockClose` and `linePrefix`. Critical correctness fix from Section 0 falsification pass (Attack 4/5).
  - **Block state machine:** forward scan within each line for unmatched `/*`/`-->` markers; last marker wins. `inBlockComment` carries across lines for multi-line block spans.
  - **Python docstrings (C7):** Python grammar uses only `linePrefix="#"`; triple-quote `"""` detection is not implemented. Triple-quoted strings have no `#` prefix and no block markers → classified as Code. This matches cloc behavior and is documented here per F28.
  - **CMake block comments:** Grammar includes `#[[`/`]]` block markers per PLAN.md grammar table. `#` line-comment prefix also set for CMake.
  - **LangJSON — no grammar:** `LangJSON` is intentionally absent from `grammarTable`. Zero grammar = no comment detection = all non-blank lines are Code. JSON has no comments per spec.
  - **LangMarkdown — HTML comment block:** Grammar set to `blockOpen="<!--"`, `blockClose="-->"`, no line prefix. Lines with HTML comment markers are classified as Comment; all other non-blank lines (including `# heading`) are Code.
  - **Known limitation (F28 YAGNI):** string literals containing `/*` or `*/` (e.g., `s := "/*"`) are mis-classified as Comment. Pinned as expected behavior; `TestSplit_StringContainsMarker_KnownLimitation` explicitly documents this.
- **Test coverage:** 12 test functions as specified in build directive. All GREEN for `internal/lang`.

## Unit 5.1 — Round 1

- **Builder:** go-builder-agent
- **Started:** 2026-05-15
- **Files touched:**
  - `internal/lang/lang.go` (new — 170 LOC)
  - `internal/lang/lang_test.go` (new — 130 LOC)
  - `cmd/rak/root.go` (extend `walkAndCount` — 6 lines added, 1 import added)
- **Mage targets run:** `mage test` (pass, all 7 packages), `mage ci` (pass — gofumpt clean, lint 0 issues, all tests green with -race)
- **Notes:**
  - **Shebang — LangShell vs LangBash decision:** `bash` maps to `LangShell` (not a separate `LangBash` constant). Rationale: rak's purpose is code counting by language; shell is shell regardless of dialect. Keeping one `LangShell` constant keeps the language surface minimal and the split grammar table simple. Decision: `LangShell`.
  - **env-indirection in shebang parser:** `#!/usr/bin/env python3` has interpreter path `/usr/bin/env`; basename is `env`, not `python3`. `detectShebang` explicitly handles this: when the basename is `env`, it skips leading `-`-flagged arguments and uses the first non-flag argument as the lookup key. This is required for the `TestDetect_Shebang_Python` case to pass.
  - **`detectedLang` storage pattern:** computed in `walkAndCount` per-iteration local variable (`detectedLang := lang.Detect(f)`) and immediately assigned to `_` (`_ = detectedLang`) with a comment. Chosen over alternative of omitting the call entirely (the plan explicitly requires the wiring in 5.1 so 5.2/5.4 have a stable call-site to build on). The `_ = detectedLang` suppresses the unused-variable compile error while preserving the call-site hook.
  - **Content heuristic (step 4):** XML mapped to `LangHTML` for v0.1.0 (treating XML as a member of the HTML family is a pragmatic simplification; the PLAN.md content heuristic section does not assign XML a separate language constant). This matches the YAGNI principle.
  - **`mage test <pkg>` caveat:** `mage test` runs `./...`; the mage target does not accept package-path arguments. "Exit code 2 / Unknown target" from `mage test github.com/evanmschultz/rak/internal/lang` is expected — the underlying test output shows `ok github.com/evanmschultz/rak/internal/lang`. Used `mage test` + `mage ci` for full verification.

## Unit 5.4 — Round 1

- **Builder:** go-builder-agent
- **Started:** 2026-05-15
- **Files touched:**
  - `cmd/rak/root.go` (add `langs []string` to `rootFlags`; register `--lang` cobra flag via `StringSliceVar`; add `langs []string` param to `runDirectory` and `walkAndCount`; build `wantedLangs map[lang.Language]struct{}` once before the per-file loop; apply filter gate after binary check and `lang.Detect`, before `countFile`; add `strings` to imports)
  - `cmd/rak/root_test.go` (add 6 new test functions: `TestRootCmd_FlagLang_FiltersToGo`, `TestRootCmd_FlagLang_MultiValue`, `TestRootCmd_FlagLang_CaseInsensitive`, `TestRootCmd_FlagLang_ExcludesUnknown`, `TestRootCmd_NoLangFlag_CountsAll`, `TestRootCmd_LangFlag_ParsesCSV`; update `runTreeFS` helper call to pass `flags.langs`)
- **Mage targets run:** requested by orchestrator post-dispatch
- **Design decisions:**
  - **Signature approach:** Passed `langs []string` to both `runDirectory` and `walkAndCount`. `walkAndCount` builds `wantedLangs map[lang.Language]struct{}` internally. Rationale: `walkAndCount` owns aggregation; keeping filter-set construction there avoids leaking `lang` package references into `runDirectory`. `runDirectory` gains only a `langs []string` param to thread through.
  - **Filter map type `map[lang.Language]struct{}`:** More idiomatic Go than `map[lang.Language]bool` for a pure membership set. Zero-size value avoids allocating a bool per entry.
  - **`nil` sentinel for no-filter:** `wantedLangs` is left nil when `len(langs) == 0`. Guard `if wantedLangs != nil` is explicit and avoids unnecessary map allocation for the common no-filter path.
  - **Case normalization:** `lang.Language(strings.ToLower(v))` converts user input (e.g. `"Go"`, `"Rust"`) to lowercase before insertion, matching the lowercase Language constant convention (C6, F29).
  - **F29 LangUnknown implicit exclusion:** `LangUnknown = ""`. A filter like `{"go": {}}` never contains `""`, so `.txt` files (LangUnknown) are implicitly excluded. No explicit `LangUnknown` handling needed.
  - **Filter ordering:** Runs AFTER binary check and `lang.Detect(f)`, BEFORE `countFile`. This ensures: (1) binary files are skipped before Detect (no wasted Detect on binary), (2) the filter is the last skip decision before counting, (3) 5.3's future `lang.Split` call will slot after the filter gate — it only runs on files that pass both binary and lang-filter checks.
  - **`runDirectory` signature:** Added `langs []string` between `binary bool` and `renderer render.Renderer`. All callers updated: `runRoot` (passes `flags.langs`) and `runTreeFS` test helper (passes `flags.langs`).
- **Parallel dispatch note:** 5.2 built concurrently in a separate agent touching `internal/lang/split.go` + `internal/lang/split_test.go` only. This unit touched only `cmd/rak/root.go` + `cmd/rak/root_test.go` with zero file overlap.
