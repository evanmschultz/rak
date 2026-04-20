# DROP_1 — CODE_SCAFFOLD_MAGE_CI

**State:** building
**Blocked by:** —
**Paths (expected):** `main/go.mod`, `main/go.sum`, `main/cmd/rak/main.go`, `main/cmd/rak/root.go`, `main/magefile.go`, `main/.github/workflows/ci.yml`, `/tmp/rak-stash/*` (source for move)
**Packages (expected):** `github.com/evanmschultz/rak/cmd/rak` (only package with Go code after Drop 1; `internal/*` packages land from Drop 2 onward)
**PLAN.md ref:** main/PLAN.md → `DROP_1_CODE_SCAFFOLD_MAGE_CI` row
**Workflow:** main/drops/WORKFLOW.md
**Started:** 2026-04-18
**Closed:** —

## Scope

Move the stashed `fwc` prototype at `/tmp/rak-stash/` into the rak layout under `main/`, rewrite the module path to `github.com/evanmschultz/rak`, split the flat `main.go` into `cmd/rak/main.go` (fang entry) + `cmd/rak/root.go` (cobra root), rewrite the root command for rak's shape (`rak [path]`, `MaximumNArgs(1)`, drop wc-style flags) with fang signal-to-context wiring, add `github.com/magefile/mage` dep, land `magefile.go` with the 9 canonical targets, and ship `.github/workflows/ci.yml` running `mage ci`. **No `internal/*` packages yet — `count(io.Reader)` stays unexported in `cmd/rak/root.go` for Drop 2.1 to lift into `internal/counting`.** Expected decomposition: 6 units (1.1–1.6) per main/PLAN.md.

## Prerequisites

Three dev-installed Go tools must be present on the dev machine before Unit 1.4 begins. All three are dev pre-state — install once, before Drop 1.4 starts; tools are not version-pinned in Drop 1 (deferred to Drop 9 — see main/PLAN.md § "Follow-Ups").

- **`mage`** — `go install github.com/magefile/mage@latest`. Required so the magefile landed in 1.5 has a runner.
- **`gofumpt`** — `go install mvdan.cc/gofumpt@latest`. Required by `mage format` and the `mage ci` empty-diff assertion in 1.5.
- **`golangci-lint`** — `curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v2.11.4` (upstream maintainers explicitly recommend against `go install` for golangci-lint — see Context7 `/golangci/golangci-lint` + local/docs/welcome/install/local.md). Required by `mage lint` (composed into `mage ci`) in 1.5.

If any tool is missing when Drop 1.4 / 1.5 starts, the builder pauses and surfaces the gap to the orchestrator rather than installing it from inside an agent.

## Planner

Six atomic units implementing the expected decomposition in main/PLAN.md lines 78–105. Dependency DAG (shortest-blocker form):

```
1.1 ──▶ 1.2 ──▶ 1.3
 │
 └──▶ 1.4 ──▶ 1.5 ──▶ 1.6
```

1.1 is the root (it creates the file layout everything else mutates). 1.2 and 1.4 both branch off 1.1 — 1.2 is the module-path rewrite (touches go.mod only), 1.4 is the mage dep add (touches go.mod + go.sum only). They do not serialize through each other, but both must land before their respective downstream units. 1.3 rewrites `cmd/rak/root.go` (needs 1.2 because the rewritten file will be imported by a compile check only meaningful once the module path is right). 1.5 (magefile.go) requires 1.4 (mage dep). 1.6 (CI workflow invokes `mage ci`) requires 1.5 (`mage ci` target must exist and pass).

### Unit 1.1 — Move stash into cmd/rak layout and split main.go

- **State:** done
- **Paths:**
  - `main/go.mod` (new — copied from `/tmp/rak-stash/go.mod`, unmodified in this unit)
  - `main/go.sum` (new — copied from `/tmp/rak-stash/go.sum`, unmodified)
  - `main/cmd/rak/main.go` (new — holds ONLY `package main` + `main()` calling `fang.Execute(context.Background(), newRootCmd())`)
  - `main/cmd/rak/root.go` (new — holds `newRootCmd()` + `Config`, `Counts`, `configFromCommand`, `run`, `count`, `printCounts` lifted verbatim from stash `main.go`; root command shape stays as stashed `fwc` for this unit — 1.3 rewrites the shape)
