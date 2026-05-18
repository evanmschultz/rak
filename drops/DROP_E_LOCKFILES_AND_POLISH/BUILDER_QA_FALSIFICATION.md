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

## Unit E.2 — Round 1

- **Reviewer:** go-qa-falsification-agent
- **Started:** 2026-05-17
- **Verdict:** PASS for E.2 (no CONFIRMED counterexample against the unit). One drop-level finding (F3) re-flagged — `mage ci` red on `TestRootCmd_Version` due to uncommitted `cmd/rak/main.go` v0.2.0-dev bump (already documented under Unit E.3; orthogonal to E.2 but blocks Phase 6).
- **Commit under review:** `31dfa0e fix(lister): friendly error for non-regular non-directory paths`
- **Files reviewed:**
  - `internal/lister/lister.go` (sentinel decl line 38-43 + guard placement in `Detect` lines 84-86)
  - `internal/lister/lister_test.go` (`TestDetect_NotRegularFile_FriendlyError` lines 540-563 + pre-existing v0.1.4 regression tests)
  - `git diff HEAD~3 -- internal/lister/` to isolate E.2's actual delta from E.1/E.3/E.5
- **Mage targets run:** `mage test` (lister package PASS cached; full repo FAIL on unrelated `TestRootCmd_Version` per Finding F3).

### Attack 1 — Sentinel correctness (`ErrFoo` shape, package scope, doc comment)

REFUTED. `ErrNotRegularFileOrDirectory` is declared at package scope (`lister.go:43`), capital `E` prefix per Go convention, with a four-line doc comment that names the identifier first (`// ErrNotRegularFileOrDirectory is returned by Detect when the resolved path is …`). Doc comment matches the `ErrNoGitignoreInRepo` sibling shape (lines 30-36) — consistent with package style. Declared via `errors.New("not a regular file or directory")`, lowercase message, no trailing punctuation per Go error-string convention. `go vet` + `golangci-lint` clean on the lister package.

### Attack 2 — Guard placement (post-EvalSymlinks/Stat, pre-git-probe; correct mode predicate)

REFUTED. Order of operations in `Detect` (lister.go:57-86):

1. `filepath.Abs` (line 58).
2. `filepath.EvalSymlinks` (line 66) — resolves symlinks before mode inspection.
3. `os.Stat(absRoot)` (line 75) — single `Stat` call, `statErr` reused.
4. Regular-file fast-path (lines 76-78): returns `SingleFileLister`.
5. **Non-regular non-directory guard (lines 84-86)**: `statErr == nil && !info.Mode().IsDir()` → returns wrapped `ErrNotRegularFileOrDirectory`.
6. Git probe (`exec.LookPath` line 90, `cmd.Output` line 97).

Guard fires AFTER both `EvalSymlinks` and `Stat` resolve, and BEFORE any git invocation. Mode predicate is correct: by step 5 a regular file has already returned at step 4, so `IsRegular() == false` is implicit; the explicit `!IsDir()` excludes directories, leaving devices/sockets/pipes/irregular to fall through. PLAN-spec wording was `IsRegular() == false AND IsDir() == false`; the implementation collapses this to `!IsDir()` by relying on the step-4 short-circuit. Operationally identical given current code, but worth noting that the predicate is order-dependent — if the step-4 fast-path were removed or reordered, the guard could fire spuriously on regular files. Latent fragility, not a counterexample today.

### Attack 3 — `%w` enables `errors.Is`

REFUTED. Wrap is `fmt.Errorf("lister: detect: %s: %w", absRoot, ErrNotRegularFileOrDirectory)` (line 85). The `%w` verb is correctly applied to the sentinel (not the path), so `errors.Is(err, lister.ErrNotRegularFileOrDirectory)` walks the wrap chain and matches. Test directly verifies (`lister_test.go:554-556`):

```go
if !errors.Is(err, lister.ErrNotRegularFileOrDirectory) {
    t.Errorf("errors.Is(err, ErrNotRegularFileOrDirectory) = false; got: %v", err)
}
```

The format string interpolates `absRoot` via `%s` (safe — string, not error) and the sentinel via `%w` (one `%w` per `fmt.Errorf` per Go 1.20+ rule; satisfied).

