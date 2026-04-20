# DROP_2 — COUNTING_DOMAIN_RENDER_BOUNDARY

**State:** planning
**Blocked by:** —
**Paths (expected):** `main/cmd/rak/root.go` (lift `count` out), `main/internal/counting/` (new package), `main/internal/render/` (new package), `main/cmd/rak/testdata/` (integration fixture), plus per-package `*_test.go` files
**Packages (expected):** `github.com/evanmschultz/rak/cmd/rak`, `github.com/evanmschultz/rak/internal/counting` (new), `github.com/evanmschultz/rak/internal/render` (new)
**PLAN.md ref:** main/PLAN.md → `DROP_2_COUNTING_DOMAIN_RENDER_BOUNDARY` row
**Workflow:** main/drops/WORKFLOW.md
**Started:** 2026-04-19
**Closed:** —

## Scope

Lift the `count(io.Reader) (Counts, error)` primitive out of `cmd/rak/root.go` into a first-class `internal/counting` package with an exported `Count` function and `Counts` struct (bytes/lines/words/chars). Land `internal/render` as the laslig-backed rendering boundary with `NewHumanRenderer` / `NewJSONRenderer` constructors (no Format enum factory per decision 27(d)) and `Format{Human,JSON}` plumbing. Wire the root command to the new counting + render layer and auto-select renderer via laslig's TTY-vs-pipe detection. Ship counting table tests and render snapshot tests. **No walker, no language detection, no tokens, no summary rollup yet** — all deferred to later drops. Expected decomposition: ~5 units (2.1 counting / 2.2 render / 2.3 wire-up / 2.4 TTY-auto / 2.5 tests) per main/PLAN.md § "Expected Decomposition" lines 107–113.

## Planner

Scope confirmed with one adjustment to the expected decomposition: 4 units instead of 5. "TTY-vs-pipe auto-detect" is not a separate unit — `laslig.ResolveMode` (policy.go:211) handles detection natively when `NewHumanRenderer` is constructed with `Policy{Format: FormatAuto, Style: StyleAuto}`. Per-package tests ride with the package they cover (TDD-first) rather than a trailing test unit. Units 2.1 and 2.2 share no paths or packages and can run in parallel; 2.3 is the fan-in wire-up; 2.4 is the end-to-end integration seal.

**Dependency DAG (shortest-blocker form):**

```
2.1 ──▶ 2.3 ──▶ 2.4
         ▲
2.2 ─────┘
```

### Unit 2.1 — Lift counting primitive into `internal/counting`

- **State:** todo
- **Paths:** `main/internal/counting/counting.go` (new), `main/internal/counting/counting_test.go` (new), `main/cmd/rak/root.go` (remove `Counts` struct + `count` function; imports shrink accordingly)
- **Packages:** `github.com/evanmschultz/rak/internal/counting` (new), `github.com/evanmschultz/rak/cmd/rak`
- **Blocked by:** —
- **Acceptance:**
    - `internal/counting/counting.go` defines exported `Counts` struct with fields `Bytes int64`, `Lines int64`, `Words int64`, `Chars int64` (Go doc comment on struct + each field starting with identifier name per CLAUDE.md naming rule 11).
    - `internal/counting/counting.go` defines exported `func Count(r io.Reader) (Counts, error)` with the same semantics as the current unexported `count` in `cmd/rak/root.go:42-78` (`bufio.NewReader` + `ReadRune` loop + `unicode.IsSpace` word split + `io.EOF` clean-exit). Go doc comment on `Count` starting with "Count …".
    - `cmd/rak/root.go` no longer declares `Counts` or `count` — both moved. `cmd/rak/root.go` imports shrink to remove `bufio` and `unicode` (the remaining imports are `fmt` and `github.com/spf13/cobra` — `io` can also go since `RunE` no longer uses it directly yet; 2.3 re-adds `io` for stdin).
    - `internal/counting/counting_test.go` ships a table-driven `TestCount` covering at minimum: empty reader (`""` → all zeros); single ASCII word no newline (`"hello"` → `{5,0,1,5}`); single ASCII word with newline (`"hello\n"` → `{6,1,1,6}`); multi-word single line (`"hello world"` → `{11,0,2,11}`); multi-word multi-line (`"hello world\nfoo bar\n"` → `{20,2,4,20}`); UTF-8 multi-byte rune (`"héllo\n"` → `Bytes=7, Chars=6, Lines=1, Words=1`); CRLF line endings (`"a\r\nb\r\n"` → verify the documented behavior — CR is whitespace per `unicode.IsSpace` so words split on it; Lines increments only on `\n`). Subtests via `t.Run` with descriptive names (CLAUDE.md naming rule 8).
    - Fixtures live in-memory via `strings.NewReader` / `bytes.NewReader`. No `testdata/` directory for this unit (CLAUDE.md § "Tests" — prefer in-memory fixtures).
    - `mage build` succeeds from `main/`.
    - `mage test` succeeds from `main/` with `-race` (mage target default). No test skips.

