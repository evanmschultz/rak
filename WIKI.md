# Rak — Project Wiki

Living **best-usage-practices guide** for rak. Captures **how to run rak's coordination right now**. Updated at the end of every drop so guidance stays aligned with the code and the lessons learned during the work.

Coordination model in one line: PLAN.md is the overarching drop tree, each drop is a directory under `main/drops/`, the per-drop lifecycle (planner → plan-QA → discuss → builder → build-QA → verify → closeout) lives in `main/drops/WORKFLOW.md`. Rak does **not** use Tillsyn.

## Update Discipline

- **Read this file at session start and after every compaction.** `CLAUDE.md` is auto-loaded; this wiki is **not** — read it deliberately before substantive orchestration. Read `main/PLAN.md` and `main/drops/WORKFLOW.md` on the same cold-start pass.
- **Update at the end of every drop**, inside the drop's `CLOSEOUT.md` step. If lessons from the drop change a best practice, rewrite the affected section **in place** — don't append `2026-04-XX update:` notes. Full audit trail lives in `REFINEMENTS.md` + `HYLLA_REFINEMENTS.md` + git history (those files land when the first finding does).
- Keep sections short and inspectable. If a section grows past ~30 lines, split it or cut older guidance that's no longer load-bearing.
- One-liner mirror per drop goes into `WIKI_CHANGELOG.md` (created when the first wiki-changing drop lands).

## Current State (Drop 0 — Bootstrap, Out of Band)

Drop 0 is the bootstrap drop, done in-conversation before the per-drop directory workflow existed. It landed:

- Bare-root + `main/` worktree layout at `/Users/evanschultz/Documents/Code/hylla/rak/` (GitHub repo `evanmschultz/rak`, private).
- `CLAUDE.md` mirrored at bare root + `main/` (same rules body, different preambles).
- This `WIKI.md`, `README.md`, MIT `LICENSE`, `.gitignore`.
- Initial commit pushed to `origin/main`.

What Drop 0 did **not** land (intentionally deferred to the first builder drop):

- `main.go`, `go.mod`, `go.sum` — carried over from the prior `fwc` prototype, stashed at `/tmp/rak-stash/`.
- `magefile.go` — build automation.
- `.github/workflows/ci.yml` — CI.
- Any feature code (directory walking, laslig-backed rendering, language detection, gitignore support, token counting, subcommands).

Drop 0 has no per-drop directory under `main/drops/` because it predates the per-drop-dir workflow. Its audit trail is git history + this section.

## Coordination Model — System of Record

**Three documents own the model. They do not duplicate each other:**

1. **`main/PLAN.md`** — overarching drop tree (10 level_1 containers + state + `blocked_by` + per-drop dir link). Decisions, intent, scope. Updated *after* a drop closes or *after* a planner restructures the tree. Not edited mid-build.
2. **`main/drops/WORKFLOW.md`** — canonical per-drop lifecycle (Phases 1–7), drop directory shape, file lifecycles, Agent Spawn Contract, restart recovery.
3. **`main/CLAUDE.md`** — orchestrator role boundaries, agent bindings, evidence sources, Go quality rules, mage discipline, commit format, safety.

**Per-drop work artifacts** live under `main/drops/DROP_N_<NAME>/`. The dir is stamped from `main/drops/_TEMPLATE/` at Phase 1 start and persists through closeout.

**No alternate trackers.** Do NOT use Claude Code's built-in `TaskCreate` / `TaskUpdate` / `TaskList` / `TaskGet` / `TaskStop` / `TaskOutput` — they evaporate on compaction or restart. Decompose finer procedural granularity into atomic units inside the active drop's `PLAN.md`.

**No worklogs outside `main/drops/`.** Per-drop dirs are the worklog substrate. No sticky notes. No "I'll track this in chat".

## Drop Decomposition

A drop is "atomic-decomposable" when the planner can break its container scope into atomic **units** inside the drop's `PLAN.md`. A unit is "atomic" when:

- One builder subagent can finish it in one working session.
- Its acceptance criteria are concrete and verifiable — a QA subagent can make a yes/no call.
- It has a clear `paths` / `packages` footprint so file- and package-level reasoning works.

If a unit is too large, **add more units** rather than stretching one.

### Ordering: `blocked_by`, Not `depends_on`

Two primitives for "this comes after that":

1. **Parent-child nesting** — a drop cannot close while any of its units is incomplete. Implicit `depends_on`. Don't layer a separate field on top.
2. **`blocked_by`** — sibling and cross-unit ordering primitive inside the drop's `PLAN.md`. Planner sets it during Phase 1 (decomposition) and Phase 3 (revision).

**Rule of thumb:** if X should finish before Y and they're **siblings**, use `blocked_by` on the unit row. If Y's completion genuinely depends on X's *result*, fold X into Y's preconditions and make them one unit instead of two.

