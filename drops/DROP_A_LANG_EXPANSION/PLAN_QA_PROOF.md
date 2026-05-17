# DROP_A — PLAN_QA_PROOF

## Round 1

**Verdict:** PASS WITH FINDINGS

The plan's decomposition is sound and the grammar choices are largely correct against
the source-language specs. Findings below are all `concern` or `nit` — none are
blockers. The serial-chain rationale, path-footprint discipline, and acceptance-
testability bar are otherwise met.

## Findings

### Finding 1 — README alphabetical position of XML is wrong

- **Severity:** nit
- **Where:** Unit A.1 — acceptance criterion #6 ("alphabetically, between YAML and
  the end").
- **Issue:** Reading `README.md:122` the current "Languages detected" paragraph ends
  with `...Swift, TOML, TypeScript, YAML.` Alphabetically `X` precedes `Y`, so XML
  belongs **before** YAML, not after. "Between YAML and the end" is alphabetically
  impossible — nothing comes after YAML, and XML is not after YAML in alpha order
  either.
- **Recommendation:** Reword to "alphabetically, immediately before YAML"
  (`...TypeScript, XML, YAML.`).

### Finding 2 — `mage test ./internal/lang/...` is not a real mage invocation

- **Severity:** concern
- **Where:** Acceptance criterion #1 in Units A.1, A.2, A.3, A.4 (and A.5 which uses
  `mage ci`).
- **Issue:** `magefile.go:35–40` defines `Test()` as `go test -race ./...` — whole-
  tree, no path argument. There is no `mage test <pkg>` form. Per CLAUDE.md "NEVER
  run raw `go test`," the builder cannot legitimately scope to `./internal/lang/...`
  via mage. The criterion as written is strictly unverifiable by a QA subagent:
  either `mage test` (whole tree, satisfies the spirit) or `go test -race ./internal/lang/...`
  (forbidden by CLAUDE.md). The spirit is clear but the literal phrasing fails
  yes/no-testability.
- **Recommendation:** Replace `mage test ./internal/lang/...` with `mage test`
  (whole-tree, includes the lang package) in all four units. Note in the unit's
  acceptance that the lang-package tests must show passing in the mage output.

### Finding 3 — Templ HTML-comment form is silently dropped

- **Severity:** concern
- **Where:** Unit A.3 — grammar entry for `LangTempl`.
- **Issue:** Per `templ.guide/syntax-and-usage/comments` (Context7), templ files
  contain **both** Go comments (`//`, `/* */`) outside templ statements **and** HTML
  comments (`<!-- -->`) inside templ statements. The plan picks only Go-style. The
  `grammar` struct (`split.go:53–67`) supports one `blockOpen`/`blockClose` pair, so
  the choice is forced — but the HTML-comment lines inside templ statements will be
  silently classified as Code (no comment markers match). This is a Policy α YAGNI
  trade-off equivalent to Vue/Svelte's already-disclosed limit, but the plan does
  not disclose it.
- **Recommendation:** Add a Notes-section bullet for `LangTempl` documenting that
  HTML comments inside templ statements are misclassified as Code (same YAGNI
  posture as Vue/Svelte). No code change needed — just the disclosure.

### Finding 4 — ERB block-comment grammar drops the `<%# %>` form

- **Severity:** concern
- **Where:** Unit A.3 — grammar entry for `LangERB`
  (`linePrefix: "<%#"`, `blockOpen: "<!--"`, `blockClose: "-->"`).
- **Issue:** ERB's native comment form is `<%# ... %>` (Ruby ERB tag-comment), not
  the line-prefix form. The plan's `linePrefix: "<%#"` only catches lines that
  **start** with `<%#` (per `split.go:174` `strings.HasPrefix(trimmed, g.linePrefix)`).
  Mid-line `<%# ... %>` tags are missed. The HTML `<!-- -->` block grammar covers
  HTML-comment forms but not ERB-native ones. This is a known YAGNI but is not
  disclosed in the plan's Notes.
- **Recommendation:** Add a Notes bullet documenting that ERB `<%# %>` comments are
  only detected when they start a line; in-line `<%# %>` tags are classified as
  Code. Acceptable for v0.2.0 but should be on the record.

### Finding 5 — `--lang bazel` end-to-end claim escapes A.5's path footprint

- **Severity:** concern
- **Where:** Unit A.5 — Scope paragraph "Also add a `--lang bazel` end-to-end note
  to acceptance..."
- **Issue:** `--lang` filtering is implemented in `cmd/rak/root.go`, not in
  `internal/lang`. A.5's declared `Paths` are `internal/lang/*` plus `README.md`. A
  test that drives `rak --lang bazel <fixture>` belongs in `cmd/rak/root_test.go` or
  `cmd/rak/integration_test.go` — outside the unit's path footprint. The plan
  acknowledges the `fstest.MapFS` alternative, but `fstest.MapFS` would only verify
  `Detect()` returns `LangBazel`, which is already covered by acceptance #2. The
  end-to-end claim is not independently testable within the declared paths.
