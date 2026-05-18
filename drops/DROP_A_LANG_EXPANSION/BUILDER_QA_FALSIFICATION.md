# DROP_A — Build QA Falsification

Append a `## Unit N.M — Round K` section per QA attempt. See `main/drops/WORKFLOW.md` § "Phase 5 — Build QA (per unit)".

## Unit A.1 — Round 1

**Verdict:** PASS

### Counterexamples / Attacks

#### Attack 1 — `.XML` / `.Xml` case sensitivity

- **Severity:** nit (REFUTED)
- **Where:** `internal/lang/lang.go:155` (`Detect` step 2)
- **Counterexample / hypothesis:** Builder might have introduced an exact-case map lookup that breaks `foo.XML` or `foo.Xml`.
- **Mitigation accepted:** REFUTED. `Detect` calls `strings.ToLower(filepath.Ext(f.RelPath))` before the table lookup (lang.go:155). `.XML`, `.Xml`, `.xml` all normalize to `.xml` and hit `extensionTable[".xml"] → LangXML`. No regression.

#### Attack 2 — `<?xml-stylesheet` and other `<?xml`-prefixed processing instructions

- **Severity:** nit (REFUTED)
- **Where:** `internal/lang/lang.go:238` (`detectContent`)
- **Counterexample / hypothesis:** A file beginning with `<?xml-stylesheet href="..."?>` (a real XML processing instruction with hyphen after `xml`) might be wrongly classified.
- **Mitigation accepted:** REFUTED. `bytes.HasPrefix(trimmed, []byte("<?xml"))` matches `<?xml-stylesheet…` and returns `LangXML`. That is the correct answer — XSLT stylesheets and other PI-prefixed content are valid XML. Behavior matches intent.

#### Attack 3 — HTML detection regression

- **Severity:** blocker (REFUTED)
- **Where:** `internal/lang/lang.go:81-82` (extensionTable), `internal/lang/lang.go:240` (detectContent `<!DOCTYPE`), `internal/lang/lang_test.go:328-351` (TestDetect_HTML_Regression)
- **Counterexample / hypothesis:** Builder might have accidentally removed `.html` / `.htm` → `LangHTML` mappings, or flipped `<!DOCTYPE` content sniff to `LangXML`.
- **Mitigation accepted:** REFUTED. `extensionTable[".html"] = LangHTML` and `extensionTable[".htm"] = LangHTML` still present (lang.go:81-82). `detectContent`'s `<!DOCTYPE` branch still returns `LangHTML` (lang.go:241). TestDetect_HTML_Regression exercises both extensions and would fail-fast on regression — extension lookup beats content sniff in priority, so the test is non-vacuous even though content sniff path isn't exercised.

#### Attack 4 — Grammar duplication / shared `<!-- -->` correctness

- **Severity:** concern (REFUTED)
- **Where:** `internal/lang/split.go:118-119` (grammarTable)
- **Counterexample / hypothesis:** Builder might have used a different comment grammar for `LangXML` than `LangHTML`, despite spec saying they share `<!-- -->`.
- **Mitigation accepted:** REFUTED. `LangXML: {blockOpen: "<!--", blockClose: "-->"}` matches `LangHTML: {blockOpen: "<!--", blockClose: "-->"}` exactly (split.go:118-119). Confirmed identical grammar; LangMarkdown also shares the same grammar in the same block. TestSplit_XML covers comment-line, element-as-Code, mixed, and multi-line block cases — adequate.

#### Attack 5 — TestDetect_HTML_Regression vacuity

- **Severity:** concern (REFUTED)
- **Where:** `internal/lang/lang_test.go:328-351`
- **Counterexample / hypothesis:** If the test exercised only content sniff (extensionless `<!DOCTYPE` files), changing `extensionTable[".html"]` would silently pass.
- **Mitigation accepted:** REFUTED. The test uses files named `index.html` and `page.htm` — extension lookup (step 2) wins before content sniff (step 4) per the pipeline. Mutating `extensionTable[".html"]` to anything other than `LangHTML` would break the test immediately. The content body (`<!DOCTYPE html>`) is incidental — the extension drives the verdict.

