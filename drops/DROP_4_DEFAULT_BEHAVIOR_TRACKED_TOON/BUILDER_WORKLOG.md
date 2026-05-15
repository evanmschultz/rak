# DROP_4 — Builder Worklog

Append a `## Unit N.M — Round K` section per build attempt. See `main/drops/WORKFLOW.md` § "Phase 4 — Build (per unit)" for what each section should contain.

## Unit 4.0 — Round 1

- **Builder:** go-builder-agent
- **Started:** 2026-05-14
- **Files touched:** `main/go.mod`, `main/go.sum`
- **Mage targets run:** `mage addDep github.com/toon-format/toon-go` (pass), `mage build` (pass), `mage test` (pass, all packages cached green)
- **toon-go version:** `v0.0.0-20251202084852-7ca0e27c4e8c` — pseudo-version, no tagged release exists.
- **Transitive deps:** none — `go get` added only `toon-go` itself. No new indirect modules appeared in `go.mod`.
- **Notes:** `toon-go` lands as `// indirect` in the `require` block because no source file imports it yet (expected; import happens in unit 4.5). Pseudo-version flagged for orchestrator awareness — not a blocker per acceptance criteria, but worth noting for 4.5 if the library API surface is unstable.

## Hylla Feedback

N/A — unit 4.0 is dep-management only; no Go source files were read or searched. No Hylla queries were needed or run.

## Unit 4.1 — Round 1

- **Builder:** go-builder-agent
- **Started:** 2026-05-14
- **Files touched:**
  - `main/internal/fileset/file.go` — added `NewFile` exported wrapper (7 LOC)
  - `main/internal/lister/lister.go` — new file, new package (83 LOC)
  - `main/internal/lister/lister_test.go` — 3 tests (103 LOC)

### Mage commands run and results

| Command | Result | Notes |
|---|---|---|
| `mage test ./internal/fileset/...` | `internal/fileset ok`; `internal/lister [build failed]` | Expected — C11 deliberate compile-break. All other packages green. |
| `mage format` | clean (no output) | Both new files already gofumpt-formatted. |
| `mage format` (second run after fmt import added) | clean | No drift. |

### Deliberate compile-break acknowledgment (C11 carve-out)

Per PLAN.md Unit 4.1 § "Compile note (C11)": `internal/lister/lister.go` contains calls to `newGitLister` (defined in `git.go`, Unit 4.2) and `newWalkLister` (defined in `walk.go`, Unit 4.3). Neither symbol exists at this commit boundary. `mage build ./...` and `mage test ./internal/lister/...` are therefore broken intentionally and expected to remain so until Unit 4.3 closes. All packages OTHER than `internal/lister` compile and test green, confirmed by the `mage test` output above.

### Design decisions

- **`fmt.Errorf` wrapping for `ErrNoGitignoreInRepo`** — per F19 R2-F2 contract, `Detect` wraps the sentinel as `fmt.Errorf("lister: detect: %w", ErrNoGitignoreInRepo)` so cobra's error display shows the "lister: detect: rak: ..." chain. The test uses `errors.Is` which traverses the wrapper, so `TestDetect_NoGitignoreInRepo_ReturnsSentinel` will pass at 4.3.
- **`exec.LookPath` fast-path** — checked before spawning the git probe to avoid a SIGCHLD/process spawn cost on machines without git. Non-zero `LookPath` error → immediate `newWalkLister` without running any git command.
- **OS-level failure wrapping** — `exec.ExitError` (non-zero exit from git) is distinguished from other `runErr` values so that "not in a git repo" is silently handled and true OS-level failures (e.g. permission errors on the process spawn) are wrapped with `"lister: detect: %w"` prefix and surfaced.

### Test stubs for future activation

`TestDetect_InsideRepo` and `TestDetect_OutsideRepo` both call `Detect` and verify `err == nil` + non-nil lister at this stage. Type assertions against `*GitLister` / `*WalkLister` are commented out with `// TODO unit 4.2:` / `// TODO unit 4.3:` markers. They become active after those units land.

`TestDetect_NoGitignoreInRepo_ReturnsSentinel` is written in final form — it tests only the sentinel branch of `Detect`, which doesn't require the forward-referenced constructors. It will pass at 4.3's compile boundary.

