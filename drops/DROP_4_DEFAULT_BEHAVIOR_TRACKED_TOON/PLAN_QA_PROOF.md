# Drop 4 — Plan QA Proof, Round 2

**Verdict:** pass-with-findings

## Summary

The Round 2 revise resolves all 22 Round 1 findings I audited against the revised PLAN.md. The four most consequential resolutions:

- **C5 / Decision A** is wired correctly via a new sentinel `lister.ErrNoGitignoreInRepo` (declared in `lister.go` at Unit 4.1), checked inside `Detect` immediately after `git rev-parse --is-inside-work-tree` succeeds and before `newGitLister` is called. F19 is rewritten end-to-end to match. Acceptance test `TestDetect_NoGitignoreInRepo_ReturnsSentinel` (Unit 4.1) exercises it via `errors.Is`. The error-text is the verbatim message the planner committed to and matches what `runRoot` surfaces in Unit 4.4.
- **C4 / hidden-segment helper** is pinned as `func anySegmentHidden(relPath string) bool` declared inline in `git.go` (Unit 4.2 acceptance), used in step 2 of `GitLister.List`. F18(a) wording is consistent.
- **C11 / P2 forward-declaration trap** is replaced by an explicit Compile Note in Unit 4.1 acknowledging that `mage build ./internal/lister/...` fails at the 4.1 commit boundary and listing exactly which other packages must still build clean. The note matches WORKFLOW.md Phase 5's "builder runs `mage build` + `mage test` for the touched packages" language — touched packages at 4.1 are `internal/fileset` only (NewFile export) plus the inert lister package that intentionally cannot compile alone.
- **P5 / F26 RelPath invariant** is now a first-class F-pin (F26): "walk-root-relative, forward-slash separated, no leading `./` and no leading separator. Both `GitLister` and `WalkLister` honor this. `cmd/rak/root.go`'s `dirKey` + `labelDirectories` rely on this invariant."

External API claims verified: Context7 confirms `cobra.MarkFlagsMutuallyExclusive` validates during command execution (before `RunE`) — F24 is correct. Toon-go `omitempty` and pipe-escape behaviors are explicitly routed to a builder-side spike before 4.5's `toon.go` is written (Unit 4.5 acceptance, "Spike first") with the spike outcome to be documented in `BUILDER_WORKLOG.md` — that is the right venue for an empirically-resolvable Unknown.

New surface introduced by the revise (4 items) all have acceptance coverage or doc-comment specs:

- `lister.ErrNoGitignoreInRepo` sentinel — declared in `lister.go`; F19 pins the contract; tested via `errors.Is` in 4.1.
- `lister.NewWalkLister(fsys, root, opts) *WalkLister` — exported in 4.3 with body noted as "Same as `newWalkLister`"; used by `runTreeFS` (4.4) to bypass `Detect` for `fstest.MapFS` tests; covered by `TestWalkLister_*` in 4.3 plus the existing `TestRootCmd_PathArg_*` table in 4.4.
- `fileset.NewFile(fsys fs.FS, path, relPath string) *File` — thin export of unexported `newFile`; 4.1 acceptance pins the doc comment text verbatim. Signature matches Drop 3's existing `newFile(fsys fs.FS, path, relPath string) *File` exactly (verified at `internal/fileset/file.go:52`). No semantic change.
- Compile-time assertion `var _ FileLister = (*WalkLister)(nil)` — pinned in walk.go or walk_test.go at 4.3.

Round 2 produces 2 findings + 2 nits — substantial convergence from Round 1's 22.

## Resolution audit (Round 1 findings)

Round 1 had 12 proof findings (P1–P12) and 15 falsification counterexamples (C1–C15). After Round 1 dedup the orchestrator surfaced 22 to the planner. Audit walks each by ID.

### Proof findings (Round 1)

