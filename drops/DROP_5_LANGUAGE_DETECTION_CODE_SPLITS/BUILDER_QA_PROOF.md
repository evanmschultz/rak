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
