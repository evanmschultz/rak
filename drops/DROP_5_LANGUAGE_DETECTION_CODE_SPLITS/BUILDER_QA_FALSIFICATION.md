# DROP_5 — Builder QA Falsification

Append a `## Unit N.M — Round K` section per build-QA falsification attempt. See `main/drops/WORKFLOW.md` § "Phase 5 — Build-QA (per unit)" for the round shape.

## Unit 5.1 — Round 1

- **Reviewer:** go-qa-falsification-agent
- **Verdict:** PASS — no CONFIRMED counterexample after 13 attack families.
- **Diff under attack:** `f159596 feat(lang): add detect pipeline, wire into walkandcount` — `internal/lang/lang.go` (new, 230 LOC), `internal/lang/lang_test.go` (new, 168 LOC), `cmd/rak/root.go` (+8 LOC; one import + 6-line wire).

### Attack outcomes

| # | Attack | Outcome | Notes |
|---|---|---|---|
| 1 | `Makefile.go` mis-classified as Makefile | REFUTED | `strings.ToLower(filepath.Base("Makefile.go"))` = `"makefile.go"`, not a key in `specialFilenames`; falls through to `.go` → LangGo. Covered by `TestDetect_SpecialFilename` row `{"Makefile.go", LangGo}`. |
| 2 | Nested-path basename (`sub/Makefile`) | REFUTED | `filepath.Base("sub/Makefile")` = `"Makefile"` → lowercase `"makefile"` → LangMakefile. Covered by test row `{"sub/Makefile", LangMakefile}`. |
| 3 | `#!/usr/bin/env python3` env-indirection | REFUTED | `detectShebang` (lang.go:163–207) detects `interp == "env"`, skips `-`-flagged args, uses `python3` as lookup key. Covered by `TestDetect_Shebang_Python`. Also handles `env -S python3` (flag-skip loop). |
| 4 | F27: Peek error must yield LangUnknown silently | REFUTED | lang.go:145–149: `buf, err := f.Peek(512); if err != nil { return LangUnknown }`. No `log`, no `fmt.Errorf`, no panic. Covered by `TestDetect_PeekError_ReturnsUnknown` using an empty `fstest.MapFS` so `Open` returns `fs.ErrNotExist`. |
| 5 | F26 RelPath invariant — no mutation | REFUTED | `f.RelPath` is read at lang.go:131 (`filepath.Base`) and :137 (`filepath.Ext`) only. No assignment anywhere in the package. The `*fileset.File` struct field is set only by `fileset.newFile` (file.go:52–58); no external write surface. |
| 6 | `_ = detectedLang` code smell | NOT A COUNTEREXAMPLE — accepted | Spec line 60 explicitly mandates "Store the resulting `Language` value in a per-file local (e.g. `detectedLang := lang.Detect(f)`)" for 5.2/5.4 to build on. The `_ = detectedLang` line with comment `// consumed by 5.3 (Split) and 5.4 (--lang filter)` suppresses the unused-variable compile error while preserving the call-site hook. Documented in BUILDER_WORKLOG.md. Acceptable. |
| 7 | Language-constant count | REFUTED | 18 constants present (LangUnknown + 17 named). Spec minimum: 15 from the named list + LangDocker + LangCMake = 17 named. All present. ".xml" → LangHTML is a builder-choice content-heuristic alias, documented in BUILDER_WORKLOG.md; spec line 43 says "Additional entries are welcome but not required." Acceptable. |
| 8 | Drop 3/4 spine preserved | REFUTED | `git diff HEAD~1 -- internal/fileset/ internal/lister/ internal/ignore/ internal/counting/ internal/render/` is empty. Only `cmd/rak/root.go` and new `internal/lang/` files were touched. |
| 9 | Case sensitivity — `MAKEFILE` (all caps) | EXHAUSTED, no counterexample | `strings.ToLower("MAKEFILE")` = `"makefile"` → LangMakefile via std-lib. The test table covers `Makefile`, `makefile`, `GNUmakefile`, but not `MAKEFILE`. This is a test-table gap, not a code defect — std-lib `strings.ToLower` is trusted. Worth widening the table in a future drop; not a falsification. |
| 10 | `mage ci` regression | REFUTED | Re-ran `mage ci` from `main/`. 0 issues; all 7 packages green (counting, fileset, ignore, lang, lister, render, cmd/rak). |
| 11 | `mage install` invocation | REFUTED | Worklog shows builder ran `mage test` and `mage ci` only. No `mage install`, no raw `go build` / `go test` / `go vet`. |
| 12 | Concurrency / goroutine leaks | REFUTED | `Detect` is pure synchronous — no goroutines, no channels, no shared mutable state. `Peek` is stateless per F4 (open-read-close per call). Safe under the future parallel walker. |
| 13 | Error swallowing | REFUTED | The single error-discard site (Peek failure at lang.go:145–149) is the F27-mandated silent fallthrough, explicitly commented. No `fmt.Errorf` without `%w` (none are emitted at all from `Detect`). |

