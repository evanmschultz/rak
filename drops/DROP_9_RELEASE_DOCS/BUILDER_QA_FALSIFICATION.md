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

## Unit 9.2 — Round 1

**Verdict:** PASS (no counterexamples found).
**Tier:** B — sole QA gate, no proof companion.
**Working dir:** `/Users/evanschultz/Documents/Code/hylla/rak/main`.
**Commit under review:** `24ac87d feat(cmd): wire fang.withversion at v0.1.0`.

### Premises

- Unit 9.2 wires `fang.WithVersion(version)` into `fang.Execute` in `cmd/rak/main.go`, with `version` declared as `const version = "v0.1.0"` (hardcoded for v0.1.0 per PLAN.md U2; build-time `-ldflags` deferred to v0.2).
- The existing `fang.WithNotifySignal(os.Interrupt, syscall.SIGTERM)` option must be preserved alongside the new option.
- `cmd/rak/root_test.go` gains `TestRootCmd_Version` asserting `--version` output contains the literal `v0.1.0`.
- Acceptance: `mage ci` green, `rak --version` prints `v0.1.0` (or fang-wrapped equivalent), test passes.

### Evidence

- `git diff HEAD~1 -- cmd/rak/main.go cmd/rak/root_test.go`: `cmd/rak/main.go` +7 −0 (4-line doc comment + `const version = "v0.1.0"` + 1-line `fang.WithVersion(version)` option appended after `fang.WithNotifySignal(...)` inside the variadic `fang.Execute` call). `cmd/rak/root_test.go` +29 −0 (new `TestRootCmd_Version` immediately before `TestRootCmd_TotalByLang_EndToEnd`).
- `Read` of current `cmd/rak/main.go:1-27` confirms `const version = "v0.1.0"` at line 16; `fang.Execute` at lines 19-26 carries both `fang.WithNotifySignal(os.Interrupt, syscall.SIGTERM)` (line 22) and `fang.WithVersion(version)` (line 23) as variadic Options.
- `go doc github.com/charmbracelet/fang.WithVersion` returns `func WithVersion(version string) Option` — confirms public API exists and signature matches the call site.
- `go doc github.com/charmbracelet/fang` lists both `WithNotifySignal(signals ...os.Signal) Option` and `WithVersion(version string) Option` as independent Option setters. `Execute(ctx, root, options ...Option) error` accepts both.
- `go doc github.com/spf13/cobra.Command.Version`: "If this value is non-empty and the command does not define a 'version' flag, a 'version' boolean flag will be added to the command and, if specified, will print content of the 'Version' variable." Cobra short-circuits `RunE` / `PersistentPreRunE` when `--version` is passed.
- `mage ci` from `main/`: lint clean, all 8 packages `ok`, `coverage: 87.8% (floor: 70.0%, scope: ./internal/...)`. Floor 17.8 pts above the gate; `cmd/rak` is excluded from coverage scope per decision 22 so adding `version` const + `TestRootCmd_Version` cannot regress the gate.
- `git grep "0\.1\.0"` across the repo: only stray hits are (a) docs referring to the v0.1.0 release, (b) the new `const version` + matching test literals, (c) v0.1.0-scope comments in `internal/lang/split*.go` and `internal/summary/sort.go` (decision pins, not version constants), (d) `github.com/muesli/{mango,mango-pflag,roff} v0.1.0` indirect-dep version coincidence in `go.mod` / `go.sum`. No competing production version constant.
- Test invocation pattern (`var out bytes.Buffer; cmd := newRootCmd(); cmd.SetOut(&out); cmd.SetErr(&out); cmd.SetArgs([]string{"--version"}); cmd.Execute()`) matches the existing `TestRootCmd_FlagJSON` (root_test.go:69-89), `TestRootCmd_ReadsStdin_RendersTOONDefault` (43-63), and `TestRootCmd_MutuallyExclusiveFlags` (95-116) harness convention.

### Trace or cases — attack surface results

