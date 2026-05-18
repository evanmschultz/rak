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

## Unit D.2 — Round 1

- **Builder:** go-builder-agent
- **Started:** 2026-05-17
- **Files touched:**
  - `cmd/rak/root.go` (flag field, flag registration, PersistentPreRunE guards, Example entries, openFilesFrom helper, runRoot --files-from branch)
- **Mage targets run:**
  - `mage format` — pass (no output)
  - `mage build` — pass (no output)
  - `mage test` — pass (all packages green, including `cmd/rak` and `internal/lister`)
- **Notes:**

  Added `filesFrom string` field to `rootFlags`. Registered `--files-from` flag (appended after `--max-files` per cross-stream serialization rule). Added two `Example:` entries (`rg --files | rak --files-from -` and `git ls-files '*.go' | rak --files-from -`).

  Renamed `PersistentPreRunE` second param from `_` to `args` to enable the positional-conflict check. Added Guard A (positional + --files-from conflict) and Guard B (--no-gitignore + --files-from conflict), both returning formatted errors.

  Added `openFilesFrom(value string, stdin io.Reader) (io.Reader, func(), error)` private helper: returns stdin + no-op closer when value is `"-"`, opens the named file and returns it plus its Close func otherwise. Error wrapped with `--files-from: %w`.

  Inserted `--files-from` branch at the top of `runRoot` (before the `len(args) == 1` branch) so it takes priority. Uses `lister.NewFilesFromLister(r)`, sets `rootLabel = "<stdin>"` when value is `"-"`, passes `maxFiles` through `runDirectoryOpts`. `--depth`, `--include`, `--exclude` are not applied in this branch (no `listerOpts` call) per design decisions Q1/Q2 in PLAN.md Notes.

  Added `"os"` to stdlib import group (needed for `os.Open` in `openFilesFrom`).

  All 9 PLAN.md "What to build" steps implemented. No scope expansion. No files touched outside the unit's declared path.

## Hylla Feedback

None — Hylla answered everything needed. The existing `root.go` symbols were confirmed by direct `Read` (file was last ingested before D.1; Hylla state is pre-D.1). Fell back to `Read` for live file state — standard practice for files changed since last ingest.

## Unit D.3 — Round 1

- **Builder:** go-builder-agent
- **Started:** 2026-05-17
- **Files touched:**
  - `cmd/rak/integration_test.go` (7 new test functions appended)
  - `drops/DROP_D_FILES_FROM_PIPE/PLAN.md` (D.3 state: todo → done)
- **Mage targets run:**
  - `mage build` — pass (no output)
  - `mage test` — pass (all 8 packages green; cmd/rak ran 1.363s)
- **Notes:**

  Added `"errors"` and `"fmt"` to `integration_test.go` import block (required for `errors.Is` in Test 7 and `fmt.Sprintf` in Test 7 loop).

  **Test 1 — StdinList:** Feeds relative paths `testdata/tree/a.txt` + `testdata/tree/sub/nested.txt` via stdin to `--files-from -`. Asserts `parsed.Total` matches the four `treeExpected*` constants (B=20, L=2, W=4, C=20). Key insight: both files map to dirKey `"."` because `FilesFromLister` yields each file with `relPath = base`; the single directory is labeled `"<stdin>"` after `labelDirectories`.

  **Test 2 — EmptyStdin:** Empty `strings.NewReader("")` as stdin. `runDirectory` always calls `RenderTree` even on zero dirs, so JSON is well-formed. Asserts `Total.Bytes == 0`.

  **Test 3 — SkipsEmptyLines:** Same fixture paths with blank lines interspersed. Same totals as Test 1 — proves empty lines are skipped without affecting count.

  **Test 4 — HashFileWorks:** Creates `#draft.md` (8 bytes) in `t.TempDir()`, feeds its absolute path via stdin. Asserts `Total.Bytes == 8`. Absolute path avoids CWD dependency.

  **Test 5 — PositionalArgConflict:** `--files-from - .` triggers Guard A in `PersistentPreRunE`. Asserts `err != nil` and `"cannot combine"` substring.

  **Test 6 — NoGitignoreHardErrors:** `--files-from - --no-gitignore` triggers Guard B. Asserts `err != nil` and `"--no-gitignore"` substring.

  **Test 7 — MaxFiles:** Creates 3 real temp files, feeds all three absolute paths, `--max-files 1`. Asserts `cmd.Execute()` returns an error and `errors.Is(err, ErrMaxFilesExceeded)` is true. Cobra preserves the wrapped error chain through `Execute()` — confirmed by parity with `TestRootCmd_NoGitignoreInRepo_Errors` pattern.

  No scope expansion. No files touched outside unit's declared path (`cmd/rak/integration_test.go`) plus the allowed drop files.

## Hylla Feedback

None — Hylla answered everything needed. All symbol lookups (`ErrMaxFilesExceeded`, `treeResult`, `dirResult`, `treeExpected*` constants) were confirmed by direct `Read` of committed + recently-modified files. No Hylla queries needed; the relevant files were all changed since last ingest (D.1, D.2 work) so `Read` is the correct evidence source.

## Unit D.4 — Round 1

- **Builder:** go-builder-agent
- **Started:** 2026-05-17
- **Files touched:**
  - `docs/tapes/pipe.tape` (NEW — VHS tape script)
  - `README.md` (Piping section + Common invocations entry + Flags table row + Roadmap update)
  - `drops/DROP_D_FILES_FROM_PIPE/PLAN.md` (D.4 state: todo → done; drop state: planning → done)
- **Mage targets run:**
  - `mage build` — pass (no output; doc-only changes, no Go changes)
- **Notes:**

  No Go code was changed in this unit. All changes are documentation only.

  **`docs/tapes/pipe.tape`**: new VHS tape script demonstrating `git ls-files '*.go' | rak --files-from -`. Follows the established style of all existing tapes in `docs/tapes/` (Output, Set Shell/FontSize/Width/Height/Theme/TypingSpeed/Padding, Hide/clear/Show, Sleep, Type, Sleep, Enter, Sleep 4s). Output path is `docs/pipe.gif` — consistent with the pattern all other tapes use. The dev must run `vhs main/docs/tapes/pipe.tape` from the repo root to generate the `.gif` binary and commit `docs/pipe.gif`.

  **`README.md` changes:**
  1. `## Common invocations`: added `git ls-files '*.go' | rak --files-from -` line.
  2. New `## Piping` section inserted after the sort-files gif, before `## Default behavior`. Contains a 2-sentence narrative, a fenced `sh` code block with all four canonical invocations (`rg --files`, `git ls-files`, `find`, `gh pr diff`), and a gif embed `![rak --files-from demo](docs/pipe.gif)`.
  3. `## Flags` table: added `--files-from <FILE>` row between `--binary` and `--version`.
  4. `## Roadmap → v0.2`: updated the `--files-from` bullet from "in development" to "shipped in v0.2.0 (see Piping)".

  **Cobra `Example:` verification**: confirmed present at `cmd/rak/root.go` lines 97-101 (`rg --files | rak --files-from -` and `git ls-files '*.go' | rak --files-from -`). D.2 placed them correctly; no re-addition needed.

## Hylla Feedback

N/A — task touched non-Go files only (`docs/tapes/pipe.tape`, `README.md`, drop PLAN.md and BUILDER_WORKLOG.md). Hylla is Go-only today.