### Edge-case probes (extra adversarial passes that produced no counterexample)

- **Empty Peek buffer**: `detectContent` guards `len(buf) == 0` → LangUnknown (lang.go:212–214). Safe.
- **`"#!"` with no interpreter body**: `firstLine[2:]` after TrimSpace yields `""`; `detectShebang` returns LangUnknown (lang.go:176–178). Safe.
- **`#! /bin/bash` (space after `#!`)**: TrimSpace + `strings.Fields` recovers `["/bin/bash"]` → basename `bash` → LangShell. Safe.
- **`#!/usr/bin/env -S python3` (env with flag)**: flag-skip loop at lang.go:194–199 skips `-S`, uses `python3`. Safe.
- **Generic-state contraband (C5)**: `Detect` returns the first concrete match OR LangUnknown. No "generic" intermediate. REFUTED.
- **Block-comment language fields in shebang table**: shebangsTable correctly excludes file-ext-only languages (Go, Rust, C, etc.) — shebang is only consulted when steps 1+2 returned LangUnknown.

### Hidden-attack residual

None survived. The single non-counterexample observation is the all-caps `MAKEFILE` test-table gap (Attack 9) — flagged as a future test-widening opportunity but explicitly NOT a falsification because `strings.ToLower` is std-lib trusted.

### Hylla Feedback

N/A — review touched the new `internal/lang/` package which would not be in the prior Hylla ingest (drop-end-only reingest per main/CLAUDE.md). Evidence-gathering went through `Read` on lang.go / lang_test.go and `git diff HEAD~1` for the deltas; no Hylla fallback miss to log because Hylla was not the right tool for an under-review uncommitted-since-ingest package.

### Certificate

- **Premises:** Unit 5.1 must (a) define `Language` + 17 named constants, (b) implement 4-step Detect pipeline per F27, (c) preserve F26 RelPath invariant, (d) silently swallow Peek errors per F27/P3, (e) wire `lang.Detect(f)` once per file into `walkAndCount` without consuming the result yet, (f) keep Drop 3/4 spine untouched, (g) pass `mage ci`.
- **Evidence:** `internal/lang/lang.go` (230 LOC, read in full); `internal/lang/lang_test.go` (168 LOC, read in full); `cmd/rak/root.go` diff (+8 LOC); `git diff HEAD~1 -- <spine>` empty; `mage ci` green on all 7 packages; BUILDER_WORKLOG.md confirms `mage test` + `mage ci` only.
- **Trace or cases:** 13 attack families + 6 edge-case probes = 19 distinct adversarial probes. All REFUTED, EXHAUSTED, or NOT-A-COUNTEREXAMPLE. Zero CONFIRMED.
- **Conclusion:** PASS. The unit survives Round 1 falsification with no unmitigated counterexample.
- **Unknowns:** None routed. One test-widening suggestion (add `MAKEFILE` all-caps to `TestDetect_SpecialFilename` table) is a minor future polish item, not a gate.

## Unit 5.2 — Round 1

- **Reviewer:** go-qa-falsification-agent
- **Verdict:** PASS — no CONFIRMED counterexample after 9 attack families + 12 edge-case probes.
- **Diff under attack:** `0e2ddc0 feat: add split and --lang filter for drop 5 units 5.2 + 5.4` — `internal/lang/split.go` (new, 190 LOC), `internal/lang/split_test.go` (new, 217 LOC). Unit 5.2 owns the `internal/lang/` half of the commit only; 5.4 owns the `cmd/rak/` half.

### Attack outcomes

