# DROP_D — PLAN_QA_FALSIFICATION (Round 1)

**Verdict:** FAIL with 7 CONFIRMED counterexamples, 6 issues recommended for dev signoff, 5 noted-and-accepted attacks. Plan is mostly sound but has a load-bearing data-loss bug (`#`-comment filtering) and one cross-cutting spec inconsistency (Q7 `errors.Join` vs iterator-per-line) that must resolve before build.

Verdicts:
- **CONFIRMED** = concrete counterexample with reproducible case or contradiction in the spec itself
- **DEV-SIGNOFF** = design call where I have a strong preference but dev owns the call
- **REFUTED** = attack attempted, plan handles it
- **ACCEPTED** = noted edge case, plan is fine as-is

---

## 1. CONFIRMED Counterexamples

### 1.1 (C1) `#` comment filter silently drops files named `#foo.txt`

**Where:** PLAN.md line 19 (Scope) + line 74 (D.1 step 2) + Q4 line 415-417.

**Counterexample:** a user runs `find . -name '#*' | rak --files-from -` against a macOS-style lockfile tree containing `#bar~`, `#tempfile`, `#.lock`. The plan says: "trim whitespace, skip if empty, skip if starts with `#`". So `#bar~` after trim is `#bar~` → starts with `#` → **silently dropped** with no error, no warning, no aggregated `aggErrs` entry. The user sees a smaller file count than `find` reports and has no signal why.

This is not theoretical: emacs lockfiles (`.#foo`), version-control turds (`#merge.bak#`), and IDE tempfiles all start with `#`. Real filenames.

**Why the precedent doesn't carry:** PLAN.md cites `git rev-list --stdin` as the comment-syntax precedent. But `git rev-list --stdin` consumes **rev specifiers** (`HEAD~3`, `v1.0..main`), not pathnames — `#` as comment is unambiguous in rev-spec syntax because no valid revision starts with `#`. In rak's `--files-from`, the inputs ARE pathnames and `#` is a perfectly legal filename leading character on every Unix filesystem.

**Mitigations to consider:**
- **Option A (recommended — drop the feature):** delete the `#` comment skip entirely. Users who want comments pipe through `grep -v '^#'`. Aligns with YAGNI; matches `xargs` and `wc --files0-from` behavior (neither strips `#` comments).
- **Option B:** require `\#foo.txt` escape syntax for literal-`#` filenames. Adds escape-parsing surface; more code; matches no precedent in Go's stdlib.
- **Option C:** keep `#` filter but emit the dropped lines into `aggErrs` as warnings. Users at least see "I dropped 3 lines as comments." Smallest UX win for the cost.

**Strongest argument:** the feature has no use case in v0.1. No user is going to author a `.rakfiles` config file with comments today. `git rev-list` precedent is the only justification, and it doesn't apply to pathname inputs. Cut the feature.

### 1.2 (C2) `rootLabel = "-"` renders as literal `-` in output

**Where:** PLAN.md line 184 (D.2 step 5) + Q3 line 405-414.

**Reproduction trace:**
1. User runs `rg --files | rak --files-from -`.
2. `flags.filesFrom == "-"`.
3. PLAN.md step 5 passes `rootLabel: flags.filesFrom` → `"-"` reaches `runDirectory`.
4. `runDirectory` calls `labelDirectories(dirs, "-")`.
5. `labelDirectories` at `root.go:536` does `rootLabel = path.Clean(rootLabel)`. `path.Clean("-")` returns `"-"` (verified: Clean only collapses `.`, `..`, multi-slash; `-` is not special).
6. Every directory bucket gets path `-` (for root) or `-/sub` (for nested). TOON output reads `path: -` and `path: -/cmd`.

Hard to read; ambiguous between "stdin sentinel" and "literal dash filename"; breaks parsers downstream that expect a path-like rootLabel.

**Mitigation:** at the entry to the `--files-from` branch, normalize:
```go
rootLabel := flags.filesFrom
if rootLabel == "-" {
    rootLabel = "<stdin>"   // or "(stdin)"
}
```
Or PLAN.md's Q3 Option C is the right call. Pick C, not A.

### 1.3 (C3) Q7 (errors.Join) contradicts unit D.1 step 5 (per-line yield)

**Where:** PLAN.md Scope line 21 ("per-line errors aggregate via `errors.Join` so one bad path doesn't crash the whole stream") AND Q7 line 432-438 ("Recommendation: `errors.Join`-style aggregation") vs unit D.1 step 5 line 81-83 ("yield `(nil, fmt.Errorf(...)`) and continue (per-line error aggregation via the iterator contract — the walk continues past bad lines)").

