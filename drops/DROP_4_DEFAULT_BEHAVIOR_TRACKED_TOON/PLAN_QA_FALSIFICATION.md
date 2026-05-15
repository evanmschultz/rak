# Drop 4 — Plan QA Falsification, Round 2

**Verdict:** pass-with-findings

## Empirical verification (Decision E)

Sandboxed `cd`/`mkdir` to `/tmp` were blocked, so the Round-1 planner's experiment was reproduced inside the rak checkout using `git -C` to switch effective CWD. The combination is more probative than the requested /tmp fixture because it spans an actual two-level subdirectory plus exercises the worktree case (`main/` is a git worktree, not a primary checkout).

### Commands actually run

```
git -C main rev-parse --show-toplevel
git -C main ls-files --full-name -z | tr '\0' '\n' | head -8

git -C main/internal/fileset rev-parse --show-toplevel
git -C main/internal/fileset ls-files --full-name -z | tr '\0' '\n' | head -10
git -C main/internal/fileset ls-files -z | tr '\0' '\n' | head -10
```

### Output

`git -C main rev-parse --show-toplevel` → `/Users/evanschultz/Documents/Code/hylla/rak/main` (the worktree root, confirming F16 + the planner's "Worktrees" Edge Case paragraph).

`git -C main ls-files --full-name -z` from the toplevel emits toplevel-relative paths (`.github/workflows/ci.yml`, `.gitignore`, `.golangci.yml`, `CLAUDE.md`, `LICENSE`, `PLAN.md`, `README.md`, `cmd/rak/integration_test.go`, …).

`git -C main/internal/fileset rev-parse --show-toplevel` → `/Users/evanschultz/Documents/Code/hylla/rak/main` (same toplevel — git resolves up to the worktree root regardless of the `-C` subdir).

`git -C main/internal/fileset ls-files --full-name -z` emits **toplevel-relative paths SCOPED TO THE SUBDIR**:
```
internal/fileset/binary.go
internal/fileset/binary_test.go
internal/fileset/file.go
internal/fileset/file_test.go
internal/fileset/walker.go
internal/fileset/walker_test.go
```

`git -C main/internal/fileset ls-files -z` (no `--full-name`) emits **CWD-relative paths SCOPED TO THE SUBDIR**:
```
binary.go
binary_test.go
file.go
file_test.go
walker.go
walker_test.go
```

### Conclusions

1. **`--full-name` emits toplevel-relative paths regardless of CWD.** This resolves Decision E definitively. The Round-1 planner's hypothesis "either toplevel-relative or CWD-relative" collapses to the first branch. F17's prefix-strip step is therefore **always active** (non-no-op) whenever `absRoot != toplevel`. The "either possible behavior" hedge in the Empirical Notes paragraph and inside F17 can be tightened: the prefix strip is unconditionally required for sub-directory walk roots.

2. **`git ls-files` (with or without `--full-name`) automatically scopes its output to the CWD subtree.** The planner's PLAN.md treats this implicitly (the prefix-strip step filters non-matching entries with `strings.HasPrefix` and then skips), but it's worth pinning: with `cmd.Dir = absRoot`, only files inside `absRoot/...` are emitted. The "skip entries that don't start with `g.prefix`" guard in step 1 of F18/Unit 4.2's per-path loop is therefore a defensive guard against an OS-edge-case that does not occur with current git; it should remain (cheap) but its real role is documentation of the invariant, not an active filter.

3. **F17 prefix computation is correct** under the empirical behavior. For `absRoot = /Users/.../main/internal/fileset` and `toplevel = /Users/.../main`, `filepath.ToSlash(strings.TrimPrefix(absRoot, toplevel))` → `/internal/fileset`; then stripping the leading `/` gives `internal/fileset` — which is exactly the prefix git emits in every row. Stripping `prefix + "/"` produces `file.go`, `walker.go`, etc. (walk-root-relative). The F17 wording in the revised plan handles both cases conservatively; it works.

4. **Worktree behavior is benign.** `show-toplevel` from a worktree returns the worktree root, not the bare repo. The drop's own dev environment is a worktree, so daily use validates this.

**Suggested tightening (non-blocking):** trim the Empirical Notes "either way F17's implementation handles it correctly" hedge to a single sentence: "Empirical verification (Round 2): `git ls-files --full-name -z` emits toplevel-relative paths regardless of CWD; the prefix-strip in F17 is always active for sub-directory walk roots."

## Round 1 carry-over verdict

All 15 Round 1 counterexamples are dead in the revised plan:

- **C1 / C2 / C6 (runTreeFS + integration fixture vendor/)**: revise pins `runTreeFS` to construct `lister.NewWalkLister(fsys, ".", listerOpts(flags))` directly, bypassing `Detect`. Unit 4.4's acceptance bullet now says this verbatim. Integration tests get their own treatment: the testdata/tree fixture is fully git-tracked, so `GitLister` will see exactly the five files Drop 3's Walker saw, and the effective set after hidden+binary filtering matches.
- **C3 (Windows `filepath.Separator`)**: revise switched F17 to `filepath.ToSlash(strings.TrimPrefix(absRoot, toplevel))` then strip a leading `/`. Both operands are forward-slash before the trim, sidestepping the Windows backslash issue. (rak's stated target is Unix, but the revise removes the latent bug regardless.)
- **C4 (`anySegmentHidden`)**: revise added the `anySegmentHidden` helper in Unit 4.2 with explicit per-segment iteration over `strings.Split(relPath, "/")`. The contract is now machine-readable.
- **C5 (`--no-gitignore` no-op)**: revise replaced F19's silent no-op with Decision A — hard error via `ErrNoGitignoreInRepo`, surfaced before `newGitLister` is constructed. Drop F19's no-op semantic, lose the user-trust violation.
- **C7 / C8 (toon-go omitempty + pipe escaping)**: revise added an explicit spike requirement at the top of Unit 4.5 with documented decision branches (omitempty fall back to conditional shaping; pipe-unsafe fall back to tab delimiter).
- **C9 (edge cases)**: revise added the "Git Edge Cases (Decision D)" Notes paragraph addressing submodules, sparse-checkout, and worktrees explicitly.
- **C10 (F17 empty-prefix off-by-one)**: revise rewrote the prefix computation to `filepath.ToSlash(strings.TrimPrefix(absRoot, toplevel))` + leading-slash strip; the empty case is now natural (`TrimPrefix("/foo", "/foo")` = `""`, then strip leading `/` is a no-op on `""`).
- **C11 (forward-declaration / 4.1 compile)**: revise accepted the compile break explicitly with the "Compile note (C11)" bullet in Unit 4.1, scoping `mage test` at the 4.1 boundary to packages other than `./internal/lister/...`. This is a deliberate trade-off documented in plan prose, not hand-waved to the builder.
- **C12 (UN3 hidden-file vs decision 34)**: revise resolved UN3 via Decision B (hidden filter applies in GitLister mode); F21 captures it explicitly with the "presentation layer vs tracking layer" framing.
- **C13 (`TestRootCmd_UnknownFlag`)**: revise added `TestRootCmd_UnknownFlag` as a third replacement in Unit 4.4.
- **C14 (`walkAndCount` signature audit)**: revise pinned `runDirectory(ctx, source lister.FileLister, rootLabel string, renderer render.Renderer) error` and `walkAndCount(ctx, source FileLister, binary bool)`. The `binary` bool threading is explicit. There is one minor residual gap — see C2.1 below.
- **C15 (F18 depth `>` vs `>=`)**: revise reworded F18(b) to "`strings.Count(relPath, "/") >= opts.Depth`" matching Walker's `depth >= w.opts.Depth` at `walker.go:223,226`.

No Round 1 CE reappears. Round 1's verdict was "fail" with 15 unmitigated counterexamples; the revise legitimately closed all of them.

## New counterexamples (Round 2 specific)

### Counterexample C2.1 — `runDirectory` parameter list in PLAN.md text drops `binary bool` from the signature line

- **Severity:** minor
- **Attack target:** Unit 4.4 acceptance — `runDirectory(ctx context.Context, source lister.FileLister, rootLabel string, renderer render.Renderer) error`
- **Construction:** The same paragraph defines `walkAndCount` as `walkAndCount(ctx context.Context, source lister.FileLister, binary bool) ([]render.Directory, counting.Counts, []error, error)` — explicitly accepting `binary bool`. But the immediately-preceding `runDirectory` line lists only `(ctx, source, rootLabel, renderer) error`. `runDirectory` is what `runRoot` calls; `walkAndCount` is what `runDirectory` calls. The binary flag must reach `walkAndCount`, which means either (a) `runDirectory` also takes `binary bool` and passes it through, or (b) `runRoot` calls `walkAndCount` directly and `runDirectory` only handles the post-walk path. The PLAN.md text picks neither — it gives `runDirectory` a 4-parameter signature and `walkAndCount` a 3-parameter signature with `binary` appearing out of thin air. Round 1's C14 explicitly flagged this and asked the planner to pin one choice; the revise pinned `walkAndCount`'s signature but left `runDirectory` unpinned for the binary threading. The builder will resolve it (probably correctly) but the plan still hands them an under-specified seam.
- **Mitigation status:** UNMITIGATED — small textual fix.
- **Suggested fix:** Either (a) add `binary bool` to the `runDirectory` signature line in Unit 4.4 acceptance: `runDirectory(ctx context.Context, source lister.FileLister, rootLabel string, binary bool, renderer render.Renderer) error`, or (b) explicitly say "`runRoot` calls `walkAndCount` directly with the binary bool; `runDirectory` is the wrapper that labels output and runs the renderer." Pin one. This is the same exact gap C14 raised — the planner closed the `walkAndCount` side but not the `runDirectory` side.

### Counterexample C2.2 — Unit 4.4's "`TestRootCmd_NoGitignoreInRepo_Errors` ... Builder decides cleanest form" reintroduces the Round 1 C2 anti-pattern at a smaller scope

- **Severity:** minor
- **Attack target:** Unit 4.4 acceptance — `TestRootCmd_NoGitignoreInRepo_Errors` test description ("Builder decides cleanest form").
- **Construction:** Round 1's C2 attacked "Builder decides cleanest form" as a load-bearing-decision hand-waved away. The revise removed it from the `runTreeFS` paragraph (good) but reintroduced it inside the `TestRootCmd_NoGitignoreInRepo_Errors` test description: "Builder decides cleanest form. Skips if git binary absent." This particular test is small enough that the choice is not critical (it's a smoke test, not a contract test), but the principle still bites — the build-QA falsification agent for Unit 4.4 will not be able to assess whether the builder's choice is correct without re-asking the planner. The two options are not equivalent: setting up a real temp git repo (option 1) exercises `lister.Detect` end-to-end including the `git rev-parse` shell-out, while invoking through cobra (option 2) exercises only the wrapping layer.
- **Mitigation status:** UNMITIGATED — small wording fix.
- **Suggested fix:** Pin the test form: "Construct a `t.TempDir()` + `git init` (skip if `exec.LookPath("git")` fails), then call `newRootCmd().Execute()` with `SetArgs([]string{"--no-gitignore", tmpDir})`. Verify the resulting error wraps `lister.ErrNoGitignoreInRepo` via `errors.Is`." This pins the broader-coverage end-to-end form and removes the builder's discretion on a contract test.

### Counterexample C2.3 — `ErrNoGitignoreInRepo` error message embeds a forward-pointing reference to v0.2 that the plan does not commit to

- **Severity:** minor
- **Attack target:** Unit 4.4 acceptance — error message string: "rak counts git-tracked files in this mode. To count untracked files, run rak outside the repository, **or wait for the v0.2 --include-untracked flag**."
- **Construction:** The error message tells the user a `--include-untracked` flag is coming in v0.2. `main/PLAN.md` does not have a `v0.2` milestone or a `--include-untracked` flag in any drop's planning notes (verified by reading the revise's prose — no other plan references this flag). If v0.2 ships without that flag, every existing user who hit this error message is left with a dangling forward-reference. A future drop adding "--include-untracked" would either need to keep the name (locking the v0.2 design now) or break the promise. Two options: (a) name the flag in `main/PLAN.md`'s future-work section so the message has a target, or (b) remove the forward reference and just say "To count untracked files, run rak outside the repository." The cleaner option is (b) — the error message should not commit the future.
- **Mitigation status:** UNMITIGATED.
- **Suggested fix:** Strip "or wait for the v0.2 --include-untracked flag" from the error message in both Unit 4.4 acceptance and F19 to: "rak: --no-gitignore has no effect when run inside a git repository. rak counts git-tracked files in this mode. To count untracked files, run rak outside the repository." If the dev wants to commit to a future flag name, add a one-line entry to `main/PLAN.md`'s decision log first.

