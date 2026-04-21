# DROP_3 — Builder Worklog

Append a `## Unit N.M — Round K` section per build attempt. See `main/drops/WORKFLOW.md` § "Phase 4 — Build (per unit)" for what each section should contain.

## Unit 3.0 — Round 1

- **Builder:** go-builder-agent
- **Started:** 2026-04-21 18:06 UTC
- **Files touched:** `main/go.mod`, `main/go.sum`, `main/drops/DROP_3_DIRECTORY_WALK_GITIGNORE_DEPTH/PLAN.md` (state flip only), `main/drops/DROP_3_DIRECTORY_WALK_GITIGNORE_DEPTH/BUILDER_WORKLOG.md` (this append).
- **Mage targets run:**
  - `mage addDep github.com/sabhiram/go-gitignore` → `go: added github.com/sabhiram/go-gitignore v0.0.0-20210923224102-525f6e181f06` (pass).
  - `mage addDep github.com/bmatcuk/doublestar/v4` → `go: added github.com/bmatcuk/doublestar/v4 v4.10.0` (pass).
  - `mage build` → clean (no output, exit 0).
  - `mage test` → all three existing test packages pass (`cmd/rak`, `internal/counting`, `internal/render`; cached because nothing Go-level changed).
- **Notes:**
  - `github.com/bmatcuk/doublestar/v4` resolved to the tagged release `v4.10.0` — a real semver tag.
  - `github.com/sabhiram/go-gitignore` has never cut a git tag, so Go's module resolver picked the latest `master` commit as pseudo-version `v0.0.0-20210923224102-525f6e181f06`. This is the expected shape for a tag-less module and is "the latest stable" the acceptance criterion asks for. The PLAN.md "latest stable tags" wording should be read as "latest stable resolver choice" for this case — if the planner wants a stricter tag-only policy, that's a Phase 3 discussion, not a Unit 3.0 blocker.
  - **Transitive-dep check (acceptance criterion):** neither library pulled a compiled transitive dep. `go.sum` delta added two `h1:` entries (the two target modules) plus four `/go.mod`-only entries that belong to `go-gitignore`'s own test-dep graph (`davecgh/go-spew v1.1.0`, `stretchr/objx v0.1.0`, `stretchr/testify v1.6.1`, `gopkg.in/yaml.v3 v3.0.0-20200313102051`). `/go.mod`-only lines are module-graph closure records, not compiled dependencies — they are never downloaded as source nor linked into rak. `go.mod` itself gained exactly two new `// indirect` lines, the two target modules.
  - Both modules are `// indirect` because no rak source imports them yet. Unit 3.1 (`internal/ignore`) will import `sabhiram/go-gitignore` and `bmatcuk/doublestar/v4`, at which point they flip from `// indirect` to direct requires. This is the documented Drop 2 workflow — `mage addDep` deliberately does not run `go mod tidy`, so unused deps sit in go.mod until the importing code lands.
  - No Go code written in this unit per spec.

## Unit 3.1 — Round 1

- **Builder:** go-builder-agent
- **Started:** 2026-04-21 (post-3.0)
- **Files touched:**
  - `main/internal/ignore/ignore.go` (new) — `Matcher` interface + `GitignoreRoot` + `New()` composite constructor + unexported `compositeMatcher`.
  - `main/internal/ignore/gitignore.go` (new) — wraps `github.com/sabhiram/go-gitignore`, hierarchical per-root scoping + `isDir` trailing-slash probe for dir-only patterns.
  - `main/internal/ignore/glob.go` (new) — wraps `github.com/bmatcuk/doublestar/v4.Match`, validates patterns up front via `doublestar.ValidatePattern`.
  - `main/internal/ignore/ignore_test.go` (new) — 12 table-driven test functions covering every acceptance clause (empty / gitignore-only / include-only / exclude-only / all-three / negation / dir-only / double-star / precedence-wins × 3 directions / `--no-gitignore` equivalence / invalid-glob error paths).
  - `main/drops/DROP_3_.../PLAN.md` (state flip `todo → in_progress → done` only).
  - `main/drops/DROP_3_.../BUILDER_WORKLOG.md` (this append).
- **Mage targets run:**
  - `mage test` → all four packages green (`cmd/rak`, `internal/counting`, `internal/ignore`, `internal/render`); race detector on.
  - `mage lint` → `0 issues.` (go vet + golangci-lint both clean on new package).
  - `mage build` → clean (no output, exit 0).
  - `mage ci` → green end-to-end: gofumpt-clean, lint-clean, tests green.
