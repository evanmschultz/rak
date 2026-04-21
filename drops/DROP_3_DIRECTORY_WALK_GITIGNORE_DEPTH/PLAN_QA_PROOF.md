# DROP_3 — Plan QA Proof — Round 1

**Agent:** go-qa-proof-agent
**Target:** `main/drops/DROP_3_DIRECTORY_WALK_GITIGNORE_DEPTH/PLAN.md`
**Round:** 1
**Verdict:** PASS (with 2 minor Phase 3 observations)

## Premises

- The plan decomposes Drop 3 into six atomic units (3.0–3.5) with explicit `blocked_by` ordering.
- Each unit's acceptance criteria are yes/no verifiable by a QA subagent.
- The plan pins 13 cross-unit invariants (F1–F13) and documents 4 open unknowns (U1–U4) for dev discussion.
- Library picks (`sabhiram/go-gitignore`, `bmatcuk/doublestar/v4`) are justified with explicit rationale vs. alternatives.

## Evidence

- Plan file read at `main/drops/DROP_3_DIRECTORY_WALK_GITIGNORE_DEPTH/PLAN.md`.
- Cross-referenced against `main/CLAUDE.md` § "Project Structure" (package map, import DAG, file breakdown), § "Errors" (sentinel + wrap conventions), § "Tests" (two-tier testdata rule).
- Cross-referenced against `main/PLAN.md` DROP_3 row (scope, `blocked_by`, paths, packages).
- Cross-referenced against Drop 2 pins that Drop 3 must honor (F9 `cmd.InOrStdin()` — not touched in Drop 3; F12 fixture coverage — extended by Drop 3.5 fixture tree).

## Trace or cases

- DAG cycle check: `ignore` (leaf) + `fileset → ignore` + `cmd/rak → fileset + counting + render`. No cycles. Matches CLAUDE.md import DAG.
- Per-unit acceptance: each unit's bullets map to concrete, testable behavior with file-path anchors.
- F-pins cross-reference: every pin cites at least one unit that enforces it.
- U1–U4 are genuinely open design calls, not hidden decisions — each carries a recommendation plus the alternative.

## Conclusion

The plan is internally consistent, decomposition is atomic, evidence is grounded in the repo's canonical docs, and library picks are defensible. **Proof-side PASS for Round 1.**

## Phase 3 observations (minor, non-blocking)

- **O1** — `main/CLAUDE.md` § "File Breakdown" table lists `file.go`, `walker.go`, `walker_test.go` but does NOT list `binary.go` / `binary_test.go`. Unit 3.4 adds these files. CLAUDE.md file breakdown should gain two rows (~80 LOC each) during Drop 3 closeout. Flag for orch attention at drop-end; not a planner blocker.
- **O2** — Unit 3.1 cites `doublestar.Match` but doublestar v4 exposes both `Match` (shell-style) and `PathMatch` (`/`-sensitive). Include/exclude globs against relative paths with directory components (e.g. `src/**/*.go`) want `PathMatch`. Planner should pin the exact API choice in Round 2 (either `Match` with rationale, or `PathMatch`).

## Unknowns

- Both observations are rendering-level nits, not evidence gaps. No unknowns on the proof side.
