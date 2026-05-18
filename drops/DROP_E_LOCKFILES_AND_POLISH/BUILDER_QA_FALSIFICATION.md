# DROP_E — Builder QA Falsification

Append a `## Unit E.M — Round K` section per QA attempt. Tier B falsification-only build-QA per `main/drops/WORKFLOW.md` § "Cascade Tiering (A / B / C)".

## Unit E.1 — Round 1

- **Reviewer:** go-qa-falsification-agent
- **Started:** 2026-05-17
- **Verdict:** PASS (no CONFIRMED counterexample; one accepted finding for dev consideration)
- **Files reviewed:**
  - `internal/lockfiles/lockfiles.go` (denylist + `IsLockfile`)
  - `internal/lockfiles/lockfiles_test.go` (table-driven test)
  - `cmd/rak/root.go` (flag wiring, filter integration in `walkAndCount`)
  - `cmd/rak/root_test.go` (`TestRootCmd_PathArg_LockfileFilter` MapFS test, `runTreeFS` plumbing)
  - `cmd/rak/integration_test.go` (`TestLockfileFilter_ExcludedByDefault`, `TestLockfileFilter_IncludeWhenFlagSet`)
  - `README.md` (Default behavior, Flags table, v0.2.0 behavior changes section)
- **Mage targets run:** `mage ci` (pass — lint 0 issues, all tests pass with -race, coverage 87.9% above 70.0% floor)

### Attack 1 — Denylist completeness (PLAN spec vs implementation)

REFUTED. All 10 PLAN.md denylist entries present in `denied` map (`internal/lockfiles/lockfiles.go:17-28`) — `go.sum`, `package-lock.json`, `yarn.lock`, `pnpm-lock.yaml`, `cargo.lock`, `gemfile.lock`, `pipfile.lock`, `poetry.lock`, `composer.lock`, `mix.lock`. Stored lowercase to support case-insensitive lookup.

### Attack 2 — Case sensitivity (`IsLockfile("Go.Sum")` should match)

REFUTED. Implementation lowercases input basename before lookup (`lockfiles.go:39`: `strings.ToLower(filepath.Base(path))`). Test covers all 10 entries in lowercase + UPPERCASE forms; 4 entries (Go.Sum / Package-Lock.Json / Yarn.Lock / Pnpm-Lock.Yaml) also covered in mixed-case (`lockfiles_test.go:33-48`). Implementation is uniform per-character, so mixed-case coverage of subset is sufficient — a regression in case-handling for one entry would fail the others as well.

### Attack 3 — Path prefix handling (`IsLockfile("/path/to/Cargo.lock")` returns true)

REFUTED. Test explicitly covers three directory-prefixed variants (`lockfiles_test.go:51-53`): `/path/to/sub/Cargo.lock`, `some/nested/dir/go.sum`, `a/b/c/package-lock.json` — all assert `true`. Implementation via `filepath.Base` discards directory components.

### Attack 4 — Non-lockfile guards (`lockfiles.txt`, `mylock.go`, bare `lock`)

REFUTED with one minor coverage gap. Negative cases tested: `main.go`, `README.md`, `lockfiles.txt`, `go.mod`, `package.json`, `.gitignore`, `Makefile` (`lockfiles_test.go:55-62`). PLAN spec called out `mylock.go` and bare `lock` (no extension) which are not in the test table; however, the implementation is a pure map lookup with no substring or prefix matching, so the absence of these specific names cannot produce false positives — only adding such a name to `denied` could. The 7 negative cases tested are sufficient evidence that substring "lock" does not cause spurious matches (`lockfiles.txt` covers this exact concern).

### Attack 5 — Filter integration (pre-binary vs post-binary; gating by `includeLockfiles`)

REFUTED. Filter sits in `walkAndCount` at `cmd/rak/root.go:471-473`, AFTER binary detection (lines 457-466) and BEFORE language detection (line 478). Gating condition: `!includeLockfiles && lockfiles.IsLockfile(f.RelPath)` — correct polarity (default false → filter active; flag true → filter bypassed). Both paths exercised by tests:
- Default exclude: `TestRootCmd_PathArg_LockfileFilter/default_excludes_lockfile` (`root_test.go:514-521`) + `TestLockfileFilter_ExcludedByDefault` (`integration_test.go:455-497`).
- Include with flag: `TestRootCmd_PathArg_LockfileFilter/include_lockfiles_flag_counts_both` + `TestLockfileFilter_IncludeWhenFlagSet` (`integration_test.go:502-544`).

Filter order vs binary detection: lockfile filter is post-binary, which means a NUL-byte-containing lockfile is skipped first by the binary filter. Per PLAN design decision 3 ("same layer, same pattern" as binary filtering), filter order between the two is implementation choice; either order produces the correct count for any non-pathological input.

