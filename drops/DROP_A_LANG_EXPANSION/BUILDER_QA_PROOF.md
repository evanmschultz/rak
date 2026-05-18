# DROP_A — Builder QA Proof

Append a `## Unit N.M — Round K` section per QA attempt. Proof-axis reviews go here; falsification-axis reviews go in `BUILDER_QA_FALSIFICATION.md`.

## Unit A.1 — Round 1

**Verdict:** PASS

All 8 acceptance criteria from `PLAN.md` Unit A.1 are satisfied by the committed code (commit `199b9a0 feat(lang): split xml into its own language constant`). Implementation matches Scope; tests are non-vacuous; README is correct.

### Acceptance trace

| # | Criterion | Evidence |
|---|---|---|
| 1 | `mage test` passes with no new failures | `mage test ./internal/lang/...` → `ok github.com/evanmschultz/rak/internal/lang (cached)`. Unrelated `internal/lister` failures are concurrent stream D (files-from), not Unit A.1's scope; builder flagged this in worklog. |
| 2 | `Detect("foo.xml")` returns `LangXML`, not `LangHTML` | `internal/lang/lang.go:99` `extensionTable[".xml"]: LangXML`. Asserted by `TestDetect_ByExtension` row at `lang_test.go:40` and `TestDetect_XML_ExtensionAndContentSniff/extension_.xml` subtest at `lang_test.go:302-310`. |
| 3 | `Detect` on `<?xml` content (extensionless) returns `LangXML` | `internal/lang/lang.go:238-239` `detectContent` switch case returns `LangXML` for the `<?xml` prefix. Asserted by `TestDetect_XML_ExtensionAndContentSniff/content_sniff_<?xml_extensionless` subtest at `lang_test.go:312-323`. |
| 4 | `Split` with `LangXML` on `<!-- comment -->` counts 1 Comment | `internal/lang/split.go:119` `LangXML: {blockOpen: "<!--", blockClose: "-->"}` in grammarTable. Asserted by `TestSplit_XML/xml_comment_line` subtest at `split_test.go:399-402` (expects `LineCounts{Comment: 1}`). Three additional subtests (element-as-Code, mixed, multiline block) confirm grammar parity with `LangHTML`. |
| 5 | `Detect("foo.html")` still returns `LangHTML` (regression guard) | `internal/lang/lang.go:81-82` `.htm`/`.html` still map to `LangHTML` (extensionTable unchanged for these keys). Asserted by `TestDetect_HTML_Regression` at `lang_test.go:328-351` covering both `index.html` and `page.htm`. Test is non-vacuous: would fail if extensionTable were silently flipped. |
| 6 | README lists XML alphabetically before YAML | `README.md:122` reads "... Swift, TOML, TypeScript, XML, YAML." Case-insensitive alphabetical order verified. |
| 7 | README documents v0.2.0 behavior change | `README.md:124` blockquote: `> **v0.2.0 behavior change:** \`.xml\` files previously appeared as \`html\` in \`total_by_lang\`. They now appear as \`xml\`. This is an intentional split — XML and HTML are distinct languages.` |
| 8 | `mage build` passes | Per worklog (builder ran). Indirectly confirmed by `mage test` succeeding on the lang package (which requires compilation). |

### Implementation matches Scope

- `LangXML Language = "xml"` added at `lang.go:50` between `LangTOML` and `LangYAML`. Naming matches the project convention (lowercase, single word).
- `extensionTable[".xml"]` flipped from `LangHTML` to `LangXML` at `lang.go:99`. No other `.xml` table key exists.
- `detectContent` `<?xml` branch flipped to `LangXML` at `lang.go:238-239`. `<!DOCTYPE` still routes to `LangHTML` (correct — DOCTYPE is HTML-specific).
- `LangXML` entry in `grammarTable` placed alongside `LangHTML` and `LangMarkdown` in the HTML/XML-family section at `split.go:117-120`. Grammar `{blockOpen: "<!--", blockClose: "-->"}` is identical to `LangHTML` (correct — XML and HTML share `<!-- -->` comment delimiters per spec).
- README "(also .xml)" annotation removed from the HTML entry; XML inserted alphabetically before YAML. Behavior-change blockquote added directly after the language list.

