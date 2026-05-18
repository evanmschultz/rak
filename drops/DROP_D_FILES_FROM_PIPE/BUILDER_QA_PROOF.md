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