- **P1 — GitLister.List CWD strategy (blocker).** Resolved: Unit 4.2 step 1 of `List` now runs `git ls-files --full-name -z` with `cmd.Dir = g.absRoot`. F17 explicitly handles BOTH possible empirical behaviors (toplevel-relative or CWD-relative paths) via the conservative prefix-strip approach. The Empirical Notes section pins an explicit Round-2-falsification probe: "run `git ls-files --full-name -z` from a non-root CWD and document the actual output." Verdict: **OK** (conservative implementation is correct regardless of empirical outcome; the explicit probe is a tight handoff to the falsification agent).
- **P2 — Forward-declare scheme (major).** Resolved: dropped the forward-declare option entirely. Unit 4.1's Compile Note explicitly accepts that `mage build ./internal/lister/...` fails at the 4.1 commit boundary; enumerates the packages that must still build clean (counting, fileset, ignore, render, summary, tokens). Verdict: **OK** — paired with C11 below for the WORKFLOW-compatibility check.
- **P3 — `TestDetect_InsideRepo` hermeticity (minor).** Partial: 4.1 still uses `filepath.Abs("../..")` against the actual checkout; not hermetic per P3's recommendation. The planner kept the trade-off (less hermeticity, less complexity). `t.Skip` guards remain. F19 sentinel test was added but uses the same non-hermetic approach. Verdict: **partial — accepted trade-off**. Not regressed in Round 2; planner did not pick up P3's `t.TempDir() + git init` suggestion. This is a Round 2 nit (R2-N3 below).
- **P4 — Detect `fs.FS` parameter ambiguity (major).** Resolved cleanly: `Detect` no longer accepts an `fsys` parameter at all. Unit 4.1 signature is now `func Detect(ctx, root string, opts fileset.WalkOptions) (FileLister, error)`. The Detect-internal WalkLister branch builds `os.DirFS(absRoot)` itself. GitLister also builds its own `fsys = os.DirFS(absRoot)`. The contract is now: "callers cannot inject a synthetic `fs.FS` through `Detect`; for that, call `lister.NewWalkLister` directly." Verdict: **OK**.
- **P5 — F26 RelPath invariant (major).** Resolved: F26 is a new first-class F-pin (visible in PLAN.md line 195). Verdict: **OK** — but acceptance criteria don't name a test that asserts the invariant (see R2-F1 below).
- **P6 — Speculative `omitempty` (minor).** Resolved: Unit 4.5 now opens with a "Spike first (C7 + C8)" gate. Builder authors a 5-line scratch program before writing `toon.go`, documents results in `BUILDER_WORKLOG.md` § "Spike: toon-go behavior". `TestTOONRenderer_RenderTree_NoErrors` still asserts "no `errors` key when empty" but accepts that the implementation strategy depends on spike outcome. Verdict: **OK**.
- **P7 — `walkAndCount` `fsys` wording (minor).** Resolved: Unit 4.4 now spells out `walkAndCount(ctx, source lister.FileLister, binary bool) (...)` with explicit "fsys parameter dropped (lister owns it)". The (P7) parenthetical cite is in the runDirectory bullet. Verdict: **OK**.
- **P8 — `runTreeFS` test-seam ambiguity (minor).** Resolved: Unit 4.4 explicitly pins `lister.NewWalkLister(fsys, ".", opts)` as the chosen path. 4.3's `NewWalkLister` is exported specifically to enable this. Verdict: **OK**.
- **P9 — `filepath.Abs("../../..")` brittleness (minor).** Partial: kept as-is. Same trade-off as P3. Verdict: **partial — accepted trade-off**. R2-N3 covers.
- **P10 — F19 testability (minor).** Resolved by re-design: F19 is now a hard error, not a no-op. The test path is fully exercised via `TestDetect_NoGitignoreInRepo_ReturnsSentinel` (4.1) and `TestRootCmd_NoGitignoreInRepo_Errors` (4.4). The pre-revise testability concern is moot. Verdict: **OK**.
- **P11 — Detect failure-mode silent degradation (minor).** Resolved: Unit 4.1 acceptance for `Detect` now explicitly distinguishes three cases: (a) exit 0 → GitLister or sentinel; (b) non-zero exit OR `exec.LookPath` failure → WalkLister silently; (c) "Unexpected OS-level command failure (not a non-zero exit from git): wrap and return the error." That last clause addresses P11's "broken `.git/` state" attack vector — though it's still permissive (anything that produces a non-zero exit silently falls back). Verdict: **OK** — the planner picked the permissive policy explicitly, which is a defensible product call.
- **P12 — Mid-walk git failure test (nit).** Resolved with explicit accepted-gap: 4.2 acceptance now reads "`TestGitLister_MidWalkGitFailure` — note: cleanly stubbing `exec.Command` at the package level is complex. Accepted gap: this path is not unit-tested in 4.2. Document the gap in `BUILDER_WORKLOG.md` § 'Hylla Feedback / Gap Notes' ..." Verdict: **OK** — explicit accepted gap is the right outcome.

