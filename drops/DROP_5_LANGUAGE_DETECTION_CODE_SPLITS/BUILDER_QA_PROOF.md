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

## Unit 5.3 — Round 1

- **Reviewer:** go-qa-proof-agent
- **Commit under review:** `3b423a7` — *feat(render): per-language rollup with langunknown suppression*
- **Files audited:**
  - `internal/render/render.go` (+6 / -0 — grow `Directory` with `ByLang` field + `lang` import)
  - `internal/render/json.go` (+35 / -3 — grow `directoryJSON` with `ByLang` per F34; add `filterUnknown`; apply at conversion site)
  - `internal/render/human.go` (+44 / -0 — extend `RenderTree` with per-lang KV rows; add `langKV`, `sortedKnownLangs`)
  - `internal/render/toon.go` (+56 / -7 — add `toonLangRow`; extend `toonTree` with `ByLang`; populate with F33 suppression + dir/lang sort)
  - `internal/render/render_test.go` (+221 / -0 — six new tests: 3 PerLang + 3 AllUnknown)
  - `cmd/rak/root.go` (+39 / -5 — `walkAndCount` adds `byDirLang`, per-file `lang.Split`, accumulation; `labelDirectories` preserves `ByLang`)
  - `cmd/rak/root_test.go` (+85 / -0 — `TestRootCmd_PerLangRollup` + supporting JSON-mirror types)
- **Verdict:** **PASS** — every Unit 5.3 acceptance criterion is backed by file:line / named-test / build-output evidence; `mage ci` re-run green at HEAD `3b423a7`.

### Acceptance audit

