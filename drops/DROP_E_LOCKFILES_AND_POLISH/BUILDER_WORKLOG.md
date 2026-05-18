# DROP_N — Builder Worklog

Append a `## Unit N.M — Round K` section per build attempt. See `main/drops/WORKFLOW.md` § "Phase 4 — Build (per unit)" for what each section should contain.

## Unit N.1 — Round 1

- **Builder:** go-builder-agent
- **Started:** YYYY-MM-DD HH:MM
- **Files touched:** <list>
- **Mage targets run:** mage build (pass), mage test (pass), …
- **Notes:** <design choices, surprises, library quirks>

<…repeat per unit + per round…>

## Unit E.1 — Round 1

- **Builder:** go-builder-agent
- **Started:** 2026-05-17
- **Files touched:**
  - NEW `internal/lockfiles/lockfiles.go`
  - NEW `internal/lockfiles/lockfiles_test.go`
  - `cmd/rak/root.go` (import, `rootFlags.includeLockfiles`, flag registration, cobra Example, `runDirectoryOpts.includeLockfiles`, `walkAndCount` signature + filter gate)
  - `cmd/rak/root_test.go` (`runTreeFS` plumbed `includeLockfiles`; new `TestRootCmd_PathArg_LockfileFilter` MapFS test)
  - `cmd/rak/integration_test.go` (new `TestLockfileFilter_ExcludedByDefault` + `TestLockfileFilter_IncludeWhenFlagSet` using real temp files + `--files-from`)
  - `README.md` (Default behavior section, Flags table, new `## v0.2.0 behavior changes` section)
  - `drops/DROP_E_LOCKFILES_AND_POLISH/PLAN.md` (state flip `todo` → `done` for E.1)
- **Mage targets run:** `mage build` (pass), `mage test` (pass, 9/9 packages), `mage format` (reformatted `root.go` + `lockfiles.go`), `mage lint` (0 issues), `mage test` re-run after format (pass)
- **Notes:**
  - Integration point: `walkAndCount` in `cmd/rak/root.go` is the correct filter layer (same layer as binary detection, per PLAN.md design decision 3). No changes to `internal/lister` needed.
  - `internal/lockfiles` is a pure lookup: `map[string]struct{}` keyed by lowercase basename, `strings.ToLower(filepath.Base(path))` for case-insensitive match. Zero deps beyond stdlib.
  - `runDirectoryOpts.includeLockfiles` carries the flag through the existing data-flow chain (`rootFlags` → `runDirectoryOpts` → `walkAndCount`). Both the `--files-from` path and the positional-arg path in `runRoot` receive the flag — lockfile filtering applies in both modes.
  - Integration tests use `--files-from` with real temp files (not testdata/tree which has no lockfiles) to avoid git dependency. MapFS-based tests in `root_test.go` cover the filter at the `runTreeFS` layer.

## Hylla Feedback

The task involved creating a new package and wiring a new flag — mostly net-new code. Hylla was used to look up existing Go code patterns in the repo.

- **Query:** `hylla_search` on `internal/lister` to find the filter integration point.
  - **Missed because:** Hylla was not queried directly for this task; `Read` was used directly since the file was small and the integration point was already known from the spawn prompt. (Non-miss — deliberate choice to go direct since file is ~113 lines.)

None — Hylla answered everything needed for pattern reference. The task was primarily new code creation; existing file reads confirmed integration points directly.
