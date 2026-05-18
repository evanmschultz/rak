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
