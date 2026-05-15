# DROP_7 — Plan QA Falsification

## Drop 7 — Plan QA Falsification Round 2

**Reviewer:** go-qa-falsification-agent
**Round:** 2
**Verdict:** FAIL — 1 CONFIRMED counterexample + 2 sub-pin observations

Round 1 produced 2 CEs + 3 sub-pins; all folded into the Round 2 revise. Round 2 attacks the new surface only.

---

### Attack 1 — C1 key-specific direction truth table

**Attempted:** Walk all four corners of (key × `--sort-asc`) against the plan's pin at line 55 (`effectiveAsc` helper: numeric returns `asc`, path returns `!asc`).

| key | `--sort-asc` (asc) | effectiveAsc | meaning |
|---|---|---|---|
| lines | absent (false) | false | desc ✓ |
| lines | present (true) | true | asc ✓ |
| files | absent (false) | false | desc ✓ |
| files | present (true) | true | asc ✓ |
| bytes | absent (false) | false | desc ✓ |
| bytes | present (true) | true | asc ✓ |
| path | absent (false) | true | asc ✓ |
| path | present (true) | false | desc ✓ |

Cross-checks against:
- Unit 7.1 acceptance line 102 test list: matches.
- Unit 7.3 acceptance lines 142-148 test list: matches.
- Notes F39 (line 161): matches.

**Verdict: REFUTED.** All four corners internally consistent across spec, helper, and tests.

---

### Attack 2 — SortDirs panic vs error on unknown key

**Attempted:** Plan line 99 pins panic-on-unknown. CLI validator at line 137 rejects before RunE. So no in-tree caller reaches the panic.

