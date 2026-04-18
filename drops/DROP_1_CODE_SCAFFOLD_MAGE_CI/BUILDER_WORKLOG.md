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
