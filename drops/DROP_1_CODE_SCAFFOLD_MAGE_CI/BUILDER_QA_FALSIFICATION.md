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

## Unit 1.2 — Round 1

**Verdict:** pass

**Attack summary:** Probed all 10 appendix attack directions against the claim that Unit 1.2 is done. Unit 1.2's footprint is a single-line `main/go.mod` rewrite (commit `aab971e`) plus a state flip + BUILDER_WORKLOG.md Round 1 + Round 2 appends. Attack surface is small; no counterexample constructed.

Evidence gathered: `git show aab971e --stat` + full diff, `git log --oneline -- go.mod main/go.mod`, `git log --oneline -- go.sum main/go.sum`, `git diff b10f5fd aab971e -- go.sum` (empty), `git diff b10f5fd aab971e -- '*.go'` (empty), `xxd main/go.mod | head -5` (byte-level line-1 check), ripgrep of `github\.com/evanmschultz/coding_challenges/fang` and `github\.com/evanmschultz/fwc` scoped to `*.{go,mod,sum}` (zero hits), case-variant probe `evanmschultz/[Rr][aA][Kk]` + `evanmschultz/(Rak|RAK)` (only correct lowercase `rak` present), H2-heading scan of `BUILDER_WORKLOG.md` for round structure (3 distinct well-formed H2s), grep sweep of drop dir for `Section 0|SEMI-FORMAL|## Planner|## QA Proof|## QA Falsification|## Convergence` (only legitimate WORKFLOW.md-defined `## Planner` section header in drop PLAN.md + Unit 1.1 QA's own explicit anti-pattern probe — no Section 0 leakage).

**Counterexamples found:** none

**Findings:** one advisory (non-FAIL): drop PLAN.md line 65 parenthetical references "main/PLAN.md line 82–83 + line 194" but the relevant content on main/PLAN.md sits at **line 195** (line 194 is a different bullet about `/tmp/rak-stash/main.go`). Line 82–83 is correct. This is documentation rot on the acceptance bullet itself — not a Unit 1.2 code defect; Unit 1.2 executed against the bullet's intent, not against its line references. Routed to orchestrator for planner touch-up consideration.

### Attack-by-attack trace

- 1. Module-path correctness. `xxd main/go.mod | head -5` shows line 1 = `6d 6f 64 75 6c 65 20 67 69 74 68 75 62 2e 63 6f 6d 2f 65 76 61 6e 6d 73 63 68 75 6c 74 7a 2f 72 61 6b 0a` = `module github.com/evanmschultz/rak\n`. Pure LF terminator, no BOM, no leading/trailing whitespace, no tab-vs-space drift, no alternate case (ripgrep `evanmschultz/(Rak|RAK)` zero hits; `evanmschultz/[Rr][aA][Kk]` returns only the one correct-case `rak` match at go.mod:1). Line 2 = empty, line 3 = `go 1.26.1` — blank line + Go directive preserved. REFUTED.
- 2. Scope creep. `git show aab971e --stat` = exactly 3 files: `drops/DROP_1_CODE_SCAFFOLD_MAGE_CI/BUILDER_WORKLOG.md` (+72), `drops/DROP_1_CODE_SCAFFOLD_MAGE_CI/PLAN.md` (±1), `go.mod` (±1). No `.go`, no `go.sum`, no other file. `git diff b10f5fd aab971e -- '*.go'` empty. `git diff b10f5fd aab971e -- go.sum` empty. REFUTED.
- 3. Self-referential acceptance loophole. Ripgrep of both legacy strings scoped to `*.{go,mod,sum}` returns zero hits. Sharpened acceptance (commit `e73e98a` — 3 line changes in drop PLAN.md only, no code change) correctly excluded markdown prose from the invariant. REFUTED.
- 4. Go.sum drift. `git diff b10f5fd aab971e -- go.sum` empty; `git log --oneline -- go.sum main/go.sum` shows only Unit 1.1's `b10f5fd feat(cmd): split stash into cmd/rak main + root` touched it. Unit 1.2 did NOT touch `go.sum`; Unit 1.4 will own the `go mod tidy` prune per drop PLAN.md Notes "go.sum drift". REFUTED.
- 5. Drop 2.1 hand-off boundary regression (`count` / `Counts` in `cmd/rak/root.go`). `git diff b10f5fd aab971e -- '*.go'` empty → no `.go` file was touched → the pinned hand-off surface cannot have regressed. REFUTED by impossibility (module-path edit physically can't touch `.go` files).
- 6. State-flip honesty. Drop PLAN.md Unit 1.2 row: `State: done`. Three current grep-based acceptance bullets (drop PLAN.md lines 63–66) all pass under sharpened wording — bullet 1 (head -1 exact match): confirmed via `xxd`. Bullet 2 (`coding_challenges/fang` scoped grep): `Grep` with `*.{go,mod,sum}` glob returned "No matches found". Bullet 3 (`fwc` scoped grep): same. Bullet 4 (compile verification) correctly deferred to 1.5. REFUTED.
- 7. Acceptance-text line-ref self-consistency. Drop PLAN.md line 65 says "main/PLAN.md line 82–83 + line 194". Actual main/PLAN.md: lines 82–83 contain the correct content (`Rewrite go.mod module path ... github.com/evanmschultz/coding_challenges/fang, not fwc`); line 194 is a `/tmp/rak-stash/main.go` bullet; the intended content lives at **line 195**. CONFIRMED drift — but it's an acceptance-text bug authored by the planner (or earlier plan-QA), NOT a Unit 1.2 code/state-flip defect. Advisory finding only; does not FAIL Unit 1.2 because the builder is bound to the bullet's intent and sharpened grep semantics, not to auditing the parenthetical line numbers inside it. REFUTED as a Unit 1.2 FAIL; recorded as an advisory for the orchestrator.
- 8. BUILDER_WORKLOG.md Round 2 honesty. Round 2 section claims: only PLAN.md state flip + this worklog append, no Go/go.mod/go.sum edits. Cross-check: commit `aab971e` (which landed Round 1 + Round 2 together in one commit — Round 2's atomic work was the state flip + appendix) shows exactly the 3 files listed. Round 2 narrative matches the actual diff. The "commit `e73e98a`" reference inside Round 2's text is correct (`e73e98a docs(drop-1): scope unit 1.2 acceptance greps to code files`). REFUTED.
- 9. Section 0 leakage. Grep of the drop dir for `Section 0|SEMI-FORMAL|## Planner|## QA Proof|## QA Falsification|## Convergence`: only legitimate hits are (a) drop PLAN.md's `## Planner` section heading (a WORKFLOW.md-defined durable heading, not Section 0 — Section 0's `## Planner` would be nested under `# Section 0 — SEMI-FORMAL REASONING`, which does not appear) and (b) Unit 1.1 QA's own attack-12 probe text in BUILDER_QA_FALSIFICATION.md describing the anti-pattern. No durable artifact contains Section 0 reasoning. REFUTED.
- 10. Round-numbering well-formedness. `grep -n '^## Unit' BUILDER_WORKLOG.md` returns: line 5 `## Unit 1.1 — Round 1`, line 44 `## Unit 1.2 — Round 1`, line 86 `## Unit 1.2 — Round 2`. Three distinct top-level H2s, correct `## Unit N.M — Round K` format per WORKFLOW.md File Lifecycle, no duplicates, no off-by-one. REFUTED.

### Hylla Feedback

N/A — task touched only `go.mod` + markdown durable artifacts (drop PLAN.md, BUILDER_WORKLOG.md). Hylla indexes Go source only; `go.mod` and markdown are outside its scope. Falsification evidence came from `git show` / `git log` / `git diff`, `xxd`, `Grep` (ripgrep), `Read`. No Hylla query attempted because none would apply.