| # | Attack | Outcome | Notes |
|---|---|---|---|
| 1 | Policy α `/* a */ b /* c */` — exactly 1 Comment, state correct | REFUTED | Trace: (b) `Contains("/*")` true → 1 Comment emitted. State machine: openIdx=0,closeIdx=5 → open<close → inBlock=true,idx=2 → openIdx=8,closeIdx=2 → close<open → inBlock=false,idx=6 → openIdx=2,closeIdx=7 → open<close → inBlock=true,idx=10 → openIdx=-1,closeIdx=3 → inBlock=false,idx=15 → both -1, break. Final inBlock=false. ✓ matches `TestSplit_BlockCommentOpenClosePerLine`. |
| 1b | Policy α `/* still open` then `closing */` — 2 Comments, state transitions | REFUTED | Line 1: (b) `Contains("/*")` true → 1 Comment; state openIdx=0,closeIdx=-1 → inBlock=true,idx=2 → both -1, break. Line 2: (a) inBlock=true → 1 Comment; state openIdx=-1,closeIdx=8 → inBlock=false. ✓ matches `TestSplit_BlockSpansMultipleLines` two-line pattern. |
| 2 | **CRITICAL — empty block-open guard** (`strings.Contains(s, "")==true` would mis-classify EVERYTHING) | REFUTED | split.go:148 `if !isComment && g.blockOpen != "" && strings.Contains(line, g.blockOpen)`. Same guard at :151 (blockClose) and :156 (linePrefix). All three are `!= ""`-gated. Python (`{linePrefix:"#"}`, blockOpen/Close empty) traces: line `x = 1` → (a) false, (b) skip on `g.blockOpen != ""` false, (c) skip on `g.blockClose != ""` false, (d) linePrefix="#" but trimmed `x = 1` does NOT start with `#` → Code. Covered by `TestSplit_PythonHash`. |
| 3 | LangJSON fall-through — `// not a comment` must be Code | REFUTED | `grammarTable[LangJSON]` is map-miss → zero `grammar{}` (all three fields ""). All three checks guarded → falls to Code branch. ✓ `TestSplit_JSON_NoComments` covers `{"key": "// not a comment", "val": 1}` → all 4 lines Code. |
| 4 | LangCMake `#[[ ]]` block markers | REFUTED | Grammar `{linePrefix:"#", blockOpen:"#[[", blockClose:"]]"}`. Trace `#[[ block ]]`: (b) Contains `#[[` → Comment. State: openIdx=0,closeIdx=9 → open<close → inBlock=true,idx=3 → openIdx=-1,closeIdx=6 → inBlock=false,idx=11 → empty,break. ✓ Also: `set(x 1) # note` → (b) no, (c) trimmed does not start with `#` → Code (correct per Policy α: trailing line-comment markers do NOT promote to Comment). Minor: no `TestSplit_CMake_*` test exists; this is a coverage gap not a defect — grammarTable entry presence is mechanically verifiable. |
| 5 | Python docstring `"""docstring"""` → Code (C7) | REFUTED | Python grammar has no block markers, linePrefix=`#`. Line `    """docstring"""` trimmed = `"""docstring"""` does not start with `#` → Code. ✓ explicit `TestSplit_PythonDocstring_IsCode`. |
| 6 | CRLF double-count or `\r` artifact | REFUTED | `bufio.Scanner` with default `ScanLines` strips final `\r` per Go stdlib (`bufio.ScanLines` doc: "drops any trailing end-of-line marker"). Test `TestSplit_CRLF` with `"line1\r\nline2\r\n"` confirms `{0,0,2}`. Trimmed tokens are `line1`/`line2` — no `\r` artifact in classification. |
| 7 | `strings.Index` off-by-one in state machine (idx arithmetic on `line[idx:]`) | REFUTED | Indices returned by `strings.Index(line[idx:], marker)` are RELATIVE to the slice. Update `idx += openIdx + len(g.blockOpen)` correctly advances past the consumed marker in original-line coordinates. Edge `/**/`: openIdx=0,closeIdx=2 → open<close → inBlock=true,idx=2 → line[2:]=`*/` → openIdx=-1,closeIdx=0 → inBlock=false,idx=4 → break. Final inBlock=false. ✓ Edge `/*/`: openIdx=0,closeIdx=1 → open<close (0<1) → inBlock=true,idx=2 → line[2:]=`/`,both -1,break. Final inBlock=true (unclosed `/*`; the trailing `/` is leftover). Semantically correct. |
| 8 | `LangCounts.Add` commutativity / associativity | REFUTED | Pure integer addition on 7 fields (3 LineCounts + 4 counting.Counts). Integer `+` is commutative+associative; overflow on `int` LOC counts is not realistic for any file rak would process. `TestSplit_LangCounts_Add` covers the basic forward direction. |
| 9 | `mage ci` regression | REFUTED | Re-ran `mage ci` from `main/`. 0 issues; all 7 packages green (counting, fileset, ignore, lang, lister, render, cmd/rak). |

