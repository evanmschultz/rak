# DROP_9 — RELEASE_DOCS

**State:** building
**Tier:** B (mixed; 9.4 + 9.5 are C — dev-manual)
**Blocked by:** DROP_8
**Paths (expected):** `main/internal/summary/summary.go` (add `TotalByLang` field), `main/internal/render/render.go` (Renderer.RenderTree signature grows), `main/internal/render/{toon,human,json}.go` (emit per-lang totals), `main/internal/render/render_test.go` (extend), `main/cmd/rak/root.go` (aggregate TotalByLang in walkAndCount), `main/cmd/rak/root_test.go`, `main/cmd/rak/main.go` (fang.WithVersion), `main/README.md`, `main/magefile.go`, `main/.github/workflows/ci.yml`
**Packages (expected):** `github.com/evanmschultz/rak/internal/summary`, `github.com/evanmschultz/rak/internal/render`, `github.com/evanmschultz/rak/cmd/rak`
**PLAN.md ref:** main/PLAN.md → `DROP_9_RELEASE_DOCS` row
**Workflow:** main/drops/WORKFLOW.md
**Started:** 2026-05-15
**Closed:** —

## Scope

Final drop for v0.1.0. Six units, slimmed per decision 30:
- **9.0** (NEW pre-9.1) — Per-language totals across all dirs. The `wc++ for LLMs` framing makes "how much Go vs Markdown in this repo" a first-class question; the existing per-dir × per-lang detail is good but doesn't aggregate. Add `Summary.TotalByLang` + render block.
- **9.1** README rewrite (orch-direct, tier C).
- **9.2** `--version` via `fang.WithVersion` (tier B builder).
- **9.3** Flip `mage coverage` to a 70% floor gate (tier B builder).
- **9.4** Flip repo public (tier C dev-manual).
- **9.5** Tag `v0.1.0` + push tag (tier C dev-manual).

Drop 9 close = v0.1.0 ship.

## Planner

Orch wrote this inline (tier B/C mixed — no planner subagent per WORKFLOW.md § "Cascade Tiering"). Six units.

### Unit 9.0 — Per-language totals across all dirs

- **State:** done
- **Paths:**
  - `main/internal/summary/summary.go` (add field)
  - `main/internal/render/render.go` (interface grows)
  - `main/internal/render/{toon,human,json}.go` (emit new block)
  - `main/internal/render/render_test.go` (extend)
  - `main/cmd/rak/root.go` (aggregate)
  - `main/cmd/rak/root_test.go`