### Falsification counterexamples (Round 1)

- **C1 — Integration fixture `vendor/` not tracked (blocker).** Resolved: Unit 4.4 now explicitly states the `testdata/tree` effective set is "`.gitignore`, `.hidden.txt`, `a.txt`, `bin.dat`, `sub/nested.txt` ... With `IncludeHidden: false` (default), `.gitignore` and `.hidden.txt` are excluded. `bin.dat` is binary-skipped. Effective set: `a.txt` + `sub/nested.txt`." The plan acknowledges the `no_gitignore_includes_vendor` semantic is dropped from integration coverage; existing `TestRootCmd_PathArg_*` tests stay on the `lister.NewWalkLister` + `fstest.MapFS` path so the historical contracts remain testable. Verdict: **OK**.
- **C2 — `runTreeFS` "builder decides" hand-wave (blocker).** Resolved: Unit 4.4 now pins `runTreeFS` to construct `lister.NewWalkLister(fsys, ".", opts)` directly. 4.3 exports `NewWalkLister`. The (C2 + C6) parenthetical cite is in the runTreeFS bullet. Verdict: **OK**.
- **C3 — Windows `filepath.Separator` bug (blocker on Windows).** Resolved: F17 is rewritten to use `filepath.ToSlash(strings.TrimPrefix(absRoot, toplevel))` followed by "strips any leading `/`" — no `string(filepath.Separator)` anywhere in the prefix-computation path. The rewrite normalizes to forward-slash before the trim. On Windows the result is correct. Verdict: **OK**.
- **C4 — F18 hidden-segment vagueness (blocker).** Resolved: `anySegmentHidden` helper is named, defined in `git.go`, and used in step 2 of `List`. F18(a) explicitly references it. Verdict: **OK**.
- **C5 — `--no-gitignore` no-op as user-trust violation (blocker).** Resolved per Decision A: now a hard error. See Summary. Verdict: **OK**.
- **C6 — Integration tests CWD inside repo (blocker).** Resolved: Unit 4.4's `runTreeFS` now bypasses `Detect` via `lister.NewWalkLister`. Integration tests that DO go through `Detect` (the `cmd/rak/integration_test.go` path-arg tests) accept the GitLister path explicitly and the effective set is enumerated. Verdict: **OK**.
- **C7 — `omitempty` unverified (major).** Resolved via the 4.5 Spike-first gate. Verdict: **OK**.
- **C8 — Pipe-in-value unverified (major).** Resolved via the same spike gate; F20 explicitly notes "If the spike reveals pipe is also unsafe, switch to `\t` (tab) and update this pin." Verdict: **OK** — but the spike's escape-test is not pinned as a per-acceptance assertion (Round 1's C8 suggested fix asked for an explicit `TestTOONRenderer_RenderTree_PipeInPath` round-trip test; the revise routes this to the spike instead). This is a tighter pin than Round 1's vague hedge, but the round-trip assertion remains a spike outcome, not a unit-test gate. Acceptable but soft. Verdict: **OK — soft**.
- **C9 — Submodule + sparse-checkout + worktree edge cases (major).** Resolved: a new "Git Edge Cases (Decision D)" Notes section enumerates submodules (pointer file counted, no recursion), sparse-checkout (whatever git returns), worktrees (validated by daily use). Verdict: **OK**.
- **C10 — F17 empty-prefix off-by-one (major).** Resolved by the F17 rewrite — `strings.TrimPrefix(absRoot, toplevel)` (not `toplevel+"/"`), then "strips any leading `/`". When `absRoot == toplevel`, TrimPrefix returns `""` and the leading-slash strip is a no-op. Result: `prefix == ""`. Verdict: **OK**.
- **C11 — 4.1 forward-declaration unsafe (major).** Resolved: explicit Compile Note accepts the broken-build trade-off. Verdict: **OK** — but see R2-N2 for WORKFLOW Phase 5 wording cross-check.
- **C12 — UN3 hidden-file decision-34 contradiction (major).** Resolved by Decision B: hidden filter applies to GitLister too. New "Hidden-file Policy" Notes section explicitly acknowledges the soft conflict with decision 34 and pins the resolution: "The hidden filter is a presentation-layer choice (which files to count), separate from the tracking-layer source (which files to enumerate from). This decision is intentional and not to be relitigated in v0.1.0." F21 wording matches. Verdict: **OK**.
- **C13 — `TestRootCmd_InvalidFormat` replacement gap (minor).** Resolved: Unit 4.4 now adds BOTH `TestRootCmd_MutuallyExclusiveFlags` AND `TestRootCmd_UnknownFlag` (with the (C13) cite). Verdict: **OK**.
- **C14 — `walkAndCount` / `runDirectory` thread-through (minor).** Resolved: both signatures now explicit with `binary bool` threaded. The (C14) cite is in the walkAndCount bullet. Verdict: **OK**.
- **C15 — F18 depth `>` vs `>=` (minor).** Resolved: F18(b) now says `strings.Count(relPath, "/") >= opts.Depth` explicitly, with cite "matching Walker's `depth >= w.opts.Depth` rule". The same `>=` is in Unit 4.2 step 3 of `List`. Verdict: **OK**.

