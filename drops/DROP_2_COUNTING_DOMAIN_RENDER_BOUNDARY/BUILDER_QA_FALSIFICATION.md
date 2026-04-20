# DROP_2 — Builder QA Falsification

Append a `## Unit N.M — Round K` section per QA attempt. See `main/drops/WORKFLOW.md`.

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
