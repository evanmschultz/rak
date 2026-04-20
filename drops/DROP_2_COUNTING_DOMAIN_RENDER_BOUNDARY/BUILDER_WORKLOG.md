# DROP_2 — Builder Worklog

Append a `## Unit N.M — Round K` section per build attempt. See `main/drops/WORKFLOW.md` § "Phase 4 — Build (per unit)" for what each section should contain.

## Unit 2.1 — Round 1

- **Builder:** go-builder-agent
- **Started:** 2026-04-20
- **Files touched:**
  - `main/internal/counting/counting.go` (new) — package doc + `Counts` struct (fields in declaration order `Bytes, Lines, Words, Chars`, no `json:` tags per F4 pin) + exported `Count(io.Reader) (Counts, error)`. Semantics verbatim from the old unexported `count` in `cmd/rak/root.go`: `bufio.NewReader` + `ReadRune` loop, `unicode.IsSpace` word split, `io.EOF` clean-exit. Go doc comments on the package, the struct, every field, and `Count` per naming rule 11.
  - `main/internal/counting/counting_test.go` (new) — table-driven `TestCount` with all 7 acceptance tuples (empty / "hello" / "hello\\n" / "hello world" / "hello world\\nfoo bar\\n" / "héllo\\n" UTF-8 divergence / "a\\r\\nb\\r\\n" CRLF F5 pin). Subtests via `t.Run` with descriptive names; fixtures via `strings.NewReader`; no `testdata/`. `t.Parallel()` at both function + subtest level to surface races via the race detector.
  - `main/cmd/rak/root.go` — removed `Counts` struct declaration and unexported `count` function. Imports shrunk to `fmt` + `github.com/spf13/cobra` (dropped `bufio`, `io`, `unicode`). `RunE` body untouched: still returns `"not implemented — see drop 2"` — Unit 2.3 is the one that rewires to `counting.Count`. `Long` description updated to reflect Drop 2's current state (counting lifted, render landing, wiring in 2.3).
  - `main/.golangci.yml` — F2 fold: shrunk to minimal `version: "2"`. Removed the rationale comment block (lines 3-17) and the `linters.exclusions.rules` entry exempting `cmd/rak/root.go` from `unused` (lines 19-24). With `count` + `Counts` moved out of `cmd/rak`, the exclusion is orphaned.
  - `main/magefile.go` — F3 fold: line 5 doc comment `"nine canonical targets"` → `"ten canonical targets"` (Unit 2.0 build-QA falsification advisory — `AddDep` made it ten).
  - `main/drops/DROP_2_COUNTING_DOMAIN_RENDER_BOUNDARY/PLAN.md` — Unit 2.1 state flip `todo` → `in_progress` at start, `in_progress` → `done` at end.
- **Mage targets run:**
  - `mage build` — pass (no output; all packages compile after the lift).
  - `mage test` — pass; `internal/counting` tests green under `-race` (1.266s first run). `cmd/rak` has no test files yet (expected — tests land in 2.3).
  - `mage lint` — pass, 0 issues (with the shrunk `.golangci.yml`; `unused` linter no longer fires on `cmd/rak/root.go` because `count` + `Counts` are no longer declared there).
  - `mage ci` — pass end-to-end (gofumpt clean, `go vet` clean, `golangci-lint` 0 issues, tests green from cache).
- **Design notes:**
  - RunE rewire choice: kept `RunE` returning the "not implemented" stub rather than calling `counting.Count` here. Unit 2.1's acceptance prose explicitly says "err on the side of the smallest change that keeps `mage build` green" and calls out that Unit 2.3 is the one that adds `counting.Count` + render wiring. Dropping `io` from imports is consistent with acceptance ("2.3 re-adds `io` for stdin"). Smallest-change path.
  - `counting_test.go` uses `t.Parallel()` at both function and subtest level. The race detector (`mage test` runs `-race` unconditionally per CLAUDE.md § "Tests") then exercises any shared state. There is none — `Count` is pure and each subtest owns its own `strings.NewReader` — but explicit parallelism + race detector is the idiomatic Go belt-and-suspenders.
  - Hand-verified the 7 tuples by walking the `count` algorithm: `é` is 2 bytes (0xC3 0xA9) so `"héllo\n"` is Bytes=7 Chars=6 Lines=1 Words=1; `"a\r\nb\r\n"` has `\r` recognized as whitespace by `unicode.IsSpace` so "a" and "b" each become words, but Lines only increments on `\n` → Lines=2 Words=2. Test execution confirmed all 7 subtests pass.
  - `.golangci.yml` deliberately does not list any linters — relying on golangci-lint v2 default linter set per the original config's comment ("Everything else stays on the golangci-lint default linter set"). The minimal `version: "2"` file preserves that default-set behavior.
- **Hylla Feedback:**
  - N/A — Hylla is Go-only and rak's committed snapshot (`github.com/evanmschultz/rak@main`) lacks both the new `internal/counting/` package (doesn't exist yet) and reflects the pre-Unit-2.0 state of `magefile.go`. Source of truth for all references was local `Read` on `cmd/rak/root.go`, `magefile.go`, `.golangci.yml`, and the drop's `PLAN.md` — none of which are candidates for Hylla queries in this unit (non-Go file + changed-since-ingest files). No Hylla miss to record.

## Unit 2.0 — Round 1

- **Builder:** go-builder-agent
- **Started:** 2026-04-20 02:02
- **Files touched:**
  - `main/magefile.go` — added `AddDep(module string) error` target (9 lines + doc comment, placed between `Install` and `Run` for alphabetical-ish grouping with other wrappers).
  - `main/CLAUDE.md` — added `mage addDep <module>` row to the Build Verification mage-targets table, placed between `mage lint` and `mage ci`.
- **Mage targets run:**
  - `mage -l` — pass (confirmed `addDep` target registered; mage lowercases PascalCase `AddDep` to `addDep` in listing, matching the documented `mage addDep <module>` invocation).
  - `mage build` — pass (magefile compiles, no output).
  - `mage ci` — pass (gofumpt clean, `go vet` clean, `golangci-lint` 0 issues, tests pass).
  - **Acceptance:** `mage addDep github.com/magefile/mage@v1.17.1` — exit 0, `git diff go.mod go.sum` empty (mage already pinned at v1.17.1, so `go get` is a no-op per Drop 2 Phase 3 decision B1).
- **Notes:**
  - Signature `func AddDep(module string) error` per decision C1 — no `context.Context`. Matches the shape of every other mage target in the file (all `func() error` or the new `func(string) error`).
  - Body is `sh.RunV("go", "get", module)` with `fmt.Errorf("mage addDep %s: %w", module, err)` wrap, mirroring the existing error-wrap style in `Build`, `Test`, etc.
  - No `go mod tidy` per decision A1 — callers handle tidy separately if needed, keeping this target a thin shell for "add one dep".
  - Doc comment starts with the identifier name per CLAUDE.md § "Go-Idiomatic Naming Rules" rule 11.
  - Paired `main/CLAUDE.md` edit landed in the same working tree change per decision D3 (markdown documenting a Go-coupled target).
