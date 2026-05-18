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
