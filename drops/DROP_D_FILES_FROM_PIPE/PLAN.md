# DROP_D — FILES_FROM_PIPE

**State:** planning
**Tier:** A
**Blocked by:** —
**Paths (expected):** NEW internal/lister/filesfrom.go (or extend internal/lister/), internal/lister/lister.go (factory routing), internal/lister/lister_test.go, cmd/rak/root.go, main/docs/tapes/pipe.tape (NEW), main/docs/pipe.gif (NEW), README.md
**Packages (expected):** internal/lister, cmd/rak
**PLAN.md ref:** — (top-level PLAN.md removed at v0.1.0 ship; see memory `session_handoff_2026_05_16_v020_planning.md`)
**Workflow:** main/drops/WORKFLOW.md
**Started:** 2026-05-16
**Closed:** —

## Scope

Add `--files-from <FILE>` for pipe composition — the missing link between rak and the wider Unix toolchain.

- **New flag**: `--files-from <FILE>`. Use `-` (literal hyphen) to read from stdin. Reads newline-separated paths; each path is counted as a single file (re-uses the existing `SingleFileLister` machinery from v0.1.4).
- **Stdin sentinel**: bare positional stdin (`cat README.md | rak`) is unchanged — still single-stream wc-parity counting. `--files-from -` is the explicit opt-in to "read a list of paths from stdin."
- **Path interpretation**: paths are interpreted relative to the current working directory. Empty lines are skipped. Each non-empty line is a path. No comment syntax — users who want comments can pre-filter with `grep -v '^#' | rak --files-from -`.
- **Path normalization**: each path goes through `filepath.Clean` + the same regular-file check from v0.1.4's `SingleFileLister`.
- **Error semantics**: missing file → friendly error (`not a regular file: <path>`); per-line errors are yielded as `(nil, err)` pairs and iteration continues so one bad path does not crash the whole stream.

**Unblocks the canonical Unix-composition workflows:**
- `rg --files | rak --files-from -`
- `git ls-files '*.go' | rak --files-from -`
- `find . -name '*.go' | rak --files-from -`
- `gh pr diff 42 --name-only | rak --files-from -`

**Out of scope (deferred per dev 2026-05-16):**
- NUL-delimited variant `--files0-from <FILE>` — defer to v0.2.1 or v0.3. Hardens against filenames with newlines/spaces.

**Feature trio (mandatory per memory `feedback_rak_docs_and_gifs_before_pr.md`):**

1. VHS demo: `main/docs/tapes/pipe.tape` + `main/docs/pipe.gif`. Show `git ls-files '*.go' | rak --files-from -` against a fixture. Embed in README near a new "Piping" narrative section.
2. README examples: at minimum the four invocations above in "Common invocations" + a "Piping" narrative section.
3. Cobra `Example:` entries in `cmd/rak/root.go` for at least two of the four invocations (typically `rg --files | rak --files-from -` and `git ls-files '*.go' | rak --files-from -`).

## Planner

Four units, linear chain D.1 → D.2 → D.3 → D.4.

Design decisions are resolved below in `## Notes`. No open dev-signoff items remain before build starts.

---

### Unit D.1 — FilesFromLister impl

**State:** done
**Paths:** `internal/lister/filesfrom.go` (NEW), `internal/lister/lister_test.go` (additions)
**Packages:** `internal/lister`
**Blocked by:** —

**What to build:**

New file `internal/lister/filesfrom.go`. Implement a `FilesFromLister` struct
that satisfies `FileLister` by reading newline-separated paths from a
caller-supplied `io.Reader`.

Constructor:

```go
// NewFilesFromLister constructs a FilesFromLister that reads paths from r.
// The caller owns r and is responsible for closing it after listing.
func NewFilesFromLister(r io.Reader) *FilesFromLister
```

`List(ctx context.Context) iter.Seq2[*fileset.File, error]` loop (in order):

1. Check `ctx.Err()` at the top of each scan iteration — terminate iteration
   with `yield(nil, ctx.Err())` if cancelled.
