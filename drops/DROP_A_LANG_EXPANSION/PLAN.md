# DROP_A — LANG_EXPANSION

**State:** planning
**Tier:** A
**Blocked by:** —
**Paths (expected):** internal/lang/lang.go, internal/lang/split.go, internal/lang/*_test.go, README.md
**Packages (expected):** internal/lang
**PLAN.md ref:** — (top-level PLAN.md removed at v0.1.0 ship; see memory `session_handoff_2026_05_16_v020_planning.md` for v0.2.0 scope)
**Workflow:** main/drops/WORKFLOW.md
**Started:** 2026-05-16
**Closed:** —

## Scope

Add ~30 new languages plus a long-overdue XML-from-HTML split. Coverage targets:

- **Programming**: C#, Scala, Lua, SQL, Dart, Elixir, Zig, R, F#, Haskell.
- **Templating + frontend variants**: templ (`.templ`), JSX (`.jsx`), TSX (`.tsx`), Sass/SCSS (`.scss`, `.sass`), LESS (`.less`), Vue (`.vue`), Svelte (`.svelte`), ERB (`.erb`), Jinja (`.j2`, `.jinja`, `.jinja2`), Liquid (`.liquid`), Mustache (`.mustache`), Handlebars (`.hbs`).
- **Config**: INI (`.ini`), `.env`, `.editorconfig`, `.properties`, HCL/Terraform (`.tf`, `.tfvars`, `.hcl`), Nix (`.nix`).
- **Data/schema**: `.proto`, `.graphql`/`.gql`, `.csv`, `.tsv`, `.jsonl`/`.ndjson`.
- **Build/task files**: Bazel (`BUILD`, `BUILD.bazel`, `WORKSPACE`, `*.bzl`), Justfile / `justfile`, Earthfile, Jenkinsfile (Groovy), Vagrantfile (Ruby), Brewfile (Ruby), Procfile, Caddyfile.
- **XML split** from HTML into its own `LangXML` constant.

Locked design principles (from dev 2026-05-16):

1. Extension-first; content-sniff only as last-resort disambiguator (e.g. `.m` MATLAB-vs-ObjC).
2. One file = one language — no Vue/Svelte sub-parsing, no notebook split.
3. Group only when distinction doesn't matter (`Shell` already groups sh/bash/zsh/fish). Do **not** group CSS preprocessors.
4. Each lang gets: `Language` constant + extension/filename/shebang table entry + comment-split rule + detection test + split test + README "Languages detected" entry.
5. Skip MATLAB, Fortran, VHDL, Verilog — let community add via PR.

## Planner

All five units share the same paths (`internal/lang/lang.go`, `internal/lang/split.go`,
`internal/lang/lang_test.go`, `internal/lang/split_test.go`, `README.md`) and therefore
form a strict serial chain. No parallelism is possible at this level — a sub-split per
file would be over-engineering given the purely additive map-literal nature of the work.

---

### Unit A.1 — XML split from HTML

**State:** todo

**Paths:**
- `internal/lang/lang.go`
- `internal/lang/split.go`
- `internal/lang/lang_test.go`
- `internal/lang/split_test.go`
- `README.md`

**Packages:** `github.com/evanmschultz/rak/internal/lang`

**Scope:**
Add `LangXML Language = "xml"` constant. Change `extensionTable[".xml"]` from `LangHTML`
to `LangXML`. Update `detectContent`: the `<?xml` branch currently returns `LangHTML` —
change it to return `LangXML`. Add `LangXML` to `grammarTable` with `<!-- -->` grammar
(identical to `LangHTML` — XML and HTML share the same comment delimiter). Update README
"Languages detected" section to list XML as a separate entry and note that `.xml` no
longer maps to HTML.

README format decision (locked here; A.2–A.5 builders inherit this choice): the current
paragraph form holds up to ~15 languages; at 50+ entries from DROP_A, switch to an
alphabetical comma-separated list. Use the format: "Language1, Language2, ..." with one
entry per language or alias group, sorted case-insensitively. If the builder judges the
list still readable as a paragraph at A.1's merge point, they may keep the paragraph
form — but the A.5 builder must switch to the comma-separated list (50+ entries at that
point). Surface to dev via PR comment if the form choice is unclear.

**Acceptance:**
1. `mage test` passes with no new failures.
2. `Detect` on a file named `foo.xml` returns `LangXML`, not `LangHTML`.
3. `Detect` on content starting with `<?xml` (extensionless file) returns `LangXML`.
4. `Split` with `LangXML` and `<!-- comment -->` input counts 1 Comment line (HTML-same grammar confirmed).
5. `Detect` on `foo.html` still returns `LangHTML` (regression guard).
6. README "Languages detected" section lists XML as a separate entry (alphabetically, before YAML).
7. README notes that `.xml` files now appear as `xml` in `total_by_lang` instead of `html` — this is an intentional v0.2.0 behavior change from v0.1.x; builder should flag it in the PR description for release notes.
8. `mage build` passes.

**Blocked by:** —

---

### Unit A.2 — Programming languages (C#, Scala, Lua, SQL, Dart, Elixir, Zig, R, F#, Haskell)

**State:** todo

**Paths:**
- `internal/lang/lang.go`
- `internal/lang/split.go`
- `internal/lang/lang_test.go`
- `internal/lang/split_test.go`
- `README.md`

**Packages:** `github.com/evanmschultz/rak/internal/lang`

**Scope:**
Add 10 `Language` constants (all new, not yet in tree):
- `LangCSharp Language = "csharp"` — `.cs` extension
- `LangScala Language = "scala"` — `.scala` extension
- `LangLua Language = "lua"` — `.lua` extension
- `LangSQL Language = "sql"` — `.sql` extension
- `LangDart Language = "dart"` — `.dart` extension
- `LangElixir Language = "elixir"` — `.ex`, `.exs` extensions
- `LangZig Language = "zig"` — `.zig` extension
- `LangR Language = "r"` — `.r` extension (lowercased; `filepath.Ext` on `.R` files lowercases to `.r`)
- `LangFSharp Language = "fsharp"` — `.fs`, `.fsi`, `.fsx` extensions
- `LangHaskell Language = "haskell"` — `.hs`, `.lhs` extensions

Add all extensions to `extensionTable`. Add grammar entries to `grammarTable`:
- `LangCSharp`: `linePrefix: "//"`, `blockOpen: "/*"`, `blockClose: "*/"` (C-family)
- `LangScala`: `linePrefix: "//"`, `blockOpen: "/*"`, `blockClose: "*/"` (C-family)
- `LangLua`: `linePrefix: "--"`, `blockOpen: "--[["`, `blockClose: "]]"` (Lua long-bracket)
- `LangSQL`: `linePrefix: "--"`, `blockOpen: "/*"`, `blockClose: "*/"` (ANSI SQL)
- `LangDart`: `linePrefix: "//"`, `blockOpen: "/*"`, `blockClose: "*/"` (C-family)
- `LangElixir`: `linePrefix: "#"` (no block-comment form in Elixir)
- `LangZig`: `linePrefix: "//"` (no block-comment form; `////` doc comments use same prefix)
- `LangR`: `linePrefix: "#"` (no block-comment form)
- `LangFSharp`: `linePrefix: "//"`, `blockOpen: "(*"`, `blockClose: "*)"` (ML-style)
- `LangHaskell`: `linePrefix: "--"`, `blockOpen: "{-"`, `blockClose: "-}"` (Haskell multi-line)

