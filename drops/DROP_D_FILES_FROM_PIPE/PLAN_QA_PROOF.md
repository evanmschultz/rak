# PLAN_QA_PROOF — DROP_D_FILES_FROM_PIPE — Round 1

**Verdict:** PASS WITH MINOR FINDINGS (planner may resolve all in-place; no axis-blocking issue).

The plan's decomposition is sound, the four units are atomic, the chain ordering is correct, the acceptance criteria are yes/no-verifiable, and every load-bearing claim about existing code has been verified against the tree. Findings below are precision asks (clarify one ambiguity, one missing dev-question, two acceptance-criteria sharpenings, one cross-stream serialization note) — none invalidate the plan.

## 1. Findings

### 1.1 Atomic decomposition — verified

- 1.1.1 [Axis: atomic-decomposition] [severity: low] D.1 = `FilesFromLister` impl + unit tests in `internal/lister/`; D.2 = `cmd/rak/root.go` wiring (flag + helper + branch); D.3 = integration tests in `cmd/rak/integration_test.go`; D.4 = docs trio. Each unit owns one production file (or one set of test additions / docs). All four fall within the 80-300 LOC/test-LOC envelope established by Drops 1–C → matches main/CLAUDE.md file-size cap.

### 1.2 Parallelization graph — verified

- 1.2.1 [Axis: parallelization-graph] [severity: low] D.1 → D.2 → D.3 → D.4 is a strict linear chain; each unit lists `Blocked by: D.{N-1}`. D.1 paths (`internal/lister/filesfrom.go`, `internal/lister/lister_test.go`) and D.2 paths (`cmd/rak/root.go`) are disjoint *but* D.2 imports the symbol introduced by D.1, so the chain is necessary. No false-positive `blocked_by` entries, no missing ones.
- 1.2.2 [Axis: parallelization-graph] [severity: medium] Notes section flags cross-stream coordination: "Streams B, C, D all add new flags to `cmd/rak/root.go`. Unit D.2 is the cmd/rak flag-wiring unit; the orchestrator must serialize it against B and C at build time." This is **correctly identified** but lives only in `## Notes` — recommend the planner promote it into D.2's `Blocked by:` line as a comment (e.g. `Blocked by: D.1 (and serialize vs B/C cmd/rak edits)`) so a future restart-recovery sweep of `PLAN.md` headers cannot miss it.

### 1.3 Specify-block well-formedness — verified

- 1.3.1 [Axis: specify-block-well-formedness] [severity: low] Each unit has Paths / Packages / Blocked by / What-to-build / Acceptance criteria. D.4 has empty `Packages:` (no Go) — correct.
- 1.3.2 [Axis: specify-block-well-formedness] [severity: low] D.1's acceptance criteria enumerate concrete bullets (test list, `-race`, `mage build`, behaviour predicates) — yes/no-verifiable.

### 1.4 Evidence verification — `Detect` not extended

- 1.4.1 [Axis: spec-conformance] [severity: low] Verified `Detect`'s signature is `Detect(ctx context.Context, root string, opts fileset.WalkOptions) (FileLister, error)` (`internal/lister/lister.go:50`). The planner's choice to bypass `Detect` in `runRoot` and construct `lister.NewFilesFromLister(r)` directly is structurally clean: `Detect` resolves "which lister for this root path?" (it does `filepath.Abs(root)`, `filepath.EvalSymlinks`, `os.Stat`, then a git probe), all of which are meaningless for a stdin/file-list source. Routing `--files-from` through `Detect` would require either adding a sentinel root value or a new opt field — both pollute `Detect`'s contract for no benefit. The plan's rationale in `## Notes` § "Factory routing decision" is correct.

### 1.5 Evidence verification — `fileset.NewFile` signature

- 1.5.1 [Axis: spec-conformance] [severity: low] Verified `NewFile(fsys fs.FS, path, relPath string) *File` at `internal/fileset/file.go:63`. Planner's D.1 step 6 spec — `fileset.NewFile(os.DirFS(dir), base, base)` — matches exactly.

### 1.6 Evidence verification — `SingleFileLister` reuse pattern

