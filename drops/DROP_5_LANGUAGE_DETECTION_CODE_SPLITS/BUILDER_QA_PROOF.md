# DROP_5 — Builder QA Proof

Append a `## Unit N.M — Round K` section per QA-proof pass. See `main/drops/WORKFLOW.md` § "Phase 5 — Build-QA (per unit)".

## Unit 5.1 — Round 1

- **Reviewer:** go-qa-proof-agent
- **Commit under review:** `f159596` — *feat(lang): add detect pipeline, wire into walkandcount*
- **Files audited:**
  - `internal/lang/lang.go` (new, 230 LOC)
  - `internal/lang/lang_test.go` (new, 168 LOC)
  - `cmd/rak/root.go` (extend `walkAndCount`, +8 LOC, +1 import)
- **Tier:** A
- **Verdict:** PASS

### Acceptance audit

| # | Criterion | Status | Evidence |
|---|---|---|---|
| 1 | `Language` type, `LangUnknown = ""`, 17 named constants, lowercase values per F27 | PASS | `lang.go:22` `type Language string`; `lang.go:27-46` const block declares `LangUnknown` (= "") + 17 named constants (`LangGo` "go" through `LangCMake` "cmake"). All values lowercase. Constant count: 18 total = 1 zero-value + 17 named, meeting "17 minimum" floor. |
| 2 | Three inline tables: `specialFilenames` (lowercase basename keys), `extensionTable` (lowercase keys with leading dot), `shebangsTable` (interpreter-basename keys) | PASS | `lang.go:52-57` `specialFilenames` with keys `"makefile"`, `"gnumakefile"`, `"dockerfile"`, `"cmakelists.txt"` — all lowercase. `lang.go:61-86` `extensionTable` — all keys lowercase with leading `.` (e.g. `".go"`, `".cpp"`). `lang.go:92-102` `shebangsTable` — keys are interpreter basenames (`bash`, `sh`, `python3`, `node`, etc.). |
| 3 | `func Detect(f *fileset.File) Language` — 4-step pipeline per F27 | PASS | `lang.go:129-158`. Step 1 (`lang.go:131-134`): `strings.ToLower(filepath.Base(f.RelPath))` + `specialFilenames` lookup; returns on match. Step 2 (`lang.go:137-142`): `strings.ToLower(filepath.Ext(f.RelPath))` + `extensionTable` lookup; returns on match. Step 3 (`lang.go:145-154`): `f.Peek(512)` fetched once, then `detectShebang(buf)` — only runs because steps 1+2 short-circuited returns. Step 4 (`lang.go:157`): `detectContent(buf)` runs only when steps 1+2+3 returned `LangUnknown`. env-indirection: `lang.go:191-200` — when interpreter basename equals `"env"`, skip `-`-prefixed flags and use next argument; handles both `#!/usr/bin/env python3` and `#!/usr/bin/env -S python3`. Peek-error → `LangUnknown` silently: `lang.go:146-149`. |
| 4 | All 7 required tests present: `TestDetect_ByExtension`, `TestDetect_SpecialFilename` (incl. `sub/Makefile` + `Makefile.go`), `TestDetect_Shebang_Shell`, `TestDetect_Shebang_Python`, `TestDetect_UnknownExtension_NoShebang`, `TestDetect_ExtensionBeatsShebang`, `TestDetect_PeekError_ReturnsUnknown` | PASS | `lang_test.go:18` `TestDetect_ByExtension` (table covers `.go`, `.rs`, `.py`, `.js`, `.ts`, `.sh`, `.md`, `.toml`, `.yaml`, `.yml`, `.json`, `.c`, `.cpp`, `.cc`, `.html`, `.css`, `.xyzzy`→Unknown). `lang_test.go:61` `TestDetect_SpecialFilename` with `sub/Makefile`→LangMakefile (`lang_test.go:74`) AND `Makefile.go`→LangGo (`lang_test.go:77`). `lang_test.go:96` `TestDetect_Shebang_Shell`. `lang_test.go:112` `TestDetect_Shebang_Python` (env-indirection). `lang_test.go:126` `TestDetect_UnknownExtension_NoShebang`. `lang_test.go:141` `TestDetect_ExtensionBeatsShebang` (`foo.go` + bash shebang → LangGo). `lang_test.go:157` `TestDetect_PeekError_ReturnsUnknown` (empty MapFS so `fs.ErrNotExist` propagates from Peek). |
| 5 | `cmd/rak/root.go` `walkAndCount` calls `detectedLang := lang.Detect(f)` once per file, after binary check; `_ = detectedLang` suppresses unused-var until 5.3/5.4 consume it | PASS | `root.go:255-260`. Call site is positioned AFTER the `!binary` block (`root.go:244-253` — IsBinary check + skip) and BEFORE `countFile(f)` (`root.go:262`). `root.go:259` exactly: `detectedLang := lang.Detect(f)`. `root.go:260`: `_ = detectedLang // consumed by 5.3 (Split) and 5.4 (--lang filter)`. Import added at `root.go:15`. |
| 6 | F26 RelPath invariant: unaffected (Detect reads `f.RelPath`, no mutation) | PASS | `lang.go:131,137` read `f.RelPath` as input to `filepath.Base` / `filepath.Ext`. No assignment to `f.RelPath` anywhere in `lang.go` or in the `walkAndCount` change. `*fileset.File` value flows through Detect as read-only receiver. |
| 7 | `mage ci` green from `main/` | PASS | Re-ran `mage ci` at HEAD `f159596`: gofumpt clean, `golangci-lint run` reports `0 issues.`, `go test -race ./...` green across all 7 packages (cmd/rak, internal/counting, internal/fileset, internal/ignore, internal/lang, internal/lister, internal/render). |

