# DROP_2 — COUNTING_DOMAIN_RENDER_BOUNDARY

**State:** building
**Blocked by:** —
**Paths (expected):** `main/magefile.go` (new `AddDep` target), `main/cmd/rak/root.go` (lift `count` out), `main/internal/counting/` (new package), `main/internal/render/` (new package), `main/cmd/rak/testdata/` (integration fixture), `main/CLAUDE.md` (mage targets table row), plus per-package `*_test.go` files
**Packages (expected):** `github.com/evanmschultz/rak/cmd/rak`, `github.com/evanmschultz/rak/internal/counting` (new), `github.com/evanmschultz/rak/internal/render` (new)
**PLAN.md ref:** main/PLAN.md → `DROP_2_COUNTING_DOMAIN_RENDER_BOUNDARY` row
**Workflow:** main/drops/WORKFLOW.md
**Started:** 2026-04-19
**Closed:** —

## Scope

Lift the `count(io.Reader) (Counts, error)` primitive out of `cmd/rak/root.go` into a first-class `internal/counting` package with an exported `Count` function and `Counts` struct (bytes/lines/words/chars). Land `internal/render` as the laslig-backed rendering boundary with `NewHumanRenderer` / `NewJSONRenderer` constructors (no Format enum factory per decision 27(d)) and `Format{Human,JSON}` plumbing. Wire the root command to the new counting + render layer and auto-select renderer via laslig's TTY-vs-pipe detection. Ship counting table tests and render snapshot tests. Also introduce a `mage addDep <module>` target so laslig's re-add (and every future dep add from Drop 2 onward) routes through mage per CLAUDE.md § "Go Development Rules" → "Dependencies". **No walker, no language detection, no tokens, no summary rollup yet** — all deferred to later drops. Expected decomposition: 5 units (2.0 mage addDep target / 2.1 counting / 2.2 render / 2.3 wire-up / 2.4 integration test). The expected-decomposition table in main/PLAN.md lines 107–113 listed a different 5-unit split (2.4 TTY-auto / 2.5 tests); Round 2 restructures to prepend a dep-tooling unit (2.0) and fold TTY-auto + tests into the packages they belong to — see § "Notes".

## Planner

Scope confirmed with a Round 2 reshape: **5 units** — new Unit 2.0 (`mage addDep` target) prepended; Units 2.1 – 2.4 otherwise mirror Round 1 with sharpened acceptance bullets and a strict-DAG `blocked_by` chain. The reshape folds two Round 1 plan-QA falsification findings (P1 DAG correctness, F13 `blocked_by` missing on 2.2) and the two dev decisions (Decision A stdin error, Decision B `mage addDep` tooling instead of bootstrap carve-out) into a decomposition where each unit has a single, yes/no-verifiable focus.

**Atomicity rationale for Unit 2.0 as its own unit (not folded into 2.2):** the `AddDep` target lands with no source-file changes outside `main/magefile.go` + a one-row CLAUDE.md table edit — build-QA verifies it by running `mage addDep github.com/magefile/mage` and asserting it's a no-op (mage is already direct in go.mod). Folding it into 2.2 would couple "tooling correctness" with "package authoring + dep re-add" in one unit, muddying build-QA. Keeping it standalone also means a future dep add (tiktoken in Drop 7, errgroup in Drop 8) can reuse the target without ever replaying Drop 2 build-QA history.

**Dependency DAG (strict import-DAG order with 2.0 prepended):**

```
2.0 ──┐
      ├──▶ 2.2 ──▶ 2.3 ──▶ 2.4
2.1 ──┘    ▲
           │
      (2.1 also feeds 2.3 directly;
       2.2 imports internal/counting,
       so 2.1 → 2.2 is the natural
       import-DAG edge too)
```

Minimal cut: `2.0 → 2.2 → 2.3 → 2.4`, with `2.1` parallel to `2.0` (2.1 has no laslig dep — pure stdlib). Natural serial build order: `(2.0 ∥ 2.1) → 2.2 → 2.3 → 2.4`. `2.2` is blocked by both `2.0` (needs the `mage addDep` target to add laslig) and `2.1` (`render` imports `internal/counting` per CLAUDE.md § "Import DAG", so the compile order is leaf → interior). `2.3` is blocked by `2.1` (`RunE` calls `counting.Count`) and `2.2` (`RunE` selects a `render.Renderer`). `2.4` is blocked by `2.3`.

