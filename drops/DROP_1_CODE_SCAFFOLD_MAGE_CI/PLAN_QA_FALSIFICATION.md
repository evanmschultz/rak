# DROP_1 Plan QA — Falsification (Round 1)

**Reviewer:** go-qa-falsification-agent
**Reviewed:** 2026-04-18
**Plan SHA at review:** 2fa8bf8

## Verdict

**fail**

Mage targets are exported Go functions; their CLI names case-fold the function identifier and namespaces use a colon separator. A target literally named `plan-check` (as Unit 1.5 acceptance line 96 + 106 demand) cannot be registered in mage at all — Go function names cannot contain hyphens, and mage's namespace separator is `:`, not `-`. The acceptance criterion "`mage -l` lists … `plan-check`" is unfalsifiable-by-construction, and the parallel string in `main/CLAUDE.md` § "Build Verification" propagates the same defect into the canonical mage table, so the builder cannot pass Unit 1.5 acceptance without rewriting both the plan and CLAUDE.md.

## Counterexamples

### Counterexample 1 — `plan-check` is not a valid mage target name (blocker)

- **Severity:** blocker
- **Unit reference:** 1.5 (acceptance line 96 + line 106; ripples into `main/CLAUDE.md` line 175 + line 213)
- **Scenario:**
  1. Builder writes `magefile.go` with `func PlanCheck() error { … }` (the closest legal Go function name).
  2. `mage -l` from `main/` lists the target as `planCheck` (mage case-folds the leading capital to lowercase for CLI display).
  3. Unit 1.5 acceptance check 96 — "`mage -l` … lists exactly the 9 targets enumerated in main/CLAUDE.md § 'Build Verification' mage target table: `build`, `test`, `format`, `lint`, `ci`, `install`, `run`, `coverage`, `plan-check`. **No extra targets, no missing targets.**" — fails because `plan-check` is missing and `planCheck` is "extra".
  4. The contradiction is structural: there is no Go-legal function name that mage will surface as the literal string `plan-check`. The closest mage-supported alternative is a namespaced target (e.g. type `Plan mg.Namespace; func (Plan) Check()`), which mage CLI invokes as `mage plan:check` (colon, not hyphen) — still not `plan-check`.
- **Evidence:**
  - Drop's `PLAN.md` line 96: `'mage -l' run from main/ lists exactly the 9 targets … 'build', 'test', 'format', 'lint', 'ci', 'install', 'run', 'coverage', 'plan-check'`.
  - Drop's `PLAN.md` line 106: `'plan-check' → diffs main/PLAN.md container titles + states …`.
  - `main/CLAUDE.md` line 175: row in mage target table reads `| 'mage plan-check' | diff main/PLAN.md container titles + states … |`.
  - `main/CLAUDE.md` line 213: same string.
  - Mage upstream docs (Context7 `/magefile/mage`):
    - "Build target is any exported function with zero args with no return or an error return." (mage requires Go-exported function names; Go function names are `[A-Za-z_][A-Za-z0-9_]*` — no hyphens.)
    - Namespaces example: `type Build mg.Namespace; func (Build) Site()` invoked as `$ mage build:site` (colon separator, not hyphen).
- **Suggested mitigation:** Decide one of:
  1. Rename the target to `planCheck` (function `PlanCheck`) and update PLAN.md + CLAUDE.md accordingly. CLI becomes `mage planCheck`. **Recommended** — single Go-idiomatic function, no namespace ceremony.
  2. Use a `Plan` namespace: `type Plan mg.Namespace; func (Plan) Check()`. CLI becomes `mage plan:check`. Heavier-weight; only worth it if a `Plan` namespace will hold sibling targets later.
  3. Drop the `plan-check` target entirely from Drop 1 and defer it to a follow-up drop (Drop 9 release / docs would be a natural home). Note: 1.5 acceptance and 9.x ordering both need to mention the deferral.

  Plan-side change: reword 1.5 acceptance bullet 96 to list whichever of the above is chosen. Down-doc change: edit `main/CLAUDE.md` line 175 and § "Build Verification" → "Mage targets" table to match. Both edits are pure-markdown — no Go code.

### Counterexample 2 — `mage install`, the dev-only tripwire, is in the canonical 9-target list and is exercised by `mage -l` parity but documented as forbidden (note → blocker if a builder treats it as runnable)