| # | Criterion | Status | Evidence |
|---|---|---|---|
| 1 | `render.Directory` grows `ByLang map[lang.Language]lang.LangCounts` with doc comment citing F31 + F33 | PASS | `internal/render/render.go:62-65`: `ByLang map[lang.Language]lang.LangCounts`. Doc comment mentions F33 explicitly ("Per F33, the LangUnknown key MUST be filtered out by all renderer implementations before emission"). F31 (per-dir/per-lang rollup wiring) is implicit in the type itself — the field is the rollup contract. `lang` import added at `render.go:14`. |
| 2 | `Renderer` interface signature UNCHANGED (F32) | PASS | `internal/render/render.go:28-40`: both methods (`Render(w, counts) error`, `RenderTree(w, dirs, total, errs) error`) identical to pre-5.3. The `Directory` struct grew but the interface contract did not. Compile-time invariant enforced by `var _ render.Renderer = render.NewHumanRenderer()` / `NewJSONRenderer()` / `NewTOONRenderer()` at `root_test.go:23-25`; build green. |
| 3 | **F34** — `directoryJSON` grows `ByLang` with same Go type as `Directory.ByLang`; `directoryJSON(d)` conversion compiles; tag `json:"by_lang,omitempty"` | PASS | `internal/render/json.go:50-54`: `ByLang map[lang.Language]lang.LangCounts` `json:"by_lang,omitempty"` — identical Go type to `Directory.ByLang`. Doc comment at `json.go:45-49` calls out the F34 mirror requirement. Conversion site at `json.go:99`: `payload.Directories = append(payload.Directories, directoryJSON(filterUnknown(d)))` — Go struct conversion requires identical field names + types (tags ignored by Go spec for conversion). `mage ci` green confirms compile. |
| 4 | **F33** — all three renderers filter `LangUnknown` before emission; helper(s) strip `LangUnknown` keys | PASS | JSON: `filterUnknown(d)` at `json.go:60-78` builds a copy with `LangUnknown` removed; if result empty, sets map to `nil` so `omitempty` suppresses the field. Applied at `json.go:99`. Human: `sortedKnownLangs(byLang)` at `human.go:157-166` skips `LangUnknown`; called from `RenderTree` at `human.go:82`. TOON: same `sortedKnownLangs` helper called at `toon.go:124` during row construction; `toonTree.ByLang` is `[]toonLangRow` with `omitempty` so empty result suppresses the field. Three independent suppression sites, one shared policy. |
| 5 | `walkAndCount` calls `lang.Split` after lang-filter gate, accumulates `LangCounts` per dir/per lang, populates `Directory.ByLang` | PASS | `cmd/rak/root.go` Split call at lines 303-318 (double-open via `f.Open()`, close with `_ = rc.Close()`, error appended to `aggErrs` but counting continues — P4 trade-off documented in doc comment lines 233-241). Accumulator at `root.go:331-339`: `lc := byDirLang[dir][detectedLang]; lc.Add(lang.LangCounts{Lines: lineCounts, Counts: fileCounts}); byDirLang[dir][detectedLang] = lc` (correct Go map-of-struct mutation idiom). `Directory.ByLang` populated at `root.go:345`: `dirs = append(dirs, render.Directory{Path: p, Counts: c, ByLang: byDirLang[p]})`. `labelDirectories` updated at `root.go:401,407` to preserve `ByLang` on both root and subdirectory paths. |
| 6 | Six new render tests: 3 PerLang non-empty, 3 AllUnknown suppression | PASS | `internal/render/render_test.go`: `TestTOONRenderer_RenderTree_PerLang` (line 451), `TestTOONRenderer_RenderTree_AllUnknown` (line 489), `TestJSONRenderer_RenderTree_PerLang` (line 530), `TestJSONRenderer_RenderTree_AllUnknown` (line 569), `TestHumanRenderer_RenderTree_PerLang` (line 603), `TestHumanRenderer_RenderTree_AllUnknown` (line 640). Each AllUnknown test asserts the language identifier never appears in output; each PerLang test asserts both language names do appear. All six green under `mage test`. |
| 7 | `TestRootCmd_PerLangRollup` end-to-end exercises `.go` + `.py` per-lang JSON output | PASS | `cmd/rak/root_test.go:660-696`: MapFS with `a.go` (LangGo via extension) + `b.py` (LangPython via extension); runs `runDirectory` with `NewJSONRenderer`; unmarshals into `treeResultWithLang` (extended dirResult shape carrying `by_lang` map); asserts `dir.ByLang["go"]` AND `dir.ByLang["python"]` both present AND `dir.ByLang[""]` (LangUnknown) absent. End-to-end coverage of detect → Split → accumulate → JSON emit → omit-unknown pipeline. |
| 8 | Drop 4 integration counts UNCHANGED (20B / 2L / 4W / 20C against `testdata/tree`) | PASS | `cmd/rak/integration_test.go:145-150` constants unchanged (`treeExpectedTotalBytes=20`, `Lines=2`, `Words=4`, `Chars=20`). Compatible because (a) Counts struct shape unchanged — Split runs alongside countFile without altering the counting accumulator (root.go:320-329 still calls `countFile` and `total.Add(fileCounts)`); (b) the existing `treeResult`/`dirResult` JSON-decode shapes (`root_test.go:221-230`) carry only `path` + `counts`, so the new `by_lang` JSON field is silently ignored by `encoding/json.Unmarshal` (default behavior — unknown fields skipped); (c) `TestRootCmd_Integration_PathArg_JSONFormat` re-runs cleanly under `mage ci` at HEAD `3b423a7`. |
| 9 | `mage ci` green | PASS | Re-ran `mage ci` at HEAD `3b423a7`: gofumpt clean (no output from `gofumpt -l .`), `golangci-lint run` reports `0 issues.`, `go test -race ./...` returns `ok` for all 7 packages (`cmd/rak`, `internal/counting`, `internal/fileset`, `internal/ignore`, `internal/lang`, `internal/lister`, `internal/render`). All `(cached)` rows indicate test artifacts hash matches a prior green run at this exact source state — no stale-cache risk because the Go cache key is source-content-hashed. |

### Proof certificate