Tests: extend `TestDetect_ByExtension` table (or add a new
`TestDetect_ProgrammingLanguages` table-driven test) covering at least one extension per
language. Add `TestSplit_ProgrammingLanguages` table-driven test covering
blank/comment/code for each grammar (one representative snippet per lang). Tests must
pass with `-race`.

README "Languages detected": append the 10 new language names in alphabetical order.

**Acceptance:**
1. `mage test` passes.
2. Each of the 10 new extensions resolves to the correct Language constant via `Detect`.
3. `Split` with each grammar returns correct Comment classification for a line matching
   that language's comment syntax (at minimum one assertion per grammar entry).
4. `LangR` detection: `filepath.Ext` on both `analysis.R` and `script.r` (both lowercased
   to `.r` by `Detect`) return `LangR`.
5. Lua block-comment limitation documented in test: a line `--[[ comment ]]` is Comment.
6. README lists the 10 new languages alphabetically.
7. `mage build` passes.

**Blocked by:** A.1

---

### Unit A.3 — Templating and frontend variants

**State:** todo

**Paths:**
- `internal/lang/lang.go`
- `internal/lang/split.go`
- `internal/lang/lang_test.go`
- `internal/lang/split_test.go`
- `README.md`

**Packages:** `github.com/evanmschultz/rak/internal/lang`