- **Severity:** nit (the plan handles this correctly via comment-text acceptance; logging because it is a known sharp edge)
- **Unit reference:** 1.5 (acceptance line 103 + 112; reinforced by Notes line 134)
- **Scenario:**
  1. `mage -l` lists `install` alongside the other 8 targets (1.5 acceptance line 96 demands all 9 are listed).
  2. A future agent reads `mage -l` output, sees `install`, and runs it — promoting a binary into `$GOBIN` from a non-dev session. CLAUDE.md § "Build Verification" line 200 says "**NEVER run `mage install` from an agent.** This is a **dev-only** dogfood target."
  3. Plan mitigates via comment-text acceptance ("Target comment must say 'dev-only; agents MUST NOT invoke.'") + spawn-preamble convention. But the mitigation is a comment string in the magefile, not a build-time gate.
- **Evidence:**
  - Drop's `PLAN.md` line 103: `'install' → 'go install ./cmd/rak' — **dev-only**, not a dep of 'mage ci'. Target comment must say "dev-only; agents MUST NOT invoke." (grep-verifiable).`
  - Drop's `PLAN.md` line 112: `**Agents MUST NOT invoke 'mage install'** — acceptance check is the comment text in the target, not an execution.`
  - `main/CLAUDE.md` line 200: `**NEVER run 'mage install' from an agent.**`
  - WORKFLOW.md preamble (line 60–83) does not mention `mage install` — relies on the per-CLAUDE.md rule.
- **Suggested mitigation:** Acceptable as-is for Drop 1. If you want a stronger gate, add an environment-variable guard inside `Install()`:
  ```go
  func Install() error {
      if os.Getenv("RAK_AGENT") != "" { return errors.New("install is dev-only") }
      …
  }
  ```
  and require every agent spawn to set `RAK_AGENT=1`. That is a Drop-2+ refinement, not a Drop 1 blocker. Logging here so it is visible in the round-1 record.

### Counterexample 3 — Stash `go.mod` ships an `indirect` line for `github.com/charmbracelet/fang` while stash `main.go` directly imports it; `go mod tidy` will rewrite it (note)

- **Severity:** nit
- **Unit reference:** 1.4 (acceptance lines 81–84) interacts with 1.1 (acceptance line 40)
- **Scenario:**
  1. Unit 1.1 copies `/tmp/rak-stash/go.mod` verbatim. Stash `go.mod` line 11: `github.com/charmbracelet/fang v1.0.0 // indirect`. But stash `main.go` line 13 directly imports `github.com/charmbracelet/fang`. The stash's `go.mod` is therefore already lying about indirectness.
  2. After 1.2 rewrites the module path and 1.4 runs `go get github.com/magefile/mage` + `go mod tidy`, tidy will move `fang` and `cobra` out of `// indirect` and onto direct-require lines, AND prune the entire transitive zoo (laslig + lipgloss + glamour + ~30 others) from `go.sum` (these are not imported by anything in Drop 1).
  3. The `go.mod` and `go.sum` diffs from `go mod tidy` will be enormous — easily 30+ removed lines from each — and a reviewer skimming the post-1.4 commit could flag the diff size as suspicious.
- **Evidence:**
  - `/tmp/rak-stash/go.mod` line 11: `github.com/charmbracelet/fang v1.0.0 // indirect`.
  - `/tmp/rak-stash/main.go` line 13: `"github.com/charmbracelet/fang"` (direct import).
  - `/tmp/rak-stash/go.sum` is 107 lines of largely transitive-dep hashes; only `fang` + `cobra` (+ their direct deps `pflag`, `mousetrap`, etc.) survive a `go mod tidy` against the stash source.
  - Drop's `PLAN.md` Notes line 132 acknowledges this in passing: "1.4's `go mod tidy` will likely prune the huge indirect-dep list".
- **Suggested mitigation:** Add a one-line acceptance criterion to 1.4 confirming the post-tidy diff: `'grep -c "// indirect" main/go.mod' returns a small number (≤ 5; the stash's 30+ indirect lines are pruned).` Also, add a short callout in `BUILDER_WORKLOG.md` for unit 1.4 noting the diff size so the QA reviewer is not surprised. The plan's existing Notes section already softens this; an explicit acceptance check would be belt-and-suspenders.

### Counterexample 4 — `golangci-lint` and `gofumpt` versions are not pinned; CI vs local can drift (note)

- **Severity:** note
- **Unit reference:** 1.5 (acceptance line 110 — `mage lint` exits 0) and 1.6 (acceptance line 123 — workflow installs the tools)
- **Scenario:**
  1. Builder runs `mage lint` locally with `golangci-lint v2.0.4` (whatever the dev installed). Passes.
  2. CI workflow installs `golangci-lint` via `actions/setup-go` or a separate step without pinning. CI gets `v2.1.0` released the next day, which adds a new default-enabled linter (e.g. `intrange`).
  3. CI fails on Drop 1 surface that locally passed. False failure attributed to the builder.