2. `bufio.Scanner` over `r`. Each line is: trim whitespace, skip if empty.
   No comment syntax — every non-empty line is a path, including lines that
   start with `#` (e.g. `#draft.md`, `#merge.bak#`).
3. `filepath.Clean` the line (OS-native clean, not `path.Clean`, since we will
   call `filepath.Abs` next which requires OS-native separators).
4. `filepath.Abs` relative to CWD. CWD must be resolved inside `List()` (not
   the constructor) via `os.Getwd()` — this ensures the test CWD is honoured
   at list time.
5. `os.Stat(absPath)` — if the path does not exist or is not a regular file,
   yield `(nil, fmt.Errorf("lister: files-from: %q is not a regular file: %w",
   line, err))` and continue (per-line error — the walk continues past bad
   lines; matches the FileLister iterator contract used by GitLister and
   WalkLister).
6. `dir, base := filepath.Dir(absPath), filepath.Base(absPath)`.
   `yield(fileset.NewFile(os.DirFS(dir), base, base), nil)`.
7. Check `yield` return value — if `false`, stop (F14 carry-over).

After the scan loop exits (step 8): call `scanner.Err()`. If non-nil, yield
`(nil, fmt.Errorf("lister: files-from: scanner: %w", err))` before the
iterator function returns.

**Scanner buffer:** use the default `bufio.NewScanner(r)` buffer (64KiB per line). No `scanner.Buffer` call needed — real-world paths are nowhere near 64KiB (most filesystems cap path components at 255 chars, total path at ~4KiB). If a user reports an issue with extremely long paths, bump in v0.2.1.

Export the type so `lister_test.go` can type-assert on it (same convention as
`GitLister`, `WalkLister`, `SingleFileLister`).

Compile-time assertion: `var _ FileLister = (*FilesFromLister)(nil)`.

`FilesFromLister` does NOT close `r`. The caller owns the reader.

**New tests in `internal/lister/lister_test.go`** (not a new test file — append
to the existing file per package convention):

- `TestFilesFromLister_EmptyReader` — `strings.NewReader("")` yields zero
  files, zero errors.
- `TestFilesFromLister_HashPrefixedFileWorks` — a file on disk actually named
  `#draft.md` (created via `t.TempDir()` + `os.WriteFile`) is yielded
  successfully. Proves that `#`-prefixed paths are not treated as comments.
- `TestFilesFromLister_SkipsEmptyLines` — interleaved empty lines are skipped;
  valid paths around them are still yielded.
- `TestFilesFromLister_MixedPaths` — mix of valid paths and empty lines: only
  the valid paths produce files, in order.
- `TestFilesFromLister_MissingFile` — a path that does not exist on disk yields
  a `(nil, err)` pair; the walk continues and subsequent valid paths still
  yield files.
- `TestFilesFromLister_ContextCancel` — cancel the context after the first
  yield; verify iteration stops at the cancellation error.
- `TestDetect_*` for `FilesFromLister` via `Detect` is NOT needed — `Detect`
  is not changed; `FilesFromLister` is constructed directly by the caller.

For `TestFilesFromLister_HashPrefixedFileWorks`, `TestFilesFromLister_MixedPaths`,
and `TestFilesFromLister_MissingFile`: use `t.TempDir()` + `os.WriteFile` to
create real on-disk files the lister can stat. Do NOT use `fstest.MapFS` —
`FilesFromLister` calls `os.Stat` and `os.DirFS`, which operate on the real
filesystem.

**Acceptance criteria:**

- `mage test ./internal/lister/...` passes with `-race`.
- `mage build` passes (package compiles cleanly).
- All six scenarios above are covered by a named test.
- Context-cancellation test verifies iteration terminates without panic.
- Per-line error for a missing file does NOT abort the iterator — the next
  valid path is still yielded.
- CWD resolution happens in `List()`, not the constructor (important for test
  isolation).
- `scanner.Err()` is checked after the scan loop; a mid-stream scanner error
  yields `(nil, err)` before the iterator terminates.
- The `#draft.md` test proves hash-prefixed file names are passed through
  without filtering.
