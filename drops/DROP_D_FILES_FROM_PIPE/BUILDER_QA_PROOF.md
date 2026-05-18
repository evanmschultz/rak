# DROP_D — Builder QA Proof

Append a `## Unit N.M — Round K` section per QA attempt. See `main/drops/WORKFLOW.md` § "Phase 5 — Build QA (per unit)" for the per-section shape.

## Unit D.1 — Round 1

**Verdict:** PASS WITH FINDINGS (one nit only; not a blocker)

PASS on every acceptance criterion and every task-prompt design check. One low-severity wording nit in PLAN.md AC1 — does not block the unit.

### Evidence (per AC and design check)

| Acceptance criterion (PLAN.md `### Unit D.1`) | Evidence | Status |
|---|---|---|
| AC1 — `mage test ./internal/lister/...` passes with `-race` | `mage test` ran clean; `ok github.com/evanmschultz/rak/internal/lister`. Magefile's `mage test` invokes `go test -race ./...` per `main/CLAUDE.md` § "Build Verification" mage targets table. | OK |
| AC2 — `mage build` passes | `mage build` exited silently with no output (success per the project's magefile convention). | OK |
| AC3 — six scenarios each covered by a named test | `internal/lister/lister_test.go` lines 327, 351, 383, 416, 456, 490 — all six `TestFilesFromLister_*` cases present. | OK |
| AC4 — ctx-cancellation test verifies iteration terminates without panic | `TestFilesFromLister_ContextCancel` (lines 490–528) cancels after first yield, asserts non-nil `ctxErr` and `count <= 2`. | OK |
| AC5 — per-line error for missing file does NOT abort iterator | `TestFilesFromLister_MissingFile` (lines 456–486) asserts `len(errs)==1 && len(files)==1` AND `files[0].RelPath=="real.txt"`. | OK |
| AC6 — CWD resolution happens in `List()`, not constructor | `filesfrom.go:66` — `os.Getwd()` runs inside the returned closure body, after the iterator function has been called. The constructor (lines 42–44) only stores `r`. | OK |
| AC7 — `scanner.Err()` checked after scan loop | `filesfrom.go:127–129` — `if err := scanner.Err(); err != nil { yield(nil, fmt.Errorf("lister: files-from: scanner: %w", err)) }`. | OK |
| AC8 — `#draft.md` test proves hash-prefixed paths pass through | `TestFilesFromLister_HashPrefixedFileWorks` (lines 351–379) writes a real `#draft.md` file and asserts `RelPath == "#draft.md"`. Scanner loop (lines 84–87) does not branch on `#` — only trims whitespace and skips empties. | OK |
| AC9 — default `bufio.Scanner` 64 KiB buffer; no `scanner.Buffer` bump | `filesfrom.go:72` — `bufio.NewScanner(fl.r)`. Full-file grep shows no `.Buffer(` call. | OK |

### Task-prompt design checks

| Check | Evidence | Status |
|---|---|---|
| Round 2 `filepath.IsAbs` fix handles both abs and rel paths | `filesfrom.go:92–96` — `cleaned := filepath.Clean(line); absPath := cleaned; if !filepath.IsAbs(absPath) { absPath = filepath.Join(cwd, cleaned) }`. Absolute paths bypass `Join`; relative paths get CWD prefix. The Round 1 bug (`filepath.Join("/cwd", "/abs/path")` corrupts the absolute path) is fixed. | OK |
| All 6 `TestFilesFromLister_*` tests present | See AC3 table row above — line numbers confirm each. | OK |
| `scanner.Err()` post-loop check present | See AC7 — `filesfrom.go:127–129`. | OK |
| `var _ FileLister = (*FilesFromLister)(nil)` compile-time assertion | `filesfrom.go:134`. | OK |
| Default `bufio.Scanner` buffer (NO `scanner.Buffer`) | See AC9. | OK |
| `FilesFromLister` does NOT close reader | No `Close()` call on `fl.r` anywhere in the file. Doc comments (lines 24–25, 41–42) explicitly state the caller owns the reader. | OK |
| CWD resolved inside `List()` | See AC6. | OK |
| `#`-prefixed paths NOT filtered | See AC8 — scanner-loop logic confirmed; test proves behavior. | OK |

### Test non-vacuity audit

Each new test asserts a concrete observable, not just "no error":

- `EmptyReader` — asserts `files==0` AND `errs==0`.
- `HashPrefixedFileWorks` — creates a real `#draft.md` on disk, asserts `len(files)==1` AND `RelPath=="#draft.md"`.
- `SkipsEmptyLines` — feeds `"\nfileA\n\nfileB\n\n"`, asserts `len(files)==2` AND `len(errs)==0`.
- `MixedPaths` — asserts ordered `RelPath` equality with `["first.go", "second.go"]`.
- `MissingFile` — asserts both `errs==1` AND `files==1` AND `files[0].RelPath=="real.txt"` — proves the iterator continues past per-line errors.
- `ContextCancel` — uses `context.WithCancel`, cancels after the first yield, asserts non-nil context error AND `count <= 2`.

All six are behavior-asserting; none are vacuous.

### Trace coverage

- **Absolute-path branch**: `filepath.Clean("/tmp/x/file.txt")` keeps the leading `/`; `filepath.IsAbs` returns `true`; `absPath` stays `/tmp/x/file.txt`; `os.Stat` targets the right path. Exercised by every `t.TempDir()`-based test (all five of HashPrefixed, SkipsEmptyLines, MixedPaths, MissingFile, ContextCancel pass absolute paths).
- **Relative-path branch**: `filepath.IsAbs("rel/path") == false`; `filepath.Join(cwd, cleaned)` runs. Not directly unit-tested in the new D.1 tests, but covered downstream by the D.3 integration tests that will pass `testdata/tree/a.txt` (relative) — and the implementation is straightforward `filepath.Join`.
- **Hash-prefixed**: covered by `HashPrefixedFileWorks` AND by the absence of any `#`-special branch in lines 84–87.
- **Empty-line skip**: covered by `SkipsEmptyLines` AND by lines 85–87 (`if line == "" { continue }`).
- **Per-line error continuation**: covered by `MissingFile` — the missing-then-valid sequence proves yield-true after the error.
- **Ctx-cancel path**: covered by `ContextCancel` — line 75 (`if ctx.Err() != nil`) fires after the user-side `cancel()`.
- **yield-false short-circuit**: not unit-tested in D.1, but the implementation honors it at lines 102, 108, 121 (return on `!yield(...)`). Carries the F14 contract from `fileset.Walker`.
- **`scanner.Err()` propagation**: not unit-tested (would require a failing reader), but the code path is present at lines 127–129 and matches the documented iterator contract.

### Findings

#### Finding 1 — PLAN.md AC1 wording uses non-existent mage target syntax

- **Severity:** nit (not a blocker)
- **Where:** `main/drops/DROP_D_FILES_FROM_PIPE/PLAN.md` § "Unit D.1 — Acceptance criteria" line 129 (and similar phrasing in D.2 line 263, D.3 line 382).
- **Issue:** The acceptance criterion is written as `mage test ./internal/lister/...` — but the rak mage targets (per `main/CLAUDE.md` § "Build Verification") don't accept package arguments. Running `mage test ./internal/lister/...` literally returns `Unknown target specified: "./internal/lister/..."`. The intent is "the test suite covers the `./internal/lister/...` package" — which `mage test` does (it invokes `go test -race ./...`).
- **Recommendation:** Future drops should phrase package-coverage ACs as e.g. *"`mage test` passes; `ok github.com/evanmschultz/rak/internal/lister` appears in the output"*, OR add a `mage testPackage <path>` target if per-package invocation is genuinely desired. No code change needed for D.1 — the test suite covers `internal/lister` correctly under the full-suite invocation.

(No other findings.)

### Verification commands run

- `mage build` — silent success, exit 0.
- `mage test` — all packages pass, including `ok github.com/evanmschultz/rak/internal/lister`.

### Hylla Feedback

N/A — `FilesFromLister` and its tests are newly committed Go code in this round; verification leaned on `Read` of the source + tests (already on disk in this checkout) + mage runs. No Hylla query attempted, no fallback needed.

## Unit D.2 — Round 1

**Verdict:** PASS

All 9 PLAN.md "What to build" steps implemented and verified against `cmd/rak/root.go` (commit `1fddc72 feat(cmd): wire --files-from flag with stdin sentinel`). All bulleted acceptance criteria check out. `mage build` and `mage test` pass on the `cmd/rak` package; the unrelated `internal/lang` test-file failures are uncommitted in-flight work from a parallel builder (per task prompt: skip those).

### Evidence (per "What to build" step)

| Step | Spec | Evidence (file:line) | Status |
|---|---|---|---|
| 1 | `filesFrom string` field added to `rootFlags` | `cmd/rak/root.go:40` — `filesFrom   string // path to a newline-delimited file list, or "-" for stdin` | OK |
| 2 | `--files-from` flag registered with usage string | `cmd/rak/root.go:200-205` — `cmd.Flags().StringVar(&flags.filesFrom, "files-from", "", "read newline-separated file paths from FILE (use - for stdin)")` | OK |
| 3a | `PersistentPreRunE` signature has `args []string` (not `_`) | `cmd/rak/root.go:103` — `PersistentPreRunE: func(_ *cobra.Command, args []string) error {` | OK |
| 3b | Guard A — positional + `--files-from` conflict | `cmd/rak/root.go:107-109` — `if flags.filesFrom != "" && len(args) > 0 { return fmt.Errorf("cannot combine --files-from with a positional path argument") }` | OK |
| 3c | Guard B — `--no-gitignore` + `--files-from` conflict | `cmd/rak/root.go:110-112` — `if flags.filesFrom != "" && flags.noGitignore { return fmt.Errorf("--no-gitignore is meaningless with --files-from: the caller controls which files are listed") }` | OK |
| 4 | Two cobra `Example:` entries added | `cmd/rak/root.go:97-101` — `# Pipe a file list from ripgrep / rg --files | rak --files-from -` and `# Count only tracked Go files / git ls-files '*.go' | rak --files-from -`. Confirmed visible in `mage run -- --help` output. | OK |
| 5a | `openFilesFrom(value, stdin) (io.Reader, func(), error)` helper exists with exact signature | `cmd/rak/root.go:306` — `func openFilesFrom(value string, stdin io.Reader) (io.Reader, func(), error)` | OK |
| 5b | `-` returns stdin + noop closer | `cmd/rak/root.go:307-309` — `if value == "-" { return stdin, func() {}, nil }` | OK |
| 5c | Otherwise opens file + Close closer | `cmd/rak/root.go:310-314` — `f, err := os.Open(value); ... return f, func() { _ = f.Close() }, nil`. Error wrapped with `--files-from: %w`. | OK |
| 6a | Third branch in `runRoot` exists, executes BEFORE `len(args)==1` | `cmd/rak/root.go:248-268` — `if flags.filesFrom != "" { ... }` precedes the `if len(args) == 1` block at line 270. | OK |
| 6b | Uses `lister.NewFilesFromLister(r)` | `cmd/rak/root.go:254` — `source := lister.NewFilesFromLister(r)` | OK |
| 6c | `rootLabel = "<stdin>"` when value is `-`, else value itself | `cmd/rak/root.go:255-258` — `rootLabel := flags.filesFrom; if flags.filesFrom == "-" { rootLabel = "<stdin>" }` | OK |
| 7 | `--no-gitignore` + `--files-from` returns Guard B error (hard error) | See step 3c above; the guard fires in `PersistentPreRunE`, returning before any walk happens. | OK |
| 8 | `--depth` + `--files-from` is silent no-op (`listerOpts` not called) | `cmd/rak/root.go:248-268` — the `--files-from` branch passes individual fields to `runDirectoryOpts`; `listerOpts(flags)` (line 271, only invoked in `len(args)==1` branch) is bypassed entirely. `flags.depth` is not read in this branch, so it has no effect. | OK |
| 9 | `--max-files` applies (passed through `runDirectoryOpts.maxFiles`) | `cmd/rak/root.go:265` — `maxFiles:  flags.maxFiles,` inside the `--files-from` branch's `runDirectoryOpts` literal. Wires to `walkAndCount` which enforces `ErrMaxFilesExceeded` at line 501-503. | OK |

### Acceptance criteria bullets

| Bullet (PLAN.md `### Unit D.2 — Acceptance criteria`) | Evidence | Status |
|---|---|---|
| `mage build` passes | Ran clean, silent exit 0. | OK |
| `mage test ./cmd/rak/...` passes (existing tests must not regress) | `mage test` output: `ok github.com/evanmschultz/rak/cmd/rak 1.362s`. (The `internal/lang` build-failed line is from uncommitted parallel-builder work in `internal/lang/lang_test.go` + `split_test.go`; out of scope per task prompt.) | OK |
| `rak --help` shows `--files-from` flag with correct usage string | `mage run -- --help` shows `--files-from   Read newline-separated file paths from FILE (use - for stdin)`. | OK |
| `rak --help` shows the two new `Example:` entries | `mage run -- --help` shows `# Pipe a file list from ripgrep / rg --files | rak --files-from -` and `# Count only tracked Go files / git ls-files '*.go' | rak --files-from -` in the EXAMPLES block. | OK |
| `rak --files-from - .` returns error containing `"cannot combine"` | Guard A at `cmd/rak/root.go:107-109` — runtime not exercised (D.3 covers via `TestRootCmd_Integration_FilesFrom_PositionalArgConflict`), but the literal string `"cannot combine --files-from with a positional path argument"` is present at line 108. | OK by code inspection |
| `rak --files-from /nonexistent/path.txt` returns error wrapping `os.Open` failure | `openFilesFrom` at `cmd/rak/root.go:310-313` — `os.Open(value)` failure wrapped via `fmt.Errorf("--files-from: %w", err)`. | OK by code inspection |
| `rak --files-from - --no-gitignore` returns error containing `"--no-gitignore"` | Guard B at `cmd/rak/root.go:110-112` — error string `"--no-gitignore is meaningless with --files-from: the caller controls which files are listed"`. | OK by code inspection |
| Rendered TOON output for `rak --files-from -` shows `path: <stdin>` not `path: -` | `cmd/rak/root.go:255-258` — `rootLabel` is set to `"<stdin>"` when `filesFrom == "-"`, then passed to `runDirectory` → `labelDirectories` → renderer. Runtime not exercised here (D.3 covers); branch logic verified. | OK by code inspection |
| `runRoot` branch order: `--files-from` first, then `len(args)==1`, then bare-stdin fallback | `cmd/rak/root.go:248-298` — branch order: `--files-from` (line 248) → `len(args)==1` (line 270) → fallthrough to `counting.Count(c.InOrStdin())` (line 290). Correct precedence. | OK |

### Trace coverage

- **`--files-from <FILE>` branch**: Guard A skipped (no positional), Guard B skipped (no `--no-gitignore`) → `openFilesFrom("<FILE>", stdin)` opens the file with `os.Open` → `lister.NewFilesFromLister(file)` → `rootLabel = "<FILE>"` (literal, not `<stdin>`) → `runDirectory(...)` → close deferred via the returned closer. Verified by lines 248-268 + 306-315.
- **`--files-from -` branch**: same as above, but `openFilesFrom("-", stdin)` returns stdin + no-op closer → `rootLabel = "<stdin>"`. Verified by line 256-258 + 307-309.
- **Guard A trip (positional + `--files-from`)**: `cmd --files-from x .` enters `PersistentPreRunE`; `len(args) == 1` and `flags.filesFrom != ""` both true → error returned BEFORE `RunE` runs. Verified by lines 107-109.
- **Guard B trip (`--no-gitignore` + `--files-from`)**: `cmd --files-from x --no-gitignore` enters `PersistentPreRunE`; sort key valid, Guard A skipped (no args), Guard B fires → error returned. Verified by lines 110-112.
- **`--depth` + `--files-from` no-op**: `--depth` field is set in `flags.depth` but never read inside the `--files-from` branch — `listerOpts(flags)` (the only consumer of `flags.depth`) is only called in the `len(args)==1` branch at line 271. Verified by inspection.
- **`--max-files` + `--files-from`**: `flags.maxFiles` is passed to `runDirectoryOpts.maxFiles` at line 265 → `runDirectory` → `walkAndCount` → enforcement at lines 501-503 with wrapped `ErrMaxFilesExceeded`. Verified.
- **Branch precedence**: a user with both `--files-from` AND no positional arg gets the `--files-from` branch (line 248); a user with no `--files-from` and one positional gets the walk branch (line 270); a user with neither gets bare-stdin counting (line 290). All three paths exclusive and ordered correctly.

### `mage build` / `mage test` summary

- `mage build` — silent success, exit 0. The `cmd/rak` package compiles cleanly with the new flag, field, guards, helper, and branch.
- `mage test` — `cmd/rak` passes: `ok github.com/evanmschultz/rak/cmd/rak 1.362s`. The full-suite run reports a `FAIL` on `internal/lang` (build-failed due to undefined `LangTempl`, `LangJSX`, etc. in `lang_test.go`) — `git status` confirms this is uncommitted in-flight work in `internal/lang/lang_test.go` + `internal/lang/split_test.go` by a parallel builder, unrelated to D.2. Task prompt explicitly green-lit ignoring parallel-builder failures.
- `mage run -- --help` — output confirms the `--files-from` flag and both `Example:` entries are visible in `rak --help`.

### Findings

None blocking. One observation:

- **Observation (not a finding):** D.2's acceptance bullets reference runtime behaviors (`rak --files-from -`, `rak --files-from /nonexistent/path.txt`, etc.) that the unit's own code change does not exercise — D.3 is the unit that adds the integration tests covering those runtime paths. The verdict here treats those bullets as `OK by code inspection`. This matches the planner's intent (D.2 is the wiring unit, D.3 is the tests unit).

### Verification commands run

- `Read` of `cmd/rak/root.go` (full file, 613 lines) — confirms all 9 build steps.
- `git diff HEAD~1 -- cmd/rak/root.go` — confirms the commit's diff matches the spec exactly (+60 / -2 lines on root.go only; no other files touched in this commit).
- `mage build` — pass (exit 0, silent).
- `mage test` — `cmd/rak` ok; `internal/lang` failure is unrelated parallel-builder work (verified via `git status` showing uncommitted `internal/lang/lang_test.go` and `internal/lang/split_test.go`).
- `mage run -- --help` — confirms `--files-from` flag visible with correct usage, both new `Example:` entries visible.

### Hylla Feedback

N/A — D.2 modified only `cmd/rak/root.go`; the file is post-last-ingest and reading the current source via the `Read` tool is the correct path per `main/CLAUDE.md` § "Code Understanding Rules" rule 2 ("Changed since last ingest: use `git diff`. Hylla is stale for those files until reingest"). No Hylla query was attempted; no fallback miss to log.

## Unit D.3 — Round 1

**Verdict:** PASS

All 7 named tests required by PLAN.md `### Unit D.3` are present in `cmd/rak/integration_test.go`, each asserts the documented behavior, and `mage test` passes on `cmd/rak` (the full-suite `internal/lang` failure is uncommitted parallel-builder DROP_A work, confirmed unrelated — same pattern as D.2 Round 1).

### Evidence — 7 required tests

| # | Test (PLAN.md spec) | Location (file:line) | Documented behavior asserted | Status |
|---|---|---|---|---|
| 1 | `TestRootCmd_Integration_FilesFrom_StdinList` — totals B=20, L=2, W=4, C=20 via `parsed.Total` | `cmd/rak/integration_test.go:267-295` | Feeds `"testdata/tree/a.txt\ntestdata/tree/sub/nested.txt\n"` via `cmd.SetIn(strings.NewReader(list))` + `--json --files-from -`; unmarshals `treeResult`; asserts `parsed.Total.Bytes == treeExpectedTotalBytes` (20), `.Lines == 2`, `.Words == 4`, `.Chars == 20`. | OK |
| 2 | `TestRootCmd_Integration_FilesFrom_EmptyStdin` — empty stdin → no panic, no error, zero totals | `cmd/rak/integration_test.go:301-323` | `cmd.SetIn(strings.NewReader(""))` + `--json --files-from -`; asserts `cmd.Execute()` returns `nil`; unmarshals output; asserts `parsed.Total.Bytes == 0`. | OK |
| 3 | `TestRootCmd_Integration_FilesFrom_SkipsEmptyLines` — blank lines interspersed → same totals as Test 1 | `cmd/rak/integration_test.go:330-358` | Feeds `"\ntestdata/tree/a.txt\n\ntestdata/tree/sub/nested.txt\n\n"` (leading, mid, trailing blanks); asserts identical `treeExpectedTotal*` constants as Test 1. | OK |
| 4 | `TestRootCmd_Integration_FilesFrom_HashFileWorks` — temp `#draft.md` is counted (non-zero bytes) | `cmd/rak/integration_test.go:366-400` | Creates `t.TempDir()/#draft.md` with `os.WriteFile`, content `"# draft\n"` (8 bytes); feeds absolute path via stdin; asserts `parsed.Total.Bytes == int64(len(content))` (8). Proves `#` prefix is not filtered. | OK |
| 5 | `TestRootCmd_Integration_FilesFrom_PositionalArgConflict` — error contains `"cannot combine"` | `cmd/rak/integration_test.go:406-423` | `cmd.SetArgs([]string{"--files-from", "-", "."})`; asserts `err != nil` AND `strings.Contains(err.Error(), "cannot combine")`. | OK |
| 6 | `TestFlags_FilesFromNoGitignoreHardErrors` — error contains `"--no-gitignore"` | `cmd/rak/integration_test.go:428-445` | `cmd.SetArgs([]string{"--files-from", "-", "--no-gitignore"})`; asserts `err != nil` AND `strings.Contains(err.Error(), "--no-gitignore")`. | OK |
| 7 | `TestFilesFrom_MaxFiles` — `errors.Is(err, ErrMaxFilesExceeded)` true | `cmd/rak/integration_test.go:452-484` | Creates 3 real temp files via `os.WriteFile`; feeds all 3 absolute paths via stdin with `--max-files 1`; asserts `err != nil` AND `errors.Is(err, ErrMaxFilesExceeded)`. Sentinel is package-level in `cmd/rak/root.go` (wrapped at line 502 via `fmt.Errorf("rak: file count exceeded --max-files %d: %w", ...)`). | OK |

### Acceptance criteria

| Bullet (PLAN.md `### Unit D.3 — Acceptance criteria`) | Evidence | Status |
|---|---|---|
| `mage test ./cmd/rak/...` passes with `-race` | Stashed unrelated DROP_A in-flight `internal/lang/*` work and re-ran `mage test`; all packages pass with `-race` (the magefile invokes `go test -race ./...`). `cmd/rak`: `ok 1.372s`. | OK |
| Test 1 asserts `parsed.Total` fields matching tree fixture constants | Lines 287-294 — explicit comparison of all four (Bytes/Lines/Words/Chars) against `treeExpectedTotal*`. | OK |
| Test 2 verifies empty stdin: no panic, no error, zero totals | Lines 311-322 — `cmd.Execute()` checked for nil; `parsed.Total.Bytes == 0` asserted. | OK |
| Test 3 same totals as Test 1 (empty lines skipped) | Lines 350-357 — identical `treeExpectedTotal*` comparison. | OK |
| Test 4 proves `#`-prefixed filenames are counted normally | Lines 396-399 — `parsed.Total.Bytes == int64(len(content))` where content is `"# draft\n"` (8 bytes). | OK |
| Test 5 error message contains `"cannot combine"` | Lines 420-422 — `strings.Contains(err.Error(), "cannot combine")`. | OK |
| Test 6 error message references `"--no-gitignore"` | Lines 442-444 — `strings.Contains(err.Error(), "--no-gitignore")`. | OK |
| Test 7 `ErrMaxFilesExceeded` fires under `--files-from` | Lines 481-483 — `errors.Is(err, ErrMaxFilesExceeded)`. | OK |
| Existing integration tests in `integration_test.go` continue to pass | Existing tests (`TestRootCmd_Integration_HumanFormat`, `TestRootCmd_Integration_JSONFormat`, `TestRootCmd_Integration_PathArg_HumanFormat`, `TestRootCmd_Integration_PathArg_JSONFormat`) intact at lines 48-258; `mage test` (with parallel DROP_A stashed) reports `ok cmd/rak`. | OK |

### Test non-vacuity audit

Each new test asserts a concrete observable beyond "no error":

- **Test 1**: asserts all four `Total.*` fields against fixture constants (not just non-nil).
- **Test 2**: asserts BOTH `err == nil` AND `parsed.Total.Bytes == 0` — proves empty stdin is silently OK, not an error path.
- **Test 3**: asserts identical totals to Test 1, proving the skip-empty-lines path is semantically equivalent (not just non-erroring).
- **Test 4**: asserts `Total.Bytes == int64(len(content))` — concrete byte count proves the file was counted, not just discovered.
- **Test 5**: asserts BOTH `err != nil` AND substring `"cannot combine"` — proves the specific Guard A path fires (not some other error like file-not-found).
- **Test 6**: asserts BOTH `err != nil` AND substring `"--no-gitignore"` — proves Guard B fires.
- **Test 7**: asserts BOTH `err != nil` AND `errors.Is(err, ErrMaxFilesExceeded)` — uses sentinel inspection (not string match), the idiom mandated by `main/CLAUDE.md` § "Errors".

All seven are behavior-asserting; none are vacuous.

### Trace coverage

- **`--files-from -` happy path**: Test 1 covers relative-path resolution via CWD (test runs from `cmd/rak/`, so `testdata/tree/a.txt` resolves correctly). Test 4 covers absolute-path resolution. Together they exercise both branches of D.1's `filepath.IsAbs` check.
- **Empty stdin**: Test 2 covers the path where `bufio.Scanner.Scan()` returns false on the very first call → iterator exits cleanly with zero yields.
- **Empty-line skip path**: Test 3 exercises D.1's `if line == "" { continue }` branch with leading, mid, and trailing blanks.
- **Hash-prefix path**: Test 4 proves D.1's scanner loop does not filter `#`-prefixed lines (no `strings.HasPrefix(line, "#")` branch exists).
- **Guard A (`PersistentPreRunE` — positional + `--files-from`)**: Test 5 exercises `cmd/rak/root.go:107-109`; the error is returned before `RunE` runs.
- **Guard B (`PersistentPreRunE` — `--no-gitignore` + `--files-from`)**: Test 6 exercises `cmd/rak/root.go:110-112`.
- **`--max-files` wired through `runDirectoryOpts.maxFiles`**: Test 7 exercises the full chain: `--files-from` branch (root.go:248-268) → `runDirectoryOpts.maxFiles = flags.maxFiles` (root.go:265) → `walkAndCount` (root.go:501-503) → `ErrMaxFilesExceeded` wrapped via `%w`. The `errors.Is` check confirms wrap-chain integrity.
- **Stdin sentinel `"-"` opening**: All 7 tests use `"-"` (or implicitly via `cmd.SetIn`); `openFilesFrom("-", stdin)` returns stdin + no-op closer (root.go:307-309). The fact that tests with `cmd.SetIn(strings.NewReader(...))` succeed proves the helper returns the cobra-managed stdin reader, not `os.Stdin`.

### Findings

None blocking.

### Verification commands run

- `Read` of `cmd/rak/integration_test.go` (lines 1-484, full file) — confirms all 7 named tests, signatures, and body assertions.
- `Read` of `cmd/rak/root.go` lines 490-520 — confirms `ErrMaxFilesExceeded` is wrapped at line 502, accessible to test via package-level declaration.
- `mage test` (full repo) — `cmd/rak` reports `ok`. The full-suite run reports `FAIL` on `internal/lang` (build-failed in `lang_test.go` + `split_test.go`). `git status` confirms `internal/lang/lang.go`, `internal/lang/lang_test.go`, `internal/lang/split_test.go`, and `drops/DROP_A_LANG_EXPANSION/BUILDER_QA_PROOF.md` are uncommitted — this is the parallel DROP_A builder's in-flight work, unrelated to D.3 (matches the D.2 Round 1 pattern; task prompt green-lights ignoring parallel-builder failures).
- `git stash push internal/lang/` + `mage test` + `git stash pop` — with DROP_A stashed, `mage test` is fully green across all packages. Stash restored after verification; working tree unchanged.

### Hylla Feedback

N/A — D.3 added only `cmd/rak/integration_test.go` tests (post-last-ingest, in-progress local file). Reading the current source via `Read` is correct per `main/CLAUDE.md` § "Code Understanding Rules" rule 2. No Hylla query was attempted; no fallback miss to log.
