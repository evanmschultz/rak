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

## Unit A.3 — Round 1

**Verdict:** PASS (with one minor non-blocking observation routed to Notes — templ HTML-comment limitation has docstring documentation but no lock-in test)

### Counterexamples / Attacks

#### Attack 1 — Extension collisions among 15 new entries

- **Severity:** blocker (REFUTED)
- **Where:** `internal/lang/lang.go:187-202`
- **Counterexample / hypothesis:** Any of the 15 new extensions (`.templ`, `.jsx`, `.tsx`, `.scss`, `.sass`, `.less`, `.vue`, `.svelte`, `.erb`, `.j2`, `.jinja`, `.jinja2`, `.liquid`, `.mustache`, `.hbs`) might clobber an existing key. Highest-risk pairs: `.jsx` vs `.js`, `.tsx` vs `.ts`, `.scss` vs `.css`, `.svelte` vs `.svg`, `.hbs` vs `.hs`.
- **Mitigation accepted:** REFUTED. Map keys are exact-string. Pre-A.3 table (lang.go:137-185, including A.2 additions) contains: `.bash, .c, .cc, .cpp, .css, .cxx, .fish, .gemspec, .go, .h, .hpp, .htm, .html, .java, .js, .json, .kt, .kts, .md, .php, .phtml, .py, .rake, .rb, .rs, .sh, .swift, .toml, .ts, .xml, .yaml, .yml, .zsh, .cs, .dart, .ex, .exs, .fs, .fsi, .fsx, .hs, .lhs, .lua, .r, .scala, .sql, .zig`. None of the 15 A.3 keys appear there. `.jsx ≠ .js`, `.tsx ≠ .ts`, `.scss ≠ .css`, `.svg` not in table at all, `.hbs ≠ .hs`. A duplicate map-literal key would fail compile-time (Go enforces unique map-literal keys); `mage build` passes — no collisions.

#### Attack 2 — ERB grammar trade-off acknowledged in test (locks in `%>` mis-classification)
  
- **Severity:** blocker (REFUTED)
- **Where:** `internal/lang/split_test.go:719-746`
- **Counterexample / hypothesis:** PLAN.md ERB grammar trade-off note says `%>` on expression-output lines like `<%= value %>` is mis-classified as Comment under Policy α. If the test does NOT lock this in, a future "fix" could silently regress the documented contract without any test catching it.
- **Mitigation accepted:** REFUTED. Two explicit lock-in tests:
  - `erb comment at line start` (split_test.go:719-725) asserts `{Comment: 2, Code: 0}` for input `<%# comment %>\n<%= @user.name %>\n`. Line 2 `<%= @user.name %>` containing `%>` is asserted as Comment — locking in the known limitation. Inline comment at lines 723-724 explicitly says "Line 2: contains `%>` (blockClose) → Comment (Policy α known limitation)."
  - `erb expression-output line is Comment (Policy α known limitation)` (split_test.go:735-746) is a dedicated subtest with verbose comment at lines 736-740: "This line is mis-classified as Comment. This is the accepted trade-off (see PLAN.md ERB grammar trade-off note and Notes § "ERB grammar trade-off"). Document here to lock in the known behavior." Expected `{Comment: 1, Code: 1}` for `<%= @title %>\n<p>plain html</p>\n`.
  - The function docstring (split_test.go:601-604) restates the limitation at the suite level.

#### Attack 3 — Vue `<script>` JS-comment blind spot test lock-in
  
- **Severity:** blocker (REFUTED)
- **Where:** `internal/lang/split_test.go:701-706`
- **Counterexample / hypothesis:** Vue uses HTML-level `<!-- -->` grammar; JS comments inside `<script>` blocks should classify as Code (one-grammar policy). If no test locks this in, a future "fix" adding JS sub-parsing to Vue/Svelte could silently regress without any failure.
- **Mitigation accepted:** REFUTED. Explicit lock-in subtest `vue script js comment is Code (sub-parsing out of scope)` (split_test.go:701-706) asserts `{Comment: 0, Code: 4}` for input `<script>\n// this js comment classifies as Code — single grammar policy\nconst x = 1\n</script>\n`. The asserted Comment=0 locks in the limitation; the inline test-comment in the input string ("classifies as Code — single grammar policy") makes the intent unambiguous to future maintainers. Function docstring (split_test.go:606-609) restates the policy at the suite level.

#### Attack 4 — `.hbs` → LangMustache explicit test
  
