# DROP_2 — Builder QA Falsification

Append a `## Unit N.M — Round K` section per QA attempt. See `main/drops/WORKFLOW.md`.

## Unit 2.4 — Round 1

- **QA agent:** go-qa-falsification-agent
- **Verdict:** pass
- **Attacks attempted:**
    1. Fixture byte pollution (BOM / CRLF / trailing whitespace) — blocked (`xxd` shows 29 clean bytes, no BOM, LF-only, trailing `\n` present).
    2. Counts math error — blocked (manual `ReadRune` walk yields `{29,2,5,27}`).
    3. `é` / `ï` NFC-vs-NFD encoding drift — blocked (`é` = `c3 a9` NFC, `ï` = `c3 af` NFC, both 2 bytes / 1 rune).
    4. F12 fixture coverage gap — blocked (multi-line, multi-word, multi-byte UTF-8 all present; Bytes > Chars 29 > 27).
    5. Human assertion tolerance weakness — accepted (`strings.Contains` on `"2"` is subsumed by `"29"` / `"27"`, carries no independent signal; builder documented the asymmetry at `integration_test.go:69` and worklog; PLAN.md line 138 is ambiguous enough to permit it).
    6. `t.Parallel()` file-handle / buffer race — blocked (fresh `*os.File` and `bytes.Buffer` per subtest, fresh `newRootCmd` factory avoids shared flag state).
    7. Wrong renderer selected — blocked (byte-exact JSON assertion would diverge immediately on a human-renderer leak).
    8. F11 pin creep — blocked (`root_test.go` unchanged at 108 LOC, `git diff` empty).
    9. Fixture path portability — blocked (`filepath.Join` + Go test CWD = package dir per `go help test`).
    10. `mage ci` green claim — blocked (fresh `mage ci` / `mage test` / `mage lint` / `mage build` all pass from `main/`).
    11. gofumpt / golangci-lint drift — blocked (`mage lint` 0 issues fresh).
    12. Race detector coverage — blocked (`magefile.go:29` `go test -race ./...`, integration tests included).
    13. Unused / shadowed imports in `integration_test.go` — blocked (all 5 imports used; `mage lint` would have caught either).
    14. LOC claim drift — accepted-minor (worklog says 119, file is 121; cosmetic, no acceptance cap on `integration_test.go`).
    15. Doc-comment rule 11 regression — blocked (both `Test*` funcs have `// TestRootCmd_Integration_*` doc comments starting with identifier name).
    16. Fixture trailing-newline semantics — blocked (`xxd` shows trailing `0a`; Lines=2 as expected).
    17. `testdata` idiom spelling — blocked (directory is `testdata/`, not `test-data/` or `fixtures/`).
    18. `mage install` invocation anywhere — blocked (no occurrences in diff or test).
- **Findings / counterexamples:** none (2 minor surface-level notes: worklog LOC undercount cosmetic; human-assertion `"2"` substring weak-but-documented. Neither blocks acceptance).
- **Hylla Feedback:** N/A — Unit 2.4 touched only a text fixture + a new `_test.go` file in a package changed since last ingest per CLAUDE.md § "Code Understanding Rules" rule 2; Go-symbol verification went via `Read` / `Grep` / `xxd` / mage targets directly.

## Unit 2.3 — Round 1

