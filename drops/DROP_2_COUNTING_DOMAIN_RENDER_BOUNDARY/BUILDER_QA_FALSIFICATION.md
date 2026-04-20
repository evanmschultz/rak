# DROP_2 — Builder QA Falsification

Append a `## Unit N.M — Round K` section per QA attempt. See `main/drops/WORKFLOW.md`.

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