These are **different mechanisms.** The iterator-contract per-line approach yields each error individually through the `iter.Seq2[*File, error]` stream — `walkAndCount` then collects them into `aggErrs []error` via the existing F6 loop at `root.go:368-380`. That is the existing convention shared with `GitLister.List` and `Walker.Walk`. **No `errors.Join` is needed or called anywhere in the code** — the renderer's error summary already iterates the `[]error` slice and stringifies each.

If a builder reads PLAN.md Scope + Q7 and decides to actually call `errors.Join(err1, err2, ...)` and yield it as a single bundled error, the behavior diverges from the per-line spec at line 81-83 and breaks parity with `GitLister` / `WalkLister`.

**Mitigation:** PLAN.md must say either "iterator-contract per-line, NO `errors.Join`" everywhere, or "buffer all errors and yield one `errors.Join` at end" everywhere — pick one. Recommend per-line (matches existing lister convention). Delete the `errors.Join` mention from Scope line 21 and Q7 entirely; rename Q7 to "Error aggregation: iterator-contract per-line (matches GitLister/WalkLister)" with no `errors.Join` reference.

### 1.4 (C4) `--max-files` interaction is unspecified

**Where:** PLAN.md Notes / Q1-Q7. None addresses `--max-files`.

**Counterexample:** user runs `find /huge/tree -name '*.go' | rak --max-files 100 --files-from -`. What happens?

- If the answer is "limit applies": the existing `walkAndCount` at `root.go:443-445` returns `ErrMaxFilesExceeded` mid-stream — the user gets a hard error after exactly 100 files. That's the v0.1.4 walk-mode behavior; consistent.
- If the answer is "limit ignored": users get no safety rail when piping in a file list, asymmetric with walk mode.

The implementation actually inherits the behavior automatically because `walkAndCount` is called from both branches — `--max-files` will fire on the `--files-from` branch too without any explicit code. So the question is "is that the correct behavior?" Yes — but **PLAN.md must say so** or the QA-proof reviewer won't know whether to write a test for it (and the build-QA reviewer won't know whether to attack it).

**Mitigation:** add Q8 "`--max-files` interaction: applies (no special handling needed; `walkAndCount` enforces). Test in D.3."

### 1.5 (C5) Path conflict with `--no-gitignore` semantics in a git checkout

**Where:** PLAN.md D.2 step 6 (line 208-212).

**Counterexample:** user runs `rak --files-from - --no-gitignore` from inside the rak repo with stdin sending a file list. PLAN.md says "`Detect` is never called in this branch, so `ErrNoGitignoreInRepo` is never raised. `--no-gitignore` silently does nothing."

But the user's mental model is "I'm in a git repo, I passed `--no-gitignore`, I expect either the hard error from v0.1.3 OR a successful walk that ignores `.gitignore`." The plan ships *neither*. The flag is just silently no-op.

This is a footgun: `rak --no-gitignore` alone hard-errors in a repo; `rak --no-gitignore --files-from -` does not. The semantics diverge.

**Mitigation options:**
- **A (silent no-op, current plan):** add a one-sentence README note. Lowest friction.
- **B (hard-error):** if `flags.noGitignore && flags.filesFrom != ""`, return `"--no-gitignore has no effect with --files-from"`. Loud; matches the v0.1.3 sentinel pattern.
- **C (warn-and-continue):** print to stderr "warning: --no-gitignore is a no-op with --files-from".

Recommend B for consistency with the v0.1.3 `ErrNoGitignoreInRepo` precedent. The whole point of that sentinel was "don't let users think a no-op flag is doing something." Same logic applies here.

### 1.6 (C6) Per-line iteration ignores `bufio.Scanner` error mode

**Where:** PLAN.md D.1 step 2 (line 73-74).

**Counterexample:** `bufio.Scanner.Scan()` returns `false` for BOTH end-of-input AND error. After the loop, the caller must check `scanner.Err()` to distinguish. PLAN.md step 2 says "`bufio.Scanner` over `r`. Each line is: trim..." — it does not say to check `scanner.Err()` after the loop.

If `r` is a pipe that closes unexpectedly mid-stream (e.g. broken pipe from a producer that crashed), `Scan()` returns `false`, the iterator exits silently, and the user sees a partial count with no error.

