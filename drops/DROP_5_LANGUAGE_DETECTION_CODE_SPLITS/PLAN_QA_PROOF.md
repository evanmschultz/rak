# Drop 5 — Plan QA Proof, Round 2

**Verdict:** pass

## Summary

The Round 2 revise comprehensively addresses every Round 1 finding the dev
accepted (C1 blocker; C2 + C4 majors; P2 + C3 merged major; P1 + P3 + P4 + P5 +
P6 + C5 + C6 + C7 + C8 sweep). The chain restructure `5.1 → {5.2 || 5.4} → 5.3`
is consistently expressed in three places (line 23 Planner header, line 25
rationale paragraph, line 220 Notes § "Parallel eligibility"), and the dep-edge
reasoning replaces the Round 1 "same-package serialization" framing.

Each Round 1 issue traces to concrete revised text. Two new F-pins were added
(F33; F27 + F28 were materially rewritten to absorb C2, C4, C5, C6, C7, P3, P5).
Acceptance test rosters in 5.1, 5.2, 5.3 now include the canonical test names
called out in the appendix. The two-step `walkAndCount` ownership story
(`detectedLang` introduced in 5.1, consumed in 5.4 and 5.3) is explicit at three
points (line 60 — 5.1 wires; line 146 — 5.4 consumes; line 186 — 5.3 calls
`lang.Split` per file using the same `detectedLang`).

Two minor findings (P-R2-1, P-R2-2) — both nits, neither blocks build. Both
are framing-drift in non-authoritative prose, easily folded into a Phase 3
discussion or deferred entirely.

## Resolution audit (Round 1 → Round 2)

| Round 1 finding | Severity | Status | Evidence pointer |
|---|---|---|---|
| C1 — Scope/Planner contradiction on parallelism | blocker | resolved | line 23 (Planner header), line 25 (chain rationale), line 220 (Notes § Parallel eligibility) — all three carry `5.1 → {5.2 || 5.4} → 5.3`. 5.1 Paths (lines 31–34) include `cmd/rak/root.go` + `cmd/rak/root_test.go`. 5.4 Blocked by `5.1` (line 156). 5.3 Blocked by `5.2, 5.4` (line 200). Dep-edge reasoning at lines 222–225 replaces "same-package" framing. |
| P2 + C3 — `LangUnknown` suppression unspecified | major (merged) | resolved | F33 added at line 216 (F-pin register). 5.3 acceptance line 177 explicitly cites F33; lines 183–184 describe per-renderer LangUnknown filtering. Three new tests at lines 194–197 carry the exact canonical names from the appendix (`TestTOONRenderer_RenderTree_AllUnknown`, `TestJSONRenderer_RenderTree_AllUnknown`, `TestHumanRenderer_RenderTree_AllUnknown`). |
| C2 — `LangMakefile` unreachable; missing special-filename step | major | resolved | F27 rewritten at line 210 to describe a 4-step pipeline starting with special-filename lookup. 5.1 acceptance lines 44–48 carry the 4-step list with step 1 = special-filename. `LangDocker` and `LangCMake` constants added at line 43. `TestDetect_SpecialFilename` added at line 53 covering `Makefile`, `makefile` (case-insensitive), `Dockerfile`, `CMakeLists.txt`, `GNUmakefile`. Keys explicitly include `gnumakefile`, `cmakelists.txt`. |
| C4 — `Split` block-comment state machine under-specified | major | resolved | F28 rewritten at line 211 to pin **Policy α**. 5.2 acceptance lines 85–91 carry Policy α with 4 canonical examples (`/* a */ b /* c */`, `x := 1 /* note */`, `/* still open`, `closing */ x := 2`, `x := 1`). Three new tests at lines 119–121 lock the canonical inputs: `TestSplit_BlockCommentOpenClosePerLine`, `TestSplit_TrailingComment`, `TestSplit_StringContainsMarker_KnownLimitation`. |
| P1 — chain rationale conflates "same package" with "must serialize" | major | resolved | Notes § "Parallel eligibility" lines 222–225 explicitly cite dep edges + disjoint file sets ("5.2 vs 5.4 — no symbol dependency and disjoint file sets" + enumeration of which files each touches). "Same package" framing absent from Round 2. |
| P3 — shebang priority spec under-defined | medium | resolved | F27 line 210 + 5.1 acceptance step 3 line 47: shebang sniff runs "only when steps 1 + 2 both returned `LangUnknown`". `Peek` error semantics pinned: "`Detect` never propagates `Peek` errors; callers must not depend on them." |
| P4 — double-IO trade-off | medium | resolved | Notes § "Double-IO trade-off" at lines 230–231 explicitly discusses the trade-off and Drop 8.1 cost amplification. 5.3 acceptance line 186 cites P4 + the double-open trade-off. Builder permitted `io.TeeReader` with documentation. |
| P5 — extension table key shape | minor | resolved | F27 line 210 + 5.1 acceptance line 46 + line 49: keys are lowercase WITH the leading dot, matching `filepath.Ext` directly. |
| P6 — `LangCounts` accumulator helper unnamed | nit | resolved | 5.2 acceptance line 79: `func (lc *LangCounts) Add(other LangCounts)` added. Cited at line 79 ("P6") and consumed in 5.3 acceptance line 186 ("Use `LangCounts.Add` (from 5.2) to accumulate"). |
| C5 — "generic language" intermediate state under-defined | minor | resolved | F27 line 210 + 5.1 acceptance step 4 line 48: explicit "There is NO 'generic language' intermediate state — the pipeline returns the first concrete match OR `LangUnknown`." |
| C6 — `Language` constant case + filter normalization | minor | resolved | F27 line 210 + 5.1 acceptance line 41: "Language values are stored lowercase by convention." 5.4 acceptance line 147: "Case-insensitive match: normalize user input with `lang.Language(strings.ToLower(v))`." Both sides of the comparison are now lowercase. |
| C7 — Python docstrings classification | minor | resolved | F28 line 211 + 5.2 acceptance line 92: "Python docstrings (C7): triple-quoted strings are strings at the language level, not comments. `Split` classifies them as Code." |
| C8 — `lang.Detect` call-site ownership | nit | resolved | 5.1 acceptance line 60: "5.1 only wires the `Detect` invocation and imports `internal/lang`." 5.4 acceptance line 146: "Consumes the `detectedLang` value already wired by Unit 5.1 (no second `lang.Detect` call)." 5.3 acceptance line 186: "(after the lang-filter gate added in 5.4, using the `detectedLang` wired in 5.1)". Three points agree. |

