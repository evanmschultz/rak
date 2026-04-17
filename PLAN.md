# Rak — Planning Doc

**Status:** **Persisted** backup mirror of the Tillsyn project tree, kept in-repo because Tillsyn is still under development and a local snapshot is cheap insurance. Tillsyn is the durable system of record; `PLAN.md` is the human-readable, git-versioned mirror that survives a Tillsyn outage or corrupted state.

**Update discipline:** every drop that mutates the Tillsyn tree (create, reword, close, add child, change `blocked_by`) also updates this file **in the same commit**, so `git log PLAN.md` tracks the plan's evolution alongside the Tillsyn changes. If the two drift, Tillsyn is authoritative and this file is reconciled to match.

## Project Intent

**Rak** — a fast project-sizing CLI for counting code. Walk a directory (or single file, or stdin), produce accurate counts with language-aware breakdowns, respect `.gitignore`, emit human-readable output through laslig on TTY and JSON when piped.

## Decisions Locked In (From Chat So Far)

1. **Name**: `rak` (binary + module), from Swedish *räkna* ("to count").
2. **Module**: `github.com/evanmschultz/rak`.
3. **License**: MIT.
4. **Repo**: `github.com/evanmschultz/rak` (private until flipped).
5. **Layout**: Go-idiomatic source tree under `main/` — `cmd/rak/` entry, `internal/{counting,fileset,lang,ignore,render,summary,tokens}/` packages, `magefile.go` at the repo root.
6. **Tech stack**:
    - **Production**: Go 1.26+, `github.com/spf13/cobra` (CLI), `github.com/charmbracelet/fang` (help/version/error polish + signal-to-context wiring via `WithNotifySignal`), `github.com/evanmschultz/laslig` (output), `github.com/tiktoken-go/tokenizer` (tokens, added in Drop 7.1), `golang.org/x/sync/errgroup` (bounded concurrency, added in Drop 8.1).
    - **Build**: `github.com/magefile/mage`.
    - **Dev tooling** (local, invoked via mage): `mvdan.cc/gofumpt` (stricter gofmt), `github.com/golangci/golangci-lint/cmd/golangci-lint` (umbrella lint).
7. **Build discipline**: mage-only, never raw `go` commands.
8. **Output discipline**: human-readable by default on TTY, JSON auto-selected when piped (laslig is TTY-aware).
9. **Pipe semantics**: on stdin, default to `wc`-parity (lines / words / bytes / chars). `--as <lang>` explicitly opts into code-aware counting on a stream. `--lang` is a separate walk-filter flag (decision 24).
10. **Ignore by default**: respect `.gitignore`. Escape hatches: `--no-gitignore`, `--include`/`--exclude` globs, `--hidden`, `--tracked-only` (backed by `git ls-files`). Binary files skipped by default; `--binary` to count separately.
11. **Token counting**: behind `--tokens`. Use `github.com/tiktoken-go/tokenizer` (pure Go, `cl100k_base` default). Approximate for Claude — document the caveat.
12. **Depth control**: `--depth N`. Also `--max-files` as a safety rail.
13. **Skip for v1 (YAGNI)**: cyclomatic complexity, churn, diff-vs-ref, caching.
14. **Progress bar**: laslig has spinner (incl. `meter` style) + a transient `Activity` live-block in `gotestout`, but no named `Progress` primitive for a known-total percentage fill. Evan will add a progress bar to laslig when rak genuinely needs it; until then, spinner + "processed N files" counter is enough.
15. **Orchestration**: never-edits-Go rule applies from the first builder drop onward. Bootstrap commit is docs + license + `.gitignore` only — no Go.
16. **Tillsyn coordination model (from updated tillsyn wiki 2026-04-16)**:
    - Only two node types: `project` + `drop`. Pre-Drop-2 creation rule: every new node under a rak project is written as `kind='task', scope='task'` but called a "drop" in prose. No other kinds. No `slice` terminology anywhere.
    - **Template-free project** — do not bind a template; tillsyn itself is template-free until cascade Drop 3+.
    - **Level addressing is 0-indexed**: project = level_0; level_1 = first child of project; level_N = N steps from the project root.
    - **Every level-1 drop opens with a planning drop** (`Role: planner`), which must close before any build drop under it starts.
    - **Every build drop gets two QA children**: `Role: qa-proof` and `Role: qa-falsification`, both `blocked_by: <build drop>`.
    - **`blocked_by`, not `depends_on`**.
    - **Rak is an external Tillsyn adopter** — every drop-end closeout writes a cross-project improvement prompt back to the Tillsyn team.

