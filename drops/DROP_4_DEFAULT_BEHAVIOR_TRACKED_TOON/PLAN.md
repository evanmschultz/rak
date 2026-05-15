# DROP_4 — DEFAULT_BEHAVIOR_TRACKED_TOON

**State:** building
**Tier:** A
**Blocked by:** DROP_3
**Paths (expected):** `main/go.mod`, `main/go.sum` (dep add), `main/internal/fileset/file.go` (export `NewFile` constructor), `main/internal/lister/` (new package — `FileLister` interface + `GitLister` + `WalkLister` + `Detect` factory + tests), `main/internal/render/render.go` (interface unchanged), `main/internal/render/toon.go` (new), `main/internal/render/render_test.go` (extend snapshot tests), `main/cmd/rak/root.go` (rewire `runDirectory` from direct `fileset.Walker` to `lister.Detect`; replace `--format` flag with bool `--human` / `--json` / `--toon`), `main/cmd/rak/root_test.go` (update flag-parsing cases), `main/cmd/rak/integration_test.go` (extend for tracked-only default behavior + TOON output snapshot)
**Packages (expected):** `github.com/evanmschultz/rak/internal/fileset` (NewFile export), `github.com/evanmschultz/rak/internal/lister` (new), `github.com/evanmschultz/rak/internal/render`, `github.com/evanmschultz/rak/cmd/rak`
**PLAN.md ref:** main/PLAN.md → `DROP_4_DEFAULT_BEHAVIOR_TRACKED_TOON` row
**Workflow:** main/drops/WORKFLOW.md
**Started:** 2026-05-15
**Closed:** —

## Scope

Refit rak's default behavior to its v0.1.0 product positioning: **wc++ for LLM-first consumption** (per `main/PLAN.md` decisions 30, 32, 33). Two surface changes ship in this drop. First, the **default file source** becomes git-tracked-only — when `git rev-parse --is-inside-work-tree` succeeds at the walk root, enumerate via `git ls-files --full-name -z` (NUL-delimited, paths relative to `git rev-parse --show-toplevel`); when not in a git repo, fall back to the existing `internal/fileset.Walker` + `.gitignore` filter from Drop 3 (unchanged). Second, the **default renderer** becomes TOON via `github.com/toon-format/toon-go`; the Drop 3.5 `--format auto|human|json` flag is replaced by mutually exclusive boolean flags `--human` / `--json` / `--toon`, with TOON as the default regardless of TTY (LLM audience). Drop 3's spine work (`Walker`, `ignore.Matcher`, `File`, binary detection, per-dir aggregation) is preserved — the `WalkLister` is a thin adapter over the existing `Walker`, and `humanRenderer` / `jsonRenderer` keep their Drop 2/3 contracts. Expected decomposition: 6 atomic units (4.0 dep add / 4.1 lister interface + fileset.NewFile export / 4.2 GitLister / 4.3 WalkLister adapter / 4.4 cmd/rak rewire + flag-surface reshape / 4.5 TOON renderer + snapshot tests).

Lockfiles (`go.sum`, `package-lock.json`, etc.) are counted by default per decision 34 — whatever git tracks, rak counts. No lockfile denylist in v0.1.0.

## Planner

Six atomic units. 4.0 adds the sole new external dep (`toon-go`). 4.1 defines the `FileLister` interface + `Detect` factory in a new `internal/lister` package, and exports `fileset.NewFile` so GitLister can construct `*fileset.File` values (previously unexported). 4.2 implements `GitLister` (git-backed source). 4.3 implements `WalkLister` (thin Walker adapter) with an exported `NewWalkLister` constructor. 4.2 and 4.3 are serialized (same package). 4.5 adds the TOON renderer to `internal/render` (blocked only by 4.0 — no dependency on lister). 4.4 is the integration unit that rewires `cmd/rak` to use `lister.Detect` and reshapes the `--human`/`--json`/`--toon` flags (blocked by 4.3 and 4.5 because it needs all lister implementations and the toon renderer to compile). After 4.0 + 4.1 clear: 4.2/4.3 chain and 4.5 run independently (different packages), then 4.4 closes the drop.

Import DAG extension: `lister → fileset, ignore, os/exec` (new leaf-ish mid-tier); `render` gains `toon-go` dep (leaf); `cmd/rak → lister, render, fileset, counting` (root, unchanged structurally).

### Unit 4.0 — Add toon-go dep via mage addDep

- **State:** done
- **Paths:** `main/go.mod`, `main/go.sum`
- **Packages:** — (dep-management only; no Go source files changed)
- **Acceptance:**
  - Run `mage addDep github.com/toon-format/toon-go` from `main/` — never a raw `go get`. If `mage addDep` is unavailable, that is a blocker, not a bypass path.
  - `main/go.mod` gains a `require` entry for `github.com/toon-format/toon-go` at its latest tagged version.
  - `main/go.sum` is populated for the new module and any transitive deps it pulls in. If toon-go pulls unexpected transitive deps, flag and return to orchestrator — do not proceed.
  - No Go source files change in this unit. `mage build` and `mage test` both pass clean (toon-go is unused until unit 4.5 imports it).