**Scope:**
Add 12 `Language` constants (all new, not yet in tree):
- `LangTempl Language = "templ"` — `.templ` extension (Go-superset; Go-style comments)
- `LangJSX Language = "jsx"` — `.jsx` extension
- `LangTSX Language = "tsx"` — `.tsx` extension
- `LangSCSS Language = "scss"` — `.scss` extension
- `LangSass Language = "sass"` — `.sass` extension (indented Sass syntax)
- `LangLESS Language = "less"` — `.less` extension
- `LangVue Language = "vue"` — `.vue` extension
- `LangSvelte Language = "svelte"` — `.svelte` extension
- `LangERB Language = "erb"` — `.erb` extension
- `LangJinja Language = "jinja"` — `.j2`, `.jinja`, `.jinja2` extensions
- `LangLiquid Language = "liquid"` — `.liquid` extension
- `LangMustache Language = "mustache"` — `.mustache`, `.hbs` extensions

Add all extensions to `extensionTable`. Add grammar entries to `grammarTable`:
- `LangTempl`: `linePrefix: "//"`, `blockOpen: "/*"`, `blockClose: "*/"` (Go-superset)
- `LangJSX`: `linePrefix: "//"`, `blockOpen: "/*"`, `blockClose: "*/"` (JS-family)
- `LangTSX`: `linePrefix: "//"`, `blockOpen: "/*"`, `blockClose: "*/"` (TS-family)
- `LangSCSS`: `linePrefix: "//"`, `blockOpen: "/*"`, `blockClose: "*/"` (SCSS supports both)
- `LangSass`: `linePrefix: "//"`, `blockOpen: "/*"`, `blockClose: "*/"` (Policy α YAGNI; see Notes)
- `LangLESS`: `linePrefix: "//"`, `blockOpen: "/*"`, `blockClose: "*/"` (LESS)
- `LangVue`: `blockOpen: "<!--"`, `blockClose: "-->"` (HTML-level; sub-parsing out of scope)
- `LangSvelte`: `blockOpen: "<!--"`, `blockClose: "-->"` (HTML-level; same policy as Vue)
- `LangERB`: `blockOpen: "<%#"`, `blockClose: "%>"` (ERB comment block — see scope note on trade-off below)
- `LangJinja`: `blockOpen: "{#"`, `blockClose: "#}"` (Jinja2 `{# comment #}` style)
- `LangLiquid`: `blockOpen: "{% comment %}"`, `blockClose: "{% endcomment %}"` (Liquid comment tags)
- `LangMustache`: `linePrefix: "{{!"`, `blockOpen: "{{!--"`, `blockClose: "--}}"` (Mustache/Handlebars)

Note: `.hbs` maps to `LangMustache`. Handlebars is a Mustache superset and shares the
same comment syntax; using one constant follows the existing pattern of grouping
closely-related variants (Shell groups sh/bash/zsh/fish).

ERB grammar trade-off note: `LangERB` uses `blockOpen: "<%#", blockClose: "%>"` rather
than `linePrefix: "<%#"`. The `linePrefix` form uses `strings.HasPrefix(trimmed, prefix)`
(split.go:174) which only matches when the ERB comment marker is at the start of the
trimmed line. Real ERB files commonly have mid-line comments like `<%= val %> <%# note %>`
where the `<%#` is not at line start. The block form uses `strings.Contains(line, "<%#")`
(split.go:166), which catches it anywhere on the line. Trade-off: `blockClose: "%>"` also
appears on expression-output lines like `<%= value %>`. Under Policy α, those lines will
be classified as Comment (same known limitation as `]]` in Lua code context, F28 YAGNI).
HTML comments (`<!-- -->`) inside ERB files are HTML output rendered to the browser — not
ERB-level comments — so they are intentionally excluded from the grammar; they will be
classified as Code. Document this in the test file comments.