**Round 1 resolution count: 22 of 22 addressed.** 20 fully resolved, 2 partial (P3 + P9, accepted trade-offs — captured as R2-N3 below).

## New findings (specific to Round 2 revise)

### Finding R2-F1 — F26 RelPath invariant has no test acceptance criterion

- **Severity:** medium
- **Unit affected:** 4.2 (acceptance test list), 4.3 (acceptance test list)
- **Claim/gap:** F26 is pinned as a load-bearing cross-lister invariant: "walk-root-relative, forward-slash separated, no leading `./` and no leading separator." `cmd/rak/root.go`'s `dirKey` + `labelDirectories` rely on it. But neither Unit 4.2 nor Unit 4.3's acceptance criteria name a test that **asserts** this invariant. 4.2's `TestGitLister_List_InRepo` asserts `go.mod` appears with `RelPath = "go.mod"` (covers root-level invariant); `TestGitLister_List_SubdirRoot` asserts `RelPath` values like `"file.go"`, `"walker.go"` (covers subdir prefix-strip invariant). But:
  - No test asserts "RelPath never has a leading `./`" (negative invariant).
  - No test asserts "RelPath never has a leading separator" (negative invariant).
  - No test in 4.3 asserts the same invariant for WalkLister output — it's tacit because `fileset.Walker` already honors this, but if Drop 5 or later changes Walker, the WalkLister invariant becomes unverified.
- **Suggested fix:** Add to 4.2 acceptance: "`TestGitLister_RelPathInvariant` — for each emitted `*fileset.File`, assert `!strings.HasPrefix(f.RelPath, \"./\")` and `!strings.HasPrefix(f.RelPath, \"/\")` and `f.RelPath == filepath.ToSlash(f.RelPath)`. Subtests: root walk, subdir walk." Mirror the same test in 4.3 against `fstest.MapFS` (subtests: empty-root walk, flat-files walk, nested walk). This makes F26 testable, which is the bar for an F-pin.

### Finding R2-F2 — `ErrNoGitignoreInRepo` sentinel contract not pinned as an F

- **Severity:** medium
- **Unit affected:** Cross-Unit F-Pins section
- **Claim/gap:** The sentinel `lister.ErrNoGitignoreInRepo` is a new public symbol with a load-bearing contract: callers branch on it via `errors.Is`. F19 mentions the sentinel ("This sentinel is declared as `var ErrNoGitignoreInRepo = errors.New(...)`") and the message text, but does not lift the wrapping/inspection contract to a first-class F-pin. Compare with the existing pattern: F25 (Render interface unchanged) is an invariant-pin for a public type; the same treatment is warranted for the sentinel's `errors.Is`-via-`%w` wrapping contract. Specifically:
  - F19 does not pin "Detect wraps the sentinel using `fmt.Errorf(\"lister: detect: %w\", ErrNoGitignoreInRepo)`" (Unit 4.4 implies this with "the `fmt.Errorf` chain wraps it cleanly", but the wording is buried in a single Unit 4.4 acceptance bullet, not an F-pin).
  - F19 does not pin "Callers MUST use `errors.Is(err, lister.ErrNoGitignoreInRepo)`; string-matching the message is forbidden" (matches main/CLAUDE.md § "Errors" rule "Never string-match an error").
- **Suggested fix:** Add a new F-pin (F27 or fold into F19) with the wording: "The `lister.ErrNoGitignoreInRepo` sentinel is wrapped via `fmt.Errorf(\"lister: detect: %w\", ErrNoGitignoreInRepo)` inside `Detect`. Callers inspect with `errors.Is(err, lister.ErrNoGitignoreInRepo)`; never via string match (per main/CLAUDE.md § Errors). The sentinel's `Error()` message is the verbatim user-facing text in [F19's pin]." This makes the error-inspection contract testable and gives QA falsification a clean target.