## Decisions Locked In (Resolved Open Questions, 2026-04-16)

17. **Subcommand shape**: single root command — `rak [path]`. No subcommands in v0.1.0. All current flags (`--tokens`, `--lang`, `--as`, `--depth`, `--sort`, `--format`) are orthogonal to the operation, not distinct operations. Subcommands can be added later without breaking the root UX; the reverse is painful.
18. **Stdin behavior**: on TTY-stdin with no path, hang and read stdin in `wc`-parity mode (matches `wc` convention — user terminates with Ctrl-D / EOF). Pipe + no path → `wc`-parity on stream. Pipe + `--as <lang>` → code-aware counting on stream.
19. **Sort default**: `lines desc` for the directory view. `--sort {lines,files,bytes,tokens,name}`; `--sort-asc` flips direction.
20. **Symlinks**: don't-follow by default; `--follow` opts in. Matches `rg` / `fd` convention.
21. **Progress indication**: deferred entirely to Drop 8. No spinner in Drops 1–7. Drop 8 adds spinner + "processed N files" counter once wall-time exceeds 250ms (time-based trigger, not count-based).
22. **Coverage gate**: no gate in Drop 1 CI — but `mage coverage` lands in Drop 1.5 as a **report-only** target (no threshold) so every subsequent drop sees its current number. This prevents a backfill mountain at Drop 9.3. Scope is `-coverpkg=./internal/...` (excludes `cmd/rak`, which is CLI wiring and would drag the number down). 70% floor flips on in Drop 9.3 before the v0.1.0 tag. Early drops have unstable APIs; locking coverage now creates churn.
23. **CI in first drop**: yes — `.github/workflows/ci.yml` ships in Drop 1.6. Matches tillsyn's discipline.
24. **Language flag split**: `--lang go,rs` = walk filter (which files to include from a tree; Drop 4.4). `--as go` = stdin stream-type assertion (treat the stream as language X; Drop 5.3). Two distinct flags, two distinct semantics. Avoids the mode-implicit CLI trap where one flag changes meaning based on input source.
25. **Drop 3 `fileset.File` contract**: `internal/fileset.File` struct exposes `Open() (io.ReadCloser, error)` and `Peek(n int) ([]byte, error)`. Binary detection (Drop 3.3) and shebang sniff (Drop 4.1) both consume `Peek(512)`. This commits Drop 3's public contract before Drop 4 starts so Drop 4 doesn't duplicate file-open logic — closes the scope crack QA falsification flagged.
26. **v0.1.0 scope cuts**: Drop 6.2 (tree view) and Drop 6.4 (TSV output) deferred to v0.2+. Drop 4.4 (`--lang <csv>` walk filter) kept — language filter is distinct intent from glob filter. Drop 8.1 (parallel walk) conditional on Drop 8 planner demonstrating >500ms wall-time on rak's own source; otherwise cut.
27. **Architecture**: 7 internal packages + `cmd/rak`, clean layered DAG (no cycles). Leaves: `counting`, `ignore`, `summary`, `tokens`. Mid: `fileset → ignore`, `lang → fileset`, `render → summary`. Root: `cmd/rak`. **No file exceeds ~400 LOC**; one primary type per file; split when the file holds two concepts. Total v0.1.0: ~1,600 non-test LOC + ~1,500 test LOC. Full breakdown in `main/CLAUDE.md` § "Project Structure". Five sub-choices locked: (a) walker uses `iter.Seq2[*File, error]`, (b) `Counts` struct + `Count()` func, (c) `type Language string`, (d) explicit `NewHumanRenderer`/`NewJSONRenderer` (no Format enum factory), (e) `testing/fstest.MapFS` for unit tests + one real `testdata/` integration test in `cmd/rak`.
28. **Quality tooling** (all via mage — agents never run raw tools): `gofumpt` (stricter gofmt, auto-format + format-check), `go vet`, `golangci-lint` (umbrella lint), `go test -race` (race detector always on), coverage gate (70% floor in Drop 9.3). `mage ci` = format-check + lint + test-with-race and is the pre-push gate. 12 Go-idiomatic naming rules live in `main/CLAUDE.md` § "Project Structure" → "Go-Idiomatic Naming Rules".
29. **Concurrency + error idioms**:
    - **Goroutines bounded** — `errgroup.Group` + `SetLimit(n)` or semaphore channel. No unbounded `go func(){}()`.
    - **Context-cancellable** — every goroutine checks `ctx.Done()`. `RunE` threads `cmd.Context()` downward.
    - **`defer` for cleanup** — file `Close()`, mutex `Unlock()`, spinner `Stop()` — always on the line after acquisition.
    - **No shared mutable state without synchronization** — prefer channels for ownership transfer; if `sync.Mutex` is needed, keep it unexported on the owning struct.
    - **Errors wrap with `%w`** at every boundary that adds info. Sentinel errors named `ErrFoo` (e.g. `ErrTooDeep`, `ErrBinaryFile`). Inspect with `errors.Is` / `errors.As` — never string-match. Never swallow (discard to `_` only with a one-line why-comment).
    - Full rules in `main/CLAUDE.md` § "Go Development Rules" → "Concurrency" + "Errors".

