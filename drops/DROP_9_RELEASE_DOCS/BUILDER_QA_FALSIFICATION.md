# Drop 9 — Builder QA Falsification

## Unit 9.0 — Round 1

**Verdict:** PASS (no counterexamples found).
**Tier:** B — sole QA gate, no proof companion.
**Working dir:** `/Users/evanschultz/Documents/Code/hylla/rak/main`.
**Commit under review:** `1d9ef24 feat: add per-lang totals and coverage gate (drop 9 units 9.0 + 9.3)`.

### Premises

- Unit 9.0 wires F46 (per-language grand totals collapsed across all directories) into the rendering pipeline across all three renderers, with F33 (LangUnknown suppression) uniformly applied.
- The change introduces `summary.Summary.TotalByLang map[lang.Language]lang.LangCounts` and refactors the `Renderer.RenderTree` signature from `(w, dirs, total, errs)` to `(w, s summary.Summary, errs)` — Option A.
- `walkAndCount` returns the new `totalByLang` map alongside its existing return values; `runDirectory` constructs the `summary.Summary` and passes it through.

### Evidence

- `git diff HEAD~1 -- internal/summary/ internal/render/ cmd/rak/` showed: `internal/summary/summary.go` (+13 −0), `internal/render/render.go` (+14 −8), `internal/render/human.go` (+33 −6), `internal/render/json.go` (+39 −14), `internal/render/toon.go` (+59 −32 after walkthrough), `cmd/rak/root.go` (+23 −9), test files updated to match.
- `mage ci` from `/Users/evanschultz/Documents/Code/hylla/rak/main`: `0 issues` from lint, all 8 packages `ok`, coverage `87.8% (floor: 70.0%, scope: ./internal/...)` — coverage gate from Unit 9.3 green.
- `mage test`: all 8 packages `ok` (cached, fresh-built earlier this session per commit `1d9ef24`).
- Hylla `hylla_refs_find` / `hylla_search_keyword "renderer.RenderTree"`: only production caller is `cmd/rak.runDirectory`; all test callers updated (verified via `Read` of `internal/render/render_test.go:160-316` and the `TestRenderer_TotalByLang_*` family at lines 700-872).
- Hylla snapshot 7 docstrings still reflect pre-9.0 state because reingest is drop-end-only — not a counterexample, expected per project rules.

### Trace or cases — attack surface results

1. **F33 LangUnknown uniformity (TOON / human / JSON).** REFUTED.
   - TOON `toon.go:176`: `knownTotalLangs := sortedKnownLangs(s.TotalByLang)` — `sortedKnownLangs` (human.go:185-194) filters out `LangUnknown` (= `""`, lang.go zero value).
   - Human `human.go:103`: same `sortedKnownLangs(s.TotalByLang)`.
   - JSON `json.go:136`: `TotalByLang: filterTotalByLangUnknown(s.TotalByLang)` (json.go:110-124) — returns nil when filtered map is empty, deletes `LangUnknown` key otherwise.
   - End-to-end coverage: `TestRenderer_TotalByLang_LangUnknownSuppressed` (render_test.go:809-872) tests all three renderers with an all-`LangUnknown` `TotalByLang` and asserts absence of `total_by_lang` / `total lang:` in output. Passes.

2. **F46 aggregation correctness.** REFUTED.
   - `root.go:420-423`: increment of `totalByLang[detectedLang]` sits at the same gating point as `byDirLang` (lines 412-418), immediately AFTER `acceptedFiles++` and the `--max-files` early-return.
   - Both blocks consume the same `lang.LangCounts{Lines: lineCounts, Counts: fileCounts}` value, so the walk-level rollup equals the sum of the per-dir rollups by construction.
   - `LangCounts.Add` (lang/split.go:41-49) is plain int64 field addition — commutative + associative.
   - End-to-end verified by `TestRootCmd_TotalByLang_EndToEnd` (root_test.go:1108-1197) which compares `TotalByLang[lang].Counts.Bytes` against `sum(Directories[].ByLang[lang].Counts.Bytes)`.

