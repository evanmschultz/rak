# Contributing to rak

Thanks for your interest! PRs are welcome.

## Quick start

```sh
git clone https://github.com/evanmschultz/rak
cd rak
mage -l           # list all build/test/lint targets
mage ci           # format-check + lint + test-with-race (the pre-push gate)
```

## Workflow

1. Branch off `main`: `git checkout -b feat/your-thing` (or `fix/`, `docs/`, etc.).
2. Make your changes. Tests live alongside source (`*_test.go`).
3. Run `mage ci` locally — it must pass before pushing.
4. Push your branch and open a PR: `gh pr create`.
5. CI must pass; merge via squash.

## Commit format

Conventional-commit subject lines: `type(scope): message`.

- Types: `feat`, `fix`, `refactor`, `chore`, `docs`, `test`, `ci`, `style`, `perf`.
- All lowercase except proper nouns / acronyms.
- Subject-line only — no body, no bullet lists. The diff is the body; the subject is the human summary.
- No trailing period. Under ~72 chars when possible. If it won't fit, split the change.

## Build targets

Run `mage -l` for the full list:

- `mage build` — compile check (`go build ./...`); does **not** produce a local binary.
- `mage test` — `go test -race ./...`.
- `mage format` — apply gofumpt.
- `mage lint` — `go vet` + `golangci-lint`.
- `mage ci` — pre-push gate (format-check + lint + test + coverage).
- `mage coverage` — coverage on `internal/...` (70% floor enforced).
- `mage run -- <args>` — run rak from source for local testing. Args after `--` pass through to rak.
- `mage install` — install the built binary to `$GOBIN` (the from-source install path for end users).
- `mage addDep <module>` — add a Go module dependency.

Never invoke raw `go test` / `go build` / `gofumpt` / `golangci-lint` directly — always go through mage so the build flags stay consistent.

## Testing changes locally

Use `mage run -- <args>` to test changes from source without installing:

```sh
mage run -- --version            # check version
mage run -- ./internal           # smoke test on a path
mage run -- --json . | jq '.'    # JSON output piped through jq
mage run -- --help               # full help text
```

`mage build` only verifies the code compiles; it does not emit a binary in the working tree. If you want a permanent local binary on your `$PATH`, run `mage install` — that puts the built binary in `$GOBIN` (typically `~/go/bin/`).

**Mage CLI quirk** — when `<args>` after `--` contains flag-style tokens (e.g. `--version`, `--help`), mage prints `Unknown target specified: "--"` and exits with code 2 *after* rak has already run successfully. The flag args still reach rak correctly; this is a mage CLI parsing artifact, not a rak bug. Non-flag args (paths) exit cleanly with code 0.

## Code style

- Idiomatic Go. `gofumpt` enforces format; `golangci-lint` catches the rest.
- Doc comments on every exported identifier, starting with the identifier name.
- Errors wrap with `fmt.Errorf("...: %w", err)`. Inspect with `errors.Is` / `errors.As` — never string-match.
- Tests are table-driven when input has variants. Behavior-oriented assertions.

## Adding a detected language

The most common contribution category. Steps:

1. Add a `Language` constant in `internal/lang/lang.go`.
2. Add extension entries (and filename / shebang entries where applicable).
3. Add comment-handling rules in `internal/lang/split.go`.
4. Extend `internal/lang/lang_test.go` and `internal/lang/split_test.go`.
5. Update the "Languages detected" sentence in `README.md`.

## Agent workflow (optional)

v0.1.0 was built using a multi-agent cascade (planner → builder → QA proof → QA falsification → orchestrator). The methodology lives in [`CLAUDE.md`](./CLAUDE.md) and [`drops/WORKFLOW.md`](./drops/WORKFLOW.md). You don't need to follow it to contribute — direct PRs are fine — but it's there if you're curious.

## License

By contributing, you agree your changes are under the [MIT License](./LICENSE).
