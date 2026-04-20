# Rak — Planning Doc

**Status:** Durable system of record for the overarching drop tree. Each drop's per-phase worklog lives under `main/drops/DROP_N_<NAME>/` — see `main/drops/WORKFLOW.md` for the canonical lifecycle. PLAN.md is updated **after** a drop closes (state flip + any structural changes that came out of the work) and **after** a planner restructures the tree. Not edited mid-build.

**Workflow:** see `main/drops/WORKFLOW.md` for the per-drop lifecycle (plan → plan-QA → discuss → revise → loop → build → build-QA → verify → closeout). PLAN.md owns the overarching plan; WORKFLOW.md owns the phase mechanics; `main/CLAUDE.md` owns role boundaries + Go quality rules.

## Project Intent

**Rak** — a fast project-sizing CLI for counting code. Walk a directory (or single file, or stdin), produce accurate counts with language-aware breakdowns, respect `.gitignore`, emit human-readable output through laslig on TTY and JSON when piped.

## Decisions Locked In

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
16. **Coordination model**: drop = directory under `main/drops/` (e.g. `main/drops/DROP_1_CODE_SCAFFOLD_MAGE_CI/`). Per-drop lifecycle (planner → plan-QA → discuss → revise → builder → build-QA → verify → closeout) lives in `main/drops/WORKFLOW.md`. PLAN.md tracks overarching containers + state; per-phase mechanics live in WORKFLOW.md; role boundaries live in `main/CLAUDE.md`. Subagents are global (`~/.claude/agents/`) but are spawned with a paradigm-override directive that tells them to ignore Tillsyn-coupled instructions and follow WORKFLOW.md instead.
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

## Drop Tree

The plan below is the working shape. Each row is a level_1 container drop. **Each drop's atomic-unit decomposition lives inside its own `main/drops/DROP_N_<NAME>/PLAN.md`**, written by the planner subagent during Phase 1 of `WORKFLOW.md`. The sub-bullets below are the **expected** decomposition — refined and committed per drop when the planner runs.

| Drop | State | Blocked by | Drop dir |
|---|---|---|---|
| `DROP_0_BOOTSTRAP` | done | — | (out-of-band; predates this workflow — see `WIKI.md` § "Current State") |
| `DROP_1_CODE_SCAFFOLD_MAGE_CI` | done | — | `main/drops/DROP_1_CODE_SCAFFOLD_MAGE_CI/` |
| `DROP_2_COUNTING_DOMAIN_RENDER_BOUNDARY` | todo | DROP_1 | `main/drops/DROP_2_COUNTING_DOMAIN_RENDER_BOUNDARY/` |
| `DROP_3_DIRECTORY_WALK_GITIGNORE_DEPTH` | todo | DROP_2 | `main/drops/DROP_3_DIRECTORY_WALK_GITIGNORE_DEPTH/` |
| `DROP_4_LANGUAGE_DETECTION_CODE_SPLITS` | todo | DROP_3 | `main/drops/DROP_4_LANGUAGE_DETECTION_CODE_SPLITS/` |
| `DROP_5_STDIN_PIPE_BEHAVIOR` | todo | DROP_4 | `main/drops/DROP_5_STDIN_PIPE_BEHAVIOR/` |
| `DROP_6_SUMMARY_SORTING` | todo | DROP_5 | `main/drops/DROP_6_SUMMARY_SORTING/` |
| `DROP_7_TOKEN_COUNTING` | todo | DROP_6 | `main/drops/DROP_7_TOKEN_COUNTING/` |
| `DROP_8_PERF_UX_POLISH` | todo | DROP_7 | `main/drops/DROP_8_PERF_UX_POLISH/` |
| `DROP_9_RELEASE_DOCS` | todo | DROP_8 | `main/drops/DROP_9_RELEASE_DOCS/` |

Drop dirs are stamped from `main/drops/_TEMPLATE/` by the orchestrator at Phase 1 start. They do not exist until the drop begins.

### Expected Decomposition (planner refines per drop)