- 1.6.1 [Axis: spec-conformance] [severity: low] Verified at `internal/lister/single.go:42-49`. `SingleFileLister.List` does `dir := filepath.Dir(s.absPath)`, `base := filepath.Base(s.absPath)`, `fsys := os.DirFS(dir)`, `f := fileset.NewFile(fsys, base, base)`, `yield(f, nil)`. The planner's D.1 step 6 description mirrors this verbatim. Reuse rationale is correct: both `dirKey(relPath)` returns `"."` for the resulting file (because `base` has no slash) and `labelDirectories` rewrites `"."` to the user-supplied `rootLabel` — meaning `flags.filesFrom` will appear as the directory label, which is Option A in the planner's Q3.

### 1.7 Evidence verification — iterator contract (ctx + yield)

- 1.7.1 [Axis: spec-conformance] [severity: low] Verified the three existing listers all implement the contract the planner states for D.1:
  - `GitLister.List` (`git.go:127-203`): `if ctx.Err() != nil { yield(nil, ctx.Err()); return }` at top of every per-path iteration; `if !yield(...) { return }` at the emit site (line 200-202, F14 carry-over comment).
  - `WalkLister.List` (`walk.go:38-40`): pure delegation to `walker.Walk(ctx)` — the underlying Walker enforces both checks.
  - `SingleFileLister.List` (`single.go:36-50`): `if ctx.Err() != nil { yield(nil, ctx.Err()); return }` at top; single `yield(f, nil)` (no need to check yield return value because there's nothing after it).
  D.1's step 1, 5, 6, 7 enumeration matches this established pattern.

### 1.8 Evidence verification — `PersistentPreRunE` and sort-key validation

- 1.8.1 [Axis: spec-conformance] [severity: low] Verified `PersistentPreRunE` exists at `cmd/rak/root.go:95-100` and currently performs the `validSortKeys` lookup. Planner's D.2 step 3 says to add the `filesFrom`-vs-positional-arg mutex check "after the sort-key check" — the current `PersistentPreRunE` signature `func(_ *cobra.Command, _ []string) error` discards `args`, so the planner's added check requires changing the second parameter from `_ []string` to `args []string`. **The plan does not call this out explicitly.** Recommend an explicit note in D.2 step 3.

### 1.9 Evidence verification — `runDirectoryOpts` field set

- 1.9.1 [Axis: spec-conformance] [severity: low] Verified `runDirectoryOpts` struct at `cmd/rak/root.go:261-269` has exactly the seven fields the planner enumerates: `rootLabel`, `binary`, `langs`, `sortKey`, `sortAsc`, `maxFiles`, `renderer`. The planner's D.2 step 5 snippet uses all seven — match.

### 1.10 Evidence verification — `c.InOrStdin()` for stdin sentinel

- 1.10.1 [Axis: spec-conformance] [severity: low] The repo already proves this is the right idiom: `runRoot` (`root.go:248`) calls `counting.Count(c.InOrStdin())` for the bare-stdin path, and four existing integration tests (`integration_test.go:57, 98, 162, 199`) use `cmd.SetIn(...)` to inject stdin. Cobra's `Command.SetIn` → `InOrStdin` getter pair is the standard pipe-test pattern in this codebase. D.2's step 5 + D.3's `TestRootCmd_Integration_FilesFrom_StdinList` correctly use this pair.

### 1.11 Evidence verification — CWD resolution at List() time

- 1.11.1 [Axis: spec-conformance] [severity: low] Existing listers behave consistently with the planner's claim: `Detect` resolves `filepath.Abs(root)` once at construction time (`lister.go:51`), but the `root` is the explicit user-supplied path arg, not CWD. `WalkLister` takes a pre-built `fs.FS`, so CWD is opaque to it. `SingleFileLister` operates on `absPath` already resolved by `Detect`. For `FilesFromLister`, each path in the input list is relative to CWD-at-list-time (the user pipes a list and expects the lines to resolve against the shell's CWD when the iteration runs). The planner correctly chooses `List()` time over constructor time — putting `os.Getwd()`/`filepath.Abs` in the constructor would freeze CWD at flag-parse time, which on a long-running test or library use is wrong. Plan claim verified.

### 1.12 Evidence verification — `filepath.Clean` vs `path.Clean`

- 1.12.1 [Axis: spec-conformance] [severity: low] Planner's D.1 step 3 specifies `filepath.Clean` "since we will call `filepath.Abs` next which requires OS-native separators." Verified by reading `os.DirFS` + `os.Stat` callers in rak: `SingleFileLister.List` uses `filepath.Dir` / `filepath.Base` (OS-native), not `path.Dir`. `Detect` uses `filepath.Abs` + `filepath.EvalSymlinks`. The codebase consistently uses `filepath.*` (OS-native) for anything that hits the host filesystem and `path.*` (forward-slash) for rendered/walk-relative paths (e.g. `dirKey` at `root.go:489-498` uses `path.Dir`, and `labelDirectories` at `root.go:524-553` uses `path.Clean` + `path.Join`). Planner's `filepath.Clean` choice is correct.