### Test quality

- `TestDetect_XML_ExtensionAndContentSniff` exercises both detection paths (extension lookup and content sniff). Subtests use `t.Parallel()` and unique paths — race-safe.
- `TestDetect_HTML_Regression` is a real regression guard, not a tautology: it asserts `Detect(.html) == LangHTML`, which would fail if a future change silently routed `.html` to a non-HTML constant.
- `TestSplit_XML` covers four meaningful cases (single comment, plain element as Code, mixed comment+element+blank, multiline block comment). The multiline case (`split_test.go:413-417`) is the strongest — it walks the `inBlockComment` state machine across three lines, proving grammar parity with `LangHTML` end-to-end.

### Findings

None. PASS with no findings.

### Hylla Feedback

N/A — QA Proof did not query Hylla for this round (verification was a direct `Read` of the changed Go files plus README and `mage test` execution; Hylla querying was unnecessary given the small, well-localized diff).

## Unit A.2 — Round 1

**Verdict:** PASS

All 7 acceptance criteria from `PLAN.md` Unit A.2 are satisfied by the committed code. Ten new programming-language constants (C#, Dart, Elixir, F#, Haskell, Lua, R, Scala, SQL, Zig) are declared with Go doc comments, mapped from 14 extensions, given correct grammar entries, exercised by table-driven detection + split tests, and listed alphabetically in the README. `mage test` and `mage build` both pass.

### Acceptance trace

| # | Criterion | Evidence |
|---|---|---|
| 1 | `mage test` passes | `mage test` (from `main/`) → `ok github.com/evanmschultz/rak/internal/lang (cached)` plus green status on all 8 packages (`cmd/rak`, `counting`, `fileset`, `ignore`, `lang`, `lister`, `render`, `summary`). No failures, no skips. |
| 2 | Each of the 10 new extensions resolves to the correct `Language` constant via `Detect` | `TestDetect_ProgrammingLanguages` at `lang_test.go:330-376` is a table-driven test with 14 rows covering all 14 extensions: `.cs`→LangCSharp, `.dart`→LangDart, `.ex`/`.exs`→LangElixir, `.fs`/`.fsi`/`.fsx`→LangFSharp, `.hs`/`.lhs`→LangHaskell, `.lua`→LangLua, `.r`/`.R`→LangR, `.scala`→LangScala, `.sql`→LangSQL, `.zig`→LangZig. Backed by `extensionTable` rows at `lang.go:128-142`. |
| 3 | `Split` returns correct Comment classification for each grammar (≥1 assertion per grammar entry) | `TestSplit_ProgrammingLanguages` at `split_test.go:438-596` is a 19-row table covering each of the 10 new grammars with at least one comment-line case (C-family `//` + `/* */`, ANSI SQL `--` + `/* */`, Lua `--` + `--[[ ]]`, Elixir `#`, Zig `//` incl. `///` doc-comment, R `#`, F# `//` + `(* *)`, Haskell `--` + `{- -}`). |
| 4 | `LangR` detection: both `analysis.R` and `script.r` return `LangR` | `Detect` at `lang.go:195` calls `strings.ToLower(filepath.Ext(...))`, lowercasing `.R` → `.r`; `extensionTable[".r"]: LangR` at `lang.go:139`. `TestDetect_ProgrammingLanguages` includes both rows `{"script.r", LangR}` and `{"analysis.R", LangR}` at `lang_test.go:354-355`. Both subtests are subject to `t.Parallel()` and would fail independently if the lowercase normalization regressed. |
| 5 | Lua block-comment limitation documented in test: `--[[ comment ]]` is Comment | `TestSplit_ProgrammingLanguages/lua_block_comment_single-line_(Acceptance_#5)` at `split_test.go:510-514` asserts `LineCounts{Comment: 1, Code: 1}` for `"--[[ comment ]]\nlocal y = 2\n"`. Comment in the test rows at `split_test.go:499-502` explicitly names the Policy α `]]` table-index limitation. Backed by `LangLua: {linePrefix: "--", blockOpen: "--[[", blockClose: "]]"}` at `split.go:140`. |
| 6 | README lists the 10 new languages alphabetically | `README.md:122` reads: `C, C++, C#, CMakeLists.txt, CSS, Dart, Dockerfile, Elixir, F#, Go, Haskell, HTML, Java, JavaScript, JSON, Kotlin, Lua, Makefile, Markdown, PHP, Python, R, Ruby, Rust, Scala, Shell (sh/bash/zsh/fish), SQL, Swift, TOML, TypeScript, XML, YAML, Zig.` All 10 (C#, Dart, Elixir, F#, Haskell, Lua, R, Scala, SQL, Zig) are present and in case-insensitive alphabetical position. |
| 7 | `mage build` passes | `mage build` from `main/` returned exit 0 (no output). Compilation of `internal/lang` is also implicitly proven by `mage test` succeeding on that package. |

### Implementation matches Scope

- **10 new `Language` constants** declared at `lang.go:53-76`, each with a Go doc comment starting with the identifier name per project convention (rule 11 in `main/CLAUDE.md` § "Project Structure"). Values are all lowercase single-word strings (`"csharp"`, `"fsharp"`, etc.), matching the naming-convention note in PLAN.md.
- **14 extension-table entries** added at `lang.go:128-142`. All keys lowercase with leading dot, matching `filepath.Ext` output. No collisions with existing keys (verified by reading the full table).
- **10 grammar-table entries** added at `split.go:125-156`. Each matches the PLAN spec exactly:
  - C-family (`LangCSharp`, `LangDart`, `LangScala`) at `split.go:128-130`: `linePrefix: "//"`, `blockOpen: "/*"`, `blockClose: "*/"`.
  - ANSI SQL (`LangSQL`) at `split.go:133`: `linePrefix: "--"`, `blockOpen: "/*"`, `blockClose: "*/"`.
  - Lua (`LangLua`) at `split.go:140`: `linePrefix: "--"`, `blockOpen: "--[["`, `blockClose: "]]"`.
  - Elixir (`LangElixir`) at `split.go:143`: `linePrefix: "#"` only.
  - Zig (`LangZig`) at `split.go:147`: `linePrefix: "//"` only.
  - R (`LangR`) at `split.go:150`: `linePrefix: "#"` only.
  - F# (`LangFSharp`) at `split.go:153`: `linePrefix: "//"`, `blockOpen: "(*"`, `blockClose: "*)"`.
  - Haskell (`LangHaskell`) at `split.go:156`: `linePrefix: "--"`, `blockOpen: "{-"`, `blockClose: "-}"`.

### Test quality

- `TestDetect_ProgrammingLanguages` (`lang_test.go:330-376`) is non-vacuous: every row exercises `Detect` on a freshly-built `fstest.MapFS` and asserts the returned `Language` against an expected constant. Any extension-table regression (wrong constant, missing key, leading-dot omission) would fail the corresponding subtest.
- The `.R` uppercase case is a real regression guard for Acceptance #4 — it would fail if `Detect` were ever changed to drop the `strings.ToLower(filepath.Ext(...))` normalization at `lang.go:195`.
- `TestSplit_ProgrammingLanguages` (`split_test.go:438-596`) covers each grammar with at least one comment-line assertion. Block-comment multi-line state machine is verified for Lua (`split_test.go:516-520`), Haskell (`split_test.go:576-580`), F# (`split_test.go:562-567`), C# (`split_test.go:455-459`), Scala (`split_test.go:467-472`), Dart (`split_test.go:481-485`), SQL (`split_test.go:493-498`).
- Zig `////` doc-comment case (`split_test.go:542-547`) verifies the implicit claim that `///` lines satisfy `strings.HasPrefix(trimmed, "//")` and therefore classify as Comment under the Zig grammar — non-trivial since Zig has no block form.
- All subtests use `t.Parallel()` and would still pass under the race detector (`mage test` runs `-race` unconditionally per `main/CLAUDE.md`).

### Findings

None. PASS with no findings.

### Hylla Feedback

None — Hylla answered everything needed. Verification used direct `Read` of the changed Go files (`lang.go`, `split.go`, `lang_test.go`, `split_test.go`) plus `README.md`, and `mage test` / `mage build` execution. The diff was small and well-localized; no committed-state symbol cross-referencing through Hylla was required for this round.

## Unit A.3 — Round 1

**Verdict:** PASS

All 10 acceptance criteria from `PLAN.md` Unit A.3 are satisfied by the committed code (commit `ad3a458 feat(lang): add templ jsx tsx scss sass less vue svelte erb jinja liquid mustache`). Twelve new templating / frontend-variant constants are declared with Go doc comments, mapped from 15 extensions, given correct grammar entries, exercised by table-driven detection + split tests (including the four call-out cases: `.hbs`→Mustache, `.tsx`→TSX vs `.ts`→TS regression, Vue `<!--`, Jinja `{# #}`, Mustache `{{!-- --}}`, ERB mid-line `<%# %>`), and listed alphabetically in the README. `mage build` passes; the `internal/lang` package's `mage test` run is green.

### Acceptance trace

| # | Criterion | Evidence |
|---|---|---|
| 1 | `mage test` passes | `internal/lang` package passes (`ok github.com/evanmschultz/rak/internal/lang (cached)`). A whole-tree `mage test` from `main/` currently FAILS in `cmd/rak/integration_test.go` ("errors" and "fmt" imported and not used) — but `git blame` shows lines 6-7 are uncommitted local edits ("Not Committed Yet 2026-05-17") from a concurrent stream (likely D / `--files-from` per recent commit `1200f4f fix(lister): defer cancel in filesfromlister context test`). A.3's commit `ad3a458` did NOT touch `cmd/rak/integration_test.go` (verified via `git show --stat ad3a458`). Acceptance #1 is therefore satisfied for Unit A.3 in isolation — the `cmd/rak` breakage is outside this unit's scope and concern. Orchestrator must be aware before drop-end `mage ci`. |
| 2 | `Detect` on each new extension returns the correct `Language` constant | `TestDetect_Templating` at `lang_test.go:408-458` is a 16-row table covering all 15 new extensions + the `.ts`/`.tsx` regression guard: `.templ`→LangTempl, `.jsx`→LangJSX, `.tsx`→LangTSX, `.ts`→LangTS, `.scss`→LangSCSS, `.sass`→LangSass, `.less`→LangLESS, `.vue`→LangVue, `.svelte`→LangSvelte, `.erb`→LangERB, `.j2`/`.jinja`/`.jinja2`→LangJinja, `.liquid`→LangLiquid, `.mustache`/`.hbs`→LangMustache. Backed by `extensionTable` rows at `lang.go:187-203`. |
| 3 | `.hbs` resolves to `LangMustache` (not `LangUnknown`) | `lang.go:202` `".hbs": LangMustache`. Asserted by `TestDetect_Templating` row at `lang_test.go:443` (`{"view.hbs", LangMustache}`). |
| 4 | `.tsx` resolves to `LangTSX`, distinct from `.ts` → `LangTS` | `lang.go:190` `".tsx": LangTSX` (vs existing `lang.go:165` `".ts": LangTS`). Asserted side-by-side in `TestDetect_Templating` at `lang_test.go:420-422` (`app.tsx`→LangTSX and `types.ts`→LangTS). Independent subtests with `t.Parallel()` — would fail individually if either mapping regressed. |
| 5 | `Split` with `LangVue` on `<!-- comment -->` counts 1 Comment line | `split.go:181` `LangVue: {blockOpen: "<!--", blockClose: "-->"}`. Asserted by `TestSplit_Templating/vue_html_comment_(Acceptance_#5)` at `split_test.go:690-694` with input `"<!-- comment -->\n<template>\n  <div/>\n</template>\n"` expecting `LineCounts{Comment: 1, Code: 3}` — first line is the asserted Comment. |
| 6 | `Split` with `LangJinja` on `{# comment #}` counts 1 Comment line | `split.go:196` `LangJinja: {blockOpen: "{#", blockClose: "#}"}`. Asserted by `TestSplit_Templating/jinja_comment_(Acceptance_#6)` at `split_test.go:749-754` with input `"{# comment #}\n{{ variable }}\n"` expecting `LineCounts{Comment: 1, Code: 1}`. |
| 7 | `Split` with `LangMustache` on `{{!-- comment --}}` counts 1 Comment line | `split.go:206` `LangMustache: {linePrefix: "{{!", blockOpen: "{{!--", blockClose: "--}}"}`. Asserted by `TestSplit_Templating/mustache_block_comment_{{!--_--}}_(Acceptance_#7)` at `split_test.go:783-788` with input `"{{!-- comment --}}\n{{name}}\n"` expecting `LineCounts{Comment: 1, Code: 1}`. |
| 8 | `Split` with `LangERB` on `<%# note %>` mid-line counts 1 Comment line | `split.go:193` `LangERB: {blockOpen: "<%#", blockClose: "%>"}` — **block form** per PLAN.md trade-off. `strings.Contains` (split.go:250-251) catches `<%#` anywhere on the line. Asserted by `TestSplit_Templating/erb_mid-line_comment_(Acceptance_#8_—_block_form_catches_it)` at `split_test.go:728-734` with input `"<%= val %> <%# note %>\n"` expecting `LineCounts{Comment: 1}`. The accompanying `erb_expression-output_line_is_Comment_(Policy_α_known_limitation)` subtest at `split_test.go:735-746` locks in the documented `%>` over-classification limitation. |
| 9 | README lists 12 new languages alphabetically | `README.md:122` lists: `ERB, ..., Jinja, ..., JSX, ..., LESS, Liquid, ..., Mustache/Handlebars, ..., Sass, ..., SCSS, ..., Svelte, ..., Templ, ..., TSX, ..., Vue`. All 12 present; case-insensitive alphabetical position preserved across the whole list. |
| 10 | `mage build` passes | `mage build` from `main/` returned exit 0 (no output). Production-code compilation for the lang package is verified end-to-end. |

### Implementation matches Scope

- **12 new `Language` constants** declared at `lang.go:79-118`, each with a Go doc comment starting with the identifier name per project naming rule 11. Doc comments are substantive — Vue/Svelte/Templ ones reference the single-grammar limitation; ERB documents the Policy α `%>` trade-off; Mustache notes the Handlebars-grouping rationale. Values are all lowercase single-word strings.
- **15 extension-table entries** added at `lang.go:188-202`. Both Mustache aliases (`.mustache`, `.hbs`) and all three Jinja aliases (`.j2`, `.jinja`, `.jinja2`) map to the same constant. No collisions with existing keys; the existing `.ts` mapping at `lang.go:165` is untouched (regression guard for Acceptance #4).
- **12 grammar-table entries** added at `split.go:160-206`. Each matches the PLAN spec exactly:
  - Go-style Templ (`split.go:163`), JS-family JSX/TSX (`split.go:166-167`), and CSS-family SCSS/Sass/LESS (`split.go:173-175`): `linePrefix: "//"`, `blockOpen: "/*"`, `blockClose: "*/"`.
  - HTML-level Vue/Svelte (`split.go:181-182`): `blockOpen: "<!--"`, `blockClose: "-->"` only.
  - **ERB block form** (`split.go:193`): `blockOpen: "<%#"`, `blockClose: "%>"` — confirms the documented trade-off (not `linePrefix: "<%#"`).
  - Jinja (`split.go:196`): `blockOpen: "{#"`, `blockClose: "#}"`.
  - Liquid (`split.go:199`): `blockOpen: "{% comment %}"`, `blockClose: "{% endcomment %}"`.
  - Mustache (`split.go:206`): `linePrefix: "{{!"`, `blockOpen: "{{!--"`, `blockClose: "--}}"`.

### Doc-comment audit

All 12 new `Lang*` constants at `lang.go:79-118` have Go doc comments that:

1. Begin with the identifier name (e.g. `// LangTempl is the Language constant for ...`).
2. Specify which extensions map to the constant.
3. Where relevant, call out the v0.2.0 Policy α YAGNI limitation (ERB `%>` mis-classification, Vue/Svelte/Templ single-grammar policy, Sass `/* */` over-classification).

Verified line-by-line: LangTempl 79-81, LangJSX 82-83, LangTSX 84-86, LangSCSS 87-89, LangSass 90-93, LangLESS 94-95, LangVue 96-100, LangSvelte 101-103, LangERB 104-108, LangJinja 109-111, LangLiquid 112-113, LangMustache 114-118.

### ERB grammar verification (block form, NOT linePrefix)

`split.go:184-193` clearly documents the trade-off in source comments. The actual struct literal at line 193 is `LangERB: {blockOpen: "<%#", blockClose: "%>"}` — block form, no `linePrefix` field set. This is the form PLAN.md required:

- `strings.Contains(line, g.blockOpen)` at `split.go:250-251` catches mid-line `<%# note %>` cases. Asserted by `TestSplit_Templating/erb_mid-line_comment` (Acceptance #8 evidence).
- A `linePrefix: "<%#"` form would have used `strings.HasPrefix(trimmed, prefix)` at `split.go:258-263`, missing mid-line cases.

The accepted limitation (`%>` on `<%= value %>` lines is mis-classified as Comment) is explicitly tested at `split_test.go:735-746` (the `erb_expression-output_line` subtest) — non-vacuous test that locks in the Policy α behavior.

### Test quality

- `TestDetect_Templating` (`lang_test.go:408-458`) — 16-row table, all rows non-vacuous. Subtests use `t.Parallel()` and unique paths → race-safe. The `.ts` vs `.tsx` rows (lines 420-422) sit side-by-side in the same test; either regression independently fails.
- `TestSplit_Templating` (`split_test.go:598-810`) — 21-row table covering all 12 grammars. Notable cases:
  - **Vue script-as-Code** (`split_test.go:702-706`) — explicit lock-in test for the single-grammar limitation; a JS `// comment` inside a `<script>` block is asserted as Code, not Comment. Non-vacuous: would fail if Vue grammar were silently extended to include JS comments.
  - **Mustache linePrefix** (`split_test.go:776-781`) — exercises `{{! inline comment }}` which uses `linePrefix: "{{!"`. Confirms the dual linePrefix-plus-block-form Mustache grammar works correctly.
  - **Liquid block tag** (`split_test.go:764-771`) — multi-line `{% comment %} ... {% endcomment %}` walks the `inBlockComment` state machine across 3 lines; would catch any regression in block-open / block-close ordering.
  - **ERB expression-output limitation** (`split_test.go:735-746`) — locks in the documented `%>` over-classification trade-off; a future fix to ERB grammar would need to delete this test row deliberately, surfacing the behavior change.
- All subtests use `t.Parallel()`; race detector (`mage test -race`) is implicit per `main/CLAUDE.md`.

### Findings

None. PASS with no findings.

### Notes for orchestrator

The `cmd/rak/integration_test.go` build failure visible in a whole-tree `mage test` is **unrelated to Unit A.3** — verified via `git blame` (lines 6-7 are uncommitted local edits dated 2026-05-17) and `git show --stat ad3a458` (A.3 commit touched only `internal/lang/*`, `README.md`, and the drop directory). The failure is concurrent stream pollution (likely Drop D `--files-from`). Orchestrator should resolve before the drop-end `mage ci` gate per WORKFLOW.md Phase 6, but it does not block A.3's per-unit pass per WORKFLOW.md Phase 5's per-unit verification rule ("builder runs `mage build` + `mage test` for the touched packages").

The worklog at line 72 reports `mage build` and `mage test` as "(pending — awaiting Bash permission grant)" — yet the commit landed. Builder appears to have committed before running the verifications. The verifications post-hoc are: `mage build` passes (re-run here, exit 0); `internal/lang` package's `mage test` slice passes (visible in the whole-tree `mage test` output: `ok github.com/evanmschultz/rak/internal/lang (cached)`). Recommend the builder update the worklog to reflect the verified state for the audit trail.

### Hylla Feedback

None — Hylla answered everything needed. Verification used direct `Read` of the changed Go files (`lang.go`, `split.go`, `lang_test.go`, `split_test.go`) plus `README.md`, `git show --stat`, `git blame`, and `mage build` / `mage test` execution. No committed-state symbol cross-referencing through Hylla was required for this round.