### Attack 4 — Test coverage (`/dev/null`, message contains "not a regular file or directory", excludes "fork/exec")

REFUTED. `TestDetect_NotRegularFile_FriendlyError` (lister_test.go:540-563) implements all PLAN-spec assertions:
- Skip-guard if `/dev/null` not stat-able (line 542-544): platform-safe.
- `err != nil` (line 548-550): friendly error returned.
- `got == nil` (line 551-553): no lister leaked through.
- `errors.Is(err, lister.ErrNotRegularFileOrDirectory)` (line 554-556): sentinel match.
- `strings.Contains(msg, "not a regular file or directory")` (line 557-559): user-visible message correct.
- `!strings.Contains(msg, "fork/exec")` (line 560-562): regression guard against the v0.1.4 obscure-error case.

All six checks named in the spawn-prompt attack #4 reproduced in the test. `/dev/null` is the canonical character-device fixture per PLAN design decision 4.

### Attack 5 — Regression guard (existing v0.1.4 tests still pass)

REFUTED. `mage test` of `./internal/lister/...` returns `ok` (cached; cache key includes `lister.go` source — cache hit means the package passed under the new code). Tracing each existing test through the new guard:
- `TestDetect_SingleFile` (line 209): regular file → step-4 fast-path returns; guard never reached.
- `TestDetect_SymlinkedFile` (line 232): symlink → regular file → step-4 returns; guard never reached.
- `TestDetect_SymlinkedDir` (line 260): symlink → git dir → step-4 false (`IsRegular() == false`), step-5 false (`!IsDir() == false`); falls through to git probe; returns `GitLister`. Guard correctly inactive.
- `TestDetect_BrokenSymlink` (line 298): `EvalSymlinks` fails at step 2; guard unreachable.
- `TestDetect_InsideRepo` / `TestDetect_OutsideRepo` / `TestDetect_BareRepo` / `TestDetect_InsideGitDir` / `TestDetect_NoGitignoreInRepo_ReturnsSentinel` / `TestDetect_BareRepo_WithDisableGitignore` / `TestDetect_InsideGitDir_WithDisableGitignore`: all directory cases; `IsDir() == true` makes `!IsDir()` false; guard inactive.

No existing test path collides with the new guard.

### Attack 6 — Symlink-to-regular-file ordering (EvalSymlinks resolves before mode check)

REFUTED. Flow: `Abs` → `EvalSymlinks` → `Stat`. `EvalSymlinks` returns the resolved target, which is then `Stat`-ed. For symlink-to-regular-file, `info.Mode().IsRegular()` is true on the target, so step-4 returns `SingleFileLister` before the guard runs. `TestDetect_SymlinkedFile` (line 232) confirms end-to-end. No counterexample.

What about a symlink to a character device (e.g. `ln -s /dev/null fakelink` → `rak fakelink`)? `EvalSymlinks` resolves `fakelink` → `/dev/null`, `Stat` reports `ModeCharDevice`, step-4 short-circuits on `IsRegular() == false`, step-5 guard fires on `!IsDir() == true`, friendly error returned. Identical behavior to direct `rak /dev/null`. No spec violation.

### Attack 7 — Named pipe / socket / block-device coverage via mode-check (not test)

REFUTED. Test fixture is `/dev/null` (character device) only. But the guard predicate `!info.Mode().IsDir()` (after the regular-file fast-path) catches every non-directory mode:
- `os.ModeDevice` (block device): guard fires.
- `os.ModeCharDevice` (char device, includes `/dev/null`): guard fires — directly tested.
- `os.ModeNamedPipe` (FIFO): guard fires.
- `os.ModeSocket`: guard fires.
- `os.ModeIrregular`: guard fires.

The implementation is mode-family-agnostic: anything reaching step 5 with `!IsDir()` triggers the friendly error. So while only `/dev/null` is in the test table, the production code path for FIFO / socket / block-device is the same single branch (lister.go:84-86). Risk of a per-mode regression is structurally zero — there is no per-mode logic to break. PLAN.md flagged the optional `syscall.Mkfifo` test as "Unix only; guard with build tag if needed"; the builder skipped it because the structural argument makes it redundant. Reasonable trade.

### Attack 8 — `mage ci` clean

