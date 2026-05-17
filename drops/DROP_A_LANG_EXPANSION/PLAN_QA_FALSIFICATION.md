# DROP_A — PLAN_QA_FALSIFICATION

## Round 1

**Verdict:** PASS WITH FINDINGS

The plan is structurally sound, the serial-chain rationale is correct, and the
`<?xml` regression-guard claim is verified (no existing test asserts
`<?xml` → `LangHTML`). Most attacks I ran against the plan were REFUTED on
inspection of `lang.go` / `split.go`. Eight findings remain — none are blockers
preventing build but several need planner action before Phase 4 to avoid
predictable build-QA failures.

## Counterexamples / Attacks

### Attack 1 — `.R` extension lowercasing claim

- **Severity:** REFUTED (no finding)
- **Where:** Unit A.2, acceptance #4
- **Counterexample / hypothesis:** I suspected `filepath.Ext(".R")` would return
  `.R` verbatim and that `extensionTable[".r"]` would miss. **REFUTED**:
  `lang.go:154` already does `ext := strings.ToLower(filepath.Ext(f.RelPath))`.
  Plan claim is accurate.
- **Mitigation:** none required. Worth adding a one-line note in A.2's acceptance
  #4 pointing at `lang.go:154` so the builder doesn't introduce a `.R`-keyed
  duplicate entry "just in case."

### Attack 2 — ERB `<%# ... %>` mid-line comments are never counted

- **Severity:** concern
- **Where:** Unit A.3, ERB grammar entry
- **Counterexample / hypothesis:** Plan assigns `LangERB`:
  `linePrefix: "<%#", blockOpen: "<!--", blockClose: "-->"`. The splitter
  (`split.go:174`) only treats `linePrefix` as a comment when the trimmed line
  *starts* with it (`strings.HasPrefix(trimmed, g.linePrefix)`). The ERB idiom
  is `<%# inline comment %>` mid-line; the most common case is appended to an
  existing tag, e.g. `<%= user.name %> <%# show name %>`. That line will
  classify as Code, not Comment. Result: ERB comment counts will be
  systematically under-reported.
- **Mitigation accepted:** change ERB grammar to
  `blockOpen: "<%#", blockClose: "%>"` (parallel to Jinja `{# #}` and Liquid
  `{% comment %} ... {% endcomment %}` — block markers, not line prefix).
  Drop the `linePrefix: "<%#"` field. Keep the HTML `<!-- -->` block as a
  *secondary* concern — but the grammar struct only supports one blockOpen /
  blockClose pair, so the planner must choose: ERB-tag comments (better signal,
  per-block) OR HTML comments (less common in ERB templates). Recommend
  ERB-tag comments win; document the HTML-block trade-off in PLAN.md Notes.

### Attack 3 — Liquid `{%- comment -%}` whitespace-trim variant won't match

- **Severity:** concern
- **Where:** Unit A.3, Liquid grammar entry
- **Counterexample / hypothesis:** Plan assigns Liquid
  `blockOpen: "{% comment %}", blockClose: "{% endcomment %}"`. The splitter
  uses `strings.Contains(line, g.blockOpen)` (literal substring). Real-world
  Liquid frequently uses the whitespace-trim form `{%- comment -%}` /
  `{%- endcomment -%}` which won't match the literal. Result: trim-form
  comments classify as Code, not Comment.
- **Mitigation accepted:** either (a) accept under Policy α YAGNI and document
  the limitation in PLAN.md Notes (same pattern as Lua `]]`-in-strings), OR
  (b) extend Unit A.3 to add a second block-grammar pair (would require a
  schema change to `grammar` struct — beyond DROP_A scope). Recommend (a) with
  an explicit one-liner in Notes. Add a comment in `grammarTable` near the
  Liquid entry pointing at the limitation so future readers don't think it's
  an oversight.

### Attack 4 — Vue / Svelte `<script>` block comments never counted

- **Severity:** concern
- **Where:** Unit A.3, Vue and Svelte grammar entries
- **Counterexample / hypothesis:** Plan assigns Vue and Svelte
  `blockOpen: "<!--", blockClose: "-->"` (HTML-zone only). But typical `.vue`
  and `.svelte` files are ~70-90% `<script>` block JS/TS code with `//` line
  comments and `/* */` block comments. Those will be counted as Code, not
  Comment. Comment counts for Vue / Svelte projects will be dramatically
  under-reported relative to a real `cloc`-style tool.
- **Mitigation accepted:** the "one file = one language" design principle
  (PLAN.md scope item 2) explicitly accepts this. But the plan should
  **explicitly document the limitation** in PLAN.md Notes (Vue/Svelte comment
  counts under-report `<script>` JS/TS comments by design) so the dev doesn't
  get surprised by build-QA test cases that look correct in isolation but
  produce confusingly low numbers on real Vue projects. Also surface for dev
  review in Phase 3 — there may be a willingness to add an HCL-style triple
  grammar (`linePrefix: "//", blockOpen: "<!--", blockClose: "-->"`) that
  approximates the JS+HTML mix.

