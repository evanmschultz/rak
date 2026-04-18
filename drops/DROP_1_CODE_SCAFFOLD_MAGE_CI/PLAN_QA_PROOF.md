# DROP_1 Plan QA — Proof (Round 2)

**Reviewer:** go-qa-proof-agent
**Reviewed:** 2026-04-18
**Plan SHA at review:** ca0ba6e

## Verdict

**pass**

The Round 2 revision absorbs every brief item I was asked to verify, every Round 1 nit is either resolved or explicitly deferred to canonical docs (CLAUDE.md `de588d7` + main/PLAN.md Follow-Ups), and every acceptance criterion remains yes/no-checkable from the touched paths. No residual findings rise above informational.

## Findings

No blocker or nit findings. Three informational notes recorded for orchestrator visibility — none require planner action.

### Finding 1 — Stash main.go body shape vs Unit 1.1 acceptance: minor cosmetic mismatch

- **Severity:** note
- **Unit reference:** 1.1
- **Issue:** Unit 1.1 acceptance line 48 pins the `main()` body to exactly `if err := fang.Execute(context.Background(), newRootCmd()); err != nil { os.Exit(1) }` (one statement). The stash `main.go` lines 34–38 use the same shape verbatim, so 1.1's "lift verbatim" instruction matches. This is a confirmation, not a defect — flagged only because the literal body-text pin is unusually strict for a "verbatim lift" unit. If `gofumpt` rewrites brace placement post-1.1, the grep-based pin still survives because the pinned text contains no whitespace-sensitive tokens. No action needed.
- **Evidence:** `/tmp/rak-stash/main.go` lines 34–38; `drops/DROP_1_CODE_SCAFFOLD_MAGE_CI/PLAN.md` line 48.
- **Suggested fix:** none.

### Finding 2 — `mage build` exit-0 acceptance for 1.5 implicitly verifies 1.2's module-path rewrite

- **Severity:** note
- **Unit reference:** 1.5 (downstream of 1.2)
- **Issue:** Unit 1.2 acceptance is grep-only (no compile check) per the explicit bullet on line 66 — that defers compile verification to 1.5's `mage build`. PLAN.md line 121 makes the chain explicit ("validates 1.1 + 1.2 + 1.3 + 1.4 + 1.5 all compile together"). This is correct, but it means a 1.2 module-path bug would not surface until 1.5 build-QA, four units later. The plan acknowledges this trade-off (raw `go build` is forbidden) and prefers it over violating the mage-only rule. No action needed; recording for orchestrator awareness during 1.2 build-QA.
- **Evidence:** `drops/DROP_1_CODE_SCAFFOLD_MAGE_CI/PLAN.md` lines 66, 121.
- **Suggested fix:** none.

### Finding 3 — Prerequisites § doc-only by design; no programmatic gate

- **Severity:** note
- **Unit reference:** Prerequisites § (between Scope and Planner)
- **Issue:** New Prerequisites § (lines 16–24) names the three dev-installed tools and tells the builder to "pause and surface the gap to the orchestrator rather than installing it from inside an agent" if any tool is missing. There is no programmatic `mage --version` / `gofumpt --version` check baked into 1.4 or 1.5 acceptance — the gate is convention. This is consistent with the "tools are dev pre-state" framing and matches the carve-out scope (only `go get github.com/magefile/mage` is allowed inside an agent, not arbitrary tool installs). The convention-only gate is the right level for Drop 1; recording so the orchestrator can verify tool presence before spawning the 1.4 builder.
- **Evidence:** `drops/DROP_1_CODE_SCAFFOLD_MAGE_CI/PLAN.md` lines 16–24, 93.
- **Suggested fix:** none — flag for orchestrator's pre-1.4 spawn checklist, not for planner.

## Coverage Summary