Tests: extend the detection table test with all new extensions. Add a `TestSplit_Templating`
table-driven test covering at minimum: one Vue `<!-- -->` comment, one Jinja `{# #}`
comment, one Mustache `{{!-- --}}` block comment, one JSX `/* */` block comment, one ERB
`<%# comment %>` mid-line occurrence (verifies block form catches it), and one ERB
`<%= value %>` line (verifies the Policy α limitation is acknowledged in test comments).

README "Languages detected": append the 12 new names alphabetically.

**Acceptance:**
1. `mage test` passes.
2. `Detect` on each new extension returns the correct Language constant.
3. `.hbs` resolves to `LangMustache` (not `LangUnknown`).
4. `.tsx` resolves to `LangTSX`, distinct from `.ts` → `LangTS`.
5. `Split` with `LangVue` on `<!-- comment -->` counts 1 Comment line.
6. `Split` with `LangJinja` on `{# comment #}` counts 1 Comment line.
7. `Split` with `LangMustache` on `{{!-- comment --}}` counts 1 Comment line.
8. `Split` with `LangERB` on a line containing `<%# note %>` counts 1 Comment line (mid-line block form).
9. README lists the 12 new languages alphabetically.
10. `mage build` passes.

**Blocked by:** A.2

---

### Unit A.4 — Config and data formats

**State:** todo

**Paths:**
- `internal/lang/lang.go`
- `internal/lang/split.go`
- `internal/lang/lang_test.go`
- `internal/lang/split_test.go`
- `README.md`

**Packages:** `github.com/evanmschultz/rak/internal/lang`

**Scope:**
Add 11 `Language` constants (all new, not yet in tree):

Config formats:
- `LangINI Language = "ini"` — `.ini` extension
- `LangEnv Language = "env"` — `.env` extension (also matches `config.env`, etc.)
- `LangEditorConfig Language = "editorconfig"` — `.editorconfig` extension
- `LangProperties Language = "properties"` — `.properties` extension
- `LangHCL Language = "hcl"` — `.tf`, `.tfvars`, `.hcl` extensions (Terraform/HCL)
- `LangNix Language = "nix"` — `.nix` extension

Data/schema formats:
- `LangProto Language = "proto"` — `.proto` extension
- `LangGraphQL Language = "graphql"` — `.graphql`, `.gql` extensions
- `LangCSV Language = "csv"` — `.csv` extension
- `LangTSV Language = "tsv"` — `.tsv` extension
- `LangJSONL Language = "jsonl"` — `.jsonl`, `.ndjson` extensions

Add all extensions to `extensionTable`. Add grammar entries to `grammarTable`:
- `LangINI`: `linePrefix: ";"`, `linePrefix2: "#"` (semicolon primary, hash secondary)
- `LangEnv`: `linePrefix: "#"` (dotenv standard)
- `LangEditorConfig`: `linePrefix: "#"` (editorconfig spec)
- `LangProperties`: `linePrefix: "#"`, `linePrefix2: "!"` (Java .properties: both `#` and `!`)
- `LangHCL`: `linePrefix: "#"`, `linePrefix2: "//"`, `blockOpen: "/*"`, `blockClose: "*/"` (HCL supports all three)
- `LangNix`: `linePrefix: "#"`, `blockOpen: "/*"`, `blockClose: "*/"` (Nix expression language)
- `LangProto`: `linePrefix: "//"`, `blockOpen: "/*"`, `blockClose: "*/"` (Protocol Buffers)
- `LangGraphQL`: `linePrefix: "#"` (GraphQL SDL; `#` is the only comment form)
- `LangCSV`: absent from grammarTable (no comment syntax; all non-blank lines = Code)
- `LangTSV`: absent from grammarTable (same as CSV)
- `LangJSONL`: absent from grammarTable (JSON Lines; no comment syntax)