- Per-line cap is the default `bufio.Scanner` 64KiB (no `scanner.Buffer` call needed). Documented as v0.2.0 limit.

---

### Unit D.2 — CLI flag wiring + runRoot third branch

**State:** done
**Paths:** `cmd/rak/root.go`
**Packages:** `cmd/rak` (package `main`)
**Blocked by:** D.1, and serialize after C's `--workers` / `--follow` flag-registration block (D's `--files-from` flag is appended last to the flag block)

**What to build:**

1. Add `filesFrom string` field to `rootFlags` struct.

2. Register the flag in `newRootCmd`:

   ```go
   cmd.Flags().StringVar(
       &flags.filesFrom,
       "files-from",
       "",
       "read newline-separated file paths from FILE (use - for stdin)",
   )
   ```

3. Mutual-exclusion guards in `PersistentPreRunE` (before `RunE`, after the
   sort-key check). Two guards:

   **Important — signature change required:** the existing `PersistentPreRunE`
   has both params blanked out (`_ *cobra.Command, _ []string`). To check
   `len(args)` in the positional-conflict guard, rename the second param:
   `func(_ *cobra.Command, args []string) error`.

   Guard A — positional argument conflict:
   ```go
   if flags.filesFrom != "" && len(args) > 0 {
       return fmt.Errorf("cannot combine --files-from with a positional path argument")
   }
   ```

   Guard B — `--no-gitignore` conflict:
   ```go
   if flags.filesFrom != "" && flags.noGitignore {
       return fmt.Errorf("--no-gitignore is meaningless with --files-from: the caller controls which files are listed")
   }
   ```

   Both guards run in `PersistentPreRunE` (hard error, not a warning). This
   matches the `ErrNoGitignoreInRepo` pattern from v0.1.3 — conflicting flags
   return an error immediately rather than silently no-oping.

4. Add two `Example:` entries to the cobra command's `Example` block:

   ```
     # Pipe a file list from ripgrep
     rg --files | rak --files-from -

     # Count only tracked Go files
     git ls-files '*.go' | rak --files-from -
   ```

5. Third branch in `runRoot` (insert BEFORE the `len(args) == 1` branch so
   `--files-from` takes precedence):

   ```go
   if flags.filesFrom != "" {
       r, closer, err := openFilesFrom(flags.filesFrom, c.InOrStdin())
       if err != nil {
           return err
       }
       defer closer()
       source := lister.NewFilesFromLister(r)
       rootLabel := flags.filesFrom
       if flags.filesFrom == "-" {
           rootLabel = "<stdin>"
       }
       return runDirectory(ctx, c.OutOrStdout(), source, runDirectoryOpts{
           rootLabel: rootLabel,
           binary:    flags.binary,
           langs:     flags.langs,
           sortKey:   flags.sort,
           sortAsc:   flags.sortAsc,
           maxFiles:  flags.maxFiles,
           renderer:  renderer,
       })
   }
   ```

   Helper (new private function in `root.go`):

   ```go
   // openFilesFrom resolves the --files-from value to an io.Reader and a
   // no-op-or-close func. When value is "-", it returns c.InOrStdin() and a
   // no-op closer (stdin is not owned by this call). Otherwise it opens the
   // named file and returns the file plus its Close method as the closer.
   func openFilesFrom(value string, stdin io.Reader) (io.Reader, func(), error)
   ```

6. `--no-gitignore` + `--files-from` is a hard error (Guard B in step 3 above).
   `Detect` is never called in the `--files-from` branch, so the `--no-gitignore`
   flag has no effect on filtering. Returning an error surfaces this immediately
   rather than silently misleading the user.

7. `--depth` + `--files-from`: `listerOpts(flags)` is not called in the
   `--files-from` branch, so `--depth` is silently ignored. No warning (resolved;
   see Notes § Q2).

8. `--include` / `--exclude` + `--files-from`: these glob filters are not applied
   in the `--files-from` branch. The caller's source (ripgrep, git, find) is the
   filter — adding a second filter layer would be unexpected. `--lang` still
   applies post-listing via `walkAndCount` (resolved; see Notes § Q1).

