# DROP_5 — LANGUAGE_DETECTION_CODE_SPLITS

**State:** building
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

Drop 4's spine is preserved: `internal/lister`, `internal/fileset`, `internal/ignore`, `internal/counting`, `internal/render`'s `Renderer` interface (subject to a possible additive growth — planner decides), and the `--human` / `--json` / `--toon` flag surface all remain. Drop 5's new code is additive. Expected decomposition: 4 atomic units (5.1 internal/lang detection + `lang.Detect` call-site wiring in walkAndCount / 5.2 code-aware splits / 5.3 per-type aggregation in render / 5.4 `--lang` walk filter). Per the `feedback_parallelize_aggressively` memory rule, 5.2 and 5.4 are eligible to run in parallel after 5.1 closes (both consume `Language` but neither blocks the other).

`--as <lang>` (stream-type assertion for stdin) is cut per decision 30; only `--lang` (walk filter) is added in Drop 5.

## Planner

**Revised chain (Round 2 — Decision C1):** `5.1 → {5.2 || 5.4} → 5.3`.

5.1 grows scope to wire `lang.Detect` into `cmd/rak/root.go`'s `walkAndCount`, so that both 5.2 (Split logic in `internal/lang`) and 5.4 (`--lang` filter gate in `cmd/rak`) can proceed independently and in parallel. 5.3 sequences after both: it needs `LangCounts` from 5.2 and merges per-lang rollup into `walkAndCount` (which 5.4 last touched in `root.go`).

### Unit 5.1 — internal/lang: Language type + detection, plus Detect call-site wiring in cmd/rak

- **State:** todo
- **Paths:**
  - `main/internal/lang/lang.go` (new file)
  - `main/internal/lang/lang_test.go` (new file)
  - `main/cmd/rak/root.go` (extend `walkAndCount` — wire `lang.Detect(f)` call per file, store the result in a per-iteration local for downstream consumers in 5.2/5.3/5.4)
  - `main/cmd/rak/root_test.go` (extend: verify detect call-site compiles + smoke test)
- **Packages:**
  - `github.com/evanmschultz/rak/internal/lang` (new package)
  - `github.com/evanmschultz/rak/cmd/rak`