- **Packages:** `github.com/evanmschultz/rak/internal/summary`, `github.com/evanmschultz/rak/internal/render`, `github.com/evanmschultz/rak/cmd/rak`
- **Acceptance:**
  - `summary.Summary` grows a `TotalByLang map[lang.Language]lang.LangCounts` field — per-language aggregate across all directories. Field order: `Dirs`, `Total`, `TotalByLang`. Doc comment notes nil = no detection, LangUnknown suppression is the renderer's responsibility (mirrors F33).
  - `Renderer.RenderTree` signature grows to accept the per-lang totals. Pick ONE of:
    - **Option A (recommended):** change signature to `RenderTree(w io.Writer, s summary.Summary, errs []error) error` — collapses `dirs`/`total`/new field into one param. Caller passes a constructed `summary.Summary{Dirs:..., Total:..., TotalByLang:...}`. F25/F32 authorized signature change (no external implementers under `internal/`).
    - **Option B (additive):** add `totalByLang map[lang.Language]lang.LangCounts` as a 5th param. Backward-compat-ish but no real benefit pre-v1.0. Builder picks; document in worklog.
  - All three renderer `RenderTree` implementations updated. Emit a per-lang totals block AFTER the existing per-dir totals, BEFORE the `errors` block (if present). Per-renderer shape:
    - **TOON:** new tabular array `total_by_lang[N|]{lang|blank|comment|code|bytes|lines}:` with one row per language (sorted alphabetically by language string, F33 LangUnknown suppression applied).
    - **JSON:** new field `total_by_lang: { "go": {Lines:{...}, Counts:{...}}, "markdown": {...} }`. `omitempty` for nil/empty. `directoryJSON` mirror — wait, no, this is on the TOP-LEVEL Summary, not per-Directory. So a `summaryJSON` struct (if exists) grows, OR the JSON output's top-level object grows. Whichever exists.
    - **Human:** new section after the existing `total` block:
      ```
      total by language
        go        Blank 813   Comment 1873  Code 4774   Bytes 242400  Lines 7177
        markdown  Blank 1764  Comment 5     Code 5465   Bytes 1044042 Lines 7434
      ```
      Sorted alphabetically by language, LangUnknown suppressed.
  - `cmd/rak/root.go` `walkAndCount` aggregates `TotalByLang` during the per-file loop (alongside the existing `byDirLang` accumulation). Each accepted file: `totalByLang[detectedLang].Add(langCounts)`. Skip `LangUnknown` keys (or include them but let renderers suppress — match the existing F33 pattern).
  - F33 LangUnknown suppression applies uniformly across all three renderers' new block.
  - Tests:
    - `TestRenderer_TotalByLang_TOON` — fixture with 2 Go files + 1 Markdown file; verify TOON output contains `total_by_lang` tabular array with both languages.
    - Same for JSON and human renderers.
    - `TestRenderer_TotalByLang_LangUnknownSuppressed` — fixture with only LangUnknown files; verify output does NOT contain `total_by_lang` block (or has an empty one).
    - `TestRootCmd_TotalByLang_EndToEnd` — fstest.MapFS with Go and Python files; verify the aggregate counts match the sum of per-dir per-lang detail.
  - `mage ci` green.
- **Blocked by:** —
- **Tier B** (builder + falsification-only QA).

### Unit 9.1 — README rewrite

- **State:** todo
- **Paths:** `main/README.md`
- **Packages:** — (markdown only)
- **Acceptance:**
  - Elevator pitch (1 paragraph): rak = wc++ for LLM-first consumption. Walk a directory, count bytes/lines/words/chars/files, detect languages, split blank/comment/code, render compact TOON output by default.
  - Sections: Install (`go install`), Quick examples (with the new per-lang totals visible in sample output), Flags reference, Decisions / Scope (link main/PLAN.md decisions + v0.2 deferred list), License (MIT).
  - Replace Drop 0 stub aspirational text. Examples use the ACTUAL output from a `mage install` + `rak` run, not hypothetical wording.
- **Blocked by:** 9.0 (so examples can show per-lang totals).
- **Tier C** (orch-direct markdown).

### Unit 9.2 — `--version` via fang.WithVersion

- **State:** done
- **Paths:** `main/cmd/rak/main.go`, `main/cmd/rak/root_test.go`
- **Packages:** `github.com/evanmschultz/rak/cmd/rak`
- **Acceptance:**
  - `cmd/rak/main.go` updated: pass `fang.WithVersion("v0.1.0")` (or hardcoded const) into `fang.Execute`. Existing `fang.WithNotifySignal` call preserved.
  - `rak --version` prints `v0.1.0` (or the fang-default format wrapping it).
  - `root_test.go`: add `TestRootCmd_Version` asserting output contains `v0.1.0`.
  - `mage ci` green.
- **Blocked by:** —
- **Tier B** (builder + falsification-only QA).

### Unit 9.3 — Flip mage coverage to a 70% floor gate