9. `--max-files` applies in `--files-from` mode identically to walk mode:
   `maxFiles: flags.maxFiles` is passed through `runDirectoryOpts` to
   `walkAndCount`, which enforces `ErrMaxFilesExceeded` mid-stream. No extra
   wiring needed.

**Acceptance criteria:**

- `mage build` passes.
- `mage test ./cmd/rak/...` passes (existing tests must not regress; D.3 adds
  new tests).
- `rak --help` shows `--files-from` flag with the correct usage string.
- `rak --help` shows the two new `Example:` entries.
- `rak --files-from - .` (both flag and positional arg) returns a non-nil
  error containing `"cannot combine"`.
- `rak --files-from /nonexistent/path.txt` returns a non-nil error wrapping
  the `os.Open` failure.
- `rak --files-from - --no-gitignore` returns a non-nil error containing
  `"--no-gitignore"` (Guard B). `TestFlags_FilesFromNoGitignoreHardErrors`
  verifies this in D.3 (or D.2's own `root_test.go` additions — either is
  acceptable).
- Rendered TOON output for `rak --files-from -` shows `path: <stdin>` not
  `path: -`.
- The `runRoot` branch order is: `--files-from` branch first, then
  `len(args)==1` branch, then bare-stdin fallback. This ensures `--files-from`
  takes priority.

---

### Unit D.3 — End-to-end integration tests

**State:** todo
**Paths:** `cmd/rak/integration_test.go`
**Packages:** `cmd/rak` (package `main`)
**Blocked by:** D.2

**What to build:**

Append new test functions to `cmd/rak/integration_test.go`. Use the
existing fixture tree at `testdata/tree/` which has `a.txt` (12 bytes) and
`sub/nested.txt` (8 bytes), with totals `treeExpectedTotalBytes=20`,
`treeExpectedTotalLines=2`, `treeExpectedTotalWords=4`,
`treeExpectedTotalChars=20`.

The `cmd/rak` package test runs from the `cmd/rak/` directory as CWD, so
relative paths like `testdata/tree/a.txt` resolve correctly when
`FilesFromLister` calls `filepath.Abs`.

**Test 1 — `TestRootCmd_Integration_FilesFrom_StdinList`:**

```go
func TestRootCmd_Integration_FilesFrom_StdinList(t *testing.T) {
    t.Parallel()

    list := "testdata/tree/a.txt\ntestdata/tree/sub/nested.txt\n"
    var out bytes.Buffer
    cmd := newRootCmd()
    cmd.SetIn(strings.NewReader(list))
    cmd.SetOut(&out)
    cmd.SetErr(&out)
    cmd.SetArgs([]string{"--json", "--files-from", "-"})

    if err := cmd.Execute(); err != nil {
        t.Fatalf("cmd.Execute: %v", err)
    }

    // Parse and assert totals match the tree fixture (a.txt=12 + nested.txt=8)
    // using the existing treeResult type and treeExpected* constants.
}
```

Assert: `parsed.Total.Bytes == treeExpectedTotalBytes` (20), Lines == 2,
Words == 4, Chars == 20. The JSON envelope for `runDirectory` output is the
tree-result shape (with `directories` array), NOT the flat-counts shape from
`counting.Counts`. Assert on `parsed.Total`.

**Test 2 — `TestRootCmd_Integration_FilesFrom_EmptyStdin`:**

`echo -n | rak --files-from -` equivalent: `strings.NewReader("")` as stdin,
`--files-from -`. Produces well-formed empty output (zero directories, zero
total) without panic or error. Assert `err == nil` and `parsed.Total.Bytes == 0`.

**Test 3 — `TestRootCmd_Integration_FilesFrom_SkipsEmptyLines`:**

Feed a list with blank lines interspersed:

```
testdata/tree/a.txt

testdata/tree/sub/nested.txt

```

Assert same totals as Test 1 — empty lines are skipped; output is identical.

**Test 4 — `TestRootCmd_Integration_FilesFrom_HashFileWorks`:**

