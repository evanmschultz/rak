# Drop 5 — Plan QA Falsification, Round 1

**Verdict:** pass-with-findings

## Summary

The plan is decomposed honestly and the F-pin numbering is sequential (Drop 4 closes at F26; F27–F32 are correct for Drop 5). The library decision (inline table vs go-enry) is defensible and locked. Acceptance criteria are largely test-table-grounded.

Eight counterexamples follow. C1 is a **blocker** — the planner's "no parallelism possible" claim in `## Notes` contradicts the Scope sentence one screen up ("5.2 and 5.4 are eligible to run in parallel after 5.1 closes") AND violates `feedback_parallelize_aggressively`; one of the two passages must be corrected. C2–C4 are **major** — coverage holes (extensionless special filenames, JSON nil-map serialization shape, single-line `*/...//*` and pre-open code positioning). C5–C8 are **minor / nit** — edge cases worth pinning before build.

## Counterexamples

### Counterexample C1 — `## Notes` "no parallelism" contradicts `## Scope` and violates `feedback_parallelize_aggressively`

- **Severity:** blocker
- **Attack target:** `## Planner` opening paragraph (line 23) + `## Notes` § "Parallel eligibility" (lines 184–192) vs `## Scope` (line 17)
- **Construction:** the Scope paragraph asserts, "Per the `feedback_parallelize_aggressively` memory rule, 5.2 and 5.4 are eligible to run in parallel after 5.1 closes (both consume `Language` but neither blocks the other)." The Planner opening then asserts the opposite: "Four atomic units in a strict linear chain: 5.1 → 5.2 → 5.3 → 5.4. No parallelism is possible because each unit depends on types or files produced by its predecessor." The `## Notes` "Parallel eligibility" section reinforces the second framing (5.4 blocked by 5.3 because they share `cmd/rak/root.go` + `cmd/rak/root_test.go`).

  Walking the actual dep graph:
  - 5.2 (split.go, split_test.go in `internal/lang`) needs only `Language` + `Detect` from 5.1. Different files in `internal/lang` than 5.1's `lang.go` / `lang_test.go`. Same-package rule does serialize 5.1 and 5.2 (one builder per package at a time is the rak default), so 5.1 → 5.2 is correct.
  - 5.3 vs 5.4: 5.3 grows `Directory` with `ByLang`, wires `lang.Detect` + `lang.Split` into `walkAndCount`, and threads per-lang aggregation through all three renderers + tests. 5.4 adds a `--lang` flag + a single skip-gate in `walkAndCount` after `lang.Detect` is called. **5.4 strictly needs only 5.1 (`Detect` + `Language`)** for the filter — it does NOT need `LangCounts`, `LineCounts`, or any rollup machinery from 5.2 / 5.3.

  The planner's justification ("5.3 and 5.4 both touch `cmd/rak/root.go` and `cmd/rak/root_test.go` — shared paths — serialized by rule") is the same-file rule, not a true logical dep. That justifies *order* but does not justify the "no parallelism" claim against the Scope paragraph, which the planner itself authored.

  Either:
  - (A) The Scope paragraph is wrong and must be corrected to drop the parallel-eligibility claim, OR
  - (B) The Planner section's "strict linear chain" / "No parallelism" framing is wrong and 5.2 + 5.4 should be marked parallel-eligible after 5.1 closes (with 5.3 serialized after 5.2 in `internal/lang`, and 5.4 after 5.3 in `cmd/rak/root.go`).

  Currently the document says both. The orchestrator cannot dispatch from a contradicting plan.