### Attack 5 — Lua `]]` in source code mis-classified as block-close

- **Severity:** nit (already covered by Notes)
- **Where:** Unit A.2, Lua grammar
- **Counterexample / hypothesis:** Lua `]]` is also the table-double-index
  operator (e.g. `t[k]]`). Policy α (split.go:166-170) flags any line containing
  `g.blockClose` as Comment. So `local x = t[k]]` mis-classifies as Comment.
  Worse: under the block-comment state machine (split.go:188-204), a stray `]]`
  in code can falsely close an open block comment, corrupting the state of
  subsequent lines.
- **Mitigation accepted:** PLAN.md Notes already calls this out
  ("`]]` also appears as a table-index operator in Lua code. Lines containing
  `]]` in code context are mis-classified as Comment"). Already documented.
  Sufficient. **However** — add one sentence noting the secondary effect
  (false block-close in the state machine) since that affects subsequent lines,
  not just the line containing `]]`.

### Attack 6 — Procfile constant inconsistent with Vagrantfile/Brewfile reuse

- **Severity:** concern
- **Where:** Unit A.5, `LangProcfile` constant choice
- **Counterexample / hypothesis:** Unit A.5 creates a dedicated `LangProcfile`
  constant for `Procfile` (no comment syntax, no grammar entry) but explicitly
  reuses `LangRuby` for `Vagrantfile` and `Brewfile`. The asymmetry is
  unjustified: Procfile is a non-Ruby format (just `process: command` lines),
  so giving it its own constant is correct **and** gives the user
  `rak --lang procfile .` filtering — which is the same argument that would
  apply to Vagrantfile/Brewfile (`rak --lang vagrantfile` may be what a user
  wants when filtering an infra repo). PLAN.md Notes claims "same pattern as
  existing Gemfile/Rakefile" — but Gemfile/Rakefile ARE Ruby (literally
  Ruby DSL); Vagrantfile/Brewfile are also Ruby DSL, so the analogy holds
  there. The PLAN's choice is internally consistent on the Ruby-DSL axis, but
  the user-facing `--lang` filter argument cuts the other way.
- **Mitigation accepted:** keep Procfile as `LangProcfile` (correct — it is
  NOT Ruby). Keep Vagrantfile/Brewfile as `LangRuby` (correct — they ARE
  Ruby DSL). But add a sentence to PLAN.md Notes explicitly stating the
  rationale axis: "Vagrantfile / Brewfile / Gemfile / Rakefile are all Ruby
  DSL files — `LangRuby` is the language. Procfile is its own micro-format,
  gets its own constant. Filter granularity is a downstream consequence, not
  the driver." Prevents Phase 5 build-QA from re-litigating.

### Attack 7 — `--lang` filter coverage gap for batches A.2–A.5

- **Severity:** concern
- **Where:** Unit A.5 only — but the gap affects A.2 / A.3 / A.4 silently
- **Counterexample / hypothesis:** Only Unit A.5's acceptance criteria mention a
  `--lang bazel` end-to-end test. Units A.2 / A.3 / A.4 add ~33 new
  Language constants between them; if any of those (say `--lang csv` or
  `--lang hcl` or `--lang vue`) is silently broken at the cmd/rak filter wiring
  layer, none of the per-unit `mage test ./internal/lang/...` acceptance
  criteria will catch it (the lang package has no `cmd/rak` integration).
  Hypothetical failure mode: someone changes how the `--lang` flag parses
  comma-separated values, breaks one new constant's lookup path, and the
  drop ships green because per-unit tests stay in-package.
- **Mitigation accepted:** add a single end-to-end smoke acceptance to **A.5**
  (the last unit, which already has the bazel smoke and runs `mage ci`): run
  `rak --lang csv,hcl,vue,mustache,jsonl,procfile <fixture>` against a
  minimal `fstest.MapFS` fixture (or temp dir) containing one file per
  language and verify each language constant appears in the output. One
  acceptance bullet, ~15-line test, catches the cross-cut. No need to add
  to every unit — A.5 sweep is sufficient since A.5 blocks on A.2-A.4 anyway.

### Attack 8 — `<?xml` detect-content change is undocumented user-visible behavior

- **Severity:** nit
- **Where:** Unit A.1, scope description
- **Counterexample / hypothesis:** A.1 changes `detectContent`'s `<?xml` branch
  to return `LangXML` instead of `LangHTML`. This is a real user-visible
  behavior change: extensionless XML-content files (rare but real — e.g. an
  SVG file accidentally renamed without extension, or an XML stream piped
  from a script) previously detected as HTML now detect as XML. The plan
  treats this as a trivial table edit, but it's a v0.2.0 changelog entry.
