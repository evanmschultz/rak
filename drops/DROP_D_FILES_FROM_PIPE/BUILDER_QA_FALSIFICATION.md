# DROP_D — Build QA Falsification

Append a `## Unit N.M — Round K` section per QA attempt. See `main/drops/WORKFLOW.md` § "Phase 5 — Build QA (per unit)".

## Unit D.1 — Round 1

**Verdict:** PASS WITH FINDINGS

Implementation is correct against every PLAN.md acceptance criterion. `mage test` is green. No blockers reproduced across the eight attack vectors. Two cosmetic concerns and three coverage / doc nits worth recording for future cleanup; none warrant a Round 2 spin on D.1.

### Counterexamples / Attacks

#### Attack 1 — `os.Stat` vs `os.Lstat` symlink semantics

- **Severity:** nit (intentional)
- **Where:** `internal/lister/filesfrom.go:99`
- **Counterexample:** Line 99 uses `os.Stat`, which follows symlinks. A symlink in the input list that points to a regular file is yielded as if it were the target file; a symlink pointing to a directory yields the "not a regular file" friendly error; a broken symlink yields a stat-wrapped error.
- **Mitigation accepted:** PLAN.md § Notes "Symlink behavior" explicitly documents this: *"`os.Stat` (used in `FilesFromLister.List`) follows symlinks. The symlink target is counted, not the symlink entry itself. Matches v0.1.4 `SingleFileLister` behavior. Consistency intentional."* Behavior matches `wc somelink` and the `SingleFileLister` / `Detect` path (which uses `filepath.EvalSymlinks` for the same effect). Accepted.

#### Attack 2 — Relative `..` traversal, embedded spaces, trailing whitespace, whitespace-only lines

- **Severity:** none (REFUTED)
- **Where:** `internal/lister/filesfrom.go:84-95`
- **Counterexample attempts:**
  - `../parent/foo.txt` — `filepath.Clean` + `filepath.Join(cwd, ...)` resolves correctly to `<cwd>/../parent/foo.txt` → cleaned to `<parent>/parent/foo.txt`.
  - `My Documents/foo.txt` — `strings.TrimSpace` only trims leading/trailing whitespace; `filepath.Clean` + `os.Stat` handle embedded space natively.
  - Trailing `\t`, leading spaces — trimmed by `strings.TrimSpace` (line 84).
  - Whitespace-only line — trimmed to `""`, skipped by line 85.
- **Mitigation accepted:** All four sub-attacks REFUTED. Implementation handles each correctly. (Minor nit: a file literally named `" foo.txt"` with leading space cannot be referenced through this lister because `TrimSpace` strips it — but this is canonical Unix tool behavior matching `xargs` and is not worth a NUL-delimited variant in v0.2.0; the PLAN already defers `--files0-from` to v0.2.1/v0.3.)

#### Attack 3 — Iterator contract: yield-false and per-line error continuation

- **Severity:** none (REFUTED)
- **Where:** `internal/lister/filesfrom.go:101-122`
- **Counterexample attempts:**
  - `yield(nil, err)` then continue to next iteration: tested by `TestFilesFromLister_MissingFile`; bad path yields one error, valid path still yielded. Implementation correct.
  - `yield` returns false mid-stream: all three "main" yield sites (lines 102, 109, 121) check the return value and `return` on false (F14 carry-over honored).
- **Mitigation accepted:** REFUTED. Three terminal yield sites do NOT check return value (lines 68 `getwd`, 76 `ctx.Err()`, 128 `scanner.Err()`) — but each is immediately followed by an unconditional return or by the natural end of the iterator function, so functionally they cannot leak iterations. Style nit only.

#### Attack 4 — Context cancellation between iterations

