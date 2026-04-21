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