Trade-off:
- `internal/` package; no external consumers. Panic-on-programmer-error is a defensible Go idiom (cf. `slices.Index`'s undefined behavior for invalid args).
- Future internal callers might forget validation. Risk is bounded because every `internal/summary` consumer lives in this repo.
- Unit 7.1 acceptance test list does not include a panic test. Coverage gap is small and intentional ("callers validate first" — line 99).

**Verdict: EXHAUSTED — no counterexample.** Acceptable internal-API idiom; missing panic test is a minor coverage gap but not a falsification.

---

### Attack 3 — JSON output `sort` field carry-over

**Attempted:** Does the plan claim the JSON output should expose the chosen sort key as a `"sort": "<key>"` field?

Plan does not make this claim. Current `treeJSON` envelope (json.go:83-87) has `Directories`, `Total`, `Errors` only. Plan does not extend the envelope. The sort is reflected by the order of the `Directories` array — sufficient for a deterministic JSON consumer.

**Verdict: REFUTED.** No such claim to attack.

---

### Attack 4 — Test count breakdown consistency

**Attempted:** Plan says "10 test cases (4 keys × 2 directions + 2 special cases)" — verify the breakdown.

Counting Unit 7.3 acceptance test list (lines 140-149): 1 default + 4 keys × 2 directions + 1 tokens-error = 10. The breakdown `4×2 + 2` = 8 + 2 = 10 matches (the two specials: default-no-flag, tokens-error).

**Verdict: REFUTED.** Count and breakdown match.

---

### Attack 5 — Struct field order F43 pin completeness

**Attempted:** Verify F43 is pinned for BOTH `summary.Directory` AND `directoryJSON`.

- Unit 7.1 acceptance line 100: pins `summary.Directory` field order as `Path, Counts, ByLang, Files`.
- Unit 7.2 acceptance line 121: pins `directoryJSON` field order as `Path, Counts, ByLang, Files`.
- Notes F43 (line 165): summarizes both.

Go struct-conversion rule allows `directoryJSON(d)` when underlying types match ignoring tags — same field sequence, same field types. Both pins line up; conversion compiles.

**Verdict: REFUTED.** F43 pinned on both sides.

---

### Attack 6 — Sort-after-label ordering (Decision 3.3)

**Attempted:** Plan line 58 + line 138 pin: `labelDirectories(dirs, rootLabel)` → `summary.SortDirs(dirs, key, asc)` → `RenderTree`. Does sorting labeled paths produce a different order than sorting raw paths would?

For `--sort path`:
- `labelDirectories` adds a constant prefix (rootLabel) to every entry except `.` → rootLabel.
- Adding the same prefix to every string preserves their relative lexical order (modulo the `.` → bare rootLabel case, which is a strict prefix of every labeled child, so root sorts first under asc — correct).
- Concrete: with rootLabel = `./testdata/tree`, entries `.`, `also`, `sub`, `sub/nested` become `./testdata/tree`, `./testdata/tree/also`, `./testdata/tree/sub`, `./testdata/tree/sub/nested`. Lexical asc order is identical to the unlabeled lexical asc order.

For numeric sorts (`--sort lines/files/bytes`): label doesn't touch `Counts` or `Files`, so numeric ordering is unaffected by labeling.

**Verdict: REFUTED.** Sort-after-label preserves intended semantics; labeling does not perturb either path or numeric orders.

---

### Attack 7 — Round 2 new bugs (scan revise for fresh contradictions)

**Attempted:** Scan the revise text for new logical errors, especially in unit acceptance criteria.

**Sub-attack 7a — Files propagation through struct reconstruction sites: CONFIRMED counterexample.**

Plan acceptance for Unit 7.2 (line 120-123) calls out:
- Renderer signature changes from `[]render.Directory` to `[]summary.Directory`.
- `directoryJSON` gains `Files int64`.
- `walkAndCount` return type changes; accumulation loop adds `dir.Files++`.
- `labelDirectories` updated to return `[]summary.Directory`.

Plan does NOT call out propagating the new `Files` field through reconstruction sites. Concrete sites that **reconstruct** a Directory struct from a source Directory:

1. **`labelDirectories` (root.go:397-414):** lines 404 and 407-411 currently build `render.Directory{Path: ..., Counts: d.Counts, ByLang: d.ByLang}` — explicitly enumerating three fields. After 7.2, the builder mechanically expands to `summary.Directory{...}` but without an explicit "propagate Files" pin, the builder will likely produce the same three-field reconstruction and silently zero `Files`.
2. **`filterUnknown` (json.go:60-78):** lines 73-77 reconstruct `Directory{Path: d.Path, Counts: d.Counts, ByLang: filtered}` — same pattern. After 7.2 this becomes a `summary.Directory` reconstruction, and without an explicit pin, Files is dropped.

**Reproduction trace:**
- Walk produces `summary.Directory{Path: ".", Counts: c, ByLang: m, Files: 17}` in walkAndCount.
- `labelDirectories` reconstructs as `summary.Directory{Path: rootLabel, Counts: d.Counts, ByLang: d.ByLang}` → Files = 0 (zero value).
- `SortDirs` with `--sort files` now sees all entries with Files = 0 → meaningless ordering, slices.SortFunc tie-break behavior.
- `RenderTree` (JSON) → `filterUnknown` reconstructs again → Files stays 0.
- JSON output: `"files"` field omitted via `omitempty` → user-facing JSON loses the per-directory file count.

**Why tests don't catch this:**
- Unit 7.1 tests SortDirs against a synthetic `[]Directory` slice with directly-set Files values — bypasses labelDirectories/filterUnknown.
- Unit 7.3 tests CLI flag wiring with synthetic `[]summary.Directory` (line 150) — bypasses the same.
- Integration tests (integration_test.go) are stdin-only — no directory walk, no labelDirectories.
- `omitempty` on directoryJSON.Files hides the zero in snapshot tests.
- `mage ci` passes clean even though production JSON output is broken for `--json` directory walks.

**Verdict: CONFIRMED CE.** Plan must pin: every struct reconstruction site (`walkAndCount` final-loop construction, `labelDirectories`, `filterUnknown`) explicitly propagates `Files` through.

**Recommended fold (sub-pin F44):** Unit 7.2 acceptance gains: "All struct reconstruction sites — the `walkAndCount` final loop building `[]summary.Directory`, `labelDirectories` rebuild branches, and `filterUnknown` reconstruction — propagate `Files` from source to destination. F44 pins this." Add a test in Unit 7.2 or 7.3 that walks a real fixture with multiple files-per-dir, asserts `dir.Files > 0` survives labelDirectories AND `filterUnknown`, and that `--sort files` produces a non-degenerate order on real fixture data.

**Sub-attack 7b — Error string format inconsistency: sub-pin observation.**

Three different specifications of the unrecognized-`--sort` error string:
- Line 34 (AcceptanceCriteria): `"X is not a valid sort key"` — no quotes around X, no key list.
- Line 56 (ContextBlocks Decision 3.4): `"X is not a valid sort key; valid keys: lines, files, bytes, path"` — no quotes around X, includes key list.
- Line 137 (Unit 7.3 acceptance): `"\"X\" is not a valid sort key; valid keys: lines, files, bytes, path"` — X is double-quoted, includes key list.

Builder will pick one form; `TestRootCmd_SortTokens_Errors` will assert against builder's choice. Tests will pass with any consistent choice, but the spec is internally inconsistent.

**Verdict: sub-pin observation, not a hard CE.** Pin one canonical error string format and use it in all three places.

**Recommended fold:** Pick line 137's form (quoted X for readability when X contains punctuation) and propagate to lines 34 and 56.

**Sub-attack 7c — `dir.Files++` phrasing precision: sub-pin observation.**

Line 122 says: "accumulation loop adds `dir.Files++` per accepted file". In current root.go (lines 262-339), the per-file walk loop has no `dir` variable bound to a Directory — the loop uses `byDir map[string]counting.Counts` keyed by string path. `dir` is the path key (string), not a Directory struct. The builder needs to either:
- Add a parallel `byDirFiles map[string]int64` accumulator and consume it during the final `dirs := make([]render.Directory, ...)` construction; OR
- Change `byDir` to a `map[string]*directoryAccumulator` or similar richer value.

Plan's loose phrasing "`dir.Files++`" is ambiguous about which path the builder takes.

**Verdict: sub-pin observation.** Pin: "Add a parallel `byDirFiles map[string]int64` accumulator. Increment per accepted file. Consume during the `[]summary.Directory` construction so each directory's `Files` field reflects the accepted count."

**Sub-attack 7d — other Round 2 inconsistencies scanned:** none beyond the above.
- Test count: 10 = 8 + 2 — consistent.
- F35–F43 pin enumeration in Notes — internally consistent.
- Decision 3.3 / 3.4 references — consistent across three sites each.
- `SortName` retired → `SortPath` — pinned once at line 98, no contradictions.

---

### Hylla Feedback

None — Round 2 falsification touched only markdown (plan), Go reconstruction sites (read via `Read`), and prior project context. No Hylla queries needed for non-Go files; the one Go read (`json.go`, `render.go`, `root.go`) was direct because the relevant question was *current* code shape vs. *planned* code shape — `git diff`-style territory, not Hylla territory.

---

### Round 2 Verdict

**FAIL — 1 CONFIRMED counterexample + 2 sub-pin observations.**

Fold list for the planner:
1. **CE (mandatory):** Pin Files propagation through all struct reconstruction sites (F44). Add a real-fixture test that asserts `--sort files` produces a non-degenerate order and `--json` output preserves non-zero `files`.
2. **Sub-pin:** Canonicalize the error string format for unrecognized `--sort` values. Pick one of the three variants and use it consistently at lines 34, 56, 137.
3. **Sub-pin:** Clarify the `dir.Files++` mechanism — explicitly pin a parallel `byDirFiles map[string]int64` accumulator (or equivalent) in Unit 7.2 acceptance.