3. **Option A signature — call sites updated.** REFUTED.
   - Production: only `runDirectory` (root.go:272) — updated to pass `summary.Summary{Dirs, Total, TotalByLang}`.
   - Tests: all 18+ `RenderTree` call sites in `internal/render/render_test.go` rewritten to construct `summary.Summary{}` (verified at lines 167-173, 210-213, 233-236, 257, 279-285, 308, plus the `TotalByLang_*` family).
   - `mage ci` green confirms no broken call sites compile-wise; `mage test` green confirms behavioral pass.

4. **`omitempty` on empty TotalByLang.** REFUTED.
   - JSON `json.go:102`: `json:"total_by_lang,omitempty"` + `filterTotalByLangUnknown` returns nil → encoding/json omits the field.
   - TOON `toon.go:90`: `toon:"total_by_lang,omitempty"` + `totalLangRows` is `var ... []toonTotalLangRow` nil-initialized when no known langs → toon-go omitempty drops the field (spike-confirmed C7 per toon.go:85 comment).
   - Human: no explicit "Total by language:" header; rows are appended in a `for _, l := range knownTotalLangs` loop which is a no-op when the slice is empty.
   - Coverage by `TestRenderer_TotalByLang_LangUnknownSuppressed` (per (1) above).

5. **Sort order alphabetical / stable.** REFUTED.
   - `sortedKnownLangs` (human.go:191-193): `sort.Slice(out, func(i, j int) bool { return string(out[i]) < string(out[j]) })` — ascending string sort. Used by both human and toon.
   - JSON: `encoding/json.Marshal` sorts map keys lexically by default (stdlib contract).
   - `TestRenderer_TotalByLang_Human` (render_test.go:783-804) verifies `total lang: go` and `total lang: markdown` both appear and ordering relative to `total` block is correct.

6. **Aggregate math vs per-dir sums.** REFUTED.
   - Single increment site for `totalByLang` (lines 421-423) consumes the same `LangCounts{Lines: lineCounts, Counts: fileCounts}` that the per-dir `byDirLang` increment consumes immediately above. Sum equality is structural.
   - `TestRootCmd_TotalByLang_EndToEnd` confirms `sum(per-dir bytes) == TotalByLang[lang].Counts.Bytes` for Go (26 bytes from 2× 13-byte files) and Python (12 bytes from 2× 6-byte files).

7. **`directoryJSON` byte-for-byte mirror still holds (F34).** REFUTED.
   - 9.0 adds `TotalByLang` at the envelope level (`treeJSON`, json.go:99-104), NOT at `directoryJSON`.
   - `summary.Directory` field order (summary.go:19-39): Path, Counts, ByLang, Files. `directoryJSON` (json.go:58-63): Path, Counts, ByLang, Files. Still match — `directoryJSON(filterUnknown(d))` at json.go:139 compiles + executes.

8. **F44 reconstruction sites unaffected.** REFUTED.
   - `filterUnknown` (json.go:74-93) still propagates `Files` at line 91 verbatim.
   - `labelDirectories` (not touched by diff — verified by `git diff` scope).

9. **`mage ci` + coverage gate green.** REFUTED.
   - `mage ci` output tail: `total: (statements) 87.8%`, `coverage: 87.8% (floor: 70.0%, scope: ./internal/...)` — 17.8 points above the floor.
   - Lint: `0 issues`. All packages `ok`.

#### Additional self-attacks (QA-Falsification self-loop)

A. **Race / concurrency on `totalByLang`?** REFUTED — `walkAndCount` consumes `source.List(ctx)` (iter.Seq2) sequentially; no goroutine spawn inside the loop.

B. **Map-value mutation idiom bug?** REFUTED — `tlc := totalByLang[detectedLang]; tlc.Add(...); totalByLang[detectedLang] = tlc` is the standard Go pattern; identical to the `byDirLang` block immediately above which already passed prior QA.

C. **Associativity of `Add`?** REFUTED — plain int64 field addition only.

D. **`--lang` filter interaction.** REFUTED — increment is post-filter (line 421 sits after the `continue` at line 372). Same gating point as `byDirLang`.

E. **Schema drift in existing JSON snapshot tests.** REFUTED — `TestJSONRenderer_RenderTree_Snapshot` (render_test.go:290-293) keeps the no-TotalByLang snapshot, and `omitempty` correctly suppresses the new field when the test fixture doesn't populate it. Snapshot still matches.

