# DROP_3 — Plan QA Falsification — Round 2

**State:** round 2 complete
**Agent:** go-qa-falsification-agent
**Target:** main/drops/DROP_3_DIRECTORY_WALK_GITIGNORE_DEPTH/PLAN.md
**Round:** 2
**Verdict:** FAIL — Round 3 revise required (one unmitigated counterexample introduced during revise)

## Premises

- Falsification Round 2 must (a) confirm each Round 1 finding was actually mitigated, not just moved, (b) attack the NEW content added during revise (F14, F15, `DisableGitignore` rename, package-helper `IsHidden`, `TestWalker_SymlinkYielded`, C8 breadcrumb, `doublestar.PathMatch` swap, drop-end O1 closeout) for its own counterexamples, (c) attack the plan holistically for anything Round 1 missed.
- An unmitigated counterexample is a blocker; a surface finding is a polish item that should still land before sign-off.
- Mitigations must be evidence-backed (Hylla / `go doc` / Context7 / source), not hand-waved.

## Evidence

- `go doc iter` — yield-after-false panic semantics confirming F14's necessity.
- `go doc io/fs.WalkDir`, `go doc io/fs.WalkDirFunc`, `go doc io/fs.SkipAll`, `go doc io/fs.SkipDir` — control-flow semantics that F14's closure-bool pattern depends on.
- `go doc testing/fstest.MapFile`, `go doc testing/fstest.MapFS.Open` — `MapFS.Open` *"opens the named file after following any symbolic links"*, so broken-target symlinks return `fs.ErrNotExist` as C4's mitigation claims.
- Context7 `/bmatcuk/doublestar` — `Match` splits on forward slash unconditionally; `PathMatch` uses OS path separator; v4 UPGRADING.md explicitly says PathMatch *"requires platform-specific path separators"*.
- `git show 5a7e893:drops/DROP_3_*/PLAN_QA_FALSIFICATION.md` + `PLAN_QA_PROOF.md` — Round 1 findings (C1–C10, O1, O2).
- `git diff ca5f237..1107cac -- drops/DROP_3_*/PLAN.md` — exact delta from pre-QA baseline to Round 2 revise.
- Plan file at `main/drops/DROP_3_DIRECTORY_WALK_GITIGNORE_DEPTH/PLAN.md` (revised Round 2 content).

## Confirmed counterexamples (blockers)

### CN1 — `doublestar.PathMatch` is the wrong function for `io/fs`-style forward-slash paths; plan line 44's justification inverts library semantics

**Attack:** Unit 3.1 line 44 reads: *"`glob.go` uses `github.com/bmatcuk/doublestar/v4.PathMatch` for `--include` / `--exclude` (O2 — `/`-sensitive matching is required because relative paths like `src/foo.go` need to match `src/**/*.go` patterns correctly; `doublestar.Match` is shell-style and treats `/` as a literal)."*

This has the two functions' semantics reversed per Context7 docs sourced from doublestar's own README + UPGRADING.md:

- **`Match`** (what the plan dismisses): *"`name` and `pattern` are split on forward slash (`/`) characters... drop-in replacement for `path.Match()` which always uses `'/'` as the path separator."* This is exactly what the walker wants — forward-slash splitting on all platforms, including Windows.
- **`PathMatch`** (what the plan picks): *"PathMatch will automatically use your system's path separator to split `name` and `pattern`."* doublestar v4 UPGRADING.md: *"PathMatch() requires platform-specific path separators for both pattern and name."* On Windows, this would require `\` separators — but the walker's `relPath` is always forward-slash per C6 / `io/fs` convention.

**Why this matters:** The plan's own C6 pin (line 41) locks `relPath` to forward-slash on all platforms. Feeding forward-slash `relPath` into `PathMatch` on Windows would split on `\` and mis-match. Feeding it into `Match` works uniformly. doublestar's own `Glob` docs reinforce this for any `io/fs`-rooted workflow: *"Glob requires `/` as the path separator in patterns, even on platforms that use different separators. This is due to the use of the io/fs package."* — rak's walker is `io/fs`-rooted (3.2 uses `fs.FS.Open`), so the same principle applies.

**Mechanism:** If the builder implements per the plan — `doublestar.PathMatch(pattern, relPath)` — then on Windows CI (if rak ever runs there) all include/exclude globs with directory components would silently fail. On macOS/Linux (dev machines) it would work because the OS separator happens to equal `/`, so the bug would hide until cross-platform testing. That's a particularly nasty failure mode because the unit tests (fixtures are `fstest.MapFS` with forward-slash keys) would continue passing on macOS/Linux.

**Mitigation required in Round 3:** Single surgical edit at Unit 3.1 line 44. Swap the function and invert the rationale:

> `glob.go` uses `github.com/bmatcuk/doublestar/v4.Match` for `--include` / `--exclude`. `Match` splits both pattern and path on forward slash (`/`) on all platforms — the correct choice because the walker feeds forward-slash `relPath` values per C6 / `io/fs` convention (doublestar's own `Glob` docs apply the same `/` rule for the same reason). `PathMatch` would split on the OS separator (`\` on Windows) and mis-match forward-slash paths, so it's rejected here. `filepath.Match` is insufficient because users will expect `**/node_modules` and `src/**/*.go` to work; `filepath.Match` rejects `**`.

No F-pin is required; this is library-API semantics, not a cross-unit invariant.

## Surface findings (non-blocking polish)

- **SF1** — Line 14 still reads *"Expected decomposition: 4 units (3.1 fileset / 3.2 ignore / 3.3 binary detection / 3.4 root wiring + per-dir aggregation)."* The actual decomposition is **six** units (3.0–3.5). Pre-existing drift from the scope paragraph, not a Round 2 regression, but worth fixing while the plan is open.
- **SF2** — Line 134 says: *"On iteration error (from the second iter element): wrap `walk: %w` and return."* This is in tension with F6 (line 164): *"Per-entry errors in the walker are yielded, not fatal. The iterator continues past a broken dir so one permission error doesn't abort the whole count."* The `runRoot` caller returning on the first yielded error terminates iteration regardless of the walker's own resilience, which effectively nullifies F6 for this call site. Either (a) change line 134 to aggregate walker errors into the render's error summary like line 143 already does for `IsBinary` errors, or (b) weaken F6 to say "the walker is resilient; callers may abort on first error if they choose." Pre-existing, not new in Round 2.

## Trace or cases

- **CN1 trace (Windows builder):** `walker.Walk` yields `*File{RelPath: "src/foo.go"}`. `runRoot` filters via `doublestar.PathMatch("src/**/*.go", "src/foo.go")`. On Windows (system separator `\`), `PathMatch` splits both on `\`, sees `src/**/*.go` and `src/foo.go` as unsplit strings, and returns false where true is expected. On macOS/Linux (system separator `/`), PathMatch splits on `/` and returns true. Tests pass on dev machines; silent cross-platform regression.
- **CN1 trace (macOS dev):** Works by coincidence because the OS separator equals the `io/fs` separator. Bug undetectable until Windows CI or first Windows user.
- **F14/C1 trace (re-verified):** caller `break`s on first emission → `yield(f, nil)` returns false → `WalkDirFunc` sets captured bool, returns `nil` → `fs.WalkDir` calls `WalkDirFunc` for the next entry → that invocation reads captured bool → returns `fs.SkipAll` → `fs.WalkDir` terminates → outer `iter.Seq2` function returns cleanly → range loop continues past the `break`. No yield call after false. No panic.
- **C4 trace (re-verified):** `TestWalker_SymlinkYielded` constructs `MapFS{"target.txt": {Data: ...}, "link_ok": {Mode: fs.ModeSymlink, Data: []byte("target.txt")}, "link_broken": {Mode: fs.ModeSymlink, Data: []byte("missing.txt")}}`. Walker yields all three as regular DirEntries (fs.WalkDir surfaces the symlink's own DirEntry without following). Test asserts `entry.Type()&fs.ModeSymlink != 0` on `link_ok` and `link_broken`. Test calls `link_broken.Open()` → `MapFS.Open` follows symlink → tries to open `missing.txt` → returns wrapped `fs.ErrNotExist`. All three assertions land.

## Conclusion

**Verdict: FAIL (Round 3 revise required).** One CONFIRMED counterexample (CN1) introduced during Round 2's O2 mitigation — the planner picked `PathMatch` and justified it with a description that inverts the library's documented semantics. All four Round 1 blockers (C1–C4) and all six surface findings (C5–C10) are genuinely mitigated, and both proof observations (O1, O2) are addressed structurally — O2's problem is the content of the fix, not its absence. Round 3 needs one line-44 edit to swap `PathMatch` → `Match` and rewrite the justification.

## Unknowns

- None on the falsification side. CN1 is mechanically specifiable per Context7 docs on doublestar v4. SF1 and SF2 are pre-existing pointer-to-pointer ambiguities, not evidence gaps.
