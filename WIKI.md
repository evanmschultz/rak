# Rak — Project Wiki

Living **best-usage-practices guide** for rak as a Tillsyn external adopter. Captures **how to run rak's coordination right now**, given the project is pre-Tillsyn-project and still pre-cascade in Tillsyn itself. Updated at the end of every rak drop so guidance stays aligned with the code and the lessons learned during the work.

Rak is an **external adopter** of Tillsyn — it is not the Tillsyn repo. That framing changes one thing: every drop-end produces a cross-project improvement prompt routed back to the Tillsyn team (see § "Drop-End Closeout Checklist" item 5). Everything else mirrors the Tillsyn wiki's own best-practice guidance.

## Update Discipline

- **Read this file at session start and after every compaction.** `CLAUDE.md` is auto-loaded; this wiki is **not** — read it deliberately before substantive orchestration.
- **Update at the end of every drop**, inside the drop-end closeout task. If lessons from the drop change a best practice, rewrite the affected section **in place** — don't append `2026-04-XX update:` notes. Full audit trail lives in `REFINEMENTS.md` + `HYLLA_REFINEMENTS.md` + git history (those files land when the first finding does).
- Keep sections short and inspectable. If a section grows past ~30 lines, split it or cut older guidance that's no longer load-bearing.
- One-liner mirror per drop goes into `WIKI_CHANGELOG.md` (created when the first wiki-changing drop lands).

## Current State (Drop 0 — Bootstrap, Out of Band)

Drop 0 is the bootstrap drop, done in-conversation before the Tillsyn project existed. It landed:

- Bare-root + `main/` worktree layout at `/Users/evanschultz/Documents/Code/hylla/rak/` (GitHub repo `evanmschultz/rak`, private).
- `CLAUDE.md` mirrored at bare root + `main/` (same rules body, different preambles).
- This `WIKI.md`, `README.md`, MIT `LICENSE`, `.gitignore`.
- Initial commit pushed to `origin/main`.

What Drop 0 did **not** land (intentionally deferred to the first builder drop):

- `main.go`, `go.mod`, `go.sum` — carried over from the prior `fwc` prototype, stashed at `/tmp/rak-stash/`.
- `magefile.go` — build automation.
- `.github/workflows/ci.yml` — CI.
- Any feature code (directory walking, laslig-backed rendering, language detection, gitignore support, token counting, subcommands).

The Tillsyn project for rak does **not yet exist**. Planning lives in `main/PLAN.md` — a **persisted** backup mirror of the Tillsyn project tree, kept in-repo because Tillsyn is still under development and a local snapshot is cheap insurance. Once the project exists, every drop-mutating Tillsyn action (create, reword, close, add child, change `blocked_by`) updates `PLAN.md` in the same commit; if the two drift, Tillsyn is authoritative and `PLAN.md` is reconciled to match.

## The Tillsyn Model (Node Types)

Tillsyn has exactly **two node types** rak uses today:

1. **Project** — the root container. Rak has one (pending creation). Never nested inside another project.
2. **Drop** — every node below the project. Drops nest **infinitely**.

A "drop" is the Tillsyn-native word for a unit of work. In the current runtime it is a plan item with `kind='task'` (the pre-Drop-2 creation rule — see `CLAUDE.md` § "Drops — The Only Plan-Item Kind"). Tillsyn's Drop 2 SQL collapses every non-project node to literal `kind='drop'`. For now, write `kind='task', scope='task'` and **refer to the node as a drop in prose**.

**The term "slice" is not used.** It was prior internal vocabulary that has been retired — if you see it in any doc, that doc is stale.

### Do Not Use Other Kinds

The `kind_catalog` still lists `build-task`, `subtask`, `qa-check`, `plan-task`, `commit-and-reingest`, `a11y-check`, `visual-qa`, `design-review`, `phase`, `branch`, `decision`, `note`. **Do not use any of them.** Stick to plain `task` so rak's writes stay consistent with the upcoming rewrite.

### Template-Free Project