Create a real temp file named `#draft.md` in `t.TempDir()`. Feed its path
through `--files-from -`. Assert the file is counted (non-zero Bytes in total),
proving that `#`-prefixed filenames are not treated as comments.

**Test 5 — `TestRootCmd_Integration_FilesFrom_PositionalArgConflict`:**

```go
cmd.SetArgs([]string{"--files-from", "-", "."})
err := cmd.Execute()
// must be non-nil and contain "cannot combine"
```

**Test 6 — `TestFlags_FilesFromNoGitignoreHardErrors`:**

```go
cmd.SetArgs([]string{"--files-from", "-", "--no-gitignore"})
err := cmd.Execute()
// must be non-nil and message must contain "--no-gitignore"
```

**Test 7 — `TestFilesFrom_MaxFiles`:**

Feed a list with three files from the fixture tree (or temp files). Set
`--max-files 1`. Assert `cmd.Execute()` returns a non-nil error wrapping
`ErrMaxFilesExceeded`. Verify via `errors.Is(err, lister.ErrMaxFilesExceeded)`
or `errors.Is(err, root.ErrMaxFilesExceeded)` — whichever is accessible from
the test's package scope. (Both are the same sentinel; `ErrMaxFilesExceeded` is
declared in `cmd/rak/root.go` as package-level `var`, accessible within
`package main` tests.)

**Acceptance criteria:**

- `mage test ./cmd/rak/...` passes with `-race`.
- Test 1 asserts `parsed.Total` fields (Bytes, Lines, Words, Chars) matching
  the tree fixture constants.
- Test 2 verifies empty stdin produces no panic, no error, and zero totals.
- Test 3 passes with the same totals as Test 1 (empty lines skipped).
- Test 4 proves `#`-prefixed filenames are counted normally (not dropped).
- Test 5 verifies the error message contains `"cannot combine"`.
- Test 6 verifies the error message references `"--no-gitignore"`.
- Test 7 verifies `ErrMaxFilesExceeded` fires when file count exceeds
  `--max-files` in `--files-from` mode.
- Existing integration tests in `integration_test.go` continue to pass
  (no regressions).

Note: `treeResult`, `dirResult`, and `treeExpected*` constants are already
defined in `root_test.go` / `integration_test.go` (same package `main`).
Builders must not redefine them — they are in scope.

---

### Unit D.4 — Feature trio docs

**State:** todo
**Paths:** `main/docs/tapes/pipe.tape` (NEW), `main/docs/pipe.gif` (NEW), `README.md`
**Packages:** — (no Go packages)
**Blocked by:** D.3

**What to build:**

1. **VHS tape** (`main/docs/tapes/pipe.tape`):
   - Demo: `git ls-files '*.go' | rak --files-from -` run against the rak
     source tree.
   - Follow the style of existing tapes in `main/docs/tapes/` if any exist;
     otherwise use standard VHS tape format.
   - Builder writes the tape; dev runs `vhs main/docs/tapes/pipe.tape` and
     commits the resulting gif to `main/docs/pipe.gif`. The gif itself is
     committed — it is not generated in CI.

2. **README.md** — add a "Piping" section:
   - Narrative: one or two sentences explaining Unix composition.
   - Code block: at minimum the four canonical invocations from the Scope:

     ```sh
     # Pipe from ripgrep
     rg --files | rak --files-from -

     # Count only tracked Go files
     git ls-files '*.go' | rak --files-from -

     # Find by name
     find . -name '*.go' | rak --files-from -

     # Count files changed in a PR
     gh pr diff 42 --name-only | rak --files-from -
     ```

   - Add at least two of these to the "Common invocations" section (if one
     exists) or create a piping sub-section.

3. **Embed the gif** in README near the Piping section:

   ```md
   ![rak --files-from demo](docs/pipe.gif)
   ```

**Acceptance criteria:**

- `main/docs/tapes/pipe.tape` exists and is syntactically valid VHS script.
- `README.md` contains a "Piping" section (or equivalent heading) with at
  least the four invocations.