- **Severity:** none (REFUTED)
- **Where:** `internal/lister/filesfrom.go:75-78`
- **Counterexample:** Line 75 checks `ctx.Err()` at the top of every loop iteration. Tested by `TestFilesFromLister_ContextCancel`. The test cancels after the first yield, asserts subsequent iteration yields a context error and stops.
- **Caveat (out of D.1 scope):** Cancellation during a blocked `scanner.Scan()` on a slow stdin pipe is NOT honored mid-read because `bufio.Scanner` does not respect context. The next iter catches it. Acceptable for a per-line iterator; PLAN does not claim per-read cancellation. Not a bug.

#### Attack 5 — `scanner.Err()` placement

- **Severity:** none (REFUTED)
- **Where:** `internal/lister/filesfrom.go:127-129`
- **Counterexample:** Verified that `scanner.Err()` is checked AFTER the `for { ... }` loop exits, not inside the loop body. Correct placement.

#### Attack 6 — Reader ownership

- **Severity:** none (REFUTED)
- **Where:** `internal/lister/filesfrom.go` (entire file)
- **Counterexample:** Searched the file for any `r.Close()`, `Close(`, or `defer` against `fl.r`. None present. Doc comment at lines 24-25 and 41 explicitly states "caller owns the reader". Reader ownership respected.

#### Attack 7 — CWD captured at iteration-start vs `List()` call-time

- **Severity:** nit (doc accuracy)
- **Where:** `internal/lister/filesfrom.go:46-48` (doc comment) and `internal/lister/filesfrom.go:66` (implementation)
- **Counterexample:** The implementation calls `os.Getwd()` INSIDE the returned `iter.Seq2` closure (line 66), which executes on first `for ... range` iteration, NOT when `List(ctx)` is called. If a caller does:
  ```go
  it := fl.List(ctx)
  os.Chdir("/elsewhere")
  for f, e := range it { ... }
  ```
  CWD will be `/elsewhere`, not the CWD at the `List()` call. The doc comment line 47-48 says *"CWD is resolved once at the start of List"* which is technically misleading — it's resolved at the start of *iteration*. PLAN.md acceptance criterion line 135-136 ("CWD resolution happens in `List()`, not the constructor") is satisfied because the closure IS in `List()`'s lexical scope, even though it executes lazily.
- **Mitigation accepted:** Functionally correct (in fact arguably better — captures the CWD active when iteration begins, which is what most callers want). Doc nit only. Suggest tightening the comment to "CWD is resolved once at the start of iteration" in a future cleanup pass.

#### Attack 8 — Concurrent `List()` / iteration on same `FilesFromLister`

- **Severity:** nit (doc gap)
- **Where:** `internal/lister/filesfrom.go:36-44` (struct + constructor doc)
- **Counterexample:** Two goroutines calling `fl.List(ctx)` and iterating concurrently each get an independent `bufio.Scanner`, but BOTH read from the same `fl.r io.Reader`. Most reader types (`*os.File`, `*strings.Reader`, `*bytes.Buffer`) are not safe for concurrent reads — would produce interleaved bytes, data races, or panics. The struct/constructor docs do not warn against this.
- **Mitigation accepted:** The other listers (`SingleFileLister`, `WalkLister`, `GitLister`) don't document concurrent-safety either; the iterator contract is implicitly single-consumer in the FileLister interface comment. Realistic concurrent-iteration on the same lister is essentially nil. Accepted; would be nice to add a one-line "single-consumer" note to the doc comment in a future pass.

#### Attack 9 — Non-regular path yields awkward error string

- **Severity:** concern (cosmetic / user-visible)
- **Where:** `internal/lister/filesfrom.go:107`
- **Counterexample:** If a directory (or named pipe, socket, etc.) is in the input list, line 99 `os.Stat` succeeds, line 106 `info.Mode().IsRegular()` returns false, line 107 yields:
  ```
  lister: files-from: "/tmp/somedir" is not a regular file: not a regular file
  ```
  The trailing `": not a regular file"` is redundant. The stat-error path (line 101) wraps an underlying error with `: %w`, which is idiomatic; the non-regular path imitates that format but has no underlying error to wrap, so it just repeats the phrase.