- **Design notes:**
  - **F1 polarity confirmed:** `go-gitignore.GitIgnore.MatchesPath(p) bool` returns `true` when the path is ignored, matching our `Matcher.Match(...) bool` "true = drop" convention. No polarity flip needed in the wrapper.
  - **F2 + F3 precedence** encoded as a short-circuit chain in `compositeMatcher.Match`: exclude → gitignore → include. Each stage short-circuits on hit; `hasIncludes` is stored once at construction so empty-include "allow everything" stays O(1).
  - **Dir-only patterns** (`node_modules/`): `go-gitignore` does NOT match the bare string `node_modules` against `node_modules/` pattern — it only matches children (`node_modules/x`). To make the F1 contract meaningful for directories, the wrapper appends `/` to the probe path when `isDir=true`. A runtime probe (four cases: `node_modules/`, `node_modules/.probe`, `foo/node_modules/`, `foo/node_modules/.probe`) showed the library handles trailing-slash probes correctly; see the test case `dir_only_pattern_matches_dir` which asserts this round-trip.
  - **Hierarchical scope (F8):** `scopePath(relPath, dir)` strips the root's directory prefix before handing the path to the compiled `GitIgnore`. Empty `dir` = walk root (no scoping, everything is in-scope). Equal `relPath == dir` = root dir itself. Sibling / out-of-scope paths are silently skipped. Covered by `scoped_rule_hits_inside_scope` / `scoped_rule_misses_outside_scope` / `scoped_rule_misses_in_sibling` cases.
  - **Doublestar API choice:** used `doublestar.Match` (forward-slash split on all platforms, per doublestar's own `Glob` docs) not `doublestar.PathMatch` (OS separator). Matches C6 and the planner's explicit direction in the unit acceptance. Validated via `doublestar.ValidatePattern` at construction so a malformed pattern fails at `New()` time, not at first use — the test pair `TestMatcher_InvalidGlob_Include` / `TestMatcher_InvalidGlob_Exclude` asserts the wrapped error unwraps to `doublestar.ErrBadPattern` via `errors.Is`.
  - **Semantic surprises worth pinning:**
    - `doublestar.Match("**/vendor/**", "vendor")` returns `true` — the leading `**/` accepts zero path components, so a `--exclude '**/vendor/**'` also drops the bare `vendor` dir itself. The test case `vendor_dir_also_matched` pins this as documented behavior.
    - `go-gitignore` with pattern `**/vendor` matches **all files underneath** a matched vendor dir, not just the dir itself — git's standard implicit-directory-ignore behavior. The test case `double_star_matches_children` asserts this.
    - `doublestar.Match("*.go", "sub/foo.go")` returns `false` (the `*` does not cross `/`). Users wanting recursive matching need `**/*.go`. Pinned by the `go_in_subdir_dropped` case in `TestMatcher_IncludeOnly`.
    - `*.go + !foo.go` gitignore ordering: `go-gitignore` honors the negation-re-includes rule correctly (probe confirmed `foo.go → not ignored`, `bar.go → ignored`). `TestMatcher_GitignoreOnly/negation_reincludes` covers this.
  - **No disk IO** in this package — `GitignoreRoot.Patterns` is consumed as-is. Comments (`# ...`) and blank lines are passed straight through to `go-gitignore.CompileIgnoreLines`, which handles them per the gitignore spec.
  - **No `--no-gitignore` flag parsing** here — the unit just tolerates the zero-gitignore case (nil or empty roots slice), and `TestMatcher_NoGitignore_EquivalentToEmptyRoots` asserts that behavior. Cobra wiring lands in 3.5.
  - **Naming / visibility:** `compositeMatcher`, `gitignoreMatcher`, `globMatcher`, `gitignoreRule`, `newGitignoreMatcher`, `newGlobMatcher`, `scopePath` are all unexported. Only `Matcher`, `GitignoreRoot`, and `New` cross the package boundary. Every exported identifier has a doc comment starting with the identifier name per CLAUDE.md § "Go-Idiomatic Naming Rules" rule 11.
  - Runtime probe of `go-gitignore` and `doublestar.Match` was done in a throwaway `/tmp/gi_probe` module — zero rak-repo artifacts, zero `go get` against `main/go.mod`.

## Hylla Feedback

N/A — task touched non-Go files only for reads (WORKFLOW.md, PLAN.md, magefile.go) and produced new Go files for a package that did not exist in the last ingest. External-library semantics for `go-gitignore` and `doublestar/v4` were resolved via `go doc` + a scratch `main.go` probe, which is the documented fallback path for third-party APIs (Context7 first, then `go doc`). Hylla was correctly not consulted because there was nothing yet in rak's ingest for `internal/ignore/` to return.
