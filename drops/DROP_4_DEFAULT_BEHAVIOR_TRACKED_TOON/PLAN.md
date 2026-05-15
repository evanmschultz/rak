# DROP_4 — DEFAULT_BEHAVIOR_TRACKED_TOON

**State:** planning
**Tier:** A
**Blocked by:** DROP_3
**Paths (expected):** `main/go.mod`, `main/go.sum` (dep add), `main/internal/lister/` (new package — `FileLister` interface + `GitLister` + `WalkLister` + `Detect` factory + tests), `main/internal/render/render.go` (interface unchanged), `main/internal/render/toon.go` (new), `main/internal/render/render_test.go` (extend snapshot tests), `main/cmd/rak/root.go` (rewire `runDirectory` from direct `fileset.Walker` to `lister.Detect`; replace `--format` flag with bool `--human` / `--json` / `--toon`), `main/cmd/rak/root_test.go` (update flag-parsing cases), `main/cmd/rak/integration_test.go` (extend for tracked-only default behavior + TOON output snapshot)
**Packages (expected):** `github.com/evanmschultz/rak/internal/lister` (new), `github.com/evanmschultz/rak/internal/render`, `github.com/evanmschultz/rak/cmd/rak`
**PLAN.md ref:** main/PLAN.md → `DROP_4_DEFAULT_BEHAVIOR_TRACKED_TOON` row
**Workflow:** main/drops/WORKFLOW.md
**Started:** 2026-05-15
**Closed:** —

## Scope

Refit rak's default behavior to its v0.1.0 product positioning: **wc++ for LLM-first consumption** (per `main/PLAN.md` decisions 30, 32, 33). Two surface changes ship in this drop. First, the **default file source** becomes git-tracked-only — when `git rev-parse --is-inside-work-tree` succeeds at the walk root, enumerate via `git ls-files --full-name -z` (NUL-delimited, paths relative to `git rev-parse --show-toplevel`); when not in a git repo, fall back to the existing `internal/fileset.Walker` + `.gitignore` filter from Drop 3 (unchanged). Second, the **default renderer** becomes TOON via `github.com/toon-format/toon-go`; the Drop 3.5 `--format auto|human|json` flag is replaced by mutually exclusive boolean flags `--human` / `--json` / `--toon`, with TOON as the default regardless of TTY (LLM audience). Drop 3's spine work (`Walker`, `ignore.Matcher`, `File`, binary detection, per-dir aggregation) is preserved — the `WalkLister` is a thin adapter over the existing `Walker`, and `humanRenderer` / `jsonRenderer` keep their Drop 2/3 contracts. Expected decomposition: 6 atomic units (4.0 dep add / 4.1 lister interface / 4.2 GitLister / 4.3 WalkLister adapter / 4.4 cmd/rak rewire + flag-surface reshape / 4.5 TOON renderer + snapshot tests).

Lockfiles (`go.sum`, `package-lock.json`, etc.) are counted by default per decision 34 — whatever git tracks, rak counts. No lockfile denylist in v0.1.0.

## Planner

Six atomic units. 4.0 adds the sole new external dep (`toon-go`). 4.1 defines the `FileLister` interface + `Detect` factory in a new `internal/lister` package. 4.2 implements `GitLister` (git-backed source). 4.3 implements `WalkLister` (thin Walker adapter). 4.2 and 4.3 are serialized (same package). 4.5 adds the TOON renderer to `internal/render` (blocked only by 4.0 — no dependency on lister). 4.4 is the integration unit that rewires `cmd/rak` to use `lister.Detect` and reshapes the `--human`/`--json`/`--toon` flags (blocked by 4.3 and 4.5 because it needs all lister implementations and the toon renderer to compile). After 4.0 + 4.1 clear: 4.2/4.3 chain and 4.5 run independently (different packages), then 4.4 closes the drop.

Import DAG extension: `lister → fileset, ignore, os/exec` (new leaf-ish mid-tier); `render` gains `toon-go` dep (leaf); `cmd/rak → lister, render, fileset, counting` (root, unchanged structurally).

