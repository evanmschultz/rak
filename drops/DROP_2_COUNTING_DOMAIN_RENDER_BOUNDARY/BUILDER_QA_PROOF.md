# DROP_2 — Builder QA Proof

Append a `## Unit N.M — Round K` section per QA attempt. See `main/drops/WORKFLOW.md`.

## Unit 2.0 — Round 1

- **QA agent:** go-qa-proof-agent
- **Verdict:** pass
- **Verified acceptance bullets:**
    - Signature `func AddDep(module string) error` — verified at `main/magefile.go:88` (C1 fork: no `context.Context` — matches every other target in the file).
    - Go doc comment starts with identifier name — `main/magefile.go:85` `// AddDep runs `go get <module>` to add or update a Go module dependency.` (CLAUDE.md naming rule 11).
    - Body uses `sh.RunV("go", "get", module)` — verified at `main/magefile.go:89` (A1 fork: no `go mod tidy`).
    - Error wrap with `%w` — `fmt.Errorf("mage addDep %s: %w", module, err)` at `main/magefile.go:90`. Wrapping present; pattern is consistent with `Build`/`Test`/`Lint`/etc. neighbours.
    - No new imports — `git diff magefile.go` shows only the new function block added; existing `sh` import at line 18 reused.
    - CLAUDE.md § "Build Verification" mage targets table row added — verified at `main/CLAUDE.md:210` `| mage addDep <module> | go get <module> | when adding a new Go dep (from Drop 2 onward) |`, placed between `mage lint` and `mage ci`.
    - No invocation of `mage install` — target body is two lines (`sh.RunV` + error wrap); does not reference `Install` or shell to `go install`.
    - Role-boundary (D3): builder touched only `magefile.go` + `CLAUDE.md`; CLAUDE.md edit is markdown documenting a Go-coupled change, permissive per role-boundary text.
- **Mage targets run:**
    - `mage -l` from `main/` → pass; `addDep` target surfaces (mage lowercases PascalCase `AddDep` to `addDep`, matching the documented `mage addDep <module>` invocation).
    - `mage build` from `main/` → pass (silent = success; magefile compiles under `//go:build mage`).
    - `mage ci` from `main/` → pass (gofumpt clean, `go vet` clean, golangci-lint 0 issues, tests green).
    - `mage addDep github.com/magefile/mage@v1.17.1` from `main/` → exit 0 (no-op as expected; mage already pinned at `v1.17.1` in `go.mod:7`).
- **Git diff check:** `git diff main/go.mod main/go.sum` → empty (no bytes emitted after the acceptance invocation). B1 fork precision hit.
- **Findings:** none. Implementation matches every Round 2 fork decision (A1 / B1 / C1 / D3) and every acceptance bullet.
- **Hylla Feedback:** None — Hylla answered everything needed. (Verification relied on `git diff` + direct `Read` against `magefile.go` and `CLAUDE.md` — both non-Hylla substrates by project rule: `magefile.go` is build-tagged `//go:build mage` and not part of the main import DAG, and `CLAUDE.md` is markdown. No Hylla query was required or attempted.)
