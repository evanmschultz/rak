# DROP_2 — Plan QA Falsification Review (Round 1)

**Reviewer:** go-qa-falsification-agent
**Round:** 1
**Date:** 2026-04-19
**Verdict:** pass

## Summary

15 attacks enumerated across all surfaces the orchestrator brief demanded — 5→4 reshape, laslig API citations, snapshot determinism, JSON field-order coupling, CRLF acceptance vagueness, Counts-struct exposure, NewHumanRenderer arglessness, bootstrap carve-out scope, stdin-source choice, path-arg rejection, root.go LOC budget, integration fixture content, 2.1/2.2 parallelism, Unknown routing completeness, laslig version-pin fragility. Of those: 6 CONFIRMED (F3, F4, F5, F8, F11, F13) — all non-blocking, all routed to Phase 3 dev discussion or minor plan revision. 9 REFUTED. No CONFIRMED attack constitutes a blocking defect; the plan's architecture, dep-DAG, acceptance structure, and hand-off pins are sound. Pattern: the CONFIRMED set clusters on "acceptance is descriptive but not concretely pinned" (F3, F4, F5, F11, F13) and "scope-rule drift since CLAUDE.md was written" (F8). These are plan-text sharpenings, not structural re-plans.

## Attacks

### F1 — 5→4 unit-count reshape soundness

- **Severity:** info
- **Unit(s):** global
- **Attack:** main/PLAN.md lines 107–113 prescribed 5 units (2.1 counting / 2.2 render / 2.3 wire-up / 2.4 TTY-auto / 2.5 tests). The drop PLAN collapses TTY-auto into 2.2 and dissolves "2.5 tests" across the other units (TDD-first). Counterexample attempt: does TTY-auto deserve its own unit so a fake-TTY test-infrastructure (e.g. `github.com/creack/pty`) gets its own acceptance?
- **Result:** REFUTED. `laslig.ResolveMode` at `/Users/evanschultz/go/pkg/mod/github.com/evanmschultz/laslig@v0.2.4/policy.go:210-211` handles TTY detection by type-asserting the writer to `term.File` and calling `term.IsTerminal(fd)`. The renderer gets auto-detection for free by passing `laslig.Policy{Format: FormatAuto, Style: StyleAuto}` to `laslig.New` (printer.go:39). No fake-TTY test infrastructure is needed; snapshot tests can pass a `*bytes.Buffer` (which is NOT a `term.File`, so ResolveMode returns `FormatPlain, Styled: false`) and assert deterministic plain output. "TTY-auto" as a standalone unit would be ~3 lines of config choice, not a unit. The "2.5 tests" collapse is forced by CLAUDE.md § "Tests" TDD-first discipline — tests ship with the code that produces them. Plan's reshape is sound.
- **Evidence:** `/Users/evanschultz/go/pkg/mod/github.com/evanmschultz/laslig@v0.2.4/policy.go:210-260`, `/Users/evanschultz/go/pkg/mod/github.com/evanmschultz/laslig@v0.2.4/printer.go:38-42`.
- **Recommendation:** accept.

### F2 — Laslig API citation accuracy

- **Severity:** minor
- **Unit(s):** 2.2
- **Attack:** Plan cites `New` at `printer.go:39`, `NewWithMode` at `printer.go:48`, `ResolveMode` at `policy.go:211`, plus `Policy{Format: FormatAuto, Style: StyleAuto}` field names. If any citation is off — wrong file, wrong line, wrong field name — builder follows a bad breadcrumb.
- **Result:** REFUTED. Verified against module cache copy: `func New` signature is on `printer.go:39`; `func NewWithMode` is on `printer.go:48`; `func ResolveMode` is on `policy.go:210-211` (plan cites :211 — the `func` line is 211, the leading doc comment is 210; close enough, not a drift). `Policy.Format` and `Policy.Style` are real field names with types `Format` and `StylePolicy` respectively (`policy.go:40-57`). `FormatAuto`, `FormatPlain`, `StyleAuto` constants all exist (`policy.go:17-35`). `Mode` struct has `Format`, `Styled`, `Width` fields (`policy.go:204-208`). Every cite lands.
- **Evidence:** `/Users/evanschultz/go/pkg/mod/github.com/evanmschultz/laslig@v0.2.4/policy.go:17-35, 40-57, 204-208`, `printer.go:38-49`.
- **Recommendation:** accept.