- **QA agent:** go-qa-falsification-agent
- **Verdict:** pass
- **Attacks attempted:**
    1. **F9 `os.Stdin` bypass** — blocked. `grep -n 'os\.Stdin' cmd/rak/` returns only `root_test.go:10`, a comment documenting the F9 pin. Production path uses `c.InOrStdin()` at `root.go:63`.
    2. **Flag double-registration panic** — blocked. `newRootCmd` constructs a fresh `*cobra.Command` literal each call (`root.go:22`); cobra lazily builds `cmd.Flags()` on first access; `StringVarP(&format, ...)` binds into that fresh pflag FlagSet. Two concurrent calls cannot collide.
    3. **Closure format-var capture** — blocked. `var format string` lives inside `newRootCmd` body (`root.go:20`); `StringVarP(&format, ...)` captures its address per-factory-call; `RunE` closure passes `format` by value to `runRoot(c, args, format)` at invocation time, after cobra has parsed flags into it. Parallel subtests each own their own `format` cell.
    4. **Error-wrap string drift** — blocked. `root.go:65` reads `fmt.Errorf("count input: %w", err)`; `root.go:74` reads `fmt.Errorf("render counts: %w", err)`. Both exact matches for PLAN bullets 114 + 116.
    5. **Fang signal wiring regression** — blocked. `main.go:13-17` still wires `fang.Execute(ctx, newRootCmd(), fang.WithNotifySignal(os.Interrupt, syscall.SIGTERM))`. Unit 2.3's scope was `root.go` + `root_test.go` only; `main.go` untouched.
    6. **`SilenceUsage` / `SilenceErrors` regression** — accepted. Neither is set on the new `cmd`, so cobra prints usage+error on RunE failure. `git show 3cb4325:cmd/rak/root.go` (pre-2.3 baseline) also did not set them — no regression introduced by 2.3, no change to the established CLI error-display behavior. If the dev wants fail-silent-usage behavior later, that's a separate follow-up, not a Unit 2.3 break.
    7. **Format-validation ordering (drain stdin before reject)** — accepted. `runRoot` calls `counting.Count` before `selectRenderer`, so `cat huge | rak --format=xml` consumes stdin before erroring. PLAN bullet 117 says "Values validated in PersistentPreRunE or inline — invalid value returns wrapped error" — both orderings permitted, no fail-fast requirement. `TestRootCmd_InvalidFormat` still proves the error surface. Not a plan violation.
    8. **Dead imports / staticcheck QF1011 / gofumpt drift** — blocked. `mage ci` green: gofumpt clean, `go vet` clean, `golangci-lint` 0 issues, tests pass. Worklog confirms the QF1011 on `var reader io.Reader = ...` was surfaced by ci and fixed before completion. Final import set (`fmt`, `cobra`, `counting`, `render`) all reachable.
    9. **`_ = c.Context()` dead-op lint** — blocked. `mage lint` reports 0 issues; Go linters do not flag method-call-to-discard when the intent (forward-compat) is documented inline (`root.go:50-52`).
    10. **`MaximumNArgs(1)` + args==1 coverage** — blocked. `TestRootCmd_RejectsPathArg` invokes with `[]string{"./somepath"}`, asserts the returned error mentions "Drop 3". args==2 case is cobra-framework territory (rejected before RunE) — not Unit 2.3's to retest.
    11. **`t.Parallel()` + shared state race** — blocked. Every test builds its own `newRootCmd()`, `bytes.Buffer`, `strings.NewReader`. No package-level var is written by any test. `mage test` (always `-race`) green.
    12. **`root.go` LOC budget** — blocked. `wc -l cmd/rak/root.go` → 93; PLAN bullet 118 ceiling is ~150.
    13. **F11 `root_test.go` ≤150 LOC pin** — blocked. `wc -l cmd/rak/root_test.go` → 108; F11 ceiling is 150. Headroom preserved for Unit 2.4's integration test if it lands here instead of a new file.
    14. **Default `--format=auto` vs TTY semantics** — blocked. `selectRenderer("auto")` returns `render.NewHumanRenderer()` (`root.go:87`); laslig's per-call printer construction inside `Render` runs `ResolveMode` against the real `c.OutOrStdout()` (a `bytes.Buffer` in tests, a TTY in production). Matches PLAN bullet 115 + Unit 2.2 design notes.
    15. **Exported doc-comment drift** — blocked. No exported identifiers in `root.go` / `root_test.go` (everything is package-private `main`). `newRootCmd`, `runRoot`, `selectRenderer` all carry descriptive unexported doc comments. Test functions carry leading doc comments describing each scenario.
    16. **`invalid --format` error mentions "format"** — blocked. `root.go:91` returns `fmt.Errorf("invalid --format %q: want auto | human | json", format)`; `TestRootCmd_InvalidFormat` asserts `err.Error()` contains `"format"` — the literal flag name guarantees it.
- **Findings / counterexamples:** none.
- **Hylla Feedback:**
    - N/A — this review touched only `main/cmd/rak/*.go` files that are either changed since last ingest (per CLAUDE.md § "Code Understanding Rules" rule 2, use `git diff` / `Read`) or brand new. Cobra + laslig APIs cross-checked against the worklog's citations rather than Hylla. No Hylla query attempted, no miss to record.

## Unit 2.2 — Round 1

