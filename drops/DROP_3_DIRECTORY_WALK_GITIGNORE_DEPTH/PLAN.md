# DROP_3 — DIRECTORY_WALK_GITIGNORE_DEPTH

**State:** building
**Blocked by:** DROP_2
**Paths (expected):** `main/internal/fileset/` (new package — `File` type, `Walker`), `main/internal/ignore/` (new package — gitignore + include/exclude globs), `main/cmd/rak/root.go` (wire `len(args)==1` path case into walker), `main/cmd/rak/root_test.go` (extend) or `main/cmd/rak/integration_test.go` (extend fixture tree), `main/cmd/rak/testdata/` (may grow a real directory fixture), plus per-package `*_test.go` files
**Packages (expected):** `github.com/evanmschultz/rak/internal/fileset` (new), `github.com/evanmschultz/rak/internal/ignore` (new), `github.com/evanmschultz/rak/cmd/rak`
**PLAN.md ref:** main/PLAN.md → `DROP_3_DIRECTORY_WALK_GITIGNORE_DEPTH` row
**Workflow:** main/drops/WORKFLOW.md
**Started:** 2026-04-21
**Closed:** —

## Scope

Land the directory-walking spine behind `rak [path]`: `internal/fileset` exposes the `File` type (with `Open() (io.ReadCloser, error)` + `Peek(n int) ([]byte, error)` per decision 25) and a `Walker` that emits `iter.Seq2[*File, error]` over a tree. `internal/ignore` unifies `.gitignore` parsing and `--include` / `--exclude` glob matchers behind a single `Matcher` interface. Binary file detection via `File.Peek(512)` skips binaries by default (adds `--binary` escape hatch). Root command's `len(args)==1` path case — which Unit 2.3 rejected with a "walker lands in Drop 3" error — now walks the directory, counts every text file, and renders per-directory aggregates through the existing `internal/render` boundary. **No language detection, no code-aware splits, no stdin changes, no token counting, no parallelism yet** — all deferred to later drops. Expected decomposition: 6 units (3.0 deps / 3.1 ignore / 3.2 fileset.File / 3.3 fileset.Walker / 3.4 binary detection / 3.5 root wiring + per-dir aggregation).

## Planner