Avoid `depends_on` entirely.

## QA Discipline — Every Build Unit Gets QA

**No build unit is `done` without per-unit QA passing.** This is a gate.

Every build unit gets two QA passes after the builder finishes:

1. **`go-qa-proof-agent`** — evidence completeness, reasoning coherence, trace coverage. Asks: *"does the evidence support the claim?"*
2. **`go-qa-falsification-agent`** — counterexamples, alternate traces, hidden dependencies, contract mismatches, YAGNI pressure. Asks: *"can I construct a case where this is wrong?"*

Both run in parallel after the builder completes. **Both must pass.** Findings append to durable `BUILDER_QA_PROOF.md` + `BUILDER_QA_FALSIFICATION.md` files in the drop's dir; rounds use `## Unit N.M — Round K` headings.

Plan-QA (Phase 2 of WORKFLOW.md) runs the same proof + falsification pair, but writes to **transient** `PLAN_QA_PROOF.md` / `PLAN_QA_FALSIFICATION.md` files that orch `git rm`s between rounds — audit lives in `git log -- main/drops/DROP_N_<NAME>/PLAN_QA_*.md`.

For the per-phase mechanics — file lifecycle, round conventions, cleanup cadence — see `main/drops/WORKFLOW.md`.

## Build-QA-Commit Loop

Per-drop lifecycle is canonical in `main/drops/WORKFLOW.md` (Phases 1–7). Headlines:

- **Code is NEVER committed or pushed without per-unit QA passing first.** No batched commits. No deferred pushes. No skipped QA. No skipped CI watch.
- `git add <paths>` — **never** `git add .`.
- Hylla reingest is **drop-end only** — once per drop, inside the closeout (Phase 7), full enrichment from remote, only after CI green. Subagents never call `hylla_ingest`.

## Orchestrator Role Boundaries

- **Orchestrator** (parent Claude Code session launched from `main/`) — plans, routes, delegates, cleans up. **Never edits Go code or `magefile.go`.** May edit markdown (this wiki, `CLAUDE.md`, `PLAN.md`, drop dir mds, `README.md`, `LEDGER.md`, agent `.md` files, refinement files).
- **Builder subagent** (`go-builder-agent`) — the ONLY role that edits Go code. Spawned via the `Agent` tool with the spawn contract preamble from WORKFLOW.md § "Agent Spawn Contract".
- **QA subagents** (`go-qa-proof-agent`, `go-qa-falsification-agent`) — gated to QA roles. Read, verify, write to their own `*_QA_*.md` file, return verdict, die. Never edit code.
- **Planner subagent** (`go-planning-agent`) — fills the drop's `PLAN.md` Planner section (Phase 1) and revises it across plan-QA rounds (Phase 3). Never edits code.
- **Dev / human** — approves design calls during plan-QA discussion (Phase 3), reviews build-QA findings (Phase 5).

## Drop-End Closeout Checklist

Every drop's final phase (Phase 7 of WORKFLOW.md) writes `CLOSEOUT.md` covering:

1. All units `done`. `git status --porcelain` clean.
2. All commits on remote. CI green (`gh run watch --exit-status`).
3. Aggregate per-subagent `## Hylla Feedback` subsections from `BUILDER_WORKLOG.md` → append to `main/HYLLA_FEEDBACK.md` (created when first such section lands).
4. Aggregate usage findings → append to `main/REFINEMENTS.md` / `main/HYLLA_REFINEMENTS.md` (created on first entry).
5. `hylla_ingest` — full enrichment, from remote, after CI green.
6. Append entry to `main/LEDGER.md` (created when the first drop closes).
7. Append one-liner to `main/WIKI_CHANGELOG.md`.
8. Update relevant section(s) of this wiki **in place** if anything shipped that changed best practice.
9. Flip the drop's container row in `main/PLAN.md` to `state: done` and the drop dir's `PLAN.md` header to `state: done` in the same commit.

## Related Files

- `CLAUDE.md` — canonical project rules (bare-root + `main/` carry the same rules body).
- `main/PLAN.md` — overarching drop tree (state of the world for the project).
- `main/drops/WORKFLOW.md` — per-drop lifecycle (state of the world for one drop).
- `main/drops/_TEMPLATE/` — stamped into a new drop dir at Phase 1 start.
- `main/drops/DROP_N_<NAME>/` — per-drop work artifacts (`PLAN.md`, transient `PLAN_QA_*.md`, durable `BUILDER_WORKLOG.md`, `BUILDER_QA_*.md`, `CLOSEOUT.md`).
- `README.md` — user-facing docs.
- Future, added as real drops close: `LEDGER.md`, `WIKI_CHANGELOG.md`, `HYLLA_FEEDBACK.md`, `REFINEMENTS.md`, `HYLLA_REFINEMENTS.md`, `magefile.go`.