| # | Attack | Result |
|---|---|---|
| A1 | Hardcoded `"v0.1.0"` drift — does any other production constant carry an older or different version literal? | **REFUTED.** `git grep "0\.1\.0"` returned no competing production constants. All matches accounted for (docs, v0.1.0-scope comments, indirect-dep version coincidences). The new `const version = "v0.1.0"` is the only production version source. |
| A2 | Does `fang.WithVersion(version)` break or replace `fang.WithNotifySignal(...)`? | **REFUTED.** Both are `Option` types per `go doc fang`. `fang.Execute(ctx, root, options ...Option) error` accepts variadic Options; both are passed independently. `Read` of main.go:19-26 confirms both options remain in the call. Each writes a disjoint field on fang's internal `*settings`. Order independence verified. |
| A3 | Does `PersistentPreRunE`'s `flags.sort` validation block `--version` from short-circuiting? | **REFUTED, twice.** (i) Cobra's `--version` handler short-circuits `PersistentPreRunE` and `RunE` (documented behavior; confirmed by `mage ci` passing the new test). (ii) Even if it didn't, `--sort` defaults to `"lines"` (root.go:147), which IS in `validSortKeys` — so validation would pass anyway. Belt-and-suspenders safe. |
| A4 | `strings.Contains(got, "v0.1.0")` too loose — would it pass on unrelated output that happens to mention `v0.1.0`? | **REFUTED.** The version string only lives in `main.go` as a `const`; no `RunE` / `PersistentPreRunE` code path in `root.go` references it. Cobra's built-in `--version` handler is the only writer of the version literal to `OutOrStdout()`. The Contains check is tightly coupled to the actual `--version` print path. Furthermore the test wires `cmd.Version = version` directly, so a future change to `version` propagates into the assertion automatically — no drift risk. |
| A5 | `mage ci` re-verify with 87.8% floor active. | **REFUTED.** Ran `mage ci` from `main/`: lint clean, all 8 packages `ok`, `coverage: 87.8% (floor: 70.0%, scope: ./internal/...)`. The coverage gate from Unit 9.3 stays green. `cmd/rak` is excluded from coverage scope per decision 22, so the new test + const cannot move the gate. |
| A6 | Variadic Option ordering — does appending `WithVersion` AFTER `WithNotifySignal` matter? | **REFUTED.** Both are setters writing disjoint fields on fang's internal `*settings` struct. Order is irrelevant. |
| A7 | `cmd.SetErr(&out)` shares the buffer with stdout — could this mask stderr-vs-stdout confusion? | **REFUTED.** Cobra's `--version` handler writes to `OutOrStdout()` (the `SetOut` target). Merging stderr into the same buffer is defensive and does not affect the assertion. If cobra ever switched to stderr for `--version`, the buffer would still capture it; the Contains assertion still holds. |
| A8 | `t.Parallel()` safety with the package-level `version` const? | **REFUTED.** `const version = "v0.1.0"` is immutable, safe to read concurrently. Each parallel test creates its own `newRootCmd()` (closure-local `flags`). No shared mutable state. No data race. |
| A9 | `fang.WithoutVersion()` somewhere in the codebase silently disabling the version flag? | **REFUTED.** `git grep` shows no calls to `fang.WithoutVersion` anywhere in the tree. Only `fang.WithNotifySignal` and the new `fang.WithVersion` are invoked. |
| A10 | Test can't reach the unexported `version` const? | **REFUTED.** `root_test.go:1` declares `package main`; `main.go:2` also `package main`. Same package — test reads `version` directly at `root_test.go:1123` (`cmd.Version = version`). Confirmed by `mage ci` (compilation + test execution green). |

### Self-attack against the verdict (QA falsification of the falsification)