#### Attack 6 — Empty / minimal `<?xml` content edge case

- **Severity:** nit (REFUTED)
- **Where:** `internal/lang/lang.go:229-232`
- **Counterexample / hypothesis:** A near-empty file containing only `<?xml` might crash or return wrong language.
- **Mitigation accepted:** REFUTED. `detectContent` guards `len(buf) == 0` and returns `LangUnknown`; for non-empty buffers, `bytes.TrimSpace` + `HasPrefix` is total — no slice indexing past length. Tested via TestDetect_XML_ExtensionAndContentSniff's extensionless `<?xml` subtest.

#### Attack 7 — `<?xml ... ?>` followed by HTML-ish content

- **Severity:** nit (REFUTED)
- **Where:** `internal/lang/lang.go:237-246`
- **Counterexample / hypothesis:** A file starting with `<?xml ... ?>\n<html>...</html>` (XHTML-style) — should XML or HTML win?
- **Mitigation accepted:** REFUTED by spec. `detectContent` is first-match (switch on prefix order: `<?xml` checked before `<!DOCTYPE`). `<?xml`-prefixed content returns `LangXML`. This matches the design — files declared as XML at the prolog are XML, regardless of inner document type. Conventional XHTML detection by extension (`.xhtml`) is out of scope and not in `extensionTable` either pre- or post-A.1.

#### Attack 8 — README accuracy (alphabetical position, v0.2.0 note, `total_by_lang` mention)

- **Severity:** concern (REFUTED)
- **Where:** `README.md:120-124`
- **Counterexample / hypothesis:** README might list XML in the wrong alphabetical position, or the v0.2.0 behavior note might omit the explicit `total_by_lang` reference required by acceptance criterion 7.
- **Mitigation accepted:** REFUTED. Languages-detected line is "...Swift, TOML, TypeScript, XML, YAML" — XML correctly placed between TypeScript and YAML (T < X < Y). The blockquote (line 124) explicitly says ".xml files previously appeared as `html` in `total_by_lang`. They now appear as `xml`." — `total_by_lang` mentioned by name, intent flagged as intentional v0.2.0 change. Acceptance criteria 6 + 7 satisfied.

#### Attack 9 — Leftover `LangHTML` reference where `LangXML` belongs

- **Severity:** blocker (REFUTED)
- **Where:** `internal/lang/lang.go`, `internal/lang/split.go`
- **Counterexample / hypothesis:** Builder may have missed a `LangHTML` site that should now route to `LangXML`.
- **Mitigation accepted:** REFUTED. Reviewed both files line by line. The only sites tied to XML are: (a) the new `LangXML = "xml"` constant (lang.go:50), (b) `extensionTable[".xml"] = LangXML` (lang.go:99), (c) `detectContent` `<?xml` branch returning `LangXML` (lang.go:239), (d) `grammarTable[LangXML]` entry (split.go:119). All four flip cleanly from HTML to XML. Remaining `LangHTML` references (`extensionTable[".htm"]`, `extensionTable[".html"]`, `detectContent` `<!DOCTYPE` branch, `grammarTable[LangHTML]`) correctly stay on HTML.

### Informational note (not a counterexample against A.1)

- Worklog (line 16, 18-19) flags a pre-existing `internal/summary` package build failure (`TestSortDirs_Lines_Asc` redeclared across `summary_test.go` and `sort_test.go`) caused by an untracked `sort_test.go` from a concurrent stream — NOT introduced by Unit A.1. Surface to orchestrator before drop-end `mage ci` (Phase 6) so it does not get attributed to A.1.

### Summary

All nine attack vectors REFUTED. Implementation matches plan, tests are non-vacuous, README is accurate, no leftover HTML/XML references, no edge case crashes. PASS.
