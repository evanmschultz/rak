# Rak — Project CLAUDE.md (main worktree)

This file lives in the **`main/` worktree** at `/Users/evanschultz/Documents/Code/hylla/rak/main/`. This is the primary work checkout — all real coding, building, testing, and committing happens here. **The dev launches orchestrators from this directory.** The bare-root `CLAUDE.md` (one directory up) carries the same rules body; only the preamble differs.

## Tillsyn Is the System of Record

All work is tracked in Tillsyn. No exceptions.

- No markdown files for work tracking, coordination, worklogs, or execution state.
- **Tillsyn = durable truth.** Every piece of work gets a Tillsyn plan item (a **drop**) before it starts.
- **Use Tillsyn exclusively for work tracking.** Do NOT use Claude Code's built-in `TaskCreate` / `TaskUpdate` / `TaskList` / `TaskGet` / `TaskStop` / `TaskOutput` — they evaporate on compaction/restart. Decompose into child drops instead.
- **When work starts on a drop, move it to `in_progress` immediately.**
- **Read `main/WIKI.md` at session start and after every compaction.** The wiki is the living best-practice snapshot and changes as the project evolves. CLAUDE.md is auto-loaded; WIKI.md is NOT — Read it deliberately on the first turn after cold-start or compaction.
- **Bare-root `PLAN.md` is a transient orchestrator scratchpad.** It exists only until the Tillsyn project is created; once the project exists, `PLAN.md` is deleted and all planning lives in Tillsyn drops.

### Drops — Single Plan-Item Kind

Tillsyn is moving to a single plan-item node type called **drops** (replacing the old `task` / `subtask` / `slice` kinds). Rak uses the drops model from day one: `project → drops → drops → ... → atomic drop`. Each drop carries its own role binding (planning / build / qa) and decomposes into child drops when the work is not yet atomic.

Until the rename lands in Tillsyn, rak's drops are created using whatever kinds the project is bound to at creation time; the orchestrator treats them as drops semantically.

## Orchestrator-as-Hub

The parent Claude Code session launched by the dev from this directory is always **the orchestrator**. Every other role (builder, qa-proof, qa-falsification, planning, research) is a subagent spawned via the `Agent` tool.

**CRITICAL: The orchestrator NEVER writes Go code.** The parent session must not use `Edit`, `Write`, or any other tool to modify `.go` source, test, or `magefile.go` files. Every code change — every single one — goes through a `go-builder-agent` subagent. Orchestrator reads code for planning/research; edits markdown only (this file, `WIKI.md`, future `LEDGER.md`, `README.md`, agent `.md` files).

### Agent Bindings

| Role | Agent | Edits Go? |
|---|---|---|
| Builder | `go-builder-agent` | **Yes** (only role that does) |
| QA Proof | `go-qa-proof-agent` | No |
| QA Falsification | `go-qa-falsification-agent` | No |
| Planning | `go-planning-agent` | No |
| Research | Claude's built-in `Explore` subagent | No |

Orchestrator dispatches via the `Agent` tool with Tillsyn auth credentials in the spawn prompt. Split of concerns: spawn prompt carries ephemeral/spawn-unique fields (task_id, auth creds, working dir, move-state directive); the drop description carries durable task content (acceptance criteria, paths, packages, mage targets, Hylla artifact ref, cross-refs).

## Build-QA-Commit Discipline

**Code is NEVER committed or pushed without QA completing first.**

1. **Build** — `go-builder-agent` implements the increment.
2. **QA Proof** — `go-qa-proof-agent` verifies evidence completeness.
3. **QA Falsification** — `go-qa-falsification-agent` tries to break the conclusion.
4. **Fix** — if QA finds issues, spawn another builder, re-run QA.
5. **Commit** — only after both QA passes clear. Conventional-commit format.
6. **Push** — `git push` so CI runs (once CI exists).
7. **CI green** — `gh run watch --exit-status` until green. If CI fails, fix before continuing.
8. **Update Tillsyn** — drop metadata, completion notes, terminal state.

No batched commits. No deferred pushes. No skipped QA. No skipped CI watch. No claiming done in chat without Tillsyn reflecting it.

## Hylla Baseline

