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

## Unit A.2 — Round 1

**Verdict:** PASS

### Counterexamples / Attacks

#### Attack 1 — Extension collisions in `extensionTable`

- **Severity:** blocker (REFUTED)
- **Where:** `internal/lang/lang.go:93-143` (`extensionTable`)
- **Counterexample / hypothesis:** Any of the 14 new extensions (`.cs`, `.scala`, `.lua`, `.sql`, `.dart`, `.ex`, `.exs`, `.zig`, `.r`, `.fs`, `.fsi`, `.fsx`, `.hs`, `.lhs`) might clobber an existing entry. `.r`/`.cs`/`.fs` are the highest-risk single-letter candidates.
- **Mitigation accepted:** REFUTED. The pre-A.2 `extensionTable` block (lang.go:94-126) contains: `.bash, .c, .cc, .cpp, .css, .cxx, .fish, .gemspec, .go, .h, .hpp, .htm, .html, .java, .js, .json, .kt, .kts, .md, .php, .phtml, .py, .rake, .rb, .rs, .sh, .swift, .toml, .ts, .xml, .yaml, .yml, .zsh`. None of `.cs`, `.scala`, `.lua`, `.sql`, `.dart`, `.ex`, `.exs`, `.zig`, `.r`, `.fs`, `.fsi`, `.fsx`, `.hs`, `.lhs` appear. The new A.2 block (lang.go:128-143) adds each exactly once. No duplicate keys; Go map literal would fail compile on duplicate keys at the same line range anyway, and `mage build` passed. Concretely: `.fs` is not `.fish` (full word), `.cs` is not `.css` (full word), `.r` is not `.rb` / `.rake` / `.rs` (single letter only).

#### Attack 2 — Lua `]]` state-machine known limitation acknowledgement