### F3 — Snapshot test determinism under `$COLUMNS` env

- **Severity:** minor
- **Unit(s):** 2.2
- **Attack:** `laslig.ResolveMode` reads `os.Getenv("COLUMNS")` at `policy.go:224` when the writer is not a terminal. If `$COLUMNS` is set in the dev or CI shell, the returned `Mode.Width` is the parsed value (else 0). The plan's `NewHumanRenderer` uses `FormatAuto`, so on a `*bytes.Buffer` writer the mode resolves to `FormatPlain, Styled: false, Width: <env>`. For laslig's `KV` block the width primarily affects table/panel wrapping — a 4-pair KV of small int64 values is unlikely to wrap at typical widths (80+) but MAY at very narrow widths. More importantly, the mere presence of `os.Getenv("COLUMNS")` in the dep-chain means the test's environment is not fully isolated from the laslig call, and a motivated CI config could set `COLUMNS=20` and break snapshots.
- **Result:** CONFIRMED (non-blocking). The plan's 2.2 acceptance punts snapshot-determinism mechanism to the builder with "test is deterministic across TTY / non-TTY CI environments (race detector already forces no-TTY in CI)" — but race detector has nothing to do with `$COLUMNS`. The punt is incomplete. Correct mitigation: builder uses `laslig.NewWithMode(buf, laslig.Mode{Format: laslig.FormatPlain, Styled: false, Width: 80})` inside the TEST (not inside `NewHumanRenderer`'s production body) to bypass `ResolveMode` entirely, OR the test calls `t.Setenv("COLUMNS", "80")` to pin the env read. The plan should name one explicitly.
- **Evidence:** `/Users/evanschultz/go/pkg/mod/github.com/evanmschultz/laslig@v0.2.4/policy.go:223-229`. Drop PLAN.md line 58 ("builder chooses mechanism — acceptance is that the test is deterministic…").
- **Recommendation:** revise — add a concrete acceptance bullet naming the mitigation (prefer `NewWithMode` in the test helper OR `t.Setenv("COLUMNS","80")`), and explicitly state the invariant: "test output is independent of `$COLUMNS`, `$TERM`, `$NO_COLOR`, `$CI`."

### F4 — JSON field-order coupling across units 2.1 and 2.2

- **Severity:** major
- **Unit(s):** 2.1, 2.2
- **Attack:** Unit 2.2 acceptance (drop PLAN.md line 59) asserts exact JSON output `{"Bytes":12,"Lines":1,"Words":2,"Chars":12}\n`. Go's `encoding/json.Encoder.Encode` serializes struct fields in **struct declaration order**, not alphabetical. Unit 2.1 acceptance (drop PLAN.md line 35) prescribes `Counts` field order as `Bytes int64`, `Lines int64`, `Words int64`, `Chars int64` — matching the JSON assertion. BUT this is a hidden cross-unit contract: if the 2.1 builder accidentally reorders (e.g. alphabetical) for aesthetic reasons, 2.2's snapshot breaks. The plan documents the field order in both places but does not FLAG the cross-unit coupling. Additionally, the default JSON marshal uses the Go field NAME (capitalized) as the JSON key. The acceptance string uses `"Bytes"` etc. (capitalized) — consistent with no `json:` tags on `Counts`. If the 2.1 builder adds lowercase `json:"bytes"` tags for any reason (for instance, thinking "JSON convention is lowercase"), 2.2's snapshot breaks.
- **Result:** CONFIRMED (non-blocking). The plan does not explicitly forbid json tags on `Counts` nor explicitly call out the field-declaration-order contract as a cross-unit invariant. Current `root.go` `Counts` struct at lines 13-18 has no json tags and declares fields in the `{Bytes, Lines, Words, Chars}` order — so naturally matches — but the plan's 2.1 acceptance says "moved verbatim" without pinning the constraint.
- **Evidence:** `main/cmd/rak/root.go:12-18` (current `Counts` struct, no json tags, field order matches); Go `encoding/json` semantic: field names default to the Go identifier, fields serialized in declaration order. Drop PLAN.md lines 35, 59.
- **Recommendation:** revise — add explicit acceptance to 2.1: "`Counts` struct declares fields in order `Bytes, Lines, Words, Chars` and carries no `json:` struct tags (2.2's JSON snapshot depends on both)." And add to 2.2: "this snapshot depends on 2.1's field order + no json tags — do not change 2.1 without also updating this snapshot."