- **Severity:** blocker (REFUTED)
- **Where:** `internal/lang/lang.go:202`, `internal/lang/lang_test.go:442-443`
- **Counterexample / hypothesis:** `.hbs` Handlebars extension might map to LangUnknown (no entry) or to a wrong constant.
- **Mitigation accepted:** REFUTED. lang.go:202 `.hbs: LangMustache`. lang_test.go:443 row `{"view.hbs", LangMustache}` exercises it. PLAN.md Acceptance #3 satisfied.

#### Attack 5 — Templ HTML-comment limitation: docstring vs test lock-in

- **Severity:** concern (PARTIALLY CONFIRMED — non-blocking)
- **Where:** `internal/lang/split.go:160-162`, `internal/lang/lang.go:79-80`, `internal/lang/split_test.go` (no templ HTML-comment lock-in subtest)
- **Counterexample / hypothesis:** PLAN.md Notes (`Templ HTML-comment fallback`) and PLAN.md A.3 scope say `LangTempl` uses Go-style grammar; HTML-like `<!-- -->` comments in `.templ` template blocks should classify as Code (single-grammar policy). Vue has an explicit lock-in test for the analogous limitation (Attack 3) — does templ?
- **Mitigation accepted:** PARTIALLY CONFIRMED. The limitation is documented in three places:
  - `LangTempl` const docstring (lang.go:79-80): "Templ uses Go-style comment syntax (// and /* */)."
  - grammar entry inline comment (split.go:160-162): "HTML-like `<!-- -->` comments inside `.templ` files classify as Code (single-grammar policy, design principle 2, out of scope v0.2.0)."
  - `TestSplit_Templating` function docstring (split_test.go:611-613) restates the policy at suite level.
- **However**, unlike the Vue case (split_test.go:701-706 has an explicit `vue script js comment is Code` lock-in subtest), there is **no analogous lock-in subtest for templ** that asserts `Split(LangTempl, "<!-- html comment -->\nfunc Foo() ...")` produces `Comment: 0`. The two templ subtests (`templ line comment` at split_test.go:629-633, `templ block comment` at split_test.go:635-639) only exercise Go-style `//` and `/* */`, not the HTML-comment-as-Code negative assertion. A future maintainer who adds HTML-comment grammar to templ would not be caught by a failing test — only the docstring would flag the contract change.
- **Severity rationale:** non-blocking because (a) the contract is documented in three places, (b) A.3's acceptance criteria 2-10 don't mandate this specific lock-in test, (c) the suite docstring covers it at the test-file level. **Routed to Notes / future-maintainer attention**, not back to builder. A future "Drop A.6 — limitation lock-in tests" would be the right place to add this; not appropriate to gate A.3 close on it.

#### Attack 6 — Jinja multi-extension coverage (`.j2`, `.jinja`, `.jinja2`)
  
- **Severity:** blocker (REFUTED)
- **Where:** `internal/lang/lang.go:197-199`, `internal/lang/lang_test.go:436-438`
- **Counterexample / hypothesis:** Any of the three Jinja extensions might be missing or mapped to a different language.
- **Mitigation accepted:** REFUTED. All three present (`.j2`, `.jinja`, `.jinja2` → `LangJinja` at lang.go:197-199). Test exercises all three (lang_test.go:436-438). No collision with other entries.

#### Attack 7 — Liquid `{% comment %}` multi-line state-machine correctness
  
- **Severity:** concern (REFUTED)
- **Where:** `internal/lang/split.go:199`, `internal/lang/split_test.go:763-772`
- **Counterexample / hypothesis:** Liquid uses `{% comment %}` / `{% endcomment %}` block tags typically on separate lines. If the state-machine update logic (split.go:272-288) mis-handles the multi-character markers, the inside-block lines would mis-classify.
- **Mitigation accepted:** REFUTED. split.go:199 grammar `{blockOpen: "{% comment %}", blockClose: "{% endcomment %}"}` uses correct full-tag strings. Test `liquid comment block` (split_test.go:763-772) exercises the canonical 4-line case: line 1 `{% comment %}` (Comment via blockOpen + sets inBlockComment), line 2 `This is hidden.` (inBlockComment=true → Comment), line 3 `{% endcomment %}` (inBlockComment=true at line start → Comment, then closes block), line 4 `{{ title }}` (Code). Expected `{Comment: 3, Code: 1}` matches the state-machine trace.

#### Attack 8 — `-race` cleanliness for `internal/lang`
  
- **Severity:** blocker (REFUTED)
- **Where:** repo-wide via `mage test` (always runs with `-race`)
- **Counterexample / hypothesis:** `internal/lang` might surface a race under `-race` (despite being a pure-function package with no goroutines, parallel subtests share package-level `grammarTable` / `extensionTable` / `specialFilenames` / `shebangsTable`).
- **Mitigation accepted:** REFUTED. `mage test` (runs `-race`) — output: all 8 packages pass including `ok  github.com/evanmschultz/rak/internal/lang`. Package-level tables are immutable (Go map literals as `var`, never mutated after init) — concurrent reads are race-free.

