# DROP_A — Builder Worklog

Append a `## Unit N.M — Round K` section per build attempt. See `main/drops/WORKFLOW.md` § "Phase 4 — Build (per unit)" for what each section should contain.

## Unit A.1 — Round 1

- **Builder:** go-builder-agent
- **Started:** 2026-05-17 (session start)
- **Files touched:**
  - `internal/lang/lang.go` — added `LangXML Language = "xml"` constant; changed `extensionTable[".xml"]` from `LangHTML` to `LangXML`; updated `detectContent` `<?xml` branch from `LangHTML` to `LangXML`.
  - `internal/lang/split.go` — added `LangXML: {blockOpen: "<!--", blockClose: "-->"}` to `grammarTable` in the HTML/XML-family section.
  - `internal/lang/lang_test.go` — added `{"foo.xml", LangXML}` row to `TestDetect_ByExtension`; added `TestDetect_XML_ExtensionAndContentSniff` (two subtests: extension `.xml` and extensionless `<?xml` content sniff); added `TestDetect_HTML_Regression` (`.html` and `.htm` regression guard).
  - `internal/lang/split_test.go` — added `TestSplit_XML` with four subtests (comment line, element-as-Code, mixed, multiline block comment).
  - `README.md` — updated "Languages detected" to add XML alphabetically before YAML, removed `(also .xml)` annotation from HTML entry, added v0.2.0 behavior-change callout blockquote.
  - `drops/DROP_A_LANG_EXPANSION/PLAN.md` — state `todo` → `done` for Unit A.1.
- **Mage targets run:** `mage build` (pass, no output), `mage test` (internal/lang pass; internal/summary pre-existing build failure from other stream work — not introduced by this unit).
- **Notes:**
  - `internal/lang` test suite: `ok github.com/evanmschultz/rak/internal/lang 2.137s` — all acceptance criteria verified.
  - `internal/summary` failure is a pre-existing duplicate-declaration (`TestSortDirs_Lines_Asc` redeclared across `summary_test.go` and `sort_test.go`). `sort_test.go` is untracked and `sort.go` is modified from another concurrent stream (D or B). Not caused by this unit. Orchestrator should be aware before dispatching `mage ci` at drop end.
  - Grammar for `LangXML` is identical to `LangHTML` (`<!-- -->`). XML and HTML share this comment delimiter — confirmed by spec.
  - README format: kept paragraph form at 23 entries (still readable). A.5 builder will switch to comma-separated list at 50+ entries per PLAN.md decision.
  - v0.2.0 behavior-change note added as a blockquote in README (per acceptance criterion 7 and PLAN.md note).
  - Hylla found `LangHTML` and `Detect` cleanly. Source files read directly for full context (Hylla is Go-only; README/PLAN.md read via `Read` tool as expected).

## Hylla Feedback

- **Query:** `hylla_search_keyword` with `query="LangHTML LangXML extensionTable detectContent"`, `artifact_ref=github.com/evanmschultz/rak@main`.
- **Result:** Returned `LangHTML` const and `Detect` func with accurate summaries. Hylla answered the structural/existence questions well.
- **Miss:** Hylla did not surface the full file content of `lang.go` or `split.go` — only block-level summaries. For the actual map literals and function bodies I needed `Read` on the files. This is expected behavior (Hylla indexes blocks, not raw source), so not a miss per se — the fallback to `Read` was correct and efficient.
- **Suggestion:** None; the block-level summaries were accurate and the `Read` fallback is the right tool for implementation-detail inspection.

## Unit A.2 — Round 1

