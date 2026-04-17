# Rak — Project CLAUDE.md (main worktree)

This file lives in the **`main/` worktree** at `/Users/evanschultz/Documents/Code/hylla/rak/main/`. This is the primary work checkout — all real coding, building, testing, and committing happens here. **The dev launches work orchestrators from this directory.** Sessions launched from the bare-root one directory up are **steward orchestrators** with a different prompt (bare-root `CLAUDE.md`) and a different scope — cross-worktree oversight and merge-conflict help, not feature work.

## Tillsyn Is the System of Record

All work is tracked in Tillsyn. No exceptions.

- No markdown files for work tracking, coordination, worklogs, or execution state.
- **Tillsyn = durable truth.** Every piece of work gets a Tillsyn plan item (a **drop**) before it starts.
- **Use Tillsyn exclusively for work tracking.** Do NOT use Claude Code's built-in `TaskCreate` / `TaskUpdate` / `TaskList` / `TaskGet` / `TaskStop` / `TaskOutput` — they evaporate on compaction/restart. Decompose into child drops instead.
- **When work starts on a drop, move it to `in_progress` immediately.**
- **Read `main/WIKI.md` at session start and after every compaction.** The wiki is the living best-practice snapshot and changes as the project evolves. CLAUDE.md is auto-loaded; WIKI.md is NOT — Read it deliberately on the first turn after cold-start or compaction.
- **`main/PLAN.md` is a persisted backup mirror of the Tillsyn project tree** (not transient). Kept in-repo because Tillsyn is still under development and a local snapshot is cheap insurance. **Every drop-mutating Tillsyn action (create, reword, close, add child, change `blocked_by`) updates `PLAN.md` in the same commit**, so `git log PLAN.md` tracks plan evolution alongside the Tillsyn changes. If the two drift, Tillsyn is authoritative and `PLAN.md` is reconciled to match.

### Drops — The Only Plan-Item Kind

Tillsyn has exactly **two node types**: `project` and **drop**. Drops nest infinitely. A drop is the Tillsyn-native word for a unit of work. The term "slice" is **not** used — it was prior internal vocabulary for this concept and has been retired.

- `project` — the rak root container (not a drop).
- **drop** — every node below the project. Nest drops until they are atomic (one builder subagent can finish cleanly, acceptance criteria are yes/no-verifiable, paths/packages footprint is clear).

**Pre-Drop-2 creation rule (current Tillsyn HEAD):** every new rak node is created with `kind='task', scope='task'`. Do **not** use any other kind (`build-task`, `subtask`, `qa-check`, `plan-task`, `commit-and-reingest`, `a11y-check`, `visual-qa`, `design-review`, `phase`, `branch`, `decision`, `note`) even though they remain in `kind_catalog`. Tillsyn's post-Drop-2 SQL rewrites every non-project node to literal `kind='drop'`. **Refer to the node as a drop in prose** regardless of the current on-disk kind value.

**Role in description prose (pre-Drop-2):** post-Drop-2 role lives on `drop.metadata.role`. Until that field lands, note the role in the drop description as `Role: <role>` where role ∈ { `planner`, `builder`, `qa-proof`, `qa-falsification`, `commit` }.

**Template-free project:** do **not** bind a template to the rak project. Tillsyn itself is `template: none` and explicitly instructs external adopters to skip template binding until `child_rules` return in Tillsyn's Drop 3+. Rak enforces tree shape manually per the rules below.

**Level addressing is 0-indexed:** the project itself is **level 0**. `level_1` = every drop directly under the rak project (first-child drops). `level_2` = drops one level below a level_1 drop. `level_N` = N steps deep from the project root. 0-indexed on purpose — the whole Tillsyn DB zero-indexes everything, so levels do too. Dotted addresses (`0.1.5.2`) are read-only shorthand — **mutations always use UUIDs**.

### Required Tree Shape

- **Every level-1 drop opens with a planning drop.** Its first child is a `Role: planner` drop whose job is a dev ↔ orchestrator discussion: confirm scope, decompose the level-1 drop into atomic nested drops, set `blocked_by` across siblings, file cross-cutting discussions as their own drops. **Until the planning drop is `done`, no build drop under the level-1 drop is eligible to start.**
- **Every build drop gets two QA children.** A `Role: qa-proof` drop and a `Role: qa-falsification` drop, both with `blocked_by: <build drop>`. Both must pass before the build drop can close.
- **Use `blocked_by`, not `depends_on`.** Parent-child nesting is the implicit depends_on (a parent cannot close while any child is incomplete). `blocked_by` is the only sibling / cross-subtree ordering primitive. Do not layer `depends_on` on top.