### Attack 6 — `--include-lockfiles` flag wiring through full plumbing chain

REFUTED. Flag declared at `cmd/rak/root.go:42` (`includeLockfiles bool`), registered at lines 211-216 (cobra `BoolVar` with documented description matching PLAN spec exactly), plumbed via `runDirectoryOpts.includeLockfiles` (line 340) through both the `--files-from` path (line 278) AND the positional-path argument path (line 299). End-to-end integration confirmed for both modes.

### Attack 7 — README accuracy

REFUTED. All three required README updates present:
- `## Default behavior` section (line 117): "Lockfiles excluded by default. `go.sum`, `package-lock.json`, and other machine-generated dep manifests are skipped so counts reflect code your team wrote. Pass `--include-lockfiles` to count them."
- `## Flags` table (line 137): `| --include-lockfiles | off | include lockfiles (go.sum, package-lock.json, etc.) in counts |`.
- `## v0.2.0 behavior changes` section (lines 151-153): explicitly calls out the silent behavior change vs v0.1.x with restore instructions.
- Bonus: Roadmap section (line 163) also mentions the feature consistently.

### Attack 8 — Cobra Example entry

REFUTED. Example entry present at `cmd/rak/root.go:105-106` matching PLAN spec exactly:
```
  # Include lockfiles in the count (default excludes them)
  rak --include-lockfiles .
```
Live verification via `mage run -- --help` confirms the entry renders correctly.

### Attack 9 — `mage ci` cleanliness

REFUTED. `mage ci` runs gofumpt check (0 issues), `go vet ./...` (clean), `golangci-lint run` (0 issues), `go test -race ./...` (all 9 packages pass), coverage 87.9% (above 70.0% floor). No staleness, no race conditions detected.

### Finding F1 — UX: `rak <path-to-single-lockfile>` silently produces empty totals (non-counterexample)

When a user explicitly passes a lockfile as the positional path arg (e.g. `rak go.sum`), `lister.Detect` returns a `singleFileLister` that yields the file, but `walkAndCount` then filters it out via `IsLockfile`. The result is empty totals with no diagnostic — surprising given v0.1.4 explicitly added single-file invocation support ("rak hello.go counts that file").

This is design-conformant per PLAN.md (the filter is uniform across all listing modes), so NOT a CONFIRMED counterexample. But it conflicts with the v0.1.4 single-file UX contract — a user who explicitly named a lockfile probably wants it counted. Two design options for future dev consideration:
1. Status quo: lockfile filter applies uniformly; document this edge case.
2. Bypass filter when source is `singleFileLister` (mode = "user-explicit single file"). Mirrors how `--include-lockfiles` would, automatically.

