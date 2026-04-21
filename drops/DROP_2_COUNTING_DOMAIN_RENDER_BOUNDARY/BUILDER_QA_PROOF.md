# DROP_2 — Builder QA Proof

Append a `## Unit N.M — Round K` section per QA attempt. See `main/drops/WORKFLOW.md`.

## Unit 2.4 — Round 1

- **QA agent:** go-qa-proof-agent
- **Verdict:** pass
- **Verified acceptance bullets:**
    - `testdata/` dir + stable fixture — `main/cmd/rak/testdata/hello.txt` (29 bytes, `wc -c` confirmed).
    - F12 fixture coverage — `od -c` confirms 2 `\n` (multi-line), 5 whitespace-delimited tokens (multi-word), `é` (0xC3 0xA9) + `ï` (0xC3 0xAF) multi-byte UTF-8 runes; Bytes=29 > Chars=27.
    - Opens via `os.Open(filepath.Join("testdata", "hello.txt"))` — `main/cmd/rak/integration_test.go:48`, `:89`.
    - Wires via `cmd.SetIn(file)` + `cmd.SetOut(&out)` through `newRootCmd()` — `integration_test.go:55-58`, `:96-99`.
    - Covers both format paths — `--format=human` at `integration_test.go:59`; `--format=json` at `integration_test.go:100`.
    - Counts derivation verified by walking `counting.Count` semantics (`counting.go:36-72`) against fixture bytes: Bytes=29 (sum UTF-8 widths), Lines=2 (`\n` count), Words=5 (IsSpace-delimited tokens), Chars=27 (rune count).
    - JSON byte-exact snapshot `{"Bytes":29,"Lines":2,"Words":5,"Chars":27}\n` — `integration_test.go:107`; matches stdlib `encoding/json.Encoder.Encode` output with `Counts` declaration-order fields and no struct tags.
    - PLAN.md Unit 2.4 state = `done` at `main/drops/DROP_2_COUNTING_DOMAIN_RENDER_BOUNDARY/PLAN.md:131`.
    - BUILDER_WORKLOG.md Unit 2.4 Round 1 at TOP (`:5`, above Unit 2.3 at `:36`).
    - F4 pin held — grep `json:` in `internal/counting/counting.go` returns no matches.
    - F9 pin held — grep `os.Stdin` in `cmd/rak/root.go` returns no matches.
    - F11 pin held — `cmd/rak/root_test.go` = 108 LOC (≤ 150); untouched by Unit 2.4 (builder split into new `integration_test.go` instead of extending `root_test.go`).
- **Mage targets run:**
    - `mage build` → pass.
    - `mage test` → pass (cmd/rak, internal/counting, internal/render all ok under `-race`).
    - `mage lint` → pass (0 issues).
    - `mage ci` → pass (0 issues + tests green).
- **Findings:** none.
- **Hylla Feedback:** None — Unit 2.4 touched only a non-Go fixture + a new `_test.go` in a package changed since last ingest, so per CLAUDE.md § "Code Understanding Rules" rule 2 evidence routed through `Read` / `Grep` / mage stdout. No miss to record.

## Unit 2.3 — Round 1

- **QA agent:** go-qa-proof-agent
- **Verdict:** pass
- **Verified acceptance bullets:**
    - `runRoot` factored at `cmd/rak/root.go:49-77`; `RunE` closure at `:31-33` captures closure-local `format` var at `:20` (not package-level — test isolation pin).
    - `selectRenderer` at `cmd/rak/root.go:84-93` maps `"auto"`/`"human"` → `render.NewHumanRenderer()`, `"json"` → `render.NewJSONRenderer()`, default → wrapped `invalid --format %q` error.
    - `cobra.MaximumNArgs(1)` at `cmd/rak/root.go:30`. `len(args)==1` rejected inside `runRoot` (`:55-61`) with Drop-3 error.
    - F9 pin: `c.InOrStdin()` at `cmd/rak/root.go:63`; `grep os\.Stdin` on root.go → zero hits.
    - F4 pin: `grep json:` on `internal/counting/counting.go` → zero hits. `Counts` tagless.
    - F11 pin: `root_test.go` = 108 LOC ≤ 150. `root.go` = 93 LOC ≤ 150.
    - Error wraps exact: `fmt.Errorf("count input: %w", err)` at `:65`; `fmt.Errorf("render counts: %w", err)` at `:74`.
    - `--format` flag: `StringVarP(&format, "format", "f", "auto", "output format: auto | human | json")` at `cmd/rak/root.go:36-42`.
    - 4 required tests present: `TestRootCmd_ReadsStdin_RendersHumanDefault` (`root_test.go:15`), `TestRootCmd_FormatJSON` (`:43`), `TestRootCmd_InvalidFormat` (`:68`), `TestRootCmd_RejectsPathArg` (`:91`). Each calls `newRootCmd()` fresh; `t.Parallel()` at function level.
    - PLAN.md Unit 2.3 state: `done` (line 105).
    - BUILDER_WORKLOG.md Unit 2.3 Round 1 entry at top (line 5), above Unit 2.2 entry.
