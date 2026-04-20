# DROP_2 — Plan QA Proof Review (Round 1)

**Reviewer:** go-qa-proof-agent
**Round:** 1
**Date:** 2026-04-19
**Verdict:** fail

## Summary

The plan is well-sourced and evidentially grounded — every cited external symbol resolves (laslig v0.2.4 `policy.go:211` `ResolveMode`, `printer.go:39` `New`, `printer.go:48` `NewWithMode`, `Policy{Format, Style}` struct shape, `KV` block type), the lift source in `cmd/rak/root.go:42-78` matches what the planner describes, `main/go.mod` + `main/go.sum` confirm the absent-laslig claim, the 5→4 unit reshape rationale is defensible, and Decisions 25 + 27(d) are explicitly respected. However, the proof review surfaces two blocking issues that prevent a clean pass: (1) Unit 2.2's `blocked_by: —` contradicts its own acceptance, which imports `internal/counting`; (2) the bootstrap carve-out invocation for laslig re-add conflicts with `main/CLAUDE.md` § "Go Development Rules" → "Bootstrap carve-out" which explicitly scopes that carve-out to Drop 1.4 only. Both are fixable in a quick Phase 3 planner round; neither touches overall scope.

## Findings

### P1 — Unit 2.2 `blocked_by: —` contradicts its own Renderer signature import

- **Severity:** blocking
- **Unit(s):** 2.2 (and indirectly the DAG on line 20–26)
- **Observation:** Unit 2.2's Renderer interface acceptance (PLAN.md line 53) specifies `type Renderer interface { Render(w io.Writer, counts counting.Counts) error }`, importing `github.com/evanmschultz/rak/internal/counting`. That package is created by Unit 2.1. If 2.2 is `blocked_by: —` (PLAN.md line 48) and runs in parallel with 2.1, the `internal/render` package cannot compile until `internal/counting` exists — `mage build` at the end of Unit 2.2's acceptance (line 61) would fail. The planner text on line 53 acknowledges this ("verifying 2.1 → 2.2 is the natural dep-DAG order") but does not translate the acknowledgement into an explicit `blocked_by: 2.1` header.
- **Evidence:** PLAN.md line 48 (`Blocked by: —`) vs PLAN.md line 53 (`Render(w io.Writer, counts counting.Counts) error`) vs PLAN.md line 61 (`mage build` acceptance bullet) vs PLAN.md § Planner DAG lines 20–26 showing 2.1 and 2.2 as parallel siblings.
- **Recommendation:** revise. Either (a) add `Blocked by: 2.1` to Unit 2.2's header and update the DAG to `2.1 → 2.2 → 2.3 → 2.4` (with 2.2's blocked_by noted), OR (b) reshape the Renderer interface to not import `counting` at all (e.g. `Render(w io.Writer, v any) error` with a type assertion — unlikely to be clean, likely worse), OR (c) split Unit 2.2 into a "render interface shell (no counting import)" piece that IS parallel with 2.1 and a "human + JSON renderer implementations" piece that depends on both. Option (a) is the cleanest — it costs the parallelism between 2.1 and 2.2 but resolves the contradiction. The drop has 4 units; linear ordering is not expensive here.

### P2 — Bootstrap carve-out invocation conflicts with CLAUDE.md § "Dependencies"

- **Severity:** blocking
- **Unit(s):** 2.2
- **Observation:** Unit 2.2's first acceptance bullet (PLAN.md line 50) invokes the bootstrap carve-out to run `go get github.com/evanmschultz/laslig@v0.2.4` + `go mod tidy` directly from `main/`. `main/CLAUDE.md` § "Go Development Rules" → "Dependencies" states: *"Today this applies only to Drop 1.4 (first-ever `github.com/magefile/mage` add). From Drop 2 onward the magefile exists, so every dep add routes through a mage target and this carve-out does not apply."* The plan's invocation directly contradicts this rule. The magefile DOES exist (Drop 1.5 landed it), but Drop 1 did not add a dep-management target to it — so the rule's assumption ("every dep add routes through a mage target") is ahead of the actual tooling. This is a real, not cosmetic, conflict.
- **Evidence:** PLAN.md line 50 ("bootstrap carve-out (one-time): builder runs `go get github.com/evanmschultz/laslig@v0.2.4`") vs CLAUDE.md § Dependencies ("From Drop 2 onward … this carve-out does not apply") vs `main/magefile.go` actual targets list from Drop 1.5 (build / test / format / lint / ci / install / run / coverage / planCheck — no `deps` or `modTidy` target).
- **Recommendation:** revise. Three viable options, each surfaces as a Phase 3 dev decision: (a) **Add a Unit 2.0** that creates `mage deps` or equivalent target (`go get` + `go mod tidy` wrapper) before Unit 2.2 runs its dep add. This honors the CLAUDE.md rule strictly. (b) **Amend CLAUDE.md § Dependencies** to extend the carve-out: "carve-out applies until a mage target exists to wrap dep operations" — then Drop 2.2's invocation is legal today. (c) **Add a mage dep-mgmt target as an explicit first sub-step of Unit 2.2**, bundling the target-add + dep-add into one atomic unit (borderline non-atomic but defensible since both operations are needed for the unit's other acceptance bullets to hold). Phase 3 with dev chooses.

