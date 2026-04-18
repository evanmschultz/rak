# Main Work-Orchestrator Kickoff Prompt

Paste the fenced block below as the **first message** of a new Claude Code session launched from `/Users/evanschultz/Documents/Code/hylla/rak/main`.

Run order:

```
cd /Users/evanschultz/Documents/Code/hylla/rak/main
claude
# (paste the block below as the first message)
```

This file is a one-shot kickoff scaffold. It is **untracked by default** — leave it ignored, delete it after use, or commit it if you want a permanent record. It will go stale the moment Drop 1's planner closes (because the next thing to do shifts from "spawn DROP_1 planner" to "spawn DROP_1 unit 1.1 builder").

---

```
Cold-start a rak work orchestrator (not a steward — pwd is main/, not the bare-root one level up).

Repo layout note: rak uses a flat bare repo at /Users/evanschultz/Documents/Code/hylla/rak/. Bare-repo internals (HEAD, config, objects, refs, worktrees, hooks, info, logs) live directly at that top level — there is no .bare/ wrapper and no top-level .git pointer. main/.git is a pointer file reading `gitdir: /Users/evanschultz/Documents/Code/hylla/rak/worktrees/main`. `git worktree list` from main/ should show two lines: the bare root marked `(bare)` and main/ at the latest commit.

Coordination model — IMPORTANT: rak does NOT use Tillsyn. Coordination lives in three docs:
- main/PLAN.md — overarching drop tree (10 level_1 containers + state + blocked_by + per-drop dir link).
- main/drops/WORKFLOW.md — canonical per-drop lifecycle (Phases 1–7), drop dir shape, file lifecycles, Agent Spawn Contract.
- main/CLAUDE.md — orchestrator role boundaries, agent bindings, Go quality rules.

Per-drop work artifacts live under main/drops/DROP_N_<NAME>/, stamped from main/drops/_TEMPLATE/ at Phase 1 start. No till_* calls, no capability leases, no capture_state, no attention items, no handoffs.

Pre-work checklist:
1. Read main/CLAUDE.md (auto-loads), main/WIKI.md, main/PLAN.md, and main/drops/WORKFLOW.md in that order. CLAUDE.md auto-loads but the others do not — read them deliberately per the "read WIKI.md after every compaction" rule.
2. Confirm `pwd` is /Users/evanschultz/Documents/Code/hylla/rak/main and `git worktree list` shows the expected two-line output.
3. No auth steps. No session claims. Just: read the docs, then proceed.

First (and only currently eligible) drop is DROP_1_CODE_SCAFFOLD_MAGE_CI:
- Container row in main/PLAN.md: state=todo, blocked_by=—, drop dir=main/drops/DROP_1_CODE_SCAFFOLD_MAGE_CI/ (does not yet exist).
- Why eligible: DROP_0 is done, DROP_1 has no blocked_by.

Run Phase 1 of main/drops/WORKFLOW.md against Drop 1:
1. Stamp the drop dir: copy main/drops/_TEMPLATE/ → main/drops/DROP_1_CODE_SCAFFOLD_MAGE_CI/. Set its PLAN.md header `state: planning`. Commit (`docs(drop-1): scaffold drop dir from template`).
2. Spawn go-planning-agent with the spawn preamble from main/drops/WORKFLOW.md § "Agent Spawn Contract" + the planner appendix from § "Per-Role Spawn Appendices". Pass: drop's PLAN.md path, the DROP_1 container row excerpt from main/PLAN.md, scope sentence from dev (you confirm the level_1 goal in one sentence and flag any ambiguity before spawning).
3. Planner fills the drop's PLAN.md `## Planner` section with the six expected units (1.1–1.6 sketched in main/PLAN.md § "Drop Tree" → "Expected Decomposition"), each with paths / packages / acceptance / blocked_by. Returns control.
4. Commit the planner output (`docs(drop-1): planner decompose into six units`). Move to Phase 2 (plan-QA) per WORKFLOW.md.
5. Loop through Phase 2 → Phase 3 (discuss + cleanup) until plan accepted, then proceed unit-by-unit through Phases 4–7.

Orchestrator discipline (from main/CLAUDE.md):
- Never edit Go source (none exists yet anyway). Never run mage targets (none exist yet — they land in Drop 1.4–1.5). Markdown edits (PLAN.md, WIKI.md, drop dir mds) are fine.
- All code-producing work starting with unit 1.1 goes through go-builder-agent subagents spawned with the spawn contract preamble in their prompt.
- No TaskCreate / TaskUpdate / TaskList — they evaporate on compaction. Per-unit state lives in the drop dir's PLAN.md Planner section.

Start by reading the four files in step 1, then come back to me for the scope-confirmation conversation in Phase 1 step 2 before stamping the drop dir or spawning the planner.
```

---

## Stable References (for the kickoff above)

- **Repo:** `github.com/evanmschultz/rak` (private until flipped in Drop 9.5).
- **Module:** `github.com/evanmschultz/rak`.
- **Hylla artifact ref:** `github.com/evanmschultz/rak@main`.
- **First drop:** `DROP_1_CODE_SCAFFOLD_MAGE_CI` (drop dir not yet stamped).