#### Attack 9 — Doc comments on all 12 new constants
  
- **Severity:** blocker (REFUTED)
- **Where:** `internal/lang/lang.go:77-118`
- **Counterexample / hypothesis:** Any of the 12 new `Lang*` constants might be missing a doc comment or have one not starting with the identifier — would fail `golangci-lint` `revive` / `staticcheck` `exported` rule.
- **Mitigation accepted:** REFUTED. All 12 have well-formed doc comments starting with `// LangX`:
  - `// LangTempl is the Language constant for Go-superset templ files (.templ). Templ uses Go-style comment syntax (// and /* */).` (line 79-80)
  - `// LangJSX is the Language constant for React JSX files (.jsx).` (line 82-83)
  - `// LangTSX is the Language constant for TypeScript JSX files (.tsx). Distinct from .ts → LangTS.` (line 84-86)
  - `// LangSCSS is the Language constant for SCSS stylesheets (.scss). SCSS supports both // line comments and /* */ block comments.` (line 87-89)
  - `// LangSass is the Language constant for indented Sass stylesheets (.sass). Uses // for line comments; /* */ block comments exist but are less common (Policy α YAGNI — some non-comment lines may be over-classified).` (line 90-93)
  - `// LangLESS is the Language constant for LESS stylesheets (.less).` (line 94-95)
  - `// LangVue is the Language constant for Vue single-file components (.vue). Grammar covers HTML-level <!-- --> comments; JS/TS inside <script> blocks uses JS/TS comment syntax not detected here (one file = one grammar, design principle 2, out of scope for v0.2.0).` (line 96-100)
  - `// LangSvelte is the Language constant for Svelte components (.svelte). Same single-grammar HTML-level policy as LangVue.` (line 101-103)
  - `// LangERB is the Language constant for Ruby ERB templates (.erb). Grammar uses block form <%# ... %> to catch mid-line ERB comments. Known limitation: %> also appears on expression-output lines like <%= value %> — those lines are mis-classified as Comment (Policy α YAGNI).` (line 104-108)
  - `// LangJinja is the Language constant for Jinja2 templates (.j2, .jinja, .jinja2).` (line 109-111)
  - `// LangLiquid is the Language constant for Liquid templates (.liquid).` (line 112-113)
  - `// LangMustache is the Language constant for Mustache and Handlebars templates (.mustache, .hbs). Handlebars is a Mustache superset sharing the same comment syntax; one constant follows the existing pattern of grouping closely-related variants (Shell groups sh/bash/zsh/fish).` (line 114-118)
  Confirmed by `mage lint`: `0 issues.`

#### Attack 10 — README alphabetical order for 12 new entries
  