### Unit 4.0 — Add toon-go dep via mage addDep

- **State:** todo
- **Paths:** `main/go.mod`, `main/go.sum`
- **Packages:** — (dep-management only; no Go source files changed)
- **Acceptance:**
  - Run `mage addDep github.com/toon-format/toon-go` from `main/` — never a raw `go get`. If `mage addDep` is unavailable, that is a blocker, not a bypass path.
  - `main/go.mod` gains a `require` entry for `github.com/toon-format/toon-go` at its latest tagged version.
  - `main/go.sum` is populated for the new module and any transitive deps it pulls in. If toon-go pulls unexpected transitive deps, flag and return to orchestrator — do not proceed.
  - No Go source files change in this unit. `mage build` and `mage test` both pass clean (toon-go is unused until unit 4.5 imports it).
- **Blocked by:** —

### Unit 4.1 — internal/lister: FileLister interface + Detect factory

- **State:** todo
- **Paths:** `main/internal/lister/lister.go`, `main/internal/lister/lister_test.go`
- **Packages:** `github.com/evanmschultz/rak/internal/lister` (new)
- **Acceptance:**
  - `lister.go` defines (all new, not yet in tree):
    - `type FileLister interface { List(ctx context.Context) iter.Seq2[*fileset.File, error] }` — same iterator contract as `fileset.Walker.Walk`: per-entry errors yielded as `(nil, err)` pairs; context cancellation terminates iteration. Implementations must not panic when the caller's `yield` returns false (F14 carry-over).
    - `func Detect(ctx context.Context, root string, fsys fs.FS, opts fileset.WalkOptions) (FileLister, error)` — resolves `root` to an absolute path via `filepath.Abs` before any git command invocations. Runs `exec.CommandContext(ctx, "git", "rev-parse", "--is-inside-work-tree")` with `cmd.Dir` set to the absolute root. On exit code 0, returns a `*GitLister` (built by `newGitLister`). On any other exit (not in git repo, git binary absent), returns a `*WalkLister` (built by `newWalkLister`). Wraps errors with context (`"lister: detect: %w"`).
  - `lister_test.go` tests `Detect` behavior:
    - `TestDetect_InsideRepo` — run against the actual checkout directory (`main/`) using `os.DirFS`; verify the returned lister's concrete type is `*GitLister` (type assertion). Skips with `t.Skip` if the `git` binary is absent (`exec.LookPath("git")` fails).
    - `TestDetect_OutsideRepo` — create a `t.TempDir()` (not inside a git repo) and verify `Detect` returns a `*WalkLister`. Skips if `git` binary absent.
  - Package imports: `context`, `io/fs`, `iter`, `os/exec`, `path/filepath`, plus `github.com/evanmschultz/rak/internal/fileset`. Does NOT import `github.com/toon-format/toon-go`.
  - `mage test github.com/evanmschultz/rak/internal/lister` green (note: only lister.go + lister_test.go exist at this point; git.go and walk.go land in 4.2 and 4.3 — builder must stub the unexported constructors `newGitLister` and `newWalkLister` in lister.go itself or in a separate file that this unit owns, or leave `Detect` as a forward-declared stub returning `nil, nil` with a compile guard. Preferred: author the full `Detect` body in this unit and forward-declare `newGitLister` / `newWalkLister` as package-level `var`s with nil defaults, to be replaced by real constructors in 4.2 and 4.3 — this lets `mage build` pass without 4.2/4.3. Alternatively, implement `Detect` as a thin dispatch that calls the constructors defined in 4.2 and 4.3, and accept that `mage build` of the package requires all three units to be done before the binary compiles. Either approach is acceptable; builder chooses the cleanest option.
  - `mage lint` green for the new package.
- **Blocked by:** 4.0

### Unit 4.2 — internal/lister.GitLister: git-backed file enumeration

- **State:** todo
- **Paths:** `main/internal/lister/git.go`, `main/internal/lister/git_test.go`
- **Packages:** `github.com/evanmschultz/rak/internal/lister`
- **Acceptance:**
  - `git.go` defines (all new, not yet in tree):
    - `type GitLister struct { root string; toplevel string; prefix string; fsys fs.FS; opts fileset.WalkOptions }` — unexported fields. `root` is the absolute walk root. `toplevel` is the output of `git rev-parse --show-toplevel`. `prefix` is `root` relative to `toplevel` (the path prefix that `git ls-files --full-name` paths must start with for them to be inside the user's walk root).
    - `func newGitLister(ctx context.Context, root string, fsys fs.FS, opts fileset.WalkOptions) (*GitLister, error)` — resolves `root` to absolute, runs `exec.CommandContext(ctx, "git", "rev-parse", "--show-toplevel")` with `cmd.Dir = root` to get `toplevel`, computes `prefix = strings.TrimPrefix(root, toplevel+"/")` (empty when `root == toplevel`). Returns wrapped error on any failure.
    - `func (g *GitLister) List(ctx context.Context) iter.Seq2[*fileset.File, error]` — runs `exec.CommandContext(ctx, "git", "ls-files", "--full-name", "-z")` with `cmd.Dir = g.toplevel`. Reads stdout into a buffer, splits on `"\x00"` (NUL byte), discards the trailing empty entry from NUL-terminated output. For each repo-relative path:
      1. Skips paths that do not have `g.prefix` as a prefix (or skips the prefix-check when `prefix == ""`).
      2. Strips `g.prefix + "/"` (or just uses the full path when `prefix == ""`) to produce `relPath`.
      3. Applies `WalkOptions` filters: hidden (check each segment of `relPath` via `fileset.IsHidden`), depth (count `/` in `relPath`), include/exclude (build `ignore.Matcher` once before the loop, then call `matcher.Match(relPath, false)`). The `DisableGitignore` flag is a no-op for GitLister (git already applied its gitignore rules).
      4. Constructs `newFile(g.fsys, relPath, relPath)` and yields it.
      5. On context cancellation (check `ctx.Err()` per entry), yields `(nil, ctx.Err())` once and stops.
    - The `ignore.Matcher` for include/exclude is constructed **once** outside the per-path loop via `ignore.New(nil, g.opts.Includes, g.opts.Excludes)` (no gitignore roots — GitLister is git-tracked, gitignore already applied). If construction fails, yield the error and stop.
  - `git_test.go` — all git-shelling tests are guarded with `testutil`-style skip when git is absent:
    - `TestGitLister_List_InRepo` — runs `newGitLister` against the actual `main/` checkout using `os.DirFS(".")` (which resolves to the test's working directory, which is `main/cmd/rak` for the `cmd/rak` package — but `internal/lister` tests run with CWD = `main/internal/lister`). Use `filepath.Abs("../../..")` to reach `main/` from `main/internal/lister/`. Verify the list includes at least `go.mod` (a known tracked file). Skips if `git` binary absent or not in a repo.
    - `TestGitLister_List_SubdirRoot` — runs `newGitLister` with root = `main/internal/fileset/` (absolute). Verifies returned files have RelPath values prefixed with `""` (root stripped) and that `file.go` appears. Does not contain paths from other packages.
    - `TestGitLister_FilterHidden` — uses the actual repo; verifies `.gitignore` at the root is NOT hidden-filtered (`.gitignore` is itself a hidden-named file that starts with `.`; but `--hidden=false` means hidden files are skipped — so `.gitignore` IS skipped by default). This is a real edge case: git tracks `.gitignore`, but rak's `--hidden` filter (default off) would skip it. Verify this behavior matches decision intent: `.gitignore` is hidden-skipped by default. Document in Notes.
    - `TestGitLister_ContextCancel` — cancel context before iterating; verify first yield is `(nil, ctx.Err())`.
  - `mage test github.com/evanmschultz/rak/internal/lister` green.
  - `mage lint` green.
- **Blocked by:** 4.1

### Unit 4.3 — internal/lister.WalkLister: Walker adapter

- **State:** todo
- **Paths:** `main/internal/lister/walk.go`, `main/internal/lister/walk_test.go`
- **Packages:** `github.com/evanmschultz/rak/internal/lister`
- **Acceptance:**
  - `walk.go` defines (all new, not yet in tree):
    - `type WalkLister struct { walker *fileset.Walker }` — thin shell.
    - `func newWalkLister(fsys fs.FS, root string, opts fileset.WalkOptions) *WalkLister` — constructs `fileset.NewWalker(fsys, root, opts)` and wraps it.
    - `func (wl *WalkLister) List(ctx context.Context) iter.Seq2[*fileset.File, error]` — delegates to `wl.walker.Walk(ctx)`. Zero filter logic here; Walker applies all `WalkOptions` filters internally.
  - `walk_test.go` — table-driven with `testing/fstest.MapFS` (no real disk, no git):
    - `TestWalkLister_EmptyFS` — empty `fstest.MapFS`, no emissions.
    - `TestWalkLister_FlatFiles` — two text files at root; verify both yielded with correct `RelPath`.
    - `TestWalkLister_DepthFilter` — three files at depths 0/1/2; `WalkOptions{Depth: 1}` yields only depth-0 file.
    - `TestWalkLister_HiddenFilter` — hidden file excluded by default; included with `IncludeHidden: true`.
    - `TestWalkLister_ImplementsFileLister` — compile-time assertion: `var _ FileLister = (*WalkLister)(nil)`.
  - `mage test github.com/evanmschultz/rak/internal/lister` green (all lister tests: lister_test.go + git_test.go + walk_test.go).
  - `mage lint` green.
- **Blocked by:** 4.2

### Unit 4.4 — cmd/rak rewire: lister.Detect + flag reshape

- **State:** todo
- **Paths:** `main/cmd/rak/root.go`, `main/cmd/rak/root_test.go`, `main/cmd/rak/integration_test.go`
- **Packages:** `github.com/evanmschultz/rak/cmd/rak`
- **Acceptance:**
  - `rootFlags` struct: remove `format string`; add `human bool`, `json bool`, `toon bool`.
  - `newRootCmd` flag registration: delete `--format` / `-f` flag; add `cmd.Flags().BoolVar` for `--human`, `--json`, `--toon`. Add `cmd.MarkFlagsMutuallyExclusive("human", "json", "toon")` so cobra enforces that at most one may be set.
  - `selectRenderer` is removed. Replace with `resolveRenderer(flags *rootFlags) render.Renderer` returning `render.NewTOONRenderer()` by default, `render.NewHumanRenderer()` when `flags.human`, `render.NewJSONRenderer()` when `flags.json`. No error return — mutual exclusivity is enforced by cobra before `RunE` fires.
  - `runRoot`: construct renderer via `resolveRenderer(flags)`. For the `len(args)==1` path, call `lister.Detect(ctx, args[0], os.DirFS(args[0]), listerOpts(flags))` where `listerOpts` translates `rootFlags` fields (`depth`, `hidden`, `noGitignore`, `includes`, `excludes`) to `fileset.WalkOptions`. Pass the resulting `FileLister` into `runDirectory`.
  - `runDirectory` signature: accept `FileLister` instead of `fs.FS` and `flags *rootFlags`; pass the lister to `walkAndCount`. `rootLabel` and `renderer` parameters unchanged.
  - `walkAndCount` signature: accept `FileLister` (not `fs.FS` + `flags`). Iterate via `lister.List(ctx)`. Preserve the per-entry error aggregation, binary-check logic (`flags.binary`), and `addCounts` / `dirKey` helpers unchanged. The `fsys` parameter is dropped (lister owns the fs.FS now). The `flags.binary` check is preserved — `walkAndCount` still receives enough information to apply the binary-skip policy; thread `binary bool` as a parameter.
  - `root_test.go`:
    - Remove `TestRootCmd_ReadsStdin_RendersHumanDefault` — replace with `TestRootCmd_ReadsStdin_RendersTOONDefault` asserting that the default (no flags) emits TOON-shaped output (contains `bytes:`, `lines:`, `words:`, `chars:` as TOON key-value fields).
    - Remove `TestRootCmd_FormatJSON` — replace with `TestRootCmd_FlagJSON` using `--json`.
    - Remove `TestRootCmd_InvalidFormat` — replace with `TestRootCmd_MutuallyExclusiveFlags` verifying that `--human --json` together produces an error from cobra.
    - Update `runTreeFS` helper: remove `selectRenderer("json")` call; instead construct `render.NewJSONRenderer()` directly. Update `runDirectory` call signature. All existing walk/count tests (`TestRootCmd_PathArg_*`) can continue using the JSON renderer via the direct constructor — they do not need a git repo because they pass `fstest.MapFS` stubs. The lister for those tests: construct a `lister.WalkLister` via `lister.NewWalkLister` (or a package-internal helper) with the injected `fstest.MapFS`. Alternatively, keep `runTreeFS` calling `runDirectory` with a stub that bypasses `lister.Detect` entirely — pass a pre-built `WalkLister` directly. Builder decides cleanest form.
    - Compile-time assertion: add `_ render.Renderer = render.NewTOONRenderer()` alongside existing two.
  - `integration_test.go`:
    - `TestRootCmd_Integration_HumanFormat` — change `--format=human` to `--human`.
    - `TestRootCmd_Integration_JSONFormat` (if present) — change `--format=json` to `--json`.
    - Any test that exercised `--format=auto` — update to verify default is TOON (no flag needed).
    - All integration tests that use a directory path arg now go through `lister.Detect`, which will return a `GitLister` for the `cmd/rak/testdata/tree` path (since it is inside the git repo). Verify integration tests still produce correct counts — `git ls-files` should enumerate the same files that the walker enumerated (minus `.hidden.txt` which is hidden-filtered; lockfiles like `go.sum` are NOT in `testdata/tree`). This is a correctness check; if `git ls-files` enumerates unexpected files in the fixture, update the expected counts.
  - `mage test github.com/evanmschultz/rak/cmd/rak` green.
  - `mage lint` green.
- **Blocked by:** 4.3, 4.5

### Unit 4.5 — internal/render.NewTOONRenderer: TOON output + snapshot tests

- **State:** todo
- **Paths:** `main/internal/render/toon.go`, `main/internal/render/render_test.go`
- **Packages:** `github.com/evanmschultz/rak/internal/render`
- **Acceptance:**
  - `toon.go` defines (all new, not yet in tree):
    - `type toonRenderer struct{}` — zero-allocation value type.
    - `func NewTOONRenderer() Renderer { return toonRenderer{} }` — exported constructor, same pattern as `NewHumanRenderer` / `NewJSONRenderer`.
    - `func (t toonRenderer) Render(w io.Writer, counts counting.Counts) error` — marshals a TOON document for a single counts value. Struct shape: `toonCounts{ Bytes int; Lines int; Words int; Chars int }` with `toon` struct tags (`toon:"bytes"`, `toon:"lines"`, etc.). Marshal via `toon.Marshal(v, toon.WithDocumentDelimiter(toon.DelimiterPipe))` (pipe delimiter for safety with path-bearing fields in RenderTree). Write the `[]byte` result to `w`. Wrap any marshal or write error with context (`"render counts as toon: %w"`).
    - `func (t toonRenderer) RenderTree(w io.Writer, dirs []Directory, total counting.Counts, errs []error) error` — marshals a TOON document for the per-directory tree. Use a struct with a `toon:"directories"` slice field (array of per-directory structs) plus `toon:"total"` scalar block plus (when `len(errs) > 0`) `toon:"errors"` string slice. Use `toon.WithArrayDelimiter(toon.DelimiterPipe)` + `toon.WithDocumentDelimiter(toon.DelimiterPipe)` consistently (F-pin: pipe delimiter throughout to avoid comma conflicts with directory paths). Emit `errors` only when non-empty (use `omitempty` equivalent — if toon-go supports `toon:",omitempty"` tag, use it; otherwise conditionally include the field). Preserve caller-supplied directory order. Wrap errors.
  - `render_test.go` — append new test cases (existing tests for `humanRenderer` and `jsonRenderer` are unchanged — no deletions):
    - `TestTOONRenderer_Render` — one subcase: known `counting.Counts{Bytes: 12, Lines: 2, Words: 2, Chars: 12}` → verify output contains `bytes: 12`, `lines: 2`, `words: 2`, `chars: 2` as TOON key-value lines (strings.Contains style, not exact snapshot, to be robust against toon-go formatting tweaks).
    - `TestTOONRenderer_RenderTree` — one subcase: two directories `[{Path: ".", Counts: {Bytes:5,...}}, {Path: "sub", Counts:{Bytes:3,...}}]` plus a total — verify the output contains `directories` as a TOON array key and the directory paths.
    - `TestTOONRenderer_RenderTree_WithErrors` — verify that when `errs` is non-empty, the output contains an `errors` field.
    - `TestTOONRenderer_RenderTree_NoErrors` — verify that when `errs` is nil/empty, no `errors` key appears in output.
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

### Cross-Unit F-Pins (F16–F24)

These invariants carry across unit boundaries. Builder must not violate them; QA falsification will attack each one.

- **F16 — Detect resolves root to absolute.** `lister.Detect` calls `filepath.Abs(root)` before any `exec.Command` invocation and sets `cmd.Dir` to the resulting absolute path. All git commands (both `rev-parse` and `ls-files`) run with `cmd.Dir` set to the absolute walk root (for `rev-parse`) or the absolute repo toplevel (for `ls-files`). Never rely on process CWD for git commands.

- **F17 — GitLister prefix computation.** When the user's walk root equals the repo toplevel, `prefix == ""` and no path filtering is needed (all `git ls-files` output is in scope). When `root` is a subdirectory, `prefix = filepath.ToSlash(strings.TrimPrefix(absRoot, toplevel+string(filepath.Separator)))`. GitLister yields only files whose repo-relative path has `prefix` as a prefix (or all files when `prefix == ""`). Strip `prefix + "/"` (or `prefix + string(os.PathSeparator)`) to obtain `relPath` for the `fileset.File`.

- **F18 — GitLister applies WalkOptions filters.** GitLister applies the same per-path filters that Walker applies internally: (a) hidden check via `fileset.IsHidden` on each path segment of `relPath` — skip entire file if any segment is hidden and `!opts.IncludeHidden`; (b) depth check via `strings.Count(relPath, "/")` — skip if depth exceeds `opts.Depth` (0 = no limit); (c) include/exclude check via `ignore.Matcher` built once from `ignore.New(nil, opts.Includes, opts.Excludes)`. The `opts.DisableGitignore` flag is a no-op for GitLister (F19).

- **F19 — --no-gitignore is a no-op for GitLister.** `git ls-files` only lists tracked files, which are already gitignore-filtered by git. The `DisableGitignore` field in `WalkOptions` has no effect on `GitLister.List`. This semantic difference is intentional and documented here; no user-visible behavior change is needed in v0.1.0.

- **F20 — Pipe delimiter for TOON output.** `toonRenderer` uses `toon.WithDocumentDelimiter(toon.DelimiterPipe)` and `toon.WithArrayDelimiter(toon.DelimiterPipe)` consistently for both `Render` and `RenderTree`. Rationale: directory paths (used as string values) may contain commas (e.g., `/Users/name,surname/project`). Default comma delimiter would corrupt such values in tabular TOON arrays. Pipe (`|`) is extremely rare in filesystem paths and is a safe alternative. Do not switch back to comma without a concrete reason.

- **F21 — Hidden files excluded from GitLister by default.** `git ls-files` tracks hidden files (`.gitignore`, `.golangci.yml`, etc.) — they appear in its output. GitLister applies the hidden check on each path segment. With default `IncludeHidden: false`, hidden files are excluded from the listing even if git tracks them. This means `rak .` in a git repo will not count `.gitignore` by default. This is consistent with the Walker's behavior. To count hidden files, pass `--hidden`.

- **F22 — WalkLister is a pure pass-through.** `WalkLister.List(ctx)` returns `walker.Walk(ctx)` with no additional filtering. All `WalkOptions` filters are applied by the underlying `fileset.Walker`. No double-filtering may occur in `walkAndCount` for this path.

- **F23 — walkAndCount binary check unchanged.** After 4.4, `walkAndCount` accepts a `FileLister` and iterates `lister.List(ctx)`. The binary-detection check (`f.IsBinary()` when `!binary`) and its error-aggregation policy (C10 from Drop 3) are preserved verbatim — only the source of `*fileset.File` values changes.

- **F24 — Mutual exclusivity enforced by cobra.** `cmd.MarkFlagsMutuallyExclusive("human", "json", "toon")` ensures cobra rejects any invocation that sets more than one of the three format flags. `resolveRenderer` does not need to handle the multi-flag case; it may panic or return the first match. The cobra gate fires before `RunE`.

- **F25 — Render interface unchanged.** The `Renderer` interface in `internal/render/render.go` (`Render` + `RenderTree`) and the `Directory` struct are not modified in Drop 4. `toonRenderer` implements the existing interface; the interface is not grown.

### Sub-Directory Walk Root Semantics

When the user passes a sub-directory (e.g., `rak ./cmd/rak`), `lister.Detect` receives `root = "./cmd/rak"`. It resolves this to an absolute path and runs `git rev-parse --is-inside-work-tree` with `cmd.Dir` set to that absolute path. If inside a git repo, `newGitLister` also resolves the repo toplevel and computes the prefix. `git ls-files --full-name -z` is run from the toplevel and results are filtered to the prefix. The resulting `*fileset.File` values have `RelPath` relative to the walk root — the same convention the Walker uses.

### --no-gitignore Flag Open Issue

When in a git repo, `--no-gitignore` currently has no effect (F19). A user who wants to count untracked files cannot do so in v0.1.0 — there is no `--include-untracked` flag (decision 32 deferred). This is a known limitation. Flag in `## Open Unknowns for Phase 3` below.

### Integration Test Impact on testdata/tree

The integration tests in `cmd/rak/integration_test.go` run `rak [path]` against `cmd/rak/testdata/tree`. After 4.4, this path goes through `lister.Detect`, which will detect the git repo and use `GitLister`. The `testdata/tree` fixture contains: `.gitignore`, `.hidden.txt`, `a.txt`, `bin.dat`, `sub/nested.txt`. Git tracks all of these (they are checked in). With default `IncludeHidden: false`, `.hidden.txt` and `.gitignore` are excluded. `bin.dat` is binary-skipped (by default). So the effective set matches what the Walker returned in Drop 3: `a.txt` + `sub/nested.txt`. If integration test expected counts change, update them.

### Open Unknowns for Phase 3 Dev Discussion

1. **`--no-gitignore` + GitLister semantic gap** — In v0.1.0, `--no-gitignore` has no effect in git repos (F19). Should rak document this in `--help` output, or silently ignore the flag? If the flag docs don't mention the limitation, users will be confused. Recommendation: add a `(no effect in git repos)` note to the `--no-gitignore` flag description in `newRootCmd`. Dev to confirm.

2. **TOON output format finalization** — The exact TOON shape (field names, nesting level, tabular vs nested) for `RenderTree` is open at planning time. Drop 5 adds language-aware columns (blank/comment/code splits) to the render output. Should the TOON tabular array be designed with extension columns in mind now, or emit a minimal shape and evolve it in Drop 5? Recommendation: emit minimal (bytes/lines/words/chars) now; Drop 5 adds columns. Dev to confirm that Drop 5 can add columns to the TOON array without a breaking change to downstream LLM consumers.

3. **Hidden-file counting semantic in git repos** — `git ls-files` returns `.gitignore`, `.golangci.yml`, etc. (all hidden files git tracks). With `--hidden=false` (default), rak skips these. Is this the desired behavior? Alternative: in git-tracked mode, count all tracked files regardless of hidden status (since the user explicitly asked for "what git tracks"). Recommendation: keep current behavior (hidden filter applies regardless of lister source) for consistency. Dev to confirm.