CONFIRMED-counterexample-against-drop-not-E.2 (see Finding F3). `mage ci` fails at `mage test` because `TestRootCmd_Version` (cmd/rak/root_test.go:1192-1195) asserts the `--version` output contains `"v0.1.4"`, but `cmd/rak/main.go` was bumped to `var version = "v0.2.0-dev"` in an uncommitted working-tree edit (`git status -s cmd/rak/main.go` → ` M cmd/rak/main.go`). E.2's own diff touches only `internal/lister/*` — the lister package itself passes cleanly. This finding was previously flagged under Unit E.3 Round 1; persists at the E.2 review window. Blocks Phase 6 drop-end gate but does NOT block per-unit attribution for E.2.

### Finding F3 (re-flag from Unit E.3 — drop-level, not E.2-attributable)

Already documented in the Unit E.3 § "Out-of-scope observation routed to orchestrator". Re-flagging here because Attack 8 above lands on the same artifact:

- Uncommitted edit: `const version = "v0.1.4"` → `var version = "v0.2.0-dev"` in `cmd/rak/main.go` (also const → var for ldflags injection).
- `cmd/rak/root_test.go:1193` still asserts the literal string `"v0.1.4"`.
- Recommended fix paths unchanged from E.3 review: (a) stash the bump until E.4 lands, OR (b) commit the bump together with `TestRootCmd_Version` updated to read from the live `version` variable / `strings.HasPrefix(got, "rak version v")` for forward-compatibility.

### Verdict

**PASS for Unit E.2.** All 8 spawn-prompt attack angles applied. Angles 1–7 REFUTED with concrete trace through the code. Angle 8 (`mage ci`) confirms the drop-level F3 contamination — orthogonal to E.2's diff, blocks Phase 6, routed to orchestrator (already known from E.3 review). One latent fragility noted under Attack 2 (guard predicate `!IsDir()` is order-dependent on the step-4 IsRegular short-circuit); non-blocking.

### Hylla Feedback

N/A — E.2 review was a surgical 2-file inspection (`lister.go` ~126 LOC + `lister_test.go` ~610 LOC). `Read` + `git diff HEAD~3` were the right primary tools; no Hylla query was attempted (integration points were named explicitly in the spawn prompt + visible in the diff). No miss to report.

## Unit E.4 — Round 1

- **Reviewer:** go-qa-falsification-agent
- **Started:** 2026-05-17
- **Verdict:** PASS (no CONFIRMED counterexample; two minor findings — F4 stale docstring, F5 README/default-version drift — routed to orchestrator for optional polish)
- **Commit under review:** `353d93f ci(release): add goreleaser config and tag-triggered workflow`
- **Files reviewed:**
  - `.goreleaser.yml` (NEW)
  - `.github/workflows/release.yml` (NEW)
  - `cmd/rak/main.go` (`const version` → `var version` for ldflags injection)
  - `cmd/rak/root_test.go` (`TestRootCmd_Version` updated to read live `version` var)
  - `README.md` (`## Download a binary` section, `--version` example output line)
- **Mage targets run:** `mage ci` (PASS — gofumpt clean, lint 0 issues, all 9 packages green with `-race`, coverage 87.9% above 70% floor)

### Attack 1 — `.goreleaser.yml` schema validity (v2 + 4 platforms + tar.gz + LICENSE/README + ldflags)

REFUTED with two deprecation observations.

Verified against `/goreleaser/goreleaser` Context7 reference and the `goreleaser-action@v6` + `version: '~> v2'` workflow pin:

- `version: 2` (line 1) — correct top-level v2 schema marker.
- `builds[0]` — `main: ./cmd/rak`, `binary: rak`, `env: [CGO_ENABLED=0]`, `goos: [linux, darwin]`, `goarch: [amd64, arm64]` — yields exactly the four required platform×arch combinations (linux/amd64, linux/arm64, darwin/amd64, darwin/arm64). NO `windows` in `goos` (matches PLAN.md "NO Windows in v0.2.0").
- `ldflags: ['-s -w -X main.version={{.Version}}']` (line 16) — `-X` flag with package path `main.version` matches the now-`var version` symbol in `cmd/rak/main.go:15`. The `-s -w` strip symbols/debug info — standard size reduction. `{{.Version}}` is the canonical GoReleaser template var that resolves to the git tag minus the leading `v` (e.g. tag `v0.2.0` → `Version = "0.2.0"`); combined with `cobra`'s default `--version` format it would print `rak version 0.2.0` (NOT `rak version v0.2.0`). See Finding F5 below for the related README drift.
- `archives[0]` — `format: tar.gz` (line 22), `name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"`, `files: [LICENSE, README.md]` (lines 24-26). LICENSE + README.md included per PLAN spec.
- `checksum.name_template: checksums.txt` (line 29) — standard.