- **QA agent:** go-qa-falsification-agent
- **Verdict:** pass
- **Attacks tried:**
    1. **F3 env pollution** — mitigated. Grep for `os.Setenv|Unsetenv|LookupEnv|Getenv|COLUMNS|NO_COLOR|TERM|CI` in `internal/render/` returned only comment hits documenting the invariant. No runtime env mutation.
    2. **F3 mode pin** — mitigated. `render_test.go` declares `testHumanMode = laslig.Mode{Format: FormatPlain, Styled: false, Width: 80}` exactly. Every snapshot calls `newHumanRendererWithMode(testHumanMode)`.
    3. **Per-call printer** — mitigated. `human.go:60-66` constructs `laslig.Printer` per `Render` call bound to the caller's writer. Nothing cached at `NewHumanRenderer()` time.
    4. **JSON exact string** — mitigated. Snapshot asserts `{"Bytes":12,"Lines":1,"Words":2,"Chars":12}\n` exactly — declaration order (not alphabetical) confirms no `json:` tags.
    5. **`Counts` tagless (F4)** — mitigated. `grep -n 'json:' internal/counting/counting.go` → zero hits.
    6. **Laslig direct-dep** — mitigated. `go.mod` line 7 inside first `require ( ... )` block, no `// indirect`.
    7. **`go mod tidy` process concern** — accepted. Builder ran `go mod tidy` directly to promote laslig indirect→direct. CLAUDE.md § "Dependencies" prescribes this flow after `mage addDep`. Not in forbidden raw-go list. Recommend future `mage tidy` wrapper for uniformity.
    8. **Laslig KV method** — mitigated. `laslig@v0.2.4/printer.go:172-218` confirms `Printer.KV` iterates `Pairs` in slice order; `human.go:68-75` supplies Pairs in Bytes/Lines/Words/Chars declaration order.
    9. **Snapshot format leak** — mitigated. Grep for ANSI codes in `render_test.go` → zero hits. Plain ASCII + `\n` only.
    10. **Large-count formatting** — mitigated. Test covers `Bytes: 1_000_000_000` with full decimal output, no scientific notation, no comma separators.
    11. **Table shape** — mitigated. `TestHumanRenderer_TablePlain` + `TestJSONRenderer_Table` each ship 3 cases (zero/small/large) with descriptive `t.Run` subtest names.
    12. **YAGNI** — mitigated. Only `NewHumanRenderer` + `NewJSONRenderer` exported. No XML/CSV/YAML/TOML. No options struct. No format-enum factory.
    13. **Test parallelism / race** — mitigated. `t.Parallel()` at function + subtest level. `mage test` with `-race` green.
    14. **Import layering** — mitigated. `internal/render` imports only `io`, `fmt`, `strconv`, `encoding/json`, `laslig`, `internal/counting`. No `cmd/rak` leak. DAG intact.
    15. **Interface conformance** — mitigated via compilation (`mage build` green proves constructors return `Renderer`-conformant values).
- **Mage / shell invocations:**
    - `mage build` → exit 0.
    - `mage test` → exit 0; render + counting OK.
    - `mage ci` → `0 issues.` end-to-end green.
- **Findings:** none.
- **Accepted trade-offs:**
    - Attack 7: raw `go mod tidy` to promote laslig indirect→direct. Explicitly sanctioned by CLAUDE.md § "Dependencies". Recommend `mage tidy` wrapper later; accept for Unit 2.2.
    - Attack 15: no compile-time `var _ Renderer = ...` assertion. Not required; `mage build` green proves conformance.
- **Hylla Feedback:** N/A — new `internal/render/` files not in last ingest; laslig external. `Read` + `Grep` + module cache.

## Unit 2.1 — Round 1