All twelve Round 1 issues addressed.

## New findings (Round 2)

### Finding P-R2-1 — Scope paragraph's unit-summary parenthetical still describes 5.1 as "internal/lang detection" only

- **Severity:** nit
- **Unit/F-pin affected:** Scope paragraph (line 17)
- **Claim:** The Scope paragraph (line 17) lists the four units as
  "(5.1 internal/lang detection / 5.2 code-aware splits / 5.3 per-type
  aggregation in render / 5.4 `--lang` walk filter)". Round 2 expanded 5.1 to
  also wire `lang.Detect` into `cmd/rak/root.go`'s `walkAndCount` (lines 33,
  60). The Scope's one-liner does not mention the call-site wiring.
- **Why nit:** the Planner-section unit header (line 27 — "Unit 5.1 —
  internal/lang: Language type + detection, plus Detect call-site wiring in
  cmd/rak") and the Acceptance criteria are authoritative. The Scope
  parenthetical is summary prose, not a contract; the planner-section trumps
  it. No risk of orchestrator dispatch confusion. But Scope is what a reader
  hits first — keeping it in sync reduces cognitive friction in Phase 4 / 5.
- **Suggested fix:** when the planner next touches PLAN.md (e.g., on a Phase
  3 re-run or any future revise), update the parenthetical to "5.1 internal/lang
  detection + Detect call-site wiring / 5.2 code-aware splits / 5.3 per-type
  aggregation in render / 5.4 `--lang` walk filter". Otherwise defer — the
  orchestrator can also fix this inline since it is markdown summary text,
  not Go code.

### Finding P-R2-2 — F27 sub-clause "case-insensitive special-filename" implicitly requires basename lookup; pipeline step description does not pin basename vs full path

- **Severity:** nit
- **Unit/F-pin affected:** Unit 5.1 step 1 (line 45); F27 (line 210)
- **Claim:** Step 1 says "consult inline `specialFilenames map[string]Language`
  (case-insensitive key lookup; normalize with `strings.ToLower`). Keys
  include at minimum: `"makefile"`, `"gnumakefile"`, `"dockerfile"`,
  `"cmakelists.txt"`." It does not explicitly say which value to look up —
  `f.RelPath` (full path) or `filepath.Base(f.RelPath)` (basename). For a
  file `tools/dev/Makefile`, `strings.ToLower(f.RelPath)` =
  `"tools/dev/makefile"` — which does NOT match the table key `"makefile"`.
  The intent is clearly basename lookup, but the prose says "key lookup;
  normalize with `strings.ToLower`" without naming the lookup-value
  extraction.
- **Why nit:** the test `TestDetect_SpecialFilename` (line 53) uses paths
  like `Makefile`, `makefile`, `Dockerfile`, `CMakeLists.txt`, `GNUmakefile` —
  these are all basename-equivalent. Any builder reading the table keys
  ("makefile", "dockerfile") will infer basename lookup is intended; an
  alternative interpretation produces obviously broken behavior caught by
  the first walked Makefile in a nested subdir. Build-QA will catch this if
  it slips through.
- **Suggested fix:** change step 1's prose at line 45 to
  "consult inline `specialFilenames map[string]Language`. Look up
  `strings.ToLower(filepath.Base(f.RelPath))`; if match → return
  immediately." One-token edit. Add a `tools/dev/Makefile` row to
  `TestDetect_SpecialFilename` to lock basename-not-full-path semantics
  explicitly. Acceptable to defer to build-QA if the planner prefers not to
  edit PLAN.md for nits.

## Unknowns

None new. The three Open Unknowns in PLAN.md (U1 — TOON/human per-type
format level of detail; U2 — `--lang unknown` as filter value; U3 — per-type
rollup always-on vs opt-in) remain appropriately routed to dev for Phase 3
discussion. No new unknowns surfaced by the Round 2 revise.

## Hylla Feedback

N/A — Round 2 plan-QA touched non-Go files only (PLAN.md is markdown). No
Hylla queries needed; no fallbacks taken.