- **Blocked by:** —

### Unit 4.1 — internal/lister: FileLister interface + Detect factory + fileset.NewFile export

- **State:** done
- **Paths:** `main/internal/lister/lister.go`, `main/internal/lister/lister_test.go`, `main/internal/fileset/file.go`
- **Packages:** `github.com/evanmschultz/rak/internal/lister` (new), `github.com/evanmschultz/rak/internal/fileset` (NewFile export)
- **Acceptance:**
  - `main/internal/fileset/file.go` gains one new exported function (new symbol, not yet in tree):
    - `func NewFile(fsys fs.FS, path, relPath string) *File` — thin wrapper over the unexported `newFile(fsys, path, relPath)`. This export exists solely so `internal/lister.GitLister` (a separate package) can construct `*fileset.File` values. Doc comment: "NewFile constructs a File for the given path. Callers outside internal/fileset use this to create File handles when they have obtained a path from a non-Walker source (e.g. GitLister)."
  - `lister.go` defines (all new, not yet in tree):
    - `type FileLister interface { List(ctx context.Context) iter.Seq2[*fileset.File, error] }` — same iterator contract as `fileset.Walker.Walk`: per-entry errors yielded as `(nil, err)` pairs; context cancellation terminates iteration. Implementations must not panic when the caller's `yield` returns false (F14 carry-over).
    - `var ErrNoGitignoreInRepo = errors.New("lister: --no-gitignore has no effect when run inside a git repository")` — sentinel returned by `Detect` when `opts.DisableGitignore && in-git-repo`. Callers use `errors.Is(err, lister.ErrNoGitignoreInRepo)` to branch on this condition. (New symbol, not yet in tree.)
    - `func Detect(ctx context.Context, root string, opts fileset.WalkOptions) (FileLister, error)` — resolves `root` to an absolute path via `filepath.Abs` before any git command invocations. Runs `exec.CommandContext(ctx, "git", "rev-parse", "--is-inside-work-tree")` with `cmd.Dir` set to the absolute root. On exit code 0 (in git repo): if `opts.DisableGitignore` is true, return `nil, ErrNoGitignoreInRepo` immediately (Decision A — hard error before any file enumeration). Otherwise, return `newGitLister(ctx, root, opts)`. On non-zero exit (not in git repo) or git binary absent (`exec.LookPath` fails): return `newWalkLister(os.DirFS(absRoot), ".", opts)` (WalkLister silently). Unexpected OS-level command failure (not a non-zero exit from git): wrap and return the error. Wraps errors with context (`"lister: detect: %w"`). Note: `Detect` does NOT accept an `fsys` parameter — it resolves its own fs.FS via `os.DirFS(absRoot)` for the WalkLister branch.
  - `lister_test.go` tests `Detect` behavior:
    - `TestDetect_InsideRepo` — run against the actual checkout directory (`main/`) using `filepath.Abs("../..")` from `main/internal/lister/`; verify the returned lister's concrete type is `*GitLister` (type assertion). Skips with `t.Skip("git binary not found")` if `exec.LookPath("git")` fails.
    - `TestDetect_OutsideRepo` — create a `t.TempDir()` (not inside a git repo) and verify `Detect` returns a `*WalkLister`. Skips if `git` binary absent.
    - `TestDetect_NoGitignoreInRepo_ReturnsSentinel` — run `Detect` with `opts.DisableGitignore = true` inside the checkout directory; verify the returned error is `ErrNoGitignoreInRepo` via `errors.Is`. Skips if `git` binary absent.
  - Package imports for `lister.go`: `context`, `errors`, `io/fs`, `iter`, `os`, `os/exec`, `path/filepath`, plus `github.com/evanmschultz/rak/internal/fileset`. Does NOT import `github.com/toon-format/toon-go`.
  - **Compile note (C11):** `lister.go` contains `Detect` which calls `newGitLister` and `newWalkLister` — symbols defined in `git.go` (4.2) and `walk.go` (4.3). As a result, `mage build ./internal/lister/...` FAILS at the 4.1 commit boundary until 4.2 and 4.3 are in tree. This is the deliberate trade for keeping per-unit blast radius small. The following packages still build and test cleanly at 4.1: `./cmd/...` is not affected yet (4.4 adds the import); `./internal/counting/...`, `./internal/fileset/...` (gains only `NewFile`), `./internal/ignore/...`, `./internal/render/...`, `./internal/summary/...`, `./internal/tokens/...` all pass. Only `./internal/lister/...` is broken. `mage test ./internal/fileset/... ./internal/counting/... ./internal/ignore/... ./internal/render/... ./internal/summary/... ./internal/tokens/...` must pass at 4.1.
  - `mage lint` green for `internal/fileset` (the one file touched).
- **Blocked by:** 4.0

### Unit 4.2 — internal/lister.GitLister: git-backed file enumeration