- **Evidence:**
  - Drop's `PLAN.md` line 123: "installs `mage`, installs `gofumpt` and `golangci-lint`" — no version constraint.
  - CLAUDE.md § "Tech Stack" lines 188–191: "Dev tooling (installed locally, invoked via mage): `mvdan.cc/gofumpt` …, `github.com/golangci/golangci-lint/cmd/golangci-lint` …" — no version pin.
  - Default linter sets in `golangci-lint` change across minor releases per their release notes (well-known historical churn).
- **Suggested mitigation:** Add to 1.6 acceptance: "workflow pins `golangci-lint` to a specific version (e.g. `v2.0.x`) and `gofumpt` to a specific tag, OR uses `golangci-lint-action@v8` (which has its own version-pin parameter)." Add a parallel local-pin discipline note to CLAUDE.md § "Tech Stack" so dev installs match CI. Drop 1 can ship without this and accept the risk for the first cycle, but it is a real divergence source that will eventually bite. Defer to a follow-up if dev prefers.

### Counterexample 5 — Plan does not require a `.golangci.yml` config; relies on golangci-lint defaults (note)

- **Severity:** note
- **Unit reference:** 1.5
- **Scenario:**
  1. No `.golangci.yml` is required by 1.5 acceptance.
  2. golangci-lint v2 default linter set includes `errcheck` + `staticcheck` + `unused` + `govet` + `ineffassign` + `gosimple` (per current upstream defaults).
  3. Stash code under refactor: `printCounts` calls `fmt.Fprintln(w, …)` discarding the `int` return — `errcheck` will flag if its default config has `--fmt-default` set… in practice `errcheck` skips `fmt.*` writes by default, so this specific case is fine, but the general point stands: untested-by-the-plan defaults will gate the build.
  4. Builder may be forced to add a `.golangci.yml` mid-1.5 to silence a default-on linter that disagrees with the stash code, which expands 1.5's surface area beyond what acceptance specifies.
- **Evidence:**
  - Drop's `PLAN.md` lines 88–113: no `.golangci.yml` listed in `Paths`, no acceptance criterion mentions config.
  - CLAUDE.md § "Tech Stack" + § "Build Verification": no `.golangci.yml` mentioned.
  - golangci-lint v2 enables `errcheck`, `gosimple`, `govet`, `ineffassign`, `staticcheck`, `unused` by default (per upstream `.golangci.reference.yml`).
- **Suggested mitigation:** Either:
  1. Add an explicit acceptance line to 1.5: "If `mage lint` fails on default linters, builder adds a minimal `main/.golangci.yml` disabling specific rules with one-line per-rule justifications. This file is NEW and counts as part of 1.5."
  2. OR pre-commit a minimal `.golangci.yml` as part of 1.5's `Paths`.
  3. OR accept the risk — stash code is small and clean, defaults will probably pass.

  Recommend option 1 — keeps the plan honest if the builder hits a default-linter snag, without bloating Drop 1 with a config nobody needs yet.

### Counterexample 6 — Tool prerequisites (gofumpt, golangci-lint) are not enumerated as dev pre-state (note)

- **Severity:** note
- **Unit reference:** 1.5 (acceptance lines 109–110)
- **Scenario:**
  1. Dev launches an orchestrator from `main/` to run Drop 1 on a fresh machine that lacks `gofumpt` and `golangci-lint`.
  2. Builder reaches Unit 1.5, writes `magefile.go`, runs `mage ci`. `gofumpt -l .` fails with `command not found`.
  3. Builder cannot install the tools — `mage install` is forbidden, and `go install <tool>@latest` is "raw `go` invocation" forbidden by CLAUDE.md § "Build Verification" rule 2 (line 198).
  4. Builder is stuck. Has to escalate to dev.
- **Evidence:**
  - Drop's `PLAN.md` lines 88–113: no pre-state check, no "Builder verifies gofumpt + golangci-lint are on PATH" step.
  - CLAUDE.md line 188 + 191: "Dev tooling (installed locally, invoked via mage)" — implicit dev pre-state.
  - CLAUDE.md line 198: forbids raw `go` invocations including `gofumpt` and `golangci-lint`.
  - 1.4 acceptance line 82 does explicitly carve out `go get github.com/magefile/mage` as the one sanctioned raw-go invocation. No such carve-out for tool installs.
