# DROP_D — Builder Worklog

Append a `## Unit N.M — Round K` section per build attempt. See `main/drops/WORKFLOW.md` § "Phase 4 — Build (per unit)" for what each section should contain.

## Unit D.1 — Round 2 (combined Round 1 + Round 2 fix)

- **Builder:** go-builder-agent
- **Started:** 2026-05-16 (Round 1), 2026-05-17 (Round 2 fix)
- **Files touched:**
  - `internal/lister/filesfrom.go` (NEW in Round 1; line 91 bug fixed in Round 2)
  - `internal/lister/lister_test.go` (new TestFilesFromLister_* tests added in Round 1)
- **Mage targets run:**
  - Round 1: `mage build` (pass), `mage test` (4 failures — see bug below)
  - Round 2: `mage build` (pass), `mage test` (all pass, including all TestFilesFromLister_*)
- **Notes:**

  **Round 1 — what was built:**
  Implemented `FilesFromLister` struct in `internal/lister/filesfrom.go` satisfying the `FileLister` interface. Constructor `NewFilesFromLister(r io.Reader)` + `List(ctx) iter.Seq2[*fileset.File, error]` loop with: context check per iteration, `bufio.Scanner` over the reader, trim+skip empty lines, `filepath.Clean`, path absolutisation, `os.Stat` regular-file check, `fileset.NewFile(os.DirFS(dir), base, base)` yield, and `scanner.Err()` check after loop. Compile-time assertion `var _ FileLister = (*FilesFromLister)(nil)`. Six test functions added to `internal/lister/lister_test.go`.

  **Round 1 — bug:**
  Line 91 used `filepath.Abs(filepath.Join(cwd, cleaned))`. `filepath.Join` does not treat an absolute second argument specially on Go/Darwin — it concatenates the second argument's path components onto cwd. So `filepath.Join("/some/cwd", "/var/folders/79/...")` produces `/some/cwd/var/folders/79/...` instead of `/var/folders/79/...`. All `t.TempDir()`-based tests pass absolute paths, so their paths were corrupted. Four tests failed: `TestFilesFromLister_HashPrefixedFileWorks`, `TestFilesFromLister_MissingFile`, `TestFilesFromLister_MixedPaths`, `TestFilesFromLister_SkipsEmptyLines`.

  **Round 2 — fix:**
  Replaced the corrupt `filepath.Abs(filepath.Join(cwd, cleaned))` call with an explicit `filepath.IsAbs` check:
  ```go
  absPath := cleaned
  if !filepath.IsAbs(absPath) {
      absPath = filepath.Join(cwd, cleaned)
  }
  ```
  Also removed the now-dead `if err != nil { yield(...) }` block (no error from `filepath.IsAbs`/`filepath.Join`). The downstream `os.Stat(absPath)` already handles all path-resolution failures. Relative paths continue to use `filepath.Join(cwd, cleaned)` — the CWD-at-List-entry semantics are preserved.

  All `TestFilesFromLister_*` tests pass green. No regressions in any other package.

## Hylla Feedback

N/A — task touched only the bug fix in `filesfrom.go` (already uncommitted). Hylla is Go-only and indexes committed state; the file was new and uncommitted, so no Hylla queries were relevant.