**Deprecation observations (not counterexamples — backward-compatible aliases):**

- `format: tar.gz` (line 22): per Context7 `/goreleaser/goreleaser` "Rename Archive Format to Formats" (v2.6 deprecation), `format` is now spelled `formats: [tar.gz]`. The singular spelling is still accepted but will emit a deprecation warning on every release run. Trivial future fix: `formats: ["tar.gz"]`.
- `builds: [rak]` inside `archives[0]` (lines 20-21): per Context7 "Update Archive Builds Field" (v2.8 deprecation), `builds` is renamed to `ids`. Backward-compatible but warns.
- Both deprecations will appear as inline warnings during `goreleaser release` on the first tag push. They do NOT break the build today; they will be hard-removed in a future major. Not a counterexample — but worth a one-line fix when convenient.

### Attack 2 — `.github/workflows/release.yml` validity (triggers, permissions, action versions, fetch-depth)

REFUTED. All spawn-prompt-required attributes present and correctly shaped:

- **Trigger** (lines 3-6): `on.push.tags: ['v*.*.*']` — single-event-single-glob shape that fires only on semver tag pushes (`v0.2.0`, `v1.2.3`), NOT on `v0.2.0-dev` (no `*.*.*` match), commits, or non-tag pushes. Per PLAN spec.
- **Permissions** (lines 8-9): `contents: write` at workflow level — required for `gh release create` via GoReleaser, no `id-token` or other broader scopes leaked.
- **Checkout** (lines 16-19): `actions/checkout@v4` + `fetch-depth: 0` — full history required for GoReleaser changelog generation (`changelog.sort: asc` in `.goreleaser.yml` depends on git log). Builder worklog explicitly calls this out.
- **Go setup** (lines 21-26): `actions/setup-go@v5`, `go-version: '1.26.x'` (matches `go.mod` `go 1.26.3`), `cache: true` with `cache-dependency-path: go.sum`. Consistent with `ci.yml`'s Go pinning.
- **GoReleaser action** (lines 28-35): `goreleaser/goreleaser-action@v6`, `distribution: goreleaser` (not `goreleaser-pro`), `version: '~> v2'` (semver constraint pinning to goreleaser v2.x — matches `version: 2` in `.goreleaser.yml`), `args: release --clean`. `--clean` removes the `dist/` directory before each release (idempotent), per GoReleaser convention.
- **Env** (lines 34-35): `GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}` — automatic per-job token, no PAT or external secret required.

No misuse: no `pull_request` trigger (which would race releases), no `workflow_dispatch` manual escape hatch (acceptable for v0.2.0), no matrix builds (single-runner is correct since GoReleaser cross-compiles internally).

### Attack 3 — `const` → `var version` change enables `-X main.version=...` linker override

REFUTED. `cmd/rak/main.go:15` now reads `var version = "v0.2.0-dev"` (was `const version = "v0.1.4"`). The Go linker's `-X importpath.name=value` flag can ONLY override **package-level variables of type string** — it silently ignores `const` declarations because constants are inlined at compile time. The builder's worklog correctly identified this as the load-bearing change: without `const` → `var`, every GoReleaser-produced binary would print the hardcoded fallback regardless of the actual tag.

Live confirmation: `var version = "v0.2.0-dev"` (line 15) is package-level (not inside a func), type string (inferred from literal), and exported-equivalent for `-ldflags` purposes (linker `-X` accesses by symbol path, not Go visibility rules — `main.version` works even though `version` is unexported).

Fallback `"v0.2.0-dev"`: present for local `go install` / `go build ./cmd/rak` invocations without `-ldflags`. Matches the GoReleaser convention of `<next-version>-dev` for the pre-release dev tree.