```
DROP_0 — Bootstrap (done, out-of-band)
  • GH repo created
  • CLAUDE.md mirrored at bare-root + main/, WIKI.md, README.md, LICENSE, .gitignore landed

DROP_1 — Code scaffold + mage + CI
  1.1 Move stashed files into target layout per decision 27: go.mod + go.sum to main/;
      rename flat main.go into main/cmd/rak/main.go + main/cmd/rak/root.go (split fang.Execute
      entry from cobra command construction so 1.3 can rewrite root.go in isolation).
  1.2 Rewrite go.mod module path to github.com/evanmschultz/rak (stash path is
      github.com/evanmschultz/coding_challenges/fang, not fwc).
  1.3 Rewrite root command for rak shape: Use: "rak [path]", Args: cobra.MaximumNArgs(1),
      drop wc-style flags. count(io.Reader) (Counts, error) stays UNEXPORTED + in-file in
      cmd/rak/root.go — Drop 2.1 owns the move into internal/counting and the export to Count
      (first-drop hand-off boundary, pinned here so neither drop re-does it). Wire
      fang.WithNotifySignal(os.Interrupt, syscall.SIGTERM) so cmd.Context() cancels on Ctrl-C
      / SIGTERM (prereq for Drop 8.1 parallel-walk cancellation).
  1.4 Add github.com/magefile/mage dep + go mod tidy (MUST land before 1.5 — magefile.go
      imports github.com/magefile/mage/mg and won't compile until the dep is present).
  1.5 Add magefile.go with 9 targets:
        • build      → go build ./...
        • test       → go test -race ./...
        • format     → gofumpt -l -w .
        • lint       → go vet ./... + golangci-lint run
        • ci         → gofumpt -l . (must be empty) + mage lint + mage test
        • install    → go install ./cmd/rak — DEV-ONLY, agents MUST NOT run
        • run        → go run ./cmd/rak (positional args pass after --)
        • coverage   → go test -race -coverpkg=./internal/... -coverprofile=coverage.out ./...
                       && go tool cover -func=coverage.out — REPORT-ONLY in 1.5; gate flips in 9.3
        • planCheck  → diff main/PLAN.md container titles + states against main/drops/*/
                       directory names + each drop dir's PLAN.md header state; fail if drift
  1.6 Add .github/workflows/ci.yml (mage ci on push/PR; no coverage gate yet — mage coverage
      is local-only report-only until 9.3 flips it on).

DROP_2 — Counting domain + render boundary
  2.1 internal/counting: move count out of cmd/rak/root.go, export as
      Count(io.Reader) (Counts, error) with Counts struct — bytes/lines/words/chars.
  2.2 internal/render: laslig-backed printer, Format{Human,JSON} plumbing.
  2.3 Wire count subcommand (or root) to counting + render.
  2.4 TTY-vs-pipe auto-detect via laslig.
  2.5 Unit tests: counting table tests, render snapshot tests.

DROP_3 — Directory walk + gitignore + depth
  3.1 internal/fileset: WalkDir-based traversal, depth limit, hidden-file skip; File struct
      with Open() (io.ReadCloser, error) + Peek(n int) ([]byte, error).
  3.2 internal/ignore: .gitignore parsing, --include/--exclude globs.
  3.3 Binary file detection via File.Peek(512) (skip by default).
  3.4 Per-dir aggregation wired into render output.

DROP_4 — Language detection + code-aware splits
  4.1 internal/lang: extension map + shebang sniff via File.Peek(512) + simple content heuristic.
  4.2 Blank/comment/code split per detected language.
  4.3 Per-type aggregation in render output.
  4.4 --lang go,rs walk filter (NOT to be confused with 5.3's --as <lang> stream-type flag).

DROP_5 — Stdin pipe behavior
  5.1 Detect stdin is piped vs TTY; on TTY-stdin + no-path, hang and read stdin in wc-parity
      (decision 18).
  5.2 Default wc-parity counts on stream.
  5.3 --as <lang> stream-type assertion — opt in to code-aware counting on a stream
      (separate flag from 4.4's --lang walk filter; decision 24).
  5.4 JSON output in pipe-to-pipe chain.

DROP_6 — Summary + sorting
  6.1 internal/summary: totals row, per-dir rollup, per-type rollup.
  6.2 --sort {lines,files,bytes,tokens,name} with --sort-asc direction flip; default lines desc
      (decision 19).
  • [DEFERRED to v0.2+: tree view, TSV output — decision 26]

DROP_7 — Token counting
  7.1 Add github.com/tiktoken-go/tokenizer dep.
  7.2 internal/tokens: count tokens per file (cl100k_base default, configurable).
  7.3 --tokens flag + output integration.
  7.4 Document approximation caveat in README.

DROP_8 — Perf + UX polish
  • Drop's planner benchmarks rak on its own source + a larger tree; decides whether 8.1
    (parallel walk) is justified per decision 26.
  8.1 Parallel directory walk (bounded worker pool) — CONDITIONAL on planner showing
      >500ms wall-time; cut otherwise.
  8.2 Progress indication on TTY — laslig spinner + "processed N files" counter, trigger when
      wall-time > 250ms (decision 21).
  8.3 --max-files safety rail.
  8.4 --tracked-only via git ls-files.
  8.5 Symlink handling (default don't-follow, --follow flag; decision 20).

DROP_9 — Release + docs
  • Drop's planner finalizes 9.x ordering; notes that 9.5 "flip repo public" is dev-manual
    with no code footprint, so its build-QA is pro-forma (planner documents this explicitly).
  9.1 Fill out README with real examples (replace aspirational usage).
  9.2 Add --version via fang.WithVersion.
  9.3 Flip Drop 1.5's report-only mage coverage into a gate: 70% floor with scope
      -coverpkg=./internal/... (excludes cmd/rak), enforced in mage ci + CI workflow
      (decision 22).
  9.4 GoReleaser config for binary releases + local dry-run (goreleaser release --snapshot).
  9.5 Flip repo public (dev-manual GitHub action; build-QA verifies repo visibility +
      CI still green).
  9.6 Tag v0.1.0 and push tag (triggers GoReleaser in CI).
```

