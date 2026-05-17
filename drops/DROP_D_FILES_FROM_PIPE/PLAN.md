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
- **Path interpretation**: paths are interpreted relative to the current working directory. Empty lines are skipped. Lines starting with `#` are treated as comments and skipped (standard Unix convention; matches `git rev-list --stdin` precedent).
- **Path normalization**: each path goes through `path.Clean` + the same regular-file check from v0.1.4's `SingleFileLister`.
- **Error semantics**: missing file → friendly error (`not a regular file or directory: <path>`); per-line errors aggregate via `errors.Join` so one bad path doesn't crash the whole stream.

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

Design decisions pending dev confirmation are in `## Notes` below and must be
resolved in Phase 3 before build starts. Builders should not unilaterally
decide Q1–Q7.

---

### Unit D.1 — FilesFromLister impl

**State:** todo
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
2. `bufio.Scanner` over `r`. Each line is: trim whitespace, skip if empty,
   skip if starts with `#` (comment per Unix convention + git precedent).
3. `filepath.Clean` the line (OS-native clean, not `path.Clean`, since we will
   call `filepath.Abs` next which requires OS-native separators).
4. `filepath.Abs` relative to CWD. CWD must be resolved inside `List()` (not
   the constructor) via `os.Getwd()` — this ensures the test CWD is honoured
   at list time.
5. `os.Stat(absPath)` — if the path does not exist or is not a regular file,
   yield `(nil, fmt.Errorf("lister: files-from: %q is not a regular file: %w",
   line, err))` and continue (per-line error aggregation via the iterator
   contract — the walk continues past bad lines).
6. `dir, base := filepath.Dir(absPath), filepath.Base(absPath)`.
   `yield(fileset.NewFile(os.DirFS(dir), base, base), nil)`.
7. Check `yield` return value — if `false`, stop (F14 carry-over).

Export the type so `lister_test.go` can type-assert on it (same convention as
`GitLister`, `WalkLister`, `SingleFileLister`).

Compile-time assertion: `var _ FileLister = (*FilesFromLister)(nil)`.

`FilesFromLister` does NOT close `r`. The caller owns the reader.

**New tests in `internal/lister/lister_test.go`** (not a new test file — append
to the existing file per package convention):

- `TestFilesFromLister_EmptyReader` — `strings.NewReader("")` yields zero
  files, zero errors.
- `TestFilesFromLister_AllComments` — reader with only `#` lines yields zero
  files, zero errors.
- `TestFilesFromLister_SkipsEmptyLines` — interleaved empty lines are skipped.
- `TestFilesFromLister_MixedWithComments` — mix of valid paths, empty lines,
  `#` comment lines: only the valid paths produce files, in order.
- `TestFilesFromLister_MissingFile` — a path that does not exist on disk yields
  a `(nil, err)` pair; the walk continues and subsequent valid paths still
  yield files.
- `TestFilesFromLister_ContextCancel` — cancel the context after the first
  yield; verify iteration stops at the cancellation error.
- `TestDetect_*` for `FilesFromLister` via `Detect` is NOT needed — `Detect`
  is not changed; `FilesFromLister` is constructed directly by the caller.

For `TestFilesFromLister_MixedWithComments` and `TestFilesFromLister_MissingFile`:
use `t.TempDir()` + `os.WriteFile` to create real on-disk files the lister can
stat. Do NOT use `fstest.MapFS` — `FilesFromLister` calls `os.Stat` and
`os.DirFS`, which operate on the real filesystem.

**Acceptance criteria:**

- `mage test ./internal/lister/...` passes with `-race`.
- `mage build` passes (package compiles cleanly).
- All five scenarios above are covered by a named test.
- Context-cancellation test verifies iteration terminates without panic.
- Per-line error for a missing file does NOT abort the iterator — the next
  valid path is still yielded.
- CWD resolution happens in `List()`, not the constructor (important for test
  isolation).

---

### Unit D.2 — CLI flag wiring + runRoot third branch

**State:** todo
**Paths:** `cmd/rak/root.go`
**Packages:** `cmd/rak` (package `main`)
**Blocked by:** D.1

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

