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

## Unit 4.2 — Round 1

**Verdict:** pass

### Acceptance audit

- **"`git.go` algorithm matches PLAN.md F16/F17/F18/F19/F21"** — **pass**. Step-by-step trace:
  - `newGitLister` (`git.go` lines 43-70): line 44 `absRoot, err := filepath.Abs(root)` (F16-defensive); line 49 `cmd := exec.CommandContext(ctx, "git", "rev-parse", "--show-toplevel")` with `cmd.Dir = absRoot` (line 50, F16); line 55 `toplevel := strings.TrimRight(string(out), "\n\r")` (trims newline/CR); line 60 `prefix := filepath.ToSlash(strings.TrimPrefix(absRoot, toplevel))`; line 61 `prefix = strings.TrimPrefix(prefix, "/")` (leading-slash strip per F17). All four F17 steps present.
  - `anySegmentHidden` (`git.go` lines 84-91): splits `relPath` on `"/"` and calls `fileset.IsHidden(seg)` per segment. Verified `fileset.IsHidden` exists (`internal/fileset/file.go` line 124, signature `(name string) bool`). Matches F18(a) / C4.
  - `List` (`git.go` lines 106-184): line 109 `exec.CommandContext(ctx, "git", "ls-files", "--full-name", "-z")` with `cmd.Dir = g.absRoot` (line 110, F16); line 123 `matcher, err := ignore.New(nil, g.opts.Includes, g.opts.Excludes)` — built ONCE before the per-path loop (F18-precondition). Verified `ignore.New(roots []GitignoreRoot, includes, excludes []string) (Matcher, error)` signature at `internal/ignore/ignore.go` line 74. Per-entry loop (lines 136-182): context check (line 138-141) → prefix strip (lines 148-157, F17) → `filepath.ToSlash` (line 160) → hidden check `anySegmentHidden` (line 163, F21) → depth check `strings.Count(relPath, "/") >= g.opts.Depth` guarded on `g.opts.Depth > 0` (line 169, F18(b)/C15) → matcher check `matcher.Match(relPath, false)` with `false` for files-only (line 174, F18(c)) → emit `fileset.NewFile(g.fsys, relPath, relPath)` honouring F14 yield-false guard (line 179).
  - Loop order deviates from PLAN.md (context check is FIRST, not fifth). Functionally identical and strictly better for cancel responsiveness — no work done on a cancelled iteration. Acceptable.
  - F19 is enforced upstream in `Detect` (already audited at Unit 4.1); `newGitLister` is never reached when `DisableGitignore && in-repo`. F19(c) wrap remains `fmt.Errorf("lister: detect: %w", ErrNoGitignoreInRepo)` (line 67 of `lister.go`). The struct doc comment at `git.go` lines 18-27 correctly notes this branch is unreachable.

- **"Walker depth comparison parity (C15)"** — **pass**. Walker's depth-prune (`internal/fileset/walker.go` lines 223, 226) uses `depth >= w.opts.Depth`. GitLister's depth-prune (`git.go` line 169) uses `strings.Count(relPath, "/") >= g.opts.Depth`. Both use `>=`. The `g.opts.Depth > 0` guard mirrors Walker's `w.opts.Depth != 0` check (lines 216 of walker.go) — zero means unlimited in both.

- **"F26 invariant test asserts (a)/(b)/(c)"** — **pass**. `git_test.go` `TestGitLister_RelPathInvariant` lines 212-243: (a) `!strings.HasPrefix(rp, "./")` at line 230-232; (b) `!strings.HasPrefix(rp, "/")` at line 233-235; (c) `rp == filepath.ToSlash(rp)` at line 236-238. All three assertions present, and the test fails at line 241 if zero files are emitted (`if count == 0`) so the invariant is verified against actual output, not vacuously true.

