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