- **QA agent:** go-qa-falsification-agent
- **Verdict:** pass
- **Attacks tried:**
    1. **F4 JSON field tags** — mitigated. `grep json: internal/counting/` → zero hits. No tags on `Counts`.
    2. **F4 field order** — mitigated. counting.go lines 20/23/26/29 declare `Bytes`, `Lines`, `Words`, `Chars` as `int64` in exact PLAN.md order.
    3. **Test tuple accuracy** — mitigated. All 7 subtests match PLAN.md lines 67-73 verbatim; UTF-8 and CRLF F5 pin agree.
    4. **Semantic drift from old `count`** — mitigated. `git show HEAD:cmd/rak/root.go` old `count` body byte-for-byte identical to new `Count`. No algorithmic change.
    5. **CRLF hand-derivation** — mitigated. `"a\r\nb\r\n"`: `\r` is `unicode.IsSpace`, so words are "a" and "b" (2); `\n` occurrences (2) → Lines=2; 6 bytes = 6 chars (all ASCII). Test tuple agrees.
    6. **Empty input** — mitigated. Tuple `{0,0,0,0}` with err=nil; `ReadRune` returns `io.EOF` → `return counts, nil`.
    7. **Non-UTF-8 input** — accepted (not in acceptance). `bufio.Reader.ReadRune` returns U+FFFD size 1 on invalid bytes, guaranteeing progress. No risk of loop.
    8. **root.go compilation** — mitigated. `mage build` exit 0. Imports shrunk to `fmt` + `cobra` only. No orphans.
    9. **RunE stub** — mitigated (per plan line 65 "smallest change that keeps mage build green"; 2.3 re-adds io for stdin).
    10. **`Long` description edit** — mitigated. root.go is a Unit 2.1 path per PLAN.md line 56, in-file prose is in-scope, new text is accurate (2.3 wiring deferred).
    11. **`.golangci.yml` shrink F2** — mitigated. File is exactly `version: "2"`. `mage lint` 0 issues.
    12. **`magefile.go` F3 fold** — mitigated. `magefile.go:5` says "ten canonical targets". `grep -n nine magefile.go` → no hits. CLAUDE.md table has 10 rows. `mage -l` lists 10 targets.
    13. **Test parallelism / race** — mitigated. `t.Parallel()` at function + subtest level. `mage test` (runs `-race`) exit 0.
    14. **Package doc comment** — mitigated. counting.go lines 1-4 carry proper package doc.
    15. **Exported/unexported split** — mitigated. Only `Counts`, four fields, and `Count` exported. No incidental exports.
    16. **`int64` discipline** — mitigated. All four fields `int64`; `+= int64(size)` + `int64` `++` idioms used. No plain `int`.
    17. **YAGNI** — mitigated. No `CountString`, no `CountsSum`, no options struct. Minimal surface.
- **Mage / shell invocations:**
    - `mage build` → exit 0, no output.
    - `mage test` → exit 0; `internal/counting` OK (cached); `cmd/rak` no test files (expected — tests land in 2.3).
    - `mage lint` → `0 issues.`
    - `mage ci` → exit 0 end-to-end (gofumpt clean, vet clean, golangci-lint 0 issues, tests pass).
    - `mage -l` → 10 targets listed.
    - `git diff --stat HEAD` → 5 files changed; scope matches Unit 2.1 paths (PLAN.md line 56) plus in-drop doc files.
    - `git show HEAD:cmd/rak/root.go` → old `count` body byte-for-byte identical to new `Count` body.
- **Findings:** none.
- **Accepted trade-offs:**
    - Non-UTF-8 input not covered by a test (attack 7). Not required by PLAN.md; `bufio.Reader.ReadRune` guarantees progress via U+FFFD substitution.
    - RunE still returns "not implemented" stub (attack 9). Per PLAN.md line 65, smallest-change policy defers wiring to Unit 2.3.
- **Hylla Feedback:** N/A — in-scope artifacts are either pre-ingest (`internal/counting/` brand new), changed-since-last-ingest (`cmd/rak/root.go`), or non-Go (`.golangci.yml`, `magefile.go` build-tagged, PLAN.md, BUILDER_WORKLOG.md). Evidence via `Read` + `git show` + `git diff` + live `mage` runs. No miss.

## Unit 2.0 — Round 1

