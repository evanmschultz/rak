# Plan-QA Proof — Drop 7 (Round 1)

**Drop:** `DROP_7_SUMMARY_SORTING`
**Reviewed:** `main/drops/DROP_7_SUMMARY_SORTING/PLAN.md`
**Round:** 1
**Verdict:** PASS (with 2 low-severity nits worth fixing in-flight; 0 blocking findings)

The plan is buildable, the unit decomposition is correct, F35–F42 cover the load-bearing invariants, and every cross-package consequence is reasoned-through. Two nits are documented for the builder's attention but do not block Phase 4.

---

## 1. Findings

### 1.1 [Axis: spec-conformance] [severity: low] `directoryJSON` field-order constraint is implicit, not stated

The plan says (line 118) that `directoryJSON` gets `Files int64 \`json:"files,omitempty"\`` so the bare struct conversion `directoryJSON(filterUnknown(d))` compiles. Per Go spec, the conversion `T(v)` between two named struct types succeeds iff their **field names, types, AND ordering** are identical (struct tags excluded). The plan correctly specifies F35's `summary.Directory` field order as `Path, Counts, ByLang, Files`, and the current `internal/render/json.go:50–54` `directoryJSON` is `Path, Counts, ByLang`. Appending `Files int64` at the end works.

**But:** if the builder unconsciously alphabetizes (`ByLang, Counts, Files, Path`) or otherwise reorders either struct, the bare conversion silently breaks with a compile error like `cannot convert filterUnknown(d) (value of type summary.Directory) to type directoryJSON`. The plan's RiskNotes (line 44) mentions the constraint indirectly but does not name "matching field order" as a requirement.

- **Evidence:** `internal/render/json.go:50–54` current `directoryJSON` shape; Drop 7 PLAN.md line 118 acceptance bullet; Go spec § "Conversions" — struct type conversions require identical field names, types, and tags (Go 1.8+ relaxes tag-identity, not order-identity).
- **Fix hint:** Add a one-line note in Unit 7.2 Acceptance after the `directoryJSON` bullet: "Field order of `directoryJSON` MUST match `summary.Directory` exactly (`Path, Counts, ByLang, Files`) so the bare struct conversion compiles."

### 1.2 [Axis: acceptance-criteria-coverage] [severity: low] `filterUnknown` location after `Directory` migration is unspecified

`internal/render/json.go:60–78` defines `filterUnknown(d Directory) Directory` — taking and returning the render-local `Directory`. After Unit 7.2 deletes `render.Directory`, this helper must change its parameter and return types to `summary.Directory`. The plan's Unit 7.2 Acceptance lists "TOON `toonDirectory`, human `countsKV`, JSON `directoryJSON` all updated" (line 117) and the bare-conversion update (line 118), but does not explicitly say "rewrite `filterUnknown` to operate on `summary.Directory`".

This is a 5-line change and a competent builder will see it immediately the moment they touch `json.go`. But the plan's CompletionChecklist (lines 73–81) does not surface it as its own checkbox, so it lives only inside the "All three renderer `RenderTree` implementations updated" line. Mild missing-evidence concern; not blocking.

- **Evidence:** `internal/render/json.go:60–78` `filterUnknown` signature; Drop 7 PLAN.md Unit 7.2 Acceptance (lines 115–121) + CompletionChecklist (73–81).
- **Fix hint:** Add either a one-line in Unit 7.2 Acceptance ("`filterUnknown` signature changes to `filterUnknown(d summary.Directory) summary.Directory`") or a CompletionChecklist item ("`filterUnknown` updated to operate on `summary.Directory`"). Either works.

---

## 2. Missing Evidence

None blocking. All F35–F42 pins are testable, all CompletionChecklist items map to runnable mage targets, and every cross-package consequence is explicit.

The minor gaps (filterUnknown migration, field-order discipline) are documented above as nits.

---

## 3. Summary

**Verdict: PASS**

The plan is buildable and correctly decomposed. Six positive findings:

1. **F35–F42 pin coverage is tight.** Each pin maps to one acceptance criterion. F37 explicitly cites F15 (pre-v1.0 internal-only) as the authorization for the `RenderTree` signature change. F42 names `Files int64` as zero-value safe via `omitempty`. F41 documents the `tokens`-omission via a doc comment on `SortKey`.