- **Artifact ref**: `github.com/evanmschultz/rak@main` — Hylla resolves `@main` to the latest ingest.
- **Also on project metadata** (`metadata.hylla_artifact_ref`) once the Tillsyn project exists, so planners can read it programmatically.
- **Hylla ingest is drop-end only**, not per-task. Only the orchestrator calls `hylla_ingest`. Always `enrichment_mode=full_enrichment`, always from the GitHub remote, never before `git push` + `gh run watch --exit-status` green. Subagents never call `hylla_ingest`.

### Code Understanding Rules

1. **All Go code**: use Hylla MCP first (`hylla_search`, `hylla_node_full`, `hylla_search_keyword`, `hylla_refs_find`, `hylla_graph_nav`). Exhaust every Hylla search mode — vector, keyword, graph-nav, refs — before falling back to `LSP`, `Read`, `Grep`, `Glob`. **Whenever a Hylla miss forces a fallback, the subagent records the miss in its closing comment** under a `## Hylla Feedback` heading.
2. **Changed since last ingest**: use `git diff`. Hylla is stale for those files until reingest.
3. **Non-Go code** (markdown, TOML, YAML, magefile, SQL): use `Read`, `Grep`, `Glob`, `Bash` directly.
4. **External semantics**: Context7 + `go doc` + `LSP` for library and language questions the repo can't answer itself.
5. **`LSP` tool** (gopls-backed): symbol search, references, diagnostics, rename safety for live / uncommitted code. Auto-targets the active checkout (`main/`).
6. **Laslig note**: `github.com/evanmschultz/laslig@main` is not yet in Context7. Use Hylla (`hylla_search` with `artifact_ref=github.com/evanmschultz/laslig@main`) or `go doc github.com/evanmschultz/laslig` as the primary laslig evidence sources.

## Evidence Sources

In order:

1. **Hylla** — committed repo-local Go code.
2. **`git diff`** — uncommitted local deltas / files changed since last ingest.
3. **Context7 + `go doc` + gopls `LSP`** — external / language / tooling semantics.

## Semi-Formal Reasoning

For semantic, high-risk, or ambiguous work:

- **Premises** — what must be true.
- **Evidence** — grounded in Hylla / `git diff` / Context7 / `go doc` / gopls.
- **Trace or cases** — concrete paths through the code.
- **Conclusion** — the claim.
- **Unknowns** — what remains uncertain, routed into Tillsyn as a comment, handoff, or attention item.

Short and inspectable.

## QA Discipline

Two asymmetric passes, not duplicates:

- **QA Proof** (`go-qa-proof-agent`, `/qa-proof`) — evidence completeness, reasoning coherence, trace coverage.
- **QA Falsification** (`go-qa-falsification-agent`, `/qa-falsification`) — counterexamples, hidden dependencies, contract mismatches, YAGNI pressure.

Run both for every build-drop. Spawn as parallel subagents for fresh context.

## Project Structure (Target)

Not yet scaffolded — lands in the first post-bootstrap drop. Target layout:

- `cmd/rak/` — CLI entrypoint (or root `main.go` during early scaffolding).
- `internal/counting/` — pure counting logic (`io.Reader`-driven): bytes, lines, words, chars, blank/comment/code split.
- `internal/fileset/` — directory walking, gitignore-aware filtering, depth limiting.
- `internal/lang/` — language detection (extension + shebang + content sniff).
- `internal/ignore/` — glob includes/excludes, gitignore parsing, `--tracked-only` (backed by `git ls-files`).
- `internal/render/` — laslig-backed output (TTY pretty, JSON, TSV, tree view).
- `internal/summary/` — per-directory / per-type aggregation, totals, sorting.
- `internal/tokens/` — tiktoken-backed token estimates (behind `--tokens`).
- `magefile.go` — mage build automation.

## Tech Stack

- Go 1.26+
- `github.com/spf13/cobra` — command tree.
- `github.com/charmbracelet/fang` — help/version/error polish.
- `github.com/evanmschultz/laslig` — structured output, TTY-aware rendering, optional spinner/activity.
- `github.com/tiktoken-go/tokenizer` — token counting (to be added).
- `github.com/magefile/mage` — build automation (to be added in first builder drop).