## Attack passes that produced no counterexample (EXHAUSTED)

The Round-2 attack surface enumerated in the spawn prompt was attacked exhaustively. Findings landed only on the three above. The remaining attack families are documented as EXHAUSTED with their evidence:

1. **`fileset.NewFile` export attack** (spawn prompt §1).
   - Signature attack: revised plan pins `func NewFile(fsys fs.FS, path, relPath string) *File`. This matches the existing unexported `newFile` body verbatim (`internal/fileset/file.go:52-58`) including parameter order. No invariants are lost — the wrapper is a thin trampoline. EXHAUSTED.
   - Drop 3 test breakage: `internal/fileset/file_test.go` calls `newFile` directly (lines 21, 45, 110, 130). Adding an exported `NewFile` wrapper does not touch the unexported `newFile`; tests continue to compile and pass. EXHAUSTED.
   - YAGNI / "could lister move INTO fileset?" attack: lister's package boundary is justified by (a) lister consumes `os/exec` for git shelling, which would pollute fileset's clean dependency surface (fileset currently imports only `io/fs`, `bufio`, `bytes`, etc. — no `os/exec`); (b) `Detect` is a CLI-policy concern (in-git vs not), not a fileset-walk concern; (c) the package keeps two implementations behind one interface, which is the textbook use case for a separate package. EXHAUSTED.