Tests: extend the detection table with all new extensions. Add
`TestSplit_ConfigDataFormats` table-driven test covering at minimum: INI `;` comment,
HCL `#` comment, HCL `//` secondary comment, HCL `/* */` block comment, Properties `!`
comment, Nix `#` and `/* */`, GraphQL `#` comment. CSV/TSV/JSONL: verify all non-blank
lines classify as Code.

README "Languages detected": append the 11 new names alphabetically (CSV, dotenv, EditorConfig, GraphQL, HCL/Terraform, INI, JSONL, Nix, Properties, Protobuf, TSV).

**Acceptance:**
1. `mage test` passes.
2. `.tf`, `.tfvars`, `.hcl` all resolve to `LangHCL`.
3. `.graphql` and `.gql` both resolve to `LangGraphQL`.
4. `.jsonl` and `.ndjson` both resolve to `LangJSONL`.
5. A file named `.env` (extension `.env` per `filepath.Ext`) resolves to `LangEnv`.
6. `Split` with `LangINI` on `; comment` counts 1 Comment line.
7. `Split` with `LangHCL` on `# comment`, `// comment`, and `/* block */` each produce 1 Comment line.
8. `Split` with `LangProperties` on `! comment` counts 1 Comment line.
9. `Split` with `LangCSV` on `a,b,c` counts 1 Code line (no grammar = all Code); same assertion for `LangTSV` on `a\tb\tc` and `LangJSONL` on `{"key":"value"}` (all three grammar-less langs must classify all non-blank lines as Code).
10. README lists the 11 new language names.
11. `mage build` passes.

**Blocked by:** A.3

---

### Unit A.5 — Build and task files

**State:** todo

**Paths:**
- `internal/lang/lang.go`
- `internal/lang/split.go`
- `internal/lang/lang_test.go`
- `internal/lang/split_test.go`
- `README.md`

**Packages:** `github.com/evanmschultz/rak/internal/lang`

**Scope:**
Add 6 new `Language` constants (all new, not yet in tree). Vagrantfile and Brewfile
re-use `LangRuby` (same pattern as Gemfile/Rakefile already in tree).

New constants:
- `LangBazel Language = "bazel"` — `BUILD`, `BUILD.bazel`, `WORKSPACE` special filenames + `.bzl` extension
- `LangGroovy Language = "groovy"` — `Jenkinsfile` special filename (Groovy = Java-like)
- `LangJust Language = "just"` — `Justfile`, `justfile` special filenames
- `LangEarth Language = "earth"` — `Earthfile` special filename (Earthly build tool)
- `LangCaddy Language = "caddy"` — `Caddyfile` special filename
- `LangProcfile Language = "procfile"` — `Procfile` special filename

No new Language constant for Vagrantfile/Brewfile — they map to `LangRuby` (same as Gemfile). Add to `specialFilenames`:
- `"build"` → `LangBazel`
- `"build.bazel"` → `LangBazel`
- `"workspace"` → `LangBazel`
- `"jenkinsfile"` → `LangGroovy`
- `"justfile"` → `LangJust`
- `"earthfile"` → `LangEarth`
- `"caddyfile"` → `LangCaddy`
- `"procfile"` → `LangProcfile`
- `"vagrantfile"` → `LangRuby`
- `"brewfile"` → `LangRuby`

Add `.bzl` to `extensionTable` → `LangBazel`.

Add grammar entries to `grammarTable`:
- `LangBazel`: `linePrefix: "#"` (Starlark = Python-like hash comments)
- `LangGroovy`: `linePrefix: "//"`, `blockOpen: "/*"`, `blockClose: "*/"` (Java-family)
- `LangJust`: `linePrefix: "#"` (Justfile uses `#` comments)
- `LangEarth`: `linePrefix: "#"` (Earthly syntax uses `#` comments)
- `LangCaddy`: `linePrefix: "#"` (Caddyfile uses `#` comments)
- `LangProcfile`: absent from grammarTable (no comment syntax; all non-blank = Code)

Tests: add `TestDetect_BuildTaskFiles` table-driven test covering all new special
filenames (e.g. `BUILD`, `BUILD.bazel`, `WORKSPACE`, `Jenkinsfile`, `Justfile`,
`justfile`, `Earthfile`, `Caddyfile`, `Procfile`, `Vagrantfile`, `Brewfile`) and the
`.bzl` extension. Add `TestSplit_BuildFiles` covering Bazel `#` comment, Groovy `//`
comment and `/* */` block comment.