- **Mage targets run:**
    - `mage build` → pass.
    - `mage test` → pass (cmd/rak, internal/counting, internal/render all green under `-race`).
    - `mage lint` → pass (0 issues).
    - `mage ci` → pass (0 issues + tests green).
- **Findings:** none. PLAN.md acceptance bullet 1 suggests `ctx := c.Context()`; builder used `_ = c.Context()` — semantically identical placeholder; not a finding (builder documents rationale in worklog).
- **Hylla Feedback:** None — task touched files either changed-since-last-ingest (`cmd/rak/root.go`) or new (`cmd/rak/root_test.go`) per CLAUDE.md § "Code Understanding Rules" rule 2. Evidence via `Read` + `Grep` + mage stdout. No Hylla miss to record.

## Unit 2.2 — Round 1

- **QA agent:** go-qa-proof-agent
- **Verdict:** pass
- **Verified acceptance bullets:**
    - `Renderer` interface at `internal/render/render.go:21-25` with single method `Render(w io.Writer, counts counting.Counts) error`. Package doc + interface doc + method doc all start with the identifier name per naming rule 11.
    - `NewHumanRenderer` at `internal/render/human.go:36-44` uses `laslig.New(w, laslig.Policy{Format: FormatAuto, Style: StyleAuto})`. Printer constructed per-call inside `Render` (`human.go:60-66`), not cached — satisfies "TTY detection against the real writer" design note.
    - `newHumanRendererWithMode(mode laslig.Mode) Renderer` at `human.go:50-55` (unexported test variant); `Render` dispatches to `laslig.NewWithMode(w, h.mode)` when `useExplicitMode` is true.
    - `NewJSONRenderer` at `json.go:23-25` returns a renderer whose `Render` uses `json.NewEncoder(w).Encode(counts)` with wrapped error. No laslig import in `json.go`.
    - `TestHumanRenderer_SnapshotPlain` uses `newHumanRendererWithMode(testHumanMode)` where `testHumanMode = laslig.Mode{Format: FormatPlain, Styled: false, Width: 80}`. `TestHumanRenderer_TablePlain` covers zero / small / large tuples.
    - `TestJSONRenderer_Snapshot` asserts exact `{"Bytes":12,"Lines":1,"Words":2,"Chars":12}\n`. `TestJSONRenderer_Table` covers zero / small / large tuples.
    - F3 env invariant: no `os.Setenv` / `os.Unsetenv` in `render_test.go`; only a comment documenting the invariant.
    - `main/go.mod` lists `github.com/evanmschultz/laslig v0.2.4` in direct `require` block (no `// indirect`). Transitive indirect deps in the indirect block.
    - `main/go.sum` has laslig v0.2.4 h1 + /go.mod h1 entries.
    - F4 pin: `grep -n 'json:' internal/counting/counting.go` → zero hits. `Counts` still tagless.
    - PLAN.md Unit 2.2 state flipped to `done`.
    - BUILDER_WORKLOG.md Unit 2.2 Round 1 entry at top, above Unit 2.1 entry.
- **Mage targets run:**
    - `mage build` → pass (silent success).
    - `mage test` → pass; `internal/render` green under `-race`.
    - `mage ci` → `0 issues.` + tests green end-to-end.
- **Findings:** none. F3 env-independence, F4 no-json-tags, per-call printer construction all held.
- **Hylla Feedback:** None — new `internal/render/*.go` files not in last ingest; laslig is external. Evidence via `Read` + `Grep` + mage stdout + laslig v0.2.4 module cache.

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