### Attack 4 — `TestRootCmd_Version` reads live `version` variable (no v0.1.4 brittleness)

REFUTED for the assertion logic. `cmd/rak/root_test.go:1182` sets `cmd.Version = version` (the live var) and line 1193 asserts `strings.Contains(got, version)`. This is correct: future version bumps to the `var version = "..."` line in `main.go` automatically flow through to the test without edits — no hardcoded `"v0.1.4"` or `"v0.2.0-dev"` literal remains in the assertion.

The Round-1 build-QA for E.2 (lines 132-150) and E.3 (lines 244-251) of this file flagged the prior-state failure (`v0.1.4` hardcoded assert against `v0.2.0-dev` working-tree state) as F3. F3 is now RESOLVED by E.4's diff. Confirmed by `mage ci` PASS (test now green).

### Finding F4 — Stale docstring comment on `TestRootCmd_Version` (non-counterexample)

`cmd/rak/root_test.go:1169-1173` doc-comment still reads `"...prints output containing \"v0.1.4\" when invoked with --version."`. The assertion was updated (line 1193 reads from live `version`), but the function's leading godoc comment was not. Today the test passes against `v0.2.0-dev` because the assertion is parameterized; the docstring is now factually wrong about what value it expects.

Not a code counterexample — `mage ci` is green and `golangci-lint` doesn't flag doc-comment staleness. Trivial polish: rewrite the comment to read "containing the value of the package-level `version` variable" (or similar). Routed to orchestrator for optional cleanup.

### Attack 5 — README "Download a binary" section placement + go install preservation

REFUTED. `README.md:7-15` introduces the new `## Download a binary` section ABOVE the existing `## Install` section (line 17). The new section:

- Names supported platforms (macOS amd64/arm64, Linux amd64/arm64) — matches `.goreleaser.yml` build matrix.
- Links to `https://github.com/evanmschultz/rak/releases` — canonical GitHub releases page URL.
- Includes a `curl | tar | mv` example for `darwin_arm64`.

Existing `## Install` section (`go install ...`) preserved unchanged at line 17-21. Both install paths visible. The `### From source` subsection (lines 27-34) also intact. PLAN spec ("Don't remove `go install` instructions — both are valid install paths") satisfied.

### Finding F5 — README example output (`v0.2.0`) drifts from main.go default (`v0.2.0-dev`)

`README.md:39-40` shows the Quick Example output:
```
$ rak --version
rak version v0.2.0
```

But `cmd/rak/main.go:15` defines the dev-default as `var version = "v0.2.0-dev"`. A user who follows the README's `## Install` section (`go install github.com/evanmschultz/rak/cmd/rak@latest`) and runs `rak --version` against the `main` branch tip will see `rak version v0.2.0-dev` (the dev fallback, since `go install` does not invoke `-ldflags`), not the example's `rak version v0.2.0`.

Additionally, GoReleaser's `{{.Version}}` template resolves to the tag **minus** the leading `v` per its templating conventions — so a release of tag `v0.2.0` would inject `-X main.version=0.2.0`, producing `rak version 0.2.0` (no `v` prefix). The README example's `rak version v0.2.0` would only match if (a) the user happens to be running a release with a manual `v`-prefixed `-ldflags` override, OR (b) the tag template is adjusted to prepend `v` (e.g. `-X main.version=v{{.Version}}` or use `{{.Tag}}` which keeps the leading `v`).

Two options for the orchestrator:
1. **Cosmetic README fix only**: change line 40 to `rak version v0.2.0-dev` to match the local-build reality, and add a parenthetical noting "(release binaries show the tagged version without the `-dev` suffix)".
2. **Tag-template fix in `.goreleaser.yml`**: switch ldflags to `-X main.version={{.Tag}}` (which preserves the `v` prefix), then `rak version v0.2.0` actually matches what a released binary prints. Most idiomatic; matches the `v`-prefixed example users will copy.

Not a CONFIRMED counterexample against E.4's stated acceptance criteria (PLAN.md does NOT specify the exact format string for the `--version` output; the criteria are limited to "version injected via ldflags" + "syntactically valid `.goreleaser.yml`"). Routed as a polish finding; dev-signoff applies per Tier B drop convention.