Also add a `--lang bazel` smoke to `TestDetect_BuildTaskFiles` (inside
`internal/lang/lang_test.go`): construct a `fstest.MapFS` containing files named `BUILD`,
`BUILD.bazel`, `WORKSPACE`, and `foo.bzl`; verify that `Detect` on each returns
`LangBazel`. This smoke lives entirely inside the `internal/lang` package — it does NOT
touch `cmd/rak/integration_test.go` or any `cmd/rak` path. No A.5 Paths or Packages
expansion needed.

README "Languages detected": append Bazel, Caddyfile, Earthfile, Groovy (Jenkinsfile),
Justfile, Procfile — in alphabetical order. Vagrantfile and Brewfile map to Ruby (already
listed); note in the README description that these filenames are detected as Ruby.

**Acceptance:**
1. `mage test` passes.
2. `Detect` on `BUILD`, `BUILD.bazel`, `WORKSPACE` each returns `LangBazel`.
3. `Detect` on `foo.bzl` returns `LangBazel`.
4. `Detect` on `Jenkinsfile` returns `LangGroovy`.
5. `Detect` on `Justfile` and `justfile` both return `LangJust`.
6. `Detect` on `Vagrantfile` returns `LangRuby`.
7. `Detect` on `Brewfile` returns `LangRuby`.
8. `Detect` on `Procfile` returns `LangProcfile`.
9. `Split` with `LangGroovy` on `// comment` + `/* block */` input counts correct Comment lines.
10. `Split` with `LangBazel` on `# comment` counts 1 Comment line.
11. `Split` with `LangProcfile` on `web: bundle exec rails server` counts 1 Code line.
12. README lists the 6 new language names (Bazel, Caddyfile, Earthfile, Groovy, Justfile, Procfile).
13. `mage ci` passes from `main/`.

**Blocked by:** A.4

---

## Notes

**Cross-stream coordination**: this is one of four v0.2.0 streams (A=langs, B=tokens, C=parallel-walk+follow, D=files-from). Stream A is isolated to `internal/lang/*` plus README — it does NOT touch `cmd/rak/root.go`, so no flag-wiring contention with B/C/D.

**Serial chain rationale**: All five units share the same five paths. Parallelism is structurally impossible without per-file splitting that would be artificial over-engineering. The chain A.1 → A.2 → A.3 → A.4 → A.5 serializes correctly.

**XML split (A.1)**: The only unit that modifies an existing entry. `extensionTable[".xml"]` changes from `LangHTML` to `LangXML`. `detectContent`'s `<?xml` branch changes from returning `LangHTML` to `LangXML`. No existing test asserts `.xml` → `LangHTML` (verified: `TestDetect_ByExtension` table does not include a `.xml` row), so no existing test breaks.

**XML behavior change (v0.2.0 release note)**: Before DROP_A, `.xml` files appeared as `html` in `total_by_lang` output. After A.1, they appear as `xml`. This is an intentional v0.2.0 behavior change. Builder must call it out in the PR description. The A.1 acceptance criteria record this explicitly (item 7).

**Lua block comments**: Lua's `--[[ ... ]]` long-bracket syntax is assigned `blockOpen: "--[["` and `blockClose: "]]"`. Policy α known limitation: `]]` also appears as a table-index operator in Lua code. Lines containing `]]` in code context are mis-classified as Comment (same YAGNI trade-off as F28). Additionally, `]]` inside a Lua string literal (e.g., `s = "array[i][j]]"`) can corrupt multi-line block-comment state across subsequent lines — the state machine exits the block-comment on the `]]` even when it was inside a string. Acknowledged; accepted under Policy α YAGNI.

**ERB grammar trade-off**: `LangERB` uses `blockOpen: "<%#", blockClose: "%>"` (block form) rather than `linePrefix: "<%#"` (line-start form). The line-start form (`strings.HasPrefix`) misses mid-line ERB comments like `<%= val %> <%# note %>`. The block form (`strings.Contains`) catches them. Trade-off: `%>` appears on expression-close lines like `<%= value %>`, which will be mis-classified as Comment under Policy α. HTML comments (`<!-- -->`) inside `.erb` files are HTML output written to the browser — they are NOT ERB-level comments — so they are intentionally excluded from the ERB grammar; those lines classify as Code.