### F5 — CRLF acceptance is descriptive, not pinned

- **Severity:** minor
- **Unit(s):** 2.1
- **Attack:** Drop PLAN.md line 38 test case for CRLF: `"a\r\nb\r\n"` → "verify the documented behavior — CR is whitespace per `unicode.IsSpace` so words split on it; Lines increments only on `\n`". Every other test case in that line names concrete expected tuples (e.g. `"hello\n"` → `{6,1,1,6}`). The CRLF case gives behavioral prose only. This is an acceptance gap: two builders could both satisfy "CR is whitespace, lines only on `\n`" with different outputs (one counting `\r` as its own character, another not).
- **Result:** CONFIRMED (non-blocking). Tracing the current `count` implementation in `root.go:42-78` against `"a\r\nb\r\n"`: 6 bytes, 6 chars (`\r` counts as a char), 2 lines (`\n` twice), 2 words (`a` + `b` split by whitespace including `\r` and `\n`). Expected tuple: `{Bytes:6, Lines:2, Words:2, Chars:6}`. The plan should pin this so 2.1's test table is mechanically verifiable against the Drop 1 pinned primitive.
- **Evidence:** `main/cmd/rak/root.go:42-78`; traced by hand. Drop PLAN.md line 38.
- **Recommendation:** revise — replace the prose clause with `"a\r\nb\r\n"` → `{Bytes:6, Lines:2, Words:2, Chars:6}`.

### F6 — `Counts` struct exposed too eagerly

- **Severity:** info
- **Unit(s):** 2.1, 2.2
- **Attack:** Unit 2.2 `Renderer` interface is `Render(w io.Writer, counts counting.Counts) error` — passes `counting.Counts` by value. If Drops 4–6 add blank/comment/code split or per-language breakdown, `Counts` grows fields and every renderer re-compiles. YAGNI pressure says this is fine today (2 renderers, no growth pressure yet), but a defensive design would have `Renderer.Render` accept a smaller view (e.g. a `render.Summary` type in `internal/render`) that the `cmd/rak` layer constructs from `counting.Counts` + whatever else. That decouples the display contract from the math contract.
- **Result:** REFUTED (design-choice, not blocker). main/PLAN.md decision 27(d) explicitly locks in "explicit `NewHumanRenderer`/`NewJSONRenderer` (no Format enum factory)" AND decision 25 commits to `Counts` + `Count()` as the v0.1.0 domain primitive (a single struct carried through). Later drops that add blank/comment/code split land their own render boundary (Drop 4 / Drop 6 acceptance; see main/PLAN.md § "Expected Decomposition"). Adding a `render.Summary` abstraction today to hedge a 4-drop-out concern is premature abstraction — YAGNI pressure cuts AGAINST the defensive design. Plan is right to pass `counting.Counts` directly.
- **Evidence:** main/PLAN.md decisions 25, 27(d); main/PLAN.md § "Expected Decomposition" Drops 4–6.
- **Recommendation:** accept.

### F7 — `NewHumanRenderer() Renderer` argless constructor