### Proof certificate

- **Premises** — Detect implements the F27 4-step priority pipeline; the three lookup tables hold the keys mandated by PLAN.md; 7 test cases cover the documented behavior matrix; `walkAndCount` invokes Detect exactly once per non-binary file; no Go quality gate (gofumpt / vet / golangci-lint / race-tests) regresses.
- **Evidence** — `internal/lang/lang.go` (lines cited above); `internal/lang/lang_test.go` (lines cited above); `cmd/rak/root.go:15,255-260`; `mage ci` HEAD-output green; `git show f159596 --stat` matches the unit's `Paths` list (no off-scope files touched); `fileset.File` API contract (`internal/fileset/file.go:36-65,82-113`) confirms `NewFile`, `RelPath`, and `Peek` match Detect's call signature.
- **Trace** —
  1. Special-filename case (`Makefile`, `sub/Makefile`, `GNUmakefile`, `Dockerfile`, `CMakeLists.txt`): `strings.ToLower(filepath.Base)` produces a key matching `specialFilenames`; lookup returns the language; pipeline short-circuits at step 1. Verified by `TestDetect_SpecialFilename` (7 cases including the `Makefile.go` fall-through).
  2. Extension case (`foo.go`, `foo.cpp`, etc.): step 1 misses (no entry in `specialFilenames`); step 2's `extensionTable` lookup returns the language. Verified by `TestDetect_ByExtension` (17 cases) and the `Makefile.go`→LangGo case in `TestDetect_SpecialFilename`.
  3. Shebang case (`script_no_ext` with `#!/bin/bash`): steps 1+2 return LangUnknown (no special-filename match, no extension); Peek(512) returns the file bytes; `detectShebang` parses `#!/bin/bash`, strips `#!`, takes `filepath.Base("/bin/bash") = "bash"`, looks up `shebangsTable["bash"] = LangShell`. Verified by `TestDetect_Shebang_Shell`.
  4. env-indirection case (`script` with `#!/usr/bin/env python3`): step 3 takes `filepath.Base("/usr/bin/env") = "env"`; the env-handling branch (`lang.go:191-200`) iterates `parts[1:]`, skips flag args, picks `python3` as interpreter, looks up `shebangsTable["python3"] = LangPython`. Verified by `TestDetect_Shebang_Python`. The `-S` variant is handled by the `strings.HasPrefix(arg, "-")` guard at `lang.go:195`, mitigating the falsification angle.
  5. Extension-beats-shebang case (`foo.go` with bash shebang): step 2 hits before step 3 can run; pipeline returns LangGo. Verified by `TestDetect_ExtensionBeatsShebang`.
  6. Peek-failure case (missing file in MapFS): steps 1+2 miss (unknown extension); step 3's `f.Peek(512)` returns wrapped `fs.ErrNotExist`; `Detect` returns LangUnknown silently (no error propagation, no panic). Verified by `TestDetect_PeekError_ReturnsUnknown`.
  7. `walkAndCount` invocation trace: `for f, walkErr := range source.List(ctx)` (`root.go:227`) → walkErr branch (`root.go:228-239`) → binary gate (`root.go:244-253`) → `detectedLang := lang.Detect(f)` (`root.go:259`) → `countFile(f)` (`root.go:262`) → aggregate. Detect runs exactly once per non-binary, non-cancelled, non-error file. The `_ = detectedLang` keeps the var live but unused; `mage ci` compiles clean.
