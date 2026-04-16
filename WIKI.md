# Rak — Project Wiki

Living reference for the rak project. Captures the **current** best practices, architecture shape, and project state. Updated whenever a drop lands that changes best practice. Paired with `WIKI_CHANGELOG.md` (added when the first real drop lands) — entries ship as one-liners into the changelog.

## Update Discipline

- Update during drop closeout, alongside any `LEDGER.md` entry.
- Keep sections short and inspectable. If a section grows past ~30 lines, split or cut older guidance.
- History does NOT live here. This is a snapshot of current best practice.
- When a refinement lands that contradicts an entry, **update in place** — don't append a "2026-XX-XX update:" note.

## Current State (Drop 0 — Bootstrap)

Drop 0 is the **bootstrap drop**. It does not change any Go code; it establishes the project scaffolding and discipline baseline. Cascade dispatch does not exist yet — the orchestrator approximates it manually per `CLAUDE.md`.

What Drop 0 landed:

- Bare-root + `main/` worktree layout at `/Users/evanschultz/Documents/Code/hylla/rak/`.
- `CLAUDE.md` mirrored at bare root + `main/` (different preamble, same body).
- This `WIKI.md`.
- `README.md` scaffold (name origin, aspirational usage).
- MIT `LICENSE`, `.gitignore`.
- Initial commit pushed to `github.com/evanmschultz/rak`.

What Drop 0 did NOT land (on purpose — deferred to the first builder drop):

- `main.go`, `go.mod`, `go.sum` — carried over from the prior `fwc` prototype but not yet committed. Stashed at `/tmp/rak-stash/`.
- `magefile.go` — build automation.
- Any real feature work (directory walking, laslig-backed rendering, language detection, gitignore support, token counting, subcommands).

Tillsyn project creation is **pending** — temp planning happens in bare-root `PLAN.md` until the project exists in Tillsyn, at which point `PLAN.md` is deleted and planning lives as nested drops.

## Project Invariants

- **Tillsyn is the system of record** for all coordination once the project exists there. Bare-root `PLAN.md` is a transient scratchpad only.
- **The orchestrator never edits Go code** (including `magefile.go`). Every Go change goes through `go-builder-agent`.
- **Hylla is primary for committed Go code.** `git diff` covers post-ingest deltas. Context7 + `go doc` + gopls `LSP` cover external semantics.
- **Mage-only build discipline.** Never raw `go build` / `go test` / `go vet`. Always `mage <target>`. `mage ci` before every push (once mage is scaffolded).
- **QA before commit.** Both proof and falsification QA pass before any commit lands. No batched commits.
- **Hylla reingest is drop-end only.** Once per drop, full enrichment from remote, only after CI green. Subagents never call `hylla_ingest`.

## Drops Model

Tillsyn is moving to a single plan-item kind called **drops**. Rak uses this model from day one: `project → drops → drops → ... → atomic drop`. A drop decomposes into child drops when the work is not yet atomic, and each drop carries its own role binding (planning / build / qa). Until the Tillsyn rename ships, rak's drops are created using whatever kinds the project is bound to — the orchestrator treats them as drops semantically.

## Pre-Cascade Workflow (Orchestrator-as-Hub)

Until Tillsyn's cascade dispatcher lands upstream, the parent Claude Code session IS the orchestrator. It plans, routes, delegates, cleans up — never edits Go code.

1. **Plan** — orchestrator (or a `go-planning-agent` subagent for bigger drops) decomposes a parent drop into children with paths / packages / acceptance criteria.
2. **Build** — `go-builder-agent` subagent implements the increment. Auth + lease + Tillsyn credentials in the spawn prompt; durable drop content in the description.
3. **QA** — `go-qa-proof-agent` + `go-qa-falsification-agent` run in parallel, each with fresh context. Both must pass.
4. **Commit** — orchestrator + dev commit with conventional-commit format (pre-cascade).
5. **Push + CI green** — `git push` then `gh run watch --exit-status` until green.
6. **Update Tillsyn** — metadata, completion notes, terminal state.
7. **Next drop** — no per-drop Hylla reingest. Reingest is closeout-only.

## Related Files

- `main/CLAUDE.md` — canonical project rules (bare-root + main/ carry the same body).
- Bare-root `PLAN.md` — **transient** orchestration scratchpad, deleted when the Tillsyn project lands.
- `main/README.md` — user-facing docs.
- Future, added as real drops close: `main/LEDGER.md`, `main/HYLLA_FEEDBACK.md`, `main/WIKI_CHANGELOG.md`, `main/magefile.go`.
