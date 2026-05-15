# Rak — Planning Doc

**Status:** Durable system of record for the overarching drop tree. Each drop's per-phase worklog lives under `main/drops/DROP_N_<NAME>/` — see `main/drops/WORKFLOW.md` for the canonical lifecycle. PLAN.md is updated **after** a drop closes (state flip + any structural changes that came out of the work) and **after** a planner restructures the tree. Not edited mid-build.

**Workflow:** see `main/drops/WORKFLOW.md` for the per-drop lifecycle (plan → plan-QA → discuss → revise → loop → build → build-QA → verify → close). PLAN.md owns the overarching plan; WORKFLOW.md owns the phase mechanics; `main/CLAUDE.md` owns role boundaries + Go quality rules.

## Project Intent

**Rak** — a fast project-sizing CLI for counting code, **built for LLMs by default**. Walk a directory (or single file, or stdin), produce accurate counts with language-aware breakdowns, count what's actually in the shared git repo (via `git ls-files` when in a repo), emit TOON output by default for LLM consumption; `--human` and `--json` are opt-ins. Decision 30 (2026-05-14) locks this scope refit: tokens, spinner, parallel walk, `--follow`, and GoReleaser are all deferred to v0.2.

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
9. **Pipe semantics**: on stdin, default to `wc`-parity (lines / words / bytes / chars). `--lang` is a separate walk-filter flag (decision 24). `--as <lang>` (stream-type assertion) is **cut from v0.1.0** per decision 30.
10. **Ignore by default**: respect `.gitignore`. Escape hatches: `--no-gitignore`, `--include`/`--exclude` globs, `--hidden`, `--tracked-only` (backed by `git ls-files`). Binary files skipped by default; `--binary` to count separately.
11. **Token counting**: **deferred to v0.2 per decision 30.** If revived, target `github.com/tiktoken-go/tokenizer` (pure Go, `cl100k_base` default) and document the Claude-approximation caveat.
12. **Depth control**: `--depth N`. Also `--max-files` as a safety rail.
13. **Skip for v1 (YAGNI)**: cyclomatic complexity, churn, diff-vs-ref, caching. **Plus the decision-30 cuts**: token counting, spinner / progress indication, parallel walk, `--follow` symlinks, GoReleaser binary releases.
14. **Progress bar / spinner**: **deferred to v0.2 per decision 30.** No progress indication in v0.1.0. When revived, dev will add a known-total `Progress` primitive to laslig upstream rather than bolting one into rak.
15. **Orchestration**: never-edits-Go rule applies from the first builder drop onward. Bootstrap commit is docs + license + `.gitignore` only — no Go.
16. **Coordination model**: drop = directory under `main/drops/` (e.g. `main/drops/DROP_1_CODE_SCAFFOLD_MAGE_CI/`). Per-drop lifecycle (planner → plan-QA → discuss → revise → builder → build-QA → verify → close) lives in `main/drops/WORKFLOW.md`. PLAN.md tracks overarching containers + state; per-phase mechanics live in WORKFLOW.md; role boundaries live in `main/CLAUDE.md`. Subagents are global (`~/.claude/agents/`) but are spawned with a paradigm-override directive that tells them to ignore Tillsyn-coupled instructions and follow WORKFLOW.md instead.
17. **Subcommand shape**: single root command — `rak [path]`. No subcommands in v0.1.0. All current flags (`--tokens`, `--lang`, `--as`, `--depth`, `--sort`, `--format`) are orthogonal to the operation, not distinct operations. Subcommands can be added later without breaking the root UX; the reverse is painful.
18. **Stdin behavior**: on TTY-stdin with no path, hang and read stdin in `wc`-parity mode (matches `wc` convention — user terminates with Ctrl-D / EOF). Pipe + no path → `wc`-parity on stream. Pipe + `--as <lang>` → code-aware counting on stream.
19. **Sort default**: `lines desc` for the directory view. `--sort {lines,files,bytes,tokens,name}`; `--sort-asc` flips direction.
20. **Symlinks**: don't-follow by default; `--follow` opts in. Matches `rg` / `fd` convention.
21. **Progress indication**: **deferred to v0.2 per decision 30.** No spinner in v0.1.0; the original "Drop 8 spinner + processed-N counter" plan is cut.
22. **Coverage gate**: no gate in Drop 1 CI — but `mage coverage` lands in Drop 1.5 as a **report-only** target (no threshold) so every subsequent drop sees its current number. This prevents a backfill mountain at Drop 9.3. Scope is `-coverpkg=./internal/...` (excludes `cmd/rak`, which is CLI wiring and would drag the number down). 70% floor flips on in Drop 9.3 before the v0.1.0 tag. Early drops have unstable APIs; locking coverage now creates churn.
23. **CI in first drop**: yes — `.github/workflows/ci.yml` ships in Drop 1.6. Matches tillsyn's discipline.
24. **Language flag split**: `--lang go,rs` = walk filter (which files to include from a tree; Drop 4.4). `--as go` = stdin stream-type assertion (treat the stream as language X; Drop 5.3). Two distinct flags, two distinct semantics. Avoids the mode-implicit CLI trap where one flag changes meaning based on input source.
25. **Drop 3 `fileset.File` contract**: `internal/fileset.File` struct exposes `Open() (io.ReadCloser, error)` and `Peek(n int) ([]byte, error)`. Binary detection (Drop 3.3) and shebang sniff (Drop 4.1) both consume `Peek(512)`. This commits Drop 3's public contract before Drop 4 starts so Drop 4 doesn't duplicate file-open logic — closes the scope crack QA falsification flagged.
26. **v0.1.0 scope cuts (superseded by decision 30 on 2026-05-14)**: original cuts were Drop 6.2 (tree view), Drop 6.4 (TSV output), and Drop 8.1-as-conditional. Decision 30 supersedes — see that decision for the full v0.1.0 cut list. Decision 24's `--lang <csv>` walk filter (now in Drop 5) is still kept.
27. **Architecture**: 7 internal packages + `cmd/rak`, clean layered DAG (no cycles). Leaves: `counting`, `ignore`, `summary`, `tokens`. Mid: `fileset → ignore`, `lang → fileset`, `render → summary`. Root: `cmd/rak`. **No file exceeds ~400 LOC**; one primary type per file; split when the file holds two concepts. Total v0.1.0: ~1,600 non-test LOC + ~1,500 test LOC. Full breakdown in `main/CLAUDE.md` § "Project Structure". Five sub-choices locked: (a) walker uses `iter.Seq2[*File, error]`, (b) `Counts` struct + `Count()` func, (c) `type Language string`, (d) explicit `NewHumanRenderer`/`NewJSONRenderer` (no Format enum factory), (e) `testing/fstest.MapFS` for unit tests + one real `testdata/` integration test in `cmd/rak`.
28. **Quality tooling** (all via mage — agents never run raw tools): `gofumpt` (stricter gofmt, auto-format + format-check), `go vet`, `golangci-lint` (umbrella lint), `go test -race` (race detector always on), coverage gate (70% floor in Drop 9.3). `mage ci` = format-check + lint + test-with-race and is the pre-push gate. 12 Go-idiomatic naming rules live in `main/CLAUDE.md` § "Project Structure" → "Go-Idiomatic Naming Rules".
29. **Concurrency + error idioms**:
    - **Goroutines bounded** — `errgroup.Group` + `SetLimit(n)` or semaphore channel. No unbounded `go func(){}()`.
    - **Context-cancellable** — every goroutine checks `ctx.Done()`. `RunE` threads `cmd.Context()` downward.
    - **`defer` for cleanup** — file `Close()`, mutex `Unlock()`, spinner `Stop()` — always on the line after acquisition.
    - **No shared mutable state without synchronization** — prefer channels for ownership transfer; if `sync.Mutex` is needed, keep it unexported on the owning struct.
    - **Errors wrap with `%w`** at every boundary that adds info. Sentinel errors named `ErrFoo` (e.g. `ErrTooDeep`, `ErrBinaryFile`). Inspect with `errors.Is` / `errors.As` — never string-match. Never swallow (discard to `_` only with a one-line why-comment).
    - Full rules in `main/CLAUDE.md` § "Go Development Rules" → "Concurrency" + "Errors".