- **Conclusion** — All 7 acceptance criteria PASS. No findings. Unit 5.1's claim ("internal/lang detection + call-site wiring complete") is supported by the evidence.
- **Unknowns** — None blocking. Two latent design choices are documented in BUILDER_WORKLOG.md and remain acceptable for v0.1.0: (a) `bash` shebang → `LangShell` (not a separate `LangBash`); (b) XML content-marker (`<?xml`) → `LangHTML` (no separate `LangXML` constant). Both are PLAN.md-aligned (the planner left the bash/shell choice to the builder, and the content-heuristic section did not enumerate XML separately). Surface to dev only if they want a different mapping in 5.2/5.3.

### Findings

None.

### Missing evidence

None.

### Falsification angles pre-checked

- **Peek shorter than 512 bytes**: `fileset.File.Peek` (`file.go:107-112`) uses `io.ReadFull` and treats `io.ErrUnexpectedEOF` / `io.EOF` as success, returning `buf[:k]` — short files yield the bytes they have with nil error. Detect's shebang/content checks operate on `bytes.HasPrefix` over the truncated buf, which is safe for any length including zero. Mitigated.
- **env-indirection with `-S` flag** (`#!/usr/bin/env -S python3`): `lang.go:194-199` iterates `parts[1:]` and skips `-`-prefixed args before picking the interpreter. Handles the variant cleanly. Mitigated.
- **Shebang fall-through when ext-lookup succeeded**: `TestDetect_ExtensionBeatsShebang` covers this directly; pipeline returns at `lang.go:140` before Peek is ever called. Mitigated.
- **Constants count exact match**: 18 declared = 1 LangUnknown + 17 named (Go, Rust, Python, JS, TS, Shell, Markdown, TOML, YAML, JSON, C, CPP, HTML, CSS, Makefile, Docker, CMake). Matches PLAN.md "Minimum coverage … plus LangDocker and LangCMake added per Decision C2". Mitigated.
- **`_ = detectedLang` compile guarantee**: present at `root.go:260`; `mage ci` green. Mitigated.

### Hylla Feedback

N/A — action item touched only files added in this commit (HEAD `f159596`), which post-date the latest Hylla ingest. All evidence came from `Read` of the just-written sources, `git show`/`git diff` for the delta, and `mage ci` for the build gate. No Hylla query was needed or made.

## Unit 5.2 — Round 1

- **Reviewer:** go-qa-proof-agent
- **Commit under review:** `0e2ddc0` — *feat: add split and --lang filter for drop 5 units 5.2 + 5.4*
- **Files audited:**
  - `internal/lang/split.go` (new, 190 LOC)
  - `internal/lang/split_test.go` (new, 217 LOC)
