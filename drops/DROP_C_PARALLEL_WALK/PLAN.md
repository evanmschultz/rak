# DROP_C — STABLE_SORT_FIX

**State:** building
**Tier:** B
**Blocked by:** —
**Paths (expected):** internal/summary/sort.go, internal/summary/sort_test.go
**Packages (expected):** internal/summary
**PLAN.md ref:** — (top-level PLAN.md removed at v0.1.0 ship; see memory `session_handoff_2026_05_16_v020_planning.md`)
**Workflow:** main/drops/WORKFLOW.md
**Started:** 2026-05-16
**Closed:** —

## Scope

**One-unit drop: fix a latent bug in `SortDirs`.**

Originally scoped as "parallel walk + `--workers` flag + `--follow` symlink traversal" but all of that was CUT from v0.2.0 per the YAGNI sweep on 2026-05-17 (full rationale in memory `feedback_yagni_strict.md`). What remained: the `slices.SortStableFunc` fix, which is genuinely a latent bug regardless of parallelism — Go's map iteration is already randomized, so an unstable sort over a map-derived slice already produces non-deterministic output today on tie-bearing repos. The drop directory is named `DROP_C_PARALLEL_WALK` for git-history continuity; the actual content is just the sort fix.

**Out of scope (deferred to v0.3.0):**
- Parallel walk via `errgroup.Group` with `SetLimit(N)` — defer to v0.3.0.
- `--workers <int>` flag — defer to v0.3.0.
- `--follow` symlink traversal — defer to v0.3.0; `find -L . -type f | rak --files-from -` (Stream D) covers the use case via Unix composition.
- VHS gif + README "Performance" section — deferred with the parallel walk feature.

## Planner

Single unit. Tier B = falsification-only build-QA; the test suite is the proof.

---

### Unit C.1 — Stable sort fix in internal/summary

**State:** done
**Paths:** `internal/summary/sort.go`, `internal/summary/sort_test.go`
**Packages:** `internal/summary`
**Blocked by:** —

**Scope:**

`SortDirs` in `internal/summary/sort.go` currently uses `slices.SortFunc` (line 75), which is UNSTABLE per Go stdlib docs. Map iteration order over the `dirs` accumulator in `walkAndCount` is random (Go map iteration is randomized by design), and an unstable sort produces different byte output for the same input whenever two directory entries tie on the sort key. This is a latent bug today (without parallelism): users with tie-bearing fixtures may see different `rak --sort lines .` output across runs.

**Change (one line):** replace `slices.SortFunc` with `slices.SortStableFunc` in `SortDirs`. No API change; the function signature and behavior are unchanged except that equal elements preserve their original relative order. `slices.SortStableFunc` is available since Go 1.21; rak requires Go 1.26+.

Also update the doc comment on `SortDirs` to reference `slices.SortStableFunc` instead of `slices.SortFunc`.

**Test (add to `sort_test.go`):** `TestSortDirs_StableTieBreak` — a tie-rich fixture with at least 3 `Directory` values that share identical `Lines`, `Files`, AND `Bytes` counts (e.g. three dirs each with Lines=100, Files=5, Bytes=2048). Sort by each of the four keys in turn; assert the output order is exactly the input order for tied elements (i.e. stable). This test would silently pass on a tie-free fixture; the tie-richness is mandatory.

**Acceptance:**
- `slices.SortFunc` does NOT appear in `SortDirs` after this unit; `slices.SortStableFunc` does.
- `TestSortDirs_StableTieBreak` passes under `mage test`.
- All existing `sort_test.go` tests continue to pass.
- `mage build` passes.

---

## Notes

**Why this shipped as one tiny unit instead of being folded into DROP_E**: the bug fix has no overlap with E's smaller items (lockfile, friendly error, GoReleaser, gif) — it's a self-contained `internal/summary` change. Keeping it as its own drop preserves a clean audit trail for "the sort bug fix" vs the polish bundle.

**Parallel walk + `--workers` + `--follow` cut from v0.2.0**: all three deferred to v0.3.0. Reasons:
- Parallel walk: rak is already fast enough on real repos that nobody has reported slowness in v0.1.x. Adding ~150 LOC of worker-pool plumbing + cancel-deadlock handling for an unrequested perf win is YAGNI.
- `--workers` flag: if parallelism does ship in v0.3.0, the flag itself may still be YAGNI (silent `runtime.NumCPU()` default may be enough).
- `--follow`: `find -L . -type f | rak --files-from -` (Stream D in v0.2.0) covers the symlink case using a 40-year-old tool. Native `--follow` only worth adding if a user explicitly asks.

**Stream C `cmd/rak/root.go` overlap with B/D**: NONE in v0.2.0. C.1 only touches `internal/summary/*`. No flag wiring, no cmd/rak/root.go contention with Stream D (`--files-from`) or DROP_E (`--include-lockfiles`).