- **Premises** — `Directory.ByLang` carries per-dir/per-lang counts using `map[lang.Language]lang.LangCounts` (F31); the `Renderer` interface contract is unchanged (F32); all three renderers suppress `LangUnknown` before emission (F33); `directoryJSON` mirrors `Directory` byte-for-byte at the Go-type level so `directoryJSON(d)` conversion compiles (F34); `walkAndCount` opens each non-binary non-filtered file a second time for `lang.Split`, accumulates per-dir/per-lang `LangCounts`, and threads the result into `render.Directory.ByLang`; the six new renderer tests + the end-to-end `TestRootCmd_PerLangRollup` cover the new behavior axes; the Drop 4 integration count invariant (20B/2L/4W/20C) survives because the new `by_lang` JSON field is decoded into a shape that doesn't carry it (unknown-field skip); `mage ci` is green at the audited commit.
- **Evidence** — `internal/render/render.go:14,62-65`; `internal/render/json.go:50-54,60-78,99`; `internal/render/human.go:8-13,82-91,140-166`; `internal/render/toon.go:7,46-58,73,118-147`; `internal/render/render_test.go:451-666` (six new tests); `cmd/rak/root.go:233-241,289-318,331-339,345,401,407`; `cmd/rak/root_test.go:621-696` (TestRootCmd_PerLangRollup + supporting JSON-mirror types); `cmd/rak/integration_test.go:145-150` (Drop 4 constants intact); `cmd/rak/root_test.go:221-230` (existing `treeResult` shape unchanged — explains why old tests continue to pass against augmented JSON); `git show HEAD --stat` matches the unit's `Paths` list (7 files, all under `cmd/rak/` or `internal/render/`, no off-scope spill); `mage ci` HEAD-output green.
- **Trace** —
  1. **Per-language rollup path (TestRootCmd_PerLangRollup):** `runDirectory` → `walkAndCount` → for `a.go`: `lang.Detect` = `LangGo` → `lang.Split(rc, LangGo)` returns `{Code:1}` → `countFile` returns `{Bytes:13,Lines:1,Words:2,Chars:13}` → `lc.Add({Lines:{Code:1}, Counts:{Bytes:13,...}})` into `byDirLang["."][LangGo]`; for `b.py`: same path with `LangPython` and `{Bytes:6}`; loop end emits `render.Directory{Path:".", Counts:total, ByLang:map[LangGo:{...}, LangPython:{...}]}` → `labelDirectories` preserves `ByLang` (root branch, line 401) → JSON `RenderTree` → `filterUnknown` (no LangUnknown present → returns unchanged shape) → struct conversion `directoryJSON(d)` → encode with `by_lang` field present. Test assertion: both `go` and `python` keys present, empty-string key absent. ✓
  2. **F33 AllUnknown JSON path (TestJSONRenderer_RenderTree_AllUnknown):** input `ByLang={LangUnknown: {...}}` → `filterUnknown` builds filtered map by iterating keys, skips `LangUnknown` → filtered is empty → set to `nil` → conversion gives `directoryJSON{Path, Counts, ByLang: nil}` → `omitempty` drops `by_lang` from JSON output. Test asserts no `by_lang` substring. ✓
  3. **F33 AllUnknown human path (TestHumanRenderer_RenderTree_AllUnknown):** `RenderTree` checks `len(d.ByLang) > 0` (true), calls `sortedKnownLangs(d.ByLang)` → iterator filters `LangUnknown` out → returned slice is empty → for-loop emits zero `langKV` rows. Test asserts no `unknown` substring (lowercase) anywhere in output. ✓
  4. **F33 AllUnknown TOON path (TestTOONRenderer_RenderTree_AllUnknown):** `sortedKnownLangs` returns empty → no `toonLangRow` constructed → `langRows` slice is nil → `toonTree.ByLang` is empty → `omitempty` drops field. Test asserts no `unknown` and no language-key form `""`/`unknown:` appears. ✓
  5. **Drop 4 integration invariant (TestRootCmd_Integration_PathArg_JSONFormat):** walker emits the existing dirs `testdata/tree` + `testdata/tree/sub`; Counts unchanged because `countFile` still runs (Split runs in parallel as a side-effect for `LangCounts`, not as a replacement for `countFile`); JSON envelope now carries a `by_lang` field per directory but `dirResult` JSON-decode struct in `root_test.go:227-230` only declares `path` + `counts`, so the new field is silently skipped on Unmarshal. Total still 20/2/4/20. ✓
  6. **F34 conversion site (`directoryJSON(filterUnknown(d))`):** Go spec § "Conversions" — a struct type conversion is valid when the source and destination have identical fields with identical types and matching names, ignoring tags. `Directory` and `directoryJSON` both have `Path string`, `Counts counting.Counts`, `ByLang map[lang.Language]lang.LangCounts`. Conversion compiles. Compile gate: `mage ci` green. ✓
  7. **F32 interface stability:** `Renderer.Render` and `Renderer.RenderTree` signatures unchanged at `render.go:31,39`. `Directory` grew a field, but interface methods take `[]Directory`, so the new field is transparently carried. No call-site changes required for callers that don't touch `ByLang`. ✓