Do **not** bind a template to the rak project. Tillsyn itself is `template: none` and explicitly instructs external adopters to skip template binding until `child_rules` return in Tillsyn's Drop 3+. Until then, **the orchestrator enforces tree shape manually** per this wiki.

## Level Addressing (0-Indexed)

Levels name depth from the project root down. **The project is level 0.** The first drop under the project is level 1. 0-indexed on purpose — the whole Tillsyn DB zero-indexes everything, so levels do too. Use this language consistently:

- `project` — the root, **level 0**. Not a drop.
- `level_1` — every drop that sits directly under the rak project (first-child drops).
- `level_2` — drops one level below a level_1 drop.
- `level_N` — N steps deep from the project root.

Dotted addresses (`0.1.5.2`, `rak-0.1.5.2`) are **read-only shorthand**. **Mutations always take UUIDs**, never dotted addresses.

## Tillsyn Is the System of Record

**Every action lives in Tillsyn.** Non-negotiable.

- Every piece of work gets a Tillsyn drop **before it starts**. Not retroactive.
- When work starts on a drop, move it to `in_progress` **immediately**. No `todo` items left while someone is working on them.
- **Do not use Claude Code's built-in `TaskCreate` / `TaskUpdate` / `TaskList` / `TaskGet` / `TaskStop` / `TaskOutput`.** They are in-session-only and evaporate on compaction or restart. Decompose finer procedural granularity into **child Tillsyn drops** instead of a parallel in-session tracker.
- No markdown worklogs. No sticky notes. No "I'll track this in chat".
- If it's not in Tillsyn, it didn't happen.

## Drop Decomposition Rules

### Every Level-1 Drop Opens With A Planning Drop + Dev Discussion

The first child of every **level-1 drop** is a **planning drop** (`Role: planner`). Its job is a dev ↔ orchestrator discussion that:

1. Confirms the level-1 scope is well-understood.
2. Decomposes the level-1 drop into **atomic nested drops** (work units a single builder subagent can finish cleanly).
3. Sets `blocked_by` across siblings where ordering matters.
4. Files any cross-cutting discussions as their own drops.

**Until the planning drop is `done`, no build drop under the level-1 drop is eligible to start.**

Nested drops (level_2 and deeper) do **not** universally require their own planning drop — but if a nested drop is itself ambiguous or large enough to need decomposition, add a planning drop under it too.

### Atomic Drop Granularity

A drop is "atomic" when:

- One builder subagent can finish it in one working session.
- Its acceptance criteria are concrete and verifiable — a QA subagent can make a yes/no call.
- It has a clear `paths` / `packages` footprint so file- and package-level blocking works.

If a drop is too large, **nest further** rather than stretching the drop.

### Ordering: `blocked_by`, Not `depends_on`

Two primitives for "this comes after that":

1. **Parent-child nesting** — a parent drop cannot move to `done` while any child is incomplete. This is the implicit `depends_on`. Do not layer a separate `depends_on` field on top.
2. **`blocked_by`** — the **only** sibling and cross-subtree ordering primitive. Planners set `blocked_by` at creation time.

**Rule of thumb:** if X should finish before Y and they're **siblings** (or in different subtrees), use `blocked_by`. If X should finish before Y and Y's completion genuinely depends on X's result, **make Y a child of X** instead of siblings-with-blocked_by, so the parent-child rule does the work.

Avoid `depends_on` entirely.

## QA Discipline — Every Build Drop Gets QA

**No build drop is `done` without QA passing.** This is a gate.

Every build drop (any drop whose role is `builder`) has **two QA children**:

1. **`Role: qa-proof`** — evidence completeness, reasoning coherence, trace coverage. Asks: *"does the evidence support the claim?"*
2. **`Role: qa-falsification`** — counterexamples, alternate traces, hidden dependencies, contract mismatches, YAGNI pressure. Asks: *"can I construct a case where this is wrong?"*

Both run in parallel after the build drop completes (`blocked_by: <build drop>`). **Both must pass.** If either finds issues, the build drop stays `in_progress`, the finding is recorded on the failed QA drop, a fix drop runs, and QA re-runs.