Six atomic units. 3.0 adds deps (must land before anything that imports them). 3.1 + 3.2 are leaves — new packages `internal/ignore` and `internal/fileset` with zero cross-dependency, so the builder can queue them in any order once 3.0 closes. 3.3 wires the Walker on top of both leaves. 3.4 adds binary detection as a File method (depends only on 3.1's `Peek`). 3.5 wires the walker into `cmd/rak/root.go`'s `len(args) == 1` path and adds the root-command flags / per-dir aggregation. Splitting `cmd/rak` wire-up out of 3.3 keeps the Walker unit test-only and lets QA falsify the walker in isolation before the CLI surface moves.

Import DAG (reconfirms main/CLAUDE.md § "Project Structure" → "Import DAG"): `ignore` + `counting` are leaves; `fileset → ignore`; `cmd/rak → fileset + counting + render`. No cycles.

### Unit 3.0 — Add gitignore + glob deps via mage addDep

**State:** done
**Paths:** `main/go.mod`, `main/go.sum`
**Packages:** — (dep-management only)
**Blocked by:** —
**Acceptance:**
- Runs `mage addDep github.com/sabhiram/go-gitignore` and `mage addDep github.com/bmatcuk/doublestar/v4` from `main/` — **no raw `go get`**. If `mage addDep` is missing (Drop 2.0 landed it), that's a blocker, not a bypass path.
- `main/go.mod` has `require (` entries for both modules at their latest stable tags.
- `main/go.sum` populated for both modules + transitive deps (sabhiram has zero transitive deps; doublestar/v4 has zero transitive deps — if either pulls one in, flag and return to orch).
- No code uses either dep yet; `mage build` and `mage test` both pass clean (the unit imports nothing, so the deps sit "unused" in `go.mod` until 3.1 and 3.2 consume them — this is the documented Drop 2 workflow: add dep first, let importing code land second, let `mage ci` surface any leftover inconsistency at drop-end).

### Unit 3.1 — internal/ignore: Matcher interface + gitignore + include/exclude globs

**State:** done
**Paths:** `main/internal/ignore/ignore.go`, `main/internal/ignore/gitignore.go`, `main/internal/ignore/glob.go`, `main/internal/ignore/ignore_test.go`
**Packages:** `github.com/evanmschultz/rak/internal/ignore` (new)
**Blocked by:** 3.0
**Acceptance:**
- `ignore.go` defines `Matcher` interface: `Match(relPath string, isDir bool) bool` (returns true when the path should be **ignored / filtered out**; F1). `relPath` uses forward-slash separators per `io/fs` convention (C6); gitignore patterns match forward-slash paths on all platforms, so matchers never see OS-native separators. One concrete constructor `New(roots []GitignoreRoot, includes []string, excludes []string) (Matcher, error)` that composes the three sub-matchers.
- `GitignoreRoot` struct carries `{Dir string, Patterns []string}` (pre-parsed). The walker populates this per directory it enters (3.3 reads `.gitignore` at each dir before descending).
- `gitignore.go` wraps `github.com/sabhiram/go-gitignore` (`ignore.CompileIgnoreLines`). Supports negation `!pattern`, dir-only `pattern/`, double-star `**`, character classes `[abc]`. Hierarchical resolution: a pattern in `foo/.gitignore` applies to files under `foo/` only.
- `glob.go` uses `github.com/bmatcuk/doublestar/v4.Match` for `--include` / `--exclude`. `Match` splits both pattern and path on forward slash (`/`) on all platforms — the correct choice because the walker feeds forward-slash `relPath` values per C6 / `io/fs` convention (doublestar's own `Glob` docs apply the same `/` rule for the same reason). `PathMatch` is rejected: it uses the OS separator (`\` on Windows) and would mis-match forward-slash `relPath` on non-Unix hosts. `filepath.Match` is insufficient because users will expect `**/node_modules` and `src/**/*.go` to work; `filepath.Match` rejects `**`. `--include` is an allow-list (empty = allow everything); `--exclude` is a deny-list (empty = deny nothing). Exclude wins on conflict (F2).
- Precedence order (F3): `--exclude` → `.gitignore` → `--include`. If `--include` is non-empty and a path does NOT match any include, it's ignored (unless already gitignored by then — order doesn't matter because both mean "drop"). If `--exclude` matches, ignored regardless of `--include`. Escape hatch `--no-gitignore` is the cobra flag that omits the gitignore matcher at construction time (wired in 3.5, not here — this unit just has to tolerate the zero-gitignore case).
- `ignore_test.go` is table-driven and covers: empty matcher (allows all), gitignore-only, include-only, exclude-only, all three combined, negation (`!foo.go` re-includes after `*.go`), dir-only pattern (`node_modules/` matches dir but not a file named `node_modules`), `**/vendor` double-star, precedence-wins cases (exclude beats gitignore-negate, include does not override exclude).
- No direct test of a real `.gitignore` file from disk — this unit takes pre-read patterns. Disk IO belongs to 3.3's Walker.
- `mage test ./internal/ignore/...` green. `mage lint` green for the new package.

### Unit 3.2 — internal/fileset.File: Open + Peek + hidden helper

**State:** done
**Paths:** `main/internal/fileset/file.go`, `main/internal/fileset/file_test.go`
**Packages:** `github.com/evanmschultz/rak/internal/fileset` (new)
**Blocked by:** 3.0
**Acceptance:**
- `file.go` defines:
  - `type File struct { Path string; RelPath string; fs fs.FS }` — `Path` is the full walk-relative path as Walker saw it; `RelPath` is the path relative to the walk root. Exported fields; zero-value is not useful (tests construct via `newFile(...)`).
  - `func (f *File) Open() (io.ReadCloser, error)` — opens via `fs.FS.Open` (so the same type works for real `os.DirFS` and `fstest.MapFS`). Wraps any open error with `open %q: %w`.
  - `func (f *File) Peek(n int) ([]byte, error)` — opens, reads up to `n` bytes via `io.ReadFull` tolerating `io.ErrUnexpectedEOF` + `io.EOF` (short files return what they have, no error), closes. Returns `(bytesRead[:k], nil)` where `k ≤ n`. **Load-bearing contract (F4)**: binary detection (3.4) and shebang sniff (Drop 4.1) both call `Peek(512)`.
  - Package-level helper `IsHidden(name string) bool` — returns true when the final path element starts with `.` (excluding `.` and `..`). Used by Walker to skip hidden dirs/files by default.
- `file_test.go`:
  - `TestFile_Open` — open file from a `fstest.MapFS`, read full content, close, verify bytes match.
  - `TestFile_Open_NotFound` — verify error is wrapped with `open ...: ...` and unwraps to `fs.ErrNotExist` via `errors.Is`.
  - `TestFile_Peek` table-driven: content shorter than n, equal to n, longer than n, empty file. Verify no error on short file, no error on exact-match, no error on long file (returns first n).
  - `TestFile_Peek_MultipleCalls` — two `Peek` calls on the same `*File` return identical bytes (each call opens/closes independently; no stateful cursor).
  - `TestIsHidden` — ".", "..", ".git", ".hidden.txt", "normal.txt", "".
- `mage test ./internal/fileset/...` green (only the file tests — walker tests land in 3.3). `mage lint` green.

### Unit 3.3 — internal/fileset.Walker + iter.Seq2 emission

**State:** done
**Paths:** `main/internal/fileset/walker.go`, `main/internal/fileset/walker_test.go`
**Packages:** `github.com/evanmschultz/rak/internal/fileset`
**Blocked by:** 3.1, 3.2
**Acceptance:**
- `walker.go` defines:
  - `type WalkOptions struct { Depth int; IncludeHidden bool; DisableGitignore bool; Includes []string; Excludes []string }` — `Depth == 0` means "no limit" (unlimited descent); `Depth == 1` means walk only the root directory (no subdirs). `Depth` counts **directory edges from the walk root**: `root/file.txt` is depth 0; `root/sub/file.txt` is depth 1 (C7). `IncludeHidden == false` (default) skips hidden files/dirs via `fileset.IsHidden(entry.Name())` (C3 — package-level helper, called on `DirEntry.Name()` inside the `WalkDirFunc` before any `*File` allocation). `DisableGitignore bool` — zero value is **false** so gitignore is ENABLED by default, matching decision 10 (C2). `--no-gitignore` sets `DisableGitignore: true` (wired in 3.5).
  - `type Walker struct { ... }` with `func NewWalker(fsys fs.FS, root string, opts WalkOptions) *Walker`.
  - `func (w *Walker) Walk(ctx context.Context) iter.Seq2[*File, error]` — returns a range-over-func iterator (F5: uses `iter.Seq2` per decision 27(a)). Implementation wraps `fs.WalkDir`. The iterator:
    0. **Yield-false handling (F14)**: the `WalkDirFunc` tracks whether the last `yield(...)` returned false. Once false, it returns `fs.SkipAll` to terminate `fs.WalkDir` cleanly. Returning `nil` (or `fs.SkipDir`) after a false yield would re-invoke `yield` and panic per `go doc iter`. See F14 in cross-unit pins.
    1. Before descending into a dir, reads that dir's `.gitignore` (if present **and** `!DisableGitignore` — F-pin per C2: flag is ENABLED by default) and builds / composes `ignore.Matcher` roots hierarchically.
    2. At each entry: check `ctx.Done()` first — on cancel, yield `(nil, ctx.Err())` and stop.
    3. Skip hidden entries when `!IncludeHidden` (via `fileset.IsHidden(entry.Name())` — package-level helper on `DirEntry.Name()`, per C3).
    4. Skip entries that match the `Matcher`. For directories that match, return `fs.SkipDir` from the WalkDir func so subtree is pruned (performance + correctness).
    5. Enforce `Depth` — count edges from the walk root; when a dir exceeds `Depth`, return `fs.SkipDir` for that dir.
    6. For regular files that survive filtering: yield `(&File{...}, nil)`.
    7. Any `fs.WalkDir` error other than `fs.SkipDir` is yielded as `(nil, err)` wrapped with `walk %q: %w`. The iterator continues past per-entry errors so one unreadable dir doesn't kill the whole walk (F6).
  - Does NOT follow symlinks — stdlib `fs.WalkDir` docs: "WalkDir does not follow symbolic links." F7: symlink `--follow` flag is **deferred to Drop 8.5** and is **not** accepted here; a symlink encountered in-tree is reported via `DirEntry.Type()&fs.ModeSymlink != 0` and simply yielded (or skipped — pick: yielded, because `File.Open` will return an error if the target is missing, letting the caller see it).
- `walker_test.go` uses `testing/fstest.MapFS` per CLAUDE.md § "Tests". Table-driven where possible:
  - `TestWalker_EmptyRoot` — empty dir, no emissions, no error.
  - `TestWalker_SingleFile` — one file at root, one emission.
  - `TestWalker_NestedTree` — three files at depths 1/2/3, `Depth=0` emits all three.
  - `TestWalker_DepthLimit` — same tree, `Depth=1` emits only the root file; `Depth=2` emits the root + depth-2 file.
  - `TestWalker_SkipsHidden` — `.hidden.txt` and `.git/a.txt` are excluded with `IncludeHidden=false`; included with `IncludeHidden=true`.
  - `TestWalker_Gitignore` — root `.gitignore` with `vendor/` → `vendor/foo.go` not emitted; file outside `vendor/` is emitted.
  - `TestWalker_NestedGitignore` — subdir `.gitignore` scopes to subdir only (F8).
  - `TestWalker_IncludeExclude` — `Includes=["*.go"]` + tree with `.go`, `.md`, `.txt` → only `.go` emitted. `Excludes=["*_test.go"]` → test files dropped.
  - `TestWalker_ContextCancelled` — `ctx` cancelled before `Walk` is consumed; first iteration yields `ctx.Err()` and terminates.
  - `TestWalker_UnreadableEntry` — simulate a `fs.WalkDir` error via a custom `fs.FS` stub, verify the iterator yields the wrapped error and continues.
  - `TestWalker_RangeBreak` — fixture tree has **at least 3 files**. Loop breaks after the first emission: `for f, err := range w.Walk(ctx) { count++; break }`. Asserts exactly one file was yielded (count == 1), no panic occurred, and the walker terminated cleanly (F5 + F14 — iter.Seq2 yield-false semantics require the `WalkDirFunc` to return `fs.SkipAll` after a false yield; returning `nil` would panic).
  - `TestWalker_SymlinkYielded` (C4 — F7 regression guard): MapFS fixture with (a) one regular file `target.txt`, (b) one symlink `link_ok` → `target.txt` (MapFile with `Mode: fs.ModeSymlink`, `Data: []byte("target.txt")`), (c) one symlink `link_broken` → `missing.txt`. Walker yields all three entries. Asserts `fs.DirEntry.Type()&fs.ModeSymlink != 0` is true for both symlink entries. Asserts `File.Open` on `link_broken` returns an error that unwraps to `fs.ErrNotExist` via `errors.Is` (wrapped with the `open %q: %w` prefix from 3.2).
- `mage test ./internal/fileset/...` green with race detector (`-race` is on by default per `mage test`). `mage lint` green.

### Unit 3.4 — Binary file detection via Peek(512) + ErrBinaryFile

**State:** todo
**Paths:** `main/internal/fileset/binary.go`, `main/internal/fileset/binary_test.go`
**Packages:** `github.com/evanmschultz/rak/internal/fileset`
**Blocked by:** 3.2
**Acceptance:**
- `binary.go` defines:
  - `var ErrBinaryFile = errors.New("binary file")` — sentinel per CLAUDE.md § "Errors". Inspected by callers via `errors.Is(err, ErrBinaryFile)` — never string-matched (F9).
  - `func (f *File) IsBinary() (bool, error)` — calls `f.Peek(512)` and applies the heuristic: if the peek buffer contains a NUL byte (`\x00`) in the first 512 bytes, it's binary. **Match rationale**: matches git's and ripgrep's standard NUL-byte test as the single fast gate (C5 — no upstream source-file citation, only the standard-behavior reference). UTF-16 is already misdetected by git itself; rak matches git here (F10). Open the file on-demand via `Peek`; do not re-open in `IsBinary`.
  - Empty file → not binary (len(peek) == 0 → false).
  - `IsBinary` only returns errors from `Peek(512)`'s open-read-close chain (C10 — no other error paths). NUL-detection on the returned buffer is a pure string scan and cannot itself fail.
- `binary_test.go` table-driven:
  - `TestFile_IsBinary` cases: empty → false; pure ASCII "hello world" → false; UTF-8 "café" → false; buffer starting with "\x00..." → true; 512 bytes of random ASCII → false; 513 bytes ASCII followed by NUL at position 520 → false (only first 512 sniffed, F10); PNG-like "\x89PNG\r\n\x1a\n..." → true.
  - Fixtures live **inline** in the test via `fstest.MapFS` — no binary files in `testdata/` (they bloat git history and leak into snapshots; F11).
- **This unit does NOT wire `IsBinary` into the Walker.** The walker yields every file; the root command's aggregation step decides to skip binaries by default and includes them when `--binary` is passed. This keeps Walker generic and `internal/fileset` free of CLI-coupling (F12 — separation of walk policy from walk mechanics).
- `mage test ./internal/fileset/...` green. `mage lint` green.

### Unit 3.5 — cmd/rak wire-up: root.go path handling + flags + per-dir aggregation

**State:** todo
**Paths:** `main/cmd/rak/root.go`, `main/cmd/rak/root_test.go`, `main/cmd/rak/integration_test.go`, `main/cmd/rak/testdata/` (grows a small tree)
**Packages:** `github.com/evanmschultz/rak/cmd/rak`
**Blocked by:** 3.3, 3.4
**Acceptance:**
- `root.go`:
  - New flags on the root command: `--depth int` (default `0` = no limit), `--hidden bool` (default `false`), `--no-gitignore bool` (default `false`), `--binary bool` (default `false`), `--include stringSlice` (default `nil`), `--exclude stringSlice` (default `nil`). Each flag gets a short doc string matching the help-text tone of `--format`.
  - `runRoot` branches on `len(args)`:
    - `len(args) == 0` → unchanged Drop 2 path (stdin → counting → render).
    - `len(args) == 1` → **new**: construct `fileset.Walker` rooted at `args[0]` with `WalkOptions` built from the flags, iterate `w.Walk(c.Context())`, for each iteration: if `err != nil` (walker-level error from `fs.WalkDir`), **aggregate the error into the render's error summary (same pattern as `IsBinary` errors per C10) and continue** — do NOT abort the walk. Only `ctx.Err()` (context cancelled) aborts iteration; wrap as `walk: %w` and return. For each emitted `*File`: skip if `--binary=false && f.IsBinary() == true`; else `Open()`, `counting.Count()`, accumulate per-directory totals + a grand total. (Cross-reference: this preserves F6 — walker continues past per-entry errors; the caller mirrors that policy at the runRoot boundary.)
  - Per-dir aggregation: extend the render surface minimally. New type `render.Directory { Path string; Counts counting.Counts }` and a new renderer method `RenderTree(w io.Writer, dirs []Directory, total counting.Counts) error` on the `Renderer` interface. Human renderer uses laslig's KV table per dir + total; JSON renderer emits `{"directories": [...], "total": {...}}`. The exact render shape is decided here — QA proof + falsification must be able to snapshot-test it.
  - F13 pin: `--tracked-only`, `--follow`, `--max-files` flags are **NOT** added here — deferred to Drop 8. Attempting to pass them returns cobra's standard "unknown flag" error.
- `root_test.go` grows to cover:
  - `TestRootCmd_PathArg_EmptyDir` — empty dir → zero grand total, render succeeds, output contains `"total"` or equivalent label.
  - `TestRootCmd_PathArg_FlatDir` — dir with two small text files → grand total equals sum of per-file counts.
  - `TestRootCmd_PathArg_Gitignore` — dir with `.gitignore` excluding `vendor/` → files under `vendor/` don't contribute to totals.
  - `TestRootCmd_PathArg_IncludeExclude` — `--include '*.go'` + `--exclude '*_test.go'` filters correctly.
  - `TestRootCmd_PathArg_Depth` — nested tree + `--depth 1` counts only root-level files.
  - `TestRootCmd_PathArg_SkipsBinary` — tree containing a file with NUL byte → excluded by default; included when `--binary` passed. C10 contract: if `IsBinary()` returns an error, the file is **skipped (not counted)** and the error is aggregated into the render's error summary. It does NOT abort the walk. The test exercises both the clean-NUL-detected path and an induced `Peek` error (e.g. via an `fs.FS` stub whose `Open` returns `fs.ErrPermission`) and asserts the error is collected, not fatal.
  - `TestRootCmd_PathArg_Hidden` — `.hidden.txt` excluded by default; included with `--hidden`.
  - Drive tests against a fixture tree under `cmd/rak/testdata/tree/` (new). Minimal shape: `tree/a.txt`, `tree/vendor/ignored.txt`, `tree/.gitignore` (`vendor/`), `tree/sub/nested.txt`, `tree/.hidden.txt`, `tree/bin.dat` (one-byte `\x00` — stored as is; F11 allows a **deliberately tiny** binary fixture in `cmd/rak/testdata/` for the integration surface only, per CLAUDE.md § "Tests" → "two-tier testdata rule").
  - Retire / update `TestRootCmd_RejectsPathArg` — the "walker lands in Drop 3" error message is gone. Either delete the test (the rejection is no longer semantic) or pivot it into a positive path test. Builder picks deletion; QA falsification validates nothing else string-matches "Drop 3" in `cmd/rak`.
- `integration_test.go` — `TestRootCmd_Integration_PathArg_HumanFormat` and `TestRootCmd_Integration_PathArg_JSONFormat` against the new fixture tree, asserting labels + stable JSON shape.
- `mage test ./cmd/rak/...` and `mage test ./...` both green. `mage lint` green. `mage ci` green from `main/`.

## Notes

### Library choices

- **Gitignore**: `github.com/sabhiram/go-gitignore`. Zero external deps, pure Go, gitignore-spec compliant (negation `!`, dir-only `/`, `**`, char classes `[abc]`). MIT license. Widely used (golangci-lint, grype, terrascan). Alternative considered: `github.com/go-git/go-git/.../gitignore` — rejected because `go-git` pulls a heavy transitive dep chain (cryptography, storage abstractions) designed for git-repo context, not standalone pattern matching. Rolling a minimal in-package parser was considered; rejected because gitignore's spec has enough sharp edges (negation re-inclusion semantics, anchored-vs-unanchored patterns, trailing-space escaping) that we'd duplicate work `sabhiram` has already debugged. YAGNI cuts the other way here — use the maintained lib.
- **Glob**: `github.com/bmatcuk/doublestar/v4`. Drop-in superset of stdlib `filepath.Match` with `**` recursive support (stdlib `filepath.Match` rejects `**`). Zero external deps. MIT license. Used by Homebrew, GoReleaser, many CLIs. Alternative considered: `github.com/gobwas/glob` — rejected because it requires a compile step per pattern (`glob.MustCompile`) and is optimized for regex-like workloads; doublestar's per-call `Match(pattern, name)` is the simpler fit for one-shot `--include` / `--exclude` matching. `filepath.Match` alone is insufficient because users will expect `**/node_modules` and `src/**/*.go` to work.

### Cross-unit pins (F-numbered invariants)

- **F1**: `ignore.Matcher.Match(relPath, isDir) bool` returns `true` when the path **should be ignored** (convention: "returns true to drop"). Docstring must say so explicitly; this is the inverse of `fs.WalkDirFunc` returning `fs.SkipDir`, so confusion is likely without the pin. QA falsification will check the return-value semantics match the docstring everywhere.
- **F2**: `--exclude` beats `--include` on conflict. Exclude is always the strongest filter.
- **F3**: Precedence order for filtering decisions: `--exclude` → `.gitignore` → `--include`. Any path matching `--exclude` is dropped immediately. Otherwise, any path hitting `.gitignore` (when enabled) is dropped. Otherwise, if `--include` is non-empty, the path must match at least one `--include` pattern to survive.
- **F4**: `File.Peek(n int) ([]byte, error)` must open-read-close per call with no stateful cursor. Multiple `Peek` calls return identical bytes. Binary detection (3.4) and shebang sniff (Drop 4.1) both depend on this contract.
- **F5**: Walker emits `iter.Seq2[*File, error]` (decision 27(a)). Caller `for f, err := range w.Walk(ctx)`; break/return halts iteration cleanly; yield returning `false` propagates upward. No channel-based alternative.
- **F6**: Per-entry errors in the walker are **yielded, not fatal.** The iterator continues past a broken dir so one permission error doesn't abort the whole count. Caller aggregates error count in render's error summary (Drop 3.5 decides the exact render).
- **F7**: No symlink following in Drop 3. `fs.WalkDir` does not follow symlinks; we accept stdlib's default. `--follow` is **Drop 8.5**. Symlinks encountered in-tree are yielded as regular entries; `File.Open` will surface any broken-target error to the caller.
- **F8**: Nested `.gitignore` scopes to its containing directory. A pattern in `foo/.gitignore` applies to paths under `foo/` only, not to siblings of `foo/`. This is git's actual behavior and sabhiram's lib matches it when called per-dir.
- **F9**: Binary detection uses sentinel `ErrBinaryFile` (CLAUDE.md § "Errors"). Callers use `errors.Is`, never string-match. Called sites: `cmd/rak/root.go`'s aggregation loop.
- **F10**: Binary detection is a single NUL-byte test over `Peek(512)`. Matches git + ripgrep behavior. UTF-16 files are misdetected as binary by the same logic git uses; rak accepts that trade for simplicity. Revisit only if users file bugs.
- **F11**: No binary fixtures in `internal/fileset/testdata/`. Unit tests build binary content inline via `fstest.MapFS` + `[]byte{0x00, ...}` literals. `cmd/rak/testdata/tree/bin.dat` is the single exception — a deliberately tiny (1–4 bytes) fixture for the end-to-end integration test, per CLAUDE.md § "Tests" → "two-tier testdata rule".
- **F12**: `internal/fileset` is CLI-free. No cobra imports, no flag parsing, no `--binary` logic. The Walker yields every non-ignored file; the aggregation layer (cmd/rak) decides to drop binaries when the flag says so. F12 enforces the layered DAG.
- **F13**: Deferred flags not added in Drop 3: `--tracked-only` (Drop 8.4), `--follow` (Drop 8.5), `--max-files` (Drop 8.3). Cobra rejects them as unknown flags until those drops land. Unit 3.5 must not pre-register stub flags "for later" — YAGNI.
- **F14**: When `yield(...)` returns false, the `WalkDirFunc` MUST return `fs.SkipAll` so `fs.WalkDir` terminates cleanly. Returning `nil` after a false yield re-invokes yield and panics per `go doc iter`. (C1 mitigation.) The walker tracks yield-return via a captured bool in the closure; the first false-yield flips the bool and the next `WalkDirFunc` invocation short-circuits with `fs.SkipAll`. `TestWalker_RangeBreak` is the regression guard.
- **F15**: Renderer interface growth is acceptable within `internal/` scope; external implementers do not exist pre-v1.0, so adding methods to the interface is safe. (C9 — addresses the implicit-interface-satisfaction concern raised by 3.5's `RenderTree` addition. Revisit at v1.0 if a public API surface emerges.)

### Render surface growth (3.5)

Unit 3.5 extends `render.Renderer` with a new `RenderTree(w, dirs, total) error` method. This grows the interface; existing callers (Drop 2's stdin path) still use the single-input `Render(w, counts)` method, which stays. The builder must add `RenderTree` to **both** `humanRenderer` and `jsonRenderer`. Snapshot tests for the tree output live in `internal/render/render_test.go` (extend) — not in `cmd/rak/root_test.go`, which sticks to behavioral / wire-up assertions per Drop 2's precedent.

**C8 breadcrumb**: `render.Directory` is **provisional**. It migrates to the canonical `summary.Summary` type in Drop 6.1; at that point Drop 6.1 refactors both renderer implementations to consume the new type. Treat the Drop 3 shape as a minimal stand-in, not a stable contract — no external code under `internal/` should grow a dependency on `render.Directory`'s field layout beyond what 3.5 itself needs.

### Deferred items

- Parallel walk → Drop 8.1 (conditional on planner showing >500ms wall-time).
- Symlink follow → Drop 8.5.
- `--max-files` safety rail → Drop 8.3.
- `--tracked-only` via `git ls-files` → Drop 8.4.
- Language detection → Drop 4.1 (consumes `File.Peek(512)` per F4).
- Tokens per file → Drop 7.x.

### Drop-end docs updates

- **O1/O3** (from Round 1 + Round 2 plan-QA proof): `main/CLAUDE.md` § "Project Structure" → "File Breakdown" table currently lists `file.go` / `walker.go` / `walker_test.go` but does NOT list `binary.go` / `binary_test.go` (Unit 3.4 additions) nor `file_test.go` (Unit 3.2 addition). Drop 3 closeout MUST add three rows to that table — `internal/fileset/file_test.go` (`Open`, `Peek`, `IsHidden` coverage, ~150 LOC), `internal/fileset/binary.go` (`ErrBinaryFile` + `(*File).IsBinary`, ~80 LOC), `internal/fileset/binary_test.go` (table-driven coverage, ~100 LOC). This is orch-side docs work at drop-end, not a builder-unit concern.

### Open Unknowns for Phase 3 dev discussion

- **U1** — Render surface. Should 3.5 extend the `Renderer` interface (`RenderTree`) or introduce a new `TreeRenderer` interface with its own `NewHumanTreeRenderer`/`NewJSONTreeRenderer` constructors? Current plan: extend the existing interface. The alternative (separate interface) is cleaner separation but doubles the constructor surface. **Recommendation: extend**; revisit at Drop 6.1 (summary) if the surface grows unwieldy.
- **U2** — Binary detection false-positive on UTF-16. Git + ripgrep accept the same miss. Document in README or silently match upstream behavior? **Recommendation: match upstream silently in Drop 3; README note lands in Drop 9.1** when we write real docs.
- **U3** — Per-dir aggregation output shape. The per-dir rollup is sketched as `[]Directory` in 3.5 but the real summary package (Drop 6.1) will formalize `internal/summary.Summary`. Do we land a minimal local `render.Directory` struct in 3.5 and migrate to `summary.Summary` in Drop 6.1, or wait for 6.1 and render a flat grand-total only in Drop 3? **Recommendation: minimal local `render.Directory` in 3.5** so the CLI is usable end-of-Drop-3; migration to `summary.Summary` is mechanical at Drop 6.1.
- **U4** — `TestRootCmd_RejectsPathArg` (Drop 2) must be deleted or pivoted in 3.5. Any Drop 2 downstream docs / snippets that reference the "Drop 3" error message? Builder greps before deleting. **Recommendation: delete the test + update any docs that mention the transient error.**