- **Packages:** `github.com/evanmschultz/rak/cmd/rak` (single Go package touched; module path remains stale `github.com/evanmschultz/coding_challenges/fang` in go.mod until 1.2 — this unit does NOT fix it)
- **Acceptance:**
  - `main/cmd/rak/main.go` exists; its file body is ≤ ~30 LOC; it contains exactly one function (`main`) whose body is `if err := fang.Execute(context.Background(), newRootCmd()); err != nil { os.Exit(1) }`.
  - `main/cmd/rak/root.go` exists; contains `newRootCmd() *cobra.Command` plus the helper types/funcs (`Config`, `Counts`, `configFromCommand`, `run`, `count`, `printCounts`) moved from stash `main.go` verbatim. Root command shape NOT yet rewritten (still `Use: "fwc [file]"`, `ExactArgs(1)`, wc flags) — 1.3 rewrites it.
  - `main/go.mod` + `main/go.sum` present in the `main/` working dir (not at repo root outside `main/`, not duplicated elsewhere). `go.mod` line 1 still reads `module github.com/evanmschultz/coding_challenges/fang` at this unit's exit — 1.2 rewrites it.
  - No `internal/*` directory created.
  - `grep -rn 'func main' main/cmd/rak/` returns exactly one line (in `main.go`).
  - `grep -rn 'func count(' main/cmd/rak/` returns exactly one line (unexported, in `root.go`).
  - `/tmp/rak-stash/` still present on disk (delete happens in Drop 1 closeout, not this unit).
- **Blocked by:** —

### Unit 1.2 — Rewrite go.mod module path to github.com/evanmschultz/rak