F. **`--max-files` partial-results inconsistency.** REFUTED — when max-files trips, `walkAndCount` discards all results (`return nil, counting.Counts{}, nil, nil, err`), so any partial inconsistency between `byDir`/`total` and `byDirLang`/`totalByLang` is never visible to callers. Pre-existing pattern from before 9.0.

### Conclusion

PASS. Zero counterexamples constructed across nine declared attack surfaces plus six follow-on self-attacks. Build / test / lint / coverage gates all green. The Option A signature refactor is consistently applied at the only production call site (`runDirectory`) and every test call site. F33 LangUnknown suppression is implemented uniformly across all three renderers via two clean primitives (`sortedKnownLangs` for human/toon, `filterTotalByLangUnknown` for JSON). F46 aggregation is co-located with the existing per-dir `byDirLang` increment so sum equality is structural.

### Unknowns

None requiring routing.

### Hylla Feedback

None — Hylla answered every needed query (call-site discovery via `hylla_search_keyword "renderer.RenderTree"`, structural confirmation via `hylla_search_keyword "LangUnknown"`, and inbound-refs probe via `hylla_refs_find`). Snapshot 7 is pre-9.0 ingest state (drop still open), so docstrings reflect pre-change behavior — this is expected, not a miss.

## Unit 9.3 — Round 1

**Verdict:** PASS (no counterexamples found).
**Tier:** B — sole QA gate, no proof companion.
**Working dir:** `/Users/evanschultz/Documents/Code/hylla/rak/main`.
**Commit under review:** `1d9ef24 feat: add per-lang totals and coverage gate (drop 9 units 9.0 + 9.3)`.

### Premises

- Unit 9.3 flips `mage coverage` from report-only to a 70% floor gate on the `-coverpkg=./internal/...` scope (decision 22).
- The change adds `const coverageFloor = 70.0`, extends `Coverage()` to capture `go tool cover -func` output via `sh.Output`, parse the `total:` line, and return a non-zero error when the percentage is below the floor.
- `CI()` now serial-deps `Coverage` after `Test` (`mg.SerialDeps(gofumptClean, Lint, Test, Coverage)`).
- `.github/workflows/ci.yml` is unchanged — its sole step already runs `mage ci`, which now transitively invokes the floor check.

### Evidence

- `git diff HEAD~1 -- magefile.go .github/workflows/ci.yml`: `magefile.go` +59 −5 (const, `strconv` import, `Coverage()` extension, new `parseCoverageTotal` helper, `CI()` SerialDeps appended); `ci.yml` zero-byte diff.
- `Read` of `magefile.go:22-24` (const), `magefile.go:61-67` (CI SerialDeps), `magefile.go:112-176` (Coverage + parser).
- `Read` of `.github/workflows/ci.yml:40-41` confirms `mage ci` is the only run step — no additional CI hop needed.
- `Read` of `.gitignore:6` — `/coverage.out` already gitignored from earlier drops.
- Empirical `mage coverage` run from `main/`: emits `total: (statements) 87.8%` followed by `coverage: 87.8% (floor: 70.0%, scope: ./internal/...)` and exits 0. 17.8 percentage points above floor.
- Empirical `mage ci` run from `main/`: full chain green, coverage gate fires within the chain.
- `git status --short`: only the (untracked) `BUILDER_QA_FALSIFICATION.md` itself in the working tree — `coverage.out` produced by the run is correctly suppressed by `.gitignore`.
- Hylla snapshot 7 has no record of `parseCoverageTotal` (`hylla_search_keyword "parseCoverageTotal"` returned 0 hits) — expected, drop-end-only ingest + magefile is `//go:build mage` package main outside the indexed module surface. Not a miss.

### Trace or cases — attack surface results