### 1.13 Evidence verification — `--depth` + `--no-gitignore` no-op behaviour

- 1.13.1 [Axis: spec-conformance] [severity: low] Verified `ErrNoGitignoreInRepo` is raised exclusively inside `Detect` (`lister.go:96`), nowhere else. Skipping `Detect` in the `--files-from` branch therefore silently no-ops `--no-gitignore` — matches D.2 step 6 claim.
- 1.13.2 [Axis: spec-conformance] [severity: low] Verified `listerOpts(flags)` at `root.go:208-216` is only called by `runRoot` in the `len(args)==1` branch (`root.go:229`). Skipping `listerOpts` in the `--files-from` branch therefore silently no-ops `--depth`, `--hidden`, `--include`, `--exclude` — matches D.2 step 7 claim.

### 1.14 Evidence verification — Reader ownership

- 1.14.1 [Axis: spec-conformance] [severity: low] D.1 says "FilesFromLister does NOT close r. The caller owns the reader." D.2 step 5's `openFilesFrom` helper returns `(io.Reader, func(), error)` and the caller does `defer closer()` — that's the correct ownership boundary. For stdin (`-`), `closer` is a no-op (stdin is owned by cobra/the OS); for a real file, `closer` is `file.Close`. D.3's tests will use `strings.NewReader` which has no Close method — confirmed safe under the "caller owns" contract.

### 1.15 Acceptance criteria — yes/no-verifiable

- 1.15.1 [Axis: acceptance-criteria-coverage] [severity: low] D.1: six named tests + `mage test -race` + `mage build` — all yes/no.
- 1.15.2 [Axis: acceptance-criteria-coverage] [severity: low] D.2: `mage build`, `mage test ./cmd/rak/...`, `rak --help` substring (`--files-from` + 2 Examples), `--files-from - .` errors with `"cannot combine"`, `--files-from /nonexistent` errors. Branch ordering claim is structural — verifiable by code inspection. All yes/no.
- 1.15.3 [Axis: acceptance-criteria-coverage] [severity: low] D.3: `mage test -race`, totals match `treeExpected*` constants, conflict test error substring. Yes/no.
- 1.15.4 [Axis: acceptance-criteria-coverage] [severity: low] D.4: tape file exists, README has "Piping" section + four invocations + gif embed, gif renders after dev runs VHS. Mostly yes/no — the "gif renders correctly" is dev-eyeball (acceptable for the feature trio per the memory note).

### 1.16 Feature trio coverage in D.4 — verified

- 1.16.1 [Axis: shipped-but-not-wired] [severity: low] D.4 enumerates all three trio elements: (1) VHS tape + gif, (2) README narrative + four invocations, (3) two cobra `Example:` entries. Item (3) lives in D.2 step 4 rather than D.4 — that's fine (D.2 is the file owning the cobra command's `Example:` field) as long as D.2's `mage build` + `rak --help` substring check in acceptance prove the examples are present. ✓ — D.2's acceptance criteria do include `rak --help shows the two new Example: entries`.

## 2. Missing Evidence

- 2.1 [Axis: spec-conformance] [severity: medium] D.2 step 3's `PersistentPreRunE` change requires renaming `_ []string` → `args []string` on the second parameter. The plan should make this explicit — a builder reading just step 3 could miss the parameter-rename and write an inner closure that captures the wrong scope.

- 2.2 [Axis: acceptance-criteria-coverage] [severity: medium] **`#`-comment handling: leading whitespace.** The Scope says "Lines starting with `#` are treated as comments and skipped." D.1 step 2 says "trim whitespace, skip if empty, skip if starts with `#`." Order matters: trim *first*, then check `strings.HasPrefix(line, "#")`. The plan reads this way to me but a builder could read step 2 as a list of independent transforms rather than an ordered pipeline. Suggest tightening: "(a) `strings.TrimSpace(line)`; (b) if empty after trim, skip; (c) if trimmed line starts with `#`, skip." Also: the comment-detection rule does NOT match `git rev-list --stdin` precisely — git's stdin protocol does *not* treat `#` as a comment introducer (it treats it as a positive ref). The actual close precedent is `git rev-list --stdin --skip-comments` (introduced ~git 2.37) and the broader Unix convention (`xargs -a`, `nproc`, `/etc/hosts`, `crontab`, etc.). The Scope's "matches `git rev-list --stdin` precedent" sentence is slightly inaccurate — recommend Q4 dev signoff explicitly cite "Unix convention" rather than git as the source.