### Unit 2.2 — Land `internal/render` package with laslig dep re-add

- **State:** todo
- **Paths:** `main/internal/render/render.go` (new — `Renderer` interface + shared helpers), `main/internal/render/human.go` (new — laslig-backed human renderer), `main/internal/render/json.go` (new — stdlib encoding/json renderer), `main/internal/render/render_test.go` (new — snapshot tests for both renderers), `main/go.mod` (bootstrap-carve-out dep add), `main/go.sum` (bootstrap-carve-out dep add)
- **Packages:** `github.com/evanmschultz/rak/internal/render` (new)
- **Blocked by:** —
- **Acceptance:**
    - **Bootstrap carve-out (one-time)**: builder runs `go get github.com/evanmschultz/laslig@v0.2.4` from `main/` to re-add laslig as a direct dep (Drop 1.5's `go mod tidy` pruned it because no source imported it; Drop 1 PLAN.md Notes predicted this). Version pin `v0.2.4` is explicit (REFINEMENTS entry 4). Then `go mod tidy` from `main/` to normalize. Both commands run with default environment — no `GOPROXY` / `GOSUMDB` / checksum overrides (CLAUDE.md § "Go Development Rules" → "Dependencies").
    - After the dep add, `main/go.mod` `require` block lists `github.com/evanmschultz/laslig v0.2.4` as a direct (non-indirect) dep.
    - `internal/render/render.go` defines:
        - Exported `Renderer` interface with a single method (Go idiom — single-method interfaces end in `-er` per CLAUDE.md naming rule 5). Proposed minimal signature: `type Renderer interface { Render(w io.Writer, counts counting.Counts) error }`. This imports `github.com/evanmschultz/rak/internal/counting` — verifying 2.1 → 2.2 is the natural dep-DAG order (leaf → interior per CLAUDE.md § "Import DAG"), even though they're siblings in the drop DAG.
        - Go doc comment on `Renderer` and the method.
    - `internal/render/human.go` defines exported `func NewHumanRenderer() Renderer` (explicit constructor per decision 27(d) — no Format enum factory). Internal implementation uses `laslig.New(out, laslig.Policy{Format: laslig.FormatAuto, Style: laslig.StyleAuto})` inside `Render` so the printer is created per-call bound to the writer — this gives automatic TTY-vs-pipe selection for free (evidence: `laslig/policy.go:211` `ResolveMode` + `laslig/printer.go:39` `New`). Renders `counting.Counts` as a laslig `KV` block with labels "Bytes", "Lines", "Words", "Chars" (values formatted as decimal int64). Go doc comments.
    - `internal/render/json.go` defines exported `func NewJSONRenderer() Renderer`. Internal implementation uses stdlib `encoding/json.NewEncoder(w).Encode(counts)` — no laslig for JSON (decision 27(d) / 27(e); laslig's JSON format is a separate path not needed for Drop 2's minimal JSON output). Go doc comments.
    - `internal/render/render_test.go`:
        - `TestHumanRenderer_SnapshotPlain` — constructs a renderer via `NewHumanRenderer()`, calls `Render(buf, counting.Counts{Bytes: 12, Lines: 1, Words: 2, Chars: 12})` against a `*bytes.Buffer`. For snapshot determinism the test MAY need to swap the internal policy — so the implementation must expose either (a) a second unexported constructor that accepts a `laslig.Mode` for test use, OR (b) use `laslig.NewWithMode(out, laslig.Mode{Format: laslig.FormatPlain, Styled: false, Width: 80})` inside the renderer when the test sets a specific unexported option. Builder chooses mechanism — acceptance is that the test is deterministic across TTY / non-TTY CI environments (race detector already forces no-TTY in CI).
        - `TestJSONRenderer_Snapshot` — calls `Render(buf, counting.Counts{Bytes: 12, Lines: 1, Words: 2, Chars: 12})` and asserts exact output `{"Bytes":12,"Lines":1,"Words":2,"Chars":12}\n` (stdlib `json.Encoder.Encode` trails with `\n`).
        - Table-driven across at least three `Counts` values per renderer: zero-counts, small-counts, large-counts (to exercise formatting).
    - `mage build` succeeds from `main/`.
    - `mage test` succeeds from `main/` with `-race`.
    - `mage ci` succeeds from `main/` (gofumpt clean, vet clean, golangci-lint clean, tests pass).

### Unit 2.3 — Wire `cmd/rak/root.go` `RunE` to counting + render

- **State:** todo
- **Paths:** `main/cmd/rak/root.go` (rewrite `RunE`), `main/cmd/rak/root_test.go` (new or extended — arg/flag parsing + renderer-selection coverage)
- **Packages:** `github.com/evanmschultz/rak/cmd/rak`
- **Blocked by:** 2.1, 2.2
- **Acceptance:**
    - `cmd/rak/root.go` imports `github.com/evanmschultz/rak/internal/counting`, `github.com/evanmschultz/rak/internal/render`, and the `--format` flag pflag wiring; re-adds `io` for stdin piping and `os` for `os.Stdin` default (or uses `cmd.InOrStdin()` for cobra-idiomatic input indirection — builder chooses based on how `root_test.go` drives input).
    - `RunE` behavior:
        1. `ctx := c.Context()` (keep for future cancellation wiring — no-op today).
        2. Read `io.Reader` input from `cmd.InOrStdin()` (stdin) when `len(args)==0`; when `len(args)==1` return `fmt.Errorf("positional path argument not supported yet — walker lands in Drop 3; pipe input via stdin for now (got %q)", args[0])`. **This behavior is subject to Phase 3 dev approval — the alternative is "always read stdin and ignore args".**
        3. Call `counts, err := counting.Count(reader)`. Wrap on error: `return fmt.Errorf("count input: %w", err)` (CLAUDE.md § "Errors" — wrap at boundaries).
        4. Select renderer from `--format` flag value: `"human"` → `render.NewHumanRenderer()`; `"json"` → `render.NewJSONRenderer()`; `"auto"` (default) → `render.NewHumanRenderer()` (human renderer's internal laslig policy auto-resolves to plain non-styled when `cmd.OutOrStdout()` is not a TTY — evidence: `laslig/policy.go:211`).
        5. Call `renderer.Render(cmd.OutOrStdout(), counts)`. Wrap on error: `return fmt.Errorf("render counts: %w", err)`.
    - `--format` flag: cobra `StringVarP(&format, "format", "f", "auto", "output format: auto | human | json")`. Values validated in `PersistentPreRunE` or inline — invalid value returns wrapped error.
    - `cmd/rak/root.go` remains under ~150 LOC (CLAUDE.md § "Project Structure" file budget).
    - `cmd/rak/root_test.go` covers:
        - `TestRootCmd_ReadsStdin_RendersHumanDefault` — uses `cmd.SetIn(strings.NewReader("hello world\n"))` + `cmd.SetOut(&buf)`, calls `cmd.Execute()`, asserts buf contains human-format counts.
        - `TestRootCmd_FormatJSON` — `--format=json`, asserts buf contains the JSON counts.
        - `TestRootCmd_InvalidFormat` — `--format=xml`, asserts returned error is non-nil and mentions "format".
        - `TestRootCmd_RejectsPathArg` — invokes with one positional arg, asserts returned error mentions "Drop 3" (pending Phase 3 dev decision on this behavior).
    - `mage build` succeeds.
    - `mage test` succeeds with `-race`.
    - No test invokes `mage install` or any raw `go` command.

### Unit 2.4 — End-to-end integration test via `cmd/rak/testdata/`

- **State:** todo
- **Paths:** `main/cmd/rak/testdata/hello.txt` (new fixture — contents TBD by builder but stable and documented in test), `main/cmd/rak/integration_test.go` (new) OR `main/cmd/rak/root_test.go` (extended — builder's call based on file-size budget)
- **Packages:** `github.com/evanmschultz/rak/cmd/rak`
- **Blocked by:** 2.3
- **Acceptance:**
    - `cmd/rak/testdata/` directory created (Go stdlib `testdata` idiom — ignored by `go` tooling per `go help test`; CLAUDE.md § "Tests" → "Two-tier testdata rule" explicitly sanctions this as the single guaranteed fixture slot).
    - At least one fixture file (e.g. `hello.txt`) with known, stable content. Test asserts the exact expected `Counts` output for that content in both human and JSON formats.
    - Integration test reads the fixture via `os.Open(filepath.Join("testdata", "hello.txt"))`, wires it to the cobra command via `cmd.SetIn(file)`, captures stdout via `cmd.SetOut(&buf)`, runs `cmd.Execute()`, asserts buf matches an expected string.
    - Test covers both format paths: `--format=human` and `--format=json`.
    - Test runs green under `mage test` with `-race`.
    - `mage ci` succeeds from `main/` end-to-end (gofumpt / vet / golangci-lint / tests all green).

## Notes

**Unit count reshape (5 → 4).** The expected decomposition in main/PLAN.md lines 107–113 had a separate "2.4 TTY-auto" unit. Once laslig's API is in view (`ResolveMode` at `/Users/evanschultz/go/pkg/mod/github.com/evanmschultz/laslig@v0.2.4/policy.go:211`), TTY-vs-pipe detection is a single-line config choice inside `NewHumanRenderer` — not a unit's worth of work. Similarly, "2.5 tests" as a trailing unit violates TDD-first discipline; tests ship with each package. Result: 2.1 counting / 2.2 render / 2.3 wire-up / 2.4 integration test. Plan-QA falsification round may challenge this collapse; the alternative (keep 5 units) would leave 2.4 as a cosmetic re-export of 2.2's internal policy.

**Laslig dep re-add (critical).** `main/go.mod` and `main/go.sum` do **not** currently list `github.com/evanmschultz/laslig` at any version, direct or indirect. Drop 1.5's `go mod tidy` pruned it because no source file imported it yet — Drop 1's PLAN.md Notes explicitly predicted this. Unit 2.2's first acceptance step is the bootstrap carve-out: `go get github.com/evanmschultz/laslig@v0.2.4` + `go mod tidy`, invoked directly from `main/` with default environment (CLAUDE.md § "Go Development Rules" → "Bootstrap carve-out" — applies because no mage target yet wraps dep adds).

**Laslig version pin.** `v0.2.4` is the latest laslig present in the local module cache at `/Users/evanschultz/go/pkg/mod/github.com/evanmschultz/laslig@v0.2.4/` (alongside v0.2.2 and v0.2.3). Explicit version in acceptance addresses REFINEMENTS entry 4 (name tool versions explicitly).

**Laslig API used (v0.2.4 citations).**
- `func New(out io.Writer, policy Policy) *Printer` — `printer.go:39` — Unit 2.2's `NewHumanRenderer` uses this with `Policy{Format: FormatAuto, Style: StyleAuto}` for automatic TTY-vs-pipe selection.
- `func NewWithMode(out io.Writer, mode Mode) *Printer` — `printer.go:48` — available to render tests if explicit `Mode{Format: FormatPlain, Styled: false, Width: 80}` is needed for snapshot determinism (builder's call at 2.2 implementation time).
- `func ResolveMode(out io.Writer, policy Policy) Mode` — `policy.go:211` — the TTY detection primitive; called internally by `New`. Rak does not need to call this directly.
- Format / Style enums and Printer methods (Section/Record/KV/Paragraph/List/Notice/Table) all in `policy.go` + `types.go` + `printer.go`.

**Decision 27(d) reinforcement.** `internal/render` ships explicit `NewHumanRenderer` + `NewJSONRenderer` constructors. **No** `NewRenderer(format Format) Renderer` factory. Callers at `cmd/rak/root.go` switch on the `--format` flag and call the right constructor directly. This keeps the dep-DAG simple and avoids a premature enum.

**Decision 25 reinforcement.** `internal/counting.Count` takes `io.Reader` only. No `fileset.File`, no path strings, no direct filesystem access. The counting domain is pure stream math.

**No path positional argument handling yet.** Unit 2.3's `RunE` reads stdin when `len(args)==0`. When `len(args)==1` it returns an error directing the user to pipe input via stdin and noting that the walker lands in Drop 3. **This behavior is subject to dev approval during Phase 3 discussion.** The alternative is "always read stdin, silently ignore args"; the proposed stricter behavior surfaces the deferral instead of hiding it.

**Deferred to later drops (not Drop 2's scope):** walker + path traversal (Drop 3); gitignore / include / exclude matching (Drop 4); language detection + blank/comment/code split (Drops 5 + 6); summary rollup + sort (Drop 7); token counting (Drop 8); parallel walk (Drop 9). Each later drop's `PLAN.md` will re-plan its own boundary.

**Hylla Feedback.** N/A — Drop 2 planning leans on non-Go evidence (current `cmd/rak/root.go` contents, `go.mod` / `go.sum` state, laslig module cache files) and on live Go semantics via `Grep` / `Read` against the checkout and the module cache. Hylla was not queried as the primary source because (a) `cmd/rak/*.go` is the source of the code being lifted and current contents trump any committed snapshot, (b) laslig lives in the module cache and is not part of rak's Hylla-indexed artifact. No Hylla miss to record.
