# DROP_1 — Build QA Falsification

## Unit 1.1 — Round 1

**Verdict:** pass

**Attack summary:** Probed all 12 appendix attack vectors: (1) hand-off boundary integrity for `count` + `Counts`, (2) root command shape preservation (Use/Args/flags), (3) `main.go` minimalism + no premature `WithNotifySignal`, (4) go.mod module-path staleness vs stash, (5) `/tmp/rak-stash/` lifecycle + no accidental copies of `test.txt` / stash `PLAN.md`, (6) `package main` declaration consistency across both files, (7) `internal/` directory guard, (8) `root.go` 177 LOC vs CLAUDE.md ~150 target, (9) dead-import hunt in both files, (10) build-tag / CRLF / BOM scan, (11) BUILDER_WORKLOG.md fidelity + Hylla Feedback presence, (12) Section 0 leakage into drop artifacts.

Evidence gathered: `git diff HEAD~1 HEAD`, `diff /tmp/rak-stash/go.mod main/go.mod` (zero-byte), `diff /tmp/rak-stash/go.sum main/go.sum` (zero-byte), `diff` of stash main.go body lines 40–186 against root.go lines 31–177 (zero-byte — verbatim lift confirmed), `hexdump` of both new files (ASCII, LF-only, no BOM, no build tag), `wc -l` line counts, `git ls-files` tree inspection (no `internal/`, no `magefile.go`, no `.github/`, no stash files in repo), `grep` sweep for `Section 0` / `SEMI-FORMAL` in drops dir (zero hits).

**Counterexamples found:** none

**Findings:** none

### Attack-by-attack trace

- 1. Hand-off boundary. `count(r io.Reader) (Counts, error)` at `cmd/rak/root.go:116` — lowercase, unexported. `Counts` struct at `cmd/rak/root.go:24-29` has fields `Bytes / Lines / Words / Chars int64` identical to stash lines 27-32. REFUTED.
- 2. Root command shape. `Use: "fwc [file]"` at root.go:34, `cobra.ExactArgs(1)` at root.go:36, `BoolP` calls for `bytes/lines/words/chars` at root.go:48-51. Unchanged from stash. REFUTED.
- 3. `main.go` minimalism. 15 LOC, exactly one `main()` function. Body matches acceptance verbatim. No `WithNotifySignal` or `syscall` reference (grep zero hits) — correctly deferred to 1.3. REFUTED.
- 4. Module path. `head -1 main/go.mod` = `module github.com/evanmschultz/coding_challenges/fang`. `diff` against stash go.mod is empty. Not yet rewritten, as required. REFUTED.
- 5. Stash lifecycle. `Glob /tmp/rak-stash/*` returns all five files (`go.mod`, `go.sum`, `main.go`, `PLAN.md`, `test.txt`). `Grep test.txt|rak-stash main/` returns only documentation references in drop/plan markdown — no copies of stash files into repo. REFUTED.
- 6. Package consistency. Both files declare `package main` in the same directory (root.go:1, main.go:2). REFUTED.
- 7. `internal/` guard. `Glob main/**/internal` returns no matches. `git ls-files` shows no `internal/*` path. REFUTED.
- 8. LOC target. `root.go` is 177 LOC vs CLAUDE.md ~150 target. Stash `main.go` was 186 LOC; after lifting `main()` + package-doc + fang/context imports into `main.go`, the residue is 177. This is the verbatim-lift phase; 1.3's rewrite will delete `Config` wc flags + `configFromCommand` flag-parsing branches + `printCounts` wc formatting, which closes the gap. 1.1 acceptance does not pin `root.go` LOC — only `main.go` ≤ ~30. REFUTED.
- 9. Import hygiene. `main.go` imports `context` (used in `context.Background()`), `os` (used in `os.Exit`), `github.com/charmbracelet/fang` (used in `fang.Execute`) — all three live. `root.go` imports `bufio, fmt, io, os, strings, unicode, github.com/spf13/cobra` — all seven live (verified by grep for each symbol inside the file). No dead imports. REFUTED.
- 10. Build-tag / CRLF / BOM. `hexdump -C` of both files shows LF-only line terminators (`0a`), no `ef bb bf` BOM, no `//go:build` or `// +build` lines (grep zero hits). `file(1)` misidentified `main.go` as "c program text" due to Go's C-like syntax — not a build-tag artifact. REFUTED.
- 11. Worklog fidelity. Worklog reports 15 LOC main.go + 177 LOC root.go — matches `wc -l` exactly. Files-touched list matches `git show --stat HEAD`. Hylla Feedback section present (`N/A` — acceptable per WORKFLOW.md; empty is fine, missing is not). REFUTED.
- 12. Section 0 leakage. `grep 'Section 0|SEMI-FORMAL|Planner pass|Builder pass|QA Proof pass' main/drops/DROP_1_*` returns zero hits. No durable rak artifact contains Section 0 text. REFUTED.

### Hylla Feedback

N/A — no committed `cmd/rak/` state existed prior to this unit; Hylla has no ingest to query for this drop. Falsification evidence came from `git diff`, side-by-side `diff` of stash vs committed files, `hexdump`, `wc -l`, `Grep`, `Glob`, `git ls-files`. Expected zero Hylla surface for a fresh-code drop.