### Attack 6 — `mage ci` cleanliness (drop-end gate)

REFUTED. Just ran `mage ci` from `main/`:

- `gofumpt -l .` → empty output (no formatting issues).
- `go vet ./...` → clean.
- `golangci-lint run` → 0 issues.
- `go test -race ./...` → all 9 packages PASS.
- Coverage: 87.9% on `./internal/...` (well above the 70% floor).

The previously red `TestRootCmd_Version` (F3 from Rounds for E.2/E.3) is now green — E.4's test fix (lines 1190-1194 of `root_test.go`) plumbs the live `version` variable through both the cobra command and the assertion. Confirmed via the coverage report showing all `cmd/rak` tests participated and the `total: 87.9%` rollup line.

### Attack 7 — `mage install` invocation by an agent (Go-quality discipline)

EXHAUSTED, no counterexample found. Builder worklog (`BUILDER_WORKLOG.md` E.4 section, lines 91-109) lists files touched + describes the const→var rationale + admits the path expansion to `cmd/rak/main.go` per scope-expansion protocol. Worklog explicitly states "`mage build` not run — CI-config-only unit per PLAN.md"; no `mage install` or raw `go build`/`go test`/`go vet` invocations recorded. Consistent with `main/CLAUDE.md` § "Build Verification" rule 3 ("NEVER run `mage install` from an agent").

### Attack 8 — Cross-stream race / concurrent worktree mutation (per CLAUDE.md concurrency rules)

N/A. CI config + 1-token Go change + README prose. No goroutines, no shared state, no channels, no `init()` side effects. Static config files; no runtime concurrency surface introduced.

### Attack 9 — Workflow trigger over-firing (release on PR, branch push, dev tags)

REFUTED. `on.push.tags: ['v*.*.*']` glob requires three dot-separated components — matches `v0.2.0`, `v1.10.3`, `v2.0.0-rc.1` (the trailing `-rc.1` is matched by the last `*`), but does NOT match:
- Non-tag branch push (different event scope: `push.branches`, not `push.tags`).
- Pull request events (not in `on:` list at all).
- Dev tags without the `v` prefix or without `.` separators (e.g. tag `nightly` → no match).
- Tags with fewer than 3 dot components (e.g. `v1` or `v0.2` → no match).

No `workflow_dispatch` means the release cannot be triggered manually from the Actions UI — a deliberate Tier-B-acceptable trade per dev signoff convention; can be added later if needed.

Note: tags like `v0.2.0-dev` WILL match the `v*.*.*` glob (the trailing `-dev` is matched by the third `*`), so pushing a `v0.2.0-dev` tag would inadvertently trigger a release. Not a counterexample since the dev controls tag-push manually and the v0.2.0-dev string is the in-tree dev placeholder (not a tag); but worth noting as a small footgun. Tighter regex (e.g. via a custom workflow filter) is out of scope for this unit.

### Verdict

**PASS for Unit E.4.** All 7 spawn-prompt attack angles (`.goreleaser.yml` validity / `release.yml` validity / const-to-var ldflags / test fix / README placement / README version example / `mage ci`) plus 2 additional adversarial probes (Go-quality discipline / workflow-trigger over-firing) applied. Six angles REFUTED with concrete trace through the artifact; two angles surfaced minor polish findings (F4 stale docstring on the test, F5 README `v0.2.0` example vs main.go `v0.2.0-dev` default); two angles N/A by construction. `mage ci` GREEN end-to-end.

The two GoReleaser-config deprecation notes (`format` → `formats` v2.6, `builds` → `ids` v2.8) are backward-compatible aliases that will emit warnings on first release run; route as a one-line follow-up at orchestrator's discretion.

### Hylla Feedback

N/A — Unit E.4 touched only non-Go files (`.goreleaser.yml`, `.github/workflows/release.yml`, `README.md`) and one 5-line Go diff in `cmd/rak/main.go` (const → var). Hylla is Go-only today; all evidence came from `Read`, `git show 353d93f`, `mage ci`, and Context7 (`/goreleaser/goreleaser`) for the v2-schema deprecation cross-checks. No Hylla query attempted; no miss forced a fallback.