- **QA agent:** go-qa-falsification-agent
- **Verdict:** pass
- **Attacks tried:**
    1. **Argument-less invocation (`mage addDep` with no module)** — mitigated. Mage prints `not enough arguments for target "AddDep", expected 1, got 0` and exits 2. Clean, actionable, no panic.
    2. **Nonexistent module (`mage addDep nonexistent.example.invalid/definitely-not-a-module@v0.0.0`)** — mitigated. Error wrapped with module name per acceptance: `Error: mage addDep nonexistent.example.invalid/definitely-not-a-module@v0.0.0: running "go get nonexistent.example.invalid/definitely-not-a-module@v0.0.0" failed with exit code 1`. Exit 1. `git diff go.mod go.sum` empty after the failure — no mutation from a failed fetch.
    3. **Shell-injection module string (`mage addDep 'foo; rm -rf /'`)** — mitigated. Verified `github.com/magefile/mage@v1.17.1/sh/cmd.go:57` `RunV` → line 130 `exec.CommandContext(context.Background(), cmd, args...)`. Uses `os/exec` directly, no shell expansion. The malformed path reaches `go get` as a single argv element; `go get` rejects with `malformed module path "foo; rm -rf ": invalid char ';'`. Mage exits 1.
    4. **Tidy-prune trap** — mitigated by design (A1 fork). `AddDep` runs `sh.RunV("go", "get", module)` only. No `go mod tidy` call in the target body (magefile.go:88-93). Unit 2.2's "add laslig before writing importing code" flow is safe.
    5. **No-op assertion (`mage addDep github.com/magefile/mage@v1.17.1` with mage already pinned)** — mitigated. Exit 0, no stdout/stderr, `git diff go.mod go.sum` empty after invocation. Matches B1 fork acceptance.
    6. **Concurrent invocation** — accepted. Two simultaneous `mage addDep` invocations run as separate mage processes; serialization is the responsibility of `go get` itself (Go tooling holds a go.mod lock). Not mage target's concern. No finding.
    7. **D3 CLAUDE.md scope** — mitigated. `git diff HEAD -- CLAUDE.md` shows exactly one line added (line 210 in post-change file), the new mage targets table row. No other CLAUDE.md sections touched by the builder. § "Dependencies" prose (lines 262-263) was already updated pre-build by orch per PLAN.md Notes.
    8. **YAGNI** — mitigated. `AddDep` body is 3 functional lines + doc comment: one `sh.RunV` call, one wrapped error. No flags, no dry-run, no rollback, no verbose mode. Thin wrapper only.
    9. **Mage target count vs docs** — **FINDING** (small, see below). `mage -l` now lists 10 targets (addDep + build + ci + coverage + format + install + lint + planCheck + run + test). CLAUDE.md table also lists 10 rows. But `magefile.go:5` package doc comment still says "The nine canonical targets mirror the table in main/CLAUDE.md". The comment's own self-referential rule says *"any drift between that table and this file is a bug"*. After adding AddDep, the count "nine" is stale by one.
    10. **Error message shape** — mitigated. Wrap template is `"mage addDep %s: %w"` (magefile.go:90). Module name is in the message, `%w` chain preserves the underlying `go get` error for `errors.Is` / `errors.As`. Observed output matches. Actionable.
    11. **Exit code propagation** — mitigated. `sh.RunV` returns a non-nil error on subprocess failure; mage's target runner exits 1 when a target returns non-nil. Verified with attacks 2 and 3: both exit 1. Successful attack 5 exits 0.

- **Mage / shell invocations:**
    - `mage addDep` → exit 2, "not enough arguments for target "AddDep", expected 1, got 0"
    - `mage addDep github.com/magefile/mage@v1.17.1` → exit 0, no output, `git diff go.mod go.sum` empty
    - `mage addDep nonexistent.example.invalid/definitely-not-a-module@v0.0.0` → exit 1, wrapped error, `git diff go.mod go.sum` empty
    - `mage addDep 'foo; rm -rf /'` → exit 1, `go get` rejects malformed path, no shell interpretation
    - `mage ci` → exit 0, gofumpt clean, vet clean, golangci-lint 0 issues, tests pass
    - `mage -l` → 10 targets listed, addDep present
    - `git diff HEAD -- magefile.go CLAUDE.md` → only AddDep func + single CLAUDE.md table row

- **Findings:**
    - **F1 (attack 9) — magefile.go package doc drift.** `magefile.go:5` says "The nine canonical targets" but there are now ten after Unit 2.0. The doc comment's own rule flags this as a bug. Low severity; one-word fix (`nine` → `ten`) in the same file the builder touched. Recommend orch surface to dev; trivial to land in Unit 2.0 Round 2 or roll into a later unit's commit.
- **Accepted trade-offs:**
    - Concurrent `mage addDep` relies on `go get`'s own locking (attack 6). Not a mage-target concern.
    - Unit 2.0 `state:` in PLAN.md is still `todo` at QA time — reading Phase 4 step 3 ("builder flips state at end") vs. the per-unit state diagram ("done on builder + both QA pass") as being in tension; the state flip is conventionally orch's job after QA passes. Not flagging as a builder defect.
- **Hylla Feedback:**
    - N/A — Unit 2.0 touched only `magefile.go` (build-tagged `//go:build mage`, out of typical Hylla Go scope) and `CLAUDE.md` (markdown). All evidence came from `Read`, `Grep`, `Bash` (live mage execution), and direct source reading at `/Users/evanschultz/go/pkg/mod/github.com/magefile/mage@v1.17.1/sh/cmd.go`. Context7 was consulted for mage sh package semantics but returned only install docs, not `sh.RunV` source. No Hylla miss to record.