- **"5 tests present + correct + skip on missing git"** — **pass**. All five enumerated tests exist in `git_test.go`: `TestGitLister_List_InRepo` (line 46), `TestGitLister_List_SubdirRoot` (line 74), `TestGitLister_FilterHidden` (line 128), `TestGitLister_ContextCancel` (line 178), `TestGitLister_RelPathInvariant` (line 212). Each calls `skipIfNoGit(t)` (lines 15-20: `exec.LookPath("git")` failure → `t.Skip("git binary not found")`). `_List_SubdirRoot` is the explicit F17/Decision-E validator: walks `internal/fileset/` and asserts no `"internal/"` prefix on any RelPath (line 98-100) AND that `"file.go"` + `"walker.go"` appear (lines 110-122). `_FilterHidden` runs the test twice (once with each polarity) and asserts `.gitignore` exclusion/inclusion (lines 147-149, 164-173). `_ContextCancel` cancels before iteration and asserts `context.Canceled` (line 204-206) with a documented `t.Skip` carve-out for buffered-output races (line 201-203) — acceptable per Builder design decision.

- **"F1 carry-over fix wrapped at Detect"** — **pass**. `git diff HEAD~1 -- internal/lister/lister.go` (visible above) shows exactly one delta in lister.go: `-		return nil, err` → `+		return nil, fmt.Errorf("lister: detect: %w", err)` at line 51 (the `filepath.Abs` error path). The other two `lister: detect: %w` wraps from Unit 4.1 (sentinel branch at line 67, OS-level failure at line 82) remain. All three error paths now consistently use the same prefix.

- **"C11 narrowing — only `undefined: newWalkLister` remains"** — **pass**. Re-ran `mage build ./internal/lister/...`: exit 1 with exactly two errors, both `undefined: newWalkLister` (`lister.go:57:10` and `lister.go:77:10`). The `undefined: newGitLister` error from 4.1 is gone. Failure mode is exactly the deliberate trade-off — symbol defined in Unit 4.3 (`walk.go`) referenced from `Detect`'s two `newWalkLister` call sites. Builder's worklog line 81-85 captured the same output verbatim.

- **"Other packages still green"** — **pass**. Re-ran `mage test ./internal/fileset/... ./internal/counting/... ./internal/ignore/... ./internal/render/... ./cmd/...`. Output: `ok cmd/rak (cached)`, `ok internal/counting (cached)`, `ok internal/fileset (cached)`, `ok internal/ignore (cached)`, `ok internal/render (cached)`. Only `internal/lister [build failed]` is reported as failing (expected C11 carve-out — that package is verified separately at 4.3 close). Note: `internal/summary` and `internal/tokens` packages do not yet exist in the tree (forward-looking in PLAN.md project map); their absence is not a regression.

- **"Doc comments on every exported symbol (rule 11)"** — **pass**. Verified by direct read:
  - `GitLister` struct — `git.go` lines 18-27, doc starts with `GitLister`, explains git ls-files mechanism + Decision-A unreachability for `DisableGitignore`.
  - `NewGitListerForTest` — `git.go` lines 72-75, doc starts with `NewGitListerForTest`, explains the `package lister_test` delegation pattern and includes the "Not intended for production use" disclaimer.
  - `List` — `git.go` lines 93-105, doc starts with `List`, documents the iterator contract (per-entry errors, context cancellation, F14 guard) and Decision E (paths toplevel-relative regardless of CWD).
  - `newGitLister` (unexported) and `anySegmentHidden` (unexported) carry doc comments too even though rule 11 does not require them — bonus hygiene, not a finding.

### F1 — `NewGitListerForTest` export pattern (design-quality, low severity)

- **Axis:** spec-conformance / Go-idiomatic naming.
- **Claim:** the project's test-package convention is mixed: `internal/counting`, `internal/fileset`, `internal/render` use internal `package <pkg>`; `internal/ignore` uses external `package <pkg>_test`. The lister package follows the `ignore` precedent (`package lister_test` from Unit 4.1 forward), so the external-test choice is consistent with one of the two coexisting in-tree patterns. Given the external-test choice, the builder needed an exported way for tests to construct a `GitLister` without going through `Detect` — they added `NewGitListerForTest` in `git.go` at lines 76-78.
- **Why this is informational, not a defect:** the export-in-production-file pattern is functional but slightly less idiomatic than the Go-stdlib `export_test.go` pattern (a `_test.go`-named file in the same package declaring `var NewGitListerForTest = newGitLister`, which links only during testing and does not appear in production builds). The current choice puts a "ForTest" symbol in the public package surface — discoverable via `go doc`, indexable by Hylla as a public symbol. Whether to prefer `export_test.go` is a small style call; Unit 4.3's planned `NewWalkLister` will land an exported constructor for non-test reasons (`cmd/rak` integration tests construct one with `fstest.MapFS` per PLAN.md Unit 4.3 acceptance), so the lister package already accepts that pattern.
- **Recommendation:** no remediation required for Unit 4.2 close. If the orchestrator later wants tighter encapsulation, a follow-up nit could replace `NewGitListerForTest` (in `git.go`) with an `export_test.go` shim (`var NewGitListerForTest = newGitLister` in the same package as the production code, but `_test.go`-suffixed so it only links during `go test`). Not blocking.

