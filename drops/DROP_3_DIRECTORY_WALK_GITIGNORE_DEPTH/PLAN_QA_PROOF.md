# DROP_3 — Plan QA Proof — Round 2

**State:** proof review complete
**Agent:** go-qa-proof-agent
**Target:** `main/drops/DROP_3_DIRECTORY_WALK_GITIGNORE_DEPTH/PLAN.md`
**Round:** 2
**Verdict:** FAIL — Round 3 planner revise required on one point (P1 below)

## Premises

- Round 2 must have addressed all 4 Round 1 falsification blockers (C1–C4) and 6 Round 1 surface findings (C5–C10).
- Round 2 must have addressed the 2 Round 1 proof observations (O1, O2).
- Every library-API claim must match authoritative upstream docs (Context7 / `go doc`).
- Plan must be internally consistent (DAG, unit ordering, F-pins, acceptance criteria yes/no verifiable).
- Plan must cross-reference cleanly against `main/CLAUDE.md` (import DAG, file breakdown, errors, tests) and `main/PLAN.md` DROP_3 row.

## Evidence

- PLAN.md full read at revision `1107cac`.
- Round 1 QA files from commit `5a7e893` (`PLAN_QA_PROOF.md`, `PLAN_QA_FALSIFICATION.md`).
- `main/PLAN.md` DROP_3 row + decisions 25, 27 cross-check.
- `main/CLAUDE.md` § "Project Structure" → "Import DAG" + "File Breakdown"; § "Errors"; § "Tests" → "two-tier testdata rule"; § "Go Development Rules" → "Errors", "Concurrency".
- `go doc iter` — "Yield panics if called after it returns false."
- `go doc io/fs.WalkDir`, `io/fs.WalkDirFunc`, `io/fs.SkipAll`, `io/fs.SkipDir` — control-flow semantics.
- `go doc testing/fstest.MapFile` — symlink fixture support (`Mode: fs.ModeSymlink`, `Data` holds target).
- Context7 `/bmatcuk/doublestar` v4 README + UPGRADING — `Match` splits on `/` on all platforms; `PathMatch` uses the OS separator.

## Trace or cases

Round 1 → Round 2 coverage audit:

| Round 1 finding | Round 2 location | Verdict |
|---|---|---|
| C1 yield-false panic | F14 (line 172) + `TestWalker_RangeBreak` (line 101) | PASS |
| C2 `GitignoreEnabled` zero-value bug | Flipped to `DisableGitignore` (line 78) | PASS |
| C3 `IsHidden` naming drift | Package-level helper consistent at lines 61, 78, 84 | PASS |
| C4 symlink untested | `TestWalker_SymlinkYielded` (line 102) | PASS |
| C5 git/ripgrep source-file citations | Removed per line 114 "no upstream source-file citation" | PASS |
| C6 relPath separator | Line 41 "forward-slash separators per `io/fs` convention" | PASS |
| C7 depth counts edges | Line 78 "`root/file.txt` is depth 0; `root/sub/file.txt` is depth 1" | PASS |
| C8 `render.Directory` provisional | Lines 178–179 C8 breadcrumb + Drop 6.1 migration note | PASS |
| C9 Renderer interface growth | F15 (line 173) | PASS |
| C10 binary error-path spec | Lines 116 + 143 "skipped, error aggregated, not fatal" + test covers `fs.ErrPermission` | PASS |
| O1 CLAUDE.md file-breakdown row | Line 192 drop-end docs update commit | PASS |
| O2 doublestar Match/PathMatch pin | Line 44 picks `PathMatch` with inverted rationale | **FAIL — see P1** |

Internal consistency:

- DAG: 3.0 → (3.1, 3.2) → 3.3; 3.2 → 3.4; (3.3, 3.4) → 3.5. No cycle. Matches CLAUDE.md import DAG.
- F1–F15 pins ordered; each cites at least one unit that enforces it.
- `mage addDep`-only dep path matches CLAUDE.md § "Dependencies".
- Sentinel error idiom for `ErrBinaryFile` + `errors.Is` inspection matches CLAUDE.md § "Errors".
- MapFS-first test idiom matches CLAUDE.md § "Tests" → "two-tier testdata rule"; `cmd/rak/testdata/tree/bin.dat` single binary fixture sits inside the allowed exception.

## Conclusion

**FAIL — Round 3 planner revise required on P1.**

Ten of eleven Round 1 findings are fully mitigated in Round 2. The library-API pin for `doublestar/v4` (O2 from Round 1) was addressed but with **inverted rationale** and a choice that contradicts another pin in the same plan. Detail in P1.

## Phase 3 blockers (must land before build)

- **P1 — `doublestar.Match` vs `PathMatch` rationale is factually inverted; chosen API conflicts with C6/F1 forward-slash pin.**

  Line 44 selects `github.com/bmatcuk/doublestar/v4.PathMatch` with rationale: *"`/`-sensitive matching is required because relative paths like `src/foo.go` need to match `src/**/*.go` patterns correctly; `doublestar.Match` is shell-style and treats `/` as a literal"*.

  Per Context7 /bmatcuk/doublestar README + UPGRADING docs:
  - `Match(pattern, name)` — *"`name` and `pattern` are **split on forward slash (`/`)** characters"* — slash-sensitive on ALL platforms.
  - `PathMatch(pattern, name)` — *"PathMatch will automatically use your **system's path separator** to split `name` and `pattern`"* — OS-specific.

  The rationale is backwards: `Match` is the slash-sensitive one. More load-bearing: line 41 (C6 / F1 pin) requires `relPath` to always use forward slashes per `io/fs` convention. `PathMatch` on Windows would split on `\`, which contradicts C6. On macOS/Linux the bug is invisible because `PathMatch` happens to split on `/`; but the rationale claim is wrong as-written, and Drop 8/9 cross-platform CI work would surface the contradiction.

  **Resolution options for Round 3 planner:**
  1. Keep C6's forward-slash contract, switch the API to `doublestar.Match`, and fix the rationale to match upstream docs (`Match` splits on `/` on all platforms; `PathMatch` uses the OS separator). Recommended — matches the `io/fs` convention the plan already pins.
  2. Drop C6, accept OS-native separators in `relPath`, keep `PathMatch`. Requires re-verifying all downstream code paths that consume `relPath` (gitignore matcher, JSON output) don't break on Windows.

  QA cannot resolve this at build time; the builder reading the current plan has no way to know whether C6 or line 44 is authoritative.

## Phase 3 observations (non-blocking polish)

- **O3 (new)** — `internal/fileset/file_test.go` is introduced by Unit 3.2 but is not listed in `main/CLAUDE.md` § "Project Structure" → "File Breakdown" (which lists `file.go` standalone; it does list `walker.go` + `walker_test.go` as two rows). This is a CLAUDE.md inconsistency rather than a plan defect. Drop-end docs update (alongside O1's binary rows) should add the `file_test.go` row. Flag for orch at closeout.

## Unknowns

None on the proof side. P1 is a concrete evidence-backed counterexample, not an unknown; Round 3 planner fixes it with one of the two surgical options above.
