# DROP_4 — Builder QA Proof

Append a `## Unit N.M — Round K` section per build-QA proof pass. See `main/drops/WORKFLOW.md` § "Phase 5 — Build-QA (per unit)" for the section contract.

## Unit 4.0 — Round 1

**Verdict:** pass-with-findings

### Acceptance audit

- "mage addDep used (not raw go get)" — **pass**. `BUILDER_WORKLOG.md` § "Unit 4.0 — Round 1" line 10 records `mage addDep github.com/toon-format/toon-go` as the invocation; `mage -l` confirms `addDep` is the canonical target wrapping `go get`. No raw `go get` trace.
- "go.mod has require entry" — **pass**. `go.mod` line 42: `github.com/toon-format/toon-go v0.0.0-20251202084852-7ca0e27c4e8c // indirect` inside the second `require` block. `// indirect` is the expected placement at this stage because no Go source file imports it yet (import lands in Unit 4.5).
- "go.sum populated" — **pass**. `git diff HEAD~1 -- go.sum` shows exactly two added lines for the new module: `h1:D8lDFovBMZywze1eh9iwMLcYor5f11mHBocLhO7cBe8=` (module zip) and `/go.mod h1:j/BOnpF2ihnz4lELs99h9mwGJBx/zdleOUCnLLRPCsc=` (go.mod). Both standard required entries.
- "no unexpected transitive deps" — **pass**. `go mod graph` shows toon-go's only outgoing edge is `go@1.23` (Go language version, not a module). Zero transitive module deps. Confirmed by `git diff HEAD~1 -- go.mod`: exactly one `+` line, no other indirect deltas. Matches builder's claim.
- "no Go source files changed" — **pass**. `git show HEAD --stat` lists only `BUILDER_WORKLOG.md`, `PLAN.md`, `go.mod`, `go.sum`. No `.go` files.
- "mage build passes" — **pass**. Re-ran `mage build` from `main/`: exit 0, no output (clean build).
- "mage test passes" — **pass**. Re-ran `mage test` from `main/`: all five packages report `ok ... (cached)` (`cmd/rak`, `internal/counting`, `internal/fileset`, `internal/ignore`, `internal/render`). Cache is sound because no Go source changed; `-race` is enabled per mage target definition.

### Findings

- **F1 (low) — "latest tagged version" wording does not apply to an untagged module.** PLAN.md line 32 reads: *"`main/go.mod` gains a `require` entry for `github.com/toon-format/toon-go` at its latest tagged version."* Verified upstream via `gh api repos/toon-format/toon-go/tags` → `[]` (zero tags exist on the repo). The pseudo-version `v0.0.0-20251202084852-7ca0e27c4e8c` is the correct resolution for a tagless module per Go modules semantics — `go get` cannot resolve to a tag that does not exist. Builder's flag in `BUILDER_WORKLOG.md` line 11 is accurate. Action for orchestrator: optionally tighten the PLAN.md wording (e.g. "at its latest tagged release, or pseudo-version if no tag exists") in any future dep-add unit; no remediation needed in Unit 4.0 itself. This is informational, not a build defect.

- **F2 (low) — Builder worklog notes pseudo-version risk for Unit 4.5.** `BUILDER_WORKLOG.md` line 13 flags: *"Pseudo-version flagged for orchestrator awareness — not a blocker per acceptance criteria, but worth noting for 4.5 if the library API surface is unstable."* This is good hygiene — surfacing for orchestrator visibility when Unit 4.5 starts. No action required at Unit 4.0 closure.

### Evidence summary

- `git show HEAD --stat`: 4 files changed (`BUILDER_WORKLOG.md`, `PLAN.md`, `go.mod`, `go.sum`); no `.go` files.
- `git diff HEAD~1 -- go.mod go.sum`: +1 line in go.mod (`// indirect` block), +2 lines in go.sum (h1: + /go.mod h1:).
- `go mod graph | line 224`: `github.com/toon-format/toon-go@v0.0.0-... go@1.23` — only `go@1.23` edge, no module deps.
- `gh api repos/toon-format/toon-go/tags` → `[]` — confirms no tagged release exists upstream.
- `mage build` → exit 0, clean.
- `mage test` → all 5 packages `ok (cached)`.
- `mage -l` → `addDep` target is canonical.

## Unit 4.1 — Round 1

**Verdict:** pass

### Acceptance audit