### Unit 2.0 — Add `mage addDep <module>` target

- **State:** todo
- **Paths:** `main/magefile.go` (add `AddDep` func), `main/CLAUDE.md` (add one row to § "Build Verification" mage targets table)
- **Packages:** `main` (the magefile package — build-tagged `//go:build mage`)
- **Blocked by:** —
- **Acceptance:**
    - `main/magefile.go` adds `func AddDep(module string) error` (exported — mage surfaces it as `mage addDep <module>`). Go doc comment on `AddDep` starting with "AddDep …" per CLAUDE.md naming rule 11. Signature matches the existing mage-target pattern in `magefile.go` (no `context.Context` — other targets don't thread it either).
    - Implementation shape: runs `sh.RunV("go", "get", module)` from the module root (`main/`) with default environment (no `GOPROXY` / `GOSUMDB` / checksum overrides per CLAUDE.md § "Dependencies"). Error wrapped: `fmt.Errorf("mage addDep: go get %s: %w", module, err)`. **No `go mod tidy` here** — `tidy` prunes deps that have no importer yet, which breaks the "add dep before writing the code that imports it" flow (Unit 2.2 does exactly that). Callers who want cleanup run `go mod tidy` separately after the importing code lands.
    - Uses `github.com/magefile/mage/sh` already imported at `main/magefile.go:18` (direct dep in `main/go.mod` line 7); no new magefile import needed.
    - `main/CLAUDE.md` § "Build Verification" mage targets table gets a new row: **Target** = `mage addDep <module>`; **Command** = `go get <module>`; **When** = "when adding a new Go dep (from Drop 2 onward)". Builder lands this markdown edit in the same commit as `magefile.go` — CLAUDE.md's role-boundary text forbids builder from editing **Go** code outside `.go` files, and is permissive about paired markdown that documents a Go-coupled change.
    - Target is invokable: `mage addDep github.com/magefile/mage@v1.17.1` from `main/` leaves `main/go.mod` + `main/go.sum` unchanged (mage is already at `v1.17.1` in `main/go.mod` line 7) — build-QA asserts `git diff main/go.mod main/go.sum` is empty after running the target, and exit code is 0.
    - `mage build` succeeds from `main/` (includes magefile compilation under `//go:build mage`).
    - `mage ci` succeeds from `main/` (gofumpt clean on `magefile.go` and `CLAUDE.md` has no gofumpt exposure — markdown only).
    - **No invocation of `mage install` anywhere** (CLAUDE.md § "Build Verification" rule 3).

### Unit 2.1 — Lift counting primitive into `internal/counting`

- **State:** todo
- **Paths:** `main/internal/counting/counting.go` (new), `main/internal/counting/counting_test.go` (new), `main/cmd/rak/root.go` (remove `Counts` struct + `count` function; imports shrink accordingly), `main/.golangci.yml` (remove orphan `cmd/rak/root.go` → `unused` exclusion rule now that `count` is exported + has a caller in `RunE`)
- **Packages:** `github.com/evanmschultz/rak/internal/counting` (new), `github.com/evanmschultz/rak/cmd/rak`
- **Blocked by:** —
- **Acceptance:**
    - `internal/counting/counting.go` defines exported `Counts` struct with fields **in declaration order** `Bytes int64`, `Lines int64`, `Words int64`, `Chars int64` (Go doc comment on struct + each field starting with identifier name per CLAUDE.md naming rule 11).
    - **`.golangci.yml` cleanup (F2 fold):** remove the `linters.exclusions.rules` entry that exempts `cmd/rak/root.go` from the `unused` linter (lines 19-24 of current file). After 2.1 lands, `count` is no longer unused (it's exported as `Count` and called from `cmd/rak/root.go`'s `RunE` in 2.3 — but 2.1's move alone is enough to stop triggering the warning since the symbol leaves `cmd/rak/`). Also remove the preceding rationale comment block (lines 3-17). File shrinks to minimal `version: "2"` with no custom rules. `mage lint` from `main/` must be green after removal.
    - **Cross-unit JSON contract (F4):** `Counts` carries **no `json:` struct tags**. Downstream Unit 2.2 JSON snapshot depends on this exact field declaration order and the absence of tags — changing 2.1's struct without updating 2.2's snapshot will break build-QA for 2.2.
    - `internal/counting/counting.go` defines exported `func Count(r io.Reader) (Counts, error)` with the same semantics as the current unexported `count` in `cmd/rak/root.go:42-78` (`bufio.NewReader` + `ReadRune` loop + `unicode.IsSpace` word split + `io.EOF` clean-exit). Go doc comment on `Count` starting with "Count …".
    - `cmd/rak/root.go` no longer declares `Counts` or `count` — both moved. `cmd/rak/root.go` imports shrink to remove `bufio` and `unicode` (the remaining imports are `fmt` and `github.com/spf13/cobra` — `io` can also go since `RunE` no longer uses it directly yet; 2.3 re-adds `io` for stdin).
    - `internal/counting/counting_test.go` ships a table-driven `TestCount` covering at minimum the following **exact** (input → expected `Counts`) tuples:
        - `""` → `{Bytes: 0, Lines: 0, Words: 0, Chars: 0}`
        - `"hello"` → `{Bytes: 5, Lines: 0, Words: 1, Chars: 5}`
        - `"hello\n"` → `{Bytes: 6, Lines: 1, Words: 1, Chars: 6}`
        - `"hello world"` → `{Bytes: 11, Lines: 0, Words: 2, Chars: 11}`
        - `"hello world\nfoo bar\n"` → `{Bytes: 20, Lines: 2, Words: 4, Chars: 20}`
        - `"héllo\n"` → `{Bytes: 7, Lines: 1, Words: 1, Chars: 6}` (UTF-8 multi-byte rune — Bytes vs Chars diverge)
        - `"a\r\nb\r\n"` → `{Bytes: 6, Lines: 2, Words: 2, Chars: 6}` (CRLF — `\r` is whitespace per `unicode.IsSpace`, so words split on CR; Lines increments only on `\n`; **F5 pin**).
    - Subtests via `t.Run` with descriptive names (CLAUDE.md naming rule 8).
    - Fixtures live in-memory via `strings.NewReader` / `bytes.NewReader`. No `testdata/` directory for this unit (CLAUDE.md § "Tests" — prefer in-memory fixtures).
    - `mage build` succeeds from `main/`.
    - `mage test` succeeds from `main/` with `-race` (mage target default). No test skips.

### Unit 2.2 — Land `internal/render` package with laslig dep re-add

- **State:** todo
- **Paths:** `main/internal/render/render.go` (new — `Renderer` interface + shared helpers), `main/internal/render/human.go` (new — laslig-backed human renderer), `main/internal/render/json.go` (new — stdlib encoding/json renderer), `main/internal/render/render_test.go` (new — snapshot tests for both renderers), `main/go.mod` (laslig dep add via `mage addDep`), `main/go.sum` (laslig dep add via `mage addDep`)
- **Packages:** `github.com/evanmschultz/rak/internal/render` (new)
- **Blocked by:** 2.0, 2.1
- **Acceptance:**
    - **Dep re-add via `mage addDep` (one-time for Drop 2):** builder runs `mage addDep github.com/evanmschultz/laslig@v0.2.4` from `main/`. The `AddDep` target (Unit 2.0) shells out to `go get github.com/evanmschultz/laslig@v0.2.4` + `go mod tidy`. Version pin `v0.2.4` is explicit (REFINEMENTS entry 4). **No direct `go get` invocation from the builder** — all dep adds from Drop 2 onward route through `mage addDep` per the tooling rule Unit 2.0 establishes.
    - After the dep add, `main/go.mod` `require` block lists `github.com/evanmschultz/laslig v0.2.4` as a direct (non-indirect) dep.
    - `internal/render/render.go` defines:
        - Exported `Renderer` interface with a single method (Go idiom — single-method interfaces end in `-er` per CLAUDE.md naming rule 5). Proposed minimal signature: `type Renderer interface { Render(w io.Writer, counts counting.Counts) error }`. This imports `github.com/evanmschultz/rak/internal/counting` — verifying 2.1 → 2.2 is the natural dep-DAG order (leaf → interior per CLAUDE.md § "Import DAG").
        - Go doc comment on `Renderer` and the method.
    - `internal/render/human.go` defines exported `func NewHumanRenderer() Renderer` (explicit constructor per decision 27(d) — no Format enum factory). Production implementation uses `laslig.New(out, laslig.Policy{Format: laslig.FormatAuto, Style: laslig.StyleAuto})` inside `Render` so the printer is created per-call bound to the writer — this gives automatic TTY-vs-pipe selection for free (evidence: `laslig/policy.go:211` `ResolveMode` + `laslig/printer.go:39` `New`). Renders `counting.Counts` as a laslig `KV` block with labels "Bytes", "Lines", "Words", "Chars" (values formatted as decimal int64). Go doc comments.
    - `internal/render/json.go` defines exported `func NewJSONRenderer() Renderer`. Implementation uses stdlib `encoding/json.NewEncoder(w).Encode(counts)` — no laslig for JSON (decision 27(d) / 27(e)). Go doc comments.
    - `internal/render/render_test.go`:
        - **Snapshot determinism mechanism (F3 pin):** the human-renderer test must use `laslig.NewWithMode(buf, laslig.Mode{Format: laslig.FormatPlain, Styled: false, Width: 80})` **inside a test-only helper or test-only subconstructor**, not inside the production `NewHumanRenderer` body. Production uses `laslig.New(out, laslig.Policy{Format: laslig.FormatAuto, Style: laslig.StyleAuto})` (above). To make the test possible without polluting the production constructor, `internal/render/human.go` exposes a second unexported constructor `newHumanRendererWithMode(mode laslig.Mode) Renderer` for test use from the same package. The snapshot test constructs this variant with the fixed `Mode`; the public-facing `NewHumanRenderer` is untouched.
        - **Invariant (F3):** human-renderer test output is **independent of `$COLUMNS`, `$TERM`, `$NO_COLOR`, `$CI`** — because the explicit `Mode` bypasses `ResolveMode`'s environment inspection entirely (evidence: `laslig/policy.go:211` vs. `laslig/printer.go:48`). Tests must not `os.Setenv` or `os.Unsetenv` any of those vars.
        - `TestHumanRenderer_SnapshotPlain` — constructs the test-only variant, calls `Render(buf, counting.Counts{Bytes: 12, Lines: 1, Words: 2, Chars: 12})` against a `*bytes.Buffer`, asserts buf equals the expected plain-mode KV output (exact string captured by builder at implementation time).
        - `TestJSONRenderer_Snapshot` — calls `Render(buf, counting.Counts{Bytes: 12, Lines: 1, Words: 2, Chars: 12})` and asserts exact output `{"Bytes":12,"Lines":1,"Words":2,"Chars":12}\n` (stdlib `json.Encoder.Encode` trails with `\n`; field order matches Unit 2.1 struct declaration order per F4 pin above).
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
    - `cmd/rak/root.go` imports `github.com/evanmschultz/rak/internal/counting`, `github.com/evanmschultz/rak/internal/render`, and the `--format` flag pflag wiring; re-adds `io` for stdin piping. Input sourcing uses `cmd.InOrStdin()` (cobra-idiomatic indirection so tests can inject via `cmd.SetIn(...)`). **`os.Stdin` direct access is not used** (F9 pin — test determinism depends on `cmd.InOrStdin()` being the single input path).
    - `RunE` behavior:
        1. `ctx := c.Context()` (keep for future cancellation wiring — no-op today).
        2. Read `io.Reader` input from `cmd.InOrStdin()` when `len(args)==0`. When `len(args)==1` return `fmt.Errorf("positional path argument not supported yet — walker lands in Drop 3; pipe input via stdin for now (got %q)", args[0])`. **Dev Decision A: A1 — error on `len(args)==1`** (fail fast and loud; silent-ignore creates a worse UX where `rak .` looks like it worked but actually blocks on stdin). No Phase 3 hedge remaining.
        3. Call `counts, err := counting.Count(reader)`. Wrap on error: `return fmt.Errorf("count input: %w", err)` (CLAUDE.md § "Errors" — wrap at boundaries).
        4. Select renderer from `--format` flag value: `"human"` → `render.NewHumanRenderer()`; `"json"` → `render.NewJSONRenderer()`; `"auto"` (default) → `render.NewHumanRenderer()` (human renderer's internal laslig policy auto-resolves to plain non-styled when `cmd.OutOrStdout()` is not a TTY — evidence: `laslig/policy.go:211`).
        5. Call `renderer.Render(cmd.OutOrStdout(), counts)`. Wrap on error: `return fmt.Errorf("render counts: %w", err)`.
    - `--format` flag: cobra `StringVarP(&format, "format", "f", "auto", "output format: auto | human | json")`. Values validated in `PersistentPreRunE` or inline — invalid value returns wrapped error.
    - `cmd/rak/root.go` remains under ~150 LOC (CLAUDE.md § "Project Structure" file budget).
    - `cmd/rak/root_test.go` stays ≤ ~150 LOC (**F11 pin**; CLAUDE.md § "Project Structure" file table). If test coverage would push past that budget, split the integration-style case into Unit 2.4's file rather than bloating root_test.go.
    - `cmd/rak/root_test.go` covers:
        - `TestRootCmd_ReadsStdin_RendersHumanDefault` — uses `cmd.SetIn(strings.NewReader("hello world\n"))` + `cmd.SetOut(&buf)`, calls `cmd.Execute()`, asserts buf contains human-format counts.
        - `TestRootCmd_FormatJSON` — `--format=json`, asserts buf contains the JSON counts.
        - `TestRootCmd_InvalidFormat` — `--format=xml`, asserts returned error is non-nil and mentions "format".
        - `TestRootCmd_RejectsPathArg` — invokes with one positional arg, asserts returned error mentions "Drop 3".
    - `mage build` succeeds.
    - `mage test` succeeds with `-race`.
    - No test invokes `mage install` or any raw `go` command.

### Unit 2.4 — End-to-end integration test via `cmd/rak/testdata/`

- **State:** todo
- **Paths:** `main/cmd/rak/testdata/hello.txt` (new fixture — contents stable and documented in test; see coverage hint below), `main/cmd/rak/integration_test.go` (new) OR `main/cmd/rak/root_test.go` (extended — builder's call based on root_test.go's remaining LOC budget from F11 pin above)
- **Packages:** `github.com/evanmschultz/rak/cmd/rak`
- **Blocked by:** 2.3
- **Acceptance:**
    - `cmd/rak/testdata/` directory created (Go stdlib `testdata` idiom — ignored by `go` tooling per `go help test`; CLAUDE.md § "Tests" → "Two-tier testdata rule" explicitly sanctions this as the single guaranteed fixture slot).
    - At least one fixture file (e.g. `hello.txt`) with known, stable content. **Fixture coverage hint (F12):** the fixture must exercise at minimum (a) multi-line content (more than one `\n`), (b) multi-word content (more than one whitespace-separated token), and (c) at least one multi-byte UTF-8 rune (to catch Bytes-vs-Chars regressions — the fixture's expected `Counts` must have `Bytes > Chars`).
    - Test asserts the exact expected `Counts` output for that content in both human and JSON formats.
    - Integration test reads the fixture via `os.Open(filepath.Join("testdata", "hello.txt"))`, wires it to the cobra command via `cmd.SetIn(file)`, captures stdout via `cmd.SetOut(&buf)`, runs `cmd.Execute()`, asserts buf matches an expected string.
    - Test covers both format paths: `--format=human` and `--format=json`.
    - Test runs green under `mage test` with `-race`.
    - `mage ci` succeeds from `main/` end-to-end (gofumpt / vet / golangci-lint / tests all green).

## Notes

**Unit count reshape (4 → 5 in Round 2, from Round 1's 4).** Round 1 proposed 4 units (dropped "2.4 TTY-auto" and "2.5 tests" from main/PLAN.md's 5-unit sketch; folded tests into their owning packages per TDD-first). Round 2 prepends a new Unit 2.0 (`mage addDep` target) per dev Decision B, giving 5 units total. The reshape rationale: 2.0 is atomic (builder lands the mage target + table row; build-QA asserts `mage addDep github.com/magefile/mage` is a no-op) and independent (no laslig imports in 2.0 itself — that's 2.2's job). Folding 2.0 into 2.2 would couple tooling correctness with package authoring, muddying per-unit build-QA; keeping it standalone also lets future dep adds (tiktoken Drop 7, errgroup Drop 8) reuse the target.

**Dev Decision A — positional path argument behavior (Unit 2.3): A1 — error on `len(args)==1`.** Resolved. Plan text removes the Round 1 "subject to Phase 3 dev approval" hedge. Error message points to Drop 3. Rationale per dev: fail fast and loud; silent-ignore makes `rak .` look like it worked but actually blocks on stdin.

**Dev Decision B — laslig bootstrap carve-out scope (Unit 2.2): B1 — add `mage addDep <module>` target first, then use it for laslig.** Resolved. Plan text removes every Round 1 reference to the CLAUDE.md § "Dependencies" → "Bootstrap carve-out" clause. Unit 2.0 lands the target; Unit 2.2 uses it. No amendment to CLAUDE.md § "Dependencies" → "Bootstrap carve-out" prose — it remains describing the Drop 1.4 exception, now with the mage target closing the "from Drop 2 onward" gap the prose assumed.

**Round 1 plan-QA sharpenings folded (traceable for Round 2 plan-QA):**

1. **F13 / P1 — DAG + `blocked_by` fix.** Round 2 DAG is strict import-DAG order: `(2.0 ∥ 2.1) → 2.2 → 2.3 → 2.4`. Unit 2.2 `Blocked by` explicitly lists `2.0, 2.1` (Round 1 said `—` — that was the bug). Unit 2.3 `Blocked by: 2.1, 2.2`. Unit 2.4 `Blocked by: 2.3`. 2.0 and 2.1 are parallel-eligible (neither blocks the other).
2. **F3 — Snapshot determinism mechanism pin.** Unit 2.2 acceptance now specifies a single mechanism: `laslig.NewWithMode(..., laslig.Mode{Format: laslig.FormatPlain, Styled: false, Width: 80})` inside a test-only unexported constructor `newHumanRendererWithMode`, not inside the production `NewHumanRenderer` body. Explicit invariant: "test output independent of `$COLUMNS`, `$TERM`, `$NO_COLOR`, `$CI`" (because `NewWithMode` bypasses `ResolveMode`'s env inspection entirely per `laslig/policy.go:211` vs. `laslig/printer.go:48`).
3. **F4 — Cross-unit JSON contract pin.** Unit 2.1 acceptance now states `Counts` fields are in declaration order `Bytes, Lines, Words, Chars` with no `json:` struct tags. Unit 2.2 acceptance calls out that its JSON snapshot depends on 2.1's field order + no-tags contract.
4. **F5 — CRLF expected tuple pin.** Unit 2.1 test table now lists `"a\r\nb\r\n"` → `{Bytes: 6, Lines: 2, Words: 2, Chars: 6}` as an explicit tuple, replacing Round 1's prose clause.
5. **F11 — `root_test.go` LOC budget.** Unit 2.3 acceptance adds an explicit ≤ ~150 LOC ceiling for `cmd/rak/root_test.go` per CLAUDE.md § "Project Structure" file budget, with an overflow hint: push cases into 2.4's integration_test.go rather than bloat root_test.go.
6. **F12 — Fixture coverage hint.** Unit 2.4 acceptance adds "fixture exercises at least multi-line, multi-word, and one multi-byte UTF-8 rune (catches Bytes-vs-Chars regressions)."
7. **F9 — Drop `os.Stdin` alternative.** Unit 2.3 paths/acceptance remove the "or `os.Stdin`" option. Tests force `cmd.InOrStdin()` as the single input path so `cmd.SetIn(...)` is reliable.

**Unknowns resolved in Round 2:**

- Unknown #1 (stdin-vs-path-arg, Round 1): resolved by Decision A above.
- Unknown #2 (laslig re-add mechanism, Round 1): resolved by Decision B above.

**Round 2 plan-QA fork decisions (applied pre-build, no Round 3 planner spawn):**

- **A1** (F1 tidy-prune): `AddDep` runs `go get` only, no `go mod tidy`. Avoids prune trap when adding a dep before its importing code exists.
- **B1** (F3 no-op precision): build-QA runs `mage addDep github.com/magefile/mage@v1.17.1` and asserts `git diff main/go.mod main/go.sum` is empty + exit 0.
- **C1** (F4/F5 ctx param): `AddDep` signature is `func AddDep(module string) error` — matches existing mage targets, no `context.Context`.
- **D3** (F15 CLAUDE.md auth): accept role-boundary text as permissive. Builder lands the CLAUDE.md table row in the same commit as `magefile.go`. No CLAUDE.md amendment to § "Orchestrator Role Boundaries" needed; CLAUDE.md § "Dependencies" is updated separately by orch to reflect the new `mage addDep` default path.
- **F2** (mechanical fold): `.golangci.yml` orphan `unused` exclusion removed in Unit 2.1 (see Unit 2.1 paths + acceptance).

**Laslig API used (v0.2.4 citations — verified Round 1 falsification F2, unchanged for Round 2).**
- `func New(out io.Writer, policy Policy) *Printer` — `printer.go:39` — Unit 2.2's `NewHumanRenderer` uses this with `Policy{Format: FormatAuto, Style: StyleAuto}` for automatic TTY-vs-pipe selection.
- `func NewWithMode(out io.Writer, mode Mode) *Printer` — `printer.go:48` — Unit 2.2's test-only `newHumanRendererWithMode` uses this with `Mode{Format: FormatPlain, Styled: false, Width: 80}` for snapshot determinism.
- `func ResolveMode(out io.Writer, policy Policy) Mode` — `policy.go:211` — the TTY detection primitive; called internally by `New`. Rak does not need to call this directly. Tests must not depend on it being called (see F3 pin above).
- Format / Style enums and Printer methods (Section/Record/KV/Paragraph/List/Notice/Table) all in `policy.go` + `types.go` + `printer.go`.

**Decision 27(d) reinforcement.** `internal/render` ships explicit `NewHumanRenderer` + `NewJSONRenderer` constructors. **No** `NewRenderer(format Format) Renderer` factory. Callers at `cmd/rak/root.go` switch on the `--format` flag and call the right constructor directly. This keeps the dep-DAG simple and avoids a premature enum.

**Decision 25 reinforcement.** `internal/counting.Count` takes `io.Reader` only. No `fileset.File`, no path strings, no direct filesystem access. The counting domain is pure stream math.

**Deferred to later drops (not Drop 2's scope):** walker + path traversal (Drop 3); gitignore / include / exclude matching (Drop 4); language detection + blank/comment/code split (Drops 5 + 6); summary rollup + sort (Drop 7); token counting (Drop 8); parallel walk (Drop 9). Each later drop's `PLAN.md` will re-plan its own boundary.

## Hylla Feedback

N/A — Drop 2 Round 2 planning revision relies on non-Go evidence: dev decisions A + B from the Phase 3 discussion, current file contents (`drops/DROP_2_COUNTING_DOMAIN_RENDER_BOUNDARY/PLAN.md` Round 1, `CLAUDE.md`, `magefile.go`, `go.mod`), and `main/drops/WORKFLOW.md` Phase 3 step 4 (default in-place edit). The only Go symbols referenced in the revision (laslig `New`, `NewWithMode`, `ResolveMode`, stdlib `encoding/json.Encoder`, `cobra.Command.InOrStdin`) were all verified in Round 1 via `Read` against the laslig module cache at `/Users/evanschultz/go/pkg/mod/github.com/evanmschultz/laslig@v0.2.4/`. No additional Hylla queries were run in Round 2 because the laslig module is not in rak's Hylla-indexed artifact (rak itself is, laslig is not), and the committed Go files being revised (`magefile.go`, future `cmd/rak/root.go`) are either non-Hylla-indexed (magefile) or subject to the "changed since last ingest" rule (Round 1 already inspected `cmd/rak/root.go` via `Read`). No Hylla miss to record.