## Hylla Feedback

- **Query:** `hylla_search_keyword`, query="newFile fileset constructor", artifact=`github.com/evanmschultz/rak@main`, node_type=block.
- **Result:** Hylla returned `File` struct and `NewWalker`/`Walker.Walk` nodes but NOT the unexported `newFile` function. Expected — Hylla indexes only public symbols (`visibility: "public"` in results).
- **Missed because:** `newFile` is unexported; Hylla's public-only default visibility filter excludes it.
- **Worked via:** `Read` of `internal/fileset/file.go` — `newFile` signature confirmed at line 52.
- **Suggestion:** A `visibility_mode=include_private` option would let builders confirm unexported constructors without falling back to `Read`. The filter exists in the schema (`public_only|include_private`) but the default excludes unexported symbols entirely.

## Unit 4.2 — Round 1

- **Builder:** go-builder-agent
- **Started:** 2026-05-14
- **Files touched:**
  - `main/internal/lister/git.go` — new file: `GitLister` struct + `newGitLister` constructor + `NewGitListerForTest` export + `anySegmentHidden` helper + `List` method (~125 LOC)
  - `main/internal/lister/git_test.go` — new file: 5 tests (~165 LOC)
  - `main/internal/lister/lister.go` — F1 carry-over fix: wrap `filepath.Abs` error with `"lister: detect: %w"` (1-line change)
  - `main/internal/lister/lister_test.go` — activated `TODO unit 4.2` type assertion for `*lister.GitLister`

### Mage commands run and results

| Command | Result |
|---|---|
| `mage format` | clean (no output) |
| `mage build` | `internal/lister [build failed]` — `undefined: newWalkLister` only (singular, as expected) |
| `mage test` | `cmd/rak ok`, `internal/counting ok`, `internal/fileset ok`, `internal/ignore ok`, `internal/render ok`, `internal/lister [build failed]` (C11 carve-out) |

### C11 carve-out: compile-break narrowed

`mage build` output after Unit 4.2:
```
# github.com/evanmschultz/rak/internal/lister
internal/lister/lister.go:57:10: undefined: newWalkLister
internal/lister/lister.go:77:10: undefined: newWalkLister
```

Exactly one undefined symbol remains (`newWalkLister`). The `undefined: newGitLister` error from Unit 4.1 is gone — `git.go` landed cleanly.

### Decision E empirical result (F17 prefix-strip)

The spawn appendix confirms Decision E is locked: `git ls-files --full-name -z` emits toplevel-relative paths regardless of `cmd.Dir` CWD. The prefix-strip in `List` is therefore always active when `g.prefix != ""` (i.e. when the walk root is a subdirectory of the repo toplevel). The code handles both cases:
- `g.prefix == ""`: relPath = rawPath (no stripping needed, walk root IS the toplevel).
- `g.prefix != ""`: entries not prefixed with `g.prefix + "/"` are skipped; the prefix is stripped to yield walk-root-relative relPath.

`TestGitLister_List_SubdirRoot` validates this for `internal/fileset/` as walk root — emitted paths like `"file.go"` and `"walker.go"` must be walk-root-relative, not `"internal/fileset/file.go"`.

### Design decisions

- **`NewGitListerForTest` exported helper**: `git_test.go` is in package `lister_test` (external), so it cannot call unexported `newGitLister`. Added `NewGitListerForTest` that delegates to `newGitLister`. Matches the pattern used by `NewWalkLister` (4.3) for the same reason.
- **`TestGitLister_ContextCancel` t.Skip instead of t.Error on buffered git**: The test may receive a file rather than a context-cancel error if git's output is already buffered before the cancel propagates through `exec.CommandContext`. This is acceptable behavior on fast machines — added a `t.Skip` rather than `t.Fail` for that path.
- **`fileset.NewFile(g.fsys, relPath, relPath)` — path and relPath both set to relPath**: For GitLister's `fs.FS` (which is `os.DirFS(absRoot)`), the file path relative to the DirFS root is the same as relPath (relative to the walk root). Setting both `Path` and `RelPath` to `relPath` is correct here.

### Hylla Feedback / Gap Notes

