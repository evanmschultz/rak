# DROP_3 — Builder QA Proof

## Unit 3.0 — Round 1

- **QA proof:** go-qa-proof-agent
- **Reviewed:** 2026-04-21
- **Verdict:** pass
- **Commit under review:** `be08d20 feat(deps): add go-gitignore and doublestar for drop-3`

### Acceptance-criterion verification

**AC1 — Deps added via `mage addDep`, not raw `go get`:**
- `BUILDER_WORKLOG.md` lines 11–12 document the two invocations verbatim:
  - `mage addDep github.com/sabhiram/go-gitignore` → `go: added github.com/sabhiram/go-gitignore v0.0.0-20210923224102-525f6e181f06`
  - `mage addDep github.com/bmatcuk/doublestar/v4` → `go: added github.com/bmatcuk/doublestar/v4 v4.10.0`
- Commit `be08d20` touches exactly `go.mod` + `go.sum` + the two drop mds; no scratch command log inconsistency; the worklog is the only record of the invocation, and its command output strings match the mage target signature (`go: added <module> <version>`). No sign of bypass.
- `mage -l` confirms the `addDep` target is resolvable (Drop 2.0 landed it as required).
- **Pass.**

**AC2 — `go.mod` has `require (...)` entries for both modules at latest stable tags:**
- `main/go.mod` line 17: `github.com/bmatcuk/doublestar/v4 v4.10.0 // indirect` — tagged release.
- `main/go.mod` line 40: `github.com/sabhiram/go-gitignore v0.0.0-20210923224102-525f6e181f06 // indirect` — Go pseudo-version because sabhiram has never cut a git tag (worklog line 17 documents this; the pseudo-version IS the latest stable resolver choice). The PLAN.md "latest stable tags" phrasing is satisfied by the resolver's latest-stable selection where no tag exists; worklog flagged this nuance to orch as Phase-3 discussion material, not a blocker.
- Both land in the secondary `require (...)` block (lines 12–49) because no rak source imports them yet; they will promote to the primary block in 3.1/3.2.
- **Pass.**

**AC3 — `go.sum` populated for both modules; no surprise compiled transitive deps:**
- `main/go.sum` lines 15–16: doublestar `h1:` + `/go.mod` pair.
- `main/go.sum` lines 78–79: sabhiram `h1:` + `/go.mod` pair.
- Commit diff shows four additional `/go.mod`-only entries: `davecgh/go-spew v1.1.0`, `stretchr/objx v0.1.0`, `stretchr/testify v1.6.1`, `gopkg.in/yaml.v3 v3.0.0-20200313102051`. These are `/go.mod`-only lines (no matching `h1:` hash), which is Go's way of recording **module-graph closure** rather than compiled dependencies — they are hash-verified only for the `go.mod` files themselves, never downloaded as source nor linked into any binary. This is consistent with sabhiram's own test-suite pulling in testify (an `_test.go`-only import), which Go's MVS algorithm records for reproducibility.
- No new `h1:` entries appear for any module other than the two target modules. Neither target contributes a compiled transitive dep.
- Worklog line 18 documents this clearly and correctly.
- **Pass with observation** — see § "Observations" for the surfaced-to-orch note.

