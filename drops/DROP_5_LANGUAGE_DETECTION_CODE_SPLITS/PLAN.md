# DROP_5 — LANGUAGE_DETECTION_CODE_SPLITS

**State:** planning
**Tier:** A
**Blocked by:** DROP_4
**Paths (expected):** `main/internal/lang/` (new package — `Language` type, `Detect`, blank/comment/code splitter + tests), `main/internal/render/render.go` (per-type rollup data shape; F25-aware — interface may grow), `main/internal/render/toon.go` / `main/internal/render/human.go` / `main/internal/render/json.go` (extend to render per-type aggregation), `main/internal/render/render_test.go` (extend snapshot/contains tests), `main/cmd/rak/root.go` (wire language detection into per-file counting + per-type aggregation + add `--lang` walk-filter flag), `main/cmd/rak/root_test.go` (flag-parsing + per-type tests), `main/cmd/rak/integration_test.go` (extend fixture or expectations for per-type rollup)
**Packages (expected):** `github.com/evanmschultz/rak/internal/lang` (new), `github.com/evanmschultz/rak/internal/render`, `github.com/evanmschultz/rak/cmd/rak`
**PLAN.md ref:** main/PLAN.md → `DROP_5_LANGUAGE_DETECTION_CODE_SPLITS` row
**Workflow:** main/drops/WORKFLOW.md
**Started:** 2026-05-15
**Closed:** —

## Scope

Add language awareness to rak's counting. Detect each file's language via (a) extension lookup, (b) shebang sniff using the existing `fileset.File.Peek(512)` contract (F4 from Drop 3), and (c) a small content-heuristic fallback for files whose extension is ambiguous or absent. Per detected language, split each file's lines into three categories — **blank**, **comment**, **code** — using language-specific comment grammar (line-comment markers, block-comment delimiters). Aggregate per-type counts in addition to the existing per-directory rollup; surface both in all three renderers (TOON, human, JSON). Add the `--lang go,rs` walk-filter flag (per main/PLAN.md decision 24) so users can scope counting to one or more detected languages.

Drop 4's spine is preserved: `internal/lister`, `internal/fileset`, `internal/ignore`, `internal/counting`, `internal/render`'s `Renderer` interface (subject to a possible additive growth — planner decides), and the `--human` / `--json` / `--toon` flag surface all remain. Drop 5's new code is additive. Expected decomposition: 4 atomic units (5.1 internal/lang detection / 5.2 code-aware splits / 5.3 per-type aggregation in render / 5.4 `--lang` walk filter). Per the `feedback_parallelize_aggressively` memory rule, 5.2 and 5.4 are eligible to run in parallel after 5.1 closes (both consume `Language` but neither blocks the other).

`--as <lang>` (stream-type assertion for stdin) is cut per decision 30; only `--lang` (walk filter) is added in Drop 5.

## Planner

Four atomic units in a strict linear chain: 5.1 → 5.2 → 5.3 → 5.4. No parallelism is possible because each unit depends on types or files produced by its predecessor. 5.1 and 5.2 both live in `internal/lang` (same package — serialized by rule). 5.3 and 5.4 both touch `cmd/rak/root.go` and `cmd/rak/root_test.go` (shared paths — serialized by rule).

### Unit 5.1 — internal/lang: Language type + detection (extension, shebang, content heuristic)

- **State:** todo
- **Paths:**
  - `main/internal/lang/lang.go` (new file)
  - `main/internal/lang/lang_test.go` (new file)