### F2 — `t.Helper()` cosmetic carry-forward (informational)

- **Claim:** `git_test.go` `skipIfNoGit` (line 16) and `mainDir` (line 25) / `filesetDir` (line 36) call `t.Helper()` correctly — they're actual subroutines invoked by tests, so the line affects failure-reporting walks meaningfully. Unit 4.1's analogous `t.Helper()` at top-of-test in `lister_test.go` was a cosmetic no-op (F1 finding at Unit 4.1 Round 1); 4.2's uses are not.
- **Why informational:** confirms that 4.2's helper usage is correct and addresses (without depending on) the 4.1 F1 informational point. No action.

### F3 — `TestGitLister_MidWalkGitFailure` gap (acknowledged in PLAN.md)

- **Claim:** PLAN.md Unit 4.2 acceptance line 81 explicitly accepts the gap: *"cleanly stubbing `exec.Command` at the package level is complex. Accepted gap: this path is not unit-tested in 4.2."* Builder's worklog § "Hylla Feedback / Gap Notes" line 106 records the gap with the agreed sentence. The integration path relies on OS-level partial-output behavior on git failure mid-iteration. The error-path code in `List` lines 112-120 (cmd.Output error → distinguishes context.Cancel from a git failure → wraps with `lister: git ls-files: %w`) is present but exercised only end-to-end. Acceptable accepted gap per the plan.

### Evidence summary

- `git show HEAD --stat`: 6 files changed in commit `e12f40e` — `BUILDER_WORKLOG.md` (+48), `PLAN.md` (state flip), `internal/lister/git.go` (+184 new), `internal/lister/git_test.go` (+243 new), `internal/lister/lister.go` (+2-1, F1 wrap), `internal/lister/lister_test.go` (+3-5, activate TODO type assertion). Matches builder's worklog § "Files touched".
- `git diff HEAD~1 -- internal/lister/lister.go`: exactly the F1 carry-over wrap at the `filepath.Abs` error path.
- `git diff HEAD~1 -- internal/lister/lister_test.go`: activates the previously-commented `*lister.GitLister` type assertion in `TestDetect_InsideRepo` — drops the `_ = got` stub.
- `git diff HEAD~1 -- internal/fileset/`: empty — no fileset changes at 4.2 (matches builder claim that only `lister/` package was touched).
- `mage build ./internal/lister/...`: exit 1 with exactly two `undefined: newWalkLister` errors; no `undefined: newGitLister`.
- `mage test ./internal/fileset/... ./internal/counting/... ./internal/ignore/... ./internal/render/... ./cmd/...`: all packages report `ok ... (cached)`; only `internal/lister [build failed]` (expected).
- Cross-package verifications: `fileset.IsHidden` (file.go:124), `fileset.NewFile` (file.go:63, signature `(fsys fs.FS, path, relPath string) *File`), `ignore.New` (ignore.go:74), Walker depth `>=` (walker.go:223/226) — all consistent with `git.go`'s call sites.

## Hylla Feedback

None — Hylla answered everything needed at the Drop 3 baseline (the symbols `GitLister` consumes — `fileset.IsHidden`, `fileset.NewFile` not yet in Hylla because added in Unit 4.1 post-baseline, `ignore.New`, `fileset.WalkOptions` — were verified via `Read` of the live tree for the Unit 4.1 deltas and from baseline knowledge for the rest). No fallback was forced by a missing Hylla result.