2. **`ErrNoGitignoreInRepo` sentinel attack** (spawn prompt §2).
   - Package location: F19 + Unit 4.1 acceptance pin it in `internal/lister/lister.go`. EXHAUSTED.
   - `errors.Is` consumer: Unit 4.4 acceptance says "Surface `lister.ErrNoGitignoreInRepo` as a normal `RunE` error return — cobra prints the wrapped message"; the test `TestRootCmd_NoGitignoreInRepo_Errors` is required to call `errors.Is`. EXHAUSTED (modulo C2.2's pinning gap).
   - README / `--version` integration: not in Drop 4 scope; deferring to Drop 9 is consistent with rak's "no ceremony" memory rule. EXHAUSTED.

3. **`anySegmentHidden` helper attack** (spawn prompt §3).
   - Empty `relPath`: `strings.Split("", "/")` returns `[""]` (one-element slice with empty string); `fileset.IsHidden("")` returns false per `internal/fileset/file.go:118`. Result: `anySegmentHidden("")` returns false — correct (empty path is the walk root, never hidden). EXHAUSTED.
   - Leading/trailing slashes: GitLister builds `relPath` from `git ls-files --full-name` output AFTER prefix-stripping `prefix + "/"` (F17). The empirical run shows git emits no leading-slash paths and no trailing-slash paths for files. After prefix-strip the result is forward-slash-clean. EXHAUSTED.
   - `..` / `.` segments: `git ls-files` enforces index-path validity; `..` and `.` are not valid index entries. A corrupted index would surface as a `git` exit-code-1 before any path emission. EXHAUSTED.
   - Walker-comparison: Walker's hidden filter calls `IsHidden(d.Name())` per-entry (one segment at a time, via `fs.WalkDir`); `anySegmentHidden` simulates the same effect by iterating each segment of the materialized `relPath`. Equivalent semantics modulo the materialization order. EXHAUSTED.

4. **4.1 deliberate-compile-break attack** (spawn prompt §4).
   - WORKFLOW.md Phase 5 compliance: WORKFLOW.md § "Phase 6 — Verify" / "Per-unit verification" says "builder runs `mage build` + `mage test` for the touched packages." The revise's Unit 4.1 Compile note scopes the at-4.1 verification to `./cmd/...`, `./internal/counting/...`, `./internal/fileset/...`, `./internal/ignore/...`, `./internal/render/...`, `./internal/summary/...`, `./internal/tokens/...` — all the touched packages **except** `./internal/lister/...`. The "touched package" wording is ambiguous: 4.1 creates `internal/lister/` so it's "touched," but the package is provably incomplete until 4.2 lands. The revise's choice — accept the build break on `./internal/lister/...` only — is a coherent reading of "touched packages" when the package is in mid-construction. The build-QA falsification on Unit 4.1 will need a paragraph saying "Build break on `./internal/lister/...` is expected per the Compile note in PLAN.md Unit 4.1; verification scope at this commit is the listed packages." EXHAUSTED (with the QA hand-off note recorded above).
   - Collapse 4.1+4.2+4.3 into one unit alternative: viable but rejected by the planner (the per-unit blast radius is "lister package as one unit" vs "lister package as three units of ~150 LOC each"). The three-unit decomposition fits rak's "atomic granularity" rule in `main/CLAUDE.md` § "Drops" better than the bundle; the compile-break is the cost. EXHAUSTED.

5. **Decision E F17 conservative resolution attack** (spawn prompt §5). Resolved above in the empirical section. F17's prefix-strip is **always active for sub-directory walk roots** and a no-op when `absRoot == toplevel`. The revise's "either possible behavior" hedge can be tightened to a single empirical statement, but the implementation is correct as-is. EXHAUSTED.

6. **`Detect(ctx, root, opts)` without `fsys` attack** (spawn prompt §6).
   - `os.DirFS(absRoot)` vs `os.DirFS(toplevel)`: revise pins `GitLister.fsys = os.DirFS(absRoot)`. F26 + Unit 4.2's per-path emit `yield(fileset.NewFile(g.fsys, relPath, relPath), nil)` where `relPath` is **walk-root-relative**. So `Open(relPath)` on `os.DirFS(absRoot)` opens `absRoot/relPath`, which is the correct file. EXHAUSTED.
   - Round 1's C6 (CWD inside rak repo for unit tests): mitigated by the runTreeFS pin (constructs `NewWalkLister` directly, bypasses `Detect`). EXHAUSTED.

7. **Builder-side toon-go spike attack** (spawn prompt §7).
   - Pass/fail criterion: the spike has two yes/no branches in the plan ("if `omitempty` works..." and "if the lib doesn't escape embedded pipes..."). Each branch's fallback is specified (conditional struct shaping; tab delimiter). EXHAUSTED.
   - Spike output durability: the plan says "Builder documents the results in `BUILDER_WORKLOG.md` § 'Spike: toon-go behavior'" — `BUILDER_WORKLOG.md` is durable per WORKFLOW.md § "File Lifecycle". EXHAUSTED.
   - Spike → build-QA gate: the spike result is recorded BEFORE `toon.go` is written, so build-QA can audit the spike output and the implementation in the same review. EXHAUSTED.

8. **UN2 (TOON shape vs Drop 5 columns) attack** (spawn prompt §8). The revise's recommendation — "emit minimal now; Drop 5 adds columns" — is deferrable: TOON arrays can grow columns without breaking earlier consumers IF the consumer reads keyed columns (the tabular `{bytes,lines,words,chars}` header tells the reader which positions hold which metric). Drop 5 adding a `blank`/`comment`/`code` column appends header positions; existing readers that look up `bytes`/`lines`/`words`/`chars` by name still work. The decision can stay deferred to Phase 3 dev confirmation. EXHAUSTED.

9. **Memory-rule and convention check** (spawn prompt §9).
   - `feedback_rak_no_ceremony.md`: revised plan does NOT add any of the forbidden ceremony files (LEDGER, WIKI, REFINEMENTS, HYLLA_FEEDBACK, CLOSEOUT). The drop dir contains only `PLAN.md` + `BUILDER_WORKLOG.md` per the template. EXHAUSTED.
   - `feedback_naming_conventions.md`: drop directory name `DROP_4_DEFAULT_BEHAVIOR_TRACKED_TOON` is `ALL_UPPERCASE_WITH_UNDERSCORES`. Unit IDs `4.0`/`4.1`/.../`4.5` are positional identifiers (numerals, not user-facing labels). EXHAUSTED.

## Summary

Three new minor counterexamples (C2.1, C2.2, C2.3). All three are small wording fixes inside Unit 4.4 (or its referenced F-pins) — none are structural and none block the build. The plan is otherwise ready to advance.

If the dev wants a clean Round 2 close: fold the three fixes into a planner round-2.1 in-place edit and re-run plan-QA, or accept them with the dev's signoff in Phase 3 and advance to Phase 4 with the minor fixes captured in the planner brief.