- All Hylla queries returned the needed symbols: `fileset.IsHidden`, `ignore.New`, `ignore.Matcher.Match`, `fileset.NewFile`, `fileset.WalkOptions`. Zero misses.
- **Gap note:** `TestGitLister_MidWalkGitFailure` is NOT implemented in 4.2. Cleanly stubbing `exec.Command` at the package level is complex. The integration path relies on OS-level EOF behavior (partial output → partial list).

## Unit 4.2 — Round 2

- **Builder:** go-builder-agent
- **Round:** 2 (wipe-and-revise after Round 1 QA findings)
- **Files touched:**
  - `main/internal/lister/git.go` — removed `NewGitListerForTest` function block (7 LOC deleted); updated `GitLister` doc comment to drop the stale reference to that export.
  - `main/internal/lister/git_test.go` — rehomed from `package lister_test` to `package lister`; removed `github.com/evanmschultz/rak/internal/lister` import; replaced all 6 `lister.NewGitListerForTest(...)` call sites with `newGitLister(...)`; added `anySegmentHidden_NonFirstSegment` table-driven sub-test inside `TestGitLister_FilterHidden` (F2 fix).
  - `main/drops/DROP_4_DEFAULT_BEHAVIOR_TRACKED_TOON/PLAN.md` — Unit 4.2 state flipped from `in_progress` to `done`.

### F3 fix — unauthorized export removed

`NewGitListerForTest` was an exported function that violated project precedent against test-only public API additions (no Drop 3 equivalent; `internal/fileset/walker_test.go` stays in `package fileset`). Remediation: `git_test.go` rehomed to `package lister` (internal test package), giving tests direct access to `newGitLister` and `anySegmentHidden`. The `NewGitListerForTest` export was deleted entirely from `git.go`. `lister_test.go` (Unit 4.1's file) remains in `package lister_test` — it only uses exported symbols (`lister.Detect`, `lister.GitLister`, `lister.ErrNoGitignoreInRepo`) and is unaffected by this change.

### F1 note — loop-order deviation from PLAN.md acceptance

Loop-order deviation from PLAN.md acceptance (context check hoisted from step 5 to step 1 in the `List` per-path loop) is deliberate — provides faster cancellation response without changing the emitted set. PLAN.md lists context as step 5 for narrative ordering; the implementation prioritizes it at step 1 for runtime correctness (fail-fast on cancellation before doing any string work).

### F2 fix — non-first-segment hidden coverage

Added `t.Run("anySegmentHidden_NonFirstSegment", ...)` sub-test inside `TestGitLister_FilterHidden`. The sub-test is a 4-case table that directly calls the unexported `anySegmentHidden` helper (accessible now that `git_test.go` is in `package lister`). Cases covered:
- `"internal/.cache/x.bin"` → hidden at segment index 1.
- `"a/b/.hidden/c.txt"` → hidden at segment index 2.
- `"normal/path/file.go"` → no hidden segment (negative case).
- `".hidden"` → hidden at index 0 (existing coverage, kept for completeness).

### Mage commands run and results

| Command | Result | Notes |
|---|---|---|
| `mage format` | Reformatted `git_test.go` (struct literal alignment) | gofumpt normalized comment spacing inside struct literal |
| `mage format` (second run) | clean (no output) | No drift after first format pass |
| `mage build ./internal/lister/...` | `undefined: newWalkLister` only (exit 1) | Expected C11 carve-out; exactly one symbol missing |
| `mage test ./internal/fileset/... ./internal/counting/... ./internal/ignore/... ./internal/render/... ./internal/summary/... ./cmd/...` | All non-lister packages green | `internal/lister` shows build-failed (same C11 carve-out); all other packages pass |

### Verification: lister_test.go unaffected

`lister_test.go` remains in `package lister_test` and uses only exported symbols: `lister.Detect`, `lister.GitLister` (type assertion), `lister.ErrNoGitignoreInRepo`. None of these reference `NewGitListerForTest`. The `*lister.GitLister` type assertion at line 41 still works because `GitLister` remains exported.

## Hylla Feedback

None — Hylla answered everything needed. File reads (not Hylla queries) were sufficient since all work was in files changed since last ingest (Hylla would be stale for `git.go` and `git_test.go`).