## Unit 4.2 — Round 2

**Verdict:** pass

### Round-1-finding resolution audit

- **F3 (`NewGitListerForTest` deleted + `git_test.go` rehomed)** — **pass**.
  - **Export deleted:** `git diff HEAD~1 -- internal/lister/git.go` shows the entire `NewGitListerForTest` block removed (7 LOC: doc comment block + `func NewGitListerForTest(...) (*GitLister, error) { return newGitLister(ctx, root, opts) }`). Direct read of `internal/lister/git.go` confirms no `NewGitListerForTest` symbol exists at any line. The only remaining exported lister symbol set is: `GitLister` (struct, line 28 — intentionally kept per Unit 4.1's `lister_test.go` type assertion at line 41) and its `List` method (line 98). No "ForTest" suffix anywhere in the public surface.
  - **`git_test.go` rehomed:** `git_test.go` line 1 is now `package lister` (verified by direct Read; diff hunk header confirms `-package lister_test` → `+package lister`). The `github.com/evanmschultz/rak/internal/lister` self-import is removed (diff shows `-	"github.com/evanmschultz/rak/internal/lister"`). The current import block is `context`, `os/exec`, `path/filepath`, `strings`, `testing`, `github.com/evanmschultz/rak/internal/fileset` — no self-import.
  - **All call sites rewritten:** the diff shows 6 substitutions of `lister.NewGitListerForTest(...)` → `newGitLister(...)`, covering every test that constructed a `GitLister`: `TestGitLister_List_InRepo` (line 50), `TestGitLister_List_SubdirRoot` (line 78), `TestGitLister_FilterHidden` (×2 at lines 137 and 155), `TestGitLister_ContextCancel` (line 206), `TestGitLister_RelPathInvariant` (line 240). All 6 now call the unexported `newGitLister` directly, accessible because the test file shares the `lister` package.
  - **`lister_test.go` unaffected:** Read of `internal/lister/lister_test.go` confirms line 1 still reads `package lister_test`; the type assertion at line 41 still references `*lister.GitLister` (exported type, unchanged). The only symbols `lister_test.go` consumes from package `lister` are `lister.Detect`, `lister.GitLister`, and `lister.ErrNoGitignoreInRepo` — none of which were affected by the F3 fix. No compile break introduced in this file.
  - **Doc comment on `GitLister` updated:** `git.go` lines 26-27 now read `"GitLister is exported so callers (e.g. lister_test.go) can perform type assertions on the value returned by lister.Detect."` — the stale reference to "TODO unit 4.2 markers" from Round 1 is gone, and the explanation correctly identifies the actual exported use-case (the external `lister_test.go` type assertion).

- **F1 (loop-order note in worklog)** — **pass**. `BUILDER_WORKLOG.md` Round 2 lines 121-123 (verified via Read) contain an explicit section heading `### F1 note — loop-order deviation from PLAN.md acceptance` followed by a 2-sentence note: *"Loop-order deviation from PLAN.md acceptance (context check hoisted from step 5 to step 1 in the `List` per-path loop) is deliberate — provides faster cancellation response without changing the emitted set. PLAN.md lists context as step 5 for narrative ordering; the implementation prioritizes it at step 1 for runtime correctness (fail-fast on cancellation before doing any string work)."* The note documents the deviation, justifies it (cancellation latency), and explicitly clarifies that the emitted set is unchanged — exactly what Round 1's F1 finding asked for.

- **F2 (non-first-segment hidden test)** — **pass**. `git_test.go` lines 180-196 contain a new `t.Run("anySegmentHidden_NonFirstSegment", ...)` sub-test inside `TestGitLister_FilterHidden`. The sub-test directly calls the unexported `anySegmentHidden` helper (accessible now that the test is in `package lister`) with 4 table-driven cases:
  - `"internal/.cache/x.bin"` → expects `true` (hidden at segment index 1) — **directly exercises the loop body past index 0**, which is the case Round 1 flagged as uncovered.
  - `"a/b/.hidden/c.txt"` → expects `true` (hidden at segment index 2) — additionally exercises a deeper-nested path.
  - `"normal/path/file.go"` → expects `false` (negative case, verifies the function correctly returns false when no segment is hidden).
  - `".hidden"` → expects `true` (hidden at index 0, kept for completeness).
  The new sub-test is paired with a doc-comment update at lines 128-129 explaining the F2 coverage purpose. The negative case (`"normal/path/file.go"`) is critical — without it, the test could pass with a buggy `anySegmentHidden` that always returns true.

### Regression checks

- **Mage scoped subset green** — **pass**. Ran `mage test ./internal/fileset/... ./internal/counting/... ./internal/ignore/... ./internal/render/... ./internal/summary/... ./cmd/...`. (Mage's `test` target wraps `go test -race ./...`, so output enumerates the full module.) Results:
  - `ok cmd/rak (cached)`
  - `ok internal/counting (cached)`
  - `ok internal/fileset (cached)`
  - `ok internal/ignore (cached)`
  - `ok internal/render (cached)`
  - `FAIL internal/lister [build failed]` — expected C11 carve-out (see next bullet).
  - `internal/summary` and `internal/tokens` packages do not exist yet (forward-looking in PLAN.md project map); absence is correct for the current tree, not a regression.
- **C11 narrowing unchanged** — **pass**. Ran `mage build ./internal/lister/...`. Exit 1 with exactly two compile errors, both `undefined: newWalkLister`:
  - `internal/lister/lister.go:57:10: undefined: newWalkLister`
  - `internal/lister/lister.go:77:10: undefined: newWalkLister`
  Identical to Round 1's output (same two lines, same symbol). No NEW undefined symbols appeared after the F3 rehome — confirming `newGitLister`, `anySegmentHidden`, and the test-internal access pattern all resolve correctly within `package lister`. The lister package will remain in this exact state until Unit 4.3 lands `newWalkLister`.
- **Unit state back to done** — **pass**. `drops/DROP_4_DEFAULT_BEHAVIOR_TRACKED_TOON/PLAN.md` line 60 reads `- **State:** done` (under `### Unit 4.2 — internal/lister.GitLister: git-backed file enumeration`). The diff confirms the line flipped from `in_progress` to `done` in the same commit (`d65b97c`). Round 1's findings were addressed in-place with no new compile or test surface area added beyond the F2 sub-test.

### Findings

None. Round 2 produced 0 new findings. The F3 remediation (test rehome) is the cleaner of the two options Round 1 floated (delete the export entirely vs. switch to `export_test.go` shim) — it removes the public-surface noise without introducing a new test-only file, and aligns the lister package's test-style with the project's mixed internal/external pattern (consistent with how `fileset` and `counting` keep tests internal). F1 worklog note is precise and load-bearing (documents intentional deviation). F2 sub-test directly exercises the previously-uncovered loop body of `anySegmentHidden` with both positive (deeper-than-first-segment) and negative cases.

### Evidence summary

- `git log --oneline -5` → revise commit is `d65b97c` ("refactor(lister): drop newgitlisterfortest, rehome gitlister tests").
- `git show HEAD --stat` → 4 files changed: `BUILDER_WORKLOG.md` (+42), `drops/.../PLAN.md` (+1-1, state flip), `internal/lister/git.go` (+2-10, NewGitListerForTest delete + doc update), `internal/lister/git_test.go` (+31-8, package decl + import drop + 6 call-site rewrites + F2 sub-test).
- `git diff HEAD~1 -- internal/lister/`: matches the four edits above; no other lister deltas.
- Direct Read of `internal/lister/git.go`: no `NewGitListerForTest` symbol present; `GitLister` still exported; doc comment refreshed.
- Direct Read of `internal/lister/git_test.go`: `package lister` at line 1; no `lister` self-import; all `newGitLister(...)` call sites; new `anySegmentHidden_NonFirstSegment` t.Run at lines 180-196 with 4 table cases (3 positive + 1 negative).
- Direct Read of `internal/lister/lister_test.go`: still `package lister_test` at line 1; still imports `internal/lister`; type assertion at line 41 still `*lister.GitLister` (exported type unaffected).
- `mage build ./internal/lister/...`: exit 1 with exactly two `undefined: newWalkLister` errors (lines 57 + 77 of lister.go). Same narrowing as Round 1.
- `mage test ./internal/fileset/... ./internal/counting/... ./internal/ignore/... ./internal/render/... ./internal/summary/... ./cmd/...`: 5 non-lister packages report `ok ... (cached)`; only `internal/lister` fails (expected C11 carve-out).

## Hylla Feedback

None — Round 2 changes were entirely within files touched in the same commit (`git.go` and `git_test.go`) plus markdown. Hylla's `@main` baseline (end of Drop 3) does not yet index the lister package, so all evidence-gathering used `git diff` and `Read` of the live tree. No fallback was forced by a missing Hylla result.

## Unit 4.3 — Round 1

**Verdict:** pass-with-findings

### Primary acceptance audit (WalkLister scope)

- **`walk.go` structure — pass.** File present at `internal/lister/walk.go` (43 LOC). Contains:
  - `type WalkLister struct { walker *fileset.Walker }` (line 15-17).
  - `func newWalkLister(fsys fs.FS, root string, opts fileset.WalkOptions) *WalkLister` (line 22-24) — unexported, calls `fileset.NewWalker`.
  - `func NewWalkLister(fsys fs.FS, root string, opts fileset.WalkOptions) *WalkLister` (line 31-33) — exported, identical body. Doc comment explains rationale (C2 — cmd/rak test injection without going through `Detect`).
  - `func (wl *WalkLister) List(ctx context.Context) iter.Seq2[*fileset.File, error]` (line 38-40) — delegates verbatim to `wl.walker.Walk(ctx)`. F22 pure pass-through confirmed: zero filter logic in WalkLister.
  - `var _ FileLister = (*WalkLister)(nil)` (line 43) — compile-time assertion present.
- **Doc comments — pass.** All exported identifiers (`WalkLister`, `NewWalkLister`, `List`) have `// Name ...` doc comments per project naming rule 11. Unexported `newWalkLister` also documented.
- **Constructor signature trust — pass via Hylla.** `hylla_search_keyword` confirmed `fileset.NewWalker(fsys fs.FS, root string, opts WalkOptions) *Walker` — matches what `walk.go` calls.
- **F22 pure pass-through — pass.** `List` body is exactly `return wl.walker.Walk(ctx)`. No double-filter, no input transform, no error wrapping. `Walker.Walk` (verified via `hylla_node_full`) applies depth, hidden, gitignore, and include/exclude filters internally — WalkLister inherits all of them transitively without re-applying.
- **`walk_test.go` — pass.** 6 tests in `package lister` (internal):
  - `TestWalkLister_EmptyFS` (line 35) — empty MapFS, no emissions, no errors.
  - `TestWalkLister_FlatFiles` (line 49) — two text files at root, both yielded with correct RelPath.
  - `TestWalkLister_DepthFilter` (line 75) — three files at depths 0/1/2; `WalkOptions{Depth:1}` yields only the depth-0 file. Walker semantics confirmed via Hylla: `Walker.Walk` uses `depth >= w.opts.Depth` matching Walker's documented behaviour.
  - `TestWalkLister_HiddenFilter` (line 97) — two subtests `default_excludes_hidden` and `include_hidden`. Both pass MapFS through; rely on `fileset.IsHidden(".hidden.txt") == true` (verified via Hylla).
  - `TestWalkLister_ImplementsFileLister` (line 138) — compile-time assertion duplicated at test scope. Pass.
  - `TestWalkLister_RelPathInvariant` (line 145) — F26 invariant: iterates a 3-file MapFS (`a.txt`, `sub/b.txt`, `sub/deep/c.txt`) and asserts all three claims: `!strings.HasPrefix(rp, "./")`, `!strings.HasPrefix(rp, "/")`, `rp == filepath.ToSlash(rp)`. All three F26 sub-claims present and tested.
- **Internal-package decision — pass.** Tests are in `package lister` (internal), consistent with the Round-2 4.2 convention (rehomed from `package lister_test`). Allows direct access to unexported `newWalkLister` and the `FileLister` type without re-importing the package.
- **`mage ci` green — pass.** Re-ran `mage ci` from `main/`: output `0 issues.` + 6 packages reporting `ok ... (cached)` (cmd/rak, internal/counting, internal/fileset, internal/ignore, internal/lister, internal/render). The `internal/lister ok` line is the first green for that package since the C11 carve-out opened in Unit 4.1. Cached `ok` reflects the prior in-builder-session real run — tests are deterministic (no network, no disk other than `fstest.MapFS`), and the cache key is content-hashed, so cached-ok is authoritative. Note: I was unable to force a `-count=1` rerun because `mage test` has no verbose flag and raw `go test` is forbidden by CLAUDE.md § "Build Verification"; the cached-ok evidence is what the harness allows, and it matches what `mage ci` would emit on a cold run.

### Scope-drift audit (4 files outside declared paths)

Builder touched 4 files outside Unit 4.3's declared `paths` (`walk.go`, `walk_test.go`). Each examined independently:

- **`internal/lister/git.go` — `gitCleanEnv()` helper (~26 LOC) + 2 `cmd.Env =` wire sites — pass-with-note.** The helper strips `GIT_DIR`, `GIT_WORK_TREE`, `GIT_INDEX_FILE` from `os.Environ()` and preserves everything else (PATH, HOME, USER, etc.). Behavior is consistent with the F16 design intent ("Detect resolves root to absolute; cmd.Dir set to that absolute path; never rely on process CWD for git commands") — extending the rule to "never rely on process ENV either, for the variables that override `cmd.Dir` semantics". The three stripped vars are exactly the ones that would override `cmd.Dir`-based repo discovery. Production users who set `GIT_DIR` explicitly (e.g. pointing rak at a non-default git dir) now find rak ignores that override. This is design-consistent for v0.1.0 — rak is path-driven, not env-driven (Decision 32). **Surfaceable note, not a blocker.**
- **`internal/lister/git_test.go` — `skipIfGitEnvBroken` helper + `errors` import + 5 wire sites — pass.** Helper uses `errors.As(err, &exitErr)` and checks `exitErr.ExitCode() == 128`; emits a non-helpful exit elsewhere (other values continue to fail). 5 sites: `TestGitLister_List_InRepo` (line 67 area), `TestGitLister_List_SubdirRoot` (line 96), `TestGitLister_FilterHidden` (line 156), `TestGitLister_ContextCancel` (line 226), `TestGitLister_RelPathInvariant` (line 261). Matches the diff's `+24 -0` and the worklog's "5 sites" claim. Acceptable pragmatic test guard.
- **`internal/lister/lister.go` — `cmd.Env = gitCleanEnv()` wire (1 line) + `ErrNoGitignoreInRepo` message edit (1 line) — pass-with-findings (see Findings #1).** Wire site for `Detect`'s git probe is consistent with the `git.go` wire pattern. The message edit needs a separate finding because it touches the F19 R2-F2 sentinel-message contract.
- **`internal/lister/lister_test.go` — activated TODO 4.3 (uncommented 3 lines) + `strings` import + inline exit-128 skip (5 lines) — pass.** Activating the `*lister.WalkLister` type assertion in `TestDetect_OutsideRepo` (lines 77-79) is required by 4.3 acceptance — the TODO comment in `lister_test.go` from Unit 4.1 explicitly said this gets uncommented when 4.3 lands. The inline exit-128 skip (lines 38-40 of `TestDetect_InsideRepo`) duplicates the `skipIfGitEnvBroken` logic from `git_test.go` rather than reusing it, but `lister_test.go` is in `package lister_test` (external), so it cannot access the unexported helper without exporting it. Minor DRY-ness gripe — acceptable.

### Walker untouched — pass

`git diff HEAD~1 -- internal/fileset/` returns empty output. Drop 3's `fileset.Walker.Walk` semantics are byte-identical to what landed in Drop 3. F22's contract therefore rests on a fixed substrate. No risk that the WalkLister adapter inherits a changed-but-uncommunicated Walker behavior.

### Findings

- 1.1 [Axis: spec-conformance] [severity: low] `ErrNoGitignoreInRepo` message text diverges from the F19 R2-F2 "literal text" pin in TWO places, not one. (a) Trailing `.` removed (staticcheck ST1005 — required). (b) First inter-sentence period replaced by `;` (NOT required by staticcheck — ST1005 forbids trailing punctuation, not mid-sentence). Old: `"... in this mode. To count untracked ..."` → New: `"... in this mode; To count untracked ..."` — wait, re-checking: actual change is `"repository. rak counts"` → `"repository; rak counts"`, so the FIRST period-space became semicolon-space. Builder's worklog mentions "semicolon kept between the two sentences" but the diff shows the semicolon was ADDED in place of a period — not "kept". **Contract impact:** `errors.Is(err, ErrNoGitignoreInRepo)` is unaffected (sentinel identity is by `errors.New` pointer, not message); `lister_test.go::TestDetect_NoGitignoreInRepo_ReturnsSentinel` uses `errors.Is` → still passes; Unit 4.4's planned `TestRootCmd_NoGitignoreInRepo_Errors` also uses `errors.Is` per PLAN.md → also fine. User-visible message changes cosmetically. **Evidence:** `git diff HEAD~1 -- internal/lister/lister.go` lines 32-35; PLAN.md line 184 for the original F19 text. **Fix hint:** either (a) restore the inter-sentence period (keep only the trailing-period removal), or (b) update F19's literal-text pin in PLAN.md to match the new wording and add a one-line note to the worklog clarifying the second edit. Either is acceptable; the current state is contract-preserving but the worklog narrative undersells the change.

### Missing evidence

- 2.1 [Axis: acceptance-criteria-coverage] [severity: low] Could not force a `-count=1` rerun of the WalkLister tests. The `mage test` cache is content-keyed, so `(cached) ok` is authoritative when the test source hasn't drifted — and the diff shows no post-commit edits to `walk.go` / `walk_test.go` — but a fresh-cache run would be stronger evidence. CLAUDE.md § "Build Verification" forbids raw `go test`, and mage has no verbose / no-cache flag. Acceptable gap given the harness; flagged for transparency.

### Evidence summary

- `git show HEAD --stat` → commit `1f16f8d` ("feat(lister): add walklister, close lister compile break"), 8 files changed: `BUILDER_WORKLOG.md` (+49), `drops/.../PLAN.md` (+1-1, state flip), `internal/lister/git.go` (+29), `internal/lister/git_test.go` (+24), `internal/lister/lister.go` (+2-1), `internal/lister/lister_test.go` (+10-5), `internal/lister/walk.go` (+43), `internal/lister/walk_test.go` (+167).
- `git diff HEAD~1 -- internal/fileset/` → empty (Walker untouched).
- `mage ci` → `0 issues.` + 6 packages `ok`, including `internal/lister ok` (first green for that package).
- `hylla_search_keyword` for `NewWalker fileset` → confirmed signature `func NewWalker(fsys fs.FS, root string, opts WalkOptions) *Walker`.
- `hylla_node_full` for `fileset.Walker.Walk` → confirmed Walker applies depth (`if w.opts.Depth != 0`), hidden (`if !w.opts.IncludeHidden ... IsHidden(d.Name())`), gitignore (`readGitignore` + matcher rebuild), and include/exclude (`ignore.New(roots, w.opts.Includes, w.opts.Excludes)`). All four filters live in Walker — F22 pure-pass-through is sound.
- `hylla_node_full` for `fileset.IsHidden` → confirmed `IsHidden(".hidden.txt") == true` (returns true for any non-empty, non-`.`, non-`..` name starting with `.`). Hidden-test assertion holds.
- `Read` of `walk.go` and `walk_test.go` → six tests, three F26 sub-claims tested, compile-time assertion triply-asserted (walk.go line 43, walk_test.go line 14, walk_test.go line 138-140).

## Hylla Feedback

None — Hylla answered everything needed for the WalkLister proof review. Three queries used: `hylla_search_keyword` for `NewWalker fileset`, `hylla_node_full` for `Walker.Walk`, `hylla_node_full` for `IsHidden`. All three returned the expected nodes with full content. No fallback to `LSP` was forced. The scope-drift files (`git.go`, `git_test.go`, `lister.go`, `lister_test.go`) were inspected via `Read` rather than Hylla because they were touched in this same commit and would be stale in the `@main` baseline — that is `git diff` territory, not a Hylla miss.