**Sass `.sass` grammar**: Indented Sass uses `//` for line comments; `/* */` block comments exist but are rarely used in `.sass` files. Grammar uses both under Policy α YAGNI — some non-comment lines may be over-classified. Acceptable for v0.2.0.

**Vue/Svelte `<script>` limitation**: `LangVue` and `LangSvelte` are assigned `blockOpen: "<!--", blockClose: "-->"` (HTML-level comment grammar). The bulk of real source logic lives inside `<script>` blocks, which use JS/TS comment syntax (`//`, `/* */`). Those comments are invisible to rak's grammar and will classify as Code. Known limitation; sub-parsing is out of scope per design principle 2 ("one file = one language"). Document in test file comments.

**Templ HTML-comment fallback**: `LangTempl` is assigned Go-style comments (`linePrefix: "//"`, `blockOpen: "/*"`, `blockClose: "*/"`). Templ files also contain HTML-like template blocks where `<!-- -->` comments may appear. Those HTML comments will classify as Code under the Go-style grammar. Same known limitation as Vue/Svelte — single-grammar policy for v0.2.0.

**HCL triple-comment forms**: HCL accepts `#`, `//`, and `/* */`. The grammar struct accommodates this via `linePrefix="#"`, `linePrefix2="//"`, `blockOpen="/*"`, `blockClose="*/"`.

**Vagrantfile / Brewfile / Gemfile / Rakefile symmetry**: Vagrantfile and Brewfile map to `LangRuby` (same as existing Gemfile/Rakefile pattern). No new Language constant for any of these — they are Ruby DSLs, and the existing `LangRuby` constant is correct. Procfile is different: it is its own deployment-config micro-format (key: command pairs), not a Ruby DSL, and has no comment syntax. Procfile gets its own `LangProcfile` constant for `--lang procfile` filtering. This asymmetry is intentional.

**Groovy constant naming**: `LangGroovy` is used (not `LangJenkinsfile`) because Groovy is the actual language. If standalone `.groovy` files are added in a future drop, this constant is already correct.

**Procfile**: No standard comment syntax. Language constant `LangProcfile` is created for `--lang procfile` filtering, but `grammarTable` has no entry — all non-blank lines classify as Code. Same applies to `LangCSV`, `LangTSV`, `LangJSONL`: no comment syntax in their specs; all non-blank lines are Code by default when absent from `grammarTable`.

**`.env` extension handling**: `filepath.Ext(".env")` returns `".env"` in Go (the leading dot is the extension separator for a basename-only dotfile). Adding `".env"` to `extensionTable` correctly matches files named `.env`, `development.env`, `config.env`, etc.

**Naming conventions for new constants** (all follow lowercase single-word rule):
- `LangCSharp = "csharp"` (not "c#" — invalid Go string but also the conventional name)
- `LangFSharp = "fsharp"` (same reasoning)
- `LangEditorConfig = "editorconfig"` (one word, lowercase)
- `LangJSONL = "jsonl"` (acronym, all-caps in constant name; value lowercase)
- `LangHCL = "hcl"` (acronym, all-caps in constant name; value lowercase)

**README format (locked in A.1, inherited by A.2–A.5)**: The current "Languages detected" paragraph holds ~22 entries. After DROP_A it will hold ~52 entries. The paragraph form becomes unreadable at that size. A.5 builder must convert to an alphabetical comma-separated list. A.1–A.4 builders may keep the paragraph form if they judge it still readable at intermediate counts, but the A.5 builder must switch. Full format decision is in the A.1 scope section.

**`mage ci` at A.5**: Only Unit A.5's acceptance criteria include `mage ci`. Units A.1–A.4 specify `mage build` + `mage test`. The full `mage ci` gate (including lint, coverage, gofumpt) is reserved for the final unit per the drop's Phase 6 verify step. This is standard WORKFLOW.md practice.