- **Severity:** concern (REFUTED)
- **Where:** `internal/lang/split.go:135-140`, `internal/lang/split_test.go:499-520` (Lua test subtests)
- **Counterexample / hypothesis:** PLAN.md Notes calls out that `]]` block-close can corrupt state when `]]` appears inside a string literal or as `table[idx]]`. The test might be silent about this known limitation, making the YAGNI cut invisible to a future maintainer.
- **Mitigation accepted:** REFUTED. `split.go:135-140` contains a block comment explicitly documenting the limitation: *"Known limitation (Policy α YAGNI): `]]` also appears as a table-index operator in Lua code. Lines containing `]]` in code context are mis-classified as Comment. Additionally, `]]` inside string literals can corrupt multi-line block-comment state. Accepted under F28 YAGNI."* In `split_test.go:500-503`, the test subtest comment for Lua also restates the limitation. The Lua test asserts `--[[ comment ]]` is classified as Comment (Acceptance #5) and exercises a multi-line `--[[ \n line two \n ]]` block. Both the implementation and the test correctly acknowledge the known limitation — no surprise for future maintainers.

#### Attack 3 — `.R` uppercase case lowering

- **Severity:** blocker (REFUTED)
- **Where:** `internal/lang/lang.go:195` (`Detect` step 2), `internal/lang/lang_test.go:354-355` (TestDetect_ProgrammingLanguages rows)
- **Counterexample / hypothesis:** If `Detect` does not call `strings.ToLower` on the result of `filepath.Ext`, then `analysis.R` would miss the `.r` key and return `LangUnknown`. Test might not exercise the uppercase case.
- **Mitigation accepted:** REFUTED. `lang.go:195` reads `ext := strings.ToLower(filepath.Ext(f.RelPath))` before the table lookup. Both `.R` (raw `filepath.Ext` returns `.R`) and `.r` lowercase to `.r`. `TestDetect_ProgrammingLanguages` rows at `lang_test.go:354-355` explicitly cover both `script.r` (lowercase) and `analysis.R` (uppercase), asserting both return `LangR`. Acceptance #4 satisfied.

#### Attack 4 — Elixir `.ex` / `.exs` distinctness

- **Severity:** nit (REFUTED)
- **Where:** `internal/lang/lang.go:131-132`, `internal/lang/lang_test.go:342-343`
- **Counterexample / hypothesis:** Either of `.ex` or `.exs` might be missing, or one might accidentally map to a different language.
- **Mitigation accepted:** REFUTED. `extensionTable` has both `.ex → LangElixir` and `.exs → LangElixir` (lang.go:131-132). Test exercises both (`app.ex`, `config.exs`). No pre-A.2 entry uses either extension. No collision risk: `.exs` is a 3-char extension distinct from any 2-char `.ex` lookup (Go maps are exact-key, not prefix-match).

#### Attack 5 — F# triplet `.fs` / `.fsi` / `.fsx`

- **Severity:** nit (REFUTED)
- **Where:** `internal/lang/lang.go:133-135`, `internal/lang/lang_test.go:345-347`
- **Counterexample / hypothesis:** Any of the three F# extensions might be missing.
- **Mitigation accepted:** REFUTED. All three extensions present in `extensionTable` mapping to `LangFSharp`: `.fs` (lang.go:133), `.fsi` (lang.go:134), `.fsx` (lang.go:135). Test covers all three (`module.fs`, `iface.fsi`, `script.fsx`). No collision with pre-A.2 entries (`.fish` is the full word, not a prefix; map keys are exact).

#### Attack 6 — Test vacuity (blank+comment+code combinations)

- **Severity:** concern (REFUTED)
- **Where:** `internal/lang/split_test.go:438-596` (TestSplit_ProgrammingLanguages)
- **Counterexample / hypothesis:** New TestSplit cases might only test "one line is a comment" without exercising the three-way blank/comment/code split required by Acceptance #3.
- **Mitigation accepted:** REFUTED (with one observed gap not load-bearing). Most subtests assert `{Blank: 0, Comment: 1, Code: 1}` — exercising the comment-and-code split. The C-family block subtests (csharp/scala/dart block comment) assert `{Blank: 0, Comment: 3, Code: 1}` — exercising the multi-line block-comment state machine plus code. Lua multi-line block also asserts `{Blank: 0, Comment: 3, Code: 1}`. **Gap (not blocker):** none of the 22 subtests asserts a non-zero Blank count for the new languages (e.g. `\n// comment\n\ncode\n` → `{Blank: 1, Comment: 1, Code: 1}`). The Blank classification path is shared with all other languages (split.go:186-189 `trimmed == ""` branch is grammar-agnostic) and is already exercised by `TestSplit_GoSimple` (split_test.go:12-24). Re-asserting Blank per new language would be belt-and-suspenders. Acceptable; not a true gap given shared codepath. Acceptance #3 — "at minimum one assertion per grammar entry" — is satisfied (22 subtests for 10 grammars).

#### Attack 7 — Doc comments on new exported constants

- **Severity:** blocker (REFUTED)
- **Where:** `internal/lang/lang.go:55-75`
- **Counterexample / hypothesis:** Any of the 10 new `Lang*` constants might be missing a doc comment, or have a malformed one (starts lowercase, missing identifier prefix, etc.) — which would fail `golangci-lint`'s `revive`/`staticcheck` `exported` rule.
- **Mitigation accepted:** REFUTED. All 10 new constants have well-formed doc comments per Go style:
  - `// LangCSharp is the Language constant for C# source files (.cs).` (line 55)
  - `// LangDart is the Language constant for Dart source files (.dart).` (line 57)
  - `// LangElixir is the Language constant for Elixir source files (.ex, .exs).` (line 59)
  - `// LangFSharp is the Language constant for F# source files (.fs, .fsi, .fsx).` (line 61)
  - `// LangHaskell is the Language constant for Haskell source files (.hs, .lhs).` (line 63)
  - `// LangLua is the Language constant for Lua source files (.lua).` (line 65)
  - `// LangR is the Language constant for R source files (.r — filepath.Ext lowercases, so both .r and .R files map here via strings.ToLower in Detect).` (line 67)
  - `// LangScala is the Language constant for Scala source files (.scala).` (line 70)
  - `// LangSQL is the Language constant for SQL source files (.sql).` (line 72)
  - `// LangZig is the Language constant for Zig source files (.zig).` (line 74)
  Each starts with `// LangX` (capitalized, matches identifier). All conform to `golint` exported-comment rule.

#### Attack 8 — README alphabetical accuracy + naming convention

- **Severity:** concern (REFUTED)
- **Where:** `README.md:122`
- **Counterexample / hypothesis:** The 10 new entries may be out of alphabetical order, may use inconsistent naming (e.g. "C#" vs "CSharp" vs "C-Sharp"), or may not match the constant-naming convention.
- **Mitigation accepted:** REFUTED. README line 122 reads: `"C, C++, C#, CMakeLists.txt, CSS, Dart, Dockerfile, Elixir, F#, Go, Haskell, HTML, Java, JavaScript, JSON, Kotlin, Lua, Makefile, Markdown, PHP, Python, R, Ruby, Rust, Scala, Shell (sh/bash/zsh/fish), SQL, Swift, TOML, TypeScript, XML, YAML, Zig."` Alphabetical verification — new entries in correct positions: C# (after C, C++), Dart (after CSS), Elixir (after Dockerfile), F# (after Elixir), Haskell (after Go), Lua (after Kotlin), R (after Python), Scala (after Rust), SQL (after Shell), Zig (last). Naming: README uses the conventional public-facing display name (`C#`, `F#`) rather than the Go constant identifier (`CSharp`, `FSharp`) — the right call for end-user documentation. `SQL` uppercase matches conventional name. Acceptance #6 satisfied.

#### Attack 9 — `mage test` and `mage lint` cleanliness (scoped to internal/lang)

- **Severity:** blocker (REFUTED for `internal/lang`; informational note on cross-stream lister lint pre-existing)
- **Where:** repo-wide
- **Counterexample / hypothesis:** A.2 might introduce a `mage test` failure or `mage lint` violation in `internal/lang` (missing doc comment, unused var, staticcheck violation, etc.).
- **Mitigation accepted:** REFUTED for A.2 scope. `mage test` from `main/` passes all 8 packages including `internal/lang` (cached). `mage lint` fails — BUT the failure is in `internal/lister/lister_test.go:505,528` (`cancel function is not used on all paths`), not `internal/lang`. `git log --oneline -- internal/lister/lister_test.go` shows the file's last modification was commit `86ba72e` (Drop D, `feat(lister): add filesfromlister`) — a pre-existing cross-stream lint regression that was not introduced by Unit A.2. The orchestrator should route this finding to whichever stream owns `internal/lister` (Drop D), not back to A.2.

#### Attack 10 — `.exs` accidentally classified as Code under Elixir grammar

- **Severity:** nit (REFUTED)
- **Where:** `internal/lang/split.go:143` (`LangElixir: {linePrefix: "#"}`), `split_test.go:521-533`
- **Counterexample / hypothesis:** Elixir has no block-comment form; the grammar correctly omits `blockOpen`/`blockClose`. A `# comment` line should be Comment; a code-only file should be all Code. If the grammar were wrong, the `#` from a `defmodule` shebang or similar could mis-classify lines.
- **Mitigation accepted:** REFUTED. `split.go:143` registers only `linePrefix: "#"`. Test subtests `elixir line comment` and `elixir code only` exercise both branches: `# comment\nx = 1\n` → `{Comment: 1, Code: 1}`, `defmodule Foo do\nend\n` → `{Code: 2}`. The split.go:200-205 block-marker detection skips the language entirely when `g.blockOpen == ""` (short-circuit on empty string). No false positives.

#### Attack 11 — Hidden interaction: `.fs` collision with future `.fish` stem matching

- **Severity:** nit (REFUTED)
- **Where:** `internal/lang/lang.go:100, 133`
- **Counterexample / hypothesis:** If somewhere in the codebase, extension lookup used `strings.HasPrefix(filename, ".fs")` instead of exact-key map lookup, then `.fish` files would mis-route to `LangFSharp`.
- **Mitigation accepted:** REFUTED. `Detect` uses `filepath.Ext` (returns the full extension including leading dot — for `script.fish` returns `.fish`, for `module.fs` returns `.fs`) then `extensionTable[ext]` exact-key map lookup. Map lookup is exact-string, never prefix. `filepath.Ext("script.fish") == ".fish"` and `extensionTable[".fish"] == LangShell` (line 100). No prefix matching anywhere in `Detect`.

### Informational note (not a counterexample against A.2)

- `mage lint` fails with two `cancel function is not used on all paths` errors in `internal/lister/lister_test.go:505,528`. Last modification of that file is commit `86ba72e` ("Drop D, `feat(lister): add filesfromlister`"). This is a cross-stream regression introduced by Drop D, NOT by Unit A.2. The orchestrator must route this finding to Drop D's QA loop (or absorb it into a pre-merge hygiene commit) before the drop-end `mage ci` (Phase 6). A.2's `internal/lang` package is clean.

### Summary

All 11 attack vectors REFUTED. No counterexamples found. Extension table additions are collision-free; Lua YAGNI limitation acknowledged in both implementation and tests; `.R` uppercase handling correct; Elixir/F#/Haskell extension triples all present and distinct; doc comments on all 10 new constants conform to golint; README alphabetical and naming-consistent; tests exercise the comment-detection path adequately per Acceptance #3. Cross-stream lint regression in `internal/lister` flagged for orchestrator routing — not an A.2 finding. **PASS.**

### Hylla Feedback

N/A — review touched only Go source files already fully resolvable via direct `Read` (lang.go, split.go, *_test.go), plus non-Go README.md / PLAN.md / BUILDER_WORKLOG.md / BUILDER_QA_FALSIFICATION.md. Hylla was not the load-bearing evidence source for any attack — the falsification axes (collision checks, doc-comment formatting, alphabetical ordering, grammar correctness) are all local to small, self-contained map literals and table-driven tests where `Read` on the full file is both faster and more authoritative than block summaries. None — Hylla answered everything needed at the structural level for the upstream Drop D / lister cross-stream context check, and was not required for the within-package A.2 review.
