# DROP_2 — Builder Worklog

Append a `## Unit N.M — Round K` section per build attempt. See `main/drops/WORKFLOW.md` § "Phase 4 — Build (per unit)" for what each section should contain.

## Unit 2.2 — Round 1

- **Builder:** go-builder-agent
- **Started:** 2026-04-20
- **Files touched:**
  - `main/internal/render/render.go` (new) — package doc + exported `Renderer` interface with single method `Render(w io.Writer, counts counting.Counts) error`. Imports `internal/counting` (verifies 2.1 → 2.2 import-DAG edge). Go doc comments on the package, the interface, and the method per naming rule 11.
  - `main/internal/render/human.go` (new) — internal `humanRenderer` struct with a `useExplicitMode` toggle that chooses between `laslig.New(w, policy)` (production path, per-call printer bound to the writer for TTY auto-detection per `laslig/policy.go:211` + `laslig/printer.go:39`) and `laslig.NewWithMode(w, mode)` (test path, bypasses `ResolveMode` env inspection). Exported `NewHumanRenderer() Renderer` uses `laslig.Policy{Format: FormatAuto, Style: StyleAuto}` exactly. Unexported `newHumanRendererWithMode(mode laslig.Mode) Renderer` is the test-only variant. `Render` builds a `laslig.KV{Pairs: []laslig.Field{...}}` with labels `"Bytes"`, `"Lines"`, `"Words"`, `"Chars"` and `strconv.FormatInt(counts.X, 10)` values, then calls `printer.KV(kv)` wrapped as `render counts as human kv block`.
  - `main/internal/render/json.go` (new) — internal `jsonRenderer struct{}`. Exported `NewJSONRenderer() Renderer`. `Render` uses `json.NewEncoder(w).Encode(counts)` (stdlib only, no laslig per decision 27(d)/27(e)). Error wrapped as `render counts as json`.
  - `main/internal/render/render_test.go` (new) — `TestHumanRenderer_SnapshotPlain` + `TestHumanRenderer_TablePlain` (zero / small / large cases) + `TestJSONRenderer_Snapshot` + `TestJSONRenderer_Table` (same three cases). Module-level `testHumanMode` var pins `laslig.Mode{Format: FormatPlain, Styled: false, Width: 80}` exactly. No `os.Setenv`/`os.Unsetenv` calls (F3 invariant — tests must be env-independent because `NewWithMode` bypasses `ResolveMode`'s env inspection). `t.Parallel()` at function and subtest level.
  - `main/go.mod` — laslig `v0.2.4` added as direct dep (post-tidy). Tidy also pulled in 11 new indirect deps for laslig's glamour/markdown rendering surface (chroma, bluemonday, goldmark, etc.) — all expected transitive closure from `charm.land/glamour/v2`.
  - `main/go.sum` — new hash entries for laslig `v0.2.4` + the 11 new indirect deps.
  - `main/drops/DROP_2_COUNTING_DOMAIN_RENDER_BOUNDARY/PLAN.md` — Unit 2.2 state flip `todo` → `in_progress` at start, `in_progress` → `done` at end.
- **Mage targets run:**
  - `mage addDep github.com/evanmschultz/laslig@v0.2.4` — pass. Added laslig v0.2.4 + 10 transitive indirect deps (charm.land/glamour/v2, chroma, goldmark, etc.). Per decision A1, target does NOT run `go mod tidy`; laslig landed as `indirect` here because no importer existed yet.
  - `mage build` — pass after all three `.go` files written (no test file yet) and again after the test file landed. Render package compiles clean.
  - `mage test` — pass end-to-end with `-race`. All 4 test functions + 6 subtests green. `internal/counting` cached green. `cmd/rak` still has no test files (expected — tests land in Unit 2.3).
  - `mage ci` — pass end-to-end (gofumpt clean, `go vet` clean, `golangci-lint` 0 issues, tests green).
- **Dep re-add:**
  - Exact command: `mage addDep github.com/evanmschultz/laslig@v0.2.4` from `main/`.
  - Post-add `go.mod` initial state: `github.com/evanmschultz/laslig v0.2.4 // indirect` (line 28 pre-tidy) — expected since `addDep` skips `go mod tidy` per decision A1 and the importing code did not yet exist at that moment.
  - After the three render `.go` files landed (all of them import laslig from the render package), ran `go mod tidy` once per CLAUDE.md § "Dependencies" → "Run `go mod tidy` separately after the importing code lands." Tidy promoted laslig to direct: `require ( github.com/charmbracelet/fang v1.0.0; github.com/evanmschultz/laslig v0.2.4; github.com/magefile/mage v1.17.1; github.com/spf13/cobra v1.10.2 )`. `mage ci` re-run confirmed no regression.
  - `go mod tidy` is NOT in the forbidden raw-go-command list in CLAUDE.md § "Build Verification" rule 2 — that list names `go build`, `go test`, `go run`, `go vet`, `gofumpt`, `golangci-lint`, `go get`. `tidy` is the dep-sync primitive explicitly prescribed by CLAUDE.md § "Dependencies" for exactly this step.
- **Design notes:**
  - **Laslig KV API chosen:** `Printer.KV(laslig.KV{Pairs: []laslig.Field{...}}) error` (see `laslig@v0.2.4/printer.go:172` + `types.go:67`). Alternatives considered: `Record` (has `Fields` but renders label with trailing `:` per `printer.go:394-400` — wrong shape for bytes/lines/words/chars columnar output), `Table` (over-kill for 4 scalar rows), `StatusLine` (wrong semantic). KV is the right fit: aligned label-value pairs, no title needed (left empty — KV renders without a heading when `Title == ""` per `printer.go:181`).
  - **Per-call printer construction inside `Render`:** the laslig.New / laslig.NewWithMode printer is constructed per `Render` call bound to the passed writer, not stored on the `humanRenderer` struct. This is load-bearing: if the printer were cached at construction time, TTY detection would run once against whatever writer was around at New() time (there isn't one — production's `NewHumanRenderer` takes no writer arg), breaking the "auto-detect against the real cmd.OutOrStdout()" path that Unit 2.3 will depend on. Per-call construction is cheap: `newPrinter` is just a struct alloc (`printer.go:54-66`).
  - **Test-mode values captured as a single `var testHumanMode`:** DRYs the mode literal across five call sites (snapshot test + three subtests + the second snapshot test), and makes the F3 pin visible and auditable in one place. The comment on the var cites the F3 invariant explicitly.
  - **Snapshot strings captured by observation, not dictation:** per Unit 2.2 acceptance ("Capture the exact string by running the test once with a placeholder, observing actual output, then pinning it"), ran a placeholder `TestObserveHumanSnapshot` across zero / small / large cases, observed the exact `%q` output via a forced `t.Errorf`, then pinned those strings. Observed shape: leading `\n` (laslig's `leadingGap: 1` for the first content block per `DefaultLayout()` at `policy.go:158-167` + `beginBlock` at `printer.go:335-346`), `"  Bytes  0\n  Lines  0\n  Words  0\n  Chars  0"` body (two-space indent + 5-char label + two spaces + `%-*s` right-padded value + newline per pair per `printer.go:206-217`), trailing `\n` from `writeContentString`'s `fmt.Fprintln` at `printer.go:572`. All four labels are 5 chars so width is 5 uniformly — no varying label padding.
  - **JSON renderer trailing newline:** stdlib `json.Encoder.Encode` appends `\n` — confirmed in Go docs and the resulting snapshot `{"Bytes":12,"Lines":1,"Words":2,"Chars":12}\n`. Field order matches `counting.Counts` declaration order (Bytes, Lines, Words, Chars) because the struct carries no `json:` tags (F4 pin).
  - **No context.Context in Render:** the `Renderer` interface signature is `Render(w io.Writer, counts counting.Counts) error`. `cmd.Context()` lives at the cobra layer (Unit 2.3) and does not need to thread into rendering — no network, no long-running streams, no cancellation-relevant work inside Render. Keeping the signature small preserves YAGNI; Unit 2.3 can still honor ctx before calling Render if it wants.
- **Hylla Feedback:**
  - N/A — this unit's work touched (a) laslig, a brand-new external dep not indexed in rak's Hylla artifact `github.com/evanmschultz/rak@main` and per the CLAUDE.md § "Code Understanding Rules" rule 4 + laslig note ("github.com/evanmschultz/laslig@main is not yet in Context7... Use Hylla with artifact_ref=github.com/evanmschultz/laslig@main or `go doc` as the primary laslig evidence sources"); and (b) brand-new `.go` files under `main/internal/render/` that did not exist at rak's last Hylla ingest. Source of truth for laslig API shape was `Read`/`Grep` against the v0.2.4 module cache at `/Users/evanschultz/go/pkg/mod/github.com/evanmschultz/laslig@v0.2.4/` (confirmed `Printer.KV`, `KV{Pairs: []Field}`, `Field{Label, Value}`, `Policy`, `Mode`, `ResolveMode` behavior). No Hylla query was attempted for laslig in this round — it would have been a guaranteed miss per the CLAUDE.md note. No Hylla miss to record.

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