- **Acceptance:**
  - `lang.go` defines the following (all new, not yet in tree):
    - `type Language string` — named string type per decision 27(c).
    - Language values are stored lowercase by convention (e.g. `LangGo Language = "go"`). All internal map lookups normalize keys to lowercase. (F27, C6.)
    - `const LangUnknown Language = ""` — zero-value constant; returned by `Detect` when no rule matches (F29).
    - Named language constants for each entry in the detection table. Minimum coverage: Go, Rust, Python, JavaScript, TypeScript, C, C++, Shell (sh/bash), Markdown, TOML, YAML, JSON, Makefile, HTML, CSS, plus `LangDocker` (`"docker"`) and `LangCMake` (`"cmake"`) added per Decision C2. Additional entries are welcome but not required.
    - `func Detect(f *fileset.File) Language` — 4-step detection pipeline in priority order (F27):
      1. **Special-filename lookup**: consult inline `specialFilenames map[string]Language`. Lookup key is `strings.ToLower(filepath.Base(f.RelPath))` so nested files like `sub/Makefile` match correctly. Keys (already lowercase) include at minimum: `"makefile"`, `"gnumakefile"`, `"dockerfile"`, `"cmakelists.txt"`. If match → return immediately. Exact basename match only (no prefix/suffix matching) so `Makefile.go` does NOT match `Makefile`. (F1 falsification carry-forward.)
      2. **Extension lookup**: `strings.ToLower(filepath.Ext(f.RelPath))` → consult inline `extensionTable map[string]Language`. Keys are lowercase WITH the leading dot (e.g. `".go"`), matching `filepath.Ext` output directly (F27, P5). No error return — extension lookup is pure. If match → return immediately.
      3. **Shebang sniff**: run only when steps 1 + 2 both returned `LangUnknown`. Calls `f.Peek(512)` (F4 contract). If first line starts with `#!`, extract the interpreter path and consult inline `shebangsTable map[string]Language`. If `Peek` returns an error, treat as no-match and log nothing — detection failure → `LangUnknown`. `Detect` never propagates `Peek` errors; callers must not depend on them (F27, P3).
      4. **Content heuristic**: run only when steps 1 + 2 + 3 all returned `LangUnknown`. Scans the first 512 bytes from `Peek(512)` for well-known markers (e.g. `<?xml`, `<!DOCTYPE`, `{`, `[` as JSON candidates; `---` as YAML front-matter). Heuristic is best-effort. If no marker matches, return `LangUnknown`. There is NO "generic language" intermediate state — the pipeline returns the first concrete match OR `LangUnknown` (C5).
    - The inline `extensionTable` maps lowercase extensions WITH the leading dot to `Language` constants (e.g. `".go"` → `LangGo`, `".rs"` → `LangRust`). See P5.
    - Doc comments on every exported identifier per naming rules.
  - `lang_test.go` — table-driven, using `testing/fstest.MapFS` to construct `*fileset.File` values via `fileset.NewFile(fsys, path, path)`:
    - `TestDetect_ByExtension` — table: `.go` → `LangGo`, `.rs` → `LangRust`, `.py` → `LangPython`, `.js` → `LangJS`, `.ts` → `LangTS`, `.sh` → `LangShell`, `.md` → `LangMarkdown`, `.toml` → `LangTOML`, `.yaml`/`.yml` → `LangYAML`, `.json` → `LangJSON`, `.c` → `LangC`, `.cpp`/`.cc` → `LangCPP`, `.html` → `LangHTML`, `.css` → `LangCSS`, unknown extension → `LangUnknown`. (All test names new, not yet in tree.)
    - `TestDetect_SpecialFilename` — table: `Makefile` → `LangMakefile`, `makefile` (lowercase) → `LangMakefile`, `Dockerfile` → `LangDocker`, `CMakeLists.txt` → `LangCMake`, `GNUmakefile` → `LangMakefile`, **`sub/Makefile` → `LangMakefile`** (nested path; basename match), **`Makefile.go` → `LangGo`** (prefix-only basename must NOT match special-filename table; falls through to extension lookup). (Decision C2 + R2 F1 fix; new, not yet in tree.)
    - `TestDetect_Shebang_Shell` — file with no extension but content `#!/bin/bash\necho hi` → `LangShell` (or `LangBash` — builder's choice; document in BUILDER_WORKLOG.md). New, not yet in tree.
    - `TestDetect_Shebang_Python` — file named `script` with `#!/usr/bin/env python3\n` → `LangPython`. New, not yet in tree.
    - `TestDetect_UnknownExtension_NoShebang` — file with `.xyzzy` extension and no shebang → `LangUnknown`. New, not yet in tree.
    - `TestDetect_ExtensionBeatsShebang` — file named `foo.go` with a `#!/usr/bin/env bash` shebang; extension wins (step 2 before step 3) → `LangGo`. New, not yet in tree.
    - `TestDetect_PeekError_ReturnsUnknown` — construct a `*fileset.File` with a `fstest.MapFS` path that succeeds extension → `LangUnknown` AND whose `Peek` would fail; verify `Detect` returns `LangUnknown` without panicking. New, not yet in tree.
  - `cmd/rak/root.go` `walkAndCount` extended in this unit:
    - After the binary-file check, call `lang.Detect(f)` once per file. Store the resulting `Language` value in a per-file local (e.g. `detectedLang := lang.Detect(f)`). This wired value is what 5.2's `Split` call (added in 5.3) and 5.4's filter gate (added in 5.4) will each consume. No `Split` call yet — 5.1 only wires the `Detect` invocation and imports `internal/lang`.
    - F26 RelPath invariant: unaffected. `lang.Detect` reads `f.RelPath` for extension and filename lookup only.
  - `mage build` passes (new leaf package, deps: `fileset` for `*File` argument type; `cmd/rak` gains import of `internal/lang`).
  - `mage test github.com/evanmschultz/rak/internal/lang` green with `-race`.
  - `mage test ./cmd/rak/...` green with `-race` (existing + new smoke tests).
  - `mage lint` green for both packages.
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
    - `func (lc *LangCounts) Add(other LangCounts)` — accumulates `other` into `lc` field-by-field. Used by 5.3's rollup accumulator to aggregate across files. (New, not yet in tree; P6.)
    - `func Split(r io.Reader, lang Language) (LineCounts, error)` — scans `r` line by line using `bufio.Scanner` (stdlib; no new dep). For each line, classifies it as one of:
      - **Blank**: trimmed line is empty.
      - **Comment**: classified per Policy α (F28, Decision C4) — see below.
      - **Code**: everything else.
      - Block-comment state machine: `inBlockComment bool` flag tracks whether the scanner is inside a block comment.
    - **Policy α (F28, Decision C4)**: a line is classified as `Comment` if it contains ANY block-comment marker (`/*` or `*/`) anywhere in the line — regardless of code preceding or following the marker. Examples (Go/C-style):
      - `/* a */ b /* c */` → Comment (block-comment marker present).
      - `x := 1 /* note */` → Comment (block-comment marker present; overrides Decision C4 over the Round 1 draft which said this is Code).
      - `/* still open` → Comment (open marker; `inBlockComment` set true).
      - `closing */ x := 2` → Comment (close marker present; `inBlockComment` set false; but the line is the close, so it is a comment line).
      - `x := 1` → Code (no marker).
      - **Known limitation**: strings containing `/*` or `*/` (e.g. `s := "/*"`) get classified as Comment lines. This is a deliberate YAGNI trade-off for v0.1.0, pinned in F28 and matching `cloc`'s default behavior. Document in BUILDER_WORKLOG.md.
    - **Python docstrings (C7)**: triple-quoted strings are strings at the language level, not comments. `Split` classifies them as Code. This matches `cloc`. Document in BUILDER_WORKLOG.md.
    - Per-language comment grammar table (inside `split.go` or a separate `grammar.go` in the same package — builder's choice):
      ```
      Language    LineCommentPrefix     BlockOpen    BlockClose
      LangGo      "//"                  "/*"         "*/"
      LangRust    "//"                  "/*"         "*/"
      LangC       "//"                  "/*"         "*/"
      LangCPP     "//"                  "/*"         "*/"
      LangPython  "#"                   '"""'        '"""'  (triple-quote; builder may simplify to # only; document)
      LangJS      "//"                  "/*"         "*/"
      LangTS      "//"                  "/*"         "*/"
      LangShell   "#"                   ""           ""
      LangMarkdown ""                   ""           ""     (no standard comment; all non-blank = code)
      LangTOML    "#"                   ""           ""
      LangYAML    "#"                   ""           ""
      LangJSON    ""                    ""           ""     (no comments; all non-blank = code)
      LangHTML    ""                    "<!--"       "-->"
      LangCSS     ""                    "/*"         "*/"
      LangMakefile "#"                  ""           ""
      LangDocker  "#"                   ""           ""
      LangCMake   "#"                   ""           ""
      LangUnknown ""                    ""           ""     (all non-blank = code)
      ```
      For Python triple-quote block detection, a simplified heuristic is acceptable (e.g., count `"""` occurrences mod 2). Document in BUILDER_WORKLOG.md.
  - `split_test.go` — table-driven per language:
    - `TestSplit_Go_LineComment` — `"// comment\ncode\n\n"` → `{Blank:1, Comment:1, Code:1}`. New, not yet in tree.
    - `TestSplit_Go_BlockComment` — `"/* start\ncontinued\nend */\ncode\n"` → `{Blank:0, Comment:3, Code:1}`. New, not yet in tree.
    - `TestSplit_BlockCommentOpenClosePerLine` — `"/* a */ b /* c */\n"` → `{Blank:0, Comment:1, Code:0}` (Policy α; Decision C4). New, not yet in tree.
    - `TestSplit_TrailingComment` — `"x := 1 /* note */\n"` → `{Blank:0, Comment:1, Code:0}` (Policy α; Decision C4). New, not yet in tree.
    - `TestSplit_StringContainsMarker_KnownLimitation` — `"s := \"/*\"\n"` → `{Blank:0, Comment:1, Code:0}` (Policy α known limitation; test documents YAGNI explicitly). New, not yet in tree.
    - `TestSplit_Shell_Hash` — `"# shell comment\necho hi\n"` → `{Blank:0, Comment:1, Code:1}`. New, not yet in tree.
    - `TestSplit_Python_LineComment` — `"# py comment\nx = 1\n"` → `{Blank:0, Comment:1, Code:1}`. New, not yet in tree.
    - `TestSplit_Markdown_AllCode` — `"# heading\ntext\n"` → `{Blank:0, Comment:0, Code:2}`. New, not yet in tree.
    - `TestSplit_LangUnknown_AllCode` — unknown language: blank lines = blank, non-blank = code. New, not yet in tree.
    - `TestSplit_EmptyReader` — empty `io.Reader` → `LineCounts{}` zero value, nil error. New, not yet in tree.
    - `TestSplit_BlankLines` — multiple `"\n\n\n"` → all blank. New, not yet in tree.
    - `TestSplit_CRLF` — `"code\r\nblank\r\n\r\n"` → `{Blank:1, Comment:0, Code:2}` (`bufio.Scanner`'s default split strips `\r`). New, not yet in tree.
  - `mage build` and `mage test github.com/evanmschultz/rak/internal/lang` green with `-race`.
  - `mage lint` green for `internal/lang`.
- **Blocked by:** 5.1

### Unit 5.4 — --lang walk-filter flag

- **State:** todo
- **Paths:**
  - `main/cmd/rak/root.go` (add `langs []string` to `rootFlags`, add cobra `--lang` flag, add lang-filter gate in `walkAndCount`)
  - `main/cmd/rak/root_test.go` (add `--lang` flag-parsing tests + filter-behavior tests)
- **Packages:** `github.com/evanmschultz/rak/cmd/rak`
- **Acceptance:**
  - `rootFlags` struct gains:
    - `langs []string` — cobra `--lang` CSV/repeatable flag. Zero-length means "no filter" (count all languages).
  - `newRootCmd` flag registration adds:
    - `cmd.Flags().StringSliceVar(&flags.langs, "lang", nil, "filter counted files to comma-separated language names (e.g. go,rs); default: count all")` — `StringSliceVar` accepts comma-separated values AND repeated flags (cobra behavior).
  - `walkAndCount` gains a lang-filter gate:
    - Consumes the `detectedLang` value already wired by Unit 5.1 (no second `lang.Detect` call). If `len(langFilter) > 0` and the detected language is not in the filter set (`map[lang.Language]struct{}`), skip the file — do not count it, do not include it in per-dir or per-type rollup. This is the decision-24 "walk filter" semantic: filtered files are not counted, not rendered, not in error summary. (C8: call-site owned by 5.1; 5.4 reads the already-wired value.)
    - Build the `map[lang.Language]struct{}` filter set once before the per-file loop from `flags.langs`. Case-insensitive match: normalize user input with `lang.Language(strings.ToLower(v))`.
    - When `len(langFilter) == 0`, no filtering occurs — all files counted.
  - `root_test.go` additions (all new, not yet in tree):
    - `TestRootCmd_LangFlag_ParsesCSV` — call `newRootCmd()`, pass `--lang go,rs` as args, verify `flags.langs` becomes `["go", "rs"]`.
    - `TestRootCmd_LangFlag_RepeatedFlag` — pass `--lang go --lang rs`, verify same result.
    - `TestRunDirectory_LangFilter_ExcludesOtherLangs` — construct a `fstest.MapFS` with `a.go` (Go source) and `b.py` (Python source), wrap in `lister.NewWalkLister`, call `runDirectory` with `--lang go`; verify only `a.go` contributes to the count. Uses `NewWalkLister` directly to avoid git dependency.
    - `TestRunDirectory_LangFilter_NoFilter` — same MapFS, no `--lang` flag; verify both files are counted.
  - `mage build` and `mage test ./cmd/rak/...` green with `-race`.
  - `mage lint` green for `cmd/rak`.
- **Blocked by:** 5.1
- **Note:** 5.4 is parallel-eligible with 5.2. Both unblock after 5.1 closes. Dep-edge reasoning: 5.4 has no symbol dependency on anything 5.2 produces (`LangCounts`, `LineCounts`, `Split`), and they touch strictly disjoint file sets (`cmd/rak/root.go` + `root_test.go` vs `internal/lang/split.go` + `split_test.go`). See Notes § "Parallel eligibility."

### Unit 5.3 — Per-type aggregation in render output (all three renderers + cmd/rak wiring)

- **State:** todo
- **Paths:**
  - `main/internal/render/render.go` (extend `Directory` struct — additive growth; F25 amended: interface method signatures unchanged)
  - `main/internal/render/human.go` (extend `RenderTree` to emit per-lang block when `ByLang` non-empty; suppress `LangUnknown` key per F33)
  - `main/internal/render/json.go` (extend `treeJSON`/`dirJSON` envelope with `by_lang` field; suppress `LangUnknown` key per F33)
  - `main/internal/render/toon.go` (extend `RenderTree` to emit per-lang rows in TOON block; suppress `LangUnknown` key per F33)
  - `main/internal/render/render_test.go` (extend snapshot tests: nil-ByLang path unchanged; non-nil ByLang path added; LangUnknown-suppression tests added)
  - `main/cmd/rak/root.go` (extend `walkAndCount`: call `lang.Split` per file; build per-dir/per-lang `LangCounts` map; populate `Directory.ByLang`)
  - `main/cmd/rak/root_test.go` (extend tests for per-type aggregation path; nil-ByLang backward-compat cases stay)
  - `main/cmd/rak/integration_test.go` (extend or add test asserting per-lang data in JSON output)
- **Packages:** `github.com/evanmschultz/rak/internal/render`, `github.com/evanmschultz/rak/cmd/rak`
- **Acceptance:**
  - `render.go` `Directory` struct gains two new fields (additive):
    - `ByLang map[lang.Language]lang.LangCounts` — per-language aggregation. `nil` when language detection was not run (backward-compatible: existing tests that omit the field still pass because nil map is the zero value).
    - Requires importing `github.com/evanmschultz/rak/internal/lang`. Add to the `render` package's import list.
    - Doc comment updated to describe the new field and the nil-means-no-detection semantic.
  - **F33 — Unknown-language suppression (Decision P2+C3):** `ByLang` maps with the `LangUnknown` key MUST be filtered out before emission in ALL three renderers. The per-dir `Counts` field (existing) continues to count those files (so totals stay accurate); only the per-type cross-cut excludes `LangUnknown` entries. Applies uniformly to TOON, JSON, and human renderers.
  - `Renderer` interface signature UNCHANGED — `RenderTree(w io.Writer, dirs []Directory, total counting.Counts, errs []error) error`. Only the `Directory` struct grows. (F25 preserved; F32 pin: nil-ByLang is silently skipped by all renderer implementations.)
  - All three renderer `RenderTree` implementations extended:
    - When `d.ByLang` is nil or empty (after filtering `LangUnknown`): output unchanged from today's behavior (existing snapshot tests continue to pass).
    - When `d.ByLang` is non-empty (after filtering `LangUnknown`): each renderer emits per-language detail in its native format:
      - **Human renderer (laslig)**: for each language `L` in a deterministic order (sorted by language string, `LangUnknown` excluded), emit one additional KV row under the directory block.
      - **JSON renderer**: `directoryJSON` struct (existing — used by `directoryJSON(d)` conversion at `internal/render/json.go`) MUST grow a `ByLang map[lang.Language]lang.LangCounts` field matching `Directory.ByLang` byte-for-byte. **Critical (F34, R2 falsification C1):** Go struct-type conversion `directoryJSON(d)` requires identical field structure between `Directory` and `directoryJSON` — failing to add the same field with the same Go type to `directoryJSON` will break compile at the existing conversion site. The JSON tag on the new field is `json:"by_lang,omitempty"`. Filter `LangUnknown` from the map BEFORE the conversion (build a filtered copy in `Directory`-shape, then convert) so the JSON marshaling never sees `LangUnknown`. Key serialization is `string(lang.Language)` via the underlying string type.
      - **TOON renderer**: emit per-language rows as additional `key: value` lines in the TOON block for the directory. `LangUnknown` key excluded. Format is builder's choice — document in BUILDER_WORKLOG.md.
  - `walkAndCount` in `cmd/rak/root.go` extended:
    - Per file (after the lang-filter gate added in 5.4, using the `detectedLang` wired in 5.1), call `lang.Split(rc, detectedLang)` on the opened reader to obtain `LineCounts`. Use `LangCounts.Add` (from 5.2) to accumulate per-dir/per-lang totals. (P4: `Detect` calls `f.Peek(512)`; `Split` opens the file via `f.Open()` — two separate opens per file. Acceptable for v0.1.0; FS page cache makes the second open cheap. See Notes § "Double-IO trade-off".)
    - Maintain per-directory `byDirLang map[string]map[lang.Language]lang.LangCounts` alongside the existing `byDir map[string]counting.Counts`. After the loop, populate `Directory.ByLang` from the per-dir lang map.
    - If `lang.Split` returns an error for a file, append to `aggErrs` and continue (do not set `LineCounts` for that file — treat as unknown split).
    - `total counting.Counts` aggregation unchanged.
  - Render tests in `render_test.go`:
    - Existing tests that construct `Directory{Path: ".", Counts: ...}` (no `ByLang` field) continue to pass unchanged.
    - New per-renderer tests with non-empty `ByLang` asserting per-language data appears in output.
    - Nil-safety: `Directory{ByLang: nil}` — all three renderers must not panic.
    - **F33 suppression tests (Decision P2+C3; all new, not yet in tree):**
      - `TestTOONRenderer_RenderTree_AllUnknown` — input `[]Directory{{Path:".", Counts:..., ByLang:map[lang.Language]lang.LangCounts{lang.LangUnknown: {...}}}}`. Verify output does NOT contain `"unknown"` or `""` as a language key.
      - `TestJSONRenderer_RenderTree_AllUnknown` — same input; verify JSON output's `by_lang` field is absent or empty.
      - `TestHumanRenderer_RenderTree_AllUnknown` — same input; verify human output does not emit a language row for unknown.
  - `mage build` and `mage test github.com/evanmschultz/rak/internal/render ./cmd/rak/...` green with `-race`.
  - `mage lint` green for both packages.
- **Blocked by:** 5.2, 5.4

## Notes

### Library choice: inline table vs go-enry

Decision: **inline table only**. `github.com/go-enry/go-enry/v2` (~1.5MB binary footprint addition, large dep surface with linguist data) conflicts with decision 30's "fast and small." An inline table covering 15+ languages handles rak's primary use cases (LLM-first code sizing). The content-heuristic fallback and shebang sniff provide reasonable coverage for extensionless files. If language coverage gaps surface post-v0.1.0, revisit go-enry then. This decision is locked — plan-QA falsification should attack it if coverage is insufficient.

### F-pin register (Drop 5, starting at F27)

- **F27**: `lang.Detect` implements a 4-step priority pipeline: (1) special-filename lookup, (2) extension lookup, (3) shebang sniff (only when 1+2 both returned `LangUnknown`), (4) content heuristic (only when 1+2+3 all returned `LangUnknown`). Extension table keys are lowercase WITH the leading dot (e.g. `".go"`), matching `filepath.Ext` output directly (P5). Special-filename lookup is case-insensitive (normalize with `strings.ToLower`). Language values are stored lowercase (C6). `Detect` never propagates `fileset.File.Peek` errors; detection failure returns `LangUnknown` silently. `Detect` has NO "generic language" intermediate state — returns the first concrete match OR `LangUnknown` (C5).
- **F28**: `lang.Split` uses **Policy α** (Decision C4): a line is classified as `Comment` if it contains ANY block-comment marker (`/*` or `*/`) anywhere in the line, regardless of code preceding or following the marker. This is intentionally coarse and matches `cloc`'s default behavior. Known limitation: string literals containing `/*` or `*/` (e.g. `s := "/*"`) are mis-classified as comment lines. YAGNI trade-off for v0.1.0. **Python docstrings** (triple-quoted strings) are strings at the language level, not comments; `Split` classifies them as Code (C7). Document both in BUILDER_WORKLOG.md.
- **F29**: `LangUnknown Language = ""` is the zero value. The `--lang` filter rejects `LangUnknown` files when any filter is set (they are not "unknown language" files; they are undetected). Builder may choose to allow `--lang unknown` as a special value to select undetected files if it is trivial; otherwise defer to v0.2.
- **F30**: `LangCounts` and `LineCounts` live in `internal/lang`. `internal/render` imports `internal/lang`. Import DAG becomes: `lang → fileset, counting`; `render → lang, counting`. No cycles.
- **F31**: `Directory.ByLang map[lang.Language]lang.LangCounts` is nil when no language detection was run (backward-compatible zero value). All renderer `RenderTree` implementations must guard on nil/empty `ByLang` before iterating.
- **F32**: `Renderer` interface method signatures are UNCHANGED in Drop 5. Only `Directory` struct grows (additive struct field). Pre-v1.0 additive growth of a struct with no external implementers is safe per F15/PLAN.md.
- **F33**: `LangUnknown` suppression in renderers (Decision P2+C3). `ByLang` maps with the `LangUnknown` key MUST be filtered out before emission in all three renderers (TOON, JSON, human). Per-dir `Counts` (existing field) still includes those files so totals stay accurate; only the per-type cross-cut excludes them. New in Round 2.
- **F34**: `directoryJSON` struct in `internal/render/json.go` MUST mirror `Directory.ByLang` byte-for-byte (same field name, same Go type `map[lang.Language]lang.LangCounts`, additive). The existing `directoryJSON(d)` Go struct-type conversion at the renderer call site requires identical field structure between source and target structs; failing to mirror `ByLang` will break compile. F33 LangUnknown suppression is applied in `Directory`-shape BEFORE the conversion (build a filtered copy of `Directory`, then convert) so JSON marshaling never sees `LangUnknown`. New in Round 2 (R2 falsification C1).

### Parallel eligibility

**Revised chain (Round 2 — Decision C1):** `5.1 → {5.2 || 5.4} → 5.3`.

After 5.1 closes, 5.2 and 5.4 are eligible to run in parallel. The eligibility follows from dep-edge reasoning (P1), not package membership:

- **5.2 vs 5.4 — no symbol dependency and disjoint file sets.** 5.4 consumes the `Language` value that 5.1 already wired into `walkAndCount`; it does not call `Split`, use `LangCounts`, or touch `internal/lang/split.go`. 5.2 adds `Split`, `LineCounts`, `LangCounts`, and `LangCounts.Add` entirely inside `internal/lang/` — no overlap with `cmd/rak/root.go` or `cmd/rak/root_test.go`. Two agents can safely work in parallel with no file or package collisions.
- **5.3 sequences after both.** It needs `LangCounts` and `LangCounts.Add` (produced by 5.2) and also merges the `lang.Split` call + per-lang rollup accumulator into `walkAndCount` in `cmd/rak/root.go` (the same file 5.4 last touched). Both blocking edges are correct.

The orchestrator dispatches 5.1 first, then 5.2 and 5.4 in parallel, then 5.3 after both close.

### Double-IO trade-off (P4)

`lang.Detect(f)` calls `f.Peek(512)` for shebang sniff and content heuristic. `lang.Split(rc, lang)` later opens the file via `f.Open()` to scan line-by-line. This means two opens per file. Acceptable for v0.1.0: the FS page cache makes the second `Open` cheap for typical project sizes, and the two-open approach is significantly simpler than wrapping with `io.TeeReader`. Builder may choose single-pass with `io.TeeReader` if preferred — document the decision in BUILDER_WORKLOG.md. See also Unit 5.3 acceptance criteria note.

### Open Unknowns for Phase 3 dev discussion

- **U1**: Exact per-type rollup format in TOON and human output — what level of detail is desired (line counts only? also byte counts? sorted how?)? Plan-QA should surface this; dev decides in Phase 3. Builder uses a reasonable default if not addressed.
- **U2**: Should `--lang unknown` be a valid filter value to explicitly select files whose language rak could not detect? Current plan: not supported (undetected files are silently excluded when a filter is set). If dev wants to count only undetected files, this needs an explicit decision.
- **U3**: Should per-type rollup be present in TOON/human output by default, or opt-in (e.g., `--detail` flag)? Current plan: always-on in all three renderers when language detection produces a non-empty map (any file was detected, after LangUnknown suppression). No new flag. Revisit if the output gets cluttered.

### Carve-out / compile note

No carve-out unit needed. 5.1 wires `lang.Detect` into `cmd/rak` but adds no `Split` calls yet — both `internal/lang` and `cmd/rak` compile clean. 5.2 adds `Split`/`LangCounts`/`LineCounts` to `internal/lang` only — `cmd/rak` does not call them until 5.3. 5.4 adds the `--lang` filter gate in `cmd/rak` using `detectedLang` already wired by 5.1 — no `internal/lang/split.go` symbols needed. The tree compiles after each unit independently.