### Rak Is an External Tillsyn Adopter

Rak uses Tillsyn as an external adopter — it is not the Tillsyn repo. The tillsyn wiki mandates that external-adopter drop-end closeouts include a **cross-project improvement prompt** routed back to the Tillsyn team, capturing: context (rak is a Go CLI, solo dev), friction (schema confusion, missing primitives, MCP ergonomics), workarounds, ranked requests, and evidence (drop/comment/handoff IDs). This deliverable is part of every rak drop-end task, alongside the local findings log (`HYLLA_FEEDBACK.md`, future `REFINEMENTS.md`).

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

## Build-QA-Commit Loop (Pre-Cascade)

**Code is NEVER committed or pushed without QA completing first.** Until Tillsyn's cascade dispatcher ships, the parent orchestrator runs this loop manually:

1. **Plan** — `go-planning-agent` (or orchestrator + dev, for trivial drops) decomposes the level-1 drop into atomic nested drops with `paths` / `packages` / acceptance criteria. First child of every level-1 drop is a `Role: planner` drop; no build drop is eligible until it closes.
2. **Build** — `go-builder-agent` subagent implements the increment. **The builder moves its own drop to `in_progress` at start**, commits evidence to `implementation_notes_agent` + `completion_notes`, moves the drop to `done` at end, and closes with a `## Hylla Feedback` section.
3. **QA Proof + QA Falsification (parallel)** — `go-qa-proof-agent` + `go-qa-falsification-agent`, both children of the build drop with `blocked_by: <build drop>`. Each moves its own QA drop to `in_progress` at start, `done` on pass, or leaves `in_progress` + posts findings on fail.
4. **Fix** — if either QA fails, respawn the builder, re-run QA. Build drop stays `in_progress` until both QA children pass.
5. **Commit** — after both QA pass, orchestrator + dev commit with conventional-commit format. **`git add <paths>` — never `git add .`**.
6. **Push + CI green** — `git push`, then `gh run watch --exit-status` until green.
7. **Update Tillsyn** — checklist + metadata + terminal state.

**No batched commits. No deferred pushes. No skipped QA. No skipped CI watch. No claiming done in chat without Tillsyn reflecting it.**

Hylla reingest is **drop-end only** — once per drop, inside the end-of-drop closeout task, full enrichment from remote, only after CI green. Subagents never call `hylla_ingest`.

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

**No build drop is `done` without QA passing.** This is a gate, not a suggestion. Two asymmetric passes, not duplicates:

- **QA Proof** (`go-qa-proof-agent`, `/qa-proof`) — evidence completeness, reasoning coherence, trace coverage. Asks: *"does the evidence support the claim?"*
- **QA Falsification** (`go-qa-falsification-agent`, `/qa-falsification`) — counterexamples, alternate traces, hidden dependencies, contract mismatches, YAGNI pressure. Asks: *"can I construct a case where this is wrong?"*

Every build drop has **two QA children** — one `Role: qa-proof` drop and one `Role: qa-falsification` drop — both with `blocked_by: <build drop>`. Both must pass before the build drop is eligible to close. If either finds issues, the build drop stays `in_progress`, the finding is recorded on the failed QA drop, a fix drop runs, and QA re-runs. Spawn QA subagents in parallel for fresh-context isolation.

## Drop Decomposition Rules

**Atomic drop granularity.** A drop is "atomic" when:

- One builder subagent (or one orchestrator + dev pairing, pre-cascade) can finish it in a single working session.
- Its acceptance criteria are concrete and verifiable — a QA subagent can make a yes/no call.
- It has a clear `paths` / `packages` footprint so file- / package-level blocking works.

If a drop is too large to fit, **nest further** rather than stretching the drop.

**Every level-1 drop opens with a planning drop.** First child is `Role: planner`; it runs the dev ↔ orchestrator decomposition discussion and sets `blocked_by` across siblings. Nested drops (level_2+) do not universally require their own planning drop — but any ambiguous or large nested drop gets one too.

**`blocked_by`, not `depends_on`.** Parent-child nesting is the implicit dependency (a parent cannot close while any child is open). `blocked_by` is the only sibling / cross-subtree ordering primitive.

**State hygiene.** A drop is moved to `in_progress` the moment work on it starts — by the subagent that owns it, not by the orchestrator. No `todo` items left while someone is working on them.

## Drop-End Closeout Checklist

Every level-1 drop ends with a closeout drop (`Role: commit`, `blocked_by` every other sibling in the level-1 subtree). Nine steps, canonical (mirrored in `WIKI.md` and `PLAN.md`):