- **Mitigation status:** UNMITIGATED — internal contradiction in the document.
- **Suggested fix:** the cleaner resolution is **(C) re-scope 5.4 to be parallel-eligible with 5.2 after 5.1 closes.** Concretely: 5.4 becomes a `cmd/rak`-only unit that (i) adds the `--lang` flag, (ii) calls `lang.Detect` on each file early-walk, (iii) skips files not in the filter set BEFORE counting. This unit only imports `internal/lang` for `Detect` and `Language`. 5.3 then handles the per-type aggregation + renderer wiring (and adds its own `lang.Detect` call site, or refactors to share with 5.4's). Same-file collision between 5.3 and 5.4 in `cmd/rak/root.go` is the only ordering reason — and that's a sequence point between 5.3 and 5.4, not a reason to serialize 5.2 → 5.3. So 5.2 || 5.4 after 5.1, then 5.3 after both close. If the planner rejects (C), then pick (A) or (B) explicitly and remove the contradicting paragraph.

### Counterexample C2 — Detection table omits extensionless special filenames (Makefile, Dockerfile, etc.)

- **Severity:** major
- **Attack target:** Unit 5.1 acceptance, `Detect` priority order (lines 37–40), F-pin coverage
- **Construction:** `Detect` priority is extension → shebang → content heuristic. Consider a file named `Makefile` (no extension, no shebang, content is Makefile syntax). The pipeline runs:
  1. `filepath.Ext("Makefile") == ""` → `extensionTable[""]` is undefined (zero-value `LangUnknown`).
  2. Shebang sniff: `Peek(512)` returns the first 512 bytes. A typical Makefile starts with `.PHONY:` or a target line, NOT `#!`. Shebang miss → `LangUnknown`.
  3. Content heuristic: scans for `<?xml`, `<!DOCTYPE`, `{`, `[`, `---`. None match Makefile content. → `LangUnknown`.

  Result: every `Makefile`, `Dockerfile`, `Gemfile`, `Rakefile`, `Vagrantfile`, `Jenkinsfile`, `Procfile`, `CMakeLists.txt` (extension is `.txt` which maps nowhere useful) etc. is `LangUnknown` despite the planner listing `LangMakefile` in the minimum-coverage set (line 36). The named constant exists but no detection path reaches it.

  Pathological case: a file named `Makefile` starts with a `#!/bin/bash` shebang (rare but legal — some projects do this to make Makefiles executable). The pipeline now classifies it as `LangShell`, which is wrong (it's still Makefile syntax). The planner needs a "special filename" lookup that runs **before** the shebang sniff for these extensionless-but-named-by-convention files.

- **Mitigation status:** UNMITIGATED — `LangMakefile` is in the minimum-coverage list, but no detection rule produces it.
- **Suggested fix:** insert a step 1.5 in the `Detect` pipeline: "If `filepath.Ext == ""` AND the basename matches an entry in `specialNamesTable map[string]Language` (e.g., `Makefile` → `LangMakefile`, `Dockerfile` → `LangDockerfile` if added, etc.), return that Language without consulting the shebang." Add `TestDetect_SpecialName_Makefile` to the test table. Either widen the F27 contract to cover special-name priority or add a new F-pin (F33?) for this rule. If special-name lookup is YAGNI for v0.1.0, then drop `LangMakefile` from the minimum-coverage set in line 36 to keep claims and behavior aligned.

### Counterexample C3 — JSON renderer with nil `ByLang` map emits `"by_lang":null`, not omission

- **Severity:** major
- **Attack target:** Unit 5.3 acceptance (lines 121–124, 129), F31 nil-ByLang contract
- **Construction:** Go's `encoding/json` encodes a nil map as the JSON literal `null`, not as an absent field, UNLESS the struct tag carries `omitempty`. The plan states (line 129): `"by_lang"` field with `omitempty` ("Omit (`omitempty`) when nil. Key is `string(lang.Language)`"). That's right for `nil` maps (nil maps satisfy `omitempty`'s emptiness check). BUT:

  1. A non-nil empty map `map[string]...{}` ALSO has `len == 0`, and `encoding/json` treats it as empty for `omitempty` → omitted. Good. But the plan's wording is ambiguous: F31 says "nil when no detection was run; renderer must guard nil/empty". An *empty* non-nil map (no files in this directory had a recognized language) currently would also be omitted — is that the desired behavior, or should it emit `"by_lang":{}` to signal "detection ran, no language found"?

  2. If `walkAndCount` builds the per-dir per-lang map but never inserts a `LangUnknown` entry (because the plan implicitly aggregates only recognized languages), a directory of all-`LangUnknown` files lands in JSON output with NO `by_lang` field — indistinguishable from a directory where `--lang` filtered everything out, OR from pre-5.3 behavior. LLM consumers cannot tell "no detection ran" from "detection ran, nothing detected" from "filter excluded everything".

  3. The `directoryJSON` struct in current code (`internal/render/json.go` lines 43–46) is a flat alias of `Directory` via `directoryJSON(d)` conversion (line 67 of `json.go`). Adding `ByLang` to `Directory` means `directoryJSON(d)` no longer compiles cleanly if `directoryJSON` doesn't grow the `ByLang` field — or if it does, the conversion still works because both structs have identical shape. The plan needs to specify which path it picks: (a) grow `directoryJSON` to mirror `Directory`'s new shape (then the conversion stays simple), or (b) keep `directoryJSON` small and build the JSON shape from `Directory` field-by-field. Builder will improvise without this spec.

- **Mitigation status:** partially mitigated by the `omitempty` mention but ambiguous on (1) and (2), and silent on (3).
- **Suggested fix:** pin in F31 (or a new F-pin) the distinction: `ByLang == nil` → omit from JSON; `ByLang != nil && len == 0` → still omit (treat empty-detected same as no-detection); `LangUnknown` entries are NOT included in `ByLang` aggregations (or explicitly are — pick one). Add an explicit `TestJSONRender_ByLang_Nil_OmitsField` and `TestJSONRender_ByLang_Empty_OmitsField` to render_test.go. Specify (3): the planner should pick path (a) — grow `directoryJSON` to mirror `Directory` — because the existing `directoryJSON(d)` conversion idiom is load-bearing for the snapshot tests.

### Counterexample C4 — `Split` block-comment state machine is under-specified for `*/ ... /*` and mid-line open

- **Severity:** major
- **Attack target:** Unit 5.2 acceptance (lines 66–70), test `TestSplit_Go_InlineBlockComment`
- **Construction:** the plan's state machine is "detect open token → set `inBlockComment = true` for the line and subsequent lines; detect close token while `inBlockComment` → set `inBlockComment = false` for subsequent lines." Three ambiguous cases:

  1. **`/* a */ b /* c */ code` (one line):** open at col 0, close at col 6, open at col 11, close at col 17, then `code`. Scan-once-per-line says: first open found → inBlockComment=true → scan-until-close → close found at col 6 → inBlockComment=false → STOP scanning (single pass). The second `/*` and the `code` after the second `*/` are lost. The line is classified Comment (because the first scan started a block). But the line also contains real code after `*/`. Spec gap: does the state machine consume one open/close per line, or scan-all?

  2. **`code /* unfinished` (line N) then `still open */ more code` (line N+1):** line N has code BEFORE the open. The plan says "A line is classified `Comment` if `inBlockComment` was true at its start, OR if the line contains a block-comment-open and has no code before it." So line N has code before the open → classified Code (per the plan's text). Line N+1 starts `inBlockComment=true` → classified Comment. Then `more code` after `*/` is lost. Spec gap: when a block closes mid-line, what's the classification of the closing line?

  3. **`TestSplit_Go_InlineBlockComment` (line 94) — `"x := 1 /* note */\n"`:** plan says "should be Code; builder may simplify to Comment — document the decision." Allowing "builder may simplify to Comment" undermines the test's value as a regression gate. If two builders make opposite choices the test PASSES for one and FAILS for the other. The acceptance for this test is unfalsifiable.

- **Mitigation status:** UNMITIGATED — three concrete inputs without spec.
- **Suggested fix:** pick one of two policies and lock it in F28:
  - **Policy α (simple, recommended):** at the line level, the classification is whichever rule matches FIRST in this order: (i) if `inBlockComment` is true at line start AND no close token on this line → Comment, advance no state; (ii) if `inBlockComment` is true at line start AND close token exists → Comment, set `inBlockComment=false`, ignore any subsequent open on same line (YAGNI); (iii) if `inBlockComment` is false at line start AND line starts with a line-comment prefix (after `TrimSpace`) → Comment; (iv) if line starts with a block-open prefix and contains no real code before it → Comment + set `inBlockComment=true` if no close on same line; (v) otherwise → Code, but a `/*` token with later `*/` on same line is treated as embedded-comment + Code (state unchanged). Document this exact policy in F28.
  - **Policy β (precise but more work):** track byte-by-byte state, classify by "non-comment code bytes after trim > 0 → Code, else Comment". Reject for v0.1.0 YAGNI.
  - Either way, change line 94's test acceptance from "Code; builder may simplify to Comment" to one of those two deterministic outcomes. Add `TestSplit_Go_BlockOpenAndCloseSameLine_WithTrailingCode` for case (1). Add `TestSplit_Go_BlockCloseAndTrailingCode` for case (2).

### Counterexample C5 — `Detect`'s "yields a generic language" branch in step 2 is undefined

- **Severity:** minor
- **Attack target:** Unit 5.1 acceptance, line 39 step 2
- **Construction:** the spec for step 2 says shebang sniff runs "when extension lookup returns `LangUnknown` OR yields a generic language". There is no definition of "generic language" anywhere in the plan. Which extensions are considered generic? `.sh` (could be bash, zsh, ksh)? `.h` (could be C or C++)? `.m` (could be Objective-C or Matlab)? Without a list, builder will either (a) skip the "generic" branch entirely (shebang only fires on `LangUnknown`) or (b) invent its own definition.

  Concrete consequence: a file `foo.sh` whose first line is `#!/usr/bin/env zsh`. Step 1 returns `LangShell` (assuming `.sh` → `LangShell` in the inline table). Is `LangShell` "generic" enough to trigger step 2? If yes, builder must add a `genericTable` set; if no, step 2's clause is dead code. `TestDetect_ExtensionBeatsShebang` (line 48) tests the opposite direction (extension wins for `.go`), so the test table doesn't reveal which interpretation the planner intended.

- **Mitigation status:** UNMITIGATED — vague wording, no test pins it.
- **Suggested fix:** delete "OR yields a generic language" from step 2 (drop the concept; YAGNI for v0.1.0; documented Unknown to revisit in v0.2). Or add an explicit `genericExtensions = {".sh"}` (or whatever the planner intends) plus a `TestDetect_ShebangOverridesGeneric` test. Pin either choice in F27.

### Counterexample C6 — `--lang` filter case normalization vs `Language` constant values

- **Severity:** minor
- **Attack target:** Unit 5.4 acceptance, line 158
- **Construction:** the plan says "Case-insensitive match: normalize user input with `lang.Language(strings.ToLower(v))`." That works IF every named Language constant value is lowercase (`LangGo = "go"`, `LangRust = "rust"`, ...). But the plan does NOT explicitly require all Language constant string values to be lowercase. Look at line 36: "Named language constants for each entry in the detection table (e.g., `const LangGo Language = "go"`, `LangRust Language = "rust"`, etc.)." — examples are lowercase but no rule enforces all.

  Concrete failure: planner-or-builder picks `LangCPP Language = "C++"` (mixed/uppercase) for readability of console output. User runs `--lang c++` → `strings.ToLower("c++") = "c++"` → no match in the filter set (set contains `Language("C++")`). The Cpp files are silently excluded despite the user's filter intent.

- **Mitigation status:** UNMITIGATED.
- **Suggested fix:** add an explicit invariant to Unit 5.1 acceptance: "All `Language` constant string values are lowercase ASCII with no spaces or punctuation other than `+` (for C++)." Or normalize the filter set with the same `strings.ToLower` transform on both sides at construction time. Add `TestRootCmd_LangFlag_CaseInsensitive` with `--lang GO,Rust` → matches Go + Rust files. Tighten F29 wording (currently silent on case).

### Counterexample C7 — Python triple-quote heuristic in `Split` will mis-count docstrings as code unless explicitly handled

- **Severity:** minor
- **Attack target:** Unit 5.2 acceptance, grammar table line 78 + simplification note
- **Construction:** the grammar table lists Python with `BlockOpen='"""'` and `BlockClose='"""'` (and notes builder may simplify to `#` only). The simplification path classifies all `"""..."""` content as code (no block-comment handling). PEP 257 docstrings — Python's idiomatic "comment" for modules/functions/classes — are then counted as code. A 50-line module-level docstring in `xyz.py` shows 50 Code lines, 0 Comment lines.

  Concrete file:
  ```
  """
  This module does X.
  Multi-line documentation.
  """
  def foo():
      pass
  ```
  6 lines. Builder choosing the simplified `#` path classifies 5 Code, 1 Code (`def foo():`/`pass` are Code) → 6 Code, 0 Comment, 0 Blank. The "documentation" lines are reported as code volume, which materially mis-represents the project's code/comment ratio for LLM-context sizing — rak's stated primary use case (line 15: "LLM-first code sizing").

  The non-simplified path (counting `"""` occurrences mod 2) has its own well-known counterexamples: a single-quoted line containing `"""` inline (rare but legal in raw strings), or a file with an odd number of `"""` tokens (e.g., a string literal containing `"""` inside another quoting scheme).

- **Mitigation status:** partially mitigated by the "document the simplification" instruction, but the F28 limitation pin (strings-as-comments) is the inverse problem; docstrings-as-code is not pinned.
- **Suggested fix:** add an explicit F-pin (F33?) — "Python docstrings are counted as Code, not Comment, in v0.1.0 due to the lack of triple-quote state tracking. Acceptable YAGNI for v0.1.0 but routes to Unknowns for v0.2 (a Python-aware Split path with proper triple-quote tracking)." Or commit to the triple-quote mod-2 heuristic AND add `TestSplit_Python_Docstring_MultiLine` exercising the 6-line case above with the expected counts the planner picks. Either way, the test acceptance must be deterministic.

### Counterexample C8 — `walkAndCount` two-call-to-Detect risk (5.3 + 5.4 ordering)

- **Severity:** nit (escalates if C1 is resolved by (B) or (C) ordering)
- **Attack target:** Unit 5.3 acceptance (line 132) + Unit 5.4 acceptance (lines 157–158)
- **Construction:** if the orchestrator dispatches 5.3 then 5.4 (current plan), 5.3 inserts `lang.Detect(f)` into `walkAndCount` for aggregation. 5.4 then needs the same `lang.Detect` call result for filter rejection. Two paths:
  - 5.4 reuses the variable 5.3 introduced → minimal diff, clean.
  - 5.4 adds its own `lang.Detect(f)` call BEFORE 5.3's call (because the filter must run before counting) → double call. `Detect` calls `f.Peek(512)`, which opens the file twice per file. Not a correctness bug, but doubles the I/O for every file when `--lang` is set.

  The plan does not pin which structure 5.4 should produce. If 5.4 lands after 5.3 with the filter gate AFTER 5.3's `Detect` call, the filter is too late — files are counted then dropped. If 5.4 puts the gate BEFORE `Detect` is callable, it has to add the call itself. The plan should specify: "5.4 reuses the `detectedLang := lang.Detect(f)` variable 5.3 introduced; the filter gate sits between that line and the `counting.Count` / `lang.Split` calls."

  Now apply C1's suggested fix (C): if 5.4 runs BEFORE 5.3 (parallel with 5.2), 5.4 must introduce the `lang.Detect` call itself, then 5.3 reuses it. The plan does not anticipate this ordering and gives no guidance to either unit on who owns the call.

- **Mitigation status:** UNMITIGATED — implementation order is implicit, not specified.
- **Suggested fix:** explicitly state in whichever unit lands first (5.3 in the current plan; 5.4 if C1 is resolved by re-ordering): "This unit introduces the `detectedLang := lang.Detect(f)` call site in `walkAndCount`, immediately after the binary check and before `countFile`. Subsequent units reuse this variable; no further calls to `lang.Detect` per file." Add a one-line F-pin to enforce: "F34 — `lang.Detect` is called at most once per file in `walkAndCount`."

## Unknowns

- The plan's library-choice decision (inline table vs go-enry) is locked, but the inline-table extension coverage (15 languages) is not benchmarked against the typical-rak-target repo distribution. If `rak` runs on a Ruby / Elixir / Zig / Haskell / Kotlin / Swift / Java repo, every file is `LangUnknown` and the per-type rollup is meaningless. Plan acknowledges this implicitly ("Additional entries are welcome but not required") but does not gate it. Recommend: dev decision in Phase 3 — accept v0.1.0 with 15-language coverage and a v0.2 widen-table follow-up unit, or pre-emptively expand the minimum-coverage list now.
- U1 (TOON/human per-type format detail level) is appropriately routed to dev in Phase 3. No falsification action.
- U2 (`--lang unknown` semantics) is appropriately routed. No falsification action.
- U3 (per-type rollup always-on vs opt-in) is appropriately routed. No falsification action.
