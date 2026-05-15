# DROP_8 — SAFETY_RAILS

**State:** done
**Tier:** B
**Blocked by:** DROP_7
**Paths (expected):** `main/cmd/rak/root.go` (add `--max-files int` flag + abort logic in walkAndCount), `main/cmd/rak/root_test.go` (flag-parsing + abort-behavior tests)
**Packages (expected):** `github.com/evanmschultz/rak/cmd/rak`
**PLAN.md ref:** main/PLAN.md → `DROP_8_SAFETY_RAILS` row
**Workflow:** main/drops/WORKFLOW.md
**Started:** 2026-05-15
**Closed:** 2026-05-15 (CI run 25943131873 green; Hylla ingest task task-346c727ad6254892 at commit a3c8e73)

## Scope

Add `--max-files <int>` as a count-cap safety rail per main/PLAN.md decision 30. When set to a positive value, rak aborts the walk if the running per-walk file count exceeds the cap — protecting users from pointing rak at an unexpectedly huge tree. Default `0` = no limit (existing behavior preserved). Tier B per the post-decision-30 trim: all other Drop 8 candidates (parallel walk, spinner, `--follow`, `--tracked-only` opt-in) are cut to v0.2. One unit, falsification-only QA per WORKFLOW.md Cascade Tiering Tier B mechanics.

## Planner

Tier B — orch wrote this section inline (no planner subagent). One unit.

### Unit 8.1 — --max-files safety rail

- **State:** done
- **Paths:**
  - `main/cmd/rak/root.go`
  - `main/cmd/rak/root_test.go`
- **Packages:** `github.com/evanmschultz/rak/cmd/rak`
- **Acceptance:**
  - `rootFlags` gains `maxFiles int` field (default `0`).
  - `newRootCmd` registers `--max-files int` with help text: `"abort the walk when the file count exceeds N (default 0 = no limit)"`.
  - `walkAndCount` (or wherever file counts are accumulated) increments a per-walk file counter as each file is **accepted** (post-binary-skip + post-`--lang` filter — the same condition that gates `byDir` / `byDirFiles` increments per F42). When `flags.maxFiles > 0 && acceptedFiles >= flags.maxFiles`, abort the walk: return a wrapped sentinel error `ErrMaxFilesExceeded` defined in `cmd/rak/root.go` (or `internal/lister` if more appropriate — builder's choice; document in worklog). The wrap message includes the limit value, e.g. `"rak: file count exceeded --max-files %d: %w"`.
  - `runRoot` surfaces the error to cobra unchanged — cobra prints the wrapped message; user sees the explanatory text.
  - Tests:
    - `TestRootCmd_MaxFiles_NotSet_CountsAll` — fstest.MapFS with 5 files; no `--max-files` flag → all 5 counted (existing behavior preserved).
    - `TestRootCmd_MaxFiles_ZeroExplicit_CountsAll` — same fixture; `--max-files 0` → all 5 counted (zero = no limit).
    - `TestRootCmd_MaxFiles_UnderLimit` — same fixture; `--max-files 10` → all 5 counted (limit not hit).
    - `TestRootCmd_MaxFiles_AtLimit_Aborts` — same fixture; `--max-files 3` → walk aborts mid-stream; returned error wraps `ErrMaxFilesExceeded` (verify via `errors.Is`).
    - `TestRootCmd_MaxFiles_NegativeValue` — `--max-files -1` → either rejected by cobra (preferred via flag validation) OR treated as 0/no-limit (document choice in worklog).
  - F26 RelPath invariant unchanged. F19 sentinel, F24 mutual exclusivity, F33 LangUnknown suppression, F44 Files propagation — all preserved.
  - `mage ci` green from `main/`.
- **Blocked by:** —

## Notes

### F-pin

- **F45 — `ErrMaxFilesExceeded` sentinel contract:** declared in `cmd/rak/root.go` as `var ErrMaxFilesExceeded = errors.New("rak: file count exceeded --max-files limit")` (or similar). Callers use `errors.Is(err, ErrMaxFilesExceeded)` to branch; the wrapped form `fmt.Errorf("rak: file count exceeded --max-files %d: %w", N, ErrMaxFilesExceeded)` carries the specific limit value in the user-visible message. Never string-match.

### Tier B mechanics applied

Per WORKFLOW.md § "Cascade Tiering" Tier B: no planner subagent (orch wrote this inline); single builder spawn for Unit 8.1; falsification-only build-QA after; proof QA skipped (the test suite is the proof per the tier-B trade-off).

### Cut from this drop per decision 30

- Drop 8.1 parallel walk → v0.2.
- Drop 8.2 spinner / progress indication → v0.2.
- Drop 8.4 `--tracked-only` opt-in flag → already default behavior since Drop 4 (decision 32 inverted it).
- Drop 8.5 `--follow` symlinks → v0.2.

### Open Unknowns

None. Spec is tight; one unit, one round expected.
