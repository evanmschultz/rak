# DROP_D — FILES_FROM_PIPE

**State:** planning
**Tier:** A
**Blocked by:** —
**Paths (expected):** NEW internal/lister/filesfrom.go (or extend internal/lister/), internal/lister/lister.go (factory routing), internal/lister/lister_test.go, cmd/rak/root.go, main/docs/tapes/pipe.tape (NEW), main/docs/pipe.gif (NEW), README.md
**Packages (expected):** internal/lister, cmd/rak
**PLAN.md ref:** — (top-level PLAN.md removed at v0.1.0 ship; see memory `session_handoff_2026_05_16_v020_planning.md`)
**Workflow:** main/drops/WORKFLOW.md
**Started:** 2026-05-16
**Closed:** —

## Scope

Add `--files-from <FILE>` for pipe composition — the missing link between rak and the wider Unix toolchain.

- **New flag**: `--files-from <FILE>`. Use `-` (literal hyphen) to read from stdin. Reads newline-separated paths; each path is counted as a single file (re-uses the existing `SingleFileLister` machinery from v0.1.4).
- **Stdin sentinel**: bare positional stdin (`cat README.md | rak`) is unchanged — still single-stream wc-parity counting. `--files-from -` is the explicit opt-in to "read a list of paths from stdin."
- **Path interpretation**: paths are interpreted relative to the current working directory. Empty lines are skipped. Lines starting with `#` are treated as comments and skipped (standard Unix convention; matches `git rev-list --stdin` precedent).
- **Path normalization**: each path goes through `path.Clean` + the same regular-file check from v0.1.4's `SingleFileLister`.
- **Error semantics**: missing file → friendly error (`not a regular file or directory: <path>`); per-line errors aggregate via `errors.Join` so one bad path doesn't crash the whole stream.

**Unblocks the canonical Unix-composition workflows:**
- `rg --files | rak --files-from -`
- `git ls-files '*.go' | rak --files-from -`
- `find . -name '*.go' | rak --files-from -`
- `gh pr diff 42 --name-only | rak --files-from -`

**Out of scope (deferred per dev 2026-05-16):**
- NUL-delimited variant `--files0-from <FILE>` — defer to v0.2.1 or v0.3. Hardens against filenames with newlines/spaces.

**Feature trio (mandatory per memory `feedback_rak_docs_and_gifs_before_pr.md`):**

1. VHS demo: `main/docs/tapes/pipe.tape` + `main/docs/pipe.gif`. Show `git ls-files '*.go' | rak --files-from -` against a fixture. Embed in README near a new "Piping" narrative section.
2. README examples: at minimum the four invocations above in "Common invocations" + a "Piping" narrative section.
3. Cobra `Example:` entries in `cmd/rak/root.go` for at least two of the four invocations (typically `rg --files | rak --files-from -` and `git ls-files '*.go' | rak --files-from -`).

## Planner

<Filled by go-planning-agent in Phase 1.>

## Notes

**Cross-stream coordination**: Streams B, C, D all add new flags to `cmd/rak/root.go`. The planner should make the cmd/rak flag-wiring unit explicit so the orchestrator can serialize it against B and C at build time. Internal-package work (`internal/lister/*`) is parallel-safe with the other streams.

**Integration with existing `internal/lister`**: `--files-from` is a new `FileLister` impl that should plug into the existing `Detect(root, fsys)` factory. The factory should branch on `--files-from` being set BEFORE the git/walk dispatch — `--files-from` overrides root-based detection entirely.