**Mitigation:** PLAN.md D.1 step 2 must add: "After the scan loop exits, if `scanner.Err() != nil`, yield `(nil, fmt.Errorf("lister: files-from: read: %w", scanner.Err()))`." Otherwise read-errors are swallowed.

Also: `bufio.Scanner` has a default 64KB line limit (`bufio.MaxScanTokenSize`). A pathname longer than 64KB will error with `bufio.ErrTooLong`. Linux's `PATH_MAX` is 4096, so this should never happen on real input — but `find -print0`-style noise piped through `tr '\0' '\n'` could exceed it. Acceptable as-is, but worth a one-line "line limit: 64KB (bufio default)" note.

### 1.7 (C7) Q1 recommendation contradicts itself on `--include`/`--exclude`

**Where:** PLAN.md Q1 line 391-399.

The recommendation says:
> Recommendation: YES. `--files-from` only sources the candidate list; `walkAndCount` already applies lang + binary filters post-listing. `include`/`exclude` glob filters live in `WalkOptions` which is not used in the `--files-from` branch — so `--include`/`--exclude` would NOT apply unless `FilesFromLister` is given an explicit filtering step.

The first sentence says YES — filters apply. The next two sentences explain that `--include`/`--exclude` will NOT apply because they're plumbed through `WalkOptions`, which `FilesFromLister` bypasses. Then: "Dev decision: should `FilesFromLister` respect `--include`/`--exclude`?"

So the answer is: **`--lang` applies, `--include`/`--exclude` don't, by accident of plumbing.** That asymmetry will surprise users who pass `rak --files-from - --include '*.go'` expecting filtering and getting nothing.

**Mitigation:** decide the policy explicitly, then either (a) document the asymmetry in `--help` for `--include`/`--exclude` ("note: ignored with --files-from") or (b) add an `ignore.Matcher` filter pass inside `FilesFromLister` so `--include`/`--exclude` apply uniformly. Recommend (b) for least surprise — the caller filtered once, but rak's user-facing contract should be "the same flag means the same thing in every mode."

---

## 2. DEV-SIGNOFF (design questions, plan is fine but needs explicit call)

### 2.1 Symlink-following inconsistency with v0.1.4 walk-root behavior

PLAN.md D.1 step 5 uses `os.Stat(absPath)` which **follows symlinks**. v0.1.4's `Detect` at `lister.go:59` calls `filepath.EvalSymlinks(absRoot)` — also follows. So `--files-from` will count the symlink TARGET, not the symlink itself. Consistent with v0.1.4 — but PLAN.md doesn't say so.

**Ask:** dev signoff that "symlinks in input list resolve to target." Add one sentence to D.1 step 5.

### 2.2 Interactive stdin UX (no EOF)

`rak --files-from -` on a TTY blocks reading lines until Ctrl-D. That's standard Unix behavior (matches `wc`, `cat`, `xargs`). PLAN.md doesn't mention it.

**Ask:** dev signoff that TTY-detection warning is NOT needed. Acceptable as-is.

### 2.3 Duplicate paths in input

`echo -e "a.txt\na.txt\na.txt" | rak --files-from -` will count `a.txt` three times. No dedup. Acceptable (matches `wc -l a.txt a.txt a.txt`), but PLAN.md doesn't say.

**Ask:** dev signoff "no dedup; matches wc/xargs." One-sentence README note.

### 2.4 Absolute paths in input

`echo "/etc/hosts" | rak --files-from -`. `filepath.Abs("/etc/hosts")` returns `/etc/hosts` unchanged. `dirKey` returns `/etc` then `labelDirectories` rewrites `.` → `rootLabel`. But the file's `RelPath` will be its absolute path; `dirKey(absPath)` returns the parent absolute path, NOT `.`. So `labelDirectories`' `if d.Path == "."` branch never fires for absolute paths — the rendered output shows `path: /etc` as the bucket, and `rootLabel` is unused for those entries.

That's actually fine — absolute paths render with their real parent dir, which is the most informative thing. But PLAN.md should say so under "Path interpretation."

**Ask:** dev signoff that absolute paths render with their real parent; `rootLabel` is unused for them.

### 2.5 Paths outside CWD

`echo "../sibling/file.txt" | rak --files-from -`. `filepath.Abs` resolves to absolute → renders with absolute parent dir as above. Acceptable.

**Ask:** none — same as 2.4.

### 2.6 Q5 conflict error message — accept

`"cannot combine --files-from with a positional path argument"` is fine. Matches cobra style. No counterexample.

---

## 3. ACCEPTED (attack attempted, plan handles it)