- **Severity:** concern (REFUTED)
- **Where:** `README.md:122`
- **Counterexample / hypothesis:** Any of the 12 new entries (ERB, Jinja, JSX, LESS, Liquid, Mustache/Handlebars, Sass, SCSS, Svelte, Templ, TSX, Vue) may be out of alphabetical position or break the paragraph form.
- **Mitigation accepted:** REFUTED. README:122 reads: `"C, C++, C#, CMakeLists.txt, CSS, Dart, Dockerfile, Elixir, ERB, F#, Go, Haskell, HTML, Java, JavaScript, Jinja, JSON, JSX, Kotlin, LESS, Liquid, Lua, Makefile, Markdown, Mustache/Handlebars, PHP, Python, R, Ruby, Rust, Sass, Scala, SCSS, Shell (sh/bash/zsh/fish), SQL, Svelte, Swift, Templ, TOML, TSX, TypeScript, Vue, XML, YAML, Zig."` Verifying case-insensitive ordering at each insertion point:
  - `Elixir, ERB, F#`: E-l < E-r < F (correct)
  - `Java, JavaScript, Jinja, JSON, JSX`: j-a-v-a-(end) < j-a-v-a-s < j-i < j-s-o < j-s-x (correct)
  - `Kotlin, LESS, Liquid, Lua`: K < l-e < l-i < l-u (correct)
  - `Markdown, Mustache/Handlebars, PHP`: m-a < m-u < P (correct)
  - `Rust, Sass, Scala, SCSS, Shell`: R < s-a < s-c-a < s-c-s < s-h (correct)
  - `SQL, Svelte, Swift`: s-q < s-v-e < s-w (correct)
  - `Swift, Templ, TOML, TSX, TypeScript`: s-w < t-e < t-o < t-s < t-y (correct)
  - `TypeScript, Vue, XML`: t-y < V < X (correct)
  All 12 new entries correctly placed alphabetically. Paragraph form held at 45 entries (still readable; A.5 will convert to a comma-separated list at 50+ per A.1's locked decision).

#### Attack 11 — `mage lint` cleanliness post-D.1 fix
  
- **Severity:** blocker (REFUTED)
- **Where:** repo-wide
- **Counterexample / hypothesis:** A.3 might introduce a `mage lint` violation (missing doc comment, unused var, staticcheck issue). Also: prior A.2 round flagged a pre-existing `internal/lister` lint failure (commit 13ac39a should have fixed it).
- **Mitigation accepted:** REFUTED. `mage lint` from `main/`: `0 issues.` — full repo clean. D.1's `cancel function is not used on all paths` lister regression has been fixed; A.3 introduces no new violations.

#### Attack 12 — `{{!` linePrefix vs `{{!--` blockOpen ordering in Split
  
- **Severity:** concern (REFUTED)
- **Where:** `internal/lang/split.go:241-263`, `internal/lang/split_test.go:775-795`
- **Counterexample / hypothesis:** `{{!` is a string-prefix of `{{!--`. If `Split` evaluated linePrefix before blockOpen, the block form would never fire — `{{!--` would always match linePrefix first and the state machine wouldn't initialize the multi-line block. Result: multi-line Mustache block comments would only count the open and close lines as Comment via linePrefix, and middle lines would be Code.
- **Mitigation accepted:** REFUTED. Split check order (split.go:241-263) is: (a) inBlockComment carry-over, (b) blockOpen contained, (c) blockClose contained, (d) linePrefix prefix, (e) linePrefix2 prefix. blockOpen fires BEFORE linePrefix. For `{{!-- ... --}}` (single-line) — Contains `{{!--` → Comment via blockOpen. For `{{!--\n  body\n--}}\n` — line 1 Contains `{{!--` → Comment + state-machine sets inBlockComment=true; line 2 → inBlockComment=true → Comment; line 3 Contains `--}}` → Comment + state-machine closes. Test `mustache multiline block comment` (split_test.go:790-795) asserts `{Comment: 3, Code: 1}` — matches state-machine trace.

#### Attack 13 — State-machine corruption on `<%= %>`-only ERB lines
  
- **Severity:** concern (REFUTED)
- **Where:** `internal/lang/split.go:272-288`
- **Counterexample / hypothesis:** For ERB grammar (`blockOpen: "<%#"`, `blockClose: "%>"`), a line `<%= @title %>` contains `%>` but no `<%#`. The state-machine pass searches both: openIdx = -1, closeIdx = N. The "else" branch fires: `inBlockComment = false`. If the prior line had legitimately opened a block (`<%# ...`), would this `<%= %>` line falsely close it?
- **Mitigation accepted:** REFUTED with caveat. The state-machine update only runs for `g.blockOpen != ""` (split.go:272). For ERB: openIdx=-1, closeIdx=N → inBlockComment is set to false. This means a `<%# ...` block legitimately opened on a prior line WOULD be falsely closed by a subsequent `<%= %>` line. **However**, this is the same Policy α YAGNI documented in PLAN.md ERB grammar trade-off section — ERB's overlapping markers make state-machine accuracy impossible without sub-parsing. The trade-off is acknowledged in two test cases + grammar docstring + function docstring + PLAN.md notes. No new finding — this is the documented limitation.

#### Attack 14 — `.tsx` vs `.ts` distinct mapping (Acceptance #4)
  
- **Severity:** blocker (REFUTED)
- **Where:** `internal/lang/lang.go:165 (.ts), 190 (.tsx)`, `internal/lang/lang_test.go:419-422`
- **Counterexample / hypothesis:** `.tsx` and `.ts` are exact-string distinct keys but a test might miss the regression-guard on `.ts → LangTS`.
- **Mitigation accepted:** REFUTED. lang.go:165 `.ts → LangTS`, lang.go:190 `.tsx → LangTSX` — distinct keys. lang_test.go:420 `{"app.tsx", LangTSX}` + lang_test.go:422 `{"types.ts", LangTS}` (explicit regression guard). Acceptance #4 satisfied with positive + negative test.

#### Attack 15 — JSON / XML / HTML regression from A.3
  
- **Severity:** blocker (REFUTED)
- **Where:** `internal/lang/lang.go:148-167`, `internal/lang/split.go:117-123`
- **Counterexample / hypothesis:** Adding 15 new extension entries might accidentally clobber `.json`, `.xml`, `.html`, `.htm` mappings.
- **Mitigation accepted:** REFUTED. Pre-A.3 entries `.htm → LangHTML` (line 148), `.html → LangHTML` (line 149), `.json → LangJSON` (line 152), `.xml → LangXML` (line 166) all present and unchanged. grammarTable LangHTML (line 118), LangXML (line 119), LangMarkdown (line 120) entries untouched. No regression.

#### Attack 16 — Sass grammar over-classification documented
  
- **Severity:** nit (REFUTED)
- **Where:** `internal/lang/split.go:170-174`, `internal/lang/lang.go:90-93`, `internal/lang/split_test.go:614-617`
- **Counterexample / hypothesis:** Indented Sass rarely uses `/* */` blocks. Assigning C-family grammar to Sass may over-classify some lines. Should be documented.
- **Mitigation accepted:** REFUTED. Documented in three places: const docstring (lang.go:90-93 mentions Policy α YAGNI), grammar inline comment (split.go:170-172), suite docstring (split_test.go:614-617 mentions Sass Policy α YAGNI). Documentation is thorough. Test `sass line comment` asserts `{Comment: 1, Code: 2}` for `// comment\n.foo\n  color: red\n` — exercises the common-case line-comment path.

#### Attack 17 — `index.html.erb` filename quirk
  
- **Severity:** nit (REFUTED)
- **Where:** `internal/lang/lang.go:255` (`filepath.Ext`), `internal/lang/lang_test.go:434`
- **Counterexample / hypothesis:** `index.html.erb` is the conventional Rails template filename. `filepath.Ext` returns only the last extension (`.erb`). Test should exercise the realistic Rails filename, not just `foo.erb`.
- **Mitigation accepted:** REFUTED. lang_test.go:434 row `{"index.html.erb", LangERB}` uses the canonical Rails name. `filepath.Ext("index.html.erb")` returns `.erb` → `extensionTable[".erb"] → LangERB`. Correct.

### Informational notes (not counterexamples against A.3)

- **Templ HTML-comment limitation lock-in test gap** (Attack 5): three layers of documentation cover the limitation but no dedicated test asserts `Split(LangTempl, "<!-- html -->")` → Comment=0. Vue has an analogous lock-in test (Attack 3); templ doesn't. Non-blocking — documentation is thorough and A.3 acceptance criteria don't require this specific test. Future maintainer attention only.
- **`{%- comment -%}` Liquid whitespace-trim form** is not detected (grammar requires literal `{% comment %}`). Not in A.3's stated scope, no PLAN.md mention. Acceptable — files using the trim form would have those lines classified as Code rather than Comment. Future PR if requested.
- **ERB state-machine false-close** (Attack 13): a `<%= %>` line after a legitimate `<%# ...` block-open will falsely close the block. Documented as Policy α YAGNI in PLAN.md + grammar docstring + test docstrings. Not new.
- **PLAN.md A.3 state**: PLAN.md (line 147) currently shows `**State:** done` for Unit A.3. Worklog says builder set it to `in_progress` pending mage permission grant (line 71). PLAN.md says done; worklog says in_progress pre-test-run. This is a worklog/PLAN.md drift bookkeeping nit — not a finding against the code. `mage test` and `mage lint` now pass, so the `done` state in PLAN.md matches verified reality.

### Summary

17 attack vectors evaluated. 16 REFUTED, 1 PARTIALLY CONFIRMED non-blocking (Attack 5 — templ HTML-comment limitation has docstring documentation but no test lock-in; routed to future-maintainer attention, not back to builder; A.3 acceptance criteria don't mandate this). No blocker counterexamples. Extension table collision-free across all 15 new entries. ERB and Vue Policy α limitations are explicitly locked in by test assertions. Mustache `{{!` / `{{!--` ordering verified correct against state-machine semantics. `mage test` passes all 8 packages including `internal/lang` with `-race`; `mage lint` clean (0 issues). README alphabetical and naming-consistent across 12 new entries. **PASS.**

### Hylla Feedback

N/A — review touched only Go source files inside `internal/lang` (lang.go, split.go, lang_test.go, split_test.go) and non-Go README.md / PLAN.md / BUILDER_WORKLOG.md / BUILDER_QA_FALSIFICATION.md. Hylla was not the load-bearing evidence source — falsification axes (extension-key collision checks, doc-comment formatting, alphabetical ordering, grammar correctness, state-machine traces, lock-in test presence) are all local to small self-contained map literals and table-driven tests where `Read` on the full file is both faster and more authoritative than block summaries. None — Hylla answered everything needed at the structural level and was not required for the within-package A.3 review.