### Edge-case probes (extra adversarial passes that produced no counterexample)

- **Blank line inside open block comment** (`/* a\n\nb */`): line 2 is blank → `lc.Blank++; continue` — DOES NOT touch state. inBlockComment carries true into line 3. Line 3 `b */` → (a) inBlock=true → Comment. Result: Blank=1, Comment=2. Reasonable choice (blank-first wins over block-comment promotion); pinned implicitly by the implementation order. Not a counterexample.
- **`mage install` invocation**: worklog shows only `mage test` + (deferred) `mage ci`. No `mage install`, no raw `go build` / `go test` / `go vet`. REFUTED.
- **Concurrency**: `Split(r io.Reader, lang Language)` is pure-function over a reader. `grammarTable` is `var` declared once, read-only post-init. No goroutines spawned. Safe for concurrent calls on independent readers. REFUTED.
- **Error swallowing**: `scanner.Err()` propagated at split.go:186–188 with `return LineCounts{}, err`. No discard. REFUTED.
- **File-gating bypass**: `git diff HEAD~1 -- internal/lang/split.go internal/lang/split_test.go` matches the full diff for the lang half of the commit. No edits outside declared paths for 5.2. REFUTED.
- **String-literal known limitation** (`s := "/*"`): explicitly pinned by `TestSplit_StringContainsMarker_KnownLimitation` + grammarTable doc comment + BUILDER_WORKLOG. Accepted YAGNI per F28. Not a counterexample.
- **LangMarkdown HTML block markers**: grammar `{blockOpen:"<!--", blockClose:"-->"}`, no linePrefix. Line `# heading` → no `<!--`/`-->` substring, linePrefix empty → Code. Correct per cloc convention (markdown headings are content, not comments).
- **LangCSS no linePrefix**: grammar `{linePrefix:"", blockOpen:"/*", blockClose:"*/"}`. Line `// not a comment in CSS` → (b) Contains `/*`? no, Contains `*/`? no, (c) linePrefix="" skip → Code. Correct (CSS has no line-comment syntax).
- **`bufio.Scanner` default 64KB token limit**: a single >64KB line returns `bufio.ErrTooLong`. Split propagates via `scanner.Err()` → returns `LineCounts{}, err`. Caller in walkAndCount handles via standard error path. Documentation gap (not surfaced in BUILDER_WORKLOG nor in Split doc comment), but propagation is correct. Routed to Unknowns as a low-priority observability nit.
- **inBlockComment doesn't reset across files**: `Split` declares `inBlockComment := false` locally per call (split.go:127). Each file gets a fresh state. ✓
- **Negative-line guard**: `lc.Blank`/`Comment`/`Code` are unsigned counts incremented monotonically. No negative-value path. REFUTED.
- **Block markers overlapping char-class** (Go `/*` and `*/` share `*`): `strings.Index` does longest-match-from-left scan, not greedy overlap; the state machine's `openIdx<closeIdx` tiebreaker on equal indices favors open (close wins on `else`). For `/*` and `*/` at the SAME position this can't happen (different first chars `/` vs `*`). For CMake `#[[` and `]]` — no shared prefix. REFUTED.

### Hidden-attack residual

