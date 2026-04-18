# DROP_1 — Builder Worklog

Append a `## Unit 1.M — Round K` section per build attempt. See `main/drops/WORKFLOW.md` § "Phase 4 — Build (per unit)" for what each section should contain.

## Unit 1.1 — Round 1

### Files touched

- `main/go.mod` (new; copied verbatim from `/tmp/rak-stash/go.mod`; line 1 still reads `module github.com/evanmschultz/coding_challenges/fang` — 1.2 rewrites).
- `main/go.sum` (new; copied verbatim from `/tmp/rak-stash/go.sum`; 1.4 will prune via `go mod tidy`).
- `main/cmd/rak/main.go` (new; 15 LOC).
- `main/cmd/rak/root.go` (new; 177 LOC).
- `main/drops/DROP_1_CODE_SCAFFOLD_MAGE_CI/PLAN.md` (Unit 1.1 `State: todo → in_progress → done`).

### Split

Stash `/tmp/rak-stash/main.go` (187 LOC flat file) split into two files under `main/cmd/rak/`:

- **`main.go`** keeps `package main`, the package doc comment (`// Package main implements a Fang/Cobra version of a small wc-style CLI.`), imports `context` / `os` / `github.com/charmbracelet/fang`, and only `func main()` with the exact body required by acceptance: `if err := fang.Execute(context.Background(), newRootCmd()); err != nil { os.Exit(1) }`. 15 LOC total (≤ ~30 target).
- **`root.go`** holds the rest verbatim: `type Config struct`, `type Counts struct`, `newRootCmd()`, `configFromCommand`, `run`, `count`, `printCounts`. Imports rebalanced so this file only pulls what it uses (`bufio`, `fmt`, `io`, `os`, `strings`, `unicode`, `github.com/spf13/cobra`); `context` and `fang` moved to `main.go` since `root.go` no longer references them directly in the Drop 1.1 state.
- Root command shape intentionally NOT rewritten: `Use: "fwc [file]"`, `Args: cobra.ExactArgs(1)`, wc flags `-b`/`-l`/`-w`/`-c` all preserved. 1.3 rewrites the command surface; 2.1 lifts `count` + `Counts` into `internal/counting`.

### Acceptance verification (grep-based, per unit rules)

- `wc -l main/cmd/rak/main.go` → 15 (≤ ~30).
- `grep -rn 'func main' main/cmd/rak/` → exactly one hit: `main.go:11`.
- `grep -rn 'func count(' main/cmd/rak/` → exactly one hit: `root.go:116` (unexported).
- `head -1 main/go.mod` → `module github.com/evanmschultz/coding_challenges/fang` (unchanged, as required; 1.2 rewrites).
- `main/go.mod` + `main/go.sum` present at `main/` level, not duplicated elsewhere.
- `ls main/internal` → does not exist (no `internal/*` created).
- `/tmp/rak-stash/` still has all five files with original mtimes (verified via `ls -la`).
- `root.go` contains `type Config`, `type Counts`, `newRootCmd`, `configFromCommand`, `run`, `count`, `printCounts` (grep-confirmed on `^(func |type )`).

No `mage` / `go build` / `go test` / `go vet` run — per unit description and `main/CLAUDE.md` § "Build Verification" the magefile doesn't exist yet and raw go commands are forbidden; compile verification defers to Unit 1.5.

### Surprises

None. The split was mechanical. The stash `main.go` package-doc-comment belongs with `package main` either way — keeping it on `main.go` felt most natural since that's where the entry point lives.

### Hylla Feedback

N/A — this was pure file-move / file-split work on non-ingested stash files. No Hylla query made, no fallback forced.

## Unit 1.2 — Round 1

### Files touched

- `main/go.mod` (line 1 rewritten: `module github.com/evanmschultz/coding_challenges/fang` → `module github.com/evanmschultz/rak`; all other lines unchanged — Go version directive and entire `require` block preserved byte-for-byte).
- `main/drops/DROP_1_CODE_SCAFFOLD_MAGE_CI/PLAN.md` (Unit 1.2 `State: todo → in_progress`; NOT flipped to `done` — see "Surprises").

### Intent