- **"fileset.NewFile thin wrapper"** — **pass**. `internal/fileset/file.go` line 63 declares `func NewFile(fsys fs.FS, path, relPath string) *File` with body line 64 `return newFile(fsys, path, relPath)` — literally a thin wrapper. Signature matches the unexported `newFile` (line 52) exactly. `git diff HEAD~1 -- internal/fileset/file.go` shows +7 lines total (3 doc lines + `func NewFile(...) *File {` + `return ...` + `}` + blank line). No other changes in `file.go`. Doc comment per project rule 11: starts with `NewFile`, explains why exported (cross-package construction for `GitLister`).
- **"FileLister interface"** — **pass**. `internal/lister/lister.go` lines 25-27 define `type FileLister interface { List(ctx context.Context) iter.Seq2[*fileset.File, error] }` — single method, satisfies rule 5 (`-er` naming). Doc comment lines 20-24 starts with `FileLister`, documents the iterator contract (per-entry errors as `(nil, err)`, context cancellation, no panic on `yield`-returns-false — F14 carry-over).
- **"ErrNoGitignoreInRepo sentinel with full message"** — **pass**. `internal/lister/lister.go` line 35 declares `var ErrNoGitignoreInRepo = errors.New("rak: --no-gitignore has no effect when run inside a git repository. rak counts git-tracked files in this mode. To count untracked files, run rak outside the repository.")` — the full R2-F2 wording is baked into the sentinel. The retired "or wait for v0.2 --include-untracked flag" wording is absent (verified via direct read of line 35). Doc comment lines 29-34 present, starts with `ErrNoGitignoreInRepo`, mandates `errors.Is` inspection (no string-matching).
- **"Detect factory algorithm"** — **pass**. `internal/lister/lister.go` lines 48-83. Step-by-step trace against PLAN.md F16-F19:
  - Line 49: `absRoot, err := filepath.Abs(root)` — first action, F16 satisfied.
  - Lines 56-58: `exec.LookPath("git")` fast-path → `newWalkLister(os.DirFS(absRoot), ".", opts)` when git is absent. Documented as an optimization in `BUILDER_WORKLOG.md` line 43. Functionally equivalent to PLAN.md's described "git binary absent" branch.
  - Lines 60-61: `exec.CommandContext(ctx, "git", "rev-parse", "--is-inside-work-tree")` with `cmd.Dir = absRoot`. Matches F16.
  - Lines 64-70: `runErr == nil` (exit 0) branch — checks `opts.DisableGitignore`. True → `return nil, fmt.Errorf("lister: detect: %w", ErrNoGitignoreInRepo)` (line 67, F19c wrap preserved). False → `return newGitLister(ctx, absRoot, opts)` (line 69, forward ref to Unit 4.2).
  - Lines 74-78: `errors.As(runErr, &exitErr)` distinguishes non-zero git exit from OS-level failure. Non-zero → `newWalkLister(os.DirFS(absRoot), ".", opts)` (line 77).
  - Line 82: OS-level failure → `fmt.Errorf("lister: detect: %w", runErr)`. All paths covered.
- **"Sentinel test exercises full sentinel-return path"** — **pass**. `internal/lister/lister_test.go` lines 84-102. `TestDetect_NoGitignoreInRepo_ReturnsSentinel` skips on missing git (lines 85-87), resolves to `main/` via `filepath.Abs("../../..")`, calls `Detect(ctx, absRoot, fileset.WalkOptions{DisableGitignore: true})`. Asserts `got != nil` is false (line 96-98) AND `errors.Is(err, lister.ErrNoGitignoreInRepo)` is true (line 99-101). This exercises the full chain: `filepath.Abs` → `LookPath` → `rev-parse` (exit 0) → `DisableGitignore` branch → wrapped sentinel return. No dependency on `newGitLister` or `newWalkLister` (sentinel returns before either constructor runs). Per PLAN.md C11, the test cannot RUN until the package compiles at 4.3 — but it is written in final form and will pass at that boundary.
- **"Stub tests have TODO markers for 4.2/4.3"** — **pass**. `TestDetect_InsideRepo` (lines 19-46): type assertion against `*lister.GitLister` commented out at lines 41-44 with `// TODO unit 4.2: uncomment after GitLister lands in git.go.` marker; `_ = got` keeps the test inert but compiling. `TestDetect_OutsideRepo` (lines 48-77): same shape, type assertion commented out at lines 72-75 with `// TODO unit 4.3: uncomment after WalkLister lands in walk.go.` marker; `_ = got` mirror. Both tests skip on missing git binary (lines 21-23, 54-56). Both will activate by uncommenting four lines each at their respective unit boundaries.
- **"Mage scoped subset passes"** — **pass**. Re-ran `mage test ./internal/fileset/... ./internal/counting/... ./internal/ignore/... ./internal/render/... ./internal/summary/... ./cmd/...` from `main/`. Mage's `test` target invokes `go test -race ./...` which expands to the full module, so the output enumerates every package; the relevant subset reports: `ok github.com/evanmschultz/rak/cmd/rak (cached)`, `ok internal/counting (cached)`, `ok internal/fileset (cached)`, `ok internal/ignore (cached)`, `ok internal/render (cached)`. (Packages `internal/summary` and `internal/tokens` are forward-looking in PLAN.md's project map and do not exist yet; their absence is correct for the current tree.) Only `internal/lister` fails, which is the expected C11 carve-out.
- **"Compile-break failure mode matches expected"** — **pass**. Re-ran `mage build ./internal/lister/...`: exit 1 with exactly three compile errors, all the expected forward-reference kind:
  - `internal/lister/lister.go:57:10: undefined: newWalkLister`
  - `internal/lister/lister.go:69:10: undefined: newGitLister`
  - `internal/lister/lister.go:77:10: undefined: newWalkLister`
  No other compile errors. No type-mismatch errors. No import-cycle errors. Failure mode is exactly the deliberate C11 trade — symbols defined in 4.2 (`newGitLister`) and 4.3 (`newWalkLister`) referenced from 4.1's `Detect`.