- **Mitigation accepted:** add an explicit acceptance criterion to A.1:
  "behavior change documented in PLAN.md Notes — pre-v0.2.0
  extensionless `<?xml`-content files detected as `LangHTML`; post-v0.2.0
  detect as `LangXML`." The dev's `gh release create` flow for v0.2.0 will
  pick it up from the commit log + PLAN.md if it's recorded; otherwise it
  silently lands.

### Attack 9 — README format coupling (paragraph vs table)

- **Severity:** nit
- **Where:** Cross-unit (A.1 through A.5)
- **Counterexample / hypothesis:** PLAN.md Notes punts the README format choice
  to A.1's builder: "consider whether the paragraph form still works or if a
  sorted list/table is clearer." If the A.1 builder switches to a table, every
  subsequent unit's README edit must also use the table format. If A.1 keeps
  the paragraph and A.3's builder decides mid-flight to switch to a table, A.2
  and A.4 are now in a mixed state. Build-QA may not catch the inconsistency.
- **Mitigation accepted:** lock the decision in **A.1**, not "punt to builder
  judgment." Two viable options: (a) keep paragraph, builder appends names
  alphabetically and accepts a longer wrap; (b) switch to a fenced bullet list
  alphabetically sorted, one entry per language (preserves diff readability
  across A.2/A.3/A.4/A.5). Recommend (b) — ~50+ entries by drop end is past
  the paragraph-readability threshold. Whichever the dev picks, lock in
  Phase 3 discussion so A.1's builder receives an unambiguous instruction.

### Attack 10 — `grammarTable` lookup for grammar-less languages is implicit

- **Severity:** nit (verify-only)
- **Where:** Unit A.4 (CSV, TSV, JSONL) and Unit A.5 (Procfile)
- **Counterexample / hypothesis:** Plan correctly notes that
  `LangCSV` / `LangTSV` / `LangJSONL` / `LangProcfile` are deliberately absent
  from `grammarTable`. The contract is "absent → zero grammar → all non-blank
  = Code" — and `split.go:141` confirms this: `g := grammarTable[lang]` returns
  a zero `grammar{}` for missing keys (no panic, all empty strings). So this
  works correctly. **But** — the acceptance criteria on A.4 (#9) and A.5 (#11)
  only test ONE such language each (`LangCSV` → 1 Code line, `LangProcfile` →
  1 Code line). If a future contributor adds a grammar entry for `LangCSV` by
  mistake, only A.4's test catches it; the analogous LangTSV / LangJSONL paths
  go untested.
- **Mitigation accepted:** extend A.4 acceptance #9 to: "`Split` with each of
  `LangCSV`, `LangTSV`, `LangJSONL` returns 1 Code, 0 Comment on a single
  non-blank input line." Table-driven addition, ~5 lines of test. Same shape
  applies to A.5's `LangProcfile` — single-line acceptance is sufficient there
  since there's only one such constant in A.5.

### Attack 11 — Unit merge candidate (YAGNI scrutiny)

- **Severity:** nit
- **Where:** A.4 + A.5 boundary
- **Counterexample / hypothesis:** Both units touch the exact same 5 files,
  both are pure additive map-literal work, both have similar acceptance shape
  (add constants, extend detection table, add grammar where applicable, add
  README entries). The serial chain serializes them anyway. Merging A.4 + A.5
  into one ~17-language batch is a viable YAGNI shrink — one fewer
  builder/QA/commit cycle.
- **Mitigation accepted:** keep as separate units. Rationale: (a) A.5's special
  filenames (`BUILD`, `WORKSPACE`, `Jenkinsfile`) interact with `specialFilenames`,
  not just `extensionTable` — different code path, larger test surface;
  (b) the `mage ci` gate moves to A.5 by design — combining loses the
  per-unit checkpoint. The cost of one extra cycle is small and the
  separation surfaces real test-surface differences. **Accepted with
  rationale: per-unit checkpointing + distinct code paths justify the split.**

## Summary of recommended planner edits before Phase 4

In priority order:

1. **A.3 ERB grammar** — change to block-comment form (`<%#` / `%>`). Drop
   `linePrefix: "<%#"`. (Attack 2)
2. **A.5 cross-cut filter smoke** — add one `rak --lang csv,hcl,vue,...` smoke
   to A.5 acceptance. (Attack 7)
3. **A.1 README format lock** — pick paragraph or bulleted list, instruct A.1
   builder explicitly. (Attack 9)
4. **A.4 grammar-less coverage** — extend Split acceptance to cover all three
   of CSV/TSV/JSONL. (Attack 10)
5. **PLAN.md Notes additions** — Liquid trim variant (Attack 3), Vue/Svelte
   script-block limitation (Attack 4), Lua `]]` state-machine secondary effect
   (Attack 5), Procfile-vs-Ruby-DSL rationale axis (Attack 6),
   `<?xml` → `LangXML` behavior-change changelog entry (Attack 8).

None of these require restructuring the unit decomposition or unblocking the
serial chain. All are surgical edits to existing units / Notes.