- 2.3 [Axis: acceptance-criteria-coverage] [severity: medium] **Empty stdin smoke test in D.3.** Q6 in `## Notes` says "Empty stdin (`echo -n | rak --files-from -`)… worth a manual smoke test. Builder should add a test for zero-file case in D.3." But D.3's `What to build` enumerates only `TestRootCmd_Integration_FilesFrom_StdinList`, `TestRootCmd_Integration_FilesFrom_WithComments`, and the optional conflict test. The empty-stdin case is not listed in D.3's What-to-build or acceptance. Recommend either (a) add `TestRootCmd_Integration_FilesFrom_EmptyStdin` to D.3's enumerated tests, or (b) push the zero-file coverage to D.1 (`TestFilesFromLister_EmptyReader` already exists and proves the lister yields nothing — but it doesn't prove the renderer survives a zero-file Summary, which is Q6's actual concern). End-to-end zero-file test belongs in D.3.

- 2.4 [Axis: spec-conformance] [severity: low] **`bufio.Scanner` line-length limit.** Default `bufio.Scanner` rejects lines >64 KiB (`bufio.ErrTooLong`). For path lists this is normally fine — even a deeply-nested macOS path tops out around 1 KiB — but a malicious or programmatically-generated input could blow past this. The plan should either explicitly accept the 64 KiB default ("paths over 64 KiB are not supported") or call `scanner.Buffer` to raise the cap. Recommend a one-line accept in D.1's What-to-build.

- 2.5 [Axis: spec-conformance] [severity: low] **Symlink handling.** `os.Stat` follows symlinks (`os.Lstat` does not). `Detect`'s `SingleFileLister` path calls `filepath.EvalSymlinks` *before* `os.Stat`, so a symlink-to-regular-file works correctly through `Detect`. D.1's plan uses `os.Stat` directly — a symlink in the input list will be followed (matching `Detect`'s behaviour), but a broken symlink will surface as `os.Stat → fs.ErrNotExist` and yield the "not a regular file" error. That's probably the right behaviour, but the plan should state it explicitly: "symlinks are followed via `os.Stat`; broken symlinks aggregate as not-a-regular-file errors."

- 2.6 [Axis: acceptance-criteria-coverage] [severity: low] **`mage ci` not in D.2 acceptance.** D.2 acceptance criteria say `mage build` + `mage test ./cmd/rak/...`. The full pre-push gate is `mage ci`, but per `main/drops/WORKFLOW.md` § "Phase 6 — Verify," `mage ci` is **drop-end** (after all units pass build-QA), not per-unit. So `mage build`+`mage test` per-unit is correct. No finding here — the line was tempting to flag but the workflow doc clears it.

## 3. Summary

PASS. The plan's evidence is sound and every load-bearing claim about existing code (`Detect` signature, `fileset.NewFile`, `SingleFileLister` reuse, iterator contract, `PersistentPreRunE`, `runDirectoryOpts`, branch-skipping for `--no-gitignore` / `--depth`, `filepath.*` vs `path.*` discipline) has been verified against the source tree. The decomposition is atomic, the chain ordering is correct, the acceptance criteria are yes/no-verifiable.

Findings 2.1–2.5 are precision asks for the planner (one parameter-rename note, one comment-pipeline clarification + Unix-not-git precedent fix, one missing empty-stdin test, one scanner-buffer-cap note, one symlink-policy statement). All are safely resolvable in-place in PLAN.md without restructuring units. Finding 1.2.2 is a cross-stream serialization reminder — promote to a more visible location than `## Notes`.

No axis-blocking issue.

## TL;DR

- T1 Decomposition + chain + specify-block well-formedness all verified — four atomic units, linear chain D.1→D.2→D.3→D.4, acceptance criteria yes/no-verifiable.
- T2 Five precision asks for the planner: explicit `PersistentPreRunE` parameter rename, comment-pipeline ordering + Unix-not-git precedent, empty-stdin integration test, `bufio.Scanner` 64 KiB cap policy, symlink-follow policy. PASS overall.
- T3 — (no separate Section 3 in body; verdict folded into Section 3 above)