### P3 — Unit 2.2 snapshot-test determinism mechanism is under-specified

- **Severity:** minor
- **Unit(s):** 2.2
- **Observation:** PLAN.md line 58 defers the snapshot-determinism choice to the builder — "builder chooses mechanism — acceptance is that the test is deterministic across TTY / non-TTY CI environments". The two offered mechanisms (unexported test-only constructor vs `NewWithMode` with explicit `Mode{Format: FormatPlain, Styled: false, Width: 80}`) have meaningfully different public surface consequences: (a) introduces an unexported `laslig.Mode` plumbing path, (b) keeps the public surface strictly `NewHumanRenderer()` and only plumbs `NewWithMode` in tests. From a proof-review perspective, QA cannot mechanically assert pass/fail against "builder chooses" — the acceptance bullet collapses into a yes/no question only after the builder picks. Falsification may reasonably flag this as ambiguity too.
- **Evidence:** PLAN.md line 58 ("builder chooses mechanism").
- **Recommendation:** revise. Pick one mechanism in the plan (recommend option b — `NewWithMode` with explicit `Mode` in tests, keeping the `NewHumanRenderer` public surface minimal; this also matches laslig's own docs in `/Users/evanschultz/go/pkg/mod/github.com/evanmschultz/laslig@v0.2.4/doc.go` and printer.go:48 which explicitly documents `NewWithMode` as "a convenience for callers that already resolved the output mode"). Pinning the mechanism makes the acceptance yes/no-verifiable.

### P4 — Unit 2.3 stdin-vs-path-arg behavior is gated on Phase 3 dev approval — acceptance bullet will change

- **Severity:** info
- **Unit(s):** 2.3
- **Observation:** PLAN.md line 75 explicitly marks the `len(args)==1` behavior ("return error directing user to pipe stdin, mentioning Drop 3 walker") as "subject to Phase 3 dev approval". The Unknown is correctly surfaced (good hygiene), but the dependent test `TestRootCmd_RejectsPathArg` (line 85) encodes the proposed behavior — if dev picks the alternative ("silently ignore args"), that test flips from asserting error to asserting silent-ignore. This is fine as long as Phase 3 resolves the Unknown before Unit 2.3 is handed to the builder.
- **Evidence:** PLAN.md line 75 (proposed behavior + explicit "subject to Phase 3 dev approval"), line 85 (dependent test).
- **Recommendation:** defer. No plan edit required now — Phase 3 discussion resolves it, and Unit 2.3's plan rewording follows naturally from the dev's pick. Call it out in the Phase 3 brief so the dev doesn't miss the handoff.

### P5 — Laslig dep re-add Unknown (surfaced correctly, but worth re-flagging)

- **Severity:** info
- **Unit(s):** 2.2
- **Observation:** PLAN.md line 108 correctly re-states that Drop 1.5's `go mod tidy` pruned laslig and that Drop 1 predicted this. The Unknown is resolved by the fact of `v0.2.4` being pinned in acceptance. No residual Unknown once P2 is resolved — P5 is just noting that the planner's rationale chain is sound.
- **Evidence:** `main/go.mod` (no laslig require), `main/go.sum` (no laslig entries), laslig module cache `/Users/evanschultz/go/pkg/mod/github.com/evanmschultz/laslig@v0.2.4/` confirms presence, DROP_1_CODE_SCAFFOLD_MAGE_CI/CLOSEOUT.md § Refinements entry 4 cites version-pinning rule.
- **Recommendation:** accept. No change needed beyond P2 resolution.

### P6 — Plan implicitly commits `cmd/rak/root.go` import-list churn across two units — worth one sentence of call-out

- **Severity:** minor
- **Unit(s):** 2.1 + 2.3 (crosscutting)
- **Observation:** Unit 2.1 acceptance (line 37) says imports shrink to `fmt` + `cobra`, with `io` removable since `RunE` no longer uses it directly yet, then "2.3 re-adds `io` for stdin". Unit 2.3 acceptance (line 72) then re-adds `io` + `os`. This is a tiny churn pattern — net two edits to the import list across two units — but it's fine if both builder handoffs cleanly pick up the current state. The risk is the Unit 2.1 builder seeing `io` in the imports (because `RunE` uses `c.Context()` which technically doesn't need `io`), removing it, then Unit 2.3 has to re-add. Not a bug, just cosmetic churn. A single-sentence note in Unit 2.1 ("leave `io` in if gofumpt complains about unused — Unit 2.3 re-adds it anyway") could smooth the handoff.
- **Evidence:** PLAN.md line 37 ("`io` can also go since `RunE` no longer uses it directly yet; 2.3 re-adds `io` for stdin"), line 72 ("re-adds `io` for stdin piping and `os` for `os.Stdin`").
- **Recommendation:** accept. Minor enough that the builder can handle it; Phase 3 does not need to revise.

