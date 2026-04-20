# DROP_2 — Builder Worklog

Append a `## Unit N.M — Round K` section per build attempt. See `main/drops/WORKFLOW.md` § "Phase 4 — Build (per unit)" for what each section should contain.

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
