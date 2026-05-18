# DROP_E — LOCKFILES_AND_POLISH

**State:** done
**Tier:** B
**Blocked by:** —
**Paths (expected):** NEW internal/lockfiles/lockfiles.go + lockfiles_test.go, internal/lister/lister.go (lockfile filter integration + non-regular-file friendly error), internal/lister/lister_test.go, cmd/rak/root.go (--include-lockfiles flag + --no-gitignore help text), README.md, docs/tapes/default-toon.tape (re-record), docs/default-toon.gif (regenerated), NEW .goreleaser.yml, NEW .github/workflows/release.yml
**Packages (expected):** NEW internal/lockfiles, internal/lister, cmd/rak
**PLAN.md ref:** — (top-level PLAN.md removed at v0.1.0 ship; see memory `session_handoff_2026_05_16_v020_planning.md`)
**Workflow:** main/drops/WORKFLOW.md
**Started:** 2026-05-17
**Closed:** 2026-05-18

## Scope

Bundle of "smaller items" remaining in v0.2.0 after the YAGNI sweep. Five units, mixed tiers internally (Tier B drop = falsification-only build-QA per unit). All units kept per dev decisions 2026-05-16 → 2026-05-17:

1. **Lockfile exclusion by default + `--include-lockfiles` opt-in** (NEW feature). The largest item; serves rak's brag-tool positioning. Lockfiles like `go.sum`, `package-lock.json`, etc. are machine-generated dep manifests, not human-authored code. By default hide them so `rak` answers "how much code did your team write." `--include-lockfiles` flag preserves the v0.1.x "count everything tracked" behavior.
2. **Friendly error for non-regular files** (v0.1.4 incident follow-up). `rak /dev/null` / sockets / pipes today fails with `fork/exec: not a directory`. Replace with explicit `not a regular file or directory: <path>`.
3. **`--no-gitignore` help-text nudge**. One-line cobra flag description tweak — currently says "inside a git repo: hard error" but doesn't document that single-file invocations are silent no-ops.
4. **GoReleaser binaries + GitHub Actions release workflow** (macOS + Linux only; Windows deferred to v0.3.0). `.goreleaser.yml` config + workflow that fires on tag push to create `gh release` with platform binaries.
5. **Hero gif regen** — re-record `docs/tapes/default-toon.tape` to show TOON output first, then `--human` output, so first-time README visitors see both formats immediately.

**Out of scope per YAGNI sweep 2026-05-16:**
- ~~`--no-lockfiles` flag~~ — replaced by default-exclude polarity (unit E.1 above).
- ~~`--include-untracked`~~ — defer to v0.3.0.
- ~~Coverage badge~~ — cosmetic, cut.
- ~~Cross-platform CI matrix (Windows)~~ — defer to v0.3.0.
- ~~Token counting (`--tokens`)~~ — defer to v0.3.0.
- ~~Parallel walk (`--workers`)~~ — defer to v0.3.0.
- ~~`--follow` symlink traversal~~ — defer to v0.3.0; `find -L | rak --files-from -` covers via Stream D composition.

## Planner

All units below filled by orch (Tier B convention — orch writes Planner section inline; no `go-planning-agent` spawn). Tier B build-QA = falsification-only per unit; the test suite is the proof.

---

### Unit E.1 — Lockfile exclusion by default + `--include-lockfiles`

**State:** done
**Paths:** NEW `internal/lockfiles/lockfiles.go`, NEW `internal/lockfiles/lockfiles_test.go`, `internal/lister/lister.go` (filter integration), `internal/lister/lister_test.go` (lockfile-filter test), `cmd/rak/root.go` (flag wiring), `cmd/rak/root_test.go` (flag-parse + integration), `README.md` (Default behavior + Flags table + v0.2.0 release note)
**Packages:** NEW `internal/lockfiles`, `internal/lister`, `cmd/rak`
**Blocked by:** —

**Scope:**

Create a new tiny package `internal/lockfiles/` with a denylist of well-known lockfile basenames + a filter helper. Integrate the filter into `internal/lister`'s post-listing pipeline. Add `--include-lockfiles` flag in `cmd/rak/root.go` to opt out of the filter. Update README to document the default + the v0.2.0 behavior change.