- **State:** done
- **Paths:** `main/go.mod`
- **Packages:** — (edits go.mod only; no Go source edits)
- **Acceptance:**
  - `main/go.mod` line 1 is exactly `module github.com/evanmschultz/rak`.
  - `grep -rn 'github.com/evanmschultz/coding_challenges/fang' main/ --include='*.go' --include='go.mod' --include='go.sum'` returns zero lines (scopes the invariant to code + module files, which is the real domain; `.md` planning/audit prose is intentionally out of scope — this unit does not rewrite history, and the acceptance bullet itself would be self-referential otherwise).
  - `grep -rn 'github.com/evanmschultz/fwc' main/ --include='*.go' --include='go.mod' --include='go.sum'` returns zero lines (same scope: guards against the mis-named `fwc` path main/PLAN.md line 82–83 + line 195 explicitly calls out landing in code or module files).
  - `mage build` not yet required (magefile.go doesn't exist yet); raw `go build ./...` is also forbidden per main/CLAUDE.md § "Build Verification" rule 2. Compile verification defers to the first unit that can run `mage build` (1.5). Until then, acceptance is grep-based.
- **Blocked by:** 1.1

### Unit 1.3 — Rewrite root command for rak shape + fang signal wiring

- **State:** done
- **Paths:** `main/cmd/rak/root.go`, `main/cmd/rak/main.go`
- **Packages:** `github.com/evanmschultz/rak/cmd/rak`
- **Acceptance:**
  - `main/cmd/rak/root.go` `newRootCmd()` returns a `*cobra.Command` with `Use: "rak [path]"`, `Args: cobra.MaximumNArgs(1)`, `Short` + `Long` describing rak, and a `RunE` whose body is exactly `return fmt.Errorf("not implemented — see drop 2")`. The stub keeps Drop 1's surface minimal and makes the Drop 2.3 hand-off cleanest (no count/print fork to rip out). The command must still execute without panic and honor `c.Context()` cancellation — the immediate error return satisfies both.
  - All wc-style flags from stash (`-b`, `-l`, `-w`, `-c`) are **removed** from `newRootCmd()` flag wiring. `grep -n 'BoolP' main/cmd/rak/root.go` returns zero lines (no old flags remain).
  - `count(io.Reader) (Counts, error)` remains **unexported** (lowercase `c`) and defined inside `main/cmd/rak/root.go`. `grep -n 'func Count(' main/cmd/rak/root.go` returns zero lines. `grep -n 'func count(' main/cmd/rak/root.go` returns exactly one line. `grep -n 'type Counts struct' main/cmd/rak/root.go` returns exactly one line — the `Counts` struct is the second half of the Drop 2.1 hand-off boundary alongside `count` and must survive intact for the lift into `internal/counting`. **This is the first-drop hand-off boundary pinned in main/PLAN.md line 86–87.**
  - `RunE` body is exactly `return fmt.Errorf("not implemented — see drop 2")` (verifiable via `grep -n 'not implemented — see drop 2' main/cmd/rak/root.go` returning exactly one line). No alternate count-and-print body permitted in Drop 1.
  - `main/cmd/rak/main.go` `fang.Execute` call passes `fang.WithNotifySignal(os.Interrupt, syscall.SIGTERM)` as an option. Exact call shape: `fang.Execute(context.Background(), newRootCmd(), fang.WithNotifySignal(os.Interrupt, syscall.SIGTERM))`. `grep -n 'WithNotifySignal' main/cmd/rak/main.go` returns exactly one line. `grep -n 'syscall.SIGTERM' main/cmd/rak/main.go` returns exactly one line.
  - `main/cmd/rak/main.go` imports `os`, `syscall`, `context`, `github.com/charmbracelet/fang` (and does NOT import `github.com/spf13/cobra` — that import lives in `root.go`).
  - `RunE` (or any future goroutine-like path in `root.go`) threads `c.Context()` down rather than inventing a fresh `context.Background()` — satisfies main/PLAN.md decision 29 ("`RunE` threads `cmd.Context()` downward") and the prereq-for-Drop-8.1 note from main/PLAN.md line 88–89. For the Drop 1 stub this is a forward-looking constraint on the file shape; the stub itself returns immediately.
  - Obsolete helper types and funcs that no longer serve the rewritten command surface (e.g. the full `Config` struct's wc-mode flags, `configFromCommand`'s flag-parsing branches, `printCounts` formatting) MAY be simplified or deleted in this unit — but `count(io.Reader) (Counts, error)` and the `Counts` struct MUST survive intact for Drop 2.1 to lift.
  - File size: `main/cmd/rak/root.go` stays ≤ ~150 LOC (main/CLAUDE.md § "Project Structure" file breakdown target).
- **Blocked by:** 1.2

### Unit 1.4 — Add mage dependency via `go get`

- **State:** done
- **Paths:** `main/go.mod`, `main/go.sum`
- **Packages:** — (dep add only; no Go source edits)
- **Acceptance:**
  - `grep -n 'github.com/magefile/mage' main/go.mod` returns at least one line (the line appears in the indirect `require` block and carries the `// indirect` marker — this is expected; 1.5's magefile.go imports `github.com/magefile/mage/mg` which promotes it to direct via `go mod tidy` in 1.5).
  - The dep is added via `go get github.com/magefile/mage` run from `main/` — NOT hand-edited. **This is the one `go get` invocation this project allows outside `mage` per main/CLAUDE.md § "Go Development Rules" → "Dependencies" → "Bootstrap carve-out"** (no mage target exists yet because magefile.go itself is added next in 1.5, which is the very condition the carve-out covers; from Drop 2 onward the magefile exists and every dep add routes through a mage target). Actor: builder by default; dev (or dev-authorized orchestrator) if the sandbox denies `go get` to the builder.
  - **`go mod tidy`'s stability-assertion verification is deferred to Unit 1.5.** `go mod tidy` with `go 1.26.1` prunes any module no source file imports; since no `.go` file imports `github.com/magefile/mage/mg` until 1.5's magefile.go lands, `tidy` re-run against 1.4's end-state would strip mage. The stability assertion (re-running `tidy` produces no diff + mage persists) is therefore only meaningful once magefile.go exists. 1.5's acceptance owns that assertion.
  - **Unit-end state is a `go get`-restored state, not a tidy-stable state.** During the unit `go mod tidy` ran as a bloat-prune side-effect (dev-run twice — see BUILDER_WORKLOG.md Unit 1.4 Round 1 "Commands run" steps 4 + 5) which pruned fwc's inherited indirect transitives from `go.sum` and shaped `go.mod` into the go-1.17+ split direct/indirect block; the same tidy also stripped mage because no source imports `mg` yet. A subsequent `go get github.com/magefile/mage` (step 7) re-added mage as `// indirect` — that restoration, not tidy, is the final module-file mutation of the unit. If `go mod tidy` were re-run now it would strip mage again. Tidy stability re-enters the picture in 1.5, where magefile.go's `mg` import holds mage through tidy and promotes it to the direct `require` block.
  - `main/go.sum` contains lines for `github.com/magefile/mage` (`grep -c 'github.com/magefile/mage' main/go.sum` ≥ 1).
  - Module path line 1 of `main/go.mod` still reads `module github.com/evanmschultz/rak` (this unit does not regress 1.2).
- **Blocked by:** 1.1

### Unit 1.5 — Add magefile.go with 9 canonical targets

- **State:** done
- **Paths:** `main/magefile.go`, optionally `main/.golangci.yml` (see fallback clause below)
- **Packages:** `main` (the magefile lives at the `main/` module root and therefore declares `package main` per Go's one-package-per-directory rule; the `//go:build mage` build tag excludes it from the normal build surface).
- **Acceptance:**
  - `main/magefile.go` exists with `//go:build mage` (or `// +build mage`) build tag on line 1 so it is excluded from normal builds.
  - File declares package `main` and imports `github.com/magefile/mage/mg` (and `github.com/magefile/mage/sh` as needed). `grep -n 'github.com/magefile/mage/mg' main/magefile.go` returns ≥ 1 line.
  - `mage -l` run from `main/` lists exactly the 9 targets enumerated in main/CLAUDE.md § "Build Verification" mage target table: `build`, `test`, `format`, `lint`, `ci`, `install`, `run`, `coverage`, `planCheck`. No extra targets, no missing targets.
  - Each target's command maps to main/CLAUDE.md § "Build Verification" table exactly:
    - `build` → `go build ./...`
    - `test` → `go test -race ./...`
    - `format` → `gofumpt -l -w .`
    - `lint` → `go vet ./...` then `golangci-lint run` (both must run; failure of either fails `lint`)
    - `ci` → assert `gofumpt -l .` output is empty, then run `mage lint`, then `mage test` (in that order; any fail fails `ci`)
    - `install` → `go install ./cmd/rak` — **dev-only**, not a dep of `mage ci`. Target comment must say "dev-only; agents MUST NOT invoke." (grep-verifiable).
    - `run` → `go run ./cmd/rak` with positional args passing through after `--`
    - `coverage` → `go test -race -coverpkg=<scope> -coverprofile=coverage.out ./... && go tool cover -func=coverage.out`. **Report-only in Drop 1** — target comment must say "report-only until Drop 9.3" (grep-verifiable). Builder picks one of two variants for `<scope>`:
      - **(a)** `-coverpkg=./internal/...` — kept strict; acceptable if `mage coverage` exits 0 in Drop 1 even though zero packages match. Target body has no `TODO` re scope.
      - **(b)** `-coverpkg=./...` — paired with `// TODO(drop-9.3): tighten to ./internal/... once internal/ exists` comment on the target body.
      - **Acceptance (whichever variant chosen):** `mage coverage` exits 0 AND the comment/`TODO` state in the magefile matches the variant picked (variant (a) → no scope-tighten TODO; variant (b) → the TODO is present). Internal consistency between the `-coverpkg` flag and the comment is what QA verifies.
    - `planCheck` → diffs `main/PLAN.md` container titles + states against `main/drops/*/` directory names + each drop dir's `PLAN.md` header state; fails if drift (may be implemented as a stub that always passes in Drop 1 — real parity-check logic is acceptable later; the TARGET's existence + `mage -l` listing is what Drop 1 acceptance requires).
  - **`go mod tidy` run from `main/` (deferred from 1.4 per Unit 1.4 acceptance + Notes entry "go mod tidy stability-assertion deferred to 1.5")** leaves `go.mod` + `go.sum` stable (re-running produces no diff). First tidy here is expected to promote `github.com/magefile/mage` from `// indirect` to the direct `require` block because the just-written magefile.go imports `github.com/magefile/mage/mg`. `grep -n 'github.com/magefile/mage' main/go.mod` after tidy must still return ≥ 1 line AND the line must NOT carry the `// indirect` marker.
  - `mage build` exits 0 (first real compile check; validates 1.1 + 1.2 + 1.3 + 1.4 + 1.5 all compile together).
  - `mage test` exits 0 (there are no `*_test.go` files in Drop 1 — `go test -race ./...` on a package with no tests exits 0 with "[no test files]" output; this is an acceptance check that the target is wired right, not that tests exist).
  - `mage format` exits 0 and produces no diff (verifies `gofumpt -l -w .` is idempotent on the freshly-written code).
  - `mage lint` exits 0 (requires `golangci-lint` + `go vet` to find no issues on the Drop 1 surface). **Fallback clause:** if `golangci-lint run` fails on the Drop 1 surface purely because of default-linter strictness (no `.golangci.yml` exists yet), builder commits a minimal `main/.golangci.yml` enabling only the default linter set and documents the rationale (which default linter fired, why the minimal config is the smallest fix) in `BUILDER_WORKLOG.md`. Empty default (no `.golangci.yml`) is the preferred outcome and should be tried first; the minimal config is a graceful escape valve, not the target state.
  - `mage ci` exits 0 (end-to-end local gate passes).
  - **Agents MUST NOT invoke `mage install`** — acceptance check is the comment text in the target, not an execution.
- **Blocked by:** 1.4

### Unit 1.6 — Add .github/workflows/ci.yml running mage ci

- **State:** done
- **Paths:** `main/.github/workflows/ci.yml`
- **Packages:** — (YAML only, non-Go file)
- **Acceptance:**
  - `main/.github/workflows/ci.yml` exists.
  - Workflow triggers on `push` to `main` and `pull_request` targeting `main`. `grep -n 'push:' main/.github/workflows/ci.yml` returns ≥ 1 line; `grep -n 'pull_request:' main/.github/workflows/ci.yml` returns ≥ 1 line.
  - Workflow's job runs on `ubuntu-latest`, checks out the repo, installs Go 1.26+ (matches `main/go.mod` `go 1.26.1` line — pinning to `1.26.x` via `actions/setup-go` is acceptable), installs `mage`, installs `gofumpt` and `golangci-lint` (the tools `mage ci` invokes), then runs `mage ci` from `main/`. `grep -n 'mage ci' main/.github/workflows/ci.yml` returns ≥ 1 line.
  - Workflow does NOT include a coverage gate — `mage coverage` is report-only per decision 22 + main/PLAN.md line 104–105. `grep -ni 'coverage' main/.github/workflows/ci.yml` may return 0 lines in Drop 1 (no coverage step), or may return a report-only step that does NOT fail the build on threshold — if present, the step's failure-on-threshold MUST be absent. Drop 9.3 flips the gate on.
  - `mage install` is NOT invoked anywhere in the workflow (agents-must-not-run rule).
  - YAML parses as a valid GitHub Actions workflow. Acceptance is verifiable via `gh workflow view` after the workflow file lands on the pushed branch (soft check; equivalent to GitHub's own validation accepting the file). `yamllint` or a schema check is not required for Drop 1.
  - **Note:** `gh run watch --exit-status` on the triggered workflow run is **not** a 1.6 unit acceptance criterion — it is drop-end verification (WORKFLOW.md § "Phase 6 — Verify"). Unit 1.6 passes when the YAML is well-formed and the assertions above hold; the green CI run is verified by orch in Phase 6, not by the builder for this unit.
- **Blocked by:** 1.5

## Notes

- **Stash lifecycle.** `/tmp/rak-stash/main.go`, `go.mod`, `go.sum` are consumed by 1.1 and 1.4. `test.txt` (342KB fixture) and `PLAN.md` (obsolete fwc plan) are explicitly NOT copied (main/PLAN.md § "Stashed Legacy Files" lines 196–198). Orchestrator deletes the entire `/tmp/rak-stash/` directory in Drop 1's closeout (Phase 7), not inside any unit.
- **`go mod tidy` stability-assertion deferred to 1.5 (ordering hole).** Original plan asserted tidy stability inside 1.4. Build-time discovery: with `go 1.26.1`, `go mod tidy` prunes any module no source file imports — so tidy against 1.4's end-state strips `github.com/magefile/mage` because the `mg` import doesn't land until 1.5's magefile.go. Tidy did run twice during the unit as a bloat-prune side-effect (fwc-inherited indirect transitives pruned from `go.sum`; `go.mod` shaped into go-1.17+ split direct/indirect block), but its stability assertion was not actionable at 1.4's end-state — a subsequent `go get` was needed to restore mage, leaving the unit in a `go get`-restored rather than tidy-stable state. 1.5 is the first point where tidy can settle (magefile.go's `mg` import holds mage through tidy + promotes it to direct). Plan amended on 2026-04-19 post-QA-green; 1.5 acceptance absorbs the stability assertion and adds a direct-vs-indirect promotion check.
- **go.sum drift.** 1.1 copies stash `go.sum` unmodified. 1.5's `go mod tidy` (moved from 1.4 — see above) will prune the huge indirect-dep list in stash `go.sum` (stash was fwc's, which pulled laslig transitively — rak Drop 1 only needs fang + cobra + mage directly). Expect a large `go.sum` diff in 1.5; this is normal and not a 1.1 or 1.4 regression.
- **No laslig import in Drop 1.** Stash `go.mod` lists `github.com/evanmschultz/laslig v0.2.4` as indirect. Drop 1 does not import laslig directly (rendering lands in Drop 2.2). `go mod tidy` in 1.5 will likely drop laslig from `go.sum` since nothing in the tree imports it. This is expected; laslig re-enters the dep list in Drop 2.2.
- **`install` target is a tripwire.** 1.5's `mage install` target exists so the dev can dogfood rak; the "agents MUST NOT invoke" comment on the target and the absence of any dep chain from `mage ci` into `install` are both acceptance-checked, but the single strongest guard is convention — every agent's spawn preamble forbids it. The target is here for the dev, not for CI or agents.
- **`planCheck` in Drop 1 can be a stub.** A real diff between `main/PLAN.md` container titles and `main/drops/*/PLAN.md` header states is nontrivial (parser, state diffing). Drop 1 acceptance is target presence + `mage -l` listing; implementing real parity logic is acceptable here or can be deferred to a follow-up drop (add to main/PLAN.md follow-ups if deferred). If stubbed, target body is `// TODO(planCheck): real parity check — stub passes in Drop 1` and exits 0.
- **Drop 2.1 hand-off boundary (pinned).** Do NOT export `count`, do NOT move it out of `cmd/rak/root.go`, do NOT create `internal/counting/` in this drop. Drop 2.1's planner owns those. The pinned surface for Drop 2.1 to lift is the unexported `count(io.Reader) (Counts, error)` function plus the `Counts` struct, both in `main/cmd/rak/root.go`.
- **Coverage scope footnote.** `mage coverage` uses `-coverpkg=./internal/...` per decision 22. In Drop 1 there are zero `internal/*` packages, so the flag matches zero packages. Verify the behavior: Go 1.26 `go test -coverpkg=./internal/... ./...` with zero matching packages produces either an empty profile or a no-op — either is acceptable. If it errors, use `-coverpkg=./...` in Drop 1 with `// TODO(drop-9.3): tighten to ./internal/... once internal/ exists` and tighten at gate-flip time. This variant choice is captured in 1.5's acceptance under the `coverage` target bullet.
- **Tool-version pinning deferred.** `gofumpt` + `golangci-lint` ship unpinned in Drop 1 (both as dev pre-state per § "Prerequisites" and in 1.6's CI install step). Pinning is tracked in main/PLAN.md § "Follow-Ups" → "Pin `gofumpt` + `golangci-lint` versions in Drop 9". This is the authoritative location; the entry is not duplicated here.