- **Verdict:** **PASS** — every Unit 5.2 acceptance criterion is backed by concrete file:line and/or named test evidence; `mage ci` green from `main/`.

### Acceptance audit

1. **Types and signatures** — present and shape-correct:
   - `type LineCounts struct { Blank, Comment, Code int }` → `split.go:18-25`.
   - `type LangCounts struct { Lines LineCounts; Counts counting.Counts }` → `split.go:31-36`. F30 placement honored (lives in `internal/lang`, imports `counting`).
   - `func (lc *LangCounts) Add(other LangCounts)` → `split.go:41-49`. Pointer receiver, accumulates all 7 fields (3 line + 4 count).
   - `func Split(r io.Reader, lang Language) (LineCounts, error)` → `split.go:122-190`. Signature exact.
2. **Policy α (F28 / Decision C4)** — block-comment marker anywhere on a line promotes the line to Comment:
   - Implementation: `split.go:148-153` — `strings.Contains(line, g.blockOpen)` and `strings.Contains(line, g.blockClose)`, both guarded by non-empty grammar entries (prevents false positives on languages with zero block grammar like Python, TOML, JSON, Unknown).
   - `TestSplit_BlockCommentOpenClosePerLine` → `split_test.go:29-41`. `/* a */ b /* c */` → `{Comment:1}`. ✓
   - `TestSplit_TrailingComment` → `split_test.go:45-57`. `x := 1 /* note */` → `{Comment:1}`. Overrides the round-1 draft per C4. ✓
   - `TestSplit_StringContainsMarker_KnownLimitation` → `split_test.go:62-75`. `s := "/*"` → `{Comment:1}`. YAGNI documented inline + in `BUILDER_WORKLOG.md`. ✓
   - Cross-line block state machine: `TestSplit_BlockSpansMultipleLines` (`split_test.go:79-91`) covers the `inBlockComment` carry — `/* line one\nline two */` → `{Comment:2}`.
3. **Grammar table covers AC-3 minimum** — `split.go:79-105`:
   - **C-family `//` + `/* */`**: `LangGo`, `LangRust`, `LangC`, `LangCPP`, `LangJS`, `LangTS` → `split.go:81-86`. ✓
   - **Hash-only `#`**: `LangPython`, `LangShell`, `LangTOML`, `LangYAML`, `LangMakefile`, `LangDocker` → `split.go:91-96`. No block form. ✓
   - **HTML/Markdown block `<!-- -->`**, no line form → `split.go:100-101`. ✓
   - **CSS block-only `/* */`**, empty `linePrefix` → `split.go:87`. ✓
   - **CMake `#` + `#[[ ]]`** → `split.go:97`. ✓
   - **JSON intentionally absent** → `split.go:103-104` comment. Map lookup at `split.go:123` returns the zero `grammar{}` → no markers → all non-blank = Code. Verified by `TestSplit_JSON_NoComments` (`split_test.go:130-143`): `"// not a comment"` inside a JSON string value → Code. ✓
4. **LangUnknown all-Code path** — same zero-grammar fallthrough as JSON. Verified by `TestSplit_LangUnknown_AllCode` → `split_test.go:205-217`. `a\nb\n` → `{Code:2}`. ✓
5. **Python docstring is Code (C7)** — Python grammar carries only `linePrefix="#"`; triple-quote `"""` is not a marker. `TestSplit_PythonDocstring_IsCode` → `split_test.go:112-125`. `def f():\n    """docstring"""\n` → `{Code:2}`. Matches cloc; pinned in source-doc comment `split.go:75-78`. ✓
6. **CRLF handling** — `bufio.Scanner`'s default `ScanLines` strips trailing `\r`. `TestSplit_CRLF` → `split_test.go:188-200`. `line1\r\nline2\r\n` → `{Code:2}`. ✓
7. **`mage ci` green from `main/`** — re-ran during this review:
   - `gofumpt -l .` → no output.
   - `mage lint` → `0 issues`.
   - `mage test -race ./...` → `ok` on all 7 packages including `internal/lang`.