## Build Verification

Before any build-drop is marked done:

1. All relevant mage targets pass (discover via `mage -l`).
2. **NEVER run raw `go test`, `go build`, `go run`, `go vet`** — always `mage <target>`. If a mage target has a bug, fix the target — don't bypass. No exceptions, orchestrator or subagent.
3. **NEVER run `mage install` from an agent.** This is a **dev-only** dogfood target that promotes a binary to `$GOBIN`. Orchestrator and every subagent must not invoke it. If a drop description asks for it, stop and return control to the orchestrator.
4. All QA drops (proof + falsification) for this build-drop have closed green.

Key targets (as they exist, post-first-drop): `mage build`, `mage test`, `mage format`, `mage lint`, `mage ci`, `mage install`. Run `mage ci` before every push. Coverage gate lands when real code does.

## Go Development Rules

- **Interface-first boundaries**, dependency inversion where warranted.
- **Smallest concrete design.** No abstraction for hypothetical future variation.
- **TDD-first** where practical. Ship small tested increments.
- **Idiomatic Go** — naming, package structure, import grouping (stdlib / third-party / local).
- **Go doc comments** on every top-level declaration and method, production and test.
- **Errors**: wrap with `%w`, bubble up at clean boundaries, don't swallow.
- **Tests**: `*_test.go` co-located, table-driven, behavior-oriented assertions. `-race` via mage.
- **Mage discipline**: plain `mage <target>` from the worktree root. No `GOCACHE=...` overrides. No workspace-local cache dirs.
- **After touching Go code**: `mage ci` before handoff. After pushing to validate CI: `gh run watch --exit-status` until green.
- **Dependencies**: ask the dev to run `go get` / module updates. No `GOPROXY=direct`, `GOSUMDB=off`, or checksum bypass.
- **Context7 / `go doc`**: before any unfamiliar external API usage, after any test failure.
- **Markdown-first authoring** for Tillsyn `description`, `summary`, `body_markdown`, thread comments.

## Skill and Slash Command Routing

| Command | When to Use |
|---|---|
| `/plan-from-hylla` | Hylla-grounded planning |
| `/qa-proof` | Proof-oriented QA |
| `/qa-falsification` | Falsification-oriented QA |
| `/select-checkout` | Confirm the active visible checkout |
| `/gopls-sync` | Verify gopls targets `main/` |
| `semi-formal-reasoning` | Explicit reasoning certificate |

## Git Commit Format

Conventional-commit: `type(scope): message`. All lowercase except proper nouns, acronyms (HTTP, CLI, JSON, TUI). Concise — describe what changed, not how.

Types: `feat`, `fix`, `refactor`, `chore`, `docs`, `test`, `ci`, `style`, `perf`.

Examples:
- `feat(count): add blank/comment/code line split per file`
- `fix(fileset): respect .gitignore in nested directories`
- `chore(deps): add tiktoken-go/tokenizer`

No co-authored-by trailers. No period at end. No capitalized first word after the colon unless proper noun/acronym.

## Safety

- Never delete files or directories without explicit dev approval.
- Never run commands outside the repo root `/Users/evanschultz/Documents/Code/hylla/rak`.
- Never push to any remote without explicit request.
- Keep secrets out of committed config files.

## Bare-Root and Worktree Discipline

- The bare repo at `/Users/evanschultz/Documents/Code/hylla/rak` (one level up) is the orchestration root — **not** a coding checkout.
- This directory (`main/`) is the primary work checkout. Real coding / building / testing / committing happens here.
- Always confirm `pwd` is this checkout before edits, tests, commits, or gopls work.
- **Dev launches orchestrators from here** — this is the canonical orchestrator working directory.
- If checkout context is unclear, use `/select-checkout`.
- Rak uses a single visible checkout (`main/`). Additional worktrees and gopls-sync only matter if a multi-lane setup is introduced later.

## Recovery After Session Restart

1. `till.capture_state` — re-anchor project and scope.
2. `till.attention_item(operation=list, all_scopes=true)` — inbox.
3. `till.handoff(operation=list)` — open routing.
4. Check `in_progress` drops for staleness.
5. Revoke orphaned auth sessions / leases.