None survived. One observability nit (Scanner 64KB token-too-long not documented in Split's godoc) routed to Unknowns. No correctness counterexample.

### Hylla Feedback

N/A — review attacked the new `internal/lang/split.go` + `internal/lang/split_test.go`, both uncommitted-relative-to-last-ingest (drop-end-only reingest per main/CLAUDE.md). Evidence came from `Read` on the two files, `git diff HEAD~1`, the existing `internal/lang/lang.go` for the Language constants, and `mage ci` re-run. No Hylla fallback to log because Hylla was not the right tool for an under-review uncommitted package.

### Certificate

- **Premises:** Unit 5.2 must (a) implement Policy α (F28, Decision C4) blank/comment/code three-way classification, (b) guard `strings.Contains` calls with `g.blockOpen/blockClose/linePrefix != ""` so Python/JSON/LangUnknown (zero-grammar) do not mis-classify every non-blank line as Comment, (c) maintain `inBlockComment` state correctly across lines including multi-line block spans, (d) classify Python `"""docstring"""` as Code (C7), (e) classify all non-blank JSON lines as Code (no grammar entry), (f) populate `grammarTable[LangCMake]` with `#[[`/`]]` block markers, (g) propagate `scanner.Err()` rather than swallow, (h) keep all changes inside `internal/lang/split.go` + `internal/lang/split_test.go`, (i) pass `mage ci`.
- **Evidence:** `internal/lang/split.go` (190 LOC, read in full); `internal/lang/split_test.go` (217 LOC, read in full); `internal/lang/lang.go` Language constants (lines 1–80 read); `git diff HEAD~1` confirms only split.go + split_test.go for the lang half of the commit; `mage ci` green on all 7 packages; BUILDER_WORKLOG.md Unit 5.2 entry confirms only `mage test` was run (no raw go, no `mage install`).
- **Trace or cases:** 9 explicit attacks + 12 edge-case probes = 21 distinct adversarial probes. All REFUTED or accepted-as-documented YAGNI. Zero CONFIRMED.
- **Conclusion:** PASS. Unit 5.2 survives Round 1 falsification with no unmitigated counterexample. The critical empty-block-open guard (Attack 2) is present and correct; the per-line Policy α trace matches its dedicated test for the `/* a */ b /* c */` case; the multi-line state machine carries `inBlockComment` across lines and resets per call; LangJSON correctly falls through to Code via map-zero `grammar{}`; LangCMake grammar table entry is populated per plan.
- **Unknowns:** (i) `bufio.Scanner` 64KB-line `ErrTooLong` propagation behavior is undocumented in Split's godoc — observability nit, not a defect; (ii) no `TestSplit_CMake_*` test exists despite grammarTable having `#[[`/`]]` — coverage gap only; both routed to dev as test-widening opportunities for a future drop, not a Round-1 gate.

## Unit 5.4 — Round 1

- **Reviewer:** go-qa-falsification-agent
- **Verdict:** PASS — no CONFIRMED counterexample after 8 attack families.
- **Diff under attack:** `0e2ddc0 feat: add split and --lang filter for drop 5 units 5.2 + 5.4` — for 5.4 specifically: `cmd/rak/root.go` (+42 -7) and `cmd/rak/root_test.go` (+120 -1). Internal packages untouched.

### Attack outcomes

| # | Attack | Outcome | Notes |
|---|---|---|---|
| 1 | Zero-allocation noop when `--lang` is empty/nil | REFUTED | root.go:244–250: `var wantedLangs map[lang.Language]struct{}` declared nil; `make(...)` only inside `if len(langs) > 0`. Per-file gate at :291 is `if wantedLangs != nil` — single nil compare, no map alloc, no per-file work. `TestRootCmd_NoLangFlag_CountsAll` exercises the nil-langs path (`&rootFlags{}` → langs is nil slice). |
| 2 | Case normalization for `--lang Go` | REFUTED | root.go:248: `wantedLangs[lang.Language(strings.ToLower(v))]` — `Go` → `go` matches `LangGo = "go"` from internal/lang/lang.go. Covered by `TestRootCmd_FlagLang_CaseInsensitive` (line 528). std-lib `strings.ToLower` is trusted. |
| 3 | F29 LangUnknown filter exclusion via `--lang ""` | EXHAUSTED, no counterexample | The wantedLangs build loop unconditionally inserts `Language(strings.ToLower(v))`, including `""` if `v == ""`. If `Language("")` ends up in `wantedLangs`, then `LangUnknown` files (which `lang.Detect` returns as `Language("")`) would match — F29 violation. The builder's defensive comment at root.go:289–290 claims "cobra's StringSliceVar rejects --lang \"\"" but this claim is UNVERIFIED (pflag source gated). Pflag uses `encoding/csv` under the hood, which returns zero records for input `""`, so the empirically expected behavior is that `--lang ""` produces an empty `langs` slice → `len(langs) > 0` is false → `wantedLangs` stays nil → safe. `TestRootCmd_FlagLang_ExcludesUnknown` (line 549) covers the normal `--lang go` case explicitly. Marked Unknown rather than CONFIRMED — a defensive `if strings.TrimSpace(v) == "" { continue }` in the build loop would harden but is YAGNI absent a pflag behavior contradiction. |
| 4 | CSV-parsing test is a sham | NOT A COUNTEREXAMPLE — accepted with caveat | `TestRootCmd_LangFlag_ParsesCSV` at line 587 sets `langs: []string{"go", "rust"}` directly via the struct literal, bypassing cobra and pflag entirely. It tests filter behavior (already covered by `TestRootCmd_FlagLang_MultiValue` at line 507), NOT cobra's CSV split. End-to-end CSV-via-`cmd.SetArgs([]string{"--lang", "go,rust"})` is not exercised in the suite. This is a proof-completeness gap surfaced to proof-agent territory, NOT a behavior counterexample — pflag's documented split-on-comma behavior is std-trusted at the library boundary. Routed to Unknowns for a future widening. |
| 5 | Filter-ordering: Detect → filter → countFile | REFUTED | root.go:284 calls `lang.Detect(f)`; root.go:291–295 is the wantedLangs gate (`continue` on miss); root.go:297 calls `countFile(f)`. Order strictly Detect → gate → count, so a filtered-out file is detected but never counted. Binary-check runs even earlier at :269–278 (correct — binary files should be skipped before language detection too, since they can't be reasonably detected). |
| 6 | Drop 4 surface preservation | REFUTED | `git diff HEAD~1 -- internal/lister/ internal/fileset/ internal/ignore/ internal/render/` returns empty. Unit 5.4's diff is confined to `cmd/rak/root.go` + `cmd/rak/root_test.go`. The `internal/lang/` half of commit `0e2ddc0` is Unit 5.2's surface, separate review. |
| 7 | Drop 3 path-arg test interaction | REFUTED | All `TestRootCmd_PathArg_*` tests call `runTreeFS(t, fsys, &rootFlags{...})` without setting `langs`, so the field is nil. `runTreeFS` at line 203 was updated to pass `flags.langs` to `runDirectory` — nil propagates → `walkAndCount` sees `langs == nil` → `len(langs) > 0` false → `wantedLangs` stays nil → gate is noop. mage ci green covers all path-arg tests in the same run. |
| 8 | `mage ci` regression + raw `go` / `mage install` invocation | REFUTED | Re-ran `mage ci` from `main/`. 0 issues; all 7 packages green (cmd/rak, counting, fileset, ignore, lang, lister, render). BUILDER_WORKLOG.md Unit 5.4 entry shows builder ran `mage test` and `mage ci` only — no raw `go test`/`go build`/`go vet`, no `mage install`. |

### Edge-case probes (extra adversarial passes that produced no counterexample)

- **Concurrency / goroutine leaks**: `walkAndCount` is a single goroutine — no `go func`, no channels, no shared state. Map writes to `wantedLangs` happen before the for-range begins; reads inside the loop are on the local map. Safe.
- **Error swallowing**: filter `continue` is the correct non-error skip path (silent per F29). No `err := ...; _ = err` patterns introduced. `fmt.Errorf` calls in this diff use `%w` (none new in this diff — binary-check error wrap was pre-existing).
- **Nil-interface trap**: `wantedLangs` is `map[lang.Language]struct{}`, not an interface. Nil-map read returns the zero value (`struct{}{}`, ok=false) safely — but the code guards with `if wantedLangs != nil` so the read never happens on nil. Safe.
- **YAGNI pressure**: The `wantedLangs` allocation has exactly one call site, lives within a single function, and is justified by `len(langs)` linear lookup avoidance. Not premature abstraction.
- **Hidden interaction with Detect's LangUnknown fall-through**: when no `--lang` is set, every file (including LangUnknown) is counted — matches Decision 24 ("no --lang" = "count all"). When `--lang go` is set, LangUnknown is excluded — matches Decision 24 ("--lang set" = "only listed languages"). Both sides of the F29 contract preserved. Test `TestRootCmd_FlagLang_ExcludesUnknown` proves the second leg; `TestRootCmd_NoLangFlag_CountsAll` proves the first.
- **`--lang go,go` duplicate**: would insert `wantedLangs["go"]` twice via map-set idempotence — no error, no double-counting, just a one-key map. Safe.
- **Whitespace in CSV value (`--lang " go "`)**: builder does NOT trim whitespace; `strings.ToLower(" go ")` = `" go "` which won't match `LangGo = "go"`. User-visible papercut, not a correctness defect (the spec doesn't mandate whitespace tolerance). Routed to Unknowns as a future polish.

### Hidden-attack residual

The two surviving Unknowns (Attack 3 `--lang ""` defensive comment, Attack 4 sham CSV test) are both proof-side gaps, not correctness counterexamples. Each is annotated above with the routing rationale.

### Hylla Feedback

N/A — review attacked uncommitted-relative-to-last-ingest `cmd/rak/root.go` + `cmd/rak/root_test.go`. Evidence came from `git diff HEAD~1`, `Read` on the two files, Context7 pflag docs for `StringSlice` semantics, `go doc -src github.com/spf13/pflag StringSliceVar` for the StringSliceVar wrapper source, and `mage ci` re-run. Hylla was not the right tool for under-review uncommitted cmd-package code. One ergonomic note: attempting `Read` on `/Users/evanschultz/go/pkg/mod/github.com/spf13/pflag@v1.0.10/string_slice.go` was permission-denied, so the empty-string CSV behavior (Attack 3) could not be ground-truthed against pflag source — Hylla doesn't index dep modules either, so this remains a vendor-source-readability gap orthogonal to Hylla.

### Certificate

- **Premises:** Unit 5.4 must (a) add `--lang` cobra flag via `StringSliceVar` accepting comma-separated values, (b) plumb `flags.langs` through `runRoot` → `runDirectory` → `walkAndCount`, (c) build `wantedLangs` set once before the per-file loop with `strings.ToLower` normalization, (d) gate `countFile` on `wantedLangs` membership AFTER `lang.Detect` and BEFORE `countFile`, (e) silently exclude `LangUnknown` files when any filter is set (F29, Decision 24), (f) preserve no-filter behavior when `--lang` is omitted, (g) keep all changes inside `cmd/rak/`, (h) pass `mage ci`.
- **Evidence:** `git diff HEAD~1 -- cmd/rak/` confirms +42/-7 in root.go and +120/-1 in root_test.go; `git diff HEAD~1 -- internal/lister/ internal/fileset/ internal/ignore/ internal/render/` empty (Drop 4 spine intact); `Read` on root.go:220–305 (walkAndCount full body) confirms filter ordering and zero-alloc nil-langs path; `Read` on root_test.go:484–601 confirms five filter-behavior tests + one CSV-named test (which is a sham, see Attack 4); Context7 pflag docs confirm comma-split semantics; `mage ci` green on all 7 packages; BUILDER_WORKLOG.md Unit 5.4 entry confirms only `mage test` + `mage ci` were run.
- **Trace or cases:** 8 attack families + 7 edge-case probes = 15 distinct adversarial probes. All REFUTED, EXHAUSTED-no-counterexample, or accepted-as-documented-YAGNI/Unknown. Zero CONFIRMED.
- **Conclusion:** PASS. Unit 5.4 survives Round 1 falsification with no unmitigated counterexample. The critical F29 / Decision 24 invariant (LangUnknown excluded when filter set) is enforced by the map-lookup logic itself (`Language("")` is not in `wantedLangs` because no `--lang` value can canonicalize to `""` under normal pflag CSV semantics). Filter ordering, case normalization, no-filter noop, and Drop 4 surface preservation all hold.
- **Unknowns:** (i) Pflag's exact behavior for `--lang ""` (whether it produces empty slice or `[""]`) could not be ground-truthed against gated vendor source — defensive `strings.TrimSpace` skip in the build loop would harden against the unlikely `[""]` case but is YAGNI absent a contradicting test; (ii) `TestRootCmd_LangFlag_ParsesCSV` (line 587) does NOT actually exercise cobra's CSV split — end-to-end `cmd.SetArgs([]string{"--lang", "go,rust"})` is unexercised; (iii) `--lang " go "` (whitespace) silently drops; user-visible papercut. All three routed to dev as future-drop test-widening / defensive-polish opportunities, NOT a Round-1 gate.