1. **Coverage parser correctness against format drift.** REFUTED.
   - `parseCoverageTotal` (magefile.go:158-176) iterates lines, matches `strings.HasPrefix(line, "total:")`, splits with `strings.Fields` (collapses multiple tabs/spaces), takes the LAST field, strips trailing `%`, parses as float64.
   - The actual `go tool cover -func` output uses multiple tabs (`total:\t\t\t\t\t\t\t\t(statements)\t\t\t87.8%`) — `strings.Fields` handles arbitrary whitespace correctly, so multi-tab variation is no risk.
   - 4-digit percentages (`100.0%`): `TrimSuffix("100.0%", "%")` = `"100.0"`, `ParseFloat` → 100.0. Passes floor. REFUTED.
   - Single-digit / `0.0%`: parses to 0.0, fails floor — correct loud failure. REFUTED.
   - Missing `total:` line: loop terminates without return → `fmt.Errorf("no total: line found...")` → wrapped at Coverage call site as `"mage coverage: parse total: ..."`. REFUTED.
   - Malformed total line with `< 3` fields: explicit `len(fields) < 3` guard returns an error. REFUTED.
   - Non-numeric percentage (e.g. `"foo%"`): `ParseFloat` returns error, wrapped via `%w`. REFUTED.
   - Stray non-summary line starting with `"total:"` — `go tool cover -func` produces `<path>:<line>:\t<func>\t<pct>` for function rows; no function row starts with `"total:"` because function rows always begin with the file path. REFUTED.

2. **Floor boundary `<` vs `<=` semantics.** REFUTED.
   - `if pct < coverageFloor { return err }` (magefile.go:142) — strict `<`. At exactly 70.0%, condition is false → Coverage returns nil. PLAN.md acceptance says "exit non-zero if below"; "below" excludes 70.0. Const + comparison + error-message printf all use the same `coverageFloor = 70.0` value, so any future tweak is single-source.