## Sketched Drop Hierarchy (For Tillsyn Once Created)

The plan below is a working shape — refined in chat before drops are created in Tillsyn. Everything below the project is a **drop** (single plan-item kind, `kind='task'` pre-Drop-2, `kind='drop'` post-rewrite).

**Required tree shape under every level-1 drop:**

- First child: a `Role: planner` drop (blocks every sibling until it closes).
- Builder work: one or more `Role: builder` drops; each builder drop has two QA children (`Role: qa-proof` + `Role: qa-falsification`, both `blocked_by: <builder>`).
- Final child: a `Role: commit` closeout drop (`blocked_by` every sibling).

The sub-items below are **builder drops** unless noted; mentally wrap each under its planner-first / qa-children / closeout-last shape when creating them in Tillsyn.

```
rak (project)
│
├── Drop 0 — Bootstrap (DONE, out-of-band — this conversation)
│   • Tree-shape exempt: predates Tillsyn project creation, so no planner/qa/closeout drops
│   • GH repo created
│   • CLAUDE.md (bare-root steward + main/ work), WIKI.md, README.md, LICENSE, .gitignore landed
│
├── Drop 1 — Code scaffold + mage + CI     [level_1]
│   ├── 1.planner — decompose, confirm acceptance criteria, set blocked_by     (Role: planner)
│   ├── 1.1 Move stashed files into target layout per decision 27: `go.mod` + `go.sum` to `main/`; rename flat `main.go` into `main/cmd/rak/main.go` + `main/cmd/rak/root.go` (split `fang.Execute` entry from cobra command construction so Drop 1.3 can rewrite `root.go` in isolation)     (Role: builder)
│   ├── 1.2 Rewrite go.mod module path to github.com/evanmschultz/rak (stash path is `github.com/evanmschultz/coding_challenges/fang`, not fwc)     (Role: builder)
│   ├── 1.3 Rewrite root command for rak shape: `Use: "rak [path]"`, `Args: cobra.MaximumNArgs(1)`, drop wc-style flags; `count(io.Reader) (Counts, error)` stays **unexported and in-file** in `cmd/rak/root.go` — Drop 2.1 owns the move into `internal/counting` and the export to `Count` (first-drop-hand-off boundary, pinned here so neither drop re-does it); wire `fang.WithNotifySignal(os.Interrupt, syscall.SIGTERM)` so `cmd.Context()` cancels on Ctrl-C / SIGTERM (prereq for Drop 8.1 parallel-walk cancellation)     (Role: builder)
│   ├── 1.4 Add github.com/magefile/mage dep; go mod tidy (MUST land before 1.5 — magefile.go imports github.com/magefile/mage/mg and won't compile until the dep is present)     (Role: builder)
│   ├── 1.5 Add magefile.go with 9 targets:                                                     (Role: builder)
│   │      • `build`      → `go build ./...`
│   │      • `test`       → `go test -race ./...` (race detector always on)
│   │      • `format`     → `gofumpt -l -w .`
│   │      • `lint`       → `go vet ./...` + `golangci-lint run` (config pinned in .golangci.yml — see 2.2 follow-up)
│   │      • `ci`         → `gofumpt -l .` (must be empty) + `mage lint` + `mage test`
│   │      • `install`    → `go install ./cmd/rak` — **dev-only, agents MUST NOT run it**
│   │      • `run`        → `go run ./cmd/rak` (positional args pass after `--`)
│   │      • `coverage`   → `go test -race -coverpkg=./internal/... -coverprofile=coverage.out ./... && go tool cover -func=coverage.out` — **report-only in Drop 1.5**; gate flips in Drop 9.3 (decision 22)
│   │      • `plan-check` → diff `main/PLAN.md` drop titles against live Tillsyn drop titles under the rak project; fail if drift (guards the "update PLAN.md in the same commit as every drop-mutating Tillsyn action" rule)
│   ├── 1.6 Add .github/workflows/ci.yml (mage ci on push/PR; no coverage gate yet — `mage coverage` is local-only report-only until Drop 9.3 flips it on)     (Role: builder)
│   │   — each builder above carries its own qa-proof + qa-falsification children (blocked_by: <builder>)
│   └── 1.closeout — git status clean, CI green, Hylla ingest, refinements + tillsyn-feedback prompt, LEDGER entry, WIKI update, delete /tmp/rak-stash/     (Role: commit)
│
├── Drop 2 — Counting domain + render boundary     [level_1]
│   ├── 2.planner     (Role: planner)
│   ├── 2.1 internal/counting: move `count` out of `cmd/rak/root.go`, export as `Count(io.Reader) (Counts, error)` with the `Counts` struct — bytes/lines/words/chars. Owns the lowercase→exported boundary (pinned per section-5 ruling, not Drop 1.3).     (Role: builder)
│   ├── 2.2 internal/render: laslig-backed printer, Format{Human,JSON} plumbing     (Role: builder)
│   ├── 2.3 Wire count subcommand (or root) to counting + render     (Role: builder)
│   ├── 2.4 TTY-vs-pipe auto-detect via laslig     (Role: builder)
│   ├── 2.5 Unit tests: counting table tests, render snapshot tests     (Role: builder)
│   │   — each builder above carries qa-proof + qa-falsification children
│   └── 2.closeout     (Role: commit)
│
├── Drop 3 — Directory walk + gitignore + depth     [level_1]
│   ├── 3.planner — dep research (gitignore lib via Context7 / go doc); commit `fileset.File` contract with Open() and Peek(n) methods (per decision 25)     (Role: planner)
│   ├── 3.1 internal/fileset: WalkDir-based traversal, depth limit, hidden-file skip; `File` struct with `Open() (io.ReadCloser, error)` + `Peek(n int) ([]byte, error)`     (Role: builder)
│   ├── 3.2 internal/ignore: .gitignore parsing, --include/--exclude globs     (Role: builder)
│   ├── 3.3 Binary file detection via `File.Peek(512)` (skip by default)     (Role: builder)
│   ├── 3.4 Per-dir aggregation wired into render output     (Role: builder)
│   │   — each builder above carries qa-proof + qa-falsification children
│   └── 3.closeout     (Role: commit)
│
├── Drop 4 — Language detection + code-aware splits     [level_1]
│   ├── 4.planner     (Role: planner)
│   ├── 4.1 internal/lang: extension map + shebang sniff via `File.Peek(512)` + simple content heuristic     (Role: builder)
│   ├── 4.2 Blank/comment/code split per detected language     (Role: builder)
│   ├── 4.3 Per-type aggregation in render output     (Role: builder)
│   ├── 4.4 `--lang go,rs` walk filter — which files to include from a tree (not to be confused with Drop 5.3's `--as <lang>` stream-type flag)     (Role: builder)
│   │   — each builder above carries qa-proof + qa-falsification children
│   └── 4.closeout     (Role: commit)
│
├── Drop 5 — Stdin pipe behavior     [level_1]
│   ├── 5.planner     (Role: planner)
│   ├── 5.1 Detect stdin is piped vs TTY; on TTY-stdin + no-path, hang and read stdin in wc-parity (decision 18)     (Role: builder)
│   ├── 5.2 Default wc-parity counts on stream     (Role: builder)
│   ├── 5.3 `--as <lang>` stream-type assertion — opt in to code-aware counting on a stream (separate flag from Drop 4.4's `--lang` walk filter; decision 24)     (Role: builder)
│   ├── 5.4 JSON output in pipe-to-pipe chain     (Role: builder)
│   │   — each builder above carries qa-proof + qa-falsification children
│   └── 5.closeout     (Role: commit)
│
├── Drop 6 — Summary + sorting     [level_1]
│   ├── 6.planner     (Role: planner)
│   ├── 6.1 internal/summary: totals row, per-dir rollup, per-type rollup     (Role: builder)
│   ├── 6.2 `--sort {lines,files,bytes,tokens,name}` with `--sort-asc` direction flip; default `lines desc` (decision 19)     (Role: builder)
│   │   — each builder above carries qa-proof + qa-falsification children
│   │   — [DEFERRED to v0.2+: tree view, TSV output — decision 26]
│   └── 6.closeout     (Role: commit)
│
├── Drop 7 — Token counting     [level_1]
│   ├── 7.planner     (Role: planner)
│   ├── 7.1 Add github.com/tiktoken-go/tokenizer dep     (Role: builder)
│   ├── 7.2 internal/tokens: count tokens per file (cl100k_base default, configurable)     (Role: builder)
│   ├── 7.3 --tokens flag + output integration     (Role: builder)
│   ├── 7.4 Document approximation caveat in README     (Role: builder)
│   │   — each builder above carries qa-proof + qa-falsification children
│   └── 7.closeout     (Role: commit)
│
├── Drop 8 — Perf + UX polish     [level_1]
│   ├── 8.planner — benchmark rak on its own source + a larger tree; decide whether 8.1 (parallel walk) is justified per decision 26     (Role: planner)
│   ├── 8.1 Parallel directory walk (bounded worker pool) — CONDITIONAL on 8.planner showing >500ms wall-time; cut otherwise     (Role: builder)
│   ├── 8.2 Progress indication on TTY — laslig spinner + "processed N files" counter, trigger when wall-time > 250ms (decision 21)     (Role: builder)
│   ├── 8.3 --max-files safety rail     (Role: builder)
│   ├── 8.4 --tracked-only via git ls-files     (Role: builder)
│   ├── 8.5 Symlink handling (default don't-follow, --follow flag; decision 20)     (Role: builder)
│   │   — each builder above carries qa-proof + qa-falsification children
│   └── 8.closeout     (Role: commit)
│
└── Drop 9 — Release + docs     [level_1]
    ├── 9.planner — finalize 9.x ordering; note that 9.5 "flip repo public" is a dev-manual action with no code footprint, so its QA children are pro-forma (planner documents this explicitly)     (Role: planner)
    ├── 9.1 Fill out README with real examples (replace aspirational usage)     (Role: builder)
    ├── 9.2 Add --version via fang.WithVersion     (Role: builder)
    ├── 9.3 Flip Drop 1.5's report-only `mage coverage` into a gate: 70% floor with scope `-coverpkg=./internal/...` (excludes `cmd/rak`), enforced in `mage ci` + CI workflow (decision 22)     (Role: builder)
    ├── 9.4 GoReleaser config for binary releases + local dry-run (`goreleaser release --snapshot`)     (Role: builder)
    ├── 9.5 Flip repo public (dev-manual GitHub action; QA children verify repo visibility + CI still green)     (Role: builder)
    ├── 9.6 Tag v0.1.0 and push tag (triggers GoReleaser in CI)     (Role: builder)
    │   — each builder above carries qa-proof + qa-falsification children
    └── 9.closeout     (Role: commit)
```