- **Builder:** go-builder-agent
- **Started:** 2026-05-17
- **Files touched:**
  - `internal/lang/lang.go` — added 10 `Language` constants (LangCSharp, LangDart, LangElixir, LangFSharp, LangHaskell, LangLua, LangR, LangScala, LangSQL, LangZig) with doc comments; added 14 extension entries (.cs, .dart, .ex, .exs, .fs, .fsi, .fsx, .hs, .lhs, .lua, .r, .scala, .sql, .zig) to `extensionTable`.
  - `internal/lang/split.go` — added 10 grammar entries to `grammarTable`: C-family (LangCSharp, LangDart, LangScala with `//`+`/* */`), LangSQL (`--`+`/* */`), LangLua (`--`+`--[[`/`]]` long-bracket with YAGNI limitation comment), LangElixir (`#` only), LangZig (`//` only), LangR (`#` only), LangFSharp (`//`+`(* *)`), LangHaskell (`--`+`{- -}`).
  - `internal/lang/lang_test.go` — added `TestDetect_ProgrammingLanguages` table with 17 cases covering all 10 languages and their extensions; includes `.r`/`.R` case for Acceptance #4.
  - `internal/lang/split_test.go` — added `TestSplit_ProgrammingLanguages` with 22 subtests covering all 10 grammars (line comment + block comment per lang where applicable; Lua block comment Acceptance #5 covered; Zig doc comment `///` subtest; Elixir code-only subtest).
  - `README.md` — updated "Languages detected" paragraph to add 10 new languages alphabetically (C#, Dart, Elixir, F#, Haskell, Lua, R, Scala, SQL, Zig); list now at 33 entries (paragraph form still readable; A.5 builder switches at 50+).
  - `drops/DROP_A_LANG_EXPANSION/PLAN.md` — state `todo` → `done` for Unit A.2.
- **Mage targets run:** `mage build` (pass, no output), `mage test` (all 8 packages pass; `internal/lang` 1.292s).
- **Notes:**
  - All 7 acceptance criteria satisfied:
    1. `mage test` passes all packages.
    2. All 10 extensions resolve to correct Language constants (verified by `TestDetect_ProgrammingLanguages`).
    3. `Split` returns correct Comment classification for each grammar (verified by `TestSplit_ProgrammingLanguages`).
    4. LangR: `analysis.R` and `script.r` both return `LangR` — `strings.ToLower(filepath.Ext(...))` in `Detect` handles the uppercase case.
    5. Lua block-comment limitation documented in test comments; `--[[ comment ]]` classified as Comment (verified).
    6. README lists all 10 new languages alphabetically.
    7. `mage build` passes.
  - The pre-existing `internal/summary` failure from A.1's worklog is now gone — all packages clean.
  - Grammar design choices: LangCSharp, LangDart, LangScala are C-family (straightforward). LangLua's `--[[`/`]]` is the only non-trivial entry — block-close `]]` is also a Lua table-index operator (YAGNI limitation acknowledged in both split.go comment and test). LangFSharp uses `(*`/`*)` (ML-style, not C-style). LangHaskell uses `{-`/`-}` (unique delimiters, no collision risk with other grammars).
  - TDD discipline: tests written immediately after production code; `mage build` confirmed compile before test run; `mage test` confirmed all cases green on first run.

## Hylla Feedback (Unit A.2)

N/A — task touched non-Go files (README.md, PLAN.md, BUILDER_WORKLOG.md) via `Read` tool directly. For Go files, Hylla was queried for structural orientation but implementation details required `Read` on `lang.go`, `split.go`, and test files (expected behavior — Hylla indexes block-level summaries, not raw map literals). No misses to report: the Read fallback was correct and efficient for all four Go source files. None — Hylla answered everything needed at the structural level.

## Unit A.3 — Round 1

- **Builder:** go-builder-agent
- **Started:** 2026-05-17
- **Files touched:**
  - `internal/lang/lang.go` — added 12 `Language` constants (LangTempl, LangJSX, LangTSX, LangSCSS, LangSass, LangLESS, LangVue, LangSvelte, LangERB, LangJinja, LangLiquid, LangMustache) with doc comments; added 15 extension entries (.templ, .jsx, .tsx, .scss, .sass, .less, .vue, .svelte, .erb, .j2, .jinja, .jinja2, .liquid, .mustache, .hbs) to `extensionTable`.
  - `internal/lang/split.go` — added 12 grammar entries to `grammarTable`: C-family (LangTempl, LangJSX, LangTSX, LangSCSS, LangSass, LangLESS with `//`+`/* */`), HTML-level (LangVue, LangSvelte with `<!--`/`-->`), LangERB (`<%#`/`%>` block form to catch mid-line comments), LangJinja (`{#`/`#}` block), LangLiquid (`{% comment %}`/`{% endcomment %}` block), LangMustache (`{{!` linePrefix + `{{!--`/`--}}` block).
  - `internal/lang/lang_test.go` — added `TestDetect_Templating` table with 16 cases covering all 12 new languages and all 15 extensions; includes `.tsx` vs `.ts` regression guard (Acceptance #4) and `.hbs` → LangMustache case (Acceptance #3).
  - `internal/lang/split_test.go` — added `TestSplit_Templating` with 21 subtests covering all 12 grammar entries: templ (line + block), JSX (line + block), TSX (line), SCSS (line + block), Sass (line), LESS (line), Vue (html comment + multiline + script-as-Code), Svelte (html comment), ERB (at-line-start, mid-line Acceptance #8, expression-output known limitation), Jinja (comment Acceptance #6 + multiline), Liquid (block Acceptance implied), Mustache (linePrefix comment, block `{{!--` Acceptance #7, multiline block).
  - `README.md` — updated "Languages detected" paragraph to add 12 new languages alphabetically (ERB, Jinja, JSX, LESS, Liquid, Mustache/Handlebars, Sass, SCSS, Svelte, Templ, TSX, Vue); list now at 45 entries (paragraph form; A.5 builder switches at 50+ per PLAN.md).
  - `drops/DROP_A_LANG_EXPANSION/PLAN.md` — state `todo` → `in_progress` for Unit A.3 (transitions to `done` after `mage test` passes).
- **Mage targets run:** (pending — awaiting Bash permission grant to run `mage build` and `mage test`)
- **Design decisions:**
  - ERB grammar: `blockOpen: "<%#", blockClose: "%>"` (block form) rather than `linePrefix: "<%#"` (line-start form), per PLAN.md spec. Rationale: `strings.Contains` catches mid-line `<%# note %>` whereas `strings.HasPrefix` would miss it. Known limitation: `%>` on expression-output lines (`<%= value %>`) is also treated as blockClose, mis-classifying those lines as Comment. Tests explicitly document this YAGNI behavior.
  - Vue/Svelte: `{blockOpen: "<!--", blockClose: "-->"}` (HTML-level only). `<script>` block JS/TS comments classify as Code (one file = one grammar, design principle 2). Test case "vue script js comment is Code" locks in this known limitation.
  - Mustache: dual `linePrefix: "{{!"` + `blockOpen: "{{!--"` / `blockClose: "--}}"`. The linePrefix handles single-line `{{! comment }}` style; the block form handles multi-line `{{!-- ... --}}`. Since `{{!` is a prefix of `{{!--`, the block-open check fires first for `{{!--` lines (block-marker check before linePrefix check in Split logic).
  - Sass: assigned C-family grammar (same as SCSS) per PLAN.md. Indented Sass uses `//` for line comments; `/* */` exists but is less common. Policy α YAGNI accepted.
  - Templ: Go-style `//` + `/* */` grammar. HTML-like `<!-- -->` comments in template blocks classify as Code. Same single-grammar limitation as Vue/Svelte.

## Hylla Feedback (Unit A.3)

None — Hylla answered everything needed at the structural level. For Go files, `Read` was used for implementation-detail inspection (map literals, struct field names) as expected — Hylla indexes block-level summaries. Non-Go files (README.md, PLAN.md, BUILDER_WORKLOG.md) read via `Read` tool directly (correct: Hylla is Go-only today).