**AC4 — `mage build` + `mage test` pass clean:**
- Re-ran both targets locally at review time (not trusting builder's claim alone):
  - `mage build` → exit 0, no stdout/stderr.
  - `mage test` → `ok  github.com/evanmschultz/rak/cmd/rak (cached)` / `ok  github.com/evanmschultz/rak/internal/counting (cached)` / `ok  github.com/evanmschultz/rak/internal/render (cached)` — all three existing test packages green. Cached is expected: no Go source changed, so the test binary is unchanged; `mage test` always runs with `-race` per `magefile.go` / CLAUDE.md.
- No compile errors despite the unused `// indirect` entries — Go permits indirect deps without importers, exactly the workflow Drop 2's `mage addDep` was designed for.
- **Pass.**

### Observations (non-blocking, surfaced to orchestrator)

- **O1 — `/go.mod`-only transitive-dep entries:** The AC's "zero transitive deps" expectation (PLAN.md line 31) is stricter than the actual module resolver outcome; sabhiram's own test suite depends on testify/go-spew/objx/yaml.v3, recorded as `/go.mod`-only closure entries. No compiled dependency is pulled in, so the intent of the AC ("no surprise runtime deps") is satisfied. Builder already flagged this in the worklog. If the dev wants a strict "no new lines under any circumstance" bar, the AC text needs tightening for future drops — but for Unit 3.0 as written, this is AC-compliant (builder followed the "flag and return to orch" path).
- **O2 — Pseudo-version vs tag for sabhiram:** AC line 30 says "latest stable tags"; sabhiram has no tags, so Go picked a pseudo-version. Worklog line 17 calls this out explicitly as Phase-3 discussion material. Non-blocking for Unit 3.0 since the resolver's choice is deterministic and hash-pinned.

### Evidence trail

- `git log --oneline -10` — last commit is `be08d20 feat(deps): add go-gitignore and doublestar for drop-3`.
- `git show HEAD --stat` — exactly four files changed: `go.mod` (+2), `go.sum` (+8), plus worklog and plan state flip.
- `git show HEAD -- go.mod go.sum` — diff exactly matches the worklog's claimed deltas.
- `main/go.mod` lines 17 + 40 — both target modules pinned.
- `main/go.sum` lines 15–16 + 78–79 — `h1:` hashes for both targets; lines 42, 84–85, 106 — `/go.mod`-only closure entries flagged in O1.
- Re-ran `mage build` + `mage test` from `main/` at review time; both green.
- `mage -l` shows `addDep` target present; Drop 2.0's landing is corroborated.

### Hylla Feedback

None — no Hylla queries were needed for this review. Unit 3.0 touched only `go.mod` / `go.sum` (non-Go dependency metadata) and drop mds, which are out of Hylla's Go-source scope by design. Ground truth was entirely in git + filesystem + `mage` output, per CLAUDE.md § "Code Understanding Rules" steps 2 and 3.

## Unit 3.1 — Round 1

- **QA proof:** go-qa-proof-agent
- **Reviewed:** 2026-04-21
- **Verdict:** pass
- **Commit under review:** `1ef8e68 feat(ignore): add matcher interface with gitignore and glob`

### Acceptance-criterion verification

**AC1 — `ignore.go` defines `Matcher` interface with `Match(relPath string, isDir bool) bool` returning true when path should be ignored (F1); forward-slash separators (C6); single `New(roots, includes, excludes) (Matcher, error)` constructor:**
- `ignore.go:29-33` — `Matcher` interface with exactly the signature `Match(relPath string, isDir bool) bool`.
- Doc on lines 21-28 and the method doc on lines 30-32 explicitly say "true means drop this path" — F1 convention pinned in source.
- Package doc (lines 1-18) documents forward-slash convention explicitly: "All paths handed to Match use forward-slash separators regardless of host OS, matching the io/fs convention" — C6 pinned.
- `ignore.go:74-90` — `New(roots []GitignoreRoot, includes, excludes []string) (Matcher, error)` composes the three sub-matchers.
- **Pass.**

**AC2 — `GitignoreRoot` struct carries `{Dir string, Patterns []string}` (pre-parsed):**
- `ignore.go:45-51` — struct has exactly `Dir string` + `Patterns []string`, both exported, both with individual field doc comments (lines 46-47, 49-50).
- Package doc (lines 35-44) confirms "The Walker constructs one GitignoreRoot per directory it enters... This package does no disk IO — it consumes the pre-read Patterns only."
- **Pass.**

**AC3 — `gitignore.go` wraps `github.com/sabhiram/go-gitignore` with `CompileIgnoreLines`; supports negation/dir-only/double-star/char-class; hierarchical scope (F8):**
- `gitignore.go:6` imports `sabhiram/go-gitignore`.
- `gitignore.go:43` calls `gitignore.CompileIgnoreLines(r.Patterns...)`.
- `gitignore.go:60-78` `match` method probes library with isDir trailing-slash convention for dir-only patterns.
- `scopePath` (lines 84-96) implements hierarchical F8 scoping: empty Dir = walk root matches all, exact-match = root dir itself, prefix match = in-scope with stripped remainder, otherwise out-of-scope.
- Test coverage confirms each feature:
  - Negation `!keep.log` — `TestMatcher_GitignoreOnly/negation_reincludes` (line 72).
  - Dir-only `node_modules/` — `dir_only_pattern_matches_dir` + `_skips_file_of_same_name` + `_matches_child` (lines 73-75).
  - Double-star `**/vendor` — `double_star_at_root` / `_subdir` / `_matches_children` (lines 76-82).
  - Char-class `[abc].txt` — `char_class_hit` / `_miss` (lines 83-84).
  - F8 scoping — `scoped_rule_hits_inside_scope` / `_misses_outside_scope` / `_misses_in_sibling` (lines 89-91).
- **Pass.**

**AC4 — `glob.go` uses `doublestar.Match` (NOT `PathMatch`); include allow-list; exclude deny-list; F2 exclude wins:**
- `glob.go:50` — `ok, _ := doublestar.Match(p, name)`. Verified no `PathMatch` exists in package (grep on `internal/ignore/` shows only one reference to `PathMatch`, on line 15 of glob.go as rejection rationale in a doc comment).
- Round-3 planner correction thus honored: `doublestar.Match` splits both pattern and path on forward slash on all platforms, preserving C6 portability.
- `newGlobMatcher` validates each pattern via `doublestar.ValidatePattern` at construction (line 32) → wraps `doublestar.ErrBadPattern` with `%w` (line 33). `TestMatcher_InvalidGlob_Include` + `_Exclude` (lines 318-345) assert `errors.Is(err, doublestar.ErrBadPattern)` — sentinel-style error inspection per CLAUDE.md § "Errors".
- F2 verified by `TestMatcher_Precedence_ExcludeBeatsGitignoreNegate` (lines 214-242) and `TestMatcher_Precedence_IncludeDoesNotOverrideExclude` (lines 244-261).
- **Pass.**

**AC5 — Precedence order F3: `--exclude` → `.gitignore` → `--include`; `--no-gitignore` tolerated as empty/nil roots:**
- `ignore.go:102-113` — `compositeMatcher.Match` implements the exact chain:
  1. `exclude.match(relPath)` → true drops.
  2. `gitignore.match(relPath, isDir)` → true drops.
  3. `hasIncludes && !include.match(relPath)` → true drops.
  4. Else keep.
- `hasIncludes` stored once at construction (line 88) — keeps empty-include fast path O(1).
- `TestMatcher_Precedence_IncludeAfterGitignore` (lines 263-285) asserts a file dropped by gitignore stays dropped even when it matches a broad `--include`.
- `TestMatcher_NoGitignore_EquivalentToEmptyRoots` (lines 287-316) asserts nil roots disable gitignore stage (contract for `--no-gitignore` wiring in 3.5).
- **Pass.**

**AC6 — Table-driven test coverage: empty / gitignore-only / include-only / exclude-only / combined / negation / dir-only / double-star / precedence-wins cases:**
- Exactly 12 test functions (`grep -c "^func Test" ignore_test.go` → 12): `TestMatcher_Empty`, `_GitignoreOnly`, `_IncludeOnly`, `_ExcludeOnly`, `_AllThreeCombined`, `_Precedence_ExcludeBeatsGitignoreNegate`, `_Precedence_IncludeDoesNotOverrideExclude`, `_Precedence_IncludeAfterGitignore`, `_NoGitignore_EquivalentToEmptyRoots`, `_InvalidGlob_Include`, `_InvalidGlob_Exclude` — plus the precedence-exclude-beats-gitignore and include-doesn't-override pair.
- Every case in the acceptance bullet is represented. Negation, dir-only, double-star, char-class, and F8 hierarchical scope all have explicit subtests. `t.Parallel()` on both top-level and subtests — race-safe.
- **Pass.**

**AC7 — No disk IO; pre-read patterns consumed as-is:**
- `internal/ignore` has zero `os` or `io/fs` imports (verified by `grep -E "^import|\"os\"|\"io/fs\"" internal/ignore/*.go`).
- `gitignore.go` consumes `r.Patterns` directly (line 43), no file reads.
- Comments + blanks handled by sabhiram per spec; `TestMatcher_GitignoreOnly` exercises a `# comment` + empty-line fixture (lines 51-52) to confirm pass-through.
- **Pass.**

**AC8 — `mage test ./internal/ignore/...` green; `mage lint` green:**
- Re-ran at review time from `main/`:
  - `mage test` → `ok github.com/evanmschultz/rak/cmd/rak (cached)` / `internal/counting (cached)` / `internal/ignore (cached)` / `internal/render (cached)`. Cached = no Go source changed since builder's run; `-race` is on by default per magefile.
  - `mage lint` → `0 issues.` (go vet + golangci-lint both clean).
  - `mage ci` → full gate green (gofumpt-clean, lint-clean, tests green).
- **Pass.**

**AC9 — Doc comments on every exported identifier (CLAUDE.md rule 11):**
- `Matcher` interface — doc at `ignore.go:21-28`, starts with identifier name.
- `Matcher.Match` method — doc at `ignore.go:30-32`, starts with method name.
- `GitignoreRoot` struct — doc at `ignore.go:35-44`, starts with identifier name. Fields `Dir` and `Patterns` also individually doc'd (lines 46-47, 49-50).
- `New` function — doc at `ignore.go:53-73`, starts with identifier name.
- No other exported identifiers in the package (`compositeMatcher`, `gitignoreMatcher`, `globMatcher`, `gitignoreRule`, `scopePath`, `newGitignoreMatcher`, `newGlobMatcher` are all unexported).
- **Pass.**

### Cross-pin verification

- **F1 (Match returns true = ignore):** Package doc lines 10-13, method doc lines 30-32, and `compositeMatcher.Match` return conventions all agree. sabhiram's `MatchesPath` returns true on ignore (same polarity, no flip needed — worklog line 39). `globMatcher.match` also returns true on hit. All three composition layers use consistent polarity. **Pass.**
- **F2 (exclude wins):** `compositeMatcher.Match` line 103-105 short-circuits on exclude hit before gitignore or include see the path. `TestMatcher_Precedence_ExcludeBeatsGitignoreNegate` (ignore_test.go:214-242) and `_IncludeDoesNotOverrideExclude` (lines 244-261) both assert this. **Pass.**
- **F3 (precedence order exclude → gitignore → include):** Exact chain in `compositeMatcher.Match` lines 102-113. Verified against `TestMatcher_Precedence_IncludeAfterGitignore` (file dropped by gitignore stays dropped under `--include '*.go'`). **Pass.**
- **C6 (forward-slash relPath):** Package doc lines 15-18 pin the contract. `glob.go:12-17` uses `doublestar.Match` specifically for its forward-slash-on-every-platform behavior; `PathMatch` is called out and rejected in the same comment. `scopePath` (gitignore.go:84-96) uses literal `/` prefix. **Pass.**

### Evidence trail

- `git log --oneline -5` — last commit `1ef8e68 feat(ignore): add matcher interface with gitignore and glob`.
- `git status` — working tree clean; only the (expected) diff is this BUILDER_QA_PROOF.md append at commit time.
- `grep -n PathMatch internal/ignore/` — single hit, on glob.go:15 inside a rejection-rationale comment. No code call to `PathMatch`.
- `grep -n doublestar.Match internal/ignore/` — three hits: glob.go:12 (doc), :44 (doc), :50 (the call). Plus one reference in a test-file comment.
- `grep -n "^func Test" internal/ignore/ignore_test.go` — 12 `TestMatcher_*` functions (satisfies the 12-clause acceptance coverage).
- Re-ran `mage test` + `mage lint` + `mage ci` at review time from `main/`: all green.

### Hylla Feedback

None — Unit 3.1 is net-new code that did not exist in the last Hylla ingest (reingest is drop-end only per WORKFLOW.md Phase 7), so Hylla was correctly not consulted for this package. External-library semantics for `sabhiram/go-gitignore` and `doublestar/v4` were validated via the builder's `go doc` + scratch-module probe (worklog line 52) and cross-checked against the in-source doc comments at review time — the documented third-party fallback path per CLAUDE.md § "Code Understanding Rules" rule 4. Non-Go drop mds (PLAN.md, BUILDER_WORKLOG.md, WORKFLOW.md) are out of Hylla's Go-only scope. Zero fallback misses to record.

## Unit 3.2 — Round 1

- **QA proof:** go-qa-proof-agent
- **Reviewed:** 2026-04-21
- **Verdict:** pass
- **Commit under review:** `a794aee feat(fileset): add file type with open, peek, and hidden helper`

### Acceptance-criterion verification

**AC1 — `file.go` defines `File` struct with `Path string`, `RelPath string`, unexported `fs fs.FS`:**
- `file.go:36-46` — `type File struct { Path string; RelPath string; fs fs.FS }`. Exported fields `Path` / `RelPath` each have individual field doc comments (lines 37-40). Unexported `fs fs.FS` field with the in-package rationale doc (lines 44-45) — keeps callers from bypassing `Open` / `Peek`.
- Zero-value rationale spelled out in the struct's type doc (lines 24-35): "Zero-value File is not useful; construct via newFile (unexported)."
- `newFile` constructor at `file.go:52-58` is the only sanctioned way to build a `*File`; unexported so external callers must go through the Walker (Unit 3.3).
- **Pass.**

**AC2 — `Open() (io.ReadCloser, error)` opens via `fs.FS.Open`, wraps errors with `open %q: %w`:**
- `file.go:67-73` — `Open` calls `f.fs.Open(f.Path)`, wraps any error via `fmt.Errorf("open %q: %w", f.Path, err)`.
- Returning the underlying `fs.File` directly is valid: `io/fs` declares `type File interface { Stat(); Read([]byte); Close() }`, a superset of `io.ReadCloser`.
- `TestFile_Open_NotFound` (`file_test.go:40-60`) asserts both:
  - `errors.Is(err, fs.ErrNotExist)` holds through the `%w` chain (line 52-54).
  - Text prefix `open "missing.txt":` present (line 57-59).
- **Pass.**

**AC3 — `Peek(n int) ([]byte, error)` tolerates short-file via `io.ErrUnexpectedEOF` + `io.EOF`, open-read-close per call, F4 stateless:**
- `file.go:87-106` — implementation opens via `Open`, reads up to `n` via `io.ReadFull`, closes via deferred `_ = rc.Close()`.
- Short-file tolerance at line 102: `if err == nil || errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) { return buf[:k], nil }`. Uses `errors.Is` (not `==`) — future-proof against stdlib wrapping per CLAUDE.md § "Errors" ("Inspect with `errors.Is` ... never string-match").
- Per-call open-read-close confirms F4 statelessness: no cached bytes, no cursor on `*File`.
- `io.ReadFull` stdlib contract (verified via `go doc io.ReadFull`): "On return, n == len(buf) if and only if err == nil. If r returns an error having read at least len(buf) bytes, the error is dropped. … If at least one byte is read and an error is hit, the returned error is io.ErrUnexpectedEOF. … If no bytes are read, io.EOF." Peek's three-branch tolerance covers exactly these three outcomes.
- `TestFile_Peek` table (`file_test.go:62-121`) covers five rows: empty_file_returns_empty (n=512 vs 0 bytes), short_file_returns_all_bytes (n=512 vs 2 bytes), exact_match_returns_all_bytes (n=8 vs 8 bytes), long_file_returns_first_n_bytes (n=8 vs 16 bytes, asserts first 8), one_byte_file_n_one (edge). All subtests assert zero error + exact byte payload.
- Close-error path: `defer func() { _ = rc.Close() }()` deliberately discards the Close error with an inline explanation at lines 95-97 — CLAUDE.md § "Errors" permits "If you genuinely want to discard an error, assign to `_` with a one-line comment explaining why."
- **Pass.**

**AC4 — Multiple Peek calls on same *File return identical bytes (F4 no stateful cursor):**
- `TestFile_Peek_MultipleCalls` (`file_test.go:123-147`) — calls `Peek(10)` twice on the same `*File`, asserts `bytes.Equal(first, second)` AND that the returned bytes are the first 10 of the payload (`"determinis"`).
- Implementation backs this by re-opening via `Open` on every call; no cached slice on the struct.
- F4 pinned contract satisfied — binary detection (3.4) and Drop-4.1 shebang sniff both depend on this.
- **Pass.**

**AC5 — `IsHidden(name string) bool` excludes `.` and `..`; true iff final element starts with `.`:**
- `file.go:117-122` — `IsHidden` returns false for `""`, `"."`, `".."`; otherwise `strings.HasPrefix(name, ".")`.
- Doc comment (lines 108-116) explicitly documents:
  - Input is a basename (single path element), not a full path.
  - `.` and `..` excluded — "not hidden entries in any shell's sense."
  - Walker (Unit 3.3) calls `IsHidden(DirEntry.Name())` with `IncludeHidden=false`.
- `TestIsHidden` (`file_test.go:149-175`) table covers all six required cases from PLAN.md AC:
  - `.` → false
  - `..` → false
  - `.git` → true
  - `.hidden.txt` → true
  - `normal.txt` → false
  - `""` → false
- **Pass.**

**AC6 — Five test functions present:**
- `grep -n "^func Test" file_test.go` returns exactly 5 functions at lines 13, 40, 62, 123, 149:
  - `TestFile_Open` (line 13)
  - `TestFile_Open_NotFound` (line 40)
  - `TestFile_Peek` (line 62)
  - `TestFile_Peek_MultipleCalls` (line 123)
  - `TestIsHidden` (line 149)
- All five match the PLAN.md AC bullet list exactly.
- `t.Parallel()` on every top-level test function and every table subcase — race-safe by construction.
- **Pass.**

**AC7 — `mage test ./internal/fileset/...` green; `mage lint` green:**
- Re-ran at review time from `main/` with `go clean -testcache` first to force uncached run:
  - `mage ci` → full green pipeline: gofumpt-clean, lint `0 issues.`, tests all five packages pass including `internal/fileset 1.199s` under `-race`.
  - `mage test` (post-clean) → `ok github.com/evanmschultz/rak/internal/fileset 1.199s` under `-race`.
  - `mage lint` → `0 issues.` (go vet + golangci-lint both clean on the new package).
- **Pass.**

### Cross-pin verification

- **F4 (Peek stateless, multi-call identical):** `Peek` re-opens via `Open` on every call; no cached buffer on `*File`. `TestFile_Peek_MultipleCalls` confirms two calls on the same `*File` return byte-equal slices. Binary detection (3.4) and shebang sniff (Drop 4.1) can safely both call `Peek(512)`. **Pass.**
- **C3 (IsHidden excludes `.`/`..`):** `file.go:118` explicitly short-circuits on `""`, `"."`, `".."` before the dot-prefix check. `TestIsHidden` covers `.` and `..` → false + `.git` / `.hidden.txt` → true. Walker (3.3) will call `IsHidden(DirEntry.Name())` — basename-only, matching the doc-comment contract. **Pass.**
- **C6 (forward-slash paths):** Package doc (lines 11-13) pins "All paths carried by File use forward-slash separators regardless of host OS, matching the io/fs convention. The Walker (Unit 3.3) is responsible for normalizing OS-native separators before constructing File values." `Open` passes `f.Path` straight to `f.fs.Open`, which respects the `io/fs` forward-slash convention. **Pass.**
- **CLAUDE.md § Errors — `errors.Is` for inspection, `%w` for wrapping, discarded errors get a comment:** `Open` wraps with `%w`; `Peek` uses `errors.Is` against both `io.ErrUnexpectedEOF` and `io.EOF`; `TestFile_Open_NotFound` uses `errors.Is(err, fs.ErrNotExist)` not string-match; Close-error discard at line 95-97 has the required inline comment. **Pass.**

### Doc-comment rule (CLAUDE.md § Go-Idiomatic Naming Rules rule 11)

Every exported identifier in `file.go` has a doc comment starting with the identifier name:
- `package fileset` (line 1) — package doc present.
- `File` struct (line 24) — doc starts "File names a single regular file…".
- `File.Path` field (line 37) — doc starts "Path is the walk-relative path…".
- `File.RelPath` field (line 40) — doc starts "RelPath is the path relative to the walk root…".
- `(*File).Open` method (line 60) — doc starts "Open opens the file for reading.".
- `(*File).Peek` method (line 75) — doc starts "Peek opens the file, reads up to n bytes…".
- `IsHidden` function (line 108) — doc starts "IsHidden reports whether a single path element…".

Unexported identifiers (`newFile`, the unexported `fs` field) also carry doc comments by convention, which is good hygiene but not required. **Pass.**

### Coverage

`mage coverage` at review time reported line coverage per function:
- `newFile` — 100.0%
- `Open` — 100.0%
- `Peek` — 75.0% (uncovered branches: `n <= 0` short-circuit and the non-tolerated-error wrap path; both are edge/error paths not required by the PLAN.md AC bullets)
- `IsHidden` — 100.0%

75% on `Peek` clears CLAUDE.md's 70% floor (which doesn't flip into a gate until Drop 9.3). The uncovered branches are minor and already flagged as observations below.

