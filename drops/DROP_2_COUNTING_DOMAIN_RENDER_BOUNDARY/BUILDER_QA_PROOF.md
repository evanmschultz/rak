# DROP_2 — Builder QA Proof

Append a `## Unit N.M — Round K` section per QA attempt. See `main/drops/WORKFLOW.md`.

## Unit 2.1 — Round 1

- **QA agent:** go-qa-proof-agent
- **Verdict:** pass
- **Verified acceptance bullets:**
    - `Counts` struct shape + field order `Bytes, Lines, Words, Chars int64` — `internal/counting/counting.go:18-30`. F4 pin held: grep for `json:` on that file returned no matches.
    - Go doc comments per naming rule 11 — package doc line 1, struct doc line 13, field docs lines 19/21/24/27, `Count` doc line 32 all begin with the identifier name.
    - `Count(r io.Reader) (Counts, error)` signature exact — `internal/counting/counting.go:36`.
    - Semantics parity with pre-2.1 `count` — compared new body (`counting.go:36-72`) to `git show 3cb4325:cmd/rak/root.go` pre-lift body: identical `bufio.NewReader` + `ReadRune` loop + `unicode.IsSpace` word split + `io.EOF` clean-exit. Line-for-line equivalent.
    - Table-driven `TestCount` with all 7 acceptance tuples — `internal/counting/counting_test.go:16-50` (empty / hello / hello\n / hello world / hello world\nfoo bar\n / héllo\n UTF-8 / a\r\nb\r\n CRLF). Subtests via `t.Run(tc.name, ...)` at line 54 with descriptive snake_case names. `t.Parallel()` at both function (line 9) and subtest (line 55) levels — belt-and-suspenders for the race detector.
    - `cmd/rak/root.go` no longer declares `Counts` or `count` — grep on `Counts|\bcount\b` returned no matches. Imports shrunk to `fmt` + `github.com/spf13/cobra` (lines 3-7); `bufio`, `io`, `unicode` dropped.
    - F2 fold — `.golangci.yml` reduced to a single line `version: "2"`. Rationale comment block + `unused` exclusion rule both gone.
    - F3 fold — `magefile.go:5` doc comment reads `"The ten canonical targets mirror the table"` (was "nine").
    - PLAN.md Unit 2.1 state flipped to `done` at `drops/DROP_2_COUNTING_DOMAIN_RENDER_BOUNDARY/PLAN.md:55`.
    - BUILDER_WORKLOG.md Unit 2.1 Round 1 entry at top, above the prior Unit 2.0 entry — most-recent-first ordering preserved.
- **Mage targets run:**
    - `mage build` from `main/` → pass (silent success).
    - `mage test` from `main/` → pass; `internal/counting` green under `-race` (cached), `cmd/rak` `[no test files]` (expected pre-Unit-2.3).
    - `mage lint` from `main/` → `0 issues.`
    - `mage ci` from `main/` → `0 issues.` + tests green end-to-end.
- **Findings:** none. Every acceptance bullet has file:line or command-output evidence. F2/F3/F4/F5 pins all held. Semantics preserved verbatim against the pre-lift `count` body in commit `3cb4325`.
- **Hylla Feedback:** None — in-scope artifacts were either non-Go (`.golangci.yml`, `magefile.go` under `//go:build mage`, PLAN.md, BUILDER_WORKLOG.md) or changed-since-last-ingest (`cmd/rak/root.go` post-lift, new `internal/counting/*`). Evidence via `Read` + `git diff` + `git show 3cb4325:cmd/rak/root.go`. No miss.

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