**Design decisions (do not relitigate):**

1. **Denylist content** (case-insensitive basename match):
   - `go.sum`
   - `package-lock.json`
   - `yarn.lock`
   - `pnpm-lock.yaml`
   - `Cargo.lock`
   - `Gemfile.lock`
   - `Pipfile.lock`
   - `poetry.lock`
   - `composer.lock`
   - `mix.lock`
2. **Package API**:
   ```go
   package lockfiles

   // IsLockfile reports whether the basename of path matches a well-known lockfile name.
   // Match is case-insensitive on the basename only; directory components are ignored.
   func IsLockfile(path string) bool
   ```
   Implementation: `strings.ToLower(filepath.Base(path))` lookup in a package-level `map[string]struct{}` initialized from the denylist constant.
3. **Filter integration**: in `internal/lister.Detect` or the equivalent post-listing layer, add a filter step that drops files where `lockfiles.IsLockfile(path)` returns true UNLESS the caller passes an opt-in flag (plumbed via `listerOpts` or equivalent options struct). The exact integration point should be the spot where binary-file filtering already happens — same layer, same pattern.
4. **Flag wiring**: `--include-lockfiles` bool, default false. Plumbed through `rootFlags → listerOpts (or runDirectoryOpts) → filter logic`. Cobra flag description: `"include lockfiles (go.sum, package-lock.json, etc.) in counts (default excludes them so you see code your team wrote, not machine-generated dep manifests)"`.
5. **Cobra `Example:` entry** appended to existing examples:
   ```
     # Include lockfiles in the count (default excludes them)
     rak --include-lockfiles .
   ```
6. **README updates**:
   - `## Default behavior` section: update the "Lockfiles counted" bullet to "Lockfiles excluded by default. `go.sum`, `package-lock.json`, etc. are skipped so counts reflect code your team wrote. Pass `--include-lockfiles` to count them."
   - `## Flags` table: add `--include-lockfiles` row.
   - Add a `## v0.2.0 changes` (or similar) note at the bottom of README OR a clearly-marked line in `## Default behavior` calling out the silent behavior change vs v0.1.x: "v0.2.0 changes the default to exclude lockfiles; v0.1.x counted everything tracked."