2. **R-2 (Files field gap) handling is correct.** Existing JSON snapshot tests in `internal/render/render_test.go` (lines 283–286, 353–356) use byte-exact `want` strings. They construct `Directory{}` literals without setting any new fields; after migration to `summary.Directory{Path, Counts, ByLang}` (no `Files`), the zero `Files: 0` value is suppressed by `omitempty` and every existing snapshot continues to match byte-for-byte. **Snapshot survival verified.** Tests that construct non-zero `Files` would land only in Unit 7.3's new test cases for the `--sort files` key — they own those new expected strings, no collision with Drop 5's existing snapshots.

3. **R-3 (interim sort) is acceptably documented.** ContextBlock warning (line 57) states "7.2 KEEPS the lexical path-sort in `walkAndCount` as an interim fallback; 7.3 replaces it with the configurable sort." Unit 7.2 Acceptance (line 119) restates: "inline path-sort in `walkAndCount` RETAINED as interim fallback (will be replaced by 7.3)". Unit 7.3 Acceptance (line 134) restates: "The inline lexical sort from Unit 7.2 is removed from `walkAndCount` at this point". Three-way internal consistency holds; no contradiction.

4. **F33 LangUnknown suppression survives the type move.** `filterUnknown` (json.go), `sortedKnownLangs` (human.go), and the per-lang loop in `toon.go` all live in `internal/render` and operate on `*.ByLang` map values, not on the wrapping type's identity. Moving `Directory` from `render` to `summary` does not strand the filters; they only need parameter-type updates (see Finding 1.2). Hylla confirms `LangUnknown = ""` is a zero-value constant in `internal/lang`; the suppression remains map-key-based and structurally unchanged.

5. **`--sort tokens` rejection path is explicit.** Unit 7.1 Acceptance (line 97) — "Default (key not recognized) falls back to `SortLines` desc". Unit 7.3 Acceptance (line 135) — "An unrecognized value falls back to `SortLines` desc (no error; documented in `SortDirs` doc comment)". This includes `--sort tokens`, which is rejected via the fallback path (not a flag-parse error). F41 backs this with the doc-comment-on-`SortKey` requirement. The plan's choice of "silent fallback to lines desc" rather than "error on unknown key" is defensible (matches the v0.1.0 forgiving-CLI posture); it is also explicit, which is what matters here.

6. **Go 1.21+ `slices.SortFunc` availability is verified.** `go.mod:3` declares `go 1.26.1`. `slices.SortFunc` is stdlib since Go 1.21. Plan's "no new external deps" claim (line 27) holds.

**Renderer.RenderTree signature change authorization** is explicitly grounded in F15 (no external implementers under `internal/`; pre-v1.0). Cross-checked: Hylla shows three `RenderTree` implementations (`humanRenderer`, `jsonRenderer`, `toonRenderer`), all in `internal/render`, all owned by this drop's scope. No external interface implementers. F37 authorization stands.

**No carve-out scenarios.** The plan's "Warning (re: Drop 4 carve-out discipline)" (line 154) correctly observes that since 7.1 produces a clean `internal/summary` package before 7.2 imports it, no `lint exclusions` style carve-out is needed.

**Recommendation:** Address Findings 1.1 and 1.2 by adding two single-line clarifications to Unit 7.2 Acceptance before spawning the builder. Then proceed to Phase 4. Neither nit is a Phase-4 blocker; both are pre-emptive precision.

---

## 4. Hylla Feedback

None — Hylla answered the LangUnknown / Detect / Directory / RenderTree queries on first call via `hylla_search_keyword(content)`. No fallback to `Read`/`Grep` required for committed Go state. Plan-MD reads and current-source reads via `Read` tool were direct file ops, expected for non-Go content + active checkout files.

---

## TL;DR

- **T1:** Two low-severity findings; both are single-line clarifications to Unit 7.2 Acceptance — none block Phase 4. The plan is correctly decomposed, F35–F42 are tight, R-2 / R-3 / F33 carry-forward are sound.
- **T2:** No missing evidence blocks the verdict.
- **T3:** **PASS.** Field-order discipline for `directoryJSON` (Finding 1.1) and explicit `filterUnknown` migration mention (Finding 1.2) are worth adding pre-build, but the plan is buildable as-is.
- **T4:** Hylla zero misses.