### 3.1 Empty input → zero directories → renderer panic? REFUTED.

Checked `toonRenderer.RenderTree` at `internal/render/toon.go:139`: builds `rows := make([]toonDirectory, 0, len(s.Dirs))` — empty slice case is handled by capacity-0 make; the for-loops over `s.Dirs` execute zero iterations; `Total` block still emits. `toon.Marshal` with empty `Directories` slice + omitempty `ByLang` + omitempty `TotalByLang` + omitempty `Errors` → output is `{directories:|, total:{bytes:0,lines:0,words:0,chars:0}}` or similar. Verified-safe.

`humanRenderer.RenderTree` at `internal/render/human.go:81`: same pattern — `for _, d := range s.Dirs` is a no-op on empty; `printer.KV(countsKV("total", s.Total))` still emits zeros. Safe.

`jsonRenderer` (not read in full but follows identical encoder pattern): safe by inspection of the type-conversion approach in `Directory`'s declaration order pin.

PLAN.md Q6 already calls this out and says "verify renderer handles this gracefully." Confirmed.

### 3.2 Path with embedded whitespace (`My Documents/foo.txt`). REFUTED.

`bufio.Scanner` splits on `\n` only; embedded spaces are preserved in the line value. `strings.TrimSpace` only trims leading/trailing — `"My Documents/foo.txt"` survives. `filepath.Clean` + `filepath.Abs` handle internal spaces fine. Plan is correct.

### 3.3 Path with tab character. REFUTED.

Same — tab is not a newline; preserved. `TrimSpace` strips leading/trailing tabs, but those would be syntax noise anyway.

### 3.4 Whitespace-then-`#` (`  #comment`). REFUTED.

PLAN.md says "trim whitespace, skip if empty, skip if starts with `#`". After trim, `"  #comment"` becomes `"#comment"` → starts with `#` → skipped. Same behavior as Unix line-comment convention. Consistent with the (debatable) decision in C1 above — but the trim-then-check order is internally consistent.

### 3.5 Windows-style paths. ACCEPTED with note.

`C:\path\to\file` on Windows: `filepath.Clean` normalizes (`C:\path\to\file`), `filepath.Abs` resolves the drive prefix. Should work on Windows. Plan does not call out Windows explicitly — but rak's target platforms are Unix-like (per `integration_test.go:233`), so this is acceptable.

### 3.6 Stream C interaction (`--workers`). ACCEPTED.

Worker pool sits AFTER the iterator. `walkAndCount` iterates `source.List(ctx)` serially today; if Stream C parallelizes it, the iterator is the work queue. `FilesFromLister.List` is a `func` that yields one File at a time — composable with any worker pool that ranges over `iter.Seq2`. Order non-determinism with workers is the existing v0.2.0 concern, not D's. Cross-stream coordination is correctly flagged in PLAN.md Notes.

---

## 4. Hylla Feedback

N/A — action item touched non-Go files only (PLAN.md is markdown; Go source reads were verification, not driving the analysis).

The Go code reads (root.go, lister/*.go, fileset/file.go, render/toon.go, render/human.go) used `Read` directly per `main/CLAUDE.md` rule 1 since the relevant files are small and the symbol lookups (per-line `yield(nil, err)` convention, `path.Clean` semantics, renderer zero-input safety) are concrete enough that exhaustive Hylla search would not have added evidence. No Hylla miss to report.

## TL;DR

- **T1** — Seven CONFIRMED counterexamples: (C1) `#`-comment filter silently drops legitimate `#foo.txt` filenames — strong YAGNI argument for cutting the feature; (C2) `rootLabel = "-"` renders as ugly literal dash, recommend `"<stdin>"`; (C3) Scope+Q7 mention `errors.Join` but D.1 step 5 uses iterator-contract per-line — pick one; (C4) `--max-files` interaction unspecified; (C5) `--no-gitignore + --files-from` should hard-error (matches v0.1.3 sentinel), not silently no-op; (C6) `bufio.Scanner.Err()` check missing — swallows mid-stream read errors; (C7) Q1 recommendation is internally contradictory on `--include`/`--exclude`.
- **T2** — Six DEV-SIGNOFF items mostly clarifying spec language (symlink-follow, TTY block, dup paths, absolute paths, conflict message). Not blockers, but cheap fixes.
- **T3** — Six attack families REFUTED or ACCEPTED — empty-input renderer-panic, embedded whitespace, trim-then-comment ordering, Windows paths, worker-pool composability — plan handles these correctly.