1. All sibling drops `done`. `git status --porcelain` clean.
2. All commits on remote. CI green (`gh run watch --exit-status`).
3. Aggregate per-subagent `## Hylla Feedback` sections into `HYLLA_FEEDBACK.md` (created when first such section lands).
4. Aggregate usage findings — ergonomic wins, ergonomic pain, bugs, lessons — into `REFINEMENTS.md` / `HYLLA_REFINEMENTS.md` (created on first entry).
5. **External-adopter cross-project improvement prompt** — rak is not the Tillsyn repo, so every drop-end writes a prompt for the Tillsyn team: context, friction, workarounds, ranked requests, evidence (drop/comment/handoff IDs). Routed via issue / PR / `till.handoff` once the Tillsyn-team identity exists.
6. `hylla_ingest` — full enrichment, from the GitHub remote, only after CI green.
7. Append an entry to `LEDGER.md` (created when the first drop closes).
8. Append a one-liner to `WIKI_CHANGELOG.md` (created when the first wiki-changing drop lands).
9. Update the relevant section(s) of `WIKI.md` if anything shipped that changed best practice (**in place** — no `2026-XX-XX update:` appended notes; git history is the audit trail).

## Orchestrator Role Boundaries

- **Orchestrator** (this parent Claude Code session) — plans, routes, delegates, cleans up. **Never edits Go code or `magefile.go`.** May edit markdown docs (this file, `WIKI.md`, `README.md`, `LEDGER.md`, `REFINEMENTS.md`, agent `.md` files).
- **Builder subagent** (`go-builder-agent`) — the ONLY role that edits Go code. Spawned via the `Agent` tool with Tillsyn auth credentials in the prompt.
- **QA subagents** (`go-qa-proof-agent`, `go-qa-falsification-agent`) — gated to QA roles. Read, verify, verdict, die. Never edit code.
- **Planner subagent** (`go-planning-agent`) — decomposes a level-1 drop into atomic nested drops. Never edits code.
- **Dev / human** — approves auth, reviews results, makes design calls that the orchestrator files as discussion drops.

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
11. **Doc comments:** every exported identifier gets a `// Name …` doc comment per `golint`. No TODO/FIXME without a tracking drop.
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

Before any build-drop is marked done:

1. All relevant mage targets pass (discover via `mage -l`).
2. **NEVER run raw `go test`, `go build`, `go run`, `go vet`, `gofumpt`, `golangci-lint`** — always `mage <target>`. If a mage target has a bug, fix the target — don't bypass. No exceptions, orchestrator or subagent.
3. **NEVER run `mage install` from an agent.** This is a **dev-only** dogfood target that promotes a binary to `$GOBIN`. Orchestrator and every subagent must not invoke it. If a drop description asks for it, stop and return control to the orchestrator.
4. All QA drops (proof + falsification) for this build-drop have closed green.

Mage targets (land in Drop 1.4, stable from there):

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
| `mage plan-check` | diff `main/PLAN.md` drop titles vs live Tillsyn | guards the "update PLAN.md in the same commit" parity rule |

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

- `mage ci` before handoff. After pushing: `gh run watch --exit-status` until green.

### Dependencies

- Ask the dev to run `go get` / module updates. No `GOPROXY=direct`, `GOSUMDB=off`, or checksum bypass.

### Reference Lookups

- **Context7** + `go doc` + gopls `LSP` before any unfamiliar external API usage, after any test failure.

### Tillsyn Authoring

- **Markdown-first** for Tillsyn `description`, `summary`, `body_markdown`, thread comments.

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

**Subject-line only. No body. No bullet lists in the commit message.** The diff records what changed file-by-file; the subject line carries the human summary. Do not enumerate per-file changes in a body — that content belongs in the PR description, WIKI changelog, or LEDGER entry, not in `git log`.

Types: `feat`, `fix`, `refactor`, `chore`, `docs`, `test`, `ci`, `style`, `perf`.

Examples:
- `feat(count): add blank/comment/code line split per file`
- `fix(fileset): respect .gitignore in nested directories`
- `chore(deps): add tiktoken-go/tokenizer`
- `docs(plan): add plan.md mirror and codify drops vocab`

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

1. `till.capture_state` — re-anchor project and scope.
2. `till.attention_item(operation=list, all_scopes=true)` — inbox.
3. `till.handoff(operation=list)` — open routing.
4. Check `in_progress` drops for staleness.
5. Revoke orphaned auth sessions / leases.