## Build-QA-Commit Loop (Pre-Cascade)

Until Tillsyn's cascade dispatcher ships, the parent orchestrator session runs this loop manually:

1. **Plan** — `go-planning-agent` (or orchestrator + dev, for trivial drops) decomposes into atomic drops with `paths` / `packages` / acceptance criteria.
2. **Build** — `go-builder-agent` subagent implements the increment. Builder moves its own drop to `in_progress` at start, commits evidence to `implementation_notes_agent` + `completion_notes`, moves to `done` at end, closes with a `## Hylla Feedback` section.
3. **QA proof + QA falsification** — parallel subagent spawn, each with fresh context. Each moves its own QA drop to `in_progress` at start, `done` on pass, or leaves `in_progress` + posts findings on fail.
4. **Fix** — if either QA fails, respawn the builder, re-run QA.
5. **Commit** — after both QA pass, orchestrator + dev commit with conventional-commit format. `git add <paths>` — **never** `git add .`.
6. **Push + CI green** — `git push` then `gh run watch --exit-status` until green.
7. **Update Tillsyn** — checklist + metadata + terminal state.

**No batched commits. No deferred pushes. No skipped QA. No skipped CI watch.**

Hylla reingest is **drop-end only** — once per drop, inside the closeout task, full enrichment from remote, only after CI green. Subagents never call `hylla_ingest`.

## Orchestrator Role Boundaries

- **Orchestrator** (parent Claude Code session launched from `main/`) — plans, routes, delegates, cleans up. **Never edits Go code or `magefile.go`.** May edit markdown (this wiki, `CLAUDE.md`, `README.md`, agent `.md` files, refinement files).
- **Builder subagent** (`go-builder-agent`) — the ONLY role that edits Go code. Spawned via the `Agent` tool with Tillsyn auth credentials in the prompt.
- **QA subagents** (`go-qa-proof-agent`, `go-qa-falsification-agent`) — gated to QA roles. Read, verify, verdict, die. Never edit code.
- **Planner subagent** (`go-planning-agent`) — decomposes a level-1 drop into atomic nested drops. Never edits code.
- **Dev / human** — approves auth, reviews results, makes design calls that the orchestrator files as discussion drops.

## Drop-End Closeout Checklist

Every level-1 drop's final child is `DROP <N> END — CLOSEOUT` (`Role: commit`, `blocked_by` every other drop in the level-1 subtree):

1. All sibling drops `done`. `git status --porcelain` clean.
2. All commits on remote. CI green (`gh run watch --exit-status`).
3. Aggregate per-subagent `## Hylla Feedback` sections into `HYLLA_FEEDBACK.md`.
4. Aggregate usage findings (ergonomic wins / pain, bugs, lessons) into `REFINEMENTS.md` / `HYLLA_REFINEMENTS.md`.
5. **Cross-project improvement prompt for Tillsyn** — rak is an external adopter, so every drop-end writes a prompt back to the Tillsyn team capturing: context (rak is a Go CLI, solo dev), friction (schema confusion, missing primitives, MCP ergonomics), workarounds, ranked requests, and evidence (drop/comment/handoff IDs). Routed via issue / PR / `till.handoff` once the Tillsyn-team identity exists.
6. `hylla_ingest` — full enrichment, from remote, after CI green.
7. Append entry to `LEDGER.md` (created when the first drop closes).
8. Append one-liner to `WIKI_CHANGELOG.md`.
9. Update the relevant section(s) of this wiki if anything shipped that changed best practice — **in place**.

## Related Files

- `CLAUDE.md` — canonical project rules (bare-root + `main/` carry the same rules body).
- `main/PLAN.md` — **persisted** backup mirror of the Tillsyn project tree. Updated in the same commit as every drop-mutating Tillsyn action. Tillsyn is authoritative on drift.
- `README.md` — user-facing docs.
- Future, added as real drops close: `LEDGER.md`, `WIKI_CHANGELOG.md`, `HYLLA_FEEDBACK.md`, `REFINEMENTS.md`, `HYLLA_REFINEMENTS.md`, `magefile.go`.