- **Mitigation accepted (with suggested fix):** Cosmetic only — functionality is correct. Suggested fix: drop the trailing `": not a regular file"` and use a cleaner message like `fmt.Errorf("lister: files-from: %q is not a regular file", line)`. Not a Round 2 blocker; queue for a future cleanup commit or roll into the D.4 / docs polish phase if convenient.

#### Attack 10 — Coverage gap: directory in input list

- **Severity:** nit (test coverage)
- **Where:** `internal/lister/lister_test.go` (FilesFromLister section)
- **Counterexample:** The "not a regular file (non-stat-failure)" branch — feeding a directory path through the lister — is implemented at line 106-110 but is NOT exercised by any test. `TestFilesFromLister_MissingFile` covers the `os.Stat` failure path; nothing covers the IsRegular-false path. Combined with Attack 9 (awkward error string), this is the path where users hitting `find . -type d | rak --files-from -` see the doubled "not a regular file" message — and no test would catch a regression in that branch.
- **Mitigation accepted:** Nit only. Adding `TestFilesFromLister_DirectoryYieldsFriendlyError` would close the gap; reasonable to bundle with the Attack 9 message-cleanup fix in a future cleanup commit. Not a Round 2 blocker.

#### Attack 11 — Coverage gap: mid-stream scanner error

- **Severity:** nit (test coverage)
- **Where:** `internal/lister/lister_test.go`
- **Counterexample:** Lines 127-129 implement `scanner.Err()` propagation, but no test feeds a reader that returns a non-EOF error mid-stream (e.g. `iotest.ErrReader` or a custom reader returning `io.ErrUnexpectedEOF`). The empty-reader test exercises clean EOF only. A regression that, for example, moved the `scanner.Err()` check inside the loop body would not be caught by existing tests.
- **Mitigation accepted:** Nit only. Suggest adding `TestFilesFromLister_ReaderError` using `iotest.ErrReader` in a future test-coverage pass. Not a Round 2 blocker.

### Summary

- 11 attack vectors attempted. 6 REFUTED outright (Attacks 2, 3, 4, 5, 6, plus most of 1).
- 0 CONFIRMED blockers.
- 1 concern (Attack 9: awkward error string for non-regular path).
- 4 nits (Attacks 1 symlink doc note, 7 doc comment accuracy, 8 concurrent-iter doc gap, 10/11 test coverage).
- `mage test` is green (full suite). Round 2's CWD/absolute-path fix is verified working against `TestFilesFromLister_HashPrefixedFileWorks`, `TestFilesFromLister_MixedPaths`, `TestFilesFromLister_SkipsEmptyLines`, `TestFilesFromLister_MissingFile`.
- Unit D.1 is GO-FOR-CLOSE. Findings are queued for a future cleanup pass (rolled into D.4 docs polish or a v0.2.0 follow-up commit), not blocking.

### Hylla Feedback