One-line module-path rewrite. `go.sum` intentionally untouched (1.4's job per Notes "go.sum drift"). No `.go` source edits. No `go mod tidy` run.

### Acceptance verification (grep-based, per unit rules)

- `head -1 main/go.mod` → `module github.com/evanmschultz/rak` (exact, as required). PASS.
- `grep -rn 'github.com/evanmschultz/coding_challenges/fang' main/` → **non-zero lines, but all hits are planning/audit prose, not Go imports or go.mod directives.** See "Surprises" below.
- `grep -rn 'github.com/evanmschultz/fwc' main/` → 1 hit, and it is the acceptance bullet itself at `drops/DROP_1_CODE_SCAFFOLD_MAGE_CI/PLAN.md:65` (self-reference). No Go/go.mod hit. See "Surprises" below.

### Surprises

Acceptance criteria as literally stated ("returns zero lines" for both greps) cannot pass because the planning documents and Unit 1.1 audit trail reference the old string in prose:

Residual `coding_challenges/fang` hits after the edit:
1. `main/PLAN.md:83` — narrative: "github.com/evanmschultz/coding_challenges/fang, not fwc".
2. `main/PLAN.md:195` — narrative: describes pre-1.2 stash state.
3. `drops/DROP_1_CODE_SCAFFOLD_MAGE_CI/PLAN.md:46, :50, :64` — drop plan documenting Unit 1.1's stale-module state + the Unit 1.2 acceptance bullet itself.
4. `drops/DROP_1_CODE_SCAFFOLD_MAGE_CI/BUILDER_QA_PROOF.md:21`, `BUILDER_QA_FALSIFICATION.md:20`, `BUILDER_WORKLOG.md:9, :28` (this file) — Unit 1.1 durable audit records confirming line 1 was still stale at 1.1 exit.

Residual `fwc` hit: only `drops/DROP_1_CODE_SCAFFOLD_MAGE_CI/PLAN.md:65` — the acceptance bullet itself.

Intent of the criteria ("no nested import path remnants" / "guards against accidental new ones" / guard against mis-named `fwc` path) is satisfied: no `.go` file, no `go.mod` require/replace, and no import statement in the tree contains either string. Only prose in markdown docs does.

Options for the orchestrator to pick from:
- **(a)** Accept the literal acceptance as unfulfillable-by-design for the `_QA_*.md` + `BUILDER_WORKLOG.md` audit rows (those are durable records of past state) and `main/PLAN.md` narrative; treat "no lines in Go/go.mod" as the real check. Flip state to `done` on that revised basis.
- **(b)** Amend the acceptance bullet on drop PLAN.md line 64–65 to exclude markdown docs (e.g. `grep --include='*.go' --include='go.mod'` or `grep ... main/ | grep -v '\.md:'`) — a planner/plan-QA concern, not a builder one.
- **(c)** Redact the old-module-path prose across all the listed .md files so the literal grep returns zero. Plausible but high-churn and overwrites audit trail.

State left `in_progress`. Not flipping `done` until orchestrator rules on which option to take.

### Hylla Feedback

N/A — one-line `go.mod` edit; no Go-semantic lookup needed. Hylla indexes Go files only and would not cover `go.mod` regardless.

## Unit 1.2 — Round 2

### Files touched

- `main/drops/DROP_1_CODE_SCAFFOLD_MAGE_CI/PLAN.md` (Unit 1.2 `State: in_progress → done`).
- `main/drops/DROP_1_CODE_SCAFFOLD_MAGE_CI/BUILDER_WORKLOG.md` (this append — Round 2 section).

No Go / `go.mod` / `go.sum` edits. Round 1 already landed the module-path rewrite correctly; this round is verification-only against the sharpened acceptance wording the orchestrator committed (rak commit `e73e98a`, which scoped the two grep bullets to `--include='*.go' --include='go.mod' --include='go.sum'`).

### Intent

Re-verify Unit 1.2's three acceptance bullets now that the wording matches the real invariant domain (code + module files, not markdown prose). If all three pass, flip state to `done`.

### Acceptance verification (post-wording-fix)

- **Bullet 1 — `main/go.mod` line 1 is exactly `module github.com/evanmschultz/rak`.** PASS. Confirmed by reading `main/go.mod` directly: line 1 is `module github.com/evanmschultz/rak` (no trailing whitespace, no CRLF artifact).
- **Bullet 2 — `grep -rn 'github.com/evanmschultz/coding_challenges/fang' main/ --include='*.go' --include='go.mod' --include='go.sum'` returns zero lines.** PASS. Equivalent ripgrep check via the `Grep` tool (pattern `github\.com/evanmschultz/coding_challenges/fang`, path `main/`, glob `*.{go,mod,sum}`) returned "No matches found".
- **Bullet 3 — `grep -rn 'github.com/evanmschultz/fwc' main/ --include='*.go' --include='go.mod' --include='go.sum'` returns zero lines.** PASS. Equivalent ripgrep check via the `Grep` tool (pattern `github\.com/evanmschultz/fwc`, path `main/`, glob `*.{go,mod,sum}`) returned "No matches found".
- **Bullet 4 — compile verification deferred to Unit 1.5.** Not actionable here; noted only for completeness.

All three grep-based bullets green. State flipped `in_progress → done` in drop `PLAN.md`.

### Surprises

None. The Round 1 edit was correct from the start; the pause was a specification-wording issue on the acceptance criteria, not a code issue. The sharpened wording (commit `e73e98a`) scopes the invariant to the domain where it actually matters (Go source + module files), which makes the bullets satisfiable without rewriting history or self-referentially flagging the acceptance bullet itself.

### Hylla Feedback

N/A — verification-only round against `go.mod` + scoped greps. Hylla indexes Go files only and would not cover `go.mod` regardless; no Go-semantic lookup needed.