- `README.md` embeds the gif reference near the Piping section.
- The tape script references `--files-from -` with correct flag spelling.
- `mage build` and `mage test` are unaffected (no Go changes).
- Dev signoff: dev runs VHS, confirms the gif renders correctly, and commits
  `main/docs/pipe.gif`.

## Notes

**Cross-stream coordination**: Streams B, C, D all add new flags to
`cmd/rak/root.go`. Unit D.2 is the cmd/rak flag-wiring unit; the orchestrator
must serialize D.2 against B and C at build time: D.2 is `Blocked by: D.1, and
after C's flag-registration block`. The `--files-from` flag registration is
appended last to the flag block. Internal-package work (D.1, `internal/lister/*`)
is parallel-safe with B and C.

**Factory routing decision**: `Detect` is NOT extended. `runRoot` constructs
`lister.NewFilesFromLister(r)` directly when `flags.filesFrom != ""`, bypassing
`Detect` entirely. Rationale: `Detect` answers "which lister for this root
path?" — `--files-from` is an orthogonal dispatch axis and should not pollute
that signature. This is consistent with `SingleFileLister` being returned by
`Detect` rather than having its own separate factory.

**Symlink behavior**: `os.Stat` (used in `FilesFromLister.List`) follows
symlinks. The symlink target is counted, not the symlink entry itself. Matches
v0.1.4 `SingleFileLister` behavior. Consistency intentional.

**Interactive stdin (TTY)**: when `--files-from -` is used interactively
without piping, stdin blocks until EOF (Ctrl-D). This matches Unix convention
(`cat`, `wc`). No special handling.

**Duplicate paths**: feeding the same path twice yields two count entries for
the same file. No deduplication. Matches `wc` behavior.

**Absolute paths**: when a line in the list is an absolute path (e.g.
`/home/user/foo.go`), `filepath.Abs` returns it unchanged. The file is grouped
under `filepath.Dir(absPath)` in the per-directory rollup. The `rootLabel` is
unused for absolute paths — they live in the per-directory bucket keyed by
their real parent dir.

**Paths outside CWD via `../`**: resolved via `filepath.Abs` and grouped under
their real parent directory. The rendered tree may show parent directories
above the CWD.

---

**Resolved design decisions:**

**Q1 — Filter interaction (`--lang`, `--include`, `--exclude`)** — RESOLVED:
`--lang` applies (it is a post-listing filter in `walkAndCount`, naturally
active in the `--files-from` branch). `--include` / `--exclude` do NOT apply:
`listerOpts` is not called in the `--files-from` branch, and the caller has
already controlled which files are listed (that is the point of `--files-from`
— the source tool is the filter). This is documented in D.2 step 8.

**Q2 — `--depth` interaction** — RESOLVED: silent no-op. `listerOpts` is not
called in the `--files-from` branch, so `--depth` is simply ignored. Walk
traversal depth has no meaning when the caller supplies an explicit file list.

**Q3 — `rootLabel` for rendered output** — RESOLVED: use `"<stdin>"` when
`flags.filesFrom == "-"`, and `flags.filesFrom` (the filename) otherwise.
This is implemented inline in the `runRoot` branch (D.2 step 5). `path.Clean`
does not mangle `<stdin>` — angle brackets are not path separators.

**Q5 — Positional arg + `--files-from` conflict** — RESOLVED: hard error
`"cannot combine --files-from with a positional path argument"` in
`PersistentPreRunE`. `--no-gitignore + --files-from` is also a hard error
(`"--no-gitignore is meaningless with --files-from"`). Both guards live in
`PersistentPreRunE`.

**Q6 — Empty stdin** — RESOLVED: zero files counted. TOON and other renderers
receive a `Summary` with empty `Dirs` and zero `Total`. Pre-existing renderer
behavior handles this gracefully. `TestRootCmd_Integration_FilesFrom_EmptyStdin`
in D.3 verifies no panic.

**Q7 — Error aggregation** — RESOLVED: per-line yield `(nil, err)` — consistent
with the `FileLister` iterator contract used by `GitLister` and `WalkLister`.
The walk continues past bad lines; `errors.Join` is not used.