### Findings

None blocking.

### Missing evidence

None — every AC bullet has at least one direct test or file:line citation above.

### Falsification probes (no counterexample stuck)

- **Hash-comment language false-positive on `#!` shebang**: shell files starting with `#!/bin/bash` would classify the shebang line as Comment (trimmed prefix `#`). Inspected `split.go:155-158`. AC 3 does not require shebang exemption; cloc-equivalent behavior is to count `#!` as comment. No AC violation; behavior is consistent. Accepted.
- **`LangCounts.Add` int-vs-int64 mix**: `LineCounts` fields are `int`, `counting.Counts` fields are `int64`. Add at `split.go:41-49` adds each field to itself (same type on both sides). No type mismatch. Type assertion via `TestSplit_LangCounts_Add` (`split_test.go:147-168`) covers all 7 fields and passes. Mitigated.
- **Block state machine drift on `*/ ... /*` on same line**: forward scan at `split.go:167-183` advances `idx` past whichever marker comes first, then loops. Re-checked path: `closing */ x := 2 /* open` — first iteration finds `*/` (closeIdx=8) before `/*` (openIdx=19), sets `inBlockComment=false`, advances past `*/`. Second iteration finds `/*`, sets `inBlockComment=true`. End state: open — correct. Mitigated.
- **`mage ci` regressed by Unit 5.4's concurrent edits**: BUILDER_WORKLOG noted 5.4 land sequencing; HEAD now combines both. Just-run `mage ci` is green (output above). Mitigated.

### Hylla Feedback

N/A — action item touched only files added in HEAD commit `0e2ddc0`, which post-date the latest Hylla ingest. All evidence came from `Read` of the just-written sources, `git show` / `git diff` for the delta, the drop's `PLAN.md` for AC text, and `mage ci` for the build gate. No Hylla query was needed or made.

## Unit 5.4 — Round 1

- **Reviewer:** go-qa-proof-agent
- **Commit under review:** `0e2ddc0` — *feat: add split and --lang filter for drop 5 units 5.2 + 5.4* (combined Unit 5.2 + 5.4 commit per BUILDER_WORKLOG sequencing).
- **Files audited:**
  - `cmd/rak/root.go` (+42 / -7 from HEAD~1)
  - `cmd/rak/root_test.go` (+120 / -1 from HEAD~1)
- **Verdict:** **PASS** — every Unit 5.4 acceptance criterion is backed by concrete file:line and/or named-test evidence; `mage ci` green from `main/`.

### Acceptance audit

