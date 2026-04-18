# DROP_1 Plan QA — Proof (Round 1)

**Reviewer:** go-qa-proof-agent
**Reviewed:** 2026-04-18
**Plan SHA at review:** 2fa8bf8

## Verdict

**pass**

Every unit's acceptance list is yes/no-verifiable, the DAG matches the scope sentence, all six expected-decomposition bullets in `main/PLAN.md` lines 78–105 are covered without scope creep, and the hand-off boundaries (decisions 22, 27, 28, 29) are pinned correctly. Five non-blocking nits noted below for the planner to consider in Phase 3, none of which gate Phase 4.

## Findings

### Finding 1 — Unit 1.5 magefile.go package name claim is incorrect

- **Severity:** nit
- **Unit reference:** 1.5 (Packages line)
- **Issue:** PLAN.md line 92 says the magefile lives "in package `main` under the `//go:build mage` constraint, per `mage` conventions". Mage convention is the opposite — magefiles use `package main` only when run with the legacy bootstrap binary; the modern `mage` tool (which is what `github.com/magefile/mage` ships) requires the file declare its package as the surrounding directory's package, NOT `main`, and uses a build tag like `//go:build mage` so the file is excluded from normal builds. Since `main/magefile.go` lives at the module root where there is **no other Go source file** (Go source lives under `cmd/rak/`), the directory has no other package and the magefile typically declares `package main` — so the claim happens to be correct for *this* layout, but the reasoning ("per mage conventions") is wrong and could mislead a builder reading the line literally.
- **Evidence:** PLAN.md line 92: "package `main` under the `//go:build mage` constraint, per `mage` conventions". Mage's actual convention (`go doc github.com/magefile/mage` + project README) is "any package name, build tag `mage`". The fact that `main/magefile.go` ends up in package `main` is a consequence of where the file sits in the tree (module root with no siblings), not a mage convention.
- **Suggested fix:** Reword to "package `main` (the magefile sits at `main/`'s module root with no sibling Go files in that directory; mage's only requirement is the `//go:build mage` build tag — the package name follows directory convention)." This is a documentation-clarity nit; the acceptance test (line 95 — checks for the build-tag + the import) is unaffected.

### Finding 2 — Unit 1.3 `count` first-drop hand-off boundary acceptance is strong but missing a `Counts` symmetry check

- **Severity:** nit
- **Unit reference:** 1.3
- **Issue:** PLAN.md line 67 pins `count` (lowercase) and forbids `Count` (uppercase) via grep. PLAN.md line 71 separately requires the `Counts` struct to "survive intact for Drop 2.1 to lift". But there is no grep-style acceptance check that `Counts` (the struct) is still defined in `root.go` after 1.3's rewrite — only a prose requirement. A QA subagent doing only the grep checks would not catch a builder who deleted `Counts` to "simplify" alongside the wc-flag removal.
- **Evidence:** PLAN.md line 71 "`count(io.Reader) (Counts, error)` and the `Counts` struct MUST survive intact for Drop 2.1 to lift" — prose. PLAN.md line 67 grep checks cover the function but not the struct.
- **Suggested fix:** Add a grep acceptance line under 1.3: "`grep -n 'type Counts struct' main/cmd/rak/root.go` returns exactly one line (struct survives the rewrite for Drop 2.1 to lift)." Trivial to add; closes the symmetry gap.

### Finding 3 — Unit 1.5 `mage coverage` zero-match-package contingency is ambiguous between two acceptable behaviors