3. **`mage ci` Coverage ordering.** REFUTED.
   - `mg.SerialDeps(gofumptClean, Lint, Test, Coverage)` (magefile.go:65) — Coverage is the LAST entry, runs after Test. mage's `SerialDeps` documented contract is sequential execution in declared order; later targets see filesystem state after prior targets complete.
   - Test target already runs `go test -race ./...`. Coverage then runs `go test -race -coverpkg=./internal/... -coverprofile=coverage.out ./...`. Tests are invoked twice in mage ci (Test then Coverage's internal `go test`). This is a minor performance inefficiency, not a correctness counterexample — both invocations exercise the same test bodies and pass, and `mg.SerialDeps` cannot dedupe because the flag sets differ. Worklog acknowledges Coverage runs last "most expensive; only worth doing if tests are already green" — fast-fail ordering is preserved.

4. **`coverage.out` race between Coverage and Test.** REFUTED.
   - `mg.SerialDeps` enforces sequential execution — Test cannot run concurrently with Coverage within a single `mage ci` invocation. Coverage owns `coverage.out` writes; Test does not touch it.
   - Concurrent `mage` invocations from a developer's shell would race, but build automation is single-threaded by convention (no real attack surface).

5. **Floor wired consistently.** REFUTED.
   - `const coverageFloor = 70.0` (magefile.go:24), `if pct < coverageFloor` (line 142), `fmt.Printf("coverage: %.1f%% (floor: %.1f%%, ...)", pct, coverageFloor)` (line 140), error printf `%.0f%% floor` (line 144) — all four sites reference the same constant. The error-message printf uses `%.0f` (prints "70" without decimal) while the info printf uses `%.1f` ("70.0") — minor cosmetic asymmetry but does not affect gate correctness.

6. **CI workflow not picking up new behavior.** REFUTED.
   - `.github/workflows/ci.yml` step `Run mage ci` (line 41) executes `mage ci`. Since `CI()` now invokes Coverage via SerialDeps, GitHub Actions transitively runs the floor check on every push/PR. No yaml edit needed — confirmed by zero-byte diff and Read of ci.yml.

7. **Error message clarity on coverage failure.** REFUTED.
   - When `pct < floor`, returned error is `"coverage 65.0% is below the 70%% floor (scope: ./internal/...)"`. Cites: actual percentage, floor percentage, scope. Wrapped at Coverage call site / mage ci as the chain's surfaced failure. Sufficient for a developer to diagnose without re-reading the mage source.

8. **Drop 0-8 spine preservation.** REFUTED.
   - The prompt's attack surface 8 framed "git diff HEAD~1 -- internal/ cmd/rak/ should be empty" assuming HEAD~1 was just the 9.3 commit. The single commit `1d9ef24` bundles BOTH unit 9.0 AND 9.3 (squashed feat), so the spine diff is from 9.0, which closed PASS in its own Round 1 review above. Unit 9.3's actual `paths` per `drops/DROP_9_RELEASE_DOCS/PLAN.md` line 94 are `main/magefile.go` + `main/.github/workflows/ci.yml` — and the diff scoped to those two files (+59 −5 in magefile, 0 in ci.yml) matches exactly. Spine packages are untouched by 9.3.

9. **`mage ci` re-verify includes the floor check firing.** REFUTED.
   - Empirical `mage ci` invocation produced the chained output `total: (statements) 87.8%` followed by `coverage: 87.8% (floor: 70.0%, scope: ./internal/...)`. The floor-check printf fires before the comparison; comparison passes; CI returns nil. Gate is active.

#### Additional self-attacks (QA-Falsification self-loop)

A. **`sh.Output` swallows stderr?** REFUTED — `sh.Output` documented contract returns stdout as string; stderr from `go tool cover -func` (rare; tool typically uses stdout) would not be parsed. If stderr-only output appeared, `parseCoverageTotal` would return "no total: line found" error and Coverage would fail loudly. Acceptable.

B. **Float precision near boundary.** REFUTED — `go tool cover -func` emits `%.1f` percentages (1 decimal place). Smallest representable boundary is 69.9% (fails) vs 70.0% (passes). IEEE-754 representation of 70.0 is exact; no precision wobble. The float comparison `pct < 70.0` is deterministic for cover's 1-decimal output domain.

C. **Concurrent `mage coverage` invocations.** REFUTED — single-threaded build automation convention; not a real attack surface.

D. **`-coverpkg=./internal/...` matches no packages.** REFUTED — would emit `total: (statements) 0.0%` or no total line; either path fails loudly via the parser. Plus this matches every existing `internal/<pkg>` so empty-match is hypothetical.

E. **Parser bypass via raw `go` calls in agents.** REFUTED — the rule "never run raw `go test`/`go build`" is documented in CLAUDE.md; this unit reinforces the funnel by gating coverage inside `mage ci`. Cannot find a counterexample in the diff (no new raw `go` invocation introduced — all go invocations are inside `Coverage()` which is itself a mage target).

F. **Decision 22 scope drift.** REFUTED — `-coverpkg=./internal/...` matches decision 22 (cmd/rak excluded). Worklog cites decision 22 explicitly. PLAN.md acceptance line 97 cites decision 22. No drift.

G. **Coverage gate triggers BEFORE tests cleanup.** REFUTED — `Coverage()` runs `go test ... -coverprofile=coverage.out` (which writes the profile on success), then `go tool cover -func=coverage.out`, then parses. If tests failed, the first `sh.RunV` returns an error and Coverage exits before parser. Profile is only consumed if tests passed.

H. **`fmt.Println(out)` after `sh.Output` suppresses live test output?** PARTIAL CONCERN, REFUTED on user impact.
   - `sh.Output` only captures the `go tool cover -func` step (line 126), NOT the test run (line 117 uses `sh.RunV` which already streams to stdout). So live test output is preserved; only the per-function coverage report is captured-then-echoed. UX equivalent to pre-9.3 behavior. REFUTED.

### Conclusion

PASS. Zero counterexamples constructed across nine declared attack surfaces plus eight follow-on self-attacks. Build / test / lint / coverage gates all green from `main/`. Coverage observed at 87.8% — 17.8 points above the 70.0% floor. CI workflow correctly unchanged because `mage ci` already its sole step. Parser is robust against the only realistic `go tool cover -func` output format and fails loudly on every malformed input considered. Floor comparison uses strict `<` (70.0% passes, anything below fails) consistent with PLAN.md acceptance wording.

### Unknowns

None requiring routing. U1 (current coverage state) resolved: 87.8%, no scope adjustment needed.

### Hylla Feedback

N/A — Unit 9.3 touched only `magefile.go` (build automation, `//go:build mage` package main, not part of Hylla's indexed module surface) and `.github/workflows/ci.yml` (YAML, non-Go). No Hylla queries were applicable. Verified by `hylla_search_keyword "parseCoverageTotal"` returning empty — expected, snapshot 7 predates the commit AND the symbol lives in a mage-only build file. Not a miss.