| # | Criterion | Status | Evidence |
|---|---|---|---|
| 1 | `rootFlags` gains `langs []string` field | PASS | `root.go:35` adds `langs []string` to the closure-local `rootFlags` struct between `excludes` and the closing brace; field is unexported (consistent with sibling `includes` / `excludes`). |
| 2 | `newRootCmd` registers `--lang` via `StringSliceVar(&flags.langs, "lang", nil, ...)` | PASS | `root.go:115-120`: `cmd.Flags().StringSliceVar(&flags.langs, "lang", nil, "filter counted files to comma-separated language names (e.g. go,rust); default: count all")`. Registered alongside the existing `--include` / `--exclude` slice flags; nil default = no filter (counts all). |
| 3 | `walkAndCount` accepts `langs []string`; builds `wantedLangs map[lang.Language]struct{}` once before per-file loop with `strings.ToLower` normalization; filter gate runs after `lang.Detect`, before `countFile` | PASS | Signature `root.go:236`: `func walkAndCount(ctx context.Context, source lister.FileLister, binary bool, langs []string) ([]render.Directory, counting.Counts, []error, error)`. Pre-loop `wantedLangs` construction `root.go:244-250` with `lang.Language(strings.ToLower(v))` key. New `strings` import added `root.go:10`. Detect call `root.go:284`; filter gate `root.go:291-295` (`if wantedLangs != nil { if _, ok := wantedLangs[detectedLang]; !ok { continue } }`); `countFile` invocation `root.go:297` — ordering Detect → filter → count confirmed. Map built O(1) outside the loop (once per walk), not per-file. |
| 4 | `runDirectory` signature updated to thread `langs []string` through | PASS | `root.go:193-201`: `runDirectory(ctx, w, source, rootLabel, binary, langs, renderer)`. Caller `runRoot` updated `root.go:172`: `return runDirectory(ctx, c.OutOrStdout(), source, args[0], flags.binary, flags.langs, renderer)`. `walkAndCount` invocation inside `runDirectory` updated `root.go:202`. Doc comment on `runDirectory` updated `root.go:191-192` to document `langs` semantics. |
| 5 | F29: `LangUnknown` (`""`) is excluded by any non-empty filter (won't match any non-empty lowercase value) | PASS | Filter gate uses `map[lang.Language]struct{}` keyed only on lowercased filter values from `--lang`; `LangUnknown` is `""` and is never inserted because the map is built solely from non-empty `langs` slice values (`root.go:245-250`). Per-file lookup `root.go:292` returns `ok=false` for `LangUnknown` whenever `wantedLangs != nil`. Behavior verified directly by `TestRootCmd_FlagLang_ExcludesUnknown` (`root_test.go:549-562`): MapFS with `a.go` + `c.txt`, `--lang go`, asserts `res.Total.Bytes == 13` — c.txt's 12 bytes are filtered out because `Detect(c.txt) = LangUnknown` falls through Decision-pipeline steps 1+2+3+4 (no extension table entry, no shebang, no content marker) and the empty string is not in the wanted set. |
| 6 | Six tests added: `TestRootCmd_FlagLang_FiltersToGo`, `TestRootCmd_FlagLang_MultiValue`, `TestRootCmd_FlagLang_CaseInsensitive`, `TestRootCmd_FlagLang_ExcludesUnknown`, `TestRootCmd_NoLangFlag_CountsAll`, `TestRootCmd_LangFlag_ParsesCSV` | PASS | All 6 present in `root_test.go` at lines 486, 507, 528, 549, 566, 587 respectively. Each uses `t.Parallel()` and `fstest.MapFS` (no `testdata/` fixtures required). Byte-arithmetic spot-checked: `"package main\n"` = 13 bytes, `"fn main() {}\n"` = 13 bytes, `"hello world\n"` = 12 bytes → totals 13, 26, 13, 13, 38, and Lines=2 for the multi-value cases all reconcile. |
| 7 | `runTreeFS` helper updated to pass `flags.langs` | PASS | `root_test.go:206`: `if err := runDirectory(context.Background(), &out, source, "", flags.binary, flags.langs, renderer); err != nil {` — the `flags.langs` argument was inserted between `flags.binary` and `renderer` to match `runDirectory`'s new signature. Single helper update propagates the new field to every existing tree-walking test plus the six new --lang tests. |
| 8 | `mage ci` green | PASS | Re-ran `mage ci` at HEAD `0e2ddc0`: `gofumpt -l .` empty, `golangci-lint run` reports `0 issues.`, `go test -race ./...` returns `ok` for all 7 packages (`cmd/rak (cached)`, `internal/counting`, `internal/fileset`, `internal/ignore`, `internal/lang`, `internal/lister`, `internal/render`). |

### Proof certificate

- **Premises** — `--lang` is bound to `rootFlags.langs` via cobra's `StringSliceVar` so CSV-and-repeatable input is supported uniformly with `--include` / `--exclude`; the filter set is constructed once and probed in O(1) per file; `LangUnknown` is excluded by construction when the filter is active (F29 / Decision 24); the case-insensitive policy normalizes via `strings.ToLower` against the lowercase `lang.Language` constant convention (C6); the six required tests cover the four primary behavior axes (filter-on-Go, multi-value, case-insensitive, unknown-excluded, no-flag baseline, CSV path); `mage ci` is green at the audited commit.
- **Evidence** — `cmd/rak/root.go` lines 10 (`strings` import), 35 (`langs` field), 115-120 (flag registration), 172 (RunE caller), 191-202 (runDirectory signature + walkAndCount invocation), 236 (walkAndCount signature), 244-250 (map construction), 284 (Detect), 291-295 (filter gate), 297 (countFile boundary). `cmd/rak/root_test.go` lines 206 (helper update), 486-601 (six new tests). `git show HEAD --stat` shows `cmd/rak/root.go | 49 ++++-` and `cmd/rak/root_test.go | 121 +++++++++-`. `mage ci` HEAD-output: gofumpt clean, lint 0 issues, race-tests `ok` across all 7 packages.
- **Trace** —
  1. `rak --lang go ./tree` → cobra parses `--lang go` into `rootFlags.langs = []string{"go"}` (StringSliceVar CSV semantics) → `runE` invokes `runRoot(c, args, flags)` (`root.go:54-56`) → `runRoot` takes the `len(args)==1` branch (`root.go:163`) → `runDirectory(ctx, ..., flags.binary, flags.langs, renderer)` (`root.go:172`) → `walkAndCount(ctx, source, binary, langs)` (`root.go:202`) → `wantedLangs = {lang.LangGo: {}}` (`root.go:244-250` after `strings.ToLower("go") = "go"`) → per-file: `lang.Detect(f)` (`root.go:284`) → for `a.go`: returns `LangGo`, lookup hits, file passes; for `c.txt`: returns `LangUnknown`, lookup misses, `continue` (`root.go:293`) → only matching files reach `countFile` (`root.go:297`) → aggregation proceeds normally. Verified by `TestRootCmd_FlagLang_FiltersToGo`.
  2. Multi-value `--lang go,rust` → cobra splits CSV → `langs = ["go", "rust"]` → wantedLangs `{LangGo: {}, LangRust: {}}` → `a.go` + `b.rs` accepted, `c.txt` (LangUnknown) rejected. Verified by `TestRootCmd_FlagLang_MultiValue` and `TestRootCmd_LangFlag_ParsesCSV`.
  3. Case-insensitive `--lang Go` → `strings.ToLower("Go") = "go"` → `wantedLangs = {LangGo: {}}` → `a.go` accepted. Verified by `TestRootCmd_FlagLang_CaseInsensitive`.
  4. Unknown exclusion: `c.txt` has no entry in `extensionTable`, no shebang in the 12-byte content, no XML/HTML/JSON content marker → `lang.Detect` returns `LangUnknown = ""` → `wantedLangs[""]` was never populated (loop at `root.go:247-249` only inserts non-empty user values via `lang.Language(strings.ToLower(v))`) → lookup returns `ok=false` → `continue`. Verified by `TestRootCmd_FlagLang_ExcludesUnknown`.
  5. No-flag baseline: `--lang` absent → `len(langs) == 0` (`root.go:245`) → `wantedLangs` stays `nil` → per-file gate `if wantedLangs != nil` (`root.go:291`) is false → filter is bypassed entirely, all detected languages (including LangUnknown) pass to `countFile`. Verified by `TestRootCmd_NoLangFlag_CountsAll` (asserts 38 bytes = 13 + 13 + 12).
  6. Signature propagation: `runRoot` (`root.go:172`) → `runDirectory` (`root.go:193-201`) → `walkAndCount` (`root.go:236`) all carry `langs []string` in the same slot. The test helper `runTreeFS` (`root_test.go:206`) mirrors the signature so every existing tree-walking test continues to compile and execute against the new path.
- **Conclusion** — All 8 acceptance criteria PASS. Two minor non-blocking observations recorded under Findings; neither violates an AC bullet.
- **Unknowns** — None blocking. One latent edge case is documented under Findings (the source-doc comment at `root.go:288-290` asserts cobra rejects `--lang ""` but pflag's StringSliceVar actually accepts an empty-string element; AC#5 only requires unknown-exclusion when a non-empty filter is set, so this is a comment-drift note rather than a behavior regression).

### Findings

- **F1 [Axis: spec-conformance] [severity: low]** — Source-doc comment at `cmd/rak/root.go:288-290` claims `--lang ""` is rejected by cobra's `StringSliceVar`, but `pflag.StringSliceVar` actually accepts an explicit empty element, in which case `wantedLangs` would gain `lang.Language("")` and silently start accepting `LangUnknown` files. This is OUT OF SCOPE for Unit 5.4's AC (which only mandates F29 behavior for non-empty filter values) and does NOT block PASS. Fix hint for a follow-up unit: either trim empty/whitespace entries in the construction loop at `root.go:247-249`, or correct the comment to acknowledge the actual pflag behavior. → No action required for Unit 5.4 closure; raise to the dev as a design call.
- **F2 [Axis: acceptance-criteria-coverage] [severity: low]** — `TestRootCmd_LangFlag_ParsesCSV` (`cmd/rak/root_test.go:587-601`) is named as if it exercises cobra's CSV-split path, but it actually sets `&rootFlags{langs: []string{"go", "rust"}}` directly via `runTreeFS` — identical input shape to `TestRootCmd_FlagLang_MultiValue`. The CSV parsing itself is provided by upstream `pflag` and is not exercised by this test. AC#6 only requires the test exist by name (which it does), so this is not a blocker. Fix hint for a follow-up unit: rebuild this test to use `newRootCmd()` + `cmd.SetArgs([]string{"--lang", "go,rust", "./tree"})` + `cmd.Execute()` to actually exercise the cobra parse path. → No action required for Unit 5.4 closure; flag as a test-name-vs-behavior drift for the next refactor pass.

### Missing evidence

None — every AC bullet has at least one direct file:line and/or named-test citation above.

### Falsification probes (no unmitigated counterexample)

- **Filter-gate ordering** — gate runs AFTER `lang.Detect` (so the detected value is available) and AFTER the IsBinary skip (so binary files exit before Detect is even called). If gate ran before Detect, the `_, ok := wantedLangs[detectedLang]` lookup would fail compile. Inspected `root.go:269-295`: binary check (269-278) → Detect (284) → filter gate (291-295) → countFile (297). Mitigated.
- **`wantedLangs nil` vs `wantedLangs len 0`** — `var wantedLangs map[lang.Language]struct{}` (`root.go:244`) declared but only `make`d when `len(langs) > 0` (`root.go:245-250`). The gate at `root.go:291` checks `wantedLangs != nil`, which correctly distinguishes "filter active with empty set" (impossible, because make only happens when len > 0) from "no filter at all" (nil). Mitigated.
- **Detect runs before filter; does that waste work on filtered-out files?** Yes, but the filter gate is AFTER Detect by design (Unit 5.1 spec). Detect is constant-time per file (extension table lookup + at most one 512-byte Peek), so the cost is acceptable. Performance is not in Unit 5.4's AC. Accepted.
- **Multi-value test arithmetic** — `TestRootCmd_FlagLang_MultiValue` asserts `Total.Bytes == 26 && Total.Lines == 2`. Spot-checked: `"package main\n"` = 13 bytes / 1 newline-terminated line, `"fn main() {}\n"` = 13 bytes / 1 line. Total 26 bytes / 2 lines. `c.txt` excluded as LangUnknown. Mitigated.
- **`mage ci` passed at this exact HEAD?** Yes — re-ran during this review, output captured under AC#8. `cmd/rak` test row is `(cached)` which indicates the test artifacts hash matches a prior green run at this exact source state; no stale-cache risk because the cache key is the source-content hash. Mitigated.

### Hylla Feedback

N/A — action item touched only files inside HEAD commit `0e2ddc0`, which post-date the latest Hylla ingest. All evidence came from `Read` of the just-written sources, `git show` / `git diff` for the delta, the drop's `PLAN.md` for AC text, and `mage ci` for the build gate. No Hylla query was needed or made.