N/A — Unit D.1 implementation is freshly-committed; Hylla indexes the previous baseline. All evidence gathered from `Read` against the working tree. No Hylla query was warranted (the file hasn't been ingested yet).

## Unit D.2 — Round 1

**Verdict:** PASS

Twelve targeted attacks attempted against the orchestrator-supplied angle list. Zero CONFIRMED counterexamples. Two doc / cosmetic nits (non-blocking) and one design-pinned consistency observation. `mage build` passes; `mage test ./cmd/rak/...` is green (`ok github.com/evanmschultz/rak/cmd/rak 1.508s`). The pre-existing `internal/lang` build failure (undefined `LangJSX`, `LangTSX`, …) is unrelated to D.2 and was explicitly excluded from this round per orchestrator directive.

### Counterexamples / Attacks

#### Attack 1 — Branch order: `--files-from` must execute BEFORE `len(args) == 1`

- **Severity:** none (REFUTED)
- **Where:** `cmd/rak/root.go:248–268` (files-from branch) vs `cmd/rak/root.go:270–288` (positional branch)
- **Counterexample attempt:** `rak --files-from - .` — `PersistentPreRunE` Guard A fires first and short-circuits before `runRoot` is reached, so the branch-order question is moot for that input. For `rak --files-from foo.txt` (no positional), `runRoot` is entered with `args=[]`, so `len(args)==1` is false regardless of ordering. The "wrong-branch" scenario requires `filesFrom != "" && len(args) == 1`, which Guard A blocks at PreRunE — impossible to reach `runRoot` with both set.
- **Verification:** `runRoot` line 248's `if flags.filesFrom != ""` is the first conditional after `renderer := resolveRenderer(flags)`. The `if len(args) == 1` at line 270 is structurally second. The bare-stdin fallthrough at line 290 is third. Order matches spec step 5.
- **Mitigation accepted:** Spec-conformant. REFUTED.

#### Attack 2 — Guard ordering in `PersistentPreRunE` (Guard A vs Guard B independence)

- **Severity:** none (REFUTED)
- **Where:** `cmd/rak/root.go:104–112`
- **Counterexample attempts:**
  - `rak --files-from - --no-gitignore .` (both conditions): Guard A (positional) fires first because the conditional sits above Guard B (line 107 < line 110). Both errors are reachable, but only one fires per call. Either error satisfies the user-visible contract.
  - `rak --files-from -` (no positional, no `--no-gitignore`): both guards skipped via the `flags.filesFrom != ""` short-circuit AND because the inner condition is false. CORRECT.
  - `rak --no-gitignore .` (no `--files-from`): both guards skipped because the leading `flags.filesFrom != ""` is false. Walk proceeds normally with `--no-gitignore` honored. CORRECT.
  - `rak --files-from -` with `--no-gitignore` UNSET, `args=[]`: both guards skipped. CORRECT.
- **Mitigation accepted:** Each guard is independently gated on `flags.filesFrom != ""` AND its specific second condition. Neither incorrectly fires when only the other's preconditions hold. REFUTED.

#### Attack 3 — `openFilesFrom("-")` closer accidentally closes stdin

- **Severity:** none (REFUTED)
- **Where:** `cmd/rak/root.go:307–308`
- **Counterexample attempt:** Inspected the closer body: `func() {}` — empty function, no `stdin.Close()`, no reference to `os.Stdin`. The reader returned IS `stdin` (the value passed in, normally `c.InOrStdin()`), but the closer does NOT call any method on it.
- **Mitigation accepted:** Closer is a true no-op. Stdin remains process-owned. REFUTED.

#### Attack 4 — `openFilesFrom(path)` failure path returns `(nil, nil, err)` causing nil-deref on `defer closer()`

- **Severity:** none (REFUTED)
- **Where:** `cmd/rak/root.go:249–253` (caller) and `cmd/rak/root.go:310–313` (callee)
- **Counterexample attempt:** When `os.Open` fails, line 312 returns `(nil, nil, fmt.Errorf("--files-from: %w", err))`. Caller at line 249–252:
  ```go
  r, closer, err := openFilesFrom(flags.filesFrom, c.InOrStdin())
  if err != nil {
      return err
  }
  defer closer()
  ```
  The `if err != nil { return err }` check (line 250–252) executes BEFORE `defer closer()` is registered (line 253). A nil `closer` is never invoked because control returns before the defer statement. NO nil-deref.
- **Mitigation accepted:** Caller order is correct. The convention "always-call-closer" only applies when `err == nil`, at which point `closer` is guaranteed non-nil (either `func(){}` for `"-"` or `func(){ _ = f.Close() }` for the file path). REFUTED.

#### Attack 5 — `rootLabel` for `"-"` must be literally `"<stdin>"` (angle-bracketed)

- **Severity:** none (REFUTED)
- **Where:** `cmd/rak/root.go:256–258`
- **Counterexample attempt:** Inspected literal: `rootLabel = "<stdin>"` — angle-bracketed, lowercase. Matches PLAN.md § Notes Q3 RESOLVED line 510 and acceptance criterion line 275 verbatim.
- **Mitigation accepted:** REFUTED.

#### Attack 6 — `rootLabel` for a file path (e.g. `/home/user/files.txt`) renders verbatim

- **Severity:** nit (design-pinned)
- **Where:** `cmd/rak/root.go:255` (`rootLabel := flags.filesFrom`)
- **Counterexample attempt:** `rak --files-from /home/user/myproject/files.txt` would set `rootLabel = "/home/user/myproject/files.txt"`. `labelDirectories` (line 582) then calls `path.Clean(rootLabel)` and uses it as the prefix for any `Path == "."` directory and as the join base for sub-directories. The rendered TOON output for a per-directory bucket would read `dir: /home/user/myproject/files.txt` or `dir: /home/user/myproject/files.txt/sub`, which is semantically odd (the LIST file is not a directory). The PLAN.md § Notes "Absolute paths" line 484–488 explicitly addresses this: *"when a line in the list is an absolute path … the file is grouped under `filepath.Dir(absPath)` in the per-directory rollup. The `rootLabel` is unused for absolute paths."*
- **Mitigation accepted:** When all list entries are absolute paths, the per-directory bucket key is `filepath.Dir(absPath)`, NOT `"."`, so `labelDirectories` does NOT rewrite them with `rootLabel`. They render under their true parent directory. Only entries whose `dirKey` resolves to `"."` (files in CWD, fed as relative paths) would surface the awkward `dir: <files.txt>` rendering. This is design-pinned by Q3 line 510 (*"use `flags.filesFrom` (the filename) otherwise"*). NOT a counterexample.

#### Attack 7 — `--include` / `--exclude` / `--depth` silently pass through to filter in files-from branch

- **Severity:** none (REFUTED)
- **Where:** `cmd/rak/root.go:248–268` (no `listerOpts(flags)` call in this branch)
- **Counterexample attempt:** Traced the `--files-from` branch. `runDirectoryOpts` struct (line 319–327) declares only `rootLabel`, `binary`, `langs`, `sortKey`, `sortAsc`, `maxFiles`, `renderer`. No `Includes`, `Excludes`, `Depth`, `IncludeHidden`, `DisableGitignore` fields. The `FilesFromLister` constructed at line 254 receives only the `io.Reader` — no filter options. `walkAndCount` (called by `runDirectory`) does not consult any glob / depth state. `--include` / `--exclude` / `--depth` / `--hidden` / `--no-gitignore` cannot affect the files-from path.
- **Verification:** Compared to `len(args) == 1` branch (line 270–288) which DOES call `listerOpts(flags)`. Branches are correctly bifurcated.
- **Mitigation accepted:** REFUTED. Silent ignoring is the documented Q1/Q2 behavior (PLAN.md line 498–508).

#### Attack 8 — Guard A error message contains literal `"cannot combine"`

- **Severity:** none (REFUTED)
- **Where:** `cmd/rak/root.go:108`
- **Counterexample attempt:** Literal: `return fmt.Errorf("cannot combine --files-from with a positional path argument")` — substring `"cannot combine"` present. Acceptance criterion line 269 satisfied.
- **Mitigation accepted:** REFUTED.

#### Attack 9 — Guard B error message contains literal `"--no-gitignore"`

- **Severity:** none (REFUTED)
- **Where:** `cmd/rak/root.go:111`
- **Counterexample attempt:** Literal: `return fmt.Errorf("--no-gitignore is meaningless with --files-from: the caller controls which files are listed")` — substring `"--no-gitignore"` present at start. Acceptance criterion line 272 satisfied.
- **Mitigation accepted:** REFUTED.

#### Attack 10 — `--files-from foo.txt --include '*.go'` is NOT a hard error

- **Severity:** none (REFUTED)
- **Where:** `cmd/rak/root.go:103–113` (PersistentPreRunE)
- **Counterexample attempt:** Enumerated `PersistentPreRunE` guards: (1) sort key, (2) Guard A (positional + filesFrom), (3) Guard B (noGitignore + filesFrom). No guard against `--include` / `--exclude` / `--depth` / `--lang` / `--hidden` combined with `--files-from`. So `rak --files-from foo.txt --include '*.go'` reaches `runRoot` cleanly, enters the files-from branch, ignores `--include` silently. Matches spec Q1 (--include silent no-op) and the broader principle that only `--no-gitignore` is a hard error.
- **Mitigation accepted:** REFUTED. Behavior matches PLAN.md § Notes Q1 line 498–503.

#### Attack 11 — `os.Open(path)` error wrap mentions the path

- **Severity:** none (REFUTED)
- **Where:** `cmd/rak/root.go:310–312`
- **Counterexample attempt:** `os.Open` returns a `*fs.PathError` whose `Error()` is `"open <path>: <syscall err>"`. Line 312 wraps as `fmt.Errorf("--files-from: %w", err)`. Final error string: `"--files-from: open /nonexistent/path.txt: no such file or directory"`. Path IS surfaced via the wrapped `*PathError`. `errors.As(err, &pathErr)` would extract `pathErr.Path == "/nonexistent/path.txt"`. Acceptance criterion line 271 (*"non-nil error wrapping the `os.Open` failure"*) satisfied — `%w` is the wrap verb, not `%v` or `%s`.
- **Mitigation accepted:** REFUTED.

#### Attack 12 — Cobra flag ordering: `--files-from` appears in `rak --help`

- **Severity:** none (REFUTED)
- **Where:** `cmd/rak/root.go:200–205`
- **Counterexample attempt:** Ran `mage run -- --help`. Output includes in the FLAGS section:
  ```
  --files-from          Read newline-separated file paths from FILE (use - for stdin)
  ```
  Alphabetically ordered between `--exclude` and `-h --help`. The two new Example entries (`rg --files | rak --files-from -` and `git ls-files '*.go' | rak --files-from -`) also appear in the EXAMPLES section.
- **Mitigation accepted:** REFUTED. Acceptance criteria lines 264–266 satisfied.

### Additional observations (not counterexamples)

- **`--hidden` flag in files-from mode (consistency note)**: PLAN.md § D.2 step 8 lists `--include` / `--exclude` / `--depth` as silent no-ops; `--hidden` is not enumerated but is also silently ignored (same mechanism — `listerOpts` not called, `IncludeHidden` not in `runDirectoryOpts`). Consistent with the design principle that the caller's pipeline is the filter, but worth documenting alongside the other silent no-ops if a future doc pass tightens the README.
- **`--files-from` with `--toon` / `--human` / `--json`**: orthogonal — `MarkFlagsMutuallyExclusive("human","json","toon")` handles renderer selection independently. Tested mentally; no conflict.
- **`cobra.MaximumNArgs(1)` interaction**: with `rak --files-from foo a b c`, cobra rejects "max 1 arg" before `PersistentPreRunE` runs, so Guard A never sees the multi-arg case. Single positional + `--files-from` IS caught by Guard A as intended.
- **D.2 ships no NEW tests**: PLAN.md acceptance criterion line 261–262 explicitly defers test coverage to D.3 (*"existing tests must not regress; D.3 adds new tests"*). cmd/rak suite passed 1.508s, no regression. Acceptable per spec.

### Summary

- 12 attack vectors attempted across orchestrator-supplied angles.
- 0 CONFIRMED counterexamples; 11 REFUTED; 1 design-pinned (Attack 6 rootLabel for file path — explicitly per Q3 Notes).
- `mage build` passes; `mage test ./cmd/rak/...` green (1.508s). The pre-existing `internal/lang` failure (`undefined: LangJSX` etc.) is unrelated to D.2 and explicitly excluded per orchestrator directive.
- Unit D.2 is GO-FOR-CLOSE. Move to D.3.

### Hylla Feedback

Hylla queried for `files-from` returned only pre-D.2 baseline symbols (Counts, Walker.Walk, WalkOptions, Detect, etc.) — expected because D.2's changes are uncommitted/recent. Fell back to `Read` against `cmd/rak/root.go` directly. Not a Hylla miss — Hylla indexes committed state, and the file is uncommitted. No suggestion.