### P7 — Paths section at PLAN.md line 5 says "(lift `count` out)" but no unit does that lift cleanly on its own

- **Severity:** info
- **Unit(s):** global (top-of-PLAN paths line)
- **Observation:** PLAN.md line 5 `Paths (expected):` lists `main/cmd/rak/root.go (lift count out)`. Unit 2.1 (line 31) covers this correctly. No gap — just verifying the header line matches unit decomposition. ✓
- **Evidence:** PLAN.md line 5 vs Unit 2.1 line 31.
- **Recommendation:** accept. No change.

### P8 — `cmd/rak/testdata/` fixture contents are TBD — acceptance is not yes/no-verifiable until the file exists

- **Severity:** minor
- **Unit(s):** 2.4
- **Observation:** PLAN.md line 93 says "contents TBD by builder but stable and documented in test". This is a defensible deferral (the builder writes the fixture + hard-codes the expected `Counts` output in the test), but for the plan-QA reviewer there's nothing to verify until the builder commits. Acceptable under TDD-first ("tests ship with the package"). Falsification may push harder here.
- **Evidence:** PLAN.md line 93 ("contents TBD by builder").
- **Recommendation:** accept. Standard fixture-authoring deferral; fine as long as the test asserts exact expected output against exact fixture content.

## Verdict Rationale

**Fail.** Two blocking findings (P1 and P2) must resolve before Unit 2.2 is handed to a builder. P1 is a DAG / compile-order contradiction — the plan's own Renderer signature forces 2.1 to land before 2.2, yet the header leaves 2.2 unblocked. P2 is a documented-rule conflict — CLAUDE.md § "Dependencies" explicitly scopes the bootstrap carve-out to Drop 1.4, and the plan invokes it in Drop 2 without amending the rule or adding the mage target that CLAUDE.md assumes exists from Drop 2 onward.

P3 is a minor under-specification (choose one test-determinism mechanism). P4 and P5 are correctly-surfaced Unknowns routed for Phase 3 dev approval. P6, P7, P8 are info/minor notes that do not block a pass.

Both blocking findings are resolvable in a single Phase 3 planner round — P1 flips a `blocked_by` header line + DAG arrow; P2 surfaces three dev-pickable options (add Unit 2.0 / amend CLAUDE.md / bundle the mage target add into Unit 2.2). No scope reshape, no unit regeneration, no re-decomposition.

## Hylla Feedback

N/A — Drop 2 plan-QA is a review of a markdown plan against non-Go evidence sources (current `cmd/rak/root.go`, `go.mod` / `go.sum`, laslig module cache, CLAUDE.md rules, DROP_1 CLOSEOUT). Hylla was not queried as the primary source because (a) the target code to lift (`count` / `Counts`) is in uncommitted-or-freshly-committed state where the live file content trumps any committed snapshot, (b) laslig lives in the module cache and is external to rak's Hylla artifact, (c) CLAUDE.md and PLAN.md are markdown per CLAUDE.md § "Code Understanding Rules" rule 3 (non-Go → Read / Grep / Glob directly). No Hylla miss to record.