## Tillsyn UUID Index

Live mirror of the Tillsyn state for the rak project. Updated in the same commit as every drop-mutating Tillsyn action (create, reword, close, add child, change `blocked_by`). If this table drifts from Tillsyn, Tillsyn is authoritative and this section is reconciled to match.

**Project** — `RAK` — `f68564e7-9b50-4bef-bc73-c70297f1d3c4`, template-free, `metadata.external_adopter=true`, `metadata.hylla_artifact_ref=github.com/evanmschultz/rak@main`.

**Columns** (supplied out-of-band 2026-04-17; no MCP `till_column` tool exists yet — known Tillsyn bug, fix pending):

| UUID | Name | Position | Use |
|---|---|---|---|
| `cb4f6695-59f4-4181-966a-ccaf4be29c08` | To Do | 0 | default write target for new drops |
| `e5ee13b9-1c16-483a-bcf5-063be8af66d0` | In Progress | 1 | active work |
| `f3d0df47-5a95-4364-97a8-dc938643401e` | Done | 2 | terminal (auto-moved by `move_state=done`) |
| `3e4e9b3a-0bcc-4517-8f03-83918b94fbd0` | Failed | 3 | archived 2026-04-17 — do not target |

**Drops created so far** (state as of the latest commit that touches this file):