- **State:** done
- **Paths:** `main/magefile.go`, `main/.github/workflows/ci.yml`
- **Packages:** — (build automation only)
- **Acceptance:**
  - `mage coverage` updated to ALSO enforce a 70% floor on the `-coverpkg=./internal/...` scope (excludes `cmd/rak` CLI wiring per decision 22). Parse `coverage.out` (or `go tool cover -func` output) for the `total:` line, compare to 70.0, exit non-zero if below.
  - `mage ci` includes the floor check (call `mage coverage` from `mage ci`, OR fold the check into `mage ci` directly).
  - `.github/workflows/ci.yml` runs the new `mage ci` (no additional steps needed if mage ci already invokes coverage).
  - **Verify local coverage is currently at or above 70%.** If below, document the gap in worklog and either raise tests OR adjust scope.
  - `mage ci` green from `main/`.
- **Blocked by:** —
- **Tier B** (builder + falsification-only QA).

### Unit 9.4 — Flip repo public

- **State:** todo
- **Paths:** — (no code; GitHub repo settings)
- **Packages:** —
- **Acceptance:**
  - Dev flips `github.com/evanmschultz/rak` from private to public via the GitHub UI.
  - Orch verifies post-flip via `gh repo view evanmschultz/rak --json visibility` returning `{"visibility":"public"}`.
  - CI on the public repo still runs green.
- **Blocked by:** 9.0, 9.1, 9.2, 9.3.
- **Tier C** (dev-manual; orch coordinates + verifies).

### Unit 9.5 — Tag v0.1.0 + push tag

- **State:** todo
- **Paths:** — (git tag only)
- **Packages:** —
- **Acceptance:**
  - `git tag v0.1.0` against the close commit of 9.4.
  - `git push origin v0.1.0`.
  - `gh release list` shows the new tag (if a Release is auto-created by tag-push) OR `git ls-remote --tags origin` shows it.
  - The tag points at the exact commit the README + per-lang totals + --version + coverage gate ship at.
- **Blocked by:** 9.4.
- **Tier C** (dev-manual; orch coordinates).

## Notes

### F-pin for 9.0

- **F46 — `Summary.TotalByLang` aggregation:** computed in `walkAndCount` during the per-file accept block, alongside `byDirLang`. The `LangCounts.Add` helper from Drop 7 5.2 does the field-wise accumulation. LangUnknown keys may be retained in the map or filtered at construction time — renderer-level F33 suppression handles emission regardless.

### Layout decision (per dev confirmation 2026-05-15)

TOON: separate `by_lang` tabular array remains; new `total_by_lang` tabular array added after `total`. Human: per-dir blocks with inlined `lang:` sub-blocks remain; new `total by language` section added after the existing `total` block. JSON: structural `total_by_lang` field added at the top level. All three renderers carry the same data in their natural shape.

### Tier mix rationale

9.0 = tier B (real Go code + tests, ships before v0.1.0). 9.1 = tier C (markdown). 9.2 + 9.3 = tier B (Go + magefile). 9.4 + 9.5 = tier C (dev-manual GitHub operations).

### v0.1.0 ship criteria

After 9.5: rak is shipped at v0.1.0. Users can `go install github.com/evanmschultz/rak/cmd/rak@v0.1.0` and get: walk, gitignore + git-tracked enumeration, language detection + split, --lang filter, --sort + --sort-asc, --max-files safety rail, TOON/human/JSON output with per-language totals, --version. Drop 9 close = release.

### v0.2 follow-ups carried in main/PLAN.md

Per main/PLAN.md "Follow-Ups": (a) Node.js 20 actions deprecation in CI workflow, (b) symlinked walk-root normalization, (c) default-TOON path-arg integration coverage, (d) render.go package doc mention of NewTOONRenderer, (e) `runDirectory` 10-param refactor. None are v0.1.0 blockers.

### Open Unknowns

- **U1** — Coverage current state: Unit 9.3 builder reports the actual percentage after `mage coverage`. If below 70%, scope-adjust or test-raise.
- **U2** — Version tag derivation: hardcoded `"v0.1.0"` vs build-time `-ldflags`. v0.1.0 uses hardcoded — simpler, no GoReleaser yet. Document.
- **U3** — `RenderTree` signature: builder picks Option A (`summary.Summary`) vs Option B (additive 5th param). Trade-off documented in worklog.