- **Suggested mitigation:** Add a pre-build note to the drop's `PLAN.md` § "Notes": "Before Phase 4 starts, dev confirms `gofumpt -l .` and `golangci-lint --version` both run successfully from `main/`. If either is missing, dev runs `go install mvdan.cc/gofumpt@latest && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest` (sanctioned dev-only setup, not subject to mage discipline)." This is a one-line note, not a unit. Alternatively: add a `mage tools` (or `mage bootstrap`) target whose body installs both via `sh.Run("go", "install", …)`. That requires another mage target slot and re-opens the 9-target acceptance count — not Drop 1's job.

  For Drop 1, adding the dev pre-state Note is the lightest fix.

### Counterexample 7 — Unit 1.3 RunE has two legal shapes; the "not implemented" stub silently regresses runnability that 1.1+1.2 did not yet establish, and the "minimal count-and-print" shape is under-specified (nit)

- **Severity:** nit
- **Unit reference:** 1.3 (acceptance line 65, "minimal RunE … OR return `fmt.Errorf("not implemented — see drop 2")`")
- **Scenario:**
  1. Builder picks the `fmt.Errorf("not implemented — see drop 2")` stub for 1.3.
  2. `mage build` (run during 1.5) compiles cleanly. `mage test` passes (no tests). `mage ci` passes.
  3. `mage run -- /tmp/rak-stash/main.go` exits 1 with "not implemented" — completely correct against 1.3's acceptance, but means rak does nothing in Drop 1 by design.
  4. A reasonable QA Proof reviewer might assert the stub is fine for Drop 1 (because the next drop is 2.1 which moves `count` into `internal/counting`). A reasonable QA Falsification reviewer (me) flags that the *other* legal shape — "open the path argument or stdin, call `count(r)`, print to `c.OutOrStdout()`" — has no `printCounts`-equivalent specified after Unit 1.3 line 71 says "`printCounts` formatting MAY be deleted". Without a print specification, two builders pick two different print shapes.
  5. This produces non-deterministic Drop 1 output that QA cannot pin down.
- **Evidence:**
  - Drop's `PLAN.md` line 65: "minimal `RunE` (for Drop 1 a stub is acceptable — e.g. open the path argument or stdin, call `count(r)`, print to `c.OutOrStdout()`; OR return `fmt.Errorf("not implemented — see drop 2")` …)"
  - Drop's `PLAN.md` line 71: "`printCounts` formatting MAY be simplified or deleted in this unit"
- **Suggested mitigation:** Pick one shape in 1.3 acceptance and commit:
  - **Recommended:** the `fmt.Errorf("not implemented — see drop 2")` stub — strictly less surface area, defers all output decisions to Drop 2.2 which owns the laslig render. This also aligns with the "smallest concrete design" rule in CLAUDE.md § "Go Development Rules" → "Structure + Style".
  - If the print-shape is preferred for early dogfooding, replace 1.3 acceptance line 65's "OR" with "AND for Drop 1 use this exact format: `fmt.Fprintf(w, "lines: %d  words: %d  bytes: %d  chars: %d\n", c.Lines, c.Words, c.Bytes, c.Chars)`" — then the print shape is testable.

### Counterexample 8 — Unit 1.6 acceptance for "workflow YAML is syntactically valid" relies on GitHub's post-push validation; locally unverifiable (nit)

- **Severity:** nit
- **Unit reference:** 1.6 (acceptance line 126: "the workflow must parse — `gh workflow view` or GitHub's own validation on the pushed branch serves as the yes/no for YAML correctness")
- **Scenario:**
  1. Builder writes `.github/workflows/ci.yml` with a typo (e.g. `runs-on: ubuntu-latests`).
  2. Local QA cannot detect — there is no `mage` target for YAML lint, and `yamllint` is not in the toolchain per CLAUDE.md.
  3. Builder commits, orch pushes, CI fails to even start the run because the workflow YAML is malformed — `gh run watch --exit-status` returns a not-yet-started or schema-error state.
  4. The drop-end verification fails on a typo, builder is back to Phase 5 for 1.6.
- **Evidence:**
  - Drop's `PLAN.md` line 126: explicit acknowledgment that local YAML validation is not required.
  - No `mage yaml-lint` target exists in the 9-target list.
- **Suggested mitigation:** Add a one-liner to 1.6 acceptance: "Builder runs `go run github.com/rhysd/actionlint/cmd/actionlint@latest .github/workflows/ci.yml` (or installs actionlint locally and runs it via `mage lint` extension) before declaring 1.6 done." This catches typos pre-push. Alternatively, accept the risk and let CI failure trigger the round-2 loop — costs one round of CI time but is simpler. For Drop 1, simpler is fine; logging here so the dev sees the choice.

## Attack Surfaces Explored