Flagging here as a finding, not a counterexample — orchestrator + dev can route as polish if desired (likely a one-line check in `runRoot`'s single-file branch). Tier B drop, dev signoff applies.

### Finding F2 — Test coverage: `TestRootCmd_HelpContainsExamples` is stale (non-counterexample)

`cmd/rak/root_test.go:1310-1347` asserts 8 example commands. The Example field now contains 10 (Drop D added `--files-from` examples, Drop E added `--include-lockfiles`). The test passes because `strings.Contains` only asserts the listed 8 are present; it does NOT assert the new 3 examples render. So a future regression that drops the lockfile or files-from examples would not be caught by this test.

Not a counterexample (the help output IS correct today, verified via `mage run -- --help`). Coverage gap only — recommend extending `wantCmds` to include the new entries when convenient.

### Verdict

**PASS.** All 9 attack angles from the spawn prompt are either REFUTED or attacked without producing a CONFIRMED counterexample. Two findings (F1, F2) routed back to orchestrator as design/test-coverage considerations, not gate-blockers. `mage ci` green.

### Hylla Feedback

None — Hylla was used minimally for this review (one `hylla_search_keyword` query confirming yaml/lock detection symbols). All evidence came from direct `Read` of source files (small Go files, pre-known integration points from spawn prompt) and `git log` / `mage ci` / `mage run -- --help` for behavioral verification. No Hylla miss forced a fallback.

## Unit E.3 — Round 1

- **Reviewer:** go-qa-falsification-agent
- **Started:** 2026-05-17
- **Verdict:** PASS (no CONFIRMED counterexample; one out-of-scope observation routed to orchestrator)
- **Commit under review:** `c1da170 docs(cmd): clarify --no-gitignore help text for single-file path`
- **Files reviewed:**
  - `cmd/rak/root.go` (`--no-gitignore` flag description, line 161)
  - `README.md` (`--no-gitignore` row in Flags table, line 134) — parallel-update audit
  - `cmd/rak/root_test.go` (sanity scan for description-string assertions)
  - `drops/DROP_E_LOCKFILES_AND_POLISH/PLAN.md` E.3 spec (lines 128-149)
- **Mage targets run:** `mage run -- --help` (verify rendered output), `mage test` (baseline regression)

### Attack 1 — Literal-text presence in `root.go`

REFUTED. `cmd/rak/root.go:161` contains the exact spec string: `"inside a git repo: hard error (rak uses git-tracked enumeration; this flag is meaningless). Outside a git repo: disable .gitignore filtering. Single-file invocations: silent no-op."`. Literal substring `"Single-file invocations: silent no-op."` present byte-for-byte.

### Attack 2 — `rak --help` actually surfaces it

REFUTED. `mage run -- --help` output for the `--no-gitignore` row reads: `Inside a git repo: hard error (rak uses git-tracked enumeration; this flag is meaningless). Outside a git repo: disable .gitignore filtering. Single-file invocations: silent no-op.`. Fang title-cases the first letter ("Inside" vs source "inside") — cosmetic-only rendering artifact, not a content drift; the spec'd phrase appears verbatim.

### Attack 3 — Behavior regression introduced by E.3

REFUTED for E.3-attributable scope. E.3's committed diff (`git show c1da170 --stat`) is exactly `cmd/rak/root.go | 2 +-` (pure description string) plus markdown housekeeping (`PLAN.md` state flip + `BUILDER_WORKLOG.md` entry). No code-path or behavior surface touched. Cross-check: rebuilt against E.1 commit `62e6a65` (immediately before E.3) — full `mage test` is green. No regression introduced by E.3's commit.

### Attack 4 — README `--no-gitignore` parallel-update obligation

REFUTED. README line 134 still reads: `**inside a git repo: hard error** (rak uses git-tracked enumeration; this flag is meaningless). Outside a git repo: disable .gitignore filtering.` (no single-file clause). PLAN.md E.3 § "Design decision" (lines 137-145) explicitly scopes the change to the cobra description only ("That's it. Pure description tweak; no behavior change."). E.3's declared `Paths:` is `cmd/rak/root.go` only — README is NOT in scope. Spec held; no counterexample.

### Attack 5 — `PersistentPreRunE` `--no-gitignore` interaction surface

EXHAUSTED, no counterexample found. The combo-error message at `cmd/rak/root.go:115-117` references the flag by name (`"--no-gitignore is meaningless with --files-from: ..."`) — wording unaffected by description-string content. No new constraint or error path introduced by E.3.

### Attack 6 — Go-quality attacks (error swallowing / goroutines / interface misuse / raw-go invocation)

N/A. Description-string-only diff has no concurrency, error-handling, or interface surface. No raw-`go` invocations. No `mage install` calls.

### Out-of-scope observation routed to orchestrator (NOT attributed to E.3)

`mage test` against current working tree fails ONE test:

```
--- FAIL: TestRootCmd_Version (0.00s)
    root_test.go:1194: --version output does not contain "v0.1.4"; got:
        rak version v0.2.0-dev
```

**Attribution chain (E.3 is NOT at fault):**

- E.3's committed diff (`c1da170`) does NOT touch `cmd/rak/main.go` or `cmd/rak/root_test.go`.
- At commit `62e6a65` (E.1, immediately before E.3): `mage test` passes; `main.go` defines `const version = "v0.1.4"`.
- Current working tree has uncommitted change to `cmd/rak/main.go` bumping `const version = "v0.1.4"` → `var version = "v0.2.0-dev"` (with ldflags-injection plumbing for GoReleaser). This is E.4 (GoReleaser unit) preparation — out of E.3's declared `Paths:`.
- `cmd/rak/root_test.go:1193` still asserts the literal string `"v0.1.4"`.

**Routing recommendation:** flag to orchestrator. Two clean fixes:
1. Stash the `main.go` bump out of the working tree before E.3 verification windows (preferred for clean per-unit attribution).
2. When E.4 lands, update `TestRootCmd_Version` to read from the live `version` variable rather than a hardcoded literal — eliminates future version-bump test-drift.

Not E.3's bug; not E.3's responsibility to fix. Flagging because the spawn prompt's attack #3 ("no behavior changes (mage test still passes)") would naively fire FAIL on this — and the failure has nothing to do with E.3's diff.

### Verdict

**PASS.** All four spawn-prompt attack vectors land as no-counterexample-found when E.3's commit is evaluated in isolation. The README/cobra-help consistency check confirms PLAN spec was honored (cobra-only tweak, README intentionally untouched). The lone `mage test` failure is out-of-scope working-tree contamination from E.4 (GoReleaser) preparation — routed to orchestrator, not attributed to E.3.

### Hylla Feedback

N/A — Unit E.3 touched only a cobra flag description string literal in `cmd/rak/root.go`; the diff is non-semantic from Hylla's perspective. Hylla was not queried; `Read` + `git show` + `git diff` covered the verification surface directly. No miss.
