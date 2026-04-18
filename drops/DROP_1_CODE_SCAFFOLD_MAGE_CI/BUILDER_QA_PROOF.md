# DROP_1 — Build QA Proof

## Unit 1.1 — Round 1

**Verdict:** pass

**Evidence (per acceptance bullet):**

1. `main/cmd/rak/main.go` exists; `wc -l` → 15 LOC (≤ ~30 target). Contains exactly one function `main` (line 11). Body (lines 11–15) is exactly:
   ```go
   func main() {
       if err := fang.Execute(context.Background(), newRootCmd()); err != nil {
           os.Exit(1)
       }
   }
   ```
   Matches the required form literally. `grep 'fang.Execute(context.Background(), newRootCmd())' main/cmd/rak/main.go` returns 1 hit on line 12. Imports are exactly `context`, `os`, `github.com/charmbracelet/fang`. PASS.

2. `main/cmd/rak/root.go` exists; `wc -l` → 177 LOC. `grep '^(func |type )' main/cmd/rak/root.go` returns exactly the required seven declarations: `type Config struct` (line 15), `type Counts struct` (line 24), `func newRootCmd() *cobra.Command` (line 32), `func configFromCommand(...)` (line 57), `func run(...)` (line 96), `func count(...)` (line 116), `func printCounts(...)` (line 155). Root command shape is still the stash-`fwc` form: `Use: "fwc [file]"` (line 34), `Args: cobra.ExactArgs(1)` (line 36), four `BoolP` wc-flag registrations (lines 48–51) — no Drop 1.3 rewrite has happened yet. Body of lines 31–177 compared against stash `/tmp/rak-stash/main.go` lines 40–186 via direct Read — bodies identical (comments, whitespace, logic unchanged). `Config` + `Counts` struct defs at lines 15–32 match stash lines 14–32. Verbatim-move condition holds. PASS.

3. `main/go.mod` + `main/go.sum` present in `main/`. `diff /tmp/rak-stash/go.mod main/go.mod` → identical; `diff /tmp/rak-stash/go.sum main/go.sum` → identical (both copied unmodified, as required). `head -1 main/go.mod` → `module github.com/evanmschultz/coding_challenges/fang` — still stale, as required for 1.1 (1.2 rewrites). Files do not appear anywhere outside `main/` (verified by Glob `main/**/go.{mod,sum}` returning only the two at `main/`). PASS.

4. No `internal/*` directory created. `Glob main/internal/**/*` → no files found. PASS.

5. `grep -rn 'func main' main/cmd/rak/` → exactly one hit (`cmd/rak/main.go:11:func main() {`). PASS.

6. `grep -rn 'func count(' main/cmd/rak/` → exactly one hit (`cmd/rak/root.go:116:func count(r io.Reader) (Counts, error) {`). Lowercase `c` — unexported. `grep 'func Count(' main/cmd/rak/root.go` → zero hits (no accidental export). PASS.

7. `/tmp/rak-stash/` still present with all five original files (`go.mod`, `go.sum`, `main.go`, `PLAN.md`, `test.txt`). Glob confirmed; no deletion occurred in this unit. PASS.

**Cross-checks:**

- Both files declare `package main`: `main/cmd/rak/root.go:1:package main` + `main/cmd/rak/main.go:2:package main` (main.go line 1 is the package doc comment, line 2 is the `package main` clause). PASS.
- `count(io.Reader) (Counts, error)` is unexported, in `root.go`, with `Counts` struct intact — the Drop 2.1 hand-off boundary pinned by PLAN.md Notes survives untouched. PASS.
- No `*_test.go` files in `main/cmd/rak/` (tests out of scope for 1.1). PASS.
- BUILDER_WORKLOG.md Unit 1.1 Round 1 section records files touched (main.go, root.go, go.mod, go.sum, PLAN.md state bump), describes the split, lists acceptance verification, has a `### Hylla Feedback` subsection with an explicit "N/A — pure file-move" note. PASS.
- Imports rebalanced correctly: `main.go` imports `context`, `os`, `fang`; `root.go` imports `bufio`, `fmt`, `io`, `os`, `strings`, `unicode`, `cobra`. No duplicate imports across files, no unused imports visible by inspection. PASS.
- No Hylla invocation this review — Unit 1.1 is a fresh stash-move with no reuse-discovery question. Noted explicitly per spawn prompt.

**Findings:** none.