**Acceptance:**
- `mage test ./internal/lockfiles/...` passes with `-race`.
- `mage test ./internal/lister/...` passes with `-race`; new test confirms lockfile-named files are filtered out by default and present when `--include-lockfiles` is set.
- `mage test ./cmd/rak/...` passes; new integration test verifies `rak .` excludes lockfiles, `rak --include-lockfiles .` includes them.
- `TestIsLockfile` table covers all 10 denylist entries (lowercase, mixed-case, uppercase) and at least 3 non-lockfile examples (`main.go`, `README.md`, `lockfiles.txt` — name contains "lock" but isn't in denylist).
- `IsLockfile("/path/to/sub/Cargo.lock")` returns true (basename match, ignores directory).
- `rak --help` shows `--include-lockfiles` flag with the documented description.
- `rak --help` shows the new cobra Example entry.
- README updated: `## Default behavior` correctly states lockfiles excluded by default; `## Flags` table includes `--include-lockfiles`; v0.2.0 behavior change explicitly documented.
- `mage build` passes.

---

### Unit E.2 — Friendly error for non-regular files

**State:** done
**Paths:** `internal/lister/lister.go`, `internal/lister/lister_test.go`
**Packages:** `internal/lister`
**Blocked by:** —

**Scope:**

Today `rak /dev/null` or `rak <socket>` fails with the obscure error `Lister: detect: fork/exec git: not a directory`. The v0.1.4 single-file fix handled regular files + symlinks but doesn't gracefully handle non-regular files (devices, sockets, named pipes). Add an explicit early check in `lister.Detect` (or the equivalent entry point) that returns a friendly error: `not a regular file or directory: <path>`.

**Design decisions:**

1. **Where to check**: in `lister.Detect`, after the `os.Stat` (or `filepath.EvalSymlinks` + `os.Stat`) call that v0.1.4 already does. Currently the check is `info.Mode().IsRegular()` for the single-file path; need a parallel check that handles the "neither regular nor directory" case BEFORE the git-mode probe runs.
2. **Error shape**: return a wrapped error like `fmt.Errorf("lister: detect: %s: not a regular file or directory", path)` — no underlying error wrap since this is rak's own classification, not a syscall failure.
3. **Sentinel** (optional but cheap): `var ErrNotRegularFileOrDirectory = errors.New("not a regular file or directory")` so callers can `errors.Is` if desired. Add to the lister package.
4. **Test cases** (use `t.TempDir()` for setup):
   - `os.OpenFile(path, os.O_CREATE, 0644)` (regular file) → no error from this guard
   - `os.MkdirAll(path, 0755)` (directory) → no error from this guard
   - `os.Symlink(target, link)` to regular file → no error (resolved via EvalSymlinks, then stat)
   - For non-regular cases, use `/dev/null` (always present on macOS + Linux) as the test input — `os.Stat` on `/dev/null` returns a character-device mode. Test asserts the friendly error fires.
   - Optional: create a named pipe via `syscall.Mkfifo` in `t.TempDir()` for the pipe case (Unix only; guard the test with build tag if needed).

**Acceptance:**
- `mage test ./internal/lister/...` passes with `-race`.
- `TestDetect_NotRegularFile_FriendlyError` (new): `lister.Detect("/dev/null", ...)` returns an error whose message contains `"not a regular file or directory"` and does NOT contain `"fork/exec"`.
- Existing v0.1.4 tests (`TestDetect_SingleFile`, `TestDetect_BrokenSymlink`, symlinked-walk-root tests) continue to pass.
- `rak /dev/null` (smoke via `mage run -- /dev/null` or equivalent) emits the friendly error.
- `mage build` passes.

---

### Unit E.3 — `--no-gitignore` help-text nudge

**State:** done
**Paths:** `cmd/rak/root.go`
**Packages:** `cmd/rak`
**Blocked by:** E.1 (E.1 also edits `cmd/rak/root.go`; bundle the help-text tweak in same touch to avoid serial root.go conflicts)

**Scope:**

The current `--no-gitignore` flag description says: `"**inside a git repo: hard error** (rak uses git-tracked enumeration; this flag is meaningless). Outside a git repo: disable .gitignore filtering."`

This is accurate for directory walks but doesn't cover the v0.1.4 single-file path. Add a one-line clarification.

**Design decision:**

Updated description: `"inside a git repo: hard error (rak uses git-tracked enumeration; this flag is meaningless). Outside a git repo: disable .gitignore filtering. Single-file invocations: silent no-op."`

That's it. Pure description tweak; no behavior change.

**Acceptance:**
- `mage build` passes.
- `rak --help` shows the updated `--no-gitignore` description including "Single-file invocations: silent no-op".

---

### Unit E.4 — GoReleaser binaries + release workflow

**State:** done
**Paths:** NEW `.goreleaser.yml`, NEW `.github/workflows/release.yml`, `README.md` (binary-download section), `cmd/rak/main.go` (const → var for ldflags injection)
**Packages:** — (CI config only)
**Blocked by:** —

**Scope:**

Add GoReleaser config + GitHub Actions workflow that fires on tag push (`v*.*.*`) to create a GitHub release with platform binaries.

**Design decisions:**

1. **`.goreleaser.yml`**: minimal config for `cmd/rak` binary. Platforms: `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`. NO Windows in v0.2.0 (deferred per YAGNI sweep). Archive format: `tar.gz` for Linux, `tar.gz` for macOS. Include `LICENSE` + `README.md` in archives. Version injected via `-ldflags "-X main.version={{.Version}}"` matching the existing `cmd/rak/main.go` version constant pattern (verify against current `main.go`).
2. **`.github/workflows/release.yml`**: standard GoReleaser action. Triggered on `push` event with `tags: ['v*.*.*']`. Uses `goreleaser/goreleaser-action@v6` (or current pinned major). Uses `GITHUB_TOKEN` from secrets (no PAT required for standard releases). Permissions: `contents: write` for release creation.
3. **README install section update**: add a "Download a binary" section above the existing `go install` instructions linking to the Releases page. Don't remove `go install` instructions — both are valid install paths.

**Acceptance:**
- `.goreleaser.yml` exists and is syntactically valid (builder verifies via `goreleaser check` or equivalent dry-run; if `goreleaser` not installed locally, builder notes in worklog that CI will validate on next tag).
- `.github/workflows/release.yml` exists and validates as YAML.
- README has a "Download a binary" section above `go install`.
- No `mage` targets are affected (this is CI/release config only).
- `mage ci` (drop-end gate) still passes.
- Dev signoff: dev verifies the workflow YAML structure is correct before merging (CI runs on tag push only, not in standard PR validation).

---

### Unit E.5 — Hero gif regen (TOON then --human)

**State:** done
**Paths:** `docs/tapes/default-toon.tape`, `docs/default-toon.gif`
**Packages:** — (docs only)
**Blocked by:** —

**Scope:**

The hero gif in README (currently `docs/default-toon.gif`) shows only TOON output. Per dev's "first-time visitor should see both formats immediately" rule, re-record the tape to show TOON first then `--human` second.

**Design decisions:**

1. **Tape structure**: extend `docs/tapes/default-toon.tape` to:
   - Run `rak internal/counting` (or equivalent small dir) → captures TOON output.
   - Pause briefly (`Sleep 2s` or similar).
   - Run `rak --human internal/counting` → captures human output.
   - Final frame holds long enough to read.
2. **Builder runs `vhs main/docs/tapes/default-toon.tape`** to regenerate the `.gif`. Commits the new gif file.
3. **README**: the existing embed reference (`![rak default TOON output](docs/default-toon.gif)`) stays — same path. Update the caption text in README to acknowledge both formats: `![rak default TOON output then --human](docs/default-toon.gif)` or similar.

**Acceptance:**
- `docs/tapes/default-toon.tape` updated to show TOON then `--human`.
- `docs/default-toon.gif` regenerated (file modified date later than the tape edit).
- README hero gif caption acknowledges both formats are shown.
- `mage build` + `mage test` unaffected (no Go changes).
- Dev signoff: dev verifies the gif visually renders well (TOON readable, `--human` readable, transition not jarring).

---

## Notes

**Tier B rationale**: lockfile feature (E.1) has user-facing behavior change worth a falsification pass. E.2 has v0.1.4-style regression risk. E.4 is CI infra worth review. E.3 + E.5 are orch-direct caliber but bundled in the same drop for coordination simplicity. Build-QA = falsification-only per Tier B; test suite is the proof for E.1/E.2; dev signoff is the proof for E.3/E.4/E.5.

**Cross-stream coordination**: E.1 + E.3 both touch `cmd/rak/root.go`. E.3 is `Blocked by: E.1` to serialize. All other units have disjoint Paths and run parallel-safe.

**Build-time root.go ordering**: D (`--files-from`) lands BEFORE E in `cmd/rak/root.go`. The flag-registration block ordering in `root.go`: existing v0.1.x flags → `--files-from` (D) → `--include-lockfiles` (E). `PersistentPreRunE` checks chained in this order.

**v0.2.0 release note for lockfiles**: the lockfile default-exclude is a silent behavior change vs v0.1.x. Users who had baseline `rak .` numbers from v0.1.x will see lower numbers in v0.2.0 (lockfile lines no longer counted). Prominent release-note line required: "v0.2.0 changes the default to exclude lockfiles. Pass `--include-lockfiles` to restore v0.1.x behavior."

**Windows policy for GoReleaser**: macOS + Linux only in v0.2.0 per YAGNI sweep 2026-05-16. Windows binaries can be added in v0.3.0 alongside the Windows CI matrix work.

**Lockfile denylist completeness**: 10 entries cover the major ecosystems (Go, Node, Ruby, Python, Rust, PHP, Elixir). Bun (`bun.lockb`), deno (`deno.lock`), and other newer ecosystems are intentionally NOT included — add via community PR if needed. The denylist is a small constant; adding entries later is trivial.

**Hero gif vs README narrative**: the hero gif is now multi-format. The existing per-format gifs (`human.gif`, `json.gif`, `sort-files.gif`) remain unchanged — they show specific features in isolation. The hero gif is the "show both formats" overview; the per-format gifs are deep-dives.