- **Units reviewed:** 1.1, 1.2, 1.3, 1.4, 1.5, 1.6.
- **Decisions cross-checked:** 22 (coverage report-only + scope), 27 (architecture + hand-off boundary), 28 (quality tooling), 29 (concurrency + error idioms / `RunE` threading `cmd.Context()`).
- **CLAUDE.md sections cross-checked:** "Build Verification" mage target table (planCheck rename verified), "Go Development Rules" → "Dependencies" → "Bootstrap carve-out" (new paragraph cited by 1.4), "Project Structure" file LOC targets (1.1 `main.go` ≤ 30 LOC, 1.3 `root.go` ≤ 150 LOC), "Project Structure" Go-Idiomatic Naming Rules (camelCase target name `planCheck`, unexported `count`).
- **main/PLAN.md cross-checked:** Decisions Locked In (22, 27, 28, 29), Expected Decomposition lines 78–105 (planCheck rename verified), Follow-Ups (new "Pin `gofumpt` + `golangci-lint`" entry verified, replaces planner duplicate).
- **Round 2 brief items verified (1–15):**
  - **1.** PASS — `grep plan-check` returns zero matches; `grep planCheck` returns three (lines 107, 120, 150).
  - **2.** PASS — line 103 attributes `package main` to "Go's one-package-per-directory rule" with the `//go:build mage` build tag explicitly excluding it from the normal build surface; no longer attributed to a mage convention.
  - **3.** PASS — line 77 carries the full triple: `func count(` returns exactly one line, `func Count(` returns zero lines, `type Counts struct` returns exactly one line. Anti-pin + pin + struct survival all present.
  - **4.** PASS — lines 116–119 spell out variants (a) `-coverpkg=./internal/...` (no scope-tighten TODO) and (b) `-coverpkg=./...` (with `// TODO(drop-9.3)` comment) plus an explicit "Internal consistency between the `-coverpkg` flag and the comment is what QA verifies" gate.
  - **5.** PASS — 1.6 line 141 explicitly says `gh run watch --exit-status` is **not** a 1.6 unit acceptance criterion and cross-references WORKFLOW.md § "Phase 6 — Verify"; the prior round's `gh run watch` bullet was replaced with a YAML-parses-only check (line 140).
  - **6.** PASS — 1.4 line 93 cites the carve-out by name: "main/CLAUDE.md § 'Go Development Rules' → 'Dependencies' → 'Bootstrap carve-out'". Carve-out paragraph confirmed present in CLAUDE.md `de588d7` (CLAUDE.md line 262).
  - **7.** PASS — `Prerequisites` § (no separate brief item but checked) — present at lines 16–24.
  - **8.** PASS — Notes line 153 cross-refs main/PLAN.md § "Follow-Ups" → "Pin `gofumpt` + `golangci-lint` versions in Drop 9" without duplicating the entry locally; main/PLAN.md line 188 carries the canonical entry.
  - **9.** PASS — 1.5 line 124 has the `.golangci.yml` fallback clause: "Empty default (no `.golangci.yml`) is the preferred outcome and should be tried first; the minimal config is a graceful escape valve, not the target state."
  - **10.** PASS — `## Prerequisites` section exists at line 16, between `## Scope` (line 12) and `## Planner` (line 26), listing all three install commands (mage, gofumpt, golangci-lint).
  - **11.** PASS — 1.3 line 75 locks `RunE` body to exactly `return fmt.Errorf("not implemented — see drop 2")`; line 78 reinforces with "No alternate count-and-print body permitted in Drop 1." The Round 1 OR-fork is gone.
  - **14.** PASS — DAG block (lines 30–34) unchanged from Round 1: 1.1 → 1.2 → 1.3, with 1.1 → 1.4 → 1.5 → 1.6 as the second branch. Prose explanation (line 36) matches.
  - **15.** PASS — Scope paragraph (line 14) unchanged from Round 1; expected decomposition pointer to "main/PLAN.md lines 78–105" still correct (lines 78–105 in main/PLAN.md cover Drop 1 decomposition).
- **Persisted-from-Round-1 properties re-verified:**
  - Unit decomposition covers expected decomposition 1:1 (six units 1.1–1.6, one per main/PLAN.md sub-bullet).
  - Acceptance criteria are falsifiable (every bullet is grep-checkable, exit-code-checkable, or file-presence-checkable).
  - All paths are under `main/` (verified — every `Paths:` row prefixes with `main/` or names `/tmp/rak-stash/` for the stash source).
  - Drop 2.1 hand-off boundary pinned (1.3 line 77 + Notes line 151).
