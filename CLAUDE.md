# Rak — Project CLAUDE.md (main worktree)

This file lives in the **`main/` worktree** at `/Users/evanschultz/Documents/Code/hylla/rak/main/`. This is the primary work checkout — all real coding, building, testing, and committing happens here. **The dev launches work orchestrators from this directory.** Sessions launched from the bare-root one directory up are **steward orchestrators** with a different prompt (bare-root `CLAUDE.md`) and a different scope — cross-worktree oversight and merge-conflict help, not feature work.

## Coordination Model — At a Glance

Rak does **not** use Tillsyn. Three documents own the coordination model; they do not duplicate each other:

- **`main/PLAN.md`** — overarching drop tree (10 level_1 container drops + state + `blocked_by` + per-drop dir link). Updated *after* a drop closes or *after* a planner restructures the tree. Not edited mid-build.
- **`main/drops/WORKFLOW.md`** — canonical per-drop lifecycle (planner → plan-QA → discuss → revise → builder → build-QA → verify → closeout). Owns: drop directory shape, file lifecycles, phase order, the **Agent Spawn Contract** (preamble pasted into every subagent spawn), restart recovery.
- **`main/CLAUDE.md`** (this file) — orchestrator role boundaries, agent bindings, evidence sources, Go quality rules, mage discipline, commit format, safety. Does not own per-phase mechanics — those live in WORKFLOW.md.

Per-drop work artifacts live under `main/drops/DROP_N_<NAME>/`. The directory is stamped from `main/drops/_TEMPLATE/` at Phase 1 start and persists through closeout.

- **Read `main/WIKI.md` + `main/PLAN.md` + `main/drops/WORKFLOW.md` at session start and after every compaction.** CLAUDE.md auto-loads; the other three do not — read them deliberately on the first turn after cold-start or compaction before substantive orchestration.
- **Use Tillsyn-style trackers for nothing.** Do NOT use Claude Code's built-in `TaskCreate` / `TaskUpdate` / `TaskList` / `TaskGet` / `TaskStop` / `TaskOutput` — they evaporate on compaction/restart. Decompose finer procedural granularity into atomic units inside the active drop's `PLAN.md` instead.
- **No markdown files outside `main/drops/` for work tracking.** Per-drop dirs are the worklog substrate.

## Drops

A **drop** is a unit of work — one entry in PLAN.md, one directory under `main/drops/`. Drops are declared in PLAN.md and refined in their own dir.

- Atomic granularity: a drop is "atomic" when one builder subagent can finish a single unit cleanly, the unit's acceptance criteria are yes/no-verifiable by a QA subagent, and its `paths` / `packages` footprint is clear. If a drop is too large, **add more units inside its `PLAN.md`** rather than stretching one unit.
- Ordering: parent-child nesting (a drop cannot close while any of its units is incomplete) + `blocked_by` for sibling and cross-unit ordering. No `depends_on` field.
- State: per-drop `state` lives in the drop dir's `PLAN.md` header (`planning` / `building` / `done` / `blocked`); per-unit `state` lives in the Planner section's unit row inside that file (`todo` / `in_progress` / `done` / `blocked`); container-level `state` lives in `main/PLAN.md`'s drop tree table.

Full lifecycle in `main/drops/WORKFLOW.md`. Drop tree in `main/PLAN.md`.

## Orchestrator-as-Hub

The parent Claude Code session launched by the dev from this directory is always **the orchestrator**. Every other role (builder, qa-proof, qa-falsification, planning, research) is a subagent spawned via the `Agent` tool.

**CRITICAL: The orchestrator NEVER writes Go code.** The parent session must not use `Edit`, `Write`, or any other tool to modify `.go` source, test, or `magefile.go` files. Every code change — every single one — goes through a `go-builder-agent` subagent. Orchestrator reads code for planning/research; edits markdown only (this file, `WIKI.md`, `PLAN.md`, drop dir mds, `LEDGER.md`, `README.md`, agent `.md` files).

### Agent Bindings

| Role | Agent | Edits Go? |
|---|---|---|
| Builder | `go-builder-agent` | **Yes** (only role that does) |
| QA Proof | `go-qa-proof-agent` | No |
| QA Falsification | `go-qa-falsification-agent` | No |
| Planning | `go-planning-agent` | No |
| Research | Claude's built-in `Explore` subagent | No |

