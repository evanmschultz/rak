# DROP_7_SUMMARY_SORTING — BUILDER_QA_FALSIFICATION

Build-QA falsification rounds for Drop 7. Each round appends; no overwrites.

---

## Unit 7.1 — Round 1

**Date:** 2026-05-15
**Verdict:** PASS (no CONFIRMED counterexamples)
**Files reviewed:**
- `internal/summary/summary.go` (new, 53 LOC)
- `internal/summary/sort.go` (new, 93 LOC)
- `internal/summary/summary_test.go` (new, 199 LOC)
**Commit:** `4717207 feat(summary): add directory + summary + sortkey + sortdirs`
**Evidence:** `git diff HEAD~1 -- internal/summary/`, direct Read of all three files, `mage ci` (green), `go doc slices.SortFunc`, Hylla `LangCounts` lookup.

### Attack pass

| # | Attack surface | Outcome | Notes |
|---|---|---|---|
| 1 | F43 field order (`Path`, `Counts`, `ByLang`, `Files`) for Drop 7.2 bare struct conversion | REFUTED | `summary.go` declares fields exactly in spec order. Confirmed via direct Read. |
| 2 | `SortDirs` stability + tiebreakers (`slices.SortFunc` vs `SortStableFunc`) | REFUTED — spec-authorized | `go doc slices.SortFunc` confirms "not guaranteed to be stable". PLAN.md F38 / unit 7.1 acceptance explicitly mandates `slices.SortFunc` (not `SortStableFunc`). Builder honored spec. Tie-stability is an open design question logged in Unknowns, not a unit 7.1 defect. |
| 3 | `effectiveAsc` inversion for `SortPath` vs numeric keys | REFUTED | Walked four cases manually: `effectiveAsc(SortPath, false)=true → ascending (a,b,c)`; `effectiveAsc(SortPath, true)=false → descending (c,b,a)`; `effectiveAsc(SortLines, false)=false → descending (20,10,5)`; `effectiveAsc(SortLines, true)=true → ascending (5,10,20)`. All four match the test expectations in `TestSortDirs_Path_{Default,Asc}` and `TestSortDirs_Lines_{Default,Asc}`. |
| 4 | Unknown-key panic message format | REFUTED | Panic emits `"summary: SortDirs called with unrecognized SortKey %q"` via `fmt.Sprintf`. Unit 7.1 acceptance only says "descriptive message" — no canonical wording mandated at this layer. The CLI-layer canonical message (`"X is not a valid sort key; valid keys: lines, files, bytes, path"`) is unit 7.3's responsibility. |
| 5 | No external deps (only stdlib + internal/counting + internal/lang) | REFUTED | `sort.go`: `cmp`, `fmt`, `slices`, `strings` (all stdlib). `summary.go`: `internal/counting`, `internal/lang` (internal only). `summary_test.go`: `testing`, `internal/counting`. Zero external imports. |
| 6 | `Files int64` consistency vs `int` | REFUTED | `counting.Counts.Bytes/Lines/Words/Chars` are all `int64`. PLAN.md F35 spec-mandates `int64`. `cmp.Compare[int64]` works correctly in `SortDirs`. Consistent across the domain. |
| 7 | `mage ci` re-verification | REFUTED | Re-ran `mage ci`: golangci-lint `0 issues`, `go test -race ./...` all packages `ok` (most cached). |
| 8 | F26 RelPath invariant | REFUTED | `summary.Directory.Path` is a raw `string` field with no normalization. F26 (forward-slash) is upstream (`fileset`); summary stores whatever caller provides — correct boundary. |

### Go-quality sweep (additional attack families)

- **Concurrency / race:** no goroutines, no shared state, no channels. `SortDirs` mutates the caller's slice in place and documents that explicitly. `mage test` runs with `-race`; no race detected. EXHAUSTED.
- **Interface misuse:** no interfaces in this unit. N/A.
- **Error swallowing:** no error-returning paths in this unit (struct defs + sort func, no I/O). N/A.
- **Raw go commands:** no shell-outs in the diff. Verified via `mage ci`, not `go test`. REFUTED.
- **`mage install` invocation:** no reference anywhere in the diff. REFUTED.
- **YAGNI pressure:** `effectiveAsc` helper is justified by the `SortPath` inversion logic (vs numeric keys) — extracting it makes the comparator readable and the inversion auditable in one place. All four `SortKey` constants have documented purposes. No premature abstraction. REFUTED.
- **Hidden dependencies / `init()`:** no `init()` functions, no package-level state, no test-order coupling. REFUTED.
- **File/package gating:** all three new files live under `internal/summary/` (within declared `paths`). No edits outside scope. REFUTED.
- **Doc comment coverage (project rule 11):** `SortKey`, `SortLines`, `SortFiles`, `SortBytes`, `SortPath`, `effectiveAsc` (internal but documented), `SortDirs`, `Directory`, `Summary`, plus every exported field — all carry doc comments. REFUTED.
- **Test coverage:** 8 sort-key cases (4 keys × 2 directions) + 2 edge cases (empty, single) + 1 panic case. Matches acceptance criteria exactly. EXHAUSTED.

### Counterexamples

None. Zero CONFIRMED counterexamples after 8 named attack surfaces and 10 Go-quality attack families.

### Unknowns

- **Tie-stability for `SortDirs`:** spec mandates `slices.SortFunc` (not stable). Two `Directory` entries with equal primary key (e.g., equal `Files`) have non-deterministic relative order. Output reproducibility (snapshot tests, diff-friendly emission) may want a secondary tiebreaker (most natural: `Path` ascending). This is NOT a unit 7.1 defect — the spec explicitly chose `SortFunc` — but it is worth surfacing to the dev as a downstream design question for Drop 7.2/7.3 (snapshot tests) or a later drop. Routed: this round's verdict; orchestrator may surface to dev or open a separate drop.
- **Panic vs error-return for unknown SortKey:** unit 7.1 panics; unit 7.3 will validate upstream at the CLI layer. The panic is correct under the unit-7.1-only contract ("CLI validates first"), but means library callers outside `cmd/rak` who skip validation would crash the binary. Acceptable for v0.1.0 with no external API; flag if rak ever exposes a stable package API.

### Hylla Feedback

None — Hylla answered everything needed (`LangCounts` lookup, `Counts` struct cross-check). The summary package was queried directly via `Read` since the diff was small and local.

### Falsification certificate

- **Premises:** unit 7.1 must ship `Directory`, `Summary`, `SortKey` constants (Lines/Files/Bytes/Path), and `SortDirs` with documented in-place mutation, key-specific direction defaults (numeric desc, path asc), and panic on unknown key. F43 field order must match the future `directoryJSON` exactly.
- **Evidence:** `git diff HEAD~1 -- internal/summary/`, full Read of all three files, `mage ci` green, `go doc slices.SortFunc` for stability claim, Hylla for `lang.LangCounts` confirmation.
- **Trace or cases:** 8 attack surfaces named in the spawn prompt + 10 Go-quality attack families. All REFUTED or EXHAUSTED with reason.
- **Conclusion:** PASS. No CONFIRMED counterexample. Unit 7.1's claim that the summary package satisfies F35/F36/F38/F41/F42/F43 is upheld.
- **Unknowns:** Tie-stability (spec-authorized but worth dev visibility); panic-vs-error for library callers (acceptable for v0.1.0).