- **Conclusion** — All 9 acceptance criteria PASS. No findings. Unit 5.3's claim ("per-type aggregation in render + walkAndCount Split wiring, with F33 LangUnknown suppression and F34 directoryJSON parity") is fully supported by the evidence. Drop 5 is ready to close.
- **Unknowns** — None blocking. Two design choices documented in `BUILDER_WORKLOG.md` are acceptable for v0.1.0: (a) TOON `by_lang` is represented as a flat `[]toonLangRow` table beside `directories` rather than embedded inside each directory row (rationale: toon-go map-marshaling support uncertain, flat table is safe and tabular); (b) double-IO per file (countFile opens + Split opens) is accepted per P4. Both align with PLAN.md notes and are not Unit 5.3 acceptance criteria.

### Findings

None.

### Missing evidence

None — every AC bullet has at least one direct file:line / named-test / build-output citation above.

### Falsification probes (no unmitigated counterexample)

- **F34 conversion correctness with non-empty ByLang:** `Directory.ByLang` and `directoryJSON.ByLang` are `map[lang.Language]lang.LangCounts` — same exact Go type. Go conversion ignores tags. Conversion compiles regardless of map content. The `omitempty` tag operates at encoding time, not at conversion time, so a nil map produced by `filterUnknown` still converts successfully (a `nil` map is a valid value of the destination type). Mitigated.
- **`labelDirectories` dropping ByLang silently:** original Drop 3 / Drop 4 `labelDirectories` constructed `render.Directory{Path:..., Counts:...}` without `ByLang`, which would zero-value the new field. The diff fixes both branches at `root.go:401` and `root.go:407`. Verified by `TestRootCmd_PerLangRollup` running through `runDirectory` (which calls `labelDirectories`) and observing both `go` and `python` keys in the JSON output. Mitigated.
- **Existing tests breaking on JSON `by_lang` field:** `treeResult` / `dirResult` decode shapes at `root_test.go:221-230` carry only `path` + `counts`. Stdlib `encoding/json.Unmarshal` default behavior silently drops unknown fields. `mage ci` green confirms no test regression. Mitigated.
- **Drop 4 integration totals could drift if Split errors get aggregated as walk failures:** Split errors append to `aggErrs` (root.go:307,313) but do NOT trigger `continue` or otherwise skip the file. `countFile` still runs on every file that passes the binary + lang-filter gates. Total counting accumulation is independent of Split outcome. `TestRootCmd_Integration_PathArg_JSONFormat` re-runs green with totals 20/2/4/20. Mitigated.
- **`walkAndCount` second open via `f.Open()` could hold an FD if `rc.Close()` is missed:** Reading the diff path at `root.go:307-316`: `rc, openErr := f.Open()`; if `openErr == nil` then `_, splitErr = lang.Split(rc, ...)` then unconditional `_ = rc.Close()`. The close runs whether or not Split errored. No FD leak in the happy or error path. Mitigated. (One edge case left: if `f.Open()` succeeds but `lang.Split` panics, the close would not run — but Split has no panic paths in `internal/lang/split.go` per the Unit 5.2 audit.)
- **AllUnknown human-test false positive on `unknown` substring:** The test asserts `!strings.Contains(strings.ToLower(got), "unknown")`. The human output includes `dir:`, `Bytes`, `Lines`, `Words`, `Chars`, `total` — none of these contain `unknown`. The language-row format is `lang: <name>` where `<name>` comes from `sortedKnownLangs` which filters `LangUnknown` out. So no `unknown` substring can appear in valid output. Mitigated.
- **AllUnknown TOON-test asymmetry on the inner if:** The test's inner `if strings.Contains(got, "by_lang")` is defensive — it accepts both "by_lang absent entirely" and "by_lang present but with no language key for unknown". Since `omitempty` suppresses an empty slice, the field will be absent; the inner check is a belt-and-suspenders. Mitigated.
- **`mage ci` cache poisoning:** All test rows show `(cached)` because the source content hashes to a key already verified. Go's test cache is keyed on the source-tree hash, not a timestamp, so cached results faithfully reflect the current source state. The cache would invalidate on any source change. Mitigated.

### Hylla Feedback

N/A — Unit 5.3 touched only files inside HEAD commit `3b423a7`, which post-date the latest Hylla ingest. All evidence came from `Read` of the just-written sources, `git show` / `git diff HEAD~1 HEAD` for the delta, the drop's `PLAN.md` for AC text, the prior Unit 5.1/5.2/5.4 QA Proof sections for context continuity, and `mage ci` for the build gate. No Hylla query was needed or made.