The agents are **global** (`~/.claude/agents/`) and reference Tillsyn tooling that rak does not use. Every spawn carries the override preamble from `main/drops/WORKFLOW.md` § "Agent Spawn Contract" — single canonical source, do not duplicate it here. Per-role appendix fields (drop's PLAN.md path, unit ID, target output file, round number, working dir) are listed in WORKFLOW.md § "Per-Role Spawn Appendices".

## Build-QA-Commit Loop

Per-drop lifecycle is canonical in `main/drops/WORKFLOW.md` (Phases 1–7: plan, plan-QA, discuss + cleanup, build, build-QA, verify, closeout). This file does not duplicate the phase steps.

**Follow WORKFLOW.md's phases in order, exactly as written. No skipped phases. No reordered phases. No shortcut paths.** If a phase looks redundant for a particular drop, return the question to the dev — do not unilaterally drop it. Phase exits gate the next phase (see WORKFLOW.md § "Phase Order").

**Code is NEVER committed or pushed without per-unit QA passing first**, and **Hylla reingest is drop-end only** — both rules are enforced inside WORKFLOW.md's phases. Subagents never call `hylla_ingest`.

## Hylla Baseline

- **Artifact ref**: `github.com/evanmschultz/rak@main` — Hylla resolves `@main` to the latest ingest.
- **Hylla ingest is drop-end only**, not per-unit. Only the orchestrator calls `hylla_ingest`. Always `enrichment_mode=full_enrichment`, always from the GitHub remote, never before `git push` + `gh run watch --exit-status` green. Subagents never call `hylla_ingest`.

### Code Understanding Rules

1. **All Go code**: use Hylla MCP first (`hylla_search`, `hylla_node_full`, `hylla_search_keyword`, `hylla_refs_find`, `hylla_graph_nav`). Exhaust every Hylla search mode — vector, keyword, graph-nav, refs — before falling back to `LSP`, `Read`, `Grep`, `Glob`. **Whenever a Hylla miss forces a fallback, the subagent records the miss in its closing comment** under a `## Hylla Feedback` heading inside the drop's `BUILDER_WORKLOG.md`.
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
- **Unknowns** — what remains uncertain. Routed to the orchestrator (subagents return Unknowns in their final response; orchestrator surfaces to dev).

Short and inspectable. Full Section 0 spec lives in `~/.claude/CLAUDE.md` § "Semi-Formal Reasoning — Section 0 Response Shape". The Agent Spawn Contract preamble (in WORKFLOW.md) requires Section 0 from every subagent — but Section 0 stays in the orchestrator-facing response **only**, never inside `PLAN.md` / `BUILDER_WORKLOG.md` / `BUILDER_QA_*.md` / `PLAN_QA_*.md` / `CLOSEOUT.md`.

## QA Discipline

**No build unit is `done` without per-unit QA passing.** This is a gate, not a suggestion. Two asymmetric passes, not duplicates:

- **QA Proof** (`go-qa-proof-agent`) — evidence completeness, reasoning coherence, trace coverage. Asks: *"does the evidence support the claim?"*
- **QA Falsification** (`go-qa-falsification-agent`) — counterexamples, alternate traces, hidden dependencies, contract mismatches, YAGNI pressure. Asks: *"can I construct a case where this is wrong?"*

Plan-QA and build-QA both run as parallel proof + falsification spawns. Plan-QA writes transient files (`PLAN_QA_PROOF.md`, `PLAN_QA_FALSIFICATION.md`) that orch `git rm`s between rounds. Build-QA appends rounds to durable files (`BUILDER_QA_PROOF.md`, `BUILDER_QA_FALSIFICATION.md`). Full file-lifecycle table in `main/drops/WORKFLOW.md`.

## Orchestrator Role Boundaries

- **Orchestrator** (this parent Claude Code session) — plans, routes, delegates, cleans up. **Never edits Go code or `magefile.go`.** May edit markdown docs (this file, `WIKI.md`, `PLAN.md`, drop dir mds, `README.md`, `LEDGER.md`, `REFINEMENTS.md`, agent `.md` files).
- **Builder subagent** (`go-builder-agent`) — the ONLY role that edits Go code. Spawned via the `Agent` tool with the spawn contract preamble + builder appendix.
- **QA subagents** (`go-qa-proof-agent`, `go-qa-falsification-agent`) — gated to QA roles. Read, verify, write to their own `*_QA_*.md` file, return verdict to orch, die. Never edit code.
- **Planner subagent** (`go-planning-agent`) — fills the drop's `PLAN.md` Planner section (Phase 1) and revises it across plan-QA rounds (Phase 3). Never edits code.
- **Dev / human** — approves design calls during plan-QA discussion (Phase 3), reviews build-QA findings (Phase 5).

## Project Structure

Small, Go-idiomatic layout. Every internal package is an implementation detail — nothing here is a public API.

### Package Map

- `cmd/rak/` — cobra+fang entry. All flag wiring here; `RunE` dispatches into internal packages.
- `internal/counting/` — bytes/lines/words/chars primitive over `io.Reader`. Zero internal deps.
- `internal/fileset/` — `File` type (`Open()`, `Peek(n)`) + `Walker` with `iter.Seq2[*File, error]`. Depends on `internal/ignore`.
- `internal/ignore/` — `Matcher` interface unifying gitignore parsing + `--include`/`--exclude` globs. Zero internal deps.
- `internal/lang/` — `Language` (`type Language string`) + `Detect(*fileset.File)` + blank/comment/code split. Depends on `internal/fileset`.
- `internal/render/` — `Renderer` interface; explicit `NewHumanRenderer` (laslig-backed) + `NewJSONRenderer` constructors. Depends on `internal/summary`.
- `internal/summary/` — `Summary` struct (totals, per-dir, per-type rollups) + sort keyed by `--sort`. Zero internal deps.
- `internal/tokens/` — tiktoken wrapper, `CountTokens(io.Reader, encoding string) (int, error)`. Zero internal deps.
- `magefile.go` at repo root — mage build automation.

### Import DAG

Leaves: `counting`, `ignore`, `summary`, `tokens`. Mid: `fileset` → `ignore`; `lang` → `fileset`; `render` → `summary`. Root: `cmd/rak` → all of the above. No cycles, strictly layered.

### File Breakdown (expected sizes — no file exceeds ~400 LOC)

| File | Role | LOC |
|---|---|---|
| `cmd/rak/main.go` | `fang.Execute(ctx, newRootCmd())` only | ~30 |
| `cmd/rak/root.go` | cobra root cmd, flags, `RunE` dispatch | ~150 |
| `cmd/rak/root_test.go` | arg/flag parsing tests | ~150 |
| `internal/counting/counting.go` | `Counts` struct + `Count(io.Reader)` | ~100 |
| `internal/counting/counting_test.go` | ASCII, UTF-8, CRLF, empty | ~150 |
| `internal/fileset/file.go` | `File` type, `Open()`, `Peek(n)` | ~100 |
| `internal/fileset/walker.go` | `Walker` + `Walk() iter.Seq2[*File, error]` | ~200 |
| `internal/fileset/walker_test.go` | walks against `testdata/` + `fstest.MapFS` | ~200 |
| `internal/ignore/ignore.go` | `Matcher` interface + constructor | ~80 |
| `internal/ignore/gitignore.go` | gitignore parsing (library wrap) | ~100 |
| `internal/ignore/glob.go` | `--include`/`--exclude` globs | ~80 |
| `internal/ignore/ignore_test.go` | match tables | ~200 |
| `internal/lang/lang.go` | `Language` + `Detect(*fileset.File)` via `Peek(512)` | ~150 |
| `internal/lang/split.go` | blank/comment/code split | ~200 |
| `internal/lang/lang_test.go` | detection tests | ~150 |
| `internal/lang/split_test.go` | table-driven per language | ~200 |
| `internal/render/render.go` | `Renderer` interface + `Format` | ~60 |
| `internal/render/human.go` | laslig TTY renderer | ~150 |
| `internal/render/json.go` | JSON renderer | ~80 |
| `internal/render/render_test.go` | snapshot tests per format | ~200 |
| `internal/summary/summary.go` | `Summary` + rollups | ~100 |
| `internal/summary/sort.go` | sort funcs per `--sort` | ~80 |
| `internal/summary/summary_test.go` | rollup + sort | ~150 |
| `internal/tokens/tokens.go` | tiktoken wrapper | ~80 |
| `internal/tokens/tokens_test.go` | token-count tests | ~100 |
| `magefile.go` | mage targets | ~100 |

Non-test Go: ~1,600 LOC. Test Go: ~1,500 LOC. Total v0.1.0: ~3,100 LOC.

**Fixture directories** (not LOC-counted): `cmd/rak/testdata/` for the end-to-end integration fixture (guaranteed). Per-package `internal/<pkg>/testdata/` added only when a package genuinely needs real-file fixtures — default to `testing/fstest.MapFS` for unit tests. See § "Go Development Rules" → "Tests".

### Go-Idiomatic Naming Rules

1. **Package names:** lowercase, single-word, singular noun. No underscores, no mixedCase, no plurals. (`counting`, `fileset`, `lang`, `ignore`, `render`, `summary`, `tokens`.)
2. **Exported:** `MixedCase`. Unexported: `mixedCase`.
3. **Acronyms:** fully capitalized when leading or standalone (`JSONEncoder`, `parseHTTP`, `URL`); lowercase when internal (`id`, `url`).
4. **Getters:** omit `Get` — `f.Size()`, not `f.GetSize()`. Direct field access for plain data.
5. **Interfaces:** single-method interfaces end in `-er` (`Matcher`, `Renderer`, `Detector`). Multi-method interfaces use a role noun.
6. **Receivers:** 1–3 chars, consistent across all methods on a type (`f *File`, `w *Walker`, `m Matcher`). Never `self` / `this`.
7. **Errors:** sentinels named `ErrFoo` (e.g. `ErrTooDeep`). Wrap with `fmt.Errorf("...: %w", err)`. Error strings start lowercase, no trailing punctuation. Inspect with `errors.Is` / `errors.As` — never string-match.
8. **Test names:** `TestFuncName`; subcases via `t.Run("descriptive_name", ...)`.
9. **File names:** match the primary type (`file.go` contains `File`). Split when a file exceeds ~400 LOC or holds two concepts.
10. **Imports:** three groups (stdlib / third-party / local), blank-line-separated. `goimports` + `gofumpt` enforced.
11. **Doc comments:** every exported identifier gets a `// Name …` doc comment per `golint`. No TODO/FIXME without a tracking unit in the active drop's `PLAN.md`.
12. **Visibility:** everything under `internal/` by default. rak has no public API beyond the binary.

## Tech Stack

Production deps:

- Go 1.26+
- `github.com/spf13/cobra` — command tree.
- `github.com/charmbracelet/fang` — help/version/error polish; `fang.WithNotifySignal(os.Interrupt, syscall.SIGTERM)` wires signal cancellation into `cmd.Context()`.
- `github.com/evanmschultz/laslig` — structured output, TTY-aware rendering, optional spinner/activity.
- `github.com/tiktoken-go/tokenizer` — token counting (added in Drop 7.1).
- `github.com/magefile/mage` — build automation (added in Drop 1.4).
- `golang.org/x/sync/errgroup` — bounded concurrency for the parallel walker (added in Drop 8.1).

Dev tooling (installed locally, invoked via mage):

- `mvdan.cc/gofumpt` — stricter `gofmt` superset. `mage format` uses `gofumpt -l -w .`; `mage ci` asserts `gofumpt -l .` prints nothing.
- `github.com/golangci/golangci-lint/cmd/golangci-lint` — umbrella linter. `mage lint` runs `go vet ./...` + `golangci-lint run`.

## Build Verification

Per-unit verification (during build-QA, Phase 5 of WORKFLOW.md): builder runs `mage build` + `mage test` for the touched packages. Drop-end verification (after all units pass build-QA, Phase 6 of WORKFLOW.md): `mage ci` from `main/`, then `git push`, then `gh run watch --exit-status` until green.

1. All relevant mage targets pass (discover via `mage -l`).
2. **NEVER run raw `go test`, `go build`, `go run`, `go vet`, `gofumpt`, `golangci-lint`** — always `mage <target>`. If a mage target has a bug, fix the target — don't bypass. No exceptions, orchestrator or subagent.
3. **NEVER run `mage install` from an agent.** This is a **dev-only** dogfood target that promotes a binary to `$GOBIN`. Orchestrator and every subagent must not invoke it. If a unit description asks for it, stop and return control to the orchestrator.
4. All build-QA rounds for every unit have closed green.

Mage targets (land in Drop 1.4–1.5, stable from there):

| Target | Command | When |
|---|---|---|
| `mage build` | `go build ./...` | compile check |
| `mage test` | `go test -race ./...` | tests, race detector always on |
| `mage format` | `gofumpt -l -w .` | auto-format (writes) |
| `mage lint` | `go vet ./...` && `golangci-lint run` | static analysis |
| `mage ci` | `gofumpt -l .` (must be empty) && `mage lint` && `mage test` | pre-push gate |
| `mage install` | `go install ./cmd/rak` | **dev-only**, never from an agent |
| `mage run` | `go run ./cmd/rak` (positional args pass after `--`) | smoke check |
| `mage coverage` | `go test -race -coverpkg=./internal/... -coverprofile=coverage.out ./... && go tool cover -func=coverage.out` | report-only until Drop 9.3 flips it into a gate |
| `mage planCheck` | diff `main/PLAN.md` container titles + states against `main/drops/*/` directory names + each drop dir's `PLAN.md` header state | guards parity between PLAN.md and the drops dir |

Run `mage ci` before every push. `mage coverage` is report-only from Drop 1.5 on so every drop can see its current number; the 70% floor (scope `-coverpkg=./internal/...`, excludes `cmd/rak` CLI wiring) flips on in Drop 9.3.

## Go Development Rules

### Structure + Style

- **Interface-first boundaries**, dependency inversion where warranted.
- **Smallest concrete design.** No abstraction for hypothetical future variation.
- **TDD-first** where practical. Ship small tested increments.
- **Idiomatic Go** — follow the 12 naming rules in § "Project Structure". `gofumpt` + `goimports` enforce layout; `go vet` + `golangci-lint` catch the rest.
- **Go doc comments** on every exported identifier, starting with the identifier name.

### Errors

- **Wrap** with `fmt.Errorf("context: %w", err)` at every boundary that adds information. Never format with `%s` or `%v` for an error you want callers to be able to inspect.
- **Sentinels** named `ErrFoo` (e.g. `ErrTooDeep`, `ErrBinaryFile`) for expected conditions callers want to branch on.
- **Inspect** with `errors.Is` (sentinel match) or `errors.As` (type extraction). **Never string-match an error.**
- **Never swallow.** If you genuinely want to discard an error, assign to `_` with a one-line comment explaining why.

### Concurrency

- **Goroutines are bounded.** No unbounded `go func(){}()`. Use an `errgroup.Group` with `SetLimit(n)` or a semaphore channel for worker pools (Drop 8.1 is the first place this lands).
- **Every goroutine is context-cancellable.** Long-running loops check `ctx.Done()`. The `RunE` entrypoint threads `ctx` from `cmd.Context()` down.
- **`defer` for cleanup.** File closers, mutex unlocks, `cancel()` funcs, spinner `.Stop()` — always `defer` the cleanup on the line after the resource acquisition.
- **No shared mutable state without synchronization.** Prefer channels for ownership transfer. If a `sync.Mutex` is needed, keep it unexported on the struct that owns the data.
- **Race detector always on** — `mage test` runs `-race` unconditionally. CI fails if a race is detected.

### Tests

- `*_test.go` co-located with source. Table-driven for anything with input variants. Behavior-oriented assertions.
- `-race` via mage (`mage test`).
- Prefer `testing/fstest.MapFS` for walker tests (fast, no IO); reserve real `testdata/` for one end-to-end integration test in `cmd/rak`.
- **Two-tier `testdata/` rule.** When a package needs real-file fixtures, they live in `internal/<pkg>/testdata/` next to the test that reads them — Go stdlib idiom (`go help test` documents that the `testdata` directory name is ignored by `go` tooling). The single guaranteed fixture tree is `cmd/rak/testdata/`, holding the end-to-end integration input. **No shared top-level `testdata/`** — keep fixtures local to the test that owns them.

### Mage Discipline

- Plain `mage <target>` from the repo root. No `GOCACHE=...` overrides. No workspace-local cache dirs.
- If a target is missing or broken, add/fix the target — never bypass with a raw `go` command.

### After Touching Go Code

- `mage ci` before handoff at drop-end (Phase 6 of WORKFLOW.md). After pushing: `gh run watch --exit-status` until green.

### Dependencies

- Ask the dev to run `go get` / module updates. No `GOPROXY=direct`, `GOSUMDB=off`, or checksum bypass.
- **Bootstrap carve-out.** When a unit introduces a mage-managed dep for the very first time and no mage target yet exists to wrap `go get`, the builder MAY run `go get <module>` + `go mod tidy` directly from `main/` with default environment (no proxy / sum / checksum bypass, no private-module shenanigans). Today this applies only to Drop 1.4 (first-ever `github.com/magefile/mage` add). From Drop 2 onward the magefile exists, so every dep add routes through a mage target and this carve-out does not apply.

### Reference Lookups

- **Context7** + `go doc` + gopls `LSP` before any unfamiliar external API usage, after any test failure.

### Markdown Authoring

- Drop dir mds (`PLAN.md`, `BUILDER_WORKLOG.md`, `*_QA_*.md`, `CLOSEOUT.md`) are markdown-first. Use fenced code blocks for snippets, tables for structured data, headings for the conventions in `main/drops/WORKFLOW.md`. No HTML.

## Skill and Slash Command Routing

| Command | When to Use |
|---|---|
| `/qa-proof` | Proof-oriented QA (used inside subagent definitions; orchestrator typically just spawns the agent) |
| `/qa-falsification` | Falsification-oriented QA (same) |
| `/select-checkout` | Confirm the active visible checkout |
| `/gopls-sync` | Verify gopls targets `main/` |
| `semi-formal-reasoning` | Explicit reasoning certificate (Section 0 shape) |

Note: `/plan-from-hylla` is a Tillsyn-coupled global skill — rak does not use it. Planner work happens via `go-planning-agent` spawned per `main/drops/WORKFLOW.md` § "Phase 1".

## Git Commit Format

Conventional-commit: `type(scope): message`. All lowercase except proper nouns, acronyms (HTTP, CLI, JSON, TUI). Concise — describe what changed, not how.

**Subject-line only. No body. No bullet lists in the commit message.** The diff records what changed file-by-file; the subject line carries the human summary. Do not enumerate per-file changes in a body — that content belongs in the PR description, WIKI changelog, or LEDGER entry, not in `git log`.

Types: `feat`, `fix`, `refactor`, `chore`, `docs`, `test`, `ci`, `style`, `perf`.

Examples:
- `feat(count): add blank/comment/code line split per file`
- `fix(fileset): respect .gitignore in nested directories`
- `chore(deps): add tiktoken-go/tokenizer`
- `docs(drop-1): planner decompose into six units`
- `docs(drop-3): clear plan qa round 2, route to planner`

No co-authored-by trailers. No period at end. No capitalized first word after the colon unless proper noun/acronym. Keep the subject under ~72 chars when possible — if it won't fit, the change is probably too bundled and should be two commits.

## Safety

- Never delete files or directories without explicit dev approval.
- Never run commands outside the repo root `/Users/evanschultz/Documents/Code/hylla/rak`.
- Never push to any remote without explicit request.
- Keep secrets out of committed config files.

## Bare-Root and Worktree Discipline

- The bare repo at `/Users/evanschultz/Documents/Code/hylla/rak` (one level up) is the **steward orchestrator** root — not a coding checkout. It is a **flat bare repo**: `HEAD`, `config`, `objects/`, `refs/`, `worktrees/`, etc. live directly at the top level. There is no `.bare/` wrapper and no top-level `.git` pointer (an earlier `.bare/`-nested variant was tried and retired because it caused issues). This worktree's own `.git` pointer at `main/.git` reads `gitdir: /Users/evanschultz/Documents/Code/hylla/rak/worktrees/main`.
- This directory (`main/`) is the primary work checkout. Real coding / building / testing / committing happens here.
- Always confirm `pwd` is this checkout before edits, tests, commits, or gopls work.
- **Dev launches work orchestrators from here.** Steward orchestrators (project oversight, merge-conflict help) launch from the bare-root one level up and never edit source.
- If checkout context is unclear, use `/select-checkout`.
- Rak uses a single visible checkout (`main/`). Additional lane worktrees and gopls-sync only matter if a multi-lane setup is introduced later — the steward coordinates across them.

## Recovery After Session Restart

Filesystem + git, no Tillsyn calls. Full procedure in `main/drops/WORKFLOW.md` § "Recovery After Restart". Quick form:

1. `git status` — uncommitted work.
2. `git log --oneline -20` — recent commits.
3. Read `main/PLAN.md` — container states.
4. List `main/drops/*/PLAN.md` headers — per-drop phase state.
5. Per active drop: presence of `PLAN_QA_*.md` = mid-plan-QA loop; absence + `BUILDER_WORKLOG.md` exists = mid-build; `CLOSEOUT.md` with `state: done` = drop closed.
6. Per active unit: scan latest `## Unit N.M — Round K` heading in `BUILDER_WORKLOG.md` + both `BUILDER_QA_*.md` to figure out next step.