- **Severity:** nit
- **Unit reference:** 1.5 + Notes
- **Issue:** PLAN.md line 105 lists two acceptable outcomes for `mage coverage` in Drop 1 (zero internal packages exist): "exiting 0 in Drop 1 or gracefully producing an empty profile" — and offers a fallback to `-coverpkg=./...` with a TODO if Go 1.26 errors on zero-match. The acceptance criterion is "exits 0", which is the load-bearing piece. But the prose surrounding it conflates two sub-cases: (a) `go test -coverpkg=./internal/...` works on zero matches in Go 1.26 (acceptance: target uses `./internal/...`); (b) it errors and the builder swaps to `./...` with TODO. A QA subagent reviewing 1.5's implementation needs to know whether the final magefile.go must contain `./internal/...` or whether `./...` is also acceptable. The current wording leaves both legal but doesn't tell QA how to disambiguate.
- **Evidence:** PLAN.md lines 105 + 137 (Notes coverage scope footnote) — both flag the contingency but neither gives QA a unique acceptance string.
- **Suggested fix:** Tighten 1.5's `coverage` acceptance to: "Final magefile.go uses one of: (a) `-coverpkg=./internal/...` with target exiting 0, OR (b) `-coverpkg=./...` with a `// TODO(drop-9.3): tighten to ./internal/... once internal/ exists` comment on the line above the flag — builder picks based on which Go 1.26 accepts. QA verifies the chosen variant is internally consistent (TODO present iff `./...` chosen)." Removes the disambiguation question.

### Finding 4 — Unit 1.6 `gh run watch --exit-status` acceptance straddles unit and drop-end verification

- **Severity:** nit
- **Unit reference:** 1.6
- **Issue:** PLAN.md line 126 says "After pushing, `gh run watch --exit-status` on the triggered workflow run exits green. This is the drop-end verification per WORKFLOW.md Phase 6 — acceptance here is the expectation the first green run will happen post-merge; the unit itself passes when `mage ci` passes locally and the YAML is syntactically valid". This conflates two distinct gates: (a) unit 1.6 acceptance (YAML valid + `mage ci` green locally), which can be checked at end of Phase 5 build-QA, and (b) drop-end Phase 6 verification (push + `gh run watch`), which happens AFTER all six units close. The current wording is *technically* correct (it explicitly defers the watch to drop-end) but a QA subagent reading "After pushing, `gh run watch --exit-status` on the triggered workflow run exits green" as bullet-point #5 of the unit's acceptance might mark the unit as `blocked` because it can't watch a run that hasn't happened yet.
- **Evidence:** PLAN.md line 126 + WORKFLOW.md Phase 6 (lines 144–154 of `WORKFLOW.md`) which clearly puts `git push` + `gh run watch --exit-status` at drop-end, not per-unit.
- **Suggested fix:** Split 1.6's acceptance into two bullets: (a) "Unit-level acceptance (verifiable end of Phase 5): YAML parses, workflow declares the right triggers/jobs/steps, `mage ci` runs green locally — `gh workflow view` or `actionlint` may be used as the YAML check." (b) "Drop-end verification (deferred to Phase 6, not this unit): `git push` + `gh run watch --exit-status` green is owned by the orchestrator's Phase 6, not by this unit's QA pass." Removes the unit-vs-drop-end ambiguity.

### Finding 5 — Unit 1.4 implicit dep on dev's `go get` permission is correctly noted but the rule citation is reversed

- **Severity:** nit
- **Unit reference:** 1.4
- **Issue:** PLAN.md line 82 says: "The dep is added via `go get github.com/magefile/mage` run from `main/` — NOT hand-edited. Builder runs the command (this is the one builder-run invocation the project allows outside `mage` per main/CLAUDE.md § 'Go Development Rules' → 'Dependencies', since no mage target exists yet)." But `main/CLAUDE.md` § "Go Development Rules" → "Dependencies" (lines 259–261) actually says: "Ask the dev to run `go get` / module updates. No `GOPROXY=direct`, `GOSUMDB=off`, or checksum bypass." That is the **opposite** of what PLAN.md cites — CLAUDE.md routes `go get` to the dev, not to the builder. So 1.4 is asking the builder to run `go get` while the project rule says only the dev does that.
- **Evidence:** `main/CLAUDE.md` lines 259–261 — "Ask the dev to run `go get` / module updates." vs `main/drops/DROP_1_CODE_SCAFFOLD_MAGE_CI/PLAN.md` line 82 — "Builder runs the command (this is the one builder-run invocation the project allows…)".
- **Suggested fix:** Either (a) reword 1.4 to "Builder requests dev to run `go get github.com/magefile/mage` from `main/`; once dev confirms, builder runs `go mod tidy` (which is read-only on the network with module cache populated and is NOT a `go get`)" — preserving the CLAUDE.md rule, OR (b) update CLAUDE.md to add an explicit Drop 1 carve-out for the bootstrap mage dep. (a) is the cleaner Phase-3 fix. Note this is a procedural / chain-of-custody nit, not a correctness blocker — the resulting `go.mod` and `go.sum` will be identical either way; what differs is who hits Enter.