- **State:** in_progress
- **Paths:** `main/internal/lister/git.go`, `main/internal/lister/git_test.go`
- **Packages:** `github.com/evanmschultz/rak/internal/lister`
- **Acceptance:**
  - `git.go` defines (all new, not yet in tree):
    - `type GitLister struct { absRoot string; toplevel string; prefix string; fsys fs.FS; opts fileset.WalkOptions }` — unexported fields. `absRoot` is the absolute walk root. `toplevel` is the output of `git rev-parse --show-toplevel` (trimmed). `prefix` is the walk-root path relative to `toplevel` (forward-slash, no trailing slash; empty when `absRoot == toplevel`). `fsys` is `os.DirFS(absRoot)` (GitLister constructs its own `fs.FS` internally — not passed by caller).
    - `func newGitLister(ctx context.Context, root string, opts fileset.WalkOptions) (*GitLister, error)` — resolves `root` to absolute via `filepath.Abs`. Runs `exec.CommandContext(ctx, "git", "rev-parse", "--show-toplevel")` with `cmd.Dir = absRoot` to get `toplevel` (trim trailing newline). Computes `prefix = filepath.ToSlash(strings.TrimPrefix(absRoot, toplevel))` then strips any leading `/` (result: `"internal/fileset"` when root is `<toplevel>/internal/fileset`; `""` when root equals toplevel). Sets `fsys = os.DirFS(absRoot)`. Returns wrapped error on any failure.
    - `func anySegmentHidden(relPath string) bool` — inline unexported helper in `git.go`. Splits `relPath` on `"/"` and calls `fileset.IsHidden(seg)` on each segment. Returns true if any segment is hidden. Used by `List` for the hidden-file filter (C4).
    - `func (g *GitLister) List(ctx context.Context) iter.Seq2[*fileset.File, error]` — runs `exec.CommandContext(ctx, "git", "ls-files", "--full-name", "-z")` with `cmd.Dir = g.absRoot` (Decision E: run from the walk root). Reads all stdout, splits on `"\x00"` (NUL byte), discards trailing empty entry from NUL-terminated output. Builds the `ignore.Matcher` **once** before the per-path loop: `matcher, err := ignore.New(nil, g.opts.Includes, g.opts.Excludes)`. If `err != nil`, yield `(nil, err)` and stop. Then for each raw path emitted by git:
      1. **Prefix handling (Decision E empirical note — see Notes § "Empirical Notes"):** The path emitted by git under `--full-name` may be toplevel-relative regardless of `cmd.Dir` CWD. If so, strip the `g.prefix + "/"` prefix to obtain `relPath` (skip entries that don't start with `g.prefix`). If empirical testing shows the CWD scoping is enough and paths are already walk-root-relative, the prefix strip is a no-op (empty prefix). Either branch produces walk-root-relative `relPath`. Forward-slash separators (already the case on all platforms for `--full-name` output; use `filepath.ToSlash` as a safety guard).
      2. **Hidden check:** if `!g.opts.IncludeHidden && anySegmentHidden(relPath)`, skip (F21).
      3. **Depth check:** if `g.opts.Depth > 0 && strings.Count(relPath, "/") >= g.opts.Depth`, skip (F18, C15 — matches Walker's `depth >= w.opts.Depth` using the same `>=` comparison).
      4. **Matcher check:** if `matcher.Match(relPath, false)`, skip. (`false` because git ls-files only emits files, never directories.)
      5. **Context check:** if `ctx.Err() != nil`, yield `(nil, ctx.Err())` and stop.
      6. **Emit:** yield `(fileset.NewFile(g.fsys, relPath, relPath), nil)`.
    - The `opts.DisableGitignore` flag is a no-op for `GitLister` — git has already applied its gitignore rules; F19 is subsumed by Decision A (the only path where `DisableGitignore` is true AND we're in a repo now errors before reaching `newGitLister`).
  - `git_test.go` — all git-shelling tests guarded with `t.Skip("git binary not found")` when `exec.LookPath("git")` fails. All use hermetic `t.TempDir() + git init` fixtures OR the actual rak checkout (named explicitly). No test relies silently on ambient repo state without a skip guard:
    - `TestGitLister_List_InRepo` — constructs `newGitLister` with `root = <absolute path to main/>` (computed via `filepath.Abs("../../..")` from `main/internal/lister/`). Iterates the list; verifies `go.mod` appears in the results with `RelPath = "go.mod"`. Skips if git absent or not in a repo.
    - `TestGitLister_List_SubdirRoot` — constructs `newGitLister` with `root = <absolute path to main/internal/fileset/>`. Verifies returned files have `RelPath` values like `"file.go"`, `"walker.go"` (walk-root-relative), and that no path from other packages appears.
    - `TestGitLister_FilterHidden` — constructs `newGitLister` with `root = <absolute main/>`, `opts.IncludeHidden = false` (default). Collects all RelPaths. Asserts that `.gitignore` (a hidden file at the repo root that git tracks) does NOT appear in results. Then re-runs with `opts.IncludeHidden = true` and asserts `.gitignore` DOES appear. This verifies Decision B: hidden filter applies in GitLister mode.
    - `TestGitLister_ContextCancel` — cancel context before iterating; verify first yield is `(nil, ctx.Err())`.
    - `TestGitLister_MidWalkGitFailure` — note: cleanly stubbing `exec.Command` at the package level is complex. Accepted gap: this path is not unit-tested in 4.2. Document the gap in `BUILDER_WORKLOG.md` § "Hylla Feedback / Gap Notes" with one sentence: "git ls-files mid-iteration exit not unit-tested; integration path relies on OS-level EOF behavior (partial output → partial list)."
    - `TestGitLister_RelPathInvariant` (R2-F1) — construct `newGitLister` with `root = <absolute main/>`, iterate `List(ctx)`, and assert for every emitted `*fileset.File`: (a) `!strings.HasPrefix(f.RelPath, "./")`, (b) `!strings.HasPrefix(f.RelPath, "/")`, (c) `f.RelPath == filepath.ToSlash(f.RelPath)` (no backslash separators). This enforces F26 at the GitLister source. Skips if git absent.
  - `mage test github.com/evanmschultz/rak/internal/lister` green (lister.go + lister_test.go + git.go + git_test.go; walk.go not yet present, but the package compiles because `newWalkLister` was forward-referenced from 4.1 — see Compile note in 4.1).
  - `mage lint` green.
- **Blocked by:** 4.1

### Unit 4.3 — internal/lister.WalkLister: Walker adapter + exported constructor

- **State:** todo
- **Paths:** `main/internal/lister/walk.go`, `main/internal/lister/walk_test.go`
- **Packages:** `github.com/evanmschultz/rak/internal/lister`
- **Acceptance:**
  - `walk.go` defines (all new, not yet in tree):
    - `type WalkLister struct { walker *fileset.Walker }` — thin shell.
    - `func newWalkLister(fsys fs.FS, root string, opts fileset.WalkOptions) *WalkLister` — unexported; constructs `fileset.NewWalker(fsys, root, opts)` and wraps it. Called by `Detect` in the non-git branch.
    - `func NewWalkLister(fsys fs.FS, root string, opts fileset.WalkOptions) *WalkLister` — exported constructor accepting an injected `fs.FS`. Allows tests in `cmd/rak` and `internal/lister` to construct a `WalkLister` with a `testing/fstest.MapFS` directly, bypassing `Detect`. Same body as `newWalkLister`. (C2)
    - `func (wl *WalkLister) List(ctx context.Context) iter.Seq2[*fileset.File, error]` — delegates to `wl.walker.Walk(ctx)`. Zero filter logic here; Walker applies all `WalkOptions` filters internally (F22 — no double-filtering).
    - Compile-time assertion: `var _ FileLister = (*WalkLister)(nil)` in `walk.go` or `walk_test.go`.
  - `walk_test.go` — table-driven with `testing/fstest.MapFS` (no real disk, no git):
    - `TestWalkLister_EmptyFS` — empty `fstest.MapFS`, no emissions.
    - `TestWalkLister_FlatFiles` — two text files at root; verify both yielded with correct `RelPath`.
    - `TestWalkLister_DepthFilter` — three files at depths 0/1/2; `WalkOptions{Depth: 1}` yields only depth-0 file.
    - `TestWalkLister_HiddenFilter` — hidden file excluded by default; included with `IncludeHidden: true`.
    - `TestWalkLister_ImplementsFileLister` — compile-time assertion: `var _ FileLister = (*WalkLister)(nil)`.
    - `TestWalkLister_RelPathInvariant` (R2-F1) — construct `NewWalkLister(fstest.MapFS{...}, ".", opts)` with a fixture that includes nested paths (e.g. `a.txt`, `sub/b.txt`, `sub/deep/c.txt`). Iterate `List(ctx)` and assert for every emitted `*fileset.File`: (a) `!strings.HasPrefix(f.RelPath, "./")`, (b) `!strings.HasPrefix(f.RelPath, "/")`, (c) `f.RelPath == filepath.ToSlash(f.RelPath)`. Enforces F26 at the WalkLister source.
  - After 4.3, `mage test github.com/evanmschultz/rak/internal/lister` green for all lister tests (lister_test.go + git_test.go + walk_test.go). The full package now compiles.
  - `mage lint` green.
- **Blocked by:** 4.2

### Unit 4.4 — cmd/rak rewire: lister.Detect + flag reshape

- **State:** todo
- **Paths:** `main/cmd/rak/root.go`, `main/cmd/rak/root_test.go`, `main/cmd/rak/integration_test.go`
- **Packages:** `github.com/evanmschultz/rak/cmd/rak`
- **Acceptance:**
  - `rootFlags` struct: remove `format string`; add `human bool`, `json bool`, `toon bool`.
  - `newRootCmd` flag registration: delete `--format` / `-f` flag; add `cmd.Flags().BoolVar` for `--human`, `--json`, `--toon`. Add `cmd.MarkFlagsMutuallyExclusive("human", "json", "toon")` so cobra enforces that at most one may be set.
  - `resolveRenderer(flags *rootFlags) render.Renderer` (replaces `selectRenderer`) — returns `render.NewTOONRenderer()` by default, `render.NewHumanRenderer()` when `flags.human`, `render.NewJSONRenderer()` when `flags.json`. No error return — cobra mutual exclusivity fires before `RunE`.
  - `runRoot`: construct renderer via `resolveRenderer(flags)`. For the `len(args)==1` path, call `lister.Detect(ctx, args[0], listerOpts(flags))` where `listerOpts` translates `rootFlags` fields (`depth`, `hidden`, `noGitignore`, `includes`, `excludes`) to `fileset.WalkOptions`. Pass the resulting `FileLister` into `runDirectory`. Surface `lister.ErrNoGitignoreInRepo` as a normal `RunE` error return — cobra prints the wrapped message. The error message format: `"rak: --no-gitignore has no effect when run inside a git repository. rak counts git-tracked files in this mode. To count untracked files, run rak outside the repository."` (Decision A — the sentinel's `Error()` carries this message; the `fmt.Errorf("lister: detect: %w", ErrNoGitignoreInRepo)` chain wraps it cleanly).
  - `runDirectory(ctx context.Context, source lister.FileLister, rootLabel string, binary bool, renderer render.Renderer) error` — accepts `FileLister` instead of `fs.FS`. Threads `binary` through to `walkAndCount`. `rootLabel` and `renderer` parameters unchanged. (P7 + C2.1 — `binary bool` slot added so the flag can traverse `runRoot` → `runDirectory` → `walkAndCount`.)
  - `walkAndCount(ctx context.Context, source lister.FileLister, binary bool) ([]render.Directory, counting.Counts, []error, error)` — accepts `FileLister` (not `fs.FS` + `flags`). Iterates via `source.List(ctx)`. Preserves per-entry error aggregation, binary-check logic, and `addCounts`/`dirKey` helpers unchanged. `fsys` parameter dropped (lister owns it). `flags.binary` threaded as a bare `bool binary`. (C14)
  - `root_test.go`:
    - Remove `TestRootCmd_ReadsStdin_RendersHumanDefault` — replace with `TestRootCmd_ReadsStdin_RendersTOONDefault` asserting default (no flags) emits TOON-shaped output (contains `"bytes:"`, `"lines:"`, `"words:"`, `"chars:"` as TOON key-value fields via `strings.Contains`).
    - Remove `TestRootCmd_FormatJSON` — replace with `TestRootCmd_FlagJSON` using `--json`.
    - Remove `TestRootCmd_InvalidFormat` — replace with:
      - `TestRootCmd_MutuallyExclusiveFlags` verifying `--human --json` together produces a cobra error.
      - `TestRootCmd_UnknownFlag` verifying `--bogus` produces a cobra error. (C13)
    - Add `TestRootCmd_NoGitignoreInRepo_Errors` — pinned form (C2.2): construct `t.TempDir()`, `exec.Command("git", "init")` inside it (skip with `t.Skip("git binary not found")` if `exec.LookPath("git")` fails), then run the cobra command via the rootcmd test harness with args `["--no-gitignore", tmpDir]`. Assert the returned error wraps `lister.ErrNoGitignoreInRepo` via `errors.Is`. This exercises the full pipeline (cobra → `runRoot` → `lister.Detect` → sentinel return → cobra error display) rather than re-testing `Detect` in isolation (which `internal/lister.TestDetect_NoGitignoreInRepo_ReturnsSentinel` already covers at 4.1).
    - Update `runTreeFS` helper (C2 + C6): construct a `lister.NewWalkLister(fsys, ".", opts)` with the injected `testing/fstest.MapFS` directly; pass to `runDirectory`. This bypasses `lister.Detect` so integration tests don't inherit the git-repo detection path. Compile-time assertions: add `_ render.Renderer = render.NewTOONRenderer()` alongside existing two.
    - All existing walk/count tests (`TestRootCmd_PathArg_*`) continue using the JSON renderer via direct constructor + `lister.NewWalkLister` with `fstest.MapFS` — no git repo required for unit tests.
  - `integration_test.go` (C6):
    - All integration tests that pass a directory path arg now go through `lister.Detect` → `GitLister` (since `cmd/rak/testdata/tree` is inside the rak git repo). Verify effective file set matches expected: git tracks `.gitignore`, `.hidden.txt`, `a.txt`, `bin.dat`, `sub/nested.txt` in `testdata/tree`. With `IncludeHidden: false` (default), `.gitignore` and `.hidden.txt` are excluded. `bin.dat` is binary-skipped. Effective set: `a.txt` + `sub/nested.txt`. If expected counts changed from Drop 3, update them.
    - `TestRootCmd_Integration_HumanFormat` — change `--format=human` to `--human`.
    - `TestRootCmd_Integration_JSONFormat` (if present) — change `--format=json` to `--json`.
    - Any test that exercised `--format=auto` — update to verify default is TOON (no flag needed).
  - `mage test github.com/evanmschultz/rak/cmd/rak` green.
  - `mage lint` green.
- **Blocked by:** 4.3, 4.5

### Unit 4.5 — internal/render.NewTOONRenderer: TOON output + snapshot tests

- **State:** todo
- **Paths:** `main/internal/render/toon.go`, `main/internal/render/render_test.go`
- **Packages:** `github.com/evanmschultz/rak/internal/render`
- **Acceptance:**
  - **Spike first (C7 + C8):** Before writing `toon.go`, builder authors a 5-line Go test program (in a temporary scratch file, NOT committed) that imports `toon-go` and empirically verifies: (a) does `toon:",omitempty"` actually drop empty/zero fields from output? (b) what does `toon-go` emit when a string value contains the configured delimiter (`|`)? Builder documents the results in `BUILDER_WORKLOG.md` § "Spike: toon-go behavior" before writing `toon.go`. If `omitempty` is unsupported, fall back to conditional struct shaping (build the errors field only when non-empty). If the lib doesn't escape embedded pipes, switch delimiter to `\t` (tab) or document the limitation. Builder's `toon.go` implementation must reflect the actual observed behavior.
  - `toon.go` defines (all new, not yet in tree):
    - `type toonRenderer struct{}` — zero-allocation value type.
    - `func NewTOONRenderer() Renderer { return toonRenderer{} }` — exported constructor, same pattern as `NewHumanRenderer` / `NewJSONRenderer`.
    - `func (t toonRenderer) Render(w io.Writer, counts counting.Counts) error` — marshals a TOON document for a single counts value. Struct shape: `toonCounts{ Bytes int; Lines int; Words int; Chars int }` with `toon` struct tags (`toon:"bytes"`, `toon:"lines"`, etc.). Marshal via `toon.Marshal(v, toon.WithDocumentDelimiter(toon.DelimiterPipe))` (pipe delimiter — F20). Write the `[]byte` result to `w`. Wrap any marshal or write error with context (`"render counts as toon: %w"`).
    - `func (t toonRenderer) RenderTree(w io.Writer, dirs []Directory, total counting.Counts, errs []error) error` — marshals a TOON document for the per-directory tree. Use a struct with a `toon:"directories"` slice field (array of per-directory structs) plus `toon:"total"` scalar block plus (when `len(errs) > 0`) `toon:"errors"` string slice. Use `toon.WithArrayDelimiter(toon.DelimiterPipe)` + `toon.WithDocumentDelimiter(toon.DelimiterPipe)` consistently (F20). Emit `errors` only when non-empty (use `toon:",omitempty"` if supported per spike; otherwise conditionally include the field). Preserve caller-supplied directory order. Wrap errors.
  - `render_test.go` — append new test cases (existing tests for `humanRenderer` and `jsonRenderer` are unchanged — no deletions):
    - `TestTOONRenderer_Render` — known `counting.Counts{Bytes: 12, Lines: 2, Words: 2, Chars: 12}` → verify output contains `"bytes: 12"`, `"lines: 2"`, `"words: 2"`, `"chars: 12"` as TOON key-value lines (`strings.Contains` style, not exact snapshot, to be robust against toon-go formatting tweaks).
    - `TestTOONRenderer_RenderTree` — two directories `[{Path: ".", Counts: {Bytes:5,...}}, {Path: "sub", Counts:{Bytes:3,...}}]` plus a total — verify output contains `"directories"` as a TOON array key and the directory paths.
    - `TestTOONRenderer_RenderTree_WithErrors` — verify that when `errs` is non-empty, the output contains an `"errors"` field.
    - `TestTOONRenderer_RenderTree_NoErrors` — verify that when `errs` is nil/empty, no `"errors"` key appears in output.
    - Compile-time assertion: add `var _ Renderer = toonRenderer{}` alongside existing compile checks in the test file.
  - `mage test github.com/evanmschultz/rak/internal/render` green.
  - `mage lint` green.
- **Blocked by:** 4.0

## Notes

### Library Choice: github.com/toon-format/toon-go

Decision 33 selected `github.com/toon-format/toon-go` as the official spec-compliant Go TOON library. Two alternative libs were considered:

- `alpkeskin/gotoon` — unofficial, sparse documentation, no struct-tag support visible in Context7.
- `sstraus/toon_go` — unofficial, low activity.

`toon-format/toon-go` is the reference implementation (Context7 source reputation: High, benchmark 82.5, 39 snippets), supports struct tags (`toon:"field"`), `toon.Marshal` / `toon.MarshalString` / `toon.NewEncoder`, array tabular output, and option functions (`WithArrayDelimiter`, `WithDocumentDelimiter`, `WithLengthMarkers`, `WithIndent`). Decision 33 is locked; no re-evaluation here.

### Cross-Unit F-Pins (F16–F26)

These invariants carry across unit boundaries. Builder must not violate them; QA falsification will attack each one.

- **F16 — Detect resolves root to absolute.** `lister.Detect` calls `filepath.Abs(root)` before any `exec.Command` invocation and sets `cmd.Dir` to the resulting absolute path. All git commands (`rev-parse` and the check inside `newGitLister`) run with `cmd.Dir` set to the absolute walk root. Never rely on process CWD for git commands.

- **F17 — GitLister prefix computation.** `newGitLister` runs `git rev-parse --show-toplevel` with `cmd.Dir = absRoot` to get `toplevel`. Computes `prefix = filepath.ToSlash(strings.TrimPrefix(absRoot, toplevel))` then strips any leading `/`. When `absRoot == toplevel`, `prefix == ""` and no filtering is needed. When `absRoot` is a subdirectory, `prefix == "internal/fileset"` (for example). `List` yields only entries whose raw git path has `prefix` as a prefix (or all entries when `prefix == ""`), then strips `prefix + "/"` (or nothing) to obtain `relPath`. This ensures `relPath` is walk-root-relative regardless of whether `--full-name` emits toplevel-relative or CWD-relative paths. **See Empirical Notes for the open verification item on Decision E.**

- **F18 — GitLister applies WalkOptions filters.** GitLister applies the same per-path filters that Walker applies internally: (a) hidden check via `anySegmentHidden(relPath)` — calls `fileset.IsHidden(seg)` on each segment of the forward-slash-split relPath; skip entire file if any segment is hidden and `!opts.IncludeHidden` (C4); (b) depth check via `strings.Count(relPath, "/") >= opts.Depth` — skip if at or beyond depth limit when `opts.Depth > 0` (zero means unlimited) (C15 — `>=` not `>`, matching Walker's actual comparison `depth >= w.opts.Depth`); (c) include/exclude check via `ignore.New(nil, opts.Includes, opts.Excludes)` built once before the loop — if construction returns an error, yield `(nil, err)` and stop; otherwise call `matcher.Match(relPath, false)` per entry (files only, not dirs). The `opts.DisableGitignore` flag is a non-reachable no-op for GitLister — Decision A ensures `Detect` returns `ErrNoGitignoreInRepo` before `newGitLister` is ever called when `DisableGitignore` is true.

- **F19 — `--no-gitignore` + in-git-repo is a hard error.** When `Detect` runs `git rev-parse --is-inside-work-tree` and gets exit 0 (in repo), it checks `opts.DisableGitignore` immediately. If true, it returns `(nil, ErrNoGitignoreInRepo)` before calling `newGitLister`. **Sentinel contract (R2-F2 upgrade):**
  - (a) **Declaration:** `var ErrNoGitignoreInRepo = errors.New("rak: --no-gitignore has no effect when run inside a git repository. rak counts git-tracked files in this mode. To count untracked files, run rak outside the repository.")` lives in `internal/lister/lister.go` (package `lister`). The full user-visible message text is baked into the sentinel.
  - (b) **Inspection:** callers MUST use `errors.Is(err, lister.ErrNoGitignoreInRepo)` to branch on this condition. Never string-match the error message.
  - (c) **Wrapping:** any wrapper uses `fmt.Errorf("...: %w", lister.ErrNoGitignoreInRepo)` to preserve the sentinel chain. `Detect` itself wraps its outer error path as `fmt.Errorf("lister: detect: %w", ErrNoGitignoreInRepo)`; `cmd/rak/runRoot` lets that flow up to cobra unchanged.
  - (d) **Decision A** is the design rationale: fail loudly rather than silently no-op.

- **F20 — Pipe delimiter for TOON output.** `toonRenderer` uses `toon.WithDocumentDelimiter(toon.DelimiterPipe)` and `toon.WithArrayDelimiter(toon.DelimiterPipe)` consistently for both `Render` and `RenderTree`. Rationale: directory paths (used as string values) may contain commas (e.g., `/Users/name,surname/project`). Default comma delimiter would corrupt such values in tabular TOON arrays. Pipe (`|`) is extremely rare in filesystem paths and is a safe alternative. Do not switch back to comma without a concrete reason. If the spike (C7+C8) reveals pipe is also unsafe, switch to `\t` (tab) and update this pin.

- **F21 — Hidden files excluded from GitLister by default (Decision B).** `git ls-files` tracks hidden files (`.gitignore`, `.golangci.yml`, etc.). GitLister applies the hidden check on each segment of `relPath` via `anySegmentHidden`. With default `IncludeHidden: false`, hidden files are excluded from the listing even if git tracks them. This is consistent with Walker's behavior and Decision B: hidden filtering applies regardless of lister source. To count hidden files, pass `--hidden`.

- **F22 — WalkLister is a pure pass-through.** `WalkLister.List(ctx)` returns `walker.Walk(ctx)` with no additional filtering. All `WalkOptions` filters are applied by the underlying `fileset.Walker`. No double-filtering may occur in `walkAndCount` for this path.

- **F23 — walkAndCount binary check unchanged.** After 4.4, `walkAndCount` accepts a `FileLister` and iterates `source.List(ctx)`. The binary-detection check (`f.IsBinary()` when `!binary`) and its error-aggregation policy (C10 from Drop 3) are preserved verbatim — only the source of `*fileset.File` values changes.

- **F24 — Mutual exclusivity enforced by cobra.** `cmd.MarkFlagsMutuallyExclusive("human", "json", "toon")` ensures cobra rejects any invocation that sets more than one of the three format flags. `resolveRenderer` does not need to handle the multi-flag case. The cobra gate fires before `RunE`.

- **F25 — Render interface unchanged.** The `Renderer` interface in `internal/render/render.go` (`Render` + `RenderTree`) and the `Directory` struct are not modified in Drop 4. `toonRenderer` implements the existing interface; the interface is not grown.

- **F26 — RelPath invariant.** `FileLister.List` emits `*fileset.File` with `RelPath` that is walk-root-relative, forward-slash separated, no leading `./` and no leading separator. Both `GitLister` and `WalkLister` honor this. `cmd/rak/root.go`'s `dirKey` + `labelDirectories` rely on this invariant.

### Hidden-file Policy

Hidden-file filtering applies in GitLister mode as well as WalkLister mode (Decision B). `.gitignore`, `.golangci.yml`, etc. are skipped by default even when the file source is `git ls-files`. The rationale: Walker-consistency wins — the everyday mental model of "hidden = skip" is preserved regardless of the underlying file source. Users who want hidden files explicitly pass `--hidden`.

This creates a soft conflict with decision 34's strict reading ("whatever git tracks, rak counts"), but Decision B explicitly resolves it in favor of consistency. The hidden filter is a presentation-layer choice (which files to count), separate from the tracking-layer source (which files to enumerate from). This decision is intentional and not to be relitigated in v0.1.0.

### Git Edge Cases (Decision D)

`git ls-files --full-name -z` output is rak's source of truth in git mode; whatever git returns is what rak counts (after apply the presentation-layer filters in F18).

- **Submodules:** appear as a single tracked entry (the submodule pointer file, not the submodule's contents — this is git's default behavior). rak counts the pointer; it does not recurse into the submodule. No special-case code in v0.1.0. A future `--recurse-submodules` flag may revisit this.
- **Sparse-checkout:** `git ls-files` returns only what is checked in under the sparse pattern. rak counts whatever git returns.
- **Worktrees:** rak's own development happens in a worktree (`main/` is a worktree of the bare repo one level up), so this configuration is validated by daily use. No special-case code needed.

### Empirical Notes: Decision E (git ls-files CWD Scoping)

**Open empirical question for Round 2 falsification:** Decision E says `newGitLister` runs `git ls-files --full-name -z` with `cmd.Dir = absRoot` (walk root as CWD). The question is: does `--full-name` with a subdirectory CWD emit only that subdir's files, and if so, are the paths still toplevel-relative or are they CWD-relative?

Empirical test from the rak repo ROOT (`main/`) confirmed: `git ls-files --full-name -z` emits toplevel-relative paths (e.g., `.github/workflows/ci.yml`, `cmd/rak/integration_test.go`). The behavior from a subdirectory CWD was not testable during planning (shell `cd` was blocked).

**Implication for F17:** The prefix-strip approach in F17 is conservative and correct for both possible behaviors:
- If `--full-name` always emits toplevel-relative paths regardless of CWD: the prefix `"internal/fileset"` is needed and the strip logic applies.
- If `--full-name` with subdirectory CWD emits CWD-relative paths (decision-E's hoped-for simplification): the prefix would be `""` and the strip is a no-op.

Either way F17's implementation handles it correctly. The Round 2 falsification agent must run `git ls-files --full-name -z` from a non-root CWD and document the actual output here before the plan is declared green.

### Sub-Directory Walk Root Semantics

When the user passes a sub-directory (e.g., `rak ./cmd/rak`), `lister.Detect` receives `root = "./cmd/rak"`. It resolves this to an absolute path and runs `git rev-parse --is-inside-work-tree` with `cmd.Dir` set to that absolute path. If inside a git repo and `DisableGitignore` is false, `newGitLister` resolves the repo toplevel, computes the prefix, and yields files with `RelPath` relative to the walk root — the same convention the Walker uses.

### Integration Test Impact on testdata/tree

The integration tests in `cmd/rak/integration_test.go` run `rak [path]` against `cmd/rak/testdata/tree`. After 4.4, this path goes through `lister.Detect`, which detects the git repo and uses `GitLister`. The `testdata/tree` fixture contains: `.gitignore`, `.hidden.txt`, `a.txt`, `bin.dat`, `sub/nested.txt`. Git tracks all of these (they are checked in). With default `IncludeHidden: false`, `.hidden.txt` and `.gitignore` are excluded (F21). `bin.dat` is binary-skipped (F23). Effective set: `a.txt` + `sub/nested.txt` — same as what the Walker returned in Drop 3. Expected counts should be unchanged. If they differ, update them.

### Open Unknowns for Phase 3 Dev Discussion

1. **TOON output format finalization** — The exact TOON shape (field names, nesting level, tabular vs nested) for `RenderTree` is open at planning time. Drop 5 adds language-aware columns (blank/comment/code splits) to the render output. Should the TOON tabular array be designed with extension columns in mind now, or emit a minimal shape and evolve it in Drop 5? Recommendation: emit minimal (bytes/lines/words/chars) now; Drop 5 adds columns. Dev to confirm that Drop 5 can add columns to the TOON array without a breaking change to downstream LLM consumers.
