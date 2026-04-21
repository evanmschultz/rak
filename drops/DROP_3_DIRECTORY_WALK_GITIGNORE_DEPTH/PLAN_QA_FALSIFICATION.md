# DROP_3 — Plan QA Falsification — Round 3

**State:** round 3 complete
**Agent:** go-qa-falsification-agent
**Target:** main/drops/DROP_3_DIRECTORY_WALK_GITIGNORE_DEPTH/PLAN.md
**Round:** 3
**Verdict:** PASS — no unmitigated counterexample survives

## Premises

- Round 3 revise (commit `5c17e29`) must mitigate Round 2's one blocker (CN1/P1 — `doublestar.PathMatch` → `Match` with corrected rationale), two surface findings (SF1 line 14 "4 units" → "6 units"; SF2 line 134 error-handling vs F6), and the Round 2 proof observation (O3 — merge `file_test.go` into O1's File Breakdown bullet).
- No regression permitted against Rounds 1–2 mitigations (C1–C10, F14, F15, `DisableGitignore`, package-helper `IsHidden`, `TestWalker_SymlinkYielded`, C8 breadcrumb).
- New Round 3 content itself must pass attack — API claims must match upstream docs, cross-references consistent, no hidden dependencies or YAGNI violations.

## Evidence

- `Read` Round 3 PLAN.md.
- `git show 5c17e29` — exact four-location revise diff (lines 14, 44, 134, 192).
- Context7 `/bmatcuk/doublestar` — `Match` splits on `/`; `PathMatch` uses OS separator; `Glob` requires `/` for `io/fs` per upstream docs.
- `main/CLAUDE.md` lines 126–155 — File Breakdown confirms `file_test.go`, `binary.go`, `binary_test.go` absent.
- `Grep ^### Unit 3\.` — enumerates six unit headings matching line 14.
- `go doc iter`, `io/fs.SkipAll`, `io/fs.WalkDirFunc`, `testing/fstest.MapFS.Open` — all cited claims check out.

## Confirmed counterexamples

None.

## Surface findings (non-blocking polish)

- **SF3** — Depth prose uses 1-based file-nesting counting in test descriptions (line 93–94) while C7 / line 78 pins 0-based ("root/file.txt is depth 0"). Pre-existing, not Round 3-introduced. Internally resolvable via 1-based vernacular ↔ 0-based formal mapping.
- **SF4** — Two different `walk`-prefixed error wraps. Walker internal errors wrap as `walk %q: %w` (line 88, with path); runRoot ctx-cancel wraps as `walk: %w` (line 134, no path). Could make error-message triage slightly ambiguous. Optional polish: rename runRoot's wrap to `walk cancelled: %w`.

## Trace or cases

- **CN1/P1 REFUTED**: line 44 swap to `Match` verified against Context7 doublestar docs (Match splits on `/` unconditionally; PathMatch uses OS separator per v4 UPGRADING.md). Rationale citing `Glob` docs ("requires `/` as the path separator ... due to the use of the io/fs package") matches exactly.
- **SF1 REFUTED**: line 14 "6 units" matches `### Unit 3.{0,1,2,3,4,5}` headings one-for-one.
- **SF2 REFUTED**: runRoot now aggregates walker errors (mirroring C10's IsBinary policy line 143), only `ctx.Err()` aborts, explicit F6 cross-reference. Walker already terminates on cancel (line 83), so runRoot aborting on yielded `ctx.Err()` is idempotent. `continue` after aggregating naturally skips `*File` processing when `f` is nil.
- **O3 REFUTED**: line 192 bullet enumerates `file_test.go` (~150 LOC), `binary.go` (~80 LOC), `binary_test.go` (~100 LOC) — all three genuinely absent from CLAUDE.md lines 135–137.
- **Regression audit clean**: C1/F14 + TestWalker_RangeBreak; C2/DisableGitignore zero-value; C3/IsHidden package-level at all three sites; C4/TestWalker_SymlinkYielded; C5 no source-file citations; C6 forward-slash; C7 depth-0 root; C8 breadcrumb at lines 178–179; C9/F15; C10 binary skip + aggregate — all intact.
- **Holistic**: library API claims grounded via Context7; YAGNI invariant F13 intact; no hidden deps introduced; `IsBinary()` + `Open()` dual-open pattern accepted per F4 contract.

## Conclusion

**Verdict: PASS.** Round 3 revise mitigates every Round 2 finding with surgical edits at exactly four lines. No unmitigated counterexample. No regression against Rounds 1–2 mitigations. Two minor pre-existing surface findings (SF3, SF4) are optional polish — neither originated in Round 3 and neither falsifies the plan's correctness.

## Unknowns

- None on the falsification side. SF3 and SF4 are surface observations, not evidence gaps.