- **"Doc comments on all exports"** — **pass**. Verified by direct read:
  - `NewFile` — `internal/fileset/file.go` lines 60-62, starts with `NewFile`.
  - `FileLister` — `internal/lister/lister.go` lines 20-24, starts with `FileLister`.
  - `ErrNoGitignoreInRepo` — `internal/lister/lister.go` lines 29-34, starts with `ErrNoGitignoreInRepo`.
  - `Detect` — `internal/lister/lister.go` lines 37-47, starts with `Detect`.
  All four obey rule 11.

### Findings

- **F1 (informational, not a defect) — `t.Helper()` in `TestDetect_InsideRepo` is a no-op.** `internal/lister/lister_test.go` line 20 calls `t.Helper()` at the top of the test function. `Helper()` marks the calling function as a test helper so failure-reporting walks past it to the caller's line — meaningful only inside subroutines invoked by tests, not at top-level test entry. Cosmetic; does not affect correctness. No action required, but the line can be deleted in a future cleanup.

- **F2 (informational, not a defect) — LookPath ordering deviates slightly from PLAN.md's described sequence.** PLAN.md unit 4.1 acceptance line 48 describes `Detect` as: probe with `rev-parse`, then on non-zero exit OR `exec.LookPath` failure → `newWalkLister`. The implementation places `exec.LookPath("git")` as a fast-path BEFORE the `rev-parse` probe (lines 56-58 in `lister.go`). Functionally identical — both orderings produce `newWalkLister(os.DirFS(absRoot), ".", opts)` when git is absent. The builder documented this in `BUILDER_WORKLOG.md` line 43 as "avoid SIGCHLD/process spawn cost on machines without git". Acceptable optimization. No action required.

### Evidence summary

- `git show HEAD --stat`: 5 files changed (BUILDER_WORKLOG.md, PLAN.md, internal/fileset/file.go +7, internal/lister/lister.go +83, internal/lister/lister_test.go +102). Matches builder claim.
- `git diff HEAD~1 -- go.mod go.sum`: empty output — confirms zero dep changes at 4.1, as required.
- `git diff HEAD~1 -- internal/fileset/file.go`: shows only the `NewFile` addition; no other deltas in file.go.
- `mage test ./...` (via mage's wrapper of `go test -race ./...`): all packages except `internal/lister` report `ok ... (cached)`; only `internal/lister` fails with the expected three `undefined:` errors.
- `mage build ./internal/lister/...`: fails with exactly three `undefined: newWalkLister` / `undefined: newGitLister` errors — no other compile issues.

## Hylla Feedback

N/A — this QA pass touched only Go source files that were freshly added or modified at HEAD. Hylla's `@main` ingest is older than HEAD (built at end of Drop 3), so the 4.1 deltas live in `git diff` territory. All source-of-truth lookups went through `Read` of the live tree; no Hylla queries were needed or attempted. (When Hylla reingest fires at end of Drop 4, the 4.1 symbols will be queryable and the same audit could be run via Hylla without `Read` fallbacks.)