## Coverage Summary

- **Units reviewed:** 1.1, 1.2, 1.3, 1.4, 1.5, 1.6 — all six covered by PLAN.md, all six map to expected-decomposition bullets in `main/PLAN.md` lines 78–105 with no extras and no gaps.
- **Decisions cross-checked:**
  - Decision 5 (Layout) — 1.1's stash-into-`cmd/rak/` move respects layout.
  - Decision 6 (Tech stack) — 1.4 adds mage; 1.3 wires fang signal handling; cobra + fang already in stash go.sum.
  - Decision 15 (Orchestrator-never-edits-Go) — meta, not unit-acceptance, but PLAN.md correctly lifts every Go-touching unit into a builder spawn (no orch-edit-Go path).
  - Decision 16 (Coordination model) — drop-dir layout matches; PLAN.md is in `main/drops/DROP_1_CODE_SCAFFOLD_MAGE_CI/` per spec.
  - Decision 22 (Coverage report-only in Drop 1) — explicitly verified in 1.5 acceptance line 105 + Notes line 137 + finding 3 above.
  - Decision 23 (CI in Drop 1) — 1.6 covers; verified.
  - Decision 27 (`count` stays unexported in 1.3, Drop 2.1 owns export) — verified in 1.3 acceptance line 67 + Notes line 136 (pinned both places).
  - Decision 28 (mage-only, no raw `go`) — 1.4's `go get` + `go mod tidy` flagged in finding 5; otherwise verified.
  - Decision 29 (RunE threads `cmd.Context()` downward) — verified in 1.3 acceptance line 70.
- **Sections of `main/CLAUDE.md` cross-checked:**
  - § "Project Structure" → "File Breakdown" (file LOC targets) — 1.1's ≤30 LOC for `main.go` and 1.3's ≤150 LOC for `root.go` match the table at CLAUDE.md lines 130–131.
  - § "Build Verification" → mage target table — 1.5's 9 targets match the table at CLAUDE.md lines 204–214 exactly (`build`, `test`, `format`, `lint`, `ci`, `install`, `run`, `coverage`, `plan-check`).
  - § "Build Verification" rule 3 (NEVER `mage install` from agent) — pinned in 1.5 acceptance line 112 + Notes line 134.
  - § "Go Development Rules" → "Dependencies" — flagged in finding 5.
  - § "Go Development Rules" → "Errors" — not directly in scope for Drop 1 (no new error-wrapping surface yet); 1.3's `RunE` stub option correctly stays minimal so error idioms can land cleanly in Drop 2.
- **`main/drops/WORKFLOW.md` cross-checked:**
  - § "Phase 1 — Plan" — PLAN.md fills the Planner section per spec; six units with state/paths/packages/acceptance/blocked_by all present.
  - § "File Lifecycle" — `PLAN.md` is the durable file the planner edits; this proof file (`PLAN_QA_PROOF.md`) is transient per the table.
  - § "Phase 6 — Verify" — flagged in finding 4 (drop-end verification belongs to Phase 6, not unit 1.6 acceptance).
