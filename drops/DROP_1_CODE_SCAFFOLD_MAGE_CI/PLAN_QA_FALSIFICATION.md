# DROP_1 Plan QA — Falsification (Round 2)

**Reviewer:** go-qa-falsification-agent
**Reviewed:** 2026-04-18
**Plan SHA at review:** ca0ba6e (`docs(drop-1): planner round 2 absorb plan-qa round 1 findings`)

## Verdict

**pass**

Round 2 plan closes the one Round 1 blocker (`plan-check` → `planCheck` rename), carries the accepted fixes (C4 tool-version pin Follow-Up, C5 RunE stub pin, C6 `Counts` struct pin, C7 coverage variant pairing, Prerequisites §, 1.6 `gh run watch` removal) into the drop's PLAN.md without regressing the still-good surface, and introduces no new attack surfaces that refute the plan. Falsification attempts below are exhaustive and every CONFIRMED row downgrades to informational note, not counterexample.

## Counterexamples

No counterexamples.

Three informational notes follow. None falsify the plan; they are record-keeping surfaces that downstream readers (builder in Phase 4, build-QA in Phase 5) may reuse.

### Note N1 — `.golangci.yml` fallback lacks a grep-verifiable scope guard
- **Severity:** note
- **Unit reference:** 1.5
- **Scenario:** 1.5 acceptance line 124 ("Fallback clause") permits the builder to commit `main/.golangci.yml` "enabling only the default linter set". A builder could commit a `.golangci.yml` with `linters: enable-all: true` (or other non-default shape) and still pass the `mage lint` / `mage ci` exit-code checks. The "default linter set" scope is prose-enforced, not grep-enforced.
- **Evidence:** drops/DROP_1_CODE_SCAFFOLD_MAGE_CI/PLAN.md:124 — acceptance is "enabling only the default linter set" but the only machine-checkable gate is `mage lint` / `mage ci` exit code.
- **Suggested mitigation:** None required for plan-acceptance; flag for build-QA. When Phase 5 QA runs on 1.5, if `.golangci.yml` exists, reader-level check should confirm the YAML body matches the minimal-config rationale. Not a blocker.

### Note N2 — Prerequisites § enforcement is failure-implicit, not acceptance-explicit
- **Severity:** note
- **Unit reference:** Prerequisites (drop-level, affects 1.4 + 1.5)
- **Scenario:** If `mage`, `gofumpt`, or `golangci-lint` is missing from the dev's `$PATH` when 1.4/1.5 spawn the builder, the Prerequisites § (drop PLAN.md lines 16–24) says the builder "pauses and surfaces the gap to the orchestrator rather than installing it from inside an agent." There is no grep-verifiable acceptance that enforces this — the pause-and-surface behavior relies on: (a) the tool-missing error surfacing from the first mage invocation ("`exec: gofumpt: executable file not found`"), and (b) the builder recognizing that class of error as a Prerequisites escalation rather than attempting to `go install` from inside the agent.
- **Evidence:** drops/DROP_1_CODE_SCAFFOLD_MAGE_CI/PLAN.md:24 — "If any tool is missing … the builder pauses and surfaces the gap to the orchestrator rather than installing it from inside an agent." No acceptance bullet binds this to a check.
- **Suggested mitigation:** None required. Enforcement is implicit-via-failure plus agent-preamble discipline (global go-builder-agent is not authorized to run `go install` for tool bootstrap). Worth knowing during Phase 4.

### Note N3 — 1.4 "re-running produces no diff" is second-run idempotency, not first-run stability
- **Severity:** note
- **Unit reference:** 1.4
- **Scenario:** 1.4 acceptance line 94 reads `go mod tidy` run from `main/` leaves `go.mod` + `go.sum` stable (re-running produces no diff). Expect a large `go.sum` diff on first `tidy` …". The parenthetical makes it unambiguous that "stable" means "settled after the first tidy, idempotent on the second run". A strict reader who stops at "leaves go.mod + go.sum stable" without the parenthetical could misread this as "first run produces no diff", which would refute against the expected large stash-vs-rak prune. The parenthetical + Notes § line 147 together clarify fully — this is prose clarity, not a falsification.
- **Evidence:** drops/DROP_1_CODE_SCAFFOLD_MAGE_CI/PLAN.md:94 (acceptance) + line 147 (`go.sum drift` note).
- **Suggested mitigation:** None. The text already reads correctly if read to completion.

## Attack Surfaces Re-Explored

Round 1 surfaces re-tested against the revised plan, plus surfaces newly exposed by the Round 2 edits. One-line outcome per lens.

**Regression from Round 1 fixes**