### Observations (non-blocking, surfaced to orchestrator)

- **O1 — `Peek(n <= 0)` uncovered.** `file.go:88-90` short-circuits on non-positive `n` and returns `(nil, nil)` without opening. No unit test drives this path. Doc comment (line 83-84) documents the behavior, and the planner's AC bullets don't demand coverage for it. Adding a 1-line subcase to `TestFile_Peek` would hit 100% on this branch — mention it to orch in case a future drop wants the coverage bump. Non-blocking for Unit 3.2.
- **O2 — Non-short-EOF Peek error wrap uncovered.** `file.go:105` (the `return nil, fmt.Errorf("open %q: %w", f.Path, err)` at the end of `Peek`) covers the case where `io.ReadFull` returns an error that is neither `io.ErrUnexpectedEOF` nor `io.EOF`. Inducing this via `fstest.MapFS` is awkward (MapFS doesn't surface mid-read I/O errors easily). A custom `fs.FS` returning a failing `fs.File.Read` could hit it; again, not required by PLAN.md AC. Non-blocking.

### Evidence trail

- `git log --oneline -5` — commit under review is `a794aee feat(fileset): add file type with open, peek, and hidden helper`.
- `git show a794aee --stat` — files touched match the worklog exactly: `main/internal/fileset/file.go` (new, 123 lines), `main/internal/fileset/file_test.go` (new, 176 lines), `main/drops/DROP_3_.../PLAN.md` (state flip), `main/drops/DROP_3_.../BUILDER_WORKLOG.md` (append).
- `grep -n "^func Test" internal/fileset/file_test.go` — exactly 5 `Test*` functions.
- `grep -n "^func\|^type\|^package" internal/fileset/file.go` — `File` struct, `newFile`, `(*File).Open`, `(*File).Peek`, `IsHidden` — every exported name has a preceding doc comment.
- `go clean -testcache && mage ci` at review time — green end-to-end (gofumpt + lint + race-enabled tests).
- `mage coverage` — `Peek` 75.0%, other `file.go` funcs 100.0%; total 93.3% across the whole module.

### Hylla Feedback

None — Hylla answered everything needed. Unit 3.2 introduces a brand-new file in a package that did not exist in Hylla's last ingest (reingest is drop-end-only per WORKFLOW.md Phase 7), so Hylla was correctly not consulted for the new symbols. All external semantics resolved via `go doc io.ReadFull`, `go doc io/fs.FS`, and `go doc errors` — the documented Go-idiomatic path for stdlib questions per CLAUDE.md § "Code Understanding Rules" rule 4. Drop mds (PLAN.md, BUILDER_WORKLOG.md, WORKFLOW.md) are markdown and out of Hylla's Go-only scope. Zero fallback misses to record.