30. **Scope refit (2026-05-14)** — rak is `wc++` for **LLM-first consumption**. The v0.1.0 cut list:
    - Token counting → v0.2 (was Drop 7; supersedes decision 11).
    - Spinner / progress indication → v0.2 (supersedes decisions 14, 21).
    - Parallel walk → v0.2 (was Drop 8.1-conditional; supersedes decision 26).
    - `--follow` symlinks → v0.2 (was Drop 8.5).
    - `--as <lang>` stream-type assertion → cut (was Drop 5.3; amends decision 9).
    - GoReleaser binary release → v0.2; v0.1.0 ships via `go install github.com/evanmschultz/rak/cmd/rak@latest` (was Drop 9.4).
    - Inverted: `--tracked-only` flipped from a v0.2-deferred opt-in (was Drop 8.4) to **default behavior** when in a git repo (decision 32).
31. **Cascade tiering (A / B / C)** — every drop dir's `PLAN.md` header carries a `Tier:` field. **A** = full cascade (planner subagent + parallel plan-QA + parallel build-QA per unit). **B** = lite (orch plans inline, no plan-QA; falsification-only build-QA per unit). **C** = orch-direct (no subagents; dev reviews diff). Full mechanics in `main/drops/WORKFLOW.md` § "Cascade Tiering (A / B / C)". Orch sets the tier at Phase 1 stamp time; default = **A** when uncertain.
32. **Default file source** — when `git rev-parse --is-inside-work-tree` succeeds at the walk root, enumerate via `git ls-files --full-name -z` (NUL-delimited, paths relative to repo top via `git rev-parse --show-toplevel`). When not in a git repo, fall back to `internal/fileset.Walker` + `.gitignore` (today's behavior). Today's filter flags (`--include`, `--exclude`, `--depth`, `--hidden`, `--no-gitignore`, `--binary`) apply on top of either source. Lands in Drop 4 via `internal/lister`. Future-mode escape hatches (e.g. `--include-untracked`, `--all-files`) are out of v0.1.0.
33. **Default renderer** — TOON via `github.com/toon-format/toon-go` (official spec-compliant lib, spec v1.0.0). Flags `--human`, `--json`, `--toon` are mutually exclusive booleans; default is TOON regardless of TTY (LLM-first audience). Replaces the Drop 3.5 `--format auto|human|json` flag. Lands in Drop 4.
34. **Lockfiles** — `go.sum`, `package-lock.json`, `yarn.lock`, `pnpm-lock.yaml`, `Cargo.lock`, etc. are counted by default. Whatever git tracks, rak counts ("what's in the repo, shared"). v0.2 may add `--no-lockfiles` as a denylist flag if usage signals it.

## Drop Tree

The plan below is the working shape. Each row is a level_1 container drop. **Each drop's atomic-unit decomposition lives inside its own `main/drops/DROP_N_<NAME>/PLAN.md`**, written by the planner subagent during Phase 1 of `WORKFLOW.md`. The sub-bullets below are the **expected** decomposition — refined and committed per drop when the planner runs.

| Drop | Tier | State | Blocked by | Drop dir |
|---|---|---|---|---|
| `DROP_0_BOOTSTRAP` | — | done | — | (out-of-band; predates this workflow) |
| `DROP_1_CODE_SCAFFOLD_MAGE_CI` | A (hindsight) | done | — | `main/drops/DROP_1_CODE_SCAFFOLD_MAGE_CI/` |
| `DROP_2_COUNTING_DOMAIN_RENDER_BOUNDARY` | A (hindsight) | done | DROP_1 | `main/drops/DROP_2_COUNTING_DOMAIN_RENDER_BOUNDARY/` |
| `DROP_3_DIRECTORY_WALK_GITIGNORE_DEPTH` | A (hindsight) | done | DROP_2 | `main/drops/DROP_3_DIRECTORY_WALK_GITIGNORE_DEPTH/` |
| `DROP_4_DEFAULT_BEHAVIOR_TRACKED_TOON` | A | todo | DROP_3 | `main/drops/DROP_4_DEFAULT_BEHAVIOR_TRACKED_TOON/` |
| `DROP_5_LANGUAGE_DETECTION_CODE_SPLITS` | A | todo | DROP_4 | `main/drops/DROP_5_LANGUAGE_DETECTION_CODE_SPLITS/` |
| `DROP_6_STDIN_PIPE_BEHAVIOR` | C | todo (no-op close expected) | DROP_5 | `main/drops/DROP_6_STDIN_PIPE_BEHAVIOR/` |
| `DROP_7_SUMMARY_SORTING` | A | todo | DROP_6 | `main/drops/DROP_7_SUMMARY_SORTING/` |
| `DROP_8_SAFETY_RAILS` | B | todo | DROP_7 | `main/drops/DROP_8_SAFETY_RAILS/` |
| `DROP_9_RELEASE_DOCS` | B (mixed; 9.4 + 9.5 are C) | todo | DROP_8 | `main/drops/DROP_9_RELEASE_DOCS/` |

### Deferred to v0.2

| Drop | Why | Original slot |
|---|---|---|
| `TOKEN_COUNTING` | Decision 30 — rak is wc++, not LLM-helper; tokens are not core | was DROP_7 |
| `PERF_PARALLEL_WALK` | Decision 30 — no perf evidence justifies pre-emptive concurrency | was DROP_8.1 |
| `SPINNER_PROGRESS_INDICATION` | Decision 30 — no UX polish in v0.1.0; dev will add laslig `Progress` primitive upstream first | was DROP_8.2 |
| `SYMLINK_FOLLOW` | Decision 30 — default don't-follow is correct; opt-in deferred | was DROP_8.5 |
| `GORELEASER_BINARIES` | Decision 30 — `go install …/cmd/rak@latest` suffices for v0.1.0 | was DROP_9.4 |

Drop dirs are stamped from `main/drops/_TEMPLATE/` by the orchestrator at Phase 1 start. They do not exist until the drop begins.

### Expected Decomposition (planner refines per drop)

```
DROP_0 — Bootstrap (done, out-of-band)
  • GH repo created
  • CLAUDE.md mirrored at bare-root + main/, README.md, LICENSE, .gitignore landed

DROP_1 — Code scaffold + mage + CI  (DONE — historical record)
  1.1 Move stashed files into target layout (go.mod to main/; cmd/rak/main.go + root.go split).
  1.2 Rewrite go.mod module path to github.com/evanmschultz/rak.
  1.3 Rewrite root command for rak shape (Use: "rak [path]", Args: cobra.MaximumNArgs(1)).
  1.4 Add github.com/magefile/mage dep.
  1.5 Add magefile.go with the standard 9 targets (build/test/format/lint/ci/install/run/
      coverage/planCheck).
  1.6 Add .github/workflows/ci.yml (mage ci on push/PR).

DROP_2 — Counting domain + render boundary  (DONE — historical record)
  2.1 internal/counting: Count(io.Reader) (Counts, error) with Counts struct.
  2.2 internal/render: laslig-backed printer, Format{Human,JSON} plumbing.
  2.3 Wire root command to counting + render (stdin path).
  2.4 TTY-vs-pipe auto-detect via laslig.
  2.5 Unit tests.

DROP_3 — Directory walk + gitignore + depth  (DONE — closed 2026-05-15)
  3.0 mage addDep for gitignore + glob deps.
  3.1 internal/ignore: Matcher interface + gitignore + --include/--exclude globs.
  3.2 internal/fileset.File: Open + Peek + IsHidden.
  3.3 internal/fileset.Walker: iter.Seq2[*File, error] emission.
  3.4 Binary file detection via Peek(512) + ErrBinaryFile sentinel.
  3.5 cmd/rak wire-up: --depth/--hidden/--no-gitignore/--binary/--include/--exclude flags +
      per-dir aggregation.

DROP_4 — Default behavior: tracked-only source + TOON renderer  (NEW; tier A)
  • Pulls forward decisions 32 + 33: when in a git repo, count what git tracks; emit TOON by
    default for LLM consumption.
  4.0 mage addDep github.com/toon-format/toon-go.
  4.1 internal/lister: FileLister interface + Detect(root, fsys) (FileLister, error) factory.
  4.2 internal/lister.GitLister: shells `git rev-parse --is-inside-work-tree` for detection
      and `git ls-files --full-name -z` (relative to `git rev-parse --show-toplevel`) for
      enumeration; threads ctx via exec.CommandContext. Emits *fileset.File values.
  4.3 internal/lister.WalkLister: thin adapter over the existing internal/fileset.Walker so
      Drop 3's walk path stays valid for the not-in-git fallback.
  4.4 cmd/rak rewire: replace direct fileset.Walker construction with lister.Detect; replace
      --format auto|human|json with mutually exclusive boolean flags --human / --json / --toon
      (default = TOON regardless of TTY). Update root_test.go flag-parsing cases.
  4.5 internal/render.NewTOONRenderer: emit TOON for both Render(counts) (stream mode) and
      RenderTree(dirs, total, errs) (path mode) via toon-format/toon-go. Snapshot tests in
      render_test.go.

DROP_5 — Language detection + code-aware splits  (was DROP_4; tier A)
  5.1 internal/lang: extension map + shebang sniff via File.Peek(512) + simple content heuristic.
  5.2 Blank/comment/code split per detected language.
  5.3 Per-type aggregation in render output (all three renderers).
  5.4 --lang go,rs walk filter (decision 24).

DROP_6 — Stdin pipe behavior  (was DROP_5; tier C — expected no-op close)
  • Drop 2 already handles stdin counting via the no-args path; this drop only verifies
    pipe-detection edges and ratifies that decisions 9/30 cuts (no --as, no TTY-hang) are real.
  6.1 Confirm pipe-vs-TTY detection works; document in README scope notes.
  6.2 (already shipped in Drop 2) wc-parity counts on stream.
  6.3 (already shipped in Drop 2) TOON/JSON/human output in pipe-to-pipe chain.
  • [CUT: 5.1 TTY-hang on no-path-no-stdin and 5.3 --as <lang> — decision 30]

DROP_7 — Summary + sorting  (was DROP_6; tier A)
  7.1 internal/summary: Summary struct (totals, per-dir rollup, per-type rollup); migrate
      render.Directory (Drop 3.5 provisional per C8) into summary.Summary.
  7.2 --sort {lines,files,bytes,name} with --sort-asc direction flip; default lines desc
      (decision 19). Note: `tokens` is NOT a sort key in v0.1.0 (decision 30 defer).
  • [DEFERRED to v0.2+: tree view, TSV output — decision 26]

DROP_8 — Safety rails  (was DROP_8; tier B; slimmed per decision 30)
  8.1 --max-files safety rail (orchestrator aborts with wrapped error when crossed).
  • [CUT per decision 30: parallel walk (was 8.1), spinner / progress indication (was 8.2),
    --tracked-only as opt-in (now default per decision 32), --follow symlinks (was 8.5)]

DROP_9 — Release + docs  (slimmed per decision 30; mixed tier)
  9.1 (tier B) Fill out README with real examples — replace aspirational usage; document
      git-tracked-only default, TOON-default, --human/--json opt-ins, decision-30 cuts.
  9.2 (tier B) Add --version via fang.WithVersion.
  9.3 (tier B) Flip Drop 1.5's report-only `mage coverage` into a gate: 70% floor with scope
      -coverpkg=./internal/... (excludes cmd/rak), enforced in `mage ci` + CI workflow
      (decision 22).
  9.4 (tier C) Flip repo public (dev-manual GitHub action; orch verifies repo visibility +
      CI still green; no Go code).
  9.5 (tier C) Tag v0.1.0 and push tag.
  • [CUT per decision 30: GoReleaser config (was 9.4) — `go install` is the v0.1.0 install path]
```

## Immediate Next Step

Drops 0/1/2/3 are done (Drop 3 closed 2026-05-15 at commit `cf021ac`, CI run 25898996914, Hylla task `task-1bbf641644105060`). The orchestrator's next moves, in order:

1. **Stamp Drop 4**: copy `main/drops/_TEMPLATE/` → `main/drops/DROP_4_DEFAULT_BEHAVIOR_TRACKED_TOON/`. Set its `PLAN.md` header `state: planning`, `Tier: A`. Commit (`docs(drop-4): scaffold drop dir from template`).
2. **Spawn `go-planning-agent`** per WORKFLOW.md § "Phase 1 — Plan" with the paradigm-override preamble. Planner decomposes Drop 4 into the six expected units (4.0–4.5 in the Drop Tree section above), each with `paths` / `packages` / `acceptance` / `blocked_by`.
3. Continue through Phase 2 (parallel plan-QA — Drop 4 is tier A), Phase 3 (discuss + cleanup), looping until plan accepted, then Phases 4–7 unit by unit.

## Follow-Ups / Outstanding Orchestration Tasks

Items tracked for future sessions, separate from the Drop 0–9 hierarchy:

- **Laslig progress-bar follow-up (v0.2)** — when rak revives progress indication, dev extends laslig upstream with a known-total `Progress` primitive first, then rak consumes it. Not in v0.1.0 per decision 30.
- **Pin `gofumpt` + `golangci-lint` versions in Drop 9** — Drop 1.6's CI workflow installs both without version pins, relying on `actions/setup-go` + latest-tag semantics. Surfaced by Drop 1 plan-QA falsification (C4) as a real CI-vs-local drift risk. Defer to Drop 9 (release polish) and pin via `go.mod` `tool` directives or a pinned `go run` invocation.
- **Drop 3 close docs update** — `main/CLAUDE.md` § "Project Structure" → "File Breakdown" was updated in the 2026-05-14 scope-refit commit; verify the `internal/fileset/file_test.go`, `binary.go`, `binary_test.go` rows are present before flipping Drop 3 to `done`. (This is the O1/O3 follow-up from the drop's PLAN.md.)
- **Drop 4 → Drop 6 cascade ratification** — Drop 6 (stdin pipe behavior) is tier C with an expected no-op close. If Drop 4 changes anything about stdin handling (e.g. `--toon` default applies to stream mode too), revisit Drop 6's no-op claim before stamping.
- **Node.js 20 actions deprecation (CI workflow)** — surfaced during Drop 3 close CI run (run 25898996914, 2026-05-15): `actions/checkout@v4` and `actions/setup-go@v5` currently run on Node.js 20, scheduled for removal from GitHub runners on 2026-09-16. Defer to Drop 9 (release polish) — bump both actions to versions that support Node.js 24, or set `FORCE_JAVASCRIPT_ACTIONS_TO_NODE24=true` on the workflow.

## Stashed Legacy Files

From the prior `fwc` prototype at `/Users/evanschultz/Documents/Code/hylla/rak/` (pre-bootstrap):

- `/tmp/rak-stash/main.go` — cobra+fang single-file wc-style CLI. Carried over in Drop 1.1; Drop 1.3 rewrites the root command for rak's shape but preserves the `count(io.Reader) (Counts, error)` primitive for Drop 2.1 reuse.
- `/tmp/rak-stash/go.mod` — module path is `github.com/evanmschultz/coding_challenges/fang` (not `fwc`). Drop 1.2 rewrites it to `github.com/evanmschultz/rak`.
- `/tmp/rak-stash/go.sum` — kept as-is for now.
- `/tmp/rak-stash/PLAN.md` — obsolete `fwc` plan. Useful phasing notes folded into the drop hierarchy above; not copied into `main/`.
- `/tmp/rak-stash/test.txt` — 342KB coding-challenge fixture. Not copied into `main/`; rak will generate its own test corpus.

Entire `/tmp/rak-stash/` directory is deleted in Drop 1 close.