| Title | UUID | State | Parent | `blocked_by` |
|---|---|---|---|---|
| `DROP_0_BOOTSTRAP` | `ab9db71d-6b4f-45dd-8472-0e96743eac49` | `done` | — (level_1) | — |
| `DROP_1_CODE_SCAFFOLD_MAGE_CI` | `8dccebb1-a22c-48ad-b2c9-2c75b6fd46fb` | `todo` | — (level_1) | — (chain start) |

**In-project mutation auth shape (pre-Drop-2 gotcha reminder):** every `till_plan_item(create)` needs the full five-field tuple (`session_id` + `session_secret` + `auth_context_id` + `agent_instance_id` + `lease_token`) **plus** `column_id`. For level_1 drops, `parent_id` must be **omitted** — passing the project's UUID there returns `not_found: authorize mutation: not found` (misleading). For nested drops, `parent_id` is the parent drop's UUID.

## Tillsyn Project Creation Parameters (Historical)

Retained for audit: the rak project was created in Tillsyn on 2026-04-17 with these parameters. No action required — this is here so a future reviewer can reconstruct how the project was authored if the Tillsyn record is ever lost.

- `operation: create` on `till_project`.
- `kind: project`, `scope: project`.
- `slug / name`: `RAK` (uppercase per Evan's naming convention — user-facing Tillsyn project names are fully uppercase).
- `template: none` (explicitly — do NOT bind `default-go` or any other template; Tillsyn itself is template-free and instructs external adopters to skip template binding until `child_rules` return).
- `metadata`:
  - `hylla_artifact_ref: github.com/evanmschultz/rak@main`
  - `homepage: https://github.com/evanmschultz/rak`
  - `language: go`
  - `external_adopter: true` (so drop-end closeouts know to produce the cross-project improvement prompt for the Tillsyn team)

All child drops under the project are created with `kind='task', scope='task'` (pre-Drop-2 creation rule) and role noted in description prose as `Role: planner | builder | qa-proof | qa-falsification | commit`.

## Per-Level-1-Drop Structure Template

Every level-1 drop below gets this shape at creation time:

```
<Drop N> (level_1, kind=task, no Role prefix — container drop)
├── N.planner          (kind=task, Role: planner)      — first child; blocks all siblings until done
├── N.1 <builder task> (kind=task, Role: builder)      — blocked_by: N.planner
│   ├── N.1.qa-proof           (kind=task, Role: qa-proof,         blocked_by: N.1)
│   └── N.1.qa-falsification   (kind=task, Role: qa-falsification, blocked_by: N.1)
├── N.2 <builder task> (kind=task, Role: builder)      — blocked_by: N.planner
│   ├── N.2.qa-proof           (blocked_by: N.2)
│   └── N.2.qa-falsification   (blocked_by: N.2)
├── ... more builder drops with the same qa-children shape ...
└── N.closeout         (kind=task, Role: commit)       — blocked_by: every other sibling in this level-1 subtree
```

### Planner Drop Description Template

Every `N.planner` drop pastes this block into its description at creation time, so planner responsibilities are written down on the drop itself rather than re-derived from `main/CLAUDE.md` by every subagent:

```
Role: planner

Deliverables (all required before this drop can close):
1. Confirm level_1 scope with dev — restate the goal of Drop N in one sentence, flag any ambiguity.
2. Decompose into atomic nested builder drops. Every child carries: `paths` (file-level footprint), `packages` (Go package footprint), and acceptance criteria (yes/no-verifiable by a QA subagent).
3. Set `blocked_by` across siblings where ordering matters. Parent-child nesting is the implicit depends_on — do not layer `depends_on` on top.
4. File cross-cutting discussions as their own drops (e.g. "decide library X vs Y"), not inline inside a builder drop description.
5. Commit the updated `main/PLAN.md` mirror in the same commit as the Tillsyn mutations that land the decomposed child drops (parity rule).
```

Closeout drop (`Role: commit`) responsibilities per `main/WIKI.md` § "Drop-End Closeout Checklist" (9 steps, canonical):

1. Every sibling `done`, `git status --porcelain` clean.
2. All commits on remote. CI green (`gh run watch --exit-status`).
3. Aggregate per-subagent `## Hylla Feedback` sections → `main/HYLLA_FEEDBACK.md`.
4. Aggregate usage findings → `main/REFINEMENTS.md` / `main/HYLLA_REFINEMENTS.md`.
5. **External-adopter cross-project improvement prompt** → prompt routed to the Tillsyn team.
6. `hylla_ingest` full-enrichment from remote, after CI green.
7. Append entry to `main/LEDGER.md` (created when the first drop closes).
8. Append one-liner to `main/WIKI_CHANGELOG.md`.
9. Update `main/WIKI.md` sections in place if anything changed best practice.

## Immediate Next Step

1. Create Drops 2–9 level_1 container drops in Tillsyn, chained by `blocked_by` (Drop N `blocked_by` Drop N-1).
2. Create nine `Role: planner` first-child drops (one per level_1, using the Planner Description Template below). Planners decompose their level_1 into builder + QA + closeout sub-drops during their own session.
3. Update the "Drops created so far" table above in the same commit as the Tillsyn mutations.

## Follow-Ups / Outstanding Orchestration Tasks

Items tracked for future sessions, separate from the Drop 0–9 hierarchy:

- **Tillsyn Drop 2 rewrite watch** — when Tillsyn ships its Drop 2 rewrite (non-project nodes created as literal `kind='drop'` instead of `kind='task'`), update rak's `main/CLAUDE.md` and `main/WIKI.md` to drop the pre-Drop-2 creation-rule paragraphs and the "Do Not Use Other Kinds" section. Move role from description prose to `drop.metadata.role`. Memory `tillsyn_drops_vocabulary.md` has the detection signals.
- **Laslig progress-bar follow-up** — when rak's perf drop (Drop 8) genuinely needs a known-total progress bar, dev extends laslig upstream (not rak). Until then, spinner + "processed N files" counter is enough.
- **`default-go` template binding re-evaluation** — if Tillsyn's Drop 3+ ships `child_rules` that enforce the planner-first / qa-children / closeout-last shape, reconsider whether rak should bind a template at that point instead of enforcing tree shape manually.
- **Stash cleanup** — `/tmp/rak-stash/` is deleted entirely in Drop 1 closeout, once Drop 1.1 has moved main.go/go.mod/go.sum into `main/` and the stashed legacy `PLAN.md` is confirmed folded into this hierarchy. `test.txt` is never copied into the repo.

## Stashed Legacy Files

From the prior `fwc` prototype at `/Users/evanschultz/Documents/Code/hylla/rak/` (pre-bootstrap):

- `/tmp/rak-stash/main.go` — cobra+fang single-file wc-style CLI. Carried over in Drop 1.1; Drop 1.3 rewrites the root command for rak's shape but preserves the `count(io.Reader) (Counts, error)` primitive for Drop 2.1 reuse.
- `/tmp/rak-stash/go.mod` — module path is `github.com/evanmschultz/coding_challenges/fang` (not `fwc`). Drop 1.2 rewrites it to `github.com/evanmschultz/rak`.
- `/tmp/rak-stash/go.sum` — kept as-is for now.
- `/tmp/rak-stash/PLAN.md` — obsolete `fwc` plan. Useful phasing notes folded into the drop hierarchy above; not copied into `main/`.
- `/tmp/rak-stash/test.txt` — 342KB coding-challenge fixture. Not copied into `main/`; rak will generate its own test corpus.

Entire `/tmp/rak-stash/` directory is deleted in Drop 1 closeout.