- **Recommendation:** Drop the `--lang bazel` end-to-end paragraph from A.5's Scope.
  Acceptance #2 (`Detect on BUILD/BUILD.bazel/WORKSPACE returns LangBazel`) already
  fully covers what `internal/lang` can verify. CLI-filter behavior is covered by
  the existing `--lang` test infrastructure; new Languages are automatically picked
  up.

### Finding 6 — Mustache linePrefix `{{!` is a prefix of blockOpen `{{!--`

- **Severity:** nit
- **Where:** Unit A.3 — grammar entry for `LangMustache`
  (`linePrefix: "{{!"`, `blockOpen: "{{!--"`, `blockClose: "--}}"`).
- **Issue:** This is functionally correct (any `{{!--` line matches `linePrefix
  "{{!"` via `HasPrefix` AND `Contains "{{!--"` — both paths classify it as
  Comment) so the result is right. But the overlap is subtle: per Context7
  (Handlebars `/websites/handlebarsjs`), `{{!--` is "a comment that may contain
  mustaches like `}}`" and `{{!` is the bare comment. Adding a Notes bullet
  explaining the overlap prevents a future maintainer from "fixing" the apparent
  redundancy.
- **Recommendation:** One-line Notes entry explaining that `linePrefix: "{{!"` is
  deliberately a prefix of `blockOpen: "{{!--"` — both paths classify Mustache/
  Handlebars comment lines correctly; the redundancy is intentional, not a bug.

### Finding 7 — Bazel `BUILD` filename is generic and may collide downstream

- **Severity:** nit
- **Where:** Unit A.5 — specialFilenames adds `"build"` → `LangBazel`.
- **Issue:** `BUILD` is also commonly used by other build systems (notably as a
  bare uppercase filename in some legacy repos and in non-Bazel contexts). The
  plan's choice is the modern-tooling default but should be acknowledged as a
  best-effort heuristic. Not blocking — Bazel is the dominant `BUILD`-filename
  user in 2026.
- **Recommendation:** One-line Notes entry mentioning that bare `BUILD` is mapped
  to Bazel as the modern-default assumption; if a future user-report shows a
  false-positive collision, the constant can be refined.

### Finding 8 — Lua long-bracket variants (`[=[`, `[==[`) are not covered

- **Severity:** nit
- **Where:** Unit A.2 — grammar entry for `LangLua` (`blockOpen: "--[["`,
  `blockClose: "]]"`).
- **Issue:** Lua long-bracket comments accept `--[=[ ... ]=]`, `--[==[ ... ]==]`,
  etc. (any matching `=` count). The plan covers only `--[[ ... ]]`. The
  uncovered variants are rare and the existing Notes bullet about Lua's `]]`
  table-index ambiguity is the right altitude of YAGNI disclosure. Add a one-line
  expansion of that Notes bullet covering the variant-bracket case.
- **Recommendation:** Extend the existing "Lua block comments" Notes bullet:
  "...long-bracket variants `--[=[`/`]=]` and deeper are not detected (rare;
  Policy α YAGNI v0.2.0)."

## Notes on what the plan got right

- **Extension collision analysis is correct.** Cross-checked `internal/lang/lang.go:68–102`
  against every new extension in the plan. Zero collisions confirmed.
- **specialFilenames collision analysis is correct.** Cross-checked
  `internal/lang/lang.go:57–64` against new entries. Zero collisions confirmed.
- **Serial-chain rationale is sound.** All five units genuinely share the same
  five paths; sub-splitting per file would be artificial. The `A.1 → A.2 → A.3 →
  A.4 → A.5` blocked_by chain is consistent.
- **XML regression-guard reasoning is verified.** `internal/lang/lang_test.go:18–56`
  (TestDetect_ByExtension) contains no `.xml` row, so A.1's change of `.xml` from
  `LangHTML` to `LangXML` will not break existing tests. Plan's claim verified.
- **Grammar struct fit for new entries is verified.** All proposed grammars
  (including HCL's three-comment-form coverage via linePrefix + linePrefix2 +
  blockOpen/Close) fit the existing 4-field `grammar` struct in `split.go:53–67`.
  No struct changes required — the work is purely table-additive.
- **`.env` extension semantics are correct.** `go doc path/filepath.Ext`
  confirms `filepath.Ext(".env") == ".env"`; the plan's note matches Go stdlib
  behavior.
- **Procfile/CSV/TSV/JSONL grammar absence is consistent.** Constants exist for
  `--lang` filtering; the empty grammar correctly classifies all non-blank lines
  as Code (acceptance #11 in A.5 etc.).
- **Policy α YAGNI disclosures for Sass, Vue, Svelte, Lua `]]`, HCL triple-form**
  are appropriately documented in the Notes section.
