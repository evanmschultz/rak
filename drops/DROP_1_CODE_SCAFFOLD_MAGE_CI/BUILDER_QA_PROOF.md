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

## Unit 1.2 — Round 1

**Verdict:** pass

**Evidence (per acceptance bullet, against current drop PLAN.md Unit 1.2 row lines 57–67, sharpened by rak commit `e73e98a`):**

1. **`main/go.mod` line 1 is exactly `module github.com/evanmschultz/rak`.** PASS. Read of `main/go.mod` line 1 returns exactly `module github.com/evanmschultz/rak` (no trailing whitespace, no CRLF). Confirmed a second time via `Grep` with pattern `^module ` scoped to `main/go.mod` — single match: `1:module github.com/evanmschultz/rak`. Confirmed a third time via `git show HEAD:go.mod` first line = `module github.com/evanmschultz/rak`.

2. **`grep -rn 'github.com/evanmschultz/coding_challenges/fang' main/ --include='*.go' --include='go.mod' --include='go.sum'` returns zero lines.** PASS. Equivalent ripgrep check via `Grep` tool — pattern `github\.com/evanmschultz/coding_challenges/fang`, path `main/`, glob `*.{go,mod,sum}` — returned "No matches found". The scoped-grep form is the authoritative wording post-commit `e73e98a`; the stale-module-path hits in markdown prose (`PLAN.md`, `BUILDER_WORKLOG.md`, `BUILDER_QA_*.md`) are out of scope by design.

3. **`grep -rn 'github.com/evanmschultz/fwc' main/ --include='*.go' --include='go.mod' --include='go.sum'` returns zero lines.** PASS. Equivalent ripgrep check via `Grep` tool — pattern `github\.com/evanmschultz/fwc`, path `main/`, glob `*.{go,mod,sum}` — returned "No matches found".

4. **Compile verification deferred to Unit 1.5 per the bullet's own wording.** Not actionable here; noted for completeness.

**Cross-checks:**