### Nit R2-N1 — `fileset.NewFile` doc-comment text pinned but no test asserts it

- **Severity:** nit
- **Unit affected:** 4.1 acceptance
- **Claim/gap:** The 4.1 acceptance pins the verbatim doc-comment text for `fileset.NewFile` ("NewFile constructs a File for the given path. Callers outside internal/fileset use this to create File handles when they have obtained a path from a non-Walker source (e.g. GitLister).") This is good. But no acceptance criterion verifies the doc comment actually lands in the source — `golint` would catch a missing comment (per main/CLAUDE.md § "Doc comments") but not a wrong-text comment. Since the doc-comment text is part of the API contract (it tells future callers when use is sanctioned), drift would be silent.
- **Suggested fix:** No action required at plan-QA time — `mage lint` + builder discipline cover this in practice. Recording for completeness. If the builder later proves to drift on doc-comment text frequently, add a Drop-N+ acceptance pattern like "doc-comment text matches plan verbatim, line for line."

### Nit R2-N2 — 4.1 commit boundary build-failure carve-out is correct but should cite WORKFLOW.md Phase 5

- **Severity:** nit
- **Unit affected:** 4.1 Compile Note
- **Claim/gap:** WORKFLOW.md Phase 5 (verified at lines 168, 149–162) reads: "builder runs `mage build` + `mage test` for the **touched packages**. QA mds note the targets run + result." This is permissive enough to allow Unit 4.1's broken-lister build state — the touched package is `internal/fileset` (the only `mage build`-able target at 4.1), and the inert `internal/lister/` package is also touched but intentionally cannot compile alone. The Compile Note in 4.1 enumerates the `mage test` invocation correctly (`./internal/fileset/... ./internal/counting/... ...`) but does not explicitly cite WORKFLOW.md Phase 5's "touched packages" language. A future builder reading 4.1 in isolation might not realize the carve-out is permitted by the workflow doctrine — they could try to "fix" the broken-lister state by inlining 4.2/4.3 stubs.
- **Suggested fix:** Add one sentence to the Compile Note: "This trade is consistent with WORKFLOW.md § Phase 5: per-unit verification runs `mage build` + `mage test` only for the touched packages, which at 4.1's commit boundary is `internal/fileset` (one file touched). The inert `internal/lister/` package is exempt until 4.2 + 4.3 land."

### Nit R2-N3 — Lister tests against actual checkout layout (P3 + P9 carry-over)

- **Severity:** nit
- **Unit affected:** 4.1 + 4.2 test bullets
- **Claim/gap:** Round 1's P3 and P9 both suggested hermetic `t.TempDir() + git init` fixtures instead of `filepath.Abs("../..")` against the actual checkout. The revise kept the non-hermetic approach with `t.Skip` guards. This is a defensible trade-off — hermetic fixtures cost more to author and the tests work reliably in CI — but it carries the latent risk P3 flagged: "the test will fail there [git binary present, .git/ absent], not skip." The revise added a `t.Skip("git binary not found")` guard but did not add a `git rev-parse --is-inside-work-tree` skip guard for the "binary present, no .git/" case. Low likelihood (CI runs always have .git/, dev runs always have .git/), so this is recorded as a known soft gap rather than blocking.
- **Suggested fix:** No action required for plan acceptance. If 4.1's `TestDetect_InsideRepo` ever fails non-deterministically in CI, retrofit the hermetic-tempdir pattern at that point.

## Unknowns

- **toon-go option names + struct-tag semantics** — `WithDocumentDelimiter`, `WithArrayDelimiter`, `DelimiterPipe`, and `toon:",omitempty"` are claimed by the plan. The revise routes empirical verification to a Unit 4.5 builder-side spike. Acceptable as a pinned spike; the spike outcome is documented in `BUILDER_WORKLOG.md` before `toon.go` is written. Not a finding.
- **`--full-name` CWD-vs-toplevel-relative emission** — pinned as Round 2 falsification probe in the Empirical Notes section. The plan's prefix-strip handles both possible outcomes correctly; the empirical answer determines whether the prefix-strip is a no-op or a substantive operation, but does not change correctness. Not a finding.