- **Ordering hazards** — checked DAG (1.1 → {1.2, 1.4} → {1.3, 1.5} → 1.6). 1.2 and 1.4 are parallel-eligible after 1.1 — does running 1.4 before 1.2 break anything? Answer: no, `go mod tidy` succeeds against any module path. **No counterexample.**
- **Contract mismatch with Drop 2.1** — checked stash `Counts` struct fields (`Bytes/Lines/Words/Chars`) against Drop 2.1 expected `Count(io.Reader) (Counts, error)` with `bytes/lines/words/chars`. Match. **No counterexample.** 1.3 acceptance correctly pins the hand-off boundary.
- **YAGNI pressure** — checked the magefile target list (9 targets) against actual Drop 1 + drop-9.3 + drop-end coverage need. `coverage` (report-only) is justified by decision 22. `plan-check` and `install` are also justified. **No YAGNI counterexample.** (`plan-check`'s name is broken — see Counterexample 1.)
- **Hidden dependencies** — checked for `.golangci.yml`, `Makefile`, `.golangci-lint-version`, `tools.go`, etc. — none mentioned. Default golangci-lint is implied. **See Counterexamples 4 + 5 + 6.**
- **Mage-discipline bypasses** — re-read every acceptance bullet. The only sanctioned non-`mage` Go invocation is 1.4's `go get github.com/magefile/mage` (acceptance line 82) — explicitly carved out per CLAUDE.md § "Dependencies". No other unit asks for raw `go`. **No counterexample on bypass.** (Tool install gap is dev-pre-state, not bypass — Counterexample 6.)
- **Decision drift vs main/PLAN.md § "Decisions Locked In"** — checked 5 (layout), 6 (tech stack), 15 (orch never edits Go), 16 (drop dir model), 17 (single root), 18 (stdin), 21 (no spinner Drops 1–7), 22 (coverage report-only Drop 1.5), 23 (CI in first drop), 25 (fileset.File contract — Drop 3, irrelevant here), 26 (v0.1.0 cuts), 27 (architecture, file-size limits), 28 (quality tooling), 29 (concurrency + errors). Plan honors all. **No counterexample on decision drift.**
- **Acceptance-check falsifiability** — every Unit 1.1–1.6 bullet is grep-based or exit-code-based. Counterexample 1 is the exception — `plan-check` listing in `mage -l` is unfalsifiable-by-construction. Also surfaced softer falsifiability concerns in Counterexamples 7 (RunE stub-vs-print under-specified) and 8 (YAML correctness deferred to GitHub).
- **Cross-unit tool readiness** — gofumpt + golangci-lint pre-state — see Counterexample 6.
- **CI vs local tool-version drift** — see Counterexample 4.
- **Stub-path explosion (1.3 RunE)** — see Counterexample 7.
- **Package `main` collision (`main/magefile.go` vs `main/cmd/rak/main.go`)** — verified via Context7 mage docs that magefile.go uses `package main` + `//go:build mage` build tag, isolating it from the normal build. `main/cmd/rak/main.go` is in a different directory (`main/cmd/rak/`) and so a different package even though the package name is also `main`. **No counterexample.** Magefile and binary do not collide; `go build ./...` compiles only `cmd/rak`; `mage`'s own runner compiles only files under the `mage` build tag inside `main/`.
- **Stash `go.mod` indirect-line fib** — see Counterexample 3.
- **Stash file lifecycle** — `/tmp/rak-stash/test.txt` (342KB) and `/tmp/rak-stash/PLAN.md` are explicitly NOT copied (PLAN.md Notes line 131). `/tmp/rak-stash/` deleted in Drop 1 closeout. Acceptance line 44 confirms `/tmp/rak-stash/` still present at end of 1.1. **No counterexample.**
- **fang.Execute signature** — Context7 confirms `fang.Execute(ctx, root, fang.WithNotifySignal(os.Interrupt, ...))` is canonical. Plan's exact call-shape (1.3 acceptance line 68) matches. **No counterexample.**
- **`fang.WithNotifySignal` signal types** — Plan asks for `os.Interrupt, syscall.SIGTERM`. Context7 example uses `os.Interrupt, os.Kill` and `os.Interrupt`. Both `os.Interrupt` and `syscall.SIGTERM` are `os.Signal`-implementing values; the variadic accepts either. **No counterexample.**
- **`-coverpkg=./internal/...` against zero matching packages** — Plan note line 137 acknowledges this and offers a fallback to `-coverpkg=./...` with a TODO. **No counterexample on the plan**, though the builder will need to verify behavior in Go 1.26 — flagged in plan as builder's call.