- `git diff HEAD~2 HEAD --name-only` from `main/` returns exactly three files: `drops/DROP_1_CODE_SCAFFOLD_MAGE_CI/BUILDER_WORKLOG.md`, `drops/DROP_1_CODE_SCAFFOLD_MAGE_CI/PLAN.md`, `go.mod`. No `.go` file edited, no `go.sum` edited. Matches the unit contract ("edits go.mod only; no Go source edits" and "go.sum diff is 1.4's job"). PASS.
- `git diff HEAD~2 HEAD -- go.mod` shows exactly the one-line change `-module github.com/evanmschultz/coding_challenges/fang` → `+module github.com/evanmschultz/rak`; the Go version directive and entire `require` block are preserved byte-for-byte. PASS.
- `git diff HEAD~2 HEAD -- go.sum` returns empty output — `go.sum` was not touched by Unit 1.2. Matches Notes "go.sum drift" (deferred to Unit 1.4's `go mod tidy`). PASS.
- Positive-hit check: `Grep` for `github\.com/evanmschultz/rak` scoped to `*.{go,mod,sum}` in `main/` returns exactly one match — `go.mod:1:module github.com/evanmschultz/rak`. The new module string exists exactly where acceptance requires and nowhere else in code/module files. PASS.
- HEAD is `aab971e chore(gomod): rewrite module path to github.com/evanmschultz/rak` as declared by the spawn prompt. `git log --oneline` confirms Unit 1.2's single code-surface commit is `aab971e`; the intermediate commit in the HEAD~2..HEAD range is `e73e98a docs(drop-1): scope unit 1.2 acceptance greps to code files`, which is the acceptance-wording sharpening, not a code change. PASS.
- BUILDER_WORKLOG.md Unit 1.2 Round 2 section honestly records the state flip (`in_progress → done`) and the post-wording-fix grep re-run. Round 1's "Surprises" section preserves the earlier unfulfillable-literal-grep finding, so the audit trail captures the full two-round sequence. PASS.
- Drop `PLAN.md` Unit 1.2 row shows `State: done` (line 59). PASS.
- gopls `documentSymbol` on `main/cmd/rak/main.go` returns `main (Function) func() - Line 11` with no diagnostics surfaced — package resolves cleanly under the new `github.com/evanmschultz/rak` module path. (Not acceptance-required for 1.2 but a low-cost sanity check.) PASS.

**Findings:** none.

### Hylla Feedback

N/A — Unit 1.2 is a `go.mod`-only edit. Hylla today indexes Go files only (non-Go artifacts like `go.mod` are out of Hylla's scope per `main/CLAUDE.md` § "Hylla Baseline"), so no Hylla query was warranted and no fallback miss occurred. Acceptance is grep-based and `git diff`-backed.

## Unit 1.3 — Round 1

**Verdict:** pass

**Evidence (per acceptance bullet, drop PLAN.md Unit 1.3 row lines 69–84):**

1. **`newRootCmd()` shape.** Read `main/cmd/rak/root.go` lines 21–39. Returns `*cobra.Command` literal with `Use: "rak [path]"` (line 23), `Args: cobra.MaximumNArgs(1)` (line 30), `Short` (line 24, "Summarize code in a directory: line, word, and token counts by language"), `Long` (lines 25–29, two-paragraph rak description + Drop 1 caveat), and a `RunE` (lines 31–37) whose body is `_ = c.Context()` followed by `return fmt.Errorf("not implemented — see drop 2")`. The `_ = c.Context()` call + doc-comment block (lines 32–34) threads command-scoped cancellation forward; the immediate `return` guarantees no panic. LSP `documentSymbol` confirms `newRootCmd (Function) func() *cobra.Command - Line 21`. PASS.

2. **No wc-style flags.** `Grep` for `BoolP` scoped to `main/cmd/rak/root.go` → 0 matches. All four stash `root.Flags().BoolP(...)` calls for `-b`/`-l`/`-w`/`-c` are gone (stash `/tmp/rak-stash/main.go` lines 57–60 had them). PASS.

3. **`count` + `Counts` preserved, unexported.** `Grep` for `func count(` → exactly one match: `root.go:42:func count(r io.Reader) (Counts, error) {`. `Grep` for `func Count(` → 0 matches (no accidental export). `Grep` for `type Counts struct` → exactly one match: `root.go:13:type Counts struct {`. Byte-compare against stash:
   - `diff` of current `root.go` lines 41–78 (doc comment + `count` body) against stash `main.go` lines 124–161 → **IDENTICAL** (no shape drift, no rename, no signature change).
   - `diff` of current `root.go` lines 12–18 (doc comment + `Counts` struct) against stash `main.go` lines 26–32 → **IDENTICAL**.
   Drop 2.1 hand-off boundary (pinned in drop PLAN.md Notes) survives intact. PASS.

4. **RunE error string with UTF-8 em dash.** `Grep` for literal `not implemented — see drop 2` (em dash U+2014) → exactly one match: `root.go:36:			return fmt.Errorf("not implemented — see drop 2")`. Em dash verification: `Grep` for `not implemented -- see` (ASCII double-hyphen) → 0 matches, so no `--` substitution slipped in. The em-dash literal matches only when the source file actually contains UTF-8 bytes `0xE2 0x80 0x94` at that position — ripgrep's match proves the em dash is present. PASS.

5. **`fang.Execute` shape in `main.go`.** Read `main/cmd/rak/main.go` lines 12–20. Exact call:
   ```go
   if err := fang.Execute(
       context.Background(),
       newRootCmd(),
       fang.WithNotifySignal(os.Interrupt, syscall.SIGTERM),
   ); err != nil {
       os.Exit(1)
   }
   ```
   `Grep` for `WithNotifySignal` → 1 match: `main.go:16:		fang.WithNotifySignal(os.Interrupt, syscall.SIGTERM),`. `Grep` for `syscall.SIGTERM` → 1 match on the same line. PASS.

6. **`main.go` imports.** Read `main/cmd/rak/main.go` lines 4–10. Import block contains exactly `context` (line 5), `os` (line 6), `syscall` (line 7), `github.com/charmbracelet/fang` (line 9). `Grep` for `spf13/cobra` scoped to `main.go` → 0 matches (cobra import lives only in `root.go`). PASS.

7. **`c.Context()` threading.** `root.go` line 35 has `_ = c.Context()` preceded by a three-line doc comment (lines 32–34) explaining the forward-looking intent. The stub itself returns immediately on line 36, which the acceptance bullet explicitly permits ("For the Drop 1 stub this is a forward-looking constraint on the file shape; the stub itself returns immediately"). No `context.Background()` invention inside `RunE`. PASS.

8. **Obsolete helpers deleted; hand-off boundary intact.** LSP `documentSymbol` on `main/cmd/rak/root.go` returned exactly three top-level symbols: `Counts` (Struct, Line 13), `newRootCmd` (Function, Line 21), `count` (Function, Line 42). No `Config`, `configFromCommand`, `run`, `printCounts` — all four obsolete helpers present in stash `main.go` (lines 17–24, 65–102, 104–122, 163–186) are deleted from current `root.go`. `count` + `Counts` retained verbatim (see bullet 3 byte-compare). PASS.

9. **`root.go` ≤ ~150 LOC.** `wc -l main/cmd/rak/root.go` → 78 LOC (target ≤ ~150). Builder's BUILDER_WORKLOG.md report of 78 LOC verified. PASS.

**Cross-checks:**

- **`count(io.Reader) (Counts, error)` signature verbatim against stash.** Byte-compare (`diff`) of current `root.go` lines 41–78 against stash `/tmp/rak-stash/main.go` lines 124–161 → IDENTICAL. Signature, body, comments, whitespace all byte-for-byte unchanged. PASS.
- **`Counts` struct fields verbatim against stash.** Byte-compare of current `root.go` lines 12–18 against stash lines 26–32 → IDENTICAL. `Bytes`, `Lines`, `Words`, `Chars` all `int64`; field order preserved. PASS.
- **`main.go` ≤ ~30 LOC.** `wc -l main/cmd/rak/main.go` → 20 LOC (target ≤ ~30). Builder's report of 20 LOC verified. PASS.
- **Em dash is UTF-8 U+2014, not ASCII `--`.** Confirmed by (a) ripgrep match on the literal em-dash pattern and (b) zero match for the ASCII `--` variant. The same character appears in the orchestrator-facing spawn prompt and rak `main/PLAN.md` Decision 29, so the canonical string is em-dash throughout. PASS.
- **`main.go` package doc comment updated.** Line 1: `// Package main implements the rak CLI entry point.` — replaces the stashed wc-style phrasing, appropriate for the post-1.3 command surface. Not an acceptance bullet, but a coherent follow-through. PASS.
- **LSP intra-package resolution clean.** `documentSymbol` on `main.go` returns exactly one top-level symbol (`main (Function) func() - Line 12`); `documentSymbol` on `root.go` returns the three-symbol set noted above. No gopls diagnostics surfaced against the 1.3 surface (the pre-declared `go.mod` drift and `count` "unused" warnings are out of scope per spawn prompt — explicitly plan-state). PASS.
- **Pre-declared diagnostics audit.** (a) `go.mod` drift (missing `x/exp/golden`, `golang.org/x/exp`; unused laslig transitives; `fang should be direct`) is scheduled for Unit 1.4's `go mod tidy` — confirmed out-of-scope per spawn prompt. (b) `count`/`Counts` "unused" diagnostics are intentional plan-state per Drop 2.1 hand-off pinning — confirmed out-of-scope per spawn prompt. Nothing else surfaced in the reviewed files would qualify as a new Unit 1.3 defect.

**Findings:** none.

### Hylla Feedback

N/A — Unit 1.3 is a within-file rewrite in `cmd/rak/root.go` + a within-file edit in `cmd/rak/main.go`. No cross-package reuse question, no symbol-lookup need against the broader tree — the only external reference the review needed was the stash file (which is filesystem-local, not a Hylla artifact). All evidence was grep/diff/LSP-backed. No Hylla query run, no fallback forced.