## Immediate Next Step

The Drop 1 dir does not yet exist. Next session is a **work orchestrator launched from `main/`** (not a steward from the bare-root) that runs Phase 1 of WORKFLOW.md against Drop 1:

1. `cd /Users/evanschultz/Documents/Code/hylla/rak/main && claude`.
2. Read `main/WIKI.md`, `main/CLAUDE.md`, `main/PLAN.md`, `main/drops/WORKFLOW.md` on cold-start (CLAUDE.md auto-loads; the others do not).
3. Stamp drop dir: copy `main/drops/_TEMPLATE/` → `main/drops/DROP_1_CODE_SCAFFOLD_MAGE_CI/`. Set its `PLAN.md` header `state: planning`. Commit (`docs(drop-1): scaffold drop dir from template`).
4. Spawn `go-planning-agent` per WORKFLOW.md § "Phase 1 — Plan" with the paradigm-override directive (see WORKFLOW.md § "Subagent Spawn Prompts"). Planner decomposes Drop 1 into the six expected units (1.1–1.6 above), each with `paths` / `packages` / `acceptance` / `blocked_by`.
5. Continue through Phase 2 (plan-QA), Phase 3 (discuss + cleanup), looping until plan accepted, then Phases 4–7 unit by unit.

## Follow-Ups / Outstanding Orchestration Tasks

Items tracked for future sessions, separate from the Drop 0–9 hierarchy:

- **Laslig progress-bar follow-up** — when rak's perf drop (Drop 8) genuinely needs a known-total progress bar, dev extends laslig upstream (not rak). Until then, spinner + "processed N files" counter is enough.
- **Pin `gofumpt` + `golangci-lint` versions in Drop 9** — Drop 1.6's CI workflow installs `gofumpt` + `golangci-lint` without version pins, relying on `actions/setup-go` + latest-tag semantics. Surfaced by Drop 1 plan-QA falsification (C4) as a real CI-vs-local drift risk. Defer to Drop 9 (release polish) and pin both tools via `go.mod` `tool` directives or a `go run` pinned-version invocation.

## Stashed Legacy Files

From the prior `fwc` prototype at `/Users/evanschultz/Documents/Code/hylla/rak/` (pre-bootstrap):

- `/tmp/rak-stash/main.go` — cobra+fang single-file wc-style CLI. Carried over in Drop 1.1; Drop 1.3 rewrites the root command for rak's shape but preserves the `count(io.Reader) (Counts, error)` primitive for Drop 2.1 reuse.
- `/tmp/rak-stash/go.mod` — module path is `github.com/evanmschultz/coding_challenges/fang` (not `fwc`). Drop 1.2 rewrites it to `github.com/evanmschultz/rak`.
- `/tmp/rak-stash/go.sum` — kept as-is for now.
- `/tmp/rak-stash/PLAN.md` — obsolete `fwc` plan. Useful phasing notes folded into the drop hierarchy above; not copied into `main/`.
- `/tmp/rak-stash/test.txt` — 342KB coding-challenge fixture. Not copied into `main/`; rak will generate its own test corpus.

Entire `/tmp/rak-stash/` directory is deleted in Drop 1 closeout.