- **Did `TestRootCmd_Version` actually execute, or was it cache-hit-skipped?** `mage test` reports `cached` for `cmd/rak`, but `mage ci` runs `Coverage` which invokes `go test -race -coverpkg=./internal/... -coverprofile=coverage.out ./...` — that re-runs the test suite (the coverage output line `coverage: 87.8%` is freshly computed). And on first commit, the test had no cache entry, so the cached subsequent runs prove a green initial run. Verdict stands.
- **End-to-end behavior of `rak --version` via the fang-wrapped binary not directly exercised in the test.** Direct binary execution is sandbox-blocked from this agent; `mage run -- --version` fails on a pre-existing magefile args-forwarding bug (`os.Args[1:]` includes "run"), out of Unit 9.2 scope. The unit test bypasses fang by setting `cmd.Version` directly — which is exactly the field `fang.WithVersion` writes to (per fang's purpose). Any divergence would be an upstream fang bug, not a Unit 9.2 issue. Acceptable; surfaceable as an unknown only.
- **`mage run -- --version` magefile bug — is that part of Unit 9.2?** No. The args-forwarding shape in `magefile.go:Run` predates Unit 9.2 and lives in `magefile.go` (build automation). Out of Unit 9.2's paths (`cmd/rak/main.go`, `cmd/rak/root_test.go`). Pre-existing, not introduced by this unit.
- **Was the `go install`-ed binary verified end-to-end?** Sandbox blocks direct binary execution. Per CLAUDE.md, `mage install` is dev-only — agents must not invoke it. The integration check `rak --version` against the installed binary is a dev-manual step at drop close. Unit 9.2's acceptance criteria are satisfied by the unit test + `mage ci` green.

### Conclusion

PASS. Zero counterexamples constructed across ten declared attack surfaces plus four follow-on self-attacks. Build / test / lint / coverage gates all green from `main/`. The `version` const is the sole production version source; no drift surface exists. `fang.WithVersion` and `fang.WithNotifySignal` compose correctly as independent Option setters on `fang.Execute`'s variadic parameter. Cobra's `--version` handler short-circuits `RunE` / `PersistentPreRunE` and emits the version literal to `OutOrStdout()`, which the test captures via `cmd.SetOut`. The `strings.Contains(got, "v0.1.0")` assertion is appropriately loose to survive fang TTY theming and appropriately tight because the version literal only originates from `cmd.Version`. Coverage gate (87.8% vs 70.0% floor) unaffected — Unit 9.2 only touches `cmd/rak`, which is outside the coverage scope.

### Unknowns

- **End-to-end `rak --version` via the fang-wrapped binary** is not directly exercised by an automated test; only the cobra-level behavior that `fang.WithVersion` writes into is unit-tested. Direct binary execution is sandbox-blocked from this agent; `mage run -- --version` is unusable due to a pre-existing magefile args-forwarding bug (`os.Args[1:]` includes the "run" target name, plus `--` is forwarded as a positional). Neither is a Unit 9.2 blocker; the dev verifies `rak --version` end-to-end at Drop 9 close as part of Unit 9.4 / 9.5. Surfaceable to orch as a non-blocking observation; consider a follow-up Drop-close smoke check or a v0.2 magefile fix to `Run` arg forwarding.

### Hylla Feedback

N/A — Unit 9.2 touched only `cmd/rak/main.go` (package `main`, entry point — Hylla indexes it but the change is a 1-line option append + const declaration, both verifiable from `git diff`) and `cmd/rak/root_test.go` (test file, not part of Hylla's exported surface in a meaningful way for this attack). All evidence sources for the attack — `git diff`, `Read`, `go doc fang`, `go doc cobra.Command`, `git grep`, `mage ci` — sufficed directly. No Hylla query attempted, no fallback miss.

## Unit 9.6 — Round 1

**Verdict:** PASS (no counterexamples found).
**Tier:** B — sole QA gate, no proof companion.
**Working dir:** `/Users/evanschultz/Documents/Code/hylla/rak/main`.
**Builder change under review:** Unit 9.6 — `files` column added to per-directory tabular output (`internal/render/toon.go`, `internal/render/human.go`, `internal/render/render_test.go`); `internal/render/json.go` field already wired (F44).

### Premises

- TOON `directories` tabular array gains a `files` column between `path` and `bytes`; canonical column order becomes `path|files|bytes|lines|words|chars`.
- JSON `directoryJSON.Files int64 \`json:"files,omitempty"\`` already existed; Unit 9.6 verifies wire end-to-end without a code change.
- Human per-directory KV blocks gain a `Files` row before `Bytes`; grand-total KV block (`countsKV("total", s.Total)`) MUST NOT emit `Files` because `counting.Counts` has no `Files` field.
- F44 (Files propagation through `filterUnknown` reconstruction) preserved; F33 (LangUnknown suppression) unaffected.
- Three new tests added: `TestRenderer_DirectoriesFilesColumn_TOON|JSON|Human` plus `dirFilesFixture` helper (3 dirs at Files=3, Files=5, Files=0).

### Evidence

- `Read internal/render/toon.go` lines 44-51: `toonDirectory` declares `Path → Files → Bytes → Lines → Words → Chars` in struct order; doc comment line 41-43 explicitly notes "Field declaration order is load-bearing: toon-go emits columns in struct order".
- `Read internal/render/json.go` lines 58-63: `directoryJSON` declares `Path, Counts, ByLang, Files` (matches `summary.Directory` order at `summary/summary.go:19-39` for bare struct conversion, F43); `Files int64 \`json:"files,omitempty"\``.
- `Read internal/render/json.go` lines 74-93: `filterUnknown` returns new `summary.Directory` with explicit `Files: d.Files` (F44 doc comment lines 71-73).
- `Read internal/render/human.go` lines 81-110: per-dir loop calls `dirKV("dir: "+d.Path, d.Files, d.Counts)` (line 84); grand-total uses `countsKV("total", s.Total)` (line 108).
- `Read internal/render/human.go` lines 143-153: `countsKV` body lists ONLY `Bytes/Lines/Words/Chars` — no `Files` field.
- `Read internal/render/human.go` lines 160-171: `dirKV` prepends a `Files` row before the four count rows.
- `Read cmd/rak/root.go`: `walkAndCount` (lines 314-433) accumulates `byDirFiles[dir]++` (line 403) and constructs `summary.Directory{... Files: byDirFiles[p]}` (line 429); `labelDirectories` (lines 486-504) propagates Files at lines 493 and 500. F44 wire intact.
- `Read internal/render/render_test.go` lines 666-836: `dirFilesFixture` (Files=3, 5, 0); three new tests covering TOON column ordering, JSON omitempty, human grand-total Files-absence.
- `mage test`: all 8 packages `ok`.
- `mage ci`: all green; coverage 87.8% (floor 70.0%); lint clean; format clean.

### Trace or cases — attack surface results

1. **Column-order regression** — REFUTED. `toonDirectory` struct field order is `Path → Files → Bytes → Lines → Words → Chars` (toon.go:44-51). toon-go marshals struct-field-order to TOON column order (confirmed via Drop 3-4 work and explicit doc comment "field declaration order is load-bearing"). Test `TestRenderer_DirectoriesFilesColumn_TOON` (render_test.go:697-736) asserts `idxPath < idxFiles < idxBytes` in header text — would catch any reorder.

2. **JSON omitempty wire end-to-end** — REFUTED. End-to-end trace: `walkAndCount` populates `byDirFiles[dir]++` (root.go:403) → constructs `summary.Directory{Files: byDirFiles[p]}` (root.go:429) → `labelDirectories` preserves Files (root.go:493, 500) → `filterUnknown` preserves Files (json.go:91) → `directoryJSON(filterUnknown(d))` bare-struct conversion (json.go:145) → `json.Encoder.Encode` honors `json:"files,omitempty"` tag. Test `TestRenderer_DirectoriesFilesColumn_JSON` (render_test.go:740-793) asserts Files=3 present, Files=5 present, Files=0 absent (via `*int64` pointer to detect omitempty). Wire confirmed.

3. *(skipped per spawn prompt)*

4. **Human renderer grand-total Files leak** — REFUTED. Grand-total path is `countsKV("total", s.Total)` (human.go:108). `countsKV` body (lines 143-153) builds `Pairs` from ONLY Bytes/Lines/Words/Chars — no Files row. Doc comment lines 140-142 explicitly states "It does NOT include a Files row because counting.Counts has no Files field". Test `TestRenderer_DirectoriesFilesColumn_Human` (render_test.go:825-836) uses `strings.LastIndex(got, "total")` to isolate the grand-total section and explicitly asserts absence of `Files` in that section. Would catch any leak.

5. **TOON omitempty mismatch** — REFUTED (acknowledged design trade-off). toon-go tabular arrays emit ALL declared columns per row (no per-row omitempty); a Files=0 dir renders as `gamma|0|...`. Test asserts `gamma|0|` substring at line 733 explicitly verifying this. The asymmetry (TOON shows zeros visibly; JSON omits zeros via omitempty) is documented in PLAN.md acceptance ("preserves existing zero-count snapshot behavior") and in the dirFilesFixture's gamma directory exercising both paths. Acceptable trade-off, not a bug.

6. **Snapshot drift in pre-existing tests** — REFUTED. Audited every pre-existing test that touches `RenderTree`:
   - `TestJSONRenderer_RenderTree_Snapshot` (line 274): byte-exact `want` string. Uses Files=0 dirs → `omitempty` suppresses `files` key → want-string remains valid. Confirmed.
   - `TestJSONRenderer_RenderTree_Empty` (line 303): byte-exact, no dirs → no `files` field. Unchanged.
   - `TestJSONRenderer_RenderTree_WithErrors` (line 327): byte-exact, Files=0 → omitted. Unchanged.
   - `TestTOONRenderer_RenderTree` (line 399): substring `".|"` — Files=0 row becomes `.|0|5|1|1|5`, still contains `".|"`. Loose assertion accommodates the new column intentionally.
   - `TestTOONRenderer_RenderTree_WithErrors` / `_NoErrors` / `_PerLang` / `_AllUnknown`: substring assertions on `errors`, `directories`, `go`, `rust`, `unknown`. None assert columns; new column does not break.
   - `TestHumanRenderer_RenderTree_Labels` (line 162): substring assertions on `dir:`, `total`, `Bytes/Lines/Words/Chars`, numeric values. Block-order assertion uses `strings.Index` for `dir:` and `total` — the new `Files 0` row appears within dir blocks but does not interfere with this test's assertions.
   - `TestHumanRenderer_RenderTree_NoErrors` / `_WithErrors` / `_EmptyDirs` / `_PerLang` / `_AllUnknown`: substring assertions, none assert absence of `Files` in dir blocks.
   No silent-pass — strict byte-exact tests use Files=0 dirs (omitempty path), loose substring tests are unaffected by the new column.

7. **F33 LangUnknown interaction** — REFUTED. `Files` lives on `summary.Directory`, not on per-lang rows. F33 lives in `sortedKnownLangs` (filters lang rows) and `filterUnknown`/`filterTotalByLangUnknown` (filters `by_lang` / `total_by_lang` map fields). The Files column on the per-directory row is independent — a dir with all LangUnknown content still has its Files count emitted in the directories tabular row, but its `by_lang` sub-object is suppressed. `filterUnknown` (json.go:74-93) explicitly preserves `Files: d.Files` even when `ByLang` is filtered to nil. Behavior matches PLAN.md acceptance.

8. **F44 Files-propagation regression** — REFUTED. `filterUnknown` (json.go:74-93) returns a NEW `summary.Directory` with explicit `Files: d.Files` at line 91; doc comment lines 71-73 explicitly call out F44. `labelDirectories` (root.go:486-504) preserves Files at lines 493 (root case) and 500 (sub case); doc comment lines 483-485 explicitly call out F44. F44 wire intact end-to-end.

9. **dirKV vs countsKV split caller audit** — REFUTED. Three call sites for the two helpers across the human renderer:
   - `Render` (single-stream, human.go:65): `countsKV("", counts)` — correct, single-stream Counts has no per-dir context.
   - `RenderTree` per-dir loop (human.go:84): `dirKV("dir: "+d.Path, d.Files, d.Counts)` — correct, per-dir uses dirKV.
   - `RenderTree` grand-total (human.go:108): `countsKV("total", s.Total)` — correct, s.Total is `counting.Counts` with no Files field.
   No caller misuses countsKV for per-dir context. The split is clean.

10. **Test fixture realism — `dirFilesFixture`** — REFUTED. `dirFilesFixture` (render_test.go:671-692) covers three boundary cases:
    - **Files > 0, distinct values (3 vs 5):** distinguishes the column from a constant or coincidental match (a fixture using only `Files=3` could pass even if the column was hardcoded to 3).
    - **Files == 0:** exercises JSON `omitempty` boundary (key absent) AND TOON tabular always-present boundary (`gamma|0|...`).
    - **Three distinct dir names (alpha/beta/gamma):** lexically ordered for deterministic assertions; covers multi-dir output ordering.
    All three v0.1.0-relevant boundary conditions covered. No additional counterexample fixture surfaces a missed case.

### Additional self-attacks

- **`directoryJSON.Files` int64 vs int unmarshal mismatch:** the test uses `Files *int64 \`json:"files"\`` to detect omitempty (nil pointer vs zero). Type matches `directoryJSON.Files int64`. Correct.
- **`laslig.Field` zero-value suppression:** verified `dirKV` emits `Files: strconv.FormatInt(files, 10)` — laslig prints zero values literally (`Files 0`). Confirmed by `TestRenderer_DirectoriesFilesColumn_Human` passing on gamma's Files=0 (the `idxFiles < idxBytes` assertion would fail if laslig dropped the row).
- **Concurrency:** rendering is single-goroutine — no race surface introduced.
- **Error swallowing:** no new error paths introduced — `dirKV` is pure construction; both renderer paths still wrap printer errors with `fmt.Errorf("...: %w", err)`.
- **Raw go commands:** none used; all verification via `mage test` / `mage ci`.
- **`mage install`:** not invoked.
- **YAGNI:** Files column has 1 user (per-dir directories tabular output) but is justified by acceptance criteria explicitly listing it. The dirKV/countsKV split has 3 call sites (one each), so the split is minimal not premature.

### Conclusion

PASS. All 10 attack surfaces from the spawn prompt + 7 supplementary self-attacks REFUTED with concrete code references. No unmitigated counterexample constructed.

### Unknowns

- **Coverage delta of new code:** `dirKV` shows 100.0% in `mage ci` output; `countsKV` shows 100.0%; `RenderTree` (toon.go) at 90.6%; `RenderTree` (json.go) at 90.0%; `RenderTree` (human.go) at 80.0%. No coverage regression. Not a blocker.
- **End-to-end `rak --sort files` + `--toon` smoke verification:** the new column would surface in real `rak` output; not directly exercised by an automated `cmd/rak` integration test. Tests live at the renderer-package level only. Acceptable for v0.1.0 because (a) the unit tests cover the boundary cases and (b) Drop 9.7 (release polish) will refresh README example output which is the de-facto end-to-end smoke check. Surfaceable to orch as a non-blocking observation.

### Hylla Feedback

- **Query 1:** `hylla_search_keyword` (implicit, recorded in builder worklog at lines 95-98) for `toonDirectory struct path bytes files`, `directoryJSON filterUnknown files omitempty`, `countsKV human renderer directory`, `summary Directory Files struct`.
- **Missed because:** Hylla's last ingest predates Drop 4 render work; the `toonDirectory`, `directoryJSON`, `humanRenderer.RenderTree`, and `summary.Directory` symbols are not in the current snapshot.
- **Worked via:** Direct `Read` of `internal/render/toon.go`, `internal/render/json.go`, `internal/render/human.go`, `internal/summary/summary.go`, `cmd/rak/root.go`.
- **Suggestion:** Re-ingest at Drop 9 close so Drop 4-9 render/summary symbols become searchable for v0.2 work. The `directoryJSON`, `toonDirectory`, `dirKV`, `countsKV`, and `filterUnknown` symbols would be valuable Hylla nodes for future render-layer work.

## Unit 9.8 — Round 1

**Verdict:** PASS (no counterexamples found).
**Tier:** B — sole QA gate, no proof companion.
**Working dir:** `/Users/evanschultz/Documents/Code/hylla/rak/main`.
**Commit under review:** `940bbb1 feat(cmd): add examples to help output via cobra Example field`.

### Premises

- Unit 9.8 adds a cobra `Example:` field on the root command containing 8 examples in a specified order, each prefixed with a `#` comment line.
- A new test `TestRootCmd_HelpContainsExamples` asserts that `--help` output contains the leading `# Default — emit TOON` comment and all 8 example command strings.
- Pre-existing `Long:` text MUST remain unchanged.
- `mage ci` must remain green; coverage must remain at-or-above the 70% floor.

### Evidence

- `git show 940bbb1 --stat`: only two production files touched — `cmd/rak/root.go` (+23) and `cmd/rak/root_test.go` (+45). Drop-dir mds also updated. No other Go files touched.
- `git diff 739d4f5 940bbb1 -- cmd/rak/root.go`: pure addition of `Example:` field between `Long:` and `Args:` lines. **No change to `Long:` text** (lines 64-70 of root.go untouched).
- `git diff 739d4f5 940bbb1 -- cmd/rak/root_test.go`: pure addition of `TestRootCmd_HelpContainsExamples` (45 lines), no edits to existing tests.
- `mage ci`: pass. lint clean (0 issues), all 8 packages `ok`, coverage `87.8% (floor 70.0%, scope ./internal/...)`. No regression.
- `mage test`: all packages `ok` (cached).
- Context7 `/spf13/cobra` confirms cobra's help template writes to `cmd.OutOrStdout()` — `cmd.SetOut(&out)` in the test captures help output correctly.
- `internal/render/json.go:107` confirms `treeJSON.TotalByLang` is tagged `json:"total_by_lang,omitempty"` — the example `rak --json . | jq '.total_by_lang'` references the actual JSON top-level key.
- `cmd/rak/root.go:94` confirms `Args: cobra.MaximumNArgs(1)` — 0 args triggers stdin path at `root.go:240` (`counting.Count(c.InOrStdin())`), so `cat README.md | rak` example is accurate (no `-` or explicit stdin token needed).

### Trace or cases — attack surface results

1. **Example rendering integrity (fang/cobra renders `Example:` verbatim).** REFUTED.
   - Per Context7 (`/spf13/cobra`), cobra's default help template emits the `Example:` field under an "Examples:" section verbatim. Fang wraps `cobra.Command.Execute` but leaves cobra's standard help template intact. The test asserts each of the 8 command literals are present in the captured output — `mage ci` green confirms.
   - Verbatim rendering also confirmed by builder worklog's own report (BUILDER_WORKLOG.md:108-112).

2. **Off-by-one in example count + order.** REFUTED.
   - Read of `cmd/rak/root.go:71-93` enumerates exactly 8 example blocks in spec order: `rak .`, `rak --human .`, `rak --json . | jq '.total_by_lang'`, `rak --sort files .`, `rak --sort path --sort-asc .`, `rak --lang go,rust .`, `rak --max-files 1000 .`, `cat README.md | rak`. No duplicates, no missing, no reordering. Matches `main/drops/DROP_9_RELEASE_DOCS/PLAN.md` § "Unit 9.8" acceptance criteria 1-8 exactly.
   - Test `wantCmds` slice (root_test.go:1167-1176) lists the same 8 strings in the same order.

3. **Comment-vs-command coupling (each `#` precedes its command).** REFUTED.
   - Visual inspection of root.go:71-93 confirms each `# <comment>` line is immediately followed by its `rak <args>` command on the next line, separated from the next pair by a blank line. Pairing is correct for all 8.

4. **`TestRootCmd_HelpContainsExamples` substring brittleness.** REFUTED.
   - Test asserts via `strings.Contains` on plain command literals (e.g. `"rak ."`, `"rak --human ."`, `"cat README.md | rak"`). These literals have no special characters that fang/cobra would line-wrap or re-indent inside (no long phrases that might break across lines at terminal widths; max length is `"rak --sort path --sort-asc ."` at 28 chars, well under any reasonable wrap point).
   - Help output is captured via `bytes.Buffer` (non-TTY); fang's TTY styling/ANSI codes are not applied because `cmd.SetOut` to a buffer bypasses TTY detection.
   - Comment assertion `"# Default — emit TOON"` uses the em-dash `—` (U+2014). The em-dash appears verbatim in both the source string (`root.go:71`) and the test (root_test.go:1162) — `strings.Contains` operates on raw bytes, so as long as both are UTF-8 the comparison is byte-equivalent. No locale dependency at the test boundary.

5. **JSON pipe example correctness — `.total_by_lang` is real JSON key.** REFUTED.
   - `internal/render/json.go:105-110` defines `treeJSON` with field `TotalByLang map[lang.Language]lang.LangCounts \`json:"total_by_lang,omitempty"\``.
   - The example `rak --json . | jq '.total_by_lang'` therefore selects an actual top-level key in `rak --json` output. (When the map is empty after F33 filtering, the key is suppressed by `omitempty` and jq returns `null` — that's a degenerate but non-broken case.)

6. **`Long:` regression.** REFUTED.
   - `git diff 739d4f5 940bbb1 -- cmd/rak/root.go` shows only an `Example:` insertion between `Long:` and `Args:`. The `Long:` block at lines 64-70 is identical pre/post commit (zero deleted lines on the `Long:` content).

7. **`Example:` indentation + fang re-indent interaction.** REFUTED.
   - The raw string at root.go:71-93 uses a 2-space leading indent on every non-blank line. Cobra's default help template emits `Example:` content verbatim without re-indenting (per Context7 cobra docs on `SetHelpTemplate` — the default template uses `{{.Example}}` directly without manipulation).
   - The test does not assert leading whitespace; it asserts substring `"rak ."`, `"# Default — emit TOON"`, etc. Even if fang stripped one space off each line, the literal substrings would still match.
   - `mage ci` green confirms the test passes, so any observed fang/cobra indent transformation does not break the substring assertions.

8. **Stdin example accuracy (`cat README.md | rak`).** REFUTED.
   - `cmd/rak/root.go:94` declares `Args: cobra.MaximumNArgs(1)` (0 or 1 args allowed).
   - `cmd/rak/root.go:228-247` `runRoot`: `len(args) == 1` triggers the directory walk; the `else` branch (0 args) falls through to `counting.Count(c.InOrStdin())` at line 240. No `-` token required; rak reads stdin by default when no path is given.
   - Example `cat README.md | rak` is therefore accurate against the current root.go behavior.

9. **Localization / em-dash encoding.** REFUTED.
   - Em-dash `—` (U+2014, UTF-8 `\xE2\x80\x94`) appears in both `root.go:71` and the matching test assertion at `root_test.go:1162`. Go source files are UTF-8 by spec; raw-string literals preserve bytes verbatim; `strings.Contains` operates on raw bytes. The comparison is byte-identical regardless of terminal locale.
   - Terminal/locale rendering is a user-display concern at runtime, not a test-correctness concern. The test asserts the em-dash byte sequence is present in `bytes.Buffer` output; that holds.

10. **Trailing-newline / final-line rendering.** REFUTED.
    - The raw string ends with `cat README.md | rak` and no trailing newline (the closing backtick follows immediately).
    - Cobra's help template appends its own framing (blank line + next section header) after `{{.Example}}`, so the final line is followed by a separator that's part of the template's structure, not the field value.
    - Even if cobra emitted no trailing newline, the test asserts `strings.Contains(got, "cat README.md | rak")` which succeeds whether or not a newline follows.

11. **Help-output channel mismatch (stdout vs stderr).** REFUTED.
    - Per Context7 cobra docs, the default help template writes to `cmd.OutOrStdout()` (not stderr). The test sets BOTH `cmd.SetOut(&out)` and `cmd.SetErr(&out)` to the same buffer — even if cobra/fang routed help to stderr, the buffer captures both. No channel-mismatch escape route.

12. **Coverage regression below 70% floor.** REFUTED.
    - `mage ci` post-9.8: `coverage: 87.8% (floor: 70.0%, scope: ./internal/...)`. Unit 9.8 added zero `./internal/...` code (only `cmd/rak/`), which is explicitly excluded from the coverage scope per decision 22 and the `-coverpkg=./internal/...` flag in `magefile.go:119`. Therefore Unit 9.8 cannot regress the coverage gate.

13. **(Self-attack) Test could pass against an empty `Example:` if fang silently injected its own example content.** REFUTED.
    - Fang (`/charmbracelet/fang`) is a styling/theming wrapper over cobra's `Execute`; it does NOT synthesize example content. The 8 example literals can only originate from the cobra command's own `Example:` field. The builder worklog's RED step (BUILDER_WORKLOG.md:101) confirms the test fails before the field is added.

14. **(Self-attack) Test passes vacuously because `--help` errors before output is captured.** REFUTED.
    - `cmd.SetArgs([]string{"--help"})` triggers cobra's built-in `-h, --help` handler. Cobra writes the help text and returns nil. The test asserts `if err := cmd.Execute(); err != nil { t.Fatalf(...) }` so a non-nil error would fail the test, not vacuously pass.

15. **(Self-attack) Builder skipped the `# Default — emit TOON` em-dash and used a hyphen.** REFUTED.
    - Re-read of `cmd/rak/root.go:71` confirms `# Default — emit TOON for LLM-first consumption` with the em-dash. Test assertion at `root_test.go:1162` uses the same em-dash. Byte-equivalent.

16. **(Self-attack) Out-of-paths file edits.** REFUTED.
    - `git show 940bbb1 --stat`: only `cmd/rak/root.go` and `cmd/rak/root_test.go` (+ drop-dir mds, which are workflow files). PLAN.md's declared paths for Unit 9.8 are exactly these two Go files. No scope creep.

17. **(Self-attack) Concurrency / goroutine / mutex regressions.** REFUTED.
    - Unit 9.8 adds a static string field on a `cobra.Command` literal and a test function. No goroutines spawned, no shared state, no synchronization primitives. The change has no concurrency surface to attack.

### Conclusion

PASS. All 12 attack surfaces from the spawn prompt + 5 supplementary self-attacks REFUTED with concrete code/diff/Context7 evidence. No counterexample constructed. `mage ci` green at 87.8% coverage.

### Unknowns

- **TTY-mode visual rendering of the `Example:` block under fang.** Fang applies ANSI styling to cobra output in TTY mode (per the Context7 `/charmbracelet/fang` summary: "fancy output ... theming"). The test runs in a non-TTY `bytes.Buffer`, so the unstyled command literals are asserted. The styled TTY rendering is a UX surface, not a correctness surface; not under test. Acceptable — the test exercises the load-bearing assertion (content present in help output) and `mage ci` runs the same non-TTY path. Recommend the dev visually inspect `rak --help` from a real terminal once during release-polish (Unit 9.7) and confirm the styling looks right.
- **Long-line wrap behavior at narrow terminal widths.** Cobra's default template does NOT wrap `Example:` field content (it's emitted verbatim). Fang may wrap at terminal width in TTY mode. The longest example line is 28 chars; at default 80-col terminals there's no risk. At 24-col-and-below terminals fang might wrap; not a v0.1.0 blocker. Not under test.

### Hylla Feedback

N/A — Unit 9.8 touched only `cmd/rak/root.go` (cobra command field addition) and `cmd/rak/root_test.go` (one test function). The change required no Go symbol navigation or cross-package reference lookup; direct `Read` + `git diff` + `git show --stat` covered all evidence needs. Context7 was queried for cobra's help-template semantics (one query) since that's an external-library contract Hylla cannot answer. No Hylla queries attempted, no fallback misses to report.