- `plan-check` → `planCheck` rename completeness — REFUTED. `grep -rn 'plan-check' main/` returns a single hit inside the transient `PLAN_QA_PROOF.md` (about to be `git rm`'d at end of Round 2). Zero stale strings in CLAUDE.md / main/PLAN.md / drop PLAN.md. Context7 (`/magefile/mage`) confirms exported-function rule + first-letter lowercasing: Go method `PlanCheck()` displays as `planCheck` in `mage -l`. Rename is internally consistent.
- Coverage variant (a)/(b) pairing ambiguity — REFUTED. Acceptance line 116–119 binds `-coverpkg` flag choice to matching TODO-comment state inside the magefile target body. QA-verifiable via reading the magefile; consistent with decision 22's locked scope.
- `gh run watch` removal from 1.6 — REFUTED. Removed cleanly per WORKFLOW.md Phase 6 drop-end verification ownership; 1.6 now ends at YAML well-formed + grep checks. New explanatory Note bullet on line 141 pins the scope reassignment to Phase 6.
- Prerequisites § as potential agent-install path — REFUTED. Section explicitly says "rather than installing it from inside an agent"; burden lands on the dev pre-state, not on the builder subagent. No `go install` invocation appears in any unit acceptance. Global `go-builder-agent` forbids unsanctioned installs.

**New surfaces introduced**

- Bootstrap carve-out fit for 1.4 — REFUTED. `main/CLAUDE.md:262` carve-out conditions ("no mage target yet exists to wrap `go get`") exactly match 1.4's state: magefile.go lands in 1.5, after 1.4. 1.4 acceptance line 93 cites the carve-out by path and re-states the exactness conditions.
- `.golangci.yml` fallback scope creep — downgraded to Note N1 above.
- 1.1 stub `main.go` signal-handling gap between 1.1 and 1.3 — REFUTED. Context7 (`/charmbracelet/fang`) shows `fang.Execute(ctx, cmd)` two-arg call is a documented valid invocation (variadic options). Between 1.1 and 1.3 the binary never runs interactively — 1.5's `mage build` is the first compile, `mage test` has no tests to run, `mage run` is not invoked in unit acceptance. No path accepts a signal during the 1.1→1.3 window.
- 1.3 `root.go` `fmt` import coherence — REFUTED. `RunE` body `return fmt.Errorf("not implemented — see drop 2")` requires `fmt` in `root.go`. Stash main.go already imports `fmt` (line 7 of `/tmp/rak-stash/main.go`); 1.3's root.go inherits from that split. `gofumpt` + `goimports` manage import state; `mage ci`'s "`gofumpt -l .` empty" assertion catches drift. Not a plan-level gap.

**Still-valid Round 1 attack surfaces — re-run against Round 2 plan**

- Ordering hazards — REFUTED. DAG at lines 30–34 shows 1.1 → {1.2, 1.4}; 1.2 → 1.3; 1.4 → 1.5 → 1.6. `blocked_by` on each unit row matches the DAG. No cycle; no hidden serialization.
- Drop 2.1 contract mismatch on `Counts` — REFUTED. 1.3 acceptance line 77 pins both `count(io.Reader) (Counts, error)` AND `type Counts struct` with explicit grep tests on each. Notes § line 151 reaffirms the hand-off surface. Pinned at plan level.
- YAGNI pressure — REFUTED. Prerequisites § and `.golangci.yml` fallback are escape-valve only; preferred outcome is the empty/absent state. No speculative abstraction introduced. Bootstrap carve-out applies to exactly one unit (1.4).
- Hidden deps — REFUTED. Every unit's `blocked_by` is explicit; no drop-level implicit ordering.
- Mage-discipline bypass — REFUTED. 1.4's `go get` is the one authorized carve-out bypass (well-scoped by CLAUDE.md line 262). 1.2 line 66 explicitly forbids raw `go build ./...` during the pre-magefile window. All other mage-verbs in acceptances are the mage targets from CLAUDE.md's mage target table.
- Decision drift — REFUTED. Locked decisions 22 (coverage scope), 27 (file-size cap), 28 (quality tooling), 29 (concurrency + error idioms) all cited faithfully. Coverage variants both trace back to locked scope `-coverpkg=./internal/...`.
- Package `main` collision — REFUTED. `main/magefile.go` has `//go:build mage` tag excluding it from normal builds; `cmd/rak/main.go` is the real `main` package under `cmd/rak/` (different directory). Plan does not create any other `.go` file at the `main/` directory root.
- Stash `go.sum` lifecycle — REFUTED. 1.1 copies unmodified; 1.4 `go mod tidy` prunes per Notes line 147; closeout Phase 7 deletes `/tmp/rak-stash/`. Full lifecycle traced.
- Stub RunE body exactness — REFUTED. Exact string `return fmt.Errorf("not implemented — see drop 2")` pinned at lines 75 + 78 with a dedicated grep acceptance. No alternate count-and-print body permitted.
- Acceptance falsifiability — REFUTED. Every unit's acceptance is either grep-checkable or mage-exit-code-checkable. Soft surfaces (Prerequisites enforcement, `.golangci.yml` content semantics, YAML validity) are explicit about being soft and reference their enforcement path.

**Proof agent's brief-item coverage — non-contiguous numbering 7/12/13**

- Item 7 (go.sum drift note, C3) — no-op confirmed. Notes § line 147 retained unchanged in substance.
- Item 12 (YAML acceptance wording, C8) — no-op confirmed. 1.6 acceptance line 140 retains the soft `gh workflow view` check.
- Item 13 (mage install tripwire, C2) — no-op confirmed. 1.5 acceptance line 114 retains the grep-verifiable "dev-only; agents MUST NOT invoke." comment requirement; line 126 retains the explicit "Agents MUST NOT invoke `mage install`" assertion.

**`go mod tidy` idempotency**

- REFUTED. With default `GOPROXY` semantics on a settled module graph, `go mod tidy` is idempotent on second run by construction. Acceptance correctly tests the second-run diff, not the first.