- **Packages:** `github.com/evanmschultz/rak/internal/lang` (new package)
- **Acceptance:**
  - `lang.go` defines the following (all new, not yet in tree):
    - `type Language string` — named string type per decision 27(c).
    - `const LangUnknown Language = ""` — zero-value constant; returned by `Detect` when no rule matches (F29).
    - Named language constants for each entry in the detection table (e.g., `const LangGo Language = "go"`, `LangRust Language = "rust"`, etc.). Minimum coverage: Go, Rust, Python, JavaScript, TypeScript, C, C++, Shell (sh/bash), Markdown, TOML, YAML, JSON, Makefile, HTML, CSS. Additional entries are welcome but not required.
    - `func Detect(f *fileset.File) Language` — detection pipeline in priority order:
      1. **Extension lookup**: `strings.ToLower(filepath.Ext(f.RelPath))` → consult inline `extensionTable map[string]Language`. No error return — extension lookup is pure.
      2. **Shebang sniff** (run only when extension lookup returns `LangUnknown` OR yields a generic language): calls `f.Peek(512)` (F4 contract). If first line starts with `#!`, extract the interpreter path and consult inline `shebangsTable map[string]Language`. If `Peek` returns an error, treat as no-match and log nothing — detection failure → `LangUnknown`. (F27: `Detect` never propagates `Peek` errors; callers must not depend on them.)
      3. **Content heuristic** (run only when both above steps return `LangUnknown`): scan the first 512 bytes from `Peek(512)` for well-known markers (e.g., `<?xml`, `<!DOCTYPE`, `{`, `[` as JSON candidates; `---` as YAML front-matter). Heuristic is best-effort. If no marker matches, return `LangUnknown`.
    - The inline `extensionTable` maps lowercase extensions (without the leading dot as the key — or with dot, builder's choice, but document it) to `Language` constants. e.g., `".go"` → `LangGo`, `".rs"` → `LangRust`, etc.
    - Doc comments on every exported identifier per naming rules.
  - `lang_test.go` — table-driven, using `testing/fstest.MapFS` to construct `*fileset.File` values via `fileset.NewFile(fsys, path, path)`:
    - `TestDetect_ByExtension` — table: `.go` → `LangGo`, `.rs` → `LangRust`, `.py` → `LangPython`, `.js` → `LangJS`, `.ts` → `LangTS`, `.sh` → `LangShell`, `.md` → `LangMarkdown`, `.toml` → `LangTOML`, `.yaml`/`.yml` → `LangYAML`, `.json` → `LangJSON`, `.c` → `LangC`, `.cpp`/`.cc` → `LangCPP`, `.html` → `LangHTML`, `.css` → `LangCSS`, unknown extension → `LangUnknown`.
    - `TestDetect_Shebang_Shell` — file with no extension but content `#!/bin/bash\necho hi` → `LangShell` (or `LangBash` — builder's choice; document in BUILDER_WORKLOG.md).
    - `TestDetect_Shebang_Python` — file named `script` with `#!/usr/bin/env python3\n` → `LangPython`.
    - `TestDetect_UnknownExtension_NoShebang` — file with `.xyzzy` extension and no shebang → `LangUnknown`.
    - `TestDetect_ExtensionBeatsShebang` — file named `foo.go` with a `#!/usr/bin/env bash` shebang; extension wins → `LangGo`. (Extension lookup is priority-1; shebang is only consulted when extension yields unknown or a generic result.)
    - `TestDetect_PeekError_ReturnsUnknown` — construct a `*fileset.File` with a `fstest.MapFS` path that succeeds extension → `LangUnknown` AND whose `Peek` would fail (can use a path that doesn't exist in the MapFS); verify `Detect` returns `LangUnknown` without panicking.
  - `mage build` passes (new leaf package, zero internal deps except `fileset` for `*File` argument type).
  - `mage test github.com/evanmschultz/rak/internal/lang` green with `-race`.
  - `mage lint` green for `internal/lang`.
- **Blocked by:** —

### Unit 5.2 — internal/lang: blank/comment/code line split

- **State:** todo
- **Paths:**
  - `main/internal/lang/split.go` (new file)
  - `main/internal/lang/split_test.go` (new file)
- **Packages:** `github.com/evanmschultz/rak/internal/lang`
- **Acceptance:**
  - `split.go` defines (all new, not yet in tree):
    - `type LineCounts struct { Blank, Comment, Code int }` — three-way line classification per file or aggregate.
    - `type LangCounts struct { Lines LineCounts; Counts counting.Counts }` — combines line split with raw byte/line/word/char counts for a language bucket (F30: lives in `internal/lang`). `Counts` is `counting.Counts` and carries the same fields as today's per-directory rollup.
    - `func Split(r io.Reader, lang Language) (LineCounts, error)` — scans `r` line by line using `bufio.Scanner` (stdlib; no new dep). For each line, classifies it as one of:
      - **Blank**: trimmed line is empty.
      - **Comment**: trimmed line starts with a known line-comment prefix for `lang`, OR the line falls between an open and still-open block-comment delimiter (stateful: `inBlockComment bool` flag).
      - **Code**: everything else.
      - Block-comment state machine: detect `blockCommentOpen` token anywhere in the (non-blank, non-already-in-block-comment) line → set `inBlockComment = true` for the line and subsequent lines. Detect `blockCommentClose` token in a line while `inBlockComment` → set `inBlockComment = false` for subsequent lines. A line is classified `Comment` if `inBlockComment` was true at its start, OR if the line contains a block-comment-open and has no code before it. **Known limitation (F28)**: strings containing comment markers (e.g., `s := "//"`) are counted as comments, not code. This is a deliberate YAGNI trade-off for v0.1.0 — document in BUILDER_WORKLOG.md.
    - Per-language comment grammar table (inside `split.go` or a separate `grammar.go` file in the same package — builder's choice):
      ```
      Language    LineCommentPrefix     BlockOpen    BlockClose
      LangGo      "//"                  "/*"         "*/"
      LangRust    "//"                  "/*"         "*/"
      LangC       "//"                  "/*"         "*/"
      LangCPP     "//"                  "/*"         "*/"
      LangPython  "#"                   '"""'        '"""'  (triple-quote; builder may simplify to # only with a comment)
      LangJS      "//"                  "/*"         "*/"
      LangTS      "//"                  "/*"         "*/"
      LangShell   "#"                   ""           ""
      LangMarkdown ""                   ""           ""     (no standard comment; all non-blank lines = code)
      LangTOML    "#"                   ""           ""
      LangYAML    "#"                   ""           ""
      LangJSON    ""                    ""           ""     (no comments in JSON; all non-blank = code)
      LangHTML    ""                    "<!--"       "-->"
      LangCSS     ""                    "/*"         "*/"
      LangUnknown ""                    ""           ""     (treat all non-blank lines as code)
      ```
      For Python triple-quote block detection, a simplified heuristic is acceptable (e.g., count `"""` occurrences mod 2). Document the simplification in BUILDER_WORKLOG.md.
  - `split_test.go` — table-driven per language:
    - `TestSplit_Go_LineComment` — `"// comment\ncode\n\n"` → `{Blank:1, Comment:1, Code:1}`.
    - `TestSplit_Go_BlockComment` — `"/* start\ncontinued\nend */\ncode\n"` → `{Blank:0, Comment:3, Code:1}`.
    - `TestSplit_Go_InlineBlockComment` — `"x := 1 /* note */\n"` → `{Blank:0, Comment:0, Code:1}` (line has code before block open, should be Code; builder may simplify to Comment — document the decision).
    - `TestSplit_Shell_Hash` — `"# shell comment\necho hi\n"` → `{Blank:0, Comment:1, Code:1}`.
    - `TestSplit_Python_LineComment` — `"# py comment\nx = 1\n"` → `{Blank:0, Comment:1, Code:1}`.
    - `TestSplit_Markdown_AllCode` — `"# heading\ntext\n"` → `{Blank:0, Comment:0, Code:2}` (no comment syntax in Markdown; headings are code).
    - `TestSplit_LangUnknown_AllCode` — unknown language: blank lines = blank, non-blank = code.
    - `TestSplit_EmptyReader` — empty `io.Reader` → `LineCounts{}` zero value, nil error.
    - `TestSplit_BlankLines` — multiple `"\n\n\n"` → all blank.
    - `TestSplit_CRLF` — `"code\r\nblank\r\n\r\n"` → `{Blank:1, Comment:0, Code:2}` (CRLF lines handled by `bufio.Scanner`'s default split which strips `\r`).
  - `mage build` and `mage test github.com/evanmschultz/rak/internal/lang` green with `-race`.
  - `mage lint` green for `internal/lang`.
- **Blocked by:** 5.1

### Unit 5.3 — Per-type aggregation in render output (all three renderers + cmd/rak wiring)

- **State:** todo
- **Paths:**
  - `main/internal/render/render.go` (extend `Directory` struct — additive growth; F25 amended: interface method signatures unchanged)
  - `main/internal/render/human.go` (extend `RenderTree` to emit per-lang block when `ByLang` non-empty)
  - `main/internal/render/json.go` (extend `treeJSON`/`dirJSON` envelope with `by_lang` field)
  - `main/internal/render/toon.go` (extend `RenderTree` to emit per-lang rows in TOON block)
  - `main/internal/render/render_test.go` (extend snapshot tests: nil-ByLang path unchanged; non-nil ByLang path added)
  - `main/cmd/rak/root.go` (extend `walkAndCount` to call `lang.Detect` + `lang.Split` per file; build per-dir/per-lang `LangCounts` map; populate `Directory.ByLang`)
  - `main/cmd/rak/root_test.go` (extend tests for per-type aggregation path; nil-ByLang backward-compat cases stay)
  - `main/cmd/rak/integration_test.go` (extend or add test asserting per-lang data in JSON output)
- **Packages:** `github.com/evanmschultz/rak/internal/render`, `github.com/evanmschultz/rak/cmd/rak`
- **Acceptance:**
  - `render.go` `Directory` struct gains two new fields (additive):
    - `ByLang map[lang.Language]lang.LangCounts` — per-language aggregation. `nil` when language detection was not run (backward-compatible: existing tests that omit the field still pass because nil map is the zero value).
    - Requires importing `github.com/evanmschultz/rak/internal/lang`. Add to the `render` package's import list.
    - Doc comment updated to describe the new field and the nil-means-no-detection semantic.
  - `Renderer` interface signature UNCHANGED — `RenderTree(w io.Writer, dirs []Directory, total counting.Counts, errs []error) error`. Only the `Directory` struct grows. (F25 preserved: no breaking change to the interface contract; the struct's new field is additive. F32 pin: nil-ByLang is silently skipped by all renderer implementations.)
  - All three renderer `RenderTree` implementations extended:
    - When `d.ByLang` is nil or empty: output unchanged from today's behavior (existing snapshot tests continue to pass).
    - When `d.ByLang` is non-empty: each renderer emits per-language detail in its native format:
      - **Human renderer (laslig)**: for each language `L` in a deterministic order (sorted by language string), emit one additional KV row under the directory block, e.g., `"lang/<L>": "<code>/<comment>/<blank> lines"`. Exact format is builder's choice — document in BUILDER_WORKLOG.md.
      - **JSON renderer**: `dirJSON` struct grows an optional `"by_lang"` field: `map[string]struct{Lines struct{Code,Comment,Blank int}; Counts counting.Counts}`. Omit (`omitempty`) when nil. Key is `string(lang.Language)`.
      - **TOON renderer**: emit per-language rows as additional `key: value` lines in the TOON block for the directory. Format is builder's choice — document in BUILDER_WORKLOG.md (U1 open for dev input in Phase 3; builder uses a reasonable default).
  - `walkAndCount` in `cmd/rak/root.go` extended:
    - Per file (after binary check, before `countFile`), call `lang.Detect(f)` to obtain the file's `Language`. Call `lang.Split(rc, detectedLang)` on the opened reader to obtain `LineCounts`. (Ordering note: `countFile` opens the file and calls `counting.Count(rc)`. The builder may choose to open the file once and run both `counting.Count` and `lang.Split` in a single pass, OR open twice via `f.Open()`. Either is acceptable — document the choice in BUILDER_WORKLOG.md. If a single-pass approach is used, `counting.Count` and `lang.Split` must both accept `io.Reader`, so the reader must be wrapped with `io.TeeReader` or the file opened twice. The two-open approach is simpler and acceptable for v0.1.0.)
    - Maintain per-directory `byDirLang map[string]map[lang.Language]lang.LangCounts` alongside the existing `byDir map[string]counting.Counts`. Accumulate `LangCounts` per dir+lang key. After the loop, populate `Directory.ByLang` from the per-dir lang map.
    - If `lang.Split` returns an error for a file, append to `aggErrs` and continue (do not set `LineCounts` for that file — treat as unknown split).
    - `total counting.Counts` aggregation unchanged.
  - Render snapshot tests in `render_test.go`:
    - Existing tests that construct `Directory{Path: ".", Counts: ...}` (no `ByLang` field) continue to pass unchanged (nil ByLang path).
    - New tests: at least one test per renderer that passes a `Directory` with non-empty `ByLang` and asserts the per-language data appears in output (e.g., `strings.Contains` for JSON; `strings.Contains` for TOON/human).
    - Nil-safety test: construct `Directory{ByLang: nil}` — all three renderers must not panic. `mage test -race` catches data races if ByLang is accessed without nil guard.
  - `mage build` and `mage test github.com/evanmschultz/rak/internal/render ./cmd/rak/...` green with `-race`.
  - `mage lint` green for both packages.
- **Blocked by:** 5.2

### Unit 5.4 — --lang walk-filter flag

- **State:** todo
- **Paths:**
  - `main/cmd/rak/root.go` (add `lang []string` to `rootFlags`, add cobra `--lang` flag, add lang-filter gate in `walkAndCount`)
  - `main/cmd/rak/root_test.go` (add `--lang` flag-parsing tests + filter-behavior tests)
- **Packages:** `github.com/evanmschultz/rak/cmd/rak`
- **Acceptance:**
  - `rootFlags` struct gains:
    - `langs []string` — cobra `--lang` CSV/repeatable flag. Zero-length means "no filter" (count all languages).
  - `newRootCmd` flag registration adds:
    - `cmd.Flags().StringSliceVar(&flags.langs, "lang", nil, "filter counted files to comma-separated language names (e.g. go,rs); default: count all")` — `StringSliceVar` accepts comma-separated values AND repeated flags (cobra behavior).
  - `walkAndCount` gains a lang-filter gate (new parameter or closure over `flags.langs`):
    - After `lang.Detect(f)` call (already added in 5.3), if `len(langFilter) > 0` and the detected language is not in the filter set (`map[lang.Language]struct{}`), skip the file — do not count it, do not include it in per-dir or per-type rollup. This is the decision-24 "walk filter" semantic: filtered files are not counted, not rendered, not in error summary.
    - Build the `map[lang.Language]struct{}` filter set once before the per-file loop from `flags.langs`. Case-insensitive match: normalize user input with `lang.Language(strings.ToLower(v))`.
    - When `len(langFilter) == 0`, no filtering occurs — all files counted.
  - `root_test.go` additions:
    - `TestRootCmd_LangFlag_ParsesCSV` — call `newRootCmd()`, pass `--lang go,rs` as args, verify `flags.langs` becomes `["go", "rs"]`.
    - `TestRootCmd_LangFlag_RepeatedFlag` — pass `--lang go --lang rs`, verify same result.
    - `TestRunDirectory_LangFilter_ExcludesOtherLangs` — construct a `fstest.MapFS` with `a.go` (Go source) and `b.py` (Python source), wrap in `lister.NewWalkLister`, call `runDirectory` with `--lang go`; verify only `a.go` contributes to the count (directory total reflects only the Go file's bytes/lines). Uses `NewWalkLister` directly to avoid git dependency.
    - `TestRunDirectory_LangFilter_NoFilter` — same MapFS, no `--lang` flag; verify both files are counted.
  - `mage build` and `mage test ./cmd/rak/...` green with `-race`.
  - `mage ci` green (gofumpt clean + lint + test-race).
- **Blocked by:** 5.3

## Notes

### Library choice: inline table vs go-enry

Decision: **inline table only**. `github.com/go-enry/go-enry/v2` (~1.5MB binary footprint addition, large dep surface with linguist data) conflicts with decision 30's "fast and small." An inline table covering 15+ languages handles rak's primary use cases (LLM-first code sizing). The content-heuristic fallback and shebang sniff provide reasonable coverage for extensionless files. If language coverage gaps surface post-v0.1.0, revisit go-enry then. This decision is locked — plan-QA falsification should attack it if coverage is insufficient.

### F-pin register (Drop 5, starting at F27)

- **F27**: `lang.Detect` never propagates `fileset.File.Peek` errors. Detection failure returns `LangUnknown`; errors are silently discarded. Callers (cmd/rak) must not depend on Peek-level error signals from detection.
- **F28**: `lang.Split` line-classification known limitation — strings containing comment markers (e.g., `s := "//"` in Go) are mis-classified as comments, not code. YAGNI trade-off; acceptable for v0.1.0. Document in BUILDER_WORKLOG.md.
- **F29**: `LangUnknown Language = ""` is the zero value. The `--lang` filter rejects `LangUnknown` files when any filter is set (they are not "unknown language" files; they are undetected). Builder may choose to allow `--lang unknown` as a special value to select undetected files if it is trivial; otherwise defer to v0.2.
- **F30**: `LangCounts` and `LineCounts` live in `internal/lang`. `internal/render` imports `internal/lang`. Import DAG becomes: `lang → fileset, counting`; `render → lang, counting`. No cycles.
- **F31**: `Directory.ByLang map[lang.Language]lang.LangCounts` is nil when no language detection was run (backward-compatible zero value). All renderer `RenderTree` implementations must guard on nil/empty `ByLang` before iterating.
- **F32**: `Renderer` interface method signatures are UNCHANGED in Drop 5. Only `Directory` struct grows (additive struct field). Pre-v1.0 additive growth of a struct with no external implementers is safe per F15/PLAN.md.

### Parallel eligibility

**No parallelism in this drop.** The dependency chain is linear:
- 5.1 → unblocked
- 5.2 → blocked by 5.1 (same package `internal/lang`)
- 5.3 → blocked by 5.2 (needs `LangCounts`, `LineCounts`, `Detect`, `Split` from `internal/lang`; also modifies `cmd/rak/root.go`)
- 5.4 → blocked by 5.3 (shares `cmd/rak/root.go` + `cmd/rak/root_test.go` with 5.3; also needs `lang.Detect` call already added in 5.3)

The orchestrator dispatches units sequentially: 5.1, then 5.2, then 5.3, then 5.4.

### Open Unknowns for Phase 3 dev discussion

- **U1**: Exact per-type rollup format in TOON and human output — what level of detail is desired (line counts only? also byte counts? sorted how?)? Plan-QA should surface this; dev decides in Phase 3. Builder uses a reasonable default if not addressed.
- **U2**: Should `--lang unknown` be a valid filter value to explicitly select files whose language rak could not detect? Current plan: not supported (undetected files are silently excluded when a filter is set). If dev wants to count only undetected files, this needs an explicit decision.
- **U3**: Should per-type rollup be present in TOON/human output by default, or opt-in (e.g., `--detail` flag)? Current plan: always-on in all three renderers when language detection produces a non-empty map (i.e., any file was detected). No new flag. Revisit if the output gets cluttered.

### Carve-out / compile note

No carve-out unit needed. 5.1 and 5.2 only write to a new `internal/lang` package — they do not break any existing package's compilation. `cmd/rak` does not import `internal/lang` until 5.3, so the first two units leave the entire existing tree compiling and testing cleanly.
