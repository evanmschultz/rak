# DROP_C — Builder QA Falsification

## Unit C.1 — Round 1

**Verdict:** PASS

### Summary

The stable-sort fix is correct and the new test genuinely exercises the
stability guarantee. All eight falsification angles attempted — none produced
a counterexample. `mage test` (which always runs `-race`) reports
`internal/summary` clean.

### Attacks attempted

#### Attack 1 — `slices.SortFunc` slipped back in elsewhere

- **Severity:** EXHAUSTED, no counterexample.
- **Where:** `internal/summary/sort.go`, `internal/summary/summary.go`.
- **Counterexample sought:** any sort call still using the unstable
  `slices.SortFunc` (or `sort.Slice`).
- **Result:** REFUTED. `sort.go` has exactly one sort call at line 76, and it
  is `slices.SortStableFunc`. `summary.go` has no sort calls. The doc comment
  on `SortDirs` (lines 64–65) correctly references `slices.SortStableFunc`.
  Hylla snapshot 10 (pre-fix, stale until drop-end reingest) confirms
  `SortDirs` was the only sort site in the package.

#### Attack 2 — Tie-rich fixture not actually fully tied

- **Severity:** EXHAUSTED, no counterexample.
- **Where:** `internal/summary/sort_test.go:17-24` (`tieRichDirs()`).
- **Counterexample sought:** a fixture where any two of `{Lines, Files,
  Bytes}` differ across the three "tied" entries, making the stability
  assertion vacuous.
- **Result:** REFUTED. `tieRichDirs()` constructs three `Directory` values
  all sharing `Counts{Lines: 100, Bytes: 2048}` and `Files: 5`. All three
  numeric keys are identical across all three entries — the comparator
  returns 0 for every pairwise compare on every numeric key. Stability check
  is genuine, not vacuous.

#### Attack 3 — Assertion accepts any order instead of input order

- **Severity:** EXHAUSTED, no counterexample.
- **Where:** `internal/summary/sort_test.go:44-58`.
- **Counterexample sought:** an assertion that checks Path-set equality
  rather than position-by-position ordering, which would pass under an
  unstable sort.
- **Result:** REFUTED. The test captures `want[i] = d.Path` BEFORE the sort
  call, then asserts `d.Path != want[i]` after. This is a position-indexed
  comparison — exactly the correct stability check. An unstable sort that
  permuted tied elements would trip the assertion.

#### Attack 4 — Path subtest doesn't actually exercise sort behavior

- **Severity:** EXHAUSTED, no counterexample.
- **Where:** `internal/summary/sort_test.go:65-77`.
- **Counterexample sought:** path subtest passing trivially without
  performing a real lex sort.
- **Result:** REFUTED. Input order is `gamma/, alpha/, beta/`; expected
  output is `alpha/, beta/, gamma/`. The expected ordering differs from input
  ordering, so the assertion can only pass if a real ascending lex sort runs.
  Asc/desc direction is correct per `effectiveAsc(SortPath, false)` →
  `!false = true` → ascending branch.

#### Attack 5 — Direction (asc/desc) not exercised for stability

- **Severity:** nit (not a blocker).
- **Where:** `internal/summary/sort_test.go:49`.
- **Counterexample sought:** stability breaking under `asc=true` for numeric
  keys.
- **Result:** REFUTED in practice. The numeric subtests pass `asc=false`
  (the default direction for numeric keys). The `if !eff { return -result }`
  line in the comparator (sort.go:88-91) only fires when `result != 0`;
  negating 0 stays 0, so stability of tied elements is direction-independent
  by construction of the comparator. Existing `TestSortDirs_*_Asc` tests in
  `summary_test.go` cover `asc=true` correctness for non-tied elements.
  Adding `asc=true` stability subtests would be belt-and-suspenders but is
  not load-bearing.

#### Attack 6 — Concurrent-stream cross-contamination

- **Severity:** EXHAUSTED, no counterexample.
- **Where:** `internal/summary/` git history.
- **Counterexample sought:** an in-flight Stream A / D / E commit touching
  `internal/summary/*` that could interact with C.1's fix.
- **Result:** REFUTED. `git log -- internal/summary/` shows three commits
  total; the most recent (`57f71dc fix(summary): use stable sort to preserve
  order on ties`) is C.1 itself. No other recent commits touch this package.

#### Attack 7 — `mage test ./internal/summary/...` regression

- **Severity:** EXHAUSTED, no counterexample.
- **Where:** project-wide.
- **Counterexample sought:** any test failure in `internal/summary` or
  elsewhere caused by the fix.
- **Result:** REFUTED. `mage test` reports clean across the whole project:
  `ok github.com/evanmschultz/rak/internal/summary 2.019s`. All eight other
  packages also pass clean (including `internal/lister`, which the worklog
  flagged as having a pre-existing failure — see Note A below).

#### Attack 8 — `-race` adequacy for the new test

- **Severity:** EXHAUSTED, no counterexample.
- **Where:** `internal/summary/sort_test.go` (the new `t.Parallel()` calls).
- **Counterexample sought:** a data race surfaced by `-race` against the new
  parallel subtests.
- **Result:** REFUTED. `mage test` runs `-race` unconditionally per project
  rules; the run was clean. The parallel subtests each call `tieRichDirs()`
  to construct their own slice — no shared mutable state across subtests.
  The captured `key := key` range variable on sort_test.go:38 is the
  correct pre-Go-1.22 pattern (harmless in Go 1.26 where it's a no-op, but
  safe to keep for clarity).

### Notes

- **Note A (informational, not a C.1 finding):** the builder's worklog
  states `internal/lister` had a pre-existing failure due to a DROP_D
  `NewFilesFromLister` stub. As of this falsification run, `mage test`
  reports `internal/lister` as `(cached) ok` — the failure no longer
  reproduces. Either DROP_D landed in the interim or the diagnosis was
  inaccurate. Not a C.1 concern; orchestrator can confirm out-of-band.

- **Note B (Hylla freshness):** Hylla snapshot 10 still returns the pre-fix
  `SortDirs` docstring (referencing `slices.SortFunc`). Expected — per
  project rules Hylla reingest is drop-end only. Disk source (verified by
  `Read`) is post-fix and correct.

### Hylla Feedback

None — Hylla answered everything needed. The only Hylla call was a
keyword search for `slices.SortFunc` / `SortStableFunc` against the
pre-fix snapshot; it correctly returned the stale pre-fix `SortDirs` block
(snapshot 10 predates the C.1 commit), and the on-disk verification via
`Read` was the correct next step. No miss, no fallback gripe.