- **Severity:** minor
- **Unit(s):** 2.2
- **Attack:** `NewHumanRenderer()` takes no args. No way for the caller (2.3's wire-up or future Drop 3+ walker output) to override the laslig policy — e.g. force `FormatPlain` for a `--no-color` flag, or configure a `SpinnerStyle` for Drop 8. When those flags land, the renderer constructor will need to grow a parameter list or an options struct, breaking the `NewHumanRenderer()` signature.
- **Result:** REFUTED. Breaking a renderer signature in a future drop is cheap — `internal/render` is not a public API (per CLAUDE.md § "Project Structure" → "Visibility" rule 12: "everything under `internal/` by default. rak has no public API beyond the binary"). Changing `NewHumanRenderer()` to `NewHumanRenderer(opts ...Option)` in Drop 8 is a one-line compile error propagation, not a breaking change for external callers. Today's simpler signature wins on YAGNI. Additionally, the orchestrator brief flagged "future callers (Drop 3+ walker output) that want interactive mode with a spinner" — but spinner lives in main/PLAN.md decision 21 as "deferred entirely to Drop 8", not Drop 3. No pressure to pre-configure for that now.
- **Evidence:** main/CLAUDE.md § "Project Structure" → "Visibility" rule 12; main/PLAN.md decision 21.
- **Recommendation:** accept.

### F8 — Bootstrap carve-out scope creep

- **Severity:** major
- **Unit(s):** 2.2
- **Attack:** CLAUDE.md § "Go Development Rules" → "Dependencies" → "Bootstrap carve-out" reads: *"When a unit introduces a mage-managed dep for the very first time and no mage target yet exists to wrap `go get`, the builder MAY run `go get <module>` + `go mod tidy` directly from `main/` with default environment."* It continues: *"Today this applies only to Drop 1.4 (first-ever `github.com/magefile/mage` add). **From Drop 2 onward the magefile exists, so every dep add routes through a mage target** and this carve-out does not apply."* The drop PLAN.md line 50 / line 108 reinvokes the bootstrap carve-out for laslig in Drop 2.2. **This directly contradicts CLAUDE.md.**
- **Result:** CONFIRMED (non-blocking for Drop 2 execution, major for process correctness). The magefile exists (landed in Drop 1.5), but it has no `deps` or `addDep` target to wrap `go get`. So EITHER (a) the plan's carve-out invocation is wrong and the builder must add a new mage target before dep-add can happen, OR (b) CLAUDE.md's carve-out clause needs a clarifying amendment ("carve-out applies whenever no mage target yet exists for dep-management, regardless of drop number"). This is a real drift between plan intent and project rules that the dev needs to resolve. It is not a blocker for writing the plan — the ambiguity is in CLAUDE.md, not the plan — but it must be surfaced for Phase 3 dev discussion.
- **Evidence:** `main/CLAUDE.md` § "Dependencies" (the "Bootstrap carve-out" subsection); Drop 2 PLAN.md lines 50, 108. Verified: `main/magefile.go` has no dep-add target by enumeration — per Drop 1 PLAN line 108 the 9 canonical targets are `build/test/format/lint/ci/install/run/coverage/planCheck`, none of which wraps `go get`.
- **Recommendation:** revise — either (a) plan adds a sub-acceptance to 2.2 creating a `mage addDep <module>` target FIRST, THEN using it for laslig; or (b) plan explicitly cites a dev-approved CLAUDE.md amendment extending the carve-out. The planner brief for Phase 3 discussion already flags "laslig re-add confirmation" as an Unknown (drop PLAN.md line 124 Unknowns #2 equivalent), which partly covers this — but the scope-rule conflict should be made explicit.

### F9 — `cmd.InOrStdin()` vs `os.Stdin` punt

- **Severity:** minor
- **Unit(s):** 2.3
- **Attack:** Drop PLAN.md line 72: "or uses `cmd.InOrStdin()` for cobra-idiomatic input indirection — builder chooses based on how `root_test.go` drives input." These two options have subtly different semantics in tests: `cmd.InOrStdin()` returns whatever was set via `cmd.SetIn(...)` falling back to `os.Stdin`, while direct `os.Stdin` ignores `SetIn`. The 2.3 test cases at lines 82-85 use `cmd.SetIn(strings.NewReader(...))` — which ONLY works if the `RunE` uses `cmd.InOrStdin()`. If the builder picks `os.Stdin` the test fails because `SetIn` has no effect. The "builder chooses" framing masks a fixed correct answer.
- **Result:** REFUTED (on careful read). The acceptance's test cases (lines 82-85) all invoke `cmd.SetIn(...)` + `cmd.Execute()`. For those tests to pass, `cmd.InOrStdin()` is forced — the "or" in the paths line is misleading but the acceptance block's test cases constrain the builder to the correct answer. The builder will either pick `cmd.InOrStdin()` directly from the test-driven inference, or pick `os.Stdin`, see the test fail, and switch. Not a blocker; the acceptance self-corrects.
- **Evidence:** Drop PLAN.md lines 72, 82-85. Cobra semantics: `cobra.Command.InOrStdin()` returns `c.In` if set else `os.Stdin` — standard library behavior documented in cobra.
- **Recommendation:** accept — but a minor polish opportunity: line 72 could drop the "or `os.Stdin`" option since the tests constrain to `cmd.InOrStdin()` anyway.

### F10 — `RunE` error on `len(args)==1` path-arg

- **Severity:** minor
- **Unit(s):** 2.3
- **Attack:** Drop PLAN.md line 75: when `len(args)==1` the command errors with "positional path argument not supported yet — walker lands in Drop 3…". From a dev UX standpoint, `rak .` on someone's first try gets a wall of error text when "just read stdin and ignore args" or "give a friendlier message" would be less jarring. main/PLAN.md decision 17 commits to `rak [path]` as the v0.1.0 shape, and Drop 3 lands walker — so the current drop is in a transitional state. Is erroring the right default?
- **Result:** REFUTED (plan correctly flags this as a dev-decision Unknown). Drop PLAN.md line 75 explicitly tags this "subject to Phase 3 dev approval — the alternative is 'always read stdin and ignore args'". The planner flagged it as an Unknown (line 122). That's exactly the right call — surface the ambiguity, don't hide it. Not a plan defect.
- **Evidence:** Drop PLAN.md lines 75, 122.
- **Recommendation:** accept.

### F11 — `root.go` ~150 LOC budget realism

- **Severity:** minor
- **Unit(s):** 2.3
- **Attack:** Drop PLAN.md line 80 prescribes `cmd/rak/root.go` stays ≤ ~150 LOC. Current `root.go` is 79 LOC (includes `count` + `Counts` + stub RunE). Unit 2.3 REMOVES `count` + `Counts` (~45 LOC worth) AND adds: `--format` flag var + wiring, `ctx := c.Context()`, stdin read, error on path-arg, `counting.Count` call, renderer selection switch (3 branches), `renderer.Render` call with error wrap, format-validation logic, plus imports. Rough count: 45 LOC removed, 40-60 LOC added. End state likely 75-100 LOC — well under 150. **But** the plan's test file `cmd/rak/root_test.go` (Drop PLAN.md lines 81-85) adds 4 sub-tests. CLAUDE.md § "Project Structure" file breakdown budgets `cmd/rak/root_test.go` at ~150 LOC — the 4 test cases are plausibly within that, but the plan ACCEPTANCE makes no LOC assertion on the test file.
- **Result:** CONFIRMED (non-blocking, minor). The plan's LOC budget on `root.go` is realistic (generous margin). But `root_test.go` LOC is not budget-checked in the plan — and CLAUDE.md § "Project Structure" sets a ~150 LOC target. Not a defect, just an unconstrained axis.
- **Evidence:** `main/cmd/rak/root.go` (current 79 LOC); Drop PLAN.md line 80; main/CLAUDE.md § "Project Structure" file table row `cmd/rak/root_test.go` → ~150 LOC.
- **Recommendation:** revise (optional) — add one-line acceptance to 2.3: "`cmd/rak/root_test.go` stays ≤ ~150 LOC per CLAUDE.md § 'Project Structure'."

### F12 — Unit 2.4 integration fixture content TBD

- **Severity:** minor
- **Unit(s):** 2.4
- **Attack:** Drop PLAN.md line 93: "`main/cmd/rak/testdata/hello.txt` (new fixture — contents TBD by builder but stable and documented in test)". Risk: fixture content depends on `counting.Counts` semantics (word boundary via `unicode.IsSpace`). If Unit 2.1 semantically changes word-split behavior mid-round (e.g. treats zero-width-joiner differently), 2.4's expected output silently shifts. Also: "TBD by builder" means no independent QA check on whether the fixture actually exercises the interesting cases (UTF-8, CRLF, multi-line, etc.).
- **Result:** REFUTED on the primary-attack axis (2.1 semantics are locked by 2.1's own acceptance at lines 34-41 and by `main/cmd/rak/root.go:42-78` which 2.1 copies verbatim — any 2.1-builder semantic drift would fail 2.1 QA first). But CONFIRMED on the secondary axis: the plan doesn't name which features the fixture should exercise. Builder could use `"hello\n"` (trivial, no UTF-8, no CRLF) and still satisfy acceptance.
- **Evidence:** Drop PLAN.md lines 93-101; 2.1 acceptance lines 34-41.
- **Recommendation:** revise — add one-line to 2.4 acceptance: "fixture exercises at least: multi-line, multi-word, and one multi-byte UTF-8 rune (to catch Bytes-vs-Chars regressions)."

### F13 — 2.1 and 2.2 true parallelism

- **Severity:** major
- **Unit(s):** 2.1, 2.2
- **Attack:** Drop PLAN.md dependency DAG (lines 20-26) says `2.1 ──▶ 2.3` and `2.2 ──▶ 2.3` with 2.1 and 2.2 in parallel. But Unit 2.2's `Renderer` interface signature (drop PLAN.md line 53) is `Render(w io.Writer, counts counting.Counts) error` — it imports `github.com/evanmschultz/rak/internal/counting`. If the builder runs 2.2 BEFORE 2.1 finishes, the `render.go` file fails to compile because `counting.Counts` doesn't exist yet. So 2.2 is LOGICALLY blocked by 2.1 even though the DAG says parallel. The plan itself notes this at line 53: "verifying 2.1 → 2.2 is the natural dep-DAG order (leaf → interior…), even though they're siblings in the drop DAG." The drop DAG and the import DAG contradict.
- **Result:** CONFIRMED (non-blocking). The plan is internally aware of this (line 53 acknowledges the tension) but the dependency DAG diagram still claims parallelism. In practice Phase 4 builds one unit at a time per WORKFLOW.md § "Phase 4" step 1 ("Orch picks the NEXT ELIGIBLE unit") — so 2.1 and 2.2 won't literally run in parallel anyway. The "parallel" in the DAG is ordering independence, not temporal parallelism. The defect is cosmetic: the plan should either (a) fix the DAG to show `2.1 → 2.2` since 2.2 imports 2.1's output package, or (b) accept "DAG-parallel + implementation-ordered" as a documented distinction.
- **Evidence:** Drop PLAN.md lines 20-26 (DAG diagram), line 53 (the self-acknowledged tension), main/drops/WORKFLOW.md § "Phase 4" step 1.
- **Recommendation:** revise — either (a) redraw DAG as `2.1 → 2.2 → 2.3` (stricter import-DAG ordering), or (b) add a plan Notes entry clarifying "DAG siblings denote ordering independence in the abstract — in practice Phase 4 serializes them, so 2.1 runs first to avoid a compile gap in 2.2."

### F14 — Unknowns routing completeness

- **Severity:** minor
- **Unit(s):** global
- **Attack:** Drop PLAN.md explicitly flags two Unknowns for Phase 3 dev discussion: (1) stdin-default-vs-error on path arg (line 75/122), (2) laslig re-add confirmation (line 108). The orchestrator brief asked whether OTHER implicit Unknowns exist that should have been flagged. Candidates: bootstrap carve-out scope (see F8), snapshot-test determinism mechanism (see F3), LOC budget on test files (see F11), integration fixture content (see F12), struct-field-order JSON coupling (see F4).
- **Result:** CONFIRMED (subset already covered by F3/F4/F8/F11/F12 above). The two explicitly-flagged Unknowns are the right ones for dev sign-off; the others surfaced here are plan-revision items, not dev-decision items. The distinction is appropriate: Unknowns (dev approval needed) vs. plan-sharpening (revise at planner re-spawn). Not a plan defect in terms of what's routed to dev, but the plan could add a "Plan revisions needed" section alongside "Unknowns" to separate the two.
- **Evidence:** Drop PLAN.md lines 75, 108, 122.
- **Recommendation:** accept — but optional improvement: add a "Plan Revisions Pending" section capturing F3/F4/F11/F12/F13 recommendations as line-items.

### F15 — Laslig version pin fragility for snapshots

- **Severity:** minor
- **Unit(s):** 2.2
- **Attack:** Laslig is pinned at `v0.2.4` via the plan's dep-add acceptance (line 50). Snapshot tests in 2.2 assert exact human-renderer output (specific KV-block formatting). If laslig releases `v0.2.5` with a harmless tweak (e.g. extra leading blank line, changed indent width), running `go get github.com/evanmschultz/laslig@latest` later breaks snapshot tests even though no rak code changed.
- **Result:** REFUTED. The plan PINS the version explicitly (line 110: "`v0.2.4` is the latest laslig present in the local module cache"). Pins don't auto-bump under `go mod tidy` (tidy respects the pinned version unless explicitly told otherwise via `go get -u`). A future laslig upgrade is a deliberate dev act, not an accidental drift. REFINEMENTS entry 4 ("name tool versions explicitly") further ratifies this approach. The plan also flags in Drop 1.5's magefile that dev-only deps (`gofumpt`, `golangci-lint`) lack version pins — that's a different class (dev tooling vs. production dep) and is already tracked in main/PLAN.md § "Follow-Ups".
- **Evidence:** Drop PLAN.md lines 50, 110; Go modules behavior.
- **Recommendation:** accept.

## Verdict Rationale

Pass. Zero CONFIRMED blocking attacks. Six CONFIRMED non-blocking attacks (F3, F4, F5, F8, F11, F13) cluster on: (a) acceptance prose too descriptive where a concrete-tuple pin is sharper (F3 determinism, F4 JSON-order, F5 CRLF, F11 test LOC, F13 DAG-vs-import-order, F12 fixture content coverage), and (b) one genuine scope-rule drift (F8 bootstrap carve-out). None of these invalidate the plan's structure — the unit decomposition is sound, the laslig API surface is correctly identified, the dep-DAG is legal, and the hand-off pins from Drop 1 (unexported `count` + `Counts` in `cmd/rak/root.go`) correctly land in 2.1. The non-blocking CONFIRMED items should route to Phase 3 dev discussion (F8, which has a CLAUDE.md-vs-plan tension requiring dev call) and to a planner re-spawn round (F3, F4, F5, F11, F12, F13, which are plan-text sharpenings the planner can fold in without dev approval). Nine attacks REFUTED outright — the plan withstands its easy attack surfaces. A zero-attack result would have been suspicious; a six-mitigation-needed result is honest adversarial review on a solid plan.

## Hylla Feedback

N/A — Drop 2 plan-QA falsification leaned on (a) non-Go evidence (`main/drops/DROP_2_*/PLAN.md`, `main/PLAN.md`, `main/CLAUDE.md`, `main/drops/WORKFLOW.md`, `main/drops/DROP_1_*/PLAN.md`), (b) Drop-1 in-flight code at `main/cmd/rak/root.go` (current working copy — Hylla ingest would lag any active-unit edits), and (c) laslig module-cache source at `/Users/evanschultz/go/pkg/mod/github.com/evanmschultz/laslig@v0.2.4/` which is not part of rak's Hylla-indexed artifact. No Hylla query was attempted because the evidence sources are all non-rak-Hylla-indexed; falling back to `Read` / `Grep` is the correct path, not a Hylla miss.