3. Mutual-exclusion guard in `PersistentPreRunE` (before `RunE`, after the
   sort-key check):

   ```go
   if flags.filesFrom != "" && len(args) > 0 {
       return fmt.Errorf("cannot combine --files-from with a positional path argument")
   }
   ```

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
       return runDirectory(ctx, c.OutOrStdout(), source, runDirectoryOpts{
           rootLabel: flags.filesFrom,
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

   `rootLabel` passed to `runDirectory` is `flags.filesFrom` (e.g. `"-"` or
   the filename). This is subject to dev confirmation of Q3.

6. `--no-gitignore` + `--files-from`: since `Detect` is never called in this
   branch, `ErrNoGitignoreInRepo` is never raised. `--no-gitignore` silently
   does nothing. The flag usage string for `--no-gitignore` does not need to
   change — it already says "during the walk" which implies walk mode.

7. `--depth` + `--files-from`: `listerOpts(flags)` is not called in the
   `--files-from` branch, so `--depth` is silently ignored. No warning. Subject
   to dev confirmation of Q2.

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

Append two new test functions to `cmd/rak/integration_test.go`. Both use the
existing fixture tree at `testdata/tree/` which has `a.txt` (12 bytes) and
`sub/nested.txt` (8 bytes).

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

**Test 2 — `TestRootCmd_Integration_FilesFrom_WithComments`:**

Same setup but the `strings.NewReader` includes comment lines and blank lines:

```
# comment line
testdata/tree/a.txt

# another comment
testdata/tree/sub/nested.txt
```

Assert same totals — comments and empty lines are skipped.

**Test 3 (optional, add if straightforward) — `TestRootCmd_Integration_FilesFrom_PositionalArgConflict`:**

```go
cmd.SetArgs([]string{"--files-from", "-", "."})
err := cmd.Execute()
// must be non-nil and contain "cannot combine"
```

**Acceptance criteria:**

- `mage test ./cmd/rak/...` passes with `-race`.
- Both primary tests assert on `parsed.Total` fields (Bytes, Lines, Words,
  Chars) matching the tree fixture constants.
- The comment-skipping test passes with the same totals as the plain test.
- The conflict test (if added) verifies the error message contains
  `"cannot combine"`.
- Existing integration tests in `integration_test.go` continue to pass
  (no regressions).

Note: the `treeResult`, `dirResult`, and `treeExpected*` constants defined
earlier in `integration_test.go` are reused. Builders should not redefine
them — they are already in scope (same package, same file).

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

**Cross-stream coordination**: Streams B, C, D all add new flags to `cmd/rak/root.go`. Unit D.2 is the cmd/rak flag-wiring unit; the orchestrator must serialize it against B and C at build time. Internal-package work (D.1, `internal/lister/*`) is parallel-safe with B and C.

**Factory routing decision**: `Detect` is NOT extended. `runRoot` constructs
`lister.NewFilesFromLister(r)` directly when `flags.filesFrom != ""`, bypassing
`Detect` entirely. Rationale: `Detect` answers "which lister for this root path?"
— `--files-from` is an orthogonal dispatch axis and should not pollute that
signature. The approach is consistent with `SingleFileLister` being a
`Detect`-returned value rather than a separate factory.

---

**Design questions requiring dev confirmation before build (Phase 3):**

**Q1 — Filter interaction** (`--lang`, `--include`, `--exclude`):
When `--files-from` is set, do these filters still apply?
- Recommendation: YES. `--files-from` only sources the candidate list;
  `walkAndCount` already applies lang + binary filters post-listing.
  `include`/`exclude` glob filters live in `WalkOptions` which is not used in
  the `--files-from` branch — so `--include`/`--exclude` would NOT apply
  unless `FilesFromLister` is given an explicit filtering step.
  Dev decision: should `FilesFromLister` respect `--include`/`--exclude`?
  Simplest answer: skip include/exclude in the `--files-from` branch (the
  caller already filtered). `--lang` still applies (it's in `walkAndCount`).

**Q2 — `--depth` interaction**:
`--depth` is a walk-traversal limit. With `--files-from` it has no meaning.
- Recommendation: silent no-op. `listerOpts` is not called in the `--files-from`
  branch, so `--depth` is simply ignored.
- Alternative: warn-and-no-op. Dev decides.

**Q3 — rootLabel for rendered output**:
When `--files-from` is set, what directory label appears in the rendered tree?
All files collapse into one directory bucket (via `dirKey` returning `"."`).
`labelDirectories` rewrites `"."` to `rootLabel`.
- Option A: `flags.filesFrom` (e.g. `"-"` or the filename). Literal, clear.
- Option B: `"."` (the io/fs root convention). Minimal.
- Option C: `"stdin"` when value is `"-"`, filename otherwise.
- Recommendation: Option A (`flags.filesFrom`). Dev decides.

**Q4 — Comment lines (`#`)** (already in Scope, surfaced for explicit signoff):
Lines starting with `#` are skipped. This matches `git rev-list --stdin`
precedent. Confirmed in Scope. Any deviation from this? Dev signoff captured here.

**Q5 — Positional arg + `--files-from` conflict**:
If both `--files-from <X>` and a positional path argument are supplied, the
behavior should be a hard error.
- Recommendation: `"cannot combine --files-from with a positional path argument"`.
- Alternative: `--files-from` wins silently. Recommendation is hard error.
- Dev confirms.

**Q6 — Empty stdin (`echo -n | rak --files-from -`)**:
Zero files counted. TOON (and other) renderers receive a `Summary` with empty
`Dirs` and zero `Total`. Verify renderer handles this gracefully (no panic, no
crash). Pre-existing behavior — but worth a manual smoke test. Builder should
add a test for zero-file case in D.3.

**Q7 — Error aggregation**:
When a path in the list is missing or not a regular file, the error is yielded
as a `(nil, err)` pair and the walk continues to the next path.
- Recommendation: `errors.Join`-style aggregation (same as `walkAndCount`
  handles per-entry errors). The renderer's error summary will display them.
- Alternative: fail-fast on first bad path. Recommendation is aggregate.
- Dev confirms.
