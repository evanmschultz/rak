# DROP_7 — SUMMARY_SORTING

**State:** done
**Tier:** A
**Blocked by:** DROP_6
**Paths (expected):** `main/internal/summary/` (new package — `Summary` struct + sort functions + tests), `main/internal/render/render.go` (migrate provisional `render.Directory` to `summary.Summary` or have render consume `summary.Summary` directly), `main/internal/render/{human,json,toon}.go` (update consumers), `main/internal/render/render_test.go` (extend), `main/cmd/rak/root.go` (add `--sort` and `--sort-asc` flags; apply sort to the directories slice before rendering), `main/cmd/rak/root_test.go` (sort behavior tests)
**Packages (expected):** `github.com/evanmschultz/rak/internal/summary` (new), `github.com/evanmschultz/rak/internal/render`, `github.com/evanmschultz/rak/cmd/rak`
**PLAN.md ref:** main/PLAN.md → `DROP_7_SUMMARY_SORTING` row
**Workflow:** main/drops/WORKFLOW.md
**Started:** 2026-05-15
**Closed:** 2026-05-15 (CI run 25939674606 green; Hylla ingest task task-2ca675dc1803694b at commit 92d5a07)

## Scope

Land `internal/summary` as the canonical home for rollup data and add `--sort` to surface the per-directory listing in the user's preferred order. The Drop 3 `render.Directory` was provisional per planner pin C8 (carried as a v0.1.0 stand-in); Drop 7 migrates it into `summary.Summary` (or `summary.Directory`) and updates all three renderers + `cmd/rak`'s `walkAndCount` accumulator to produce the new shape. The migration is mechanical for the Drop 4/5 spine (TOON/human/JSON renderers, per-language ByLang map) — all the existing shape stays; only the type's location changes.

Sort surface: `--sort {lines,files,bytes,path}` selects the key; `--sort-asc` flips direction. Defaults are **key-specific** per Round 2 dev decision C1: numeric keys (`lines`, `files`, `bytes`) default desc (decision 19); path key defaults asc (matches `ls` convention). The sort applies to the directories slice ONLY (per-language rollup inside each directory remains alphabetically sorted by language string per F33 / Drop 5 deterministic-order convention). `tokens` is NOT a sort key in v0.1.0 — decision 30 cut tokens to v0.2; `--sort tokens` is rejected with an explicit error per F41.

Drop 5 spine preserved: `internal/lang` Detect + Split + LangCounts unchanged; F26 RelPath invariant; F33 LangUnknown suppression; cobra `--human` / `--json` / `--toon` / `--lang` / `--include` / `--exclude` / `--depth` / `--hidden` / `--no-gitignore` / `--binary` all unchanged. Renderer interface (F25/F32) may grow if necessary; planner decides per same dep-edge reasoning as Drop 5.

## Planner

Three units, fully serialized (7.1 → 7.2 → 7.3). No parallelism is possible: each unit's output is a prerequisite for the next. Drop 5 ended at F34; new pins in this drop are F35–F43.

### Specify

**Objective:** Introduce `internal/summary` as the canonical home for per-directory rollup data (migrating Drop 3's provisional `render.Directory`) and add `--sort`/`--sort-asc` flags so the user can control directory listing order. All three renderers (TOON, human, JSON) are updated to consume `summary.Directory`. No new external deps; sort uses `slices.SortFunc` from stdlib (Go 1.21+, project requires 1.26+).

**AcceptanceCriteria:**
- `internal/summary` package compiles with `Directory`, `Summary`, `SortKey`, and `SortDirs` exported; `mage test` passes for `github.com/evanmschultz/rak/internal/summary` (F35, F36).
- `render.Directory` is deleted; all three renderers and `cmd/rak/root.go` compile against `summary.Directory`; `mage build` and `mage test` pass for `internal/render` and `cmd/rak` (F37). `summary.Directory` and `directoryJSON` declare fields in identical order (F43).
- `--sort {lines,files,bytes,path}` and `--sort-asc` flags are wired; default direction is KEY-SPECIFIC (numeric keys `lines`/`files`/`bytes` default desc; `path` defaults asc per Decision 19 / C1); `--sort-asc` flips whichever default applies; `mage test` passes for `cmd/rak` covering all four sort keys in both default and flipped directions (F38, F39).
- `mage ci` passes clean on the full tree after all three units land (F40).
- `tokens` is NOT a valid `--sort` key; the omission is documented in a doc comment on `SortKey` (F41). An unrecognized `--sort` value (including `tokens`) returns an explicit error with canonical text `"\"X\" is not a valid sort key; valid keys: lines, files, bytes, path"` rather than silently falling back (Decision 3.4). The Unit 7.3 acceptance bullet at line 137 is the operative canonical form — any other site in the plan that quotes the message MUST match this exact wording.
- `summary.Directory.Files` carries the per-directory file count; `walkAndCount` increments it per accepted file; the `files` sort key uses this field (F42).

**ValidationPlan:**
- Unit 7.1: `mage build` + `mage test` scoped to `./internal/summary/...`
- Unit 7.2: `mage build` + `mage test` scoped to `./internal/render/...` + `./cmd/rak/...`
- Unit 7.3: `mage build` + `mage test` scoped to `./cmd/rak/...`
- Drop-end: `mage ci` (gofumpt + lint + test -race ./...) then `git push` + `gh run watch --exit-status`

**RiskNotes:**
- `directoryJSON` in `json.go` currently uses bare struct conversion `directoryJSON(filterUnknown(d))`. Adding `Files int64` to `summary.Directory` breaks this unless `directoryJSON` gets a matching `Files int64` field (with `json:"files,omitempty"` to keep the zero-value transparent in snapshot tests). Builder must update `directoryJSON` and any snapshot strings that would change (F34 update).
- `render_test.go` snapshot tests may pin exact JSON output. The `json:"files,omitempty"` tag on `directoryJSON.Files` lets zero-count dirs remain invisible in existing snapshots — but tests that construct `summary.Directory` with non-zero `Files` will produce new JSON output. Builder must audit all snapshot strings.
- The `Renderer` interface's `RenderTree` signature changes from `[]render.Directory` to `[]summary.Directory`. This is authorized by F15 (no external implementers under `internal/`; pre-v1.0). The signature change pins as F37.
- `slices.SortFunc` modifies in place; `SortDirs` should document this clearly so callers don't expect a copy.

**ContextBlocks:**
- constraint (critical): F25/F32 — Renderer interface may GROW additively (add new methods) but must NOT remove or rename existing methods. Unit 7.2 changes a parameter type, not method names; F15 authorizes this.
- constraint (critical): F26 — RelPath invariant unchanged; `summary.Directory.Path` follows the same forward-slash convention as `render.Directory.Path`.
- constraint (high): F33 — LangUnknown suppression is the renderer's responsibility; `summary.Directory.ByLang` retains LangUnknown and renderers filter it before emission. No change.
- constraint (high): F34 — `directoryJSON` mirrors `Directory` for struct conversion. Unit 7.2 must update `directoryJSON` to match the new `summary.Directory` shape including `Files`.
- decision (normal): `tokens` is not a sort key in v0.1.0 per Decision 30. Document the omission on the `SortKey` type. Future drops re-add via `SortTokens` constant.
- decision (normal): sort direction default is KEY-SPECIFIC per Decision 19 / C1. Numeric keys (`lines`, `files`, `bytes`) default descending; `path` defaults ascending (A→Z, matches `ls`). `--sort-asc` flips whichever default applies. The default `--sort` value is `"lines"` and `--sort-asc` defaults `false`. Implementation: `SortDirs` resolves `effectiveAsc(key, asc bool)` — numeric keys return `asc`, path key returns `!asc` (i.e., reverses the flag's meaning so the user's intuition holds).
- decision (normal): `--sort` with an unrecognized value (e.g. `tokens`, `foo`) returns an explicit cobra-level validation error "X is not a valid sort key; valid keys: lines, files, bytes, path" (Decision 3.4). No silent fallback.
- decision (normal): the per-language ordering within a directory stays alphabetical (F33 convention) regardless of `--sort`. Sort applies only to the top-level directories slice.
- constraint (critical): sort applies AFTER `labelDirectories` (Decision 3.3). `runDirectory` call order is `labelDirectories(dirs, rootLabel)` → `summary.SortDirs(dirs, key, asc)` → `renderer.RenderTree(...)`. Sort must NOT run on the raw walk-root-relative paths from `walkAndCount`.
- warning (high): `walkAndCount` currently owns the path-sort (`sort.Slice` on line 345 of root.go). Unit 7.2 migrates the return type but must REMOVE this inline sort; unit 7.3 replaces it with `summary.SortDirs`. If 7.2 removes the sort without 7.3 landing, the output order is map-iteration order (non-deterministic). 7.2 must either keep a lexical sort until 7.3 lands or the acceptance criteria must explicitly note the interim state. Decision: 7.2 KEEPS the lexical path-sort in `walkAndCount` as an interim fallback; 7.3 replaces it with the configurable sort running in `runDirectory` AFTER `labelDirectories` (Decision 3.3).

**KindPayload:**
```json
{
  "children": [
    {"kind": "build", "title": "7.1 — internal/summary package", "blocked_by": []},
    {"kind": "build", "title": "7.2 — migrate render.Directory to summary.Directory", "blocked_by": ["7.1"]},
    {"kind": "build", "title": "7.3 — --sort/--sort-asc flags + sort application", "blocked_by": ["7.2"]}
  ]
}
```

**CompletionContract:**
- StartCriteria: `internal/summary/` does not exist; `render.Directory` is the live type; no sort flags on the root command.
- CompletionCriteria: `internal/summary/` compiles with full API; all renderers compile against `summary.Directory`; `--sort`/`--sort-asc` flags exist; `mage ci` passes clean.
- CompletionChecklist:
  - [ ] `internal/summary/summary.go` and `internal/summary/sort.go` created
  - [ ] `internal/summary/summary_test.go` created
  - [ ] `render.Directory` removed; `Renderer.RenderTree` signature updated
  - [ ] All three renderer `RenderTree` implementations updated
  - [ ] `cmd/rak/root.go` `walkAndCount` return type + accumulator updated (including `Files` increment)
  - [ ] `--sort` and `--sort-asc` flags wired in `newRootCmd`
  - [ ] `runDirectory` applies `summary.SortDirs` before calling `RenderTree`
  - [ ] `mage ci` green

---

### Unit 7.1 — internal/summary: Directory struct, Summary struct, SortKey, SortDirs

- **State:** done
- **Paths:**
  - `internal/summary/summary.go` (new — not yet in tree)
  - `internal/summary/sort.go` (new — not yet in tree)
  - `internal/summary/summary_test.go` (new — not yet in tree)
- **Packages:** `github.com/evanmschultz/rak/internal/summary`
- **Acceptance:**
  - `summary.Directory` struct has fields: `Path string`, `Counts counting.Counts`, `ByLang map[lang.Language]lang.LangCounts`, `Files int64` (F35). `Files` carries the per-directory count of accepted (non-skipped) files.
  - `summary.Summary` struct has fields: `Dirs []Directory`, `Total counting.Counts` (F36).
  - `SortKey` is a named string type with constants `SortLines`, `SortFiles`, `SortBytes`, `SortPath` (string values `"lines"`, `"files"`, `"bytes"`, `"path"`). `SortName` is retired; the constant is `SortPath`. Doc comment on the type notes `tokens` is omitted per Decision 30 / v0.2 scope (F41).
  - `SortDirs(dirs []Directory, key SortKey, asc bool)` sorts `dirs` in place. `asc` is the raw flag value from the caller (`--sort-asc`). Internally `SortDirs` resolves the effective direction via a helper: numeric keys (`SortLines`, `SortFiles`, `SortBytes`) use `asc` directly (default false → descending); `SortPath` inverts: effective direction is `!asc` (default false → ascending, matching `ls`). If `key` is unrecognized, function panics with a descriptive message — callers are expected to validate the key before calling (the CLI layer validates; see Unit 7.3). Documents "modifies in place; does not return a copy." Uses `slices.SortFunc` from stdlib (F38).
  - `summary.Directory` struct fields in this exact order: `Path string`, `Counts counting.Counts`, `ByLang map[lang.Language]lang.LangCounts`, `Files int64`. Order must match `directoryJSON` exactly for bare struct conversion to compile (F43).
  - `mage build` and `mage test` pass for `./internal/summary/...`.
  - Table-driven tests cover: all four sort keys in default direction (no `--sort-asc`, i.e. `asc=false`) — `SortLines` desc, `SortFiles` desc, `SortBytes` desc, `SortPath` asc; all four keys in flipped direction (`asc=true`) — `SortLines` asc, `SortFiles` asc, `SortBytes` asc, `SortPath` desc; zero-length slice (no panic); single-entry slice (no reorder).
- **Blocked by:** —

### Unit 7.2 — Migrate render.Directory to summary.Directory

- **State:** done
- **Paths:**
  - `internal/render/render.go`
  - `internal/render/toon.go`
  - `internal/render/human.go`
  - `internal/render/json.go`
  - `internal/render/render_test.go`
  - `cmd/rak/root.go`
- **Packages:**
  - `github.com/evanmschultz/rak/internal/render`
  - `github.com/evanmschultz/rak/cmd/rak`
- **Acceptance:**
  - `render.Directory` type is deleted from `render.go`. `Renderer.RenderTree` signature becomes `RenderTree(w io.Writer, dirs []summary.Directory, total counting.Counts, errs []error) error` (F37). `internal/render` now imports `internal/summary`.
  - All three renderer `RenderTree` implementations compile against `summary.Directory`. No behavioral change to output (same fields available: Path, Counts, ByLang — Files is ignored by renderers). LangUnknown suppression (F33) preserved. TOON `toonDirectory`, human `countsKV`, JSON `directoryJSON` all updated.
  - `directoryJSON` updated to include `Files int64 \`json:"files,omitempty"\`` so the bare struct conversion `directoryJSON(filterUnknown(d))` compiles against `summary.Directory` (F34 update). `omitempty` keeps the zero-value transparent in existing snapshots. `directoryJSON` field order must match `summary.Directory` exactly: `Path`, `Counts`, `ByLang`, `Files` (F43).
  - `cmd/rak/root.go`: `walkAndCount` return type changes from `[]render.Directory` to `[]summary.Directory`. Add a parallel `byDirFiles map[string]int64` accumulator alongside the existing `byDir map[string]counting.Counts` (since `byDir`'s value is `Counts`, not `Directory`, so `dir.Files++` has no `dir` variable to bind to in the per-file loop). Increment `byDirFiles[dirKey]++` per accepted file (post-binary-skip, post-`--lang` filter — same "accepted file" condition as `byDir`). After the loop, when constructing each `summary.Directory`, populate `Files: byDirFiles[path]`. `labelDirectories` updated to return `[]summary.Directory` AND propagate `Files` through reconstruction: `summary.Directory{Path: ..., Counts: d.Counts, ByLang: d.ByLang, Files: d.Files}` (F44). Inline path-sort in `walkAndCount` RETAINED as interim fallback (will be replaced by 7.3). `runDirectory` updated to pass `[]summary.Directory` to `RenderTree`.
  - **F44 — Files propagation through reconstruction sites (Round 2 falsification C2.1):** Two sites in the codebase reconstruct `Directory` values from existing instances. BOTH must propagate the new `Files` field or production behavior degrades silently:
    - `cmd/rak/root.go.labelDirectories` — rebuilds `Directory{Path: <labeled>, Counts: d.Counts, ByLang: d.ByLang, Files: d.Files}`. **Must carry `Files`**, otherwise `SortDirs --sort files` produces degenerate order and JSON `files` field is omitted due to `omitempty`.
    - `internal/render/json.go.filterUnknown` — rebuilds `Directory{Path: d.Path, Counts: d.Counts, ByLang: <filtered>, Files: d.Files}`. **Must carry `Files`**, same failure mode.
  - **Test required for F44 (Round 2 falsification C2.1):** add `TestRootCmd_SortFiles_NonDegenerate` to `cmd/rak/root_test.go` using a fixture with multiple directories of differing file counts (e.g. 2 files in `./`, 5 files in `./sub`); run with `--sort files` and assert the per-dir ordering reflects the actual file counts (not all-zero). This test ONLY passes if `Files` survives `labelDirectories` reconstruction. Add a parallel JSON-output test asserting `"files": <n>` appears in the rendered output for non-zero directories.
  - `render_test.go` snapshot strings unchanged for zero-Files cases (omitempty handles it). Any test that constructs a `summary.Directory` with non-zero `Files` and checks JSON output must be updated.
  - `mage build` and `mage test` pass for `./internal/render/...` and `./cmd/rak/...`.
- **Blocked by:** 7.1

### Unit 7.3 — --sort / --sort-asc flags + sort application

- **State:** todo
- **Paths:**
  - `cmd/rak/root.go`
  - `cmd/rak/root_test.go`
- **Packages:** `github.com/evanmschultz/rak/cmd/rak`
- **Acceptance:**
  - `rootFlags` gains `sort string` (default `"lines"`) and `sortAsc bool` (default `false`) fields (F38, F39).
  - `newRootCmd` binds `--sort` with usage `"sort directories by key: lines, files, bytes, path (default: lines; numeric keys default descending, path defaults ascending)"` and `--sort-asc` with usage `"flip sort direction from its key-specific default"`. Both flags are optional; omitting them gives `lines desc` per Decision 19 / C1.
  - A `cobra.Args`-style (or `PersistentPreRunE`) validator in `newRootCmd` rejects unrecognized `--sort` values with an explicit error: `"\"X\" is not a valid sort key; valid keys: lines, files, bytes, path"` (Decision 3.4). This fires before `RunE` so no sort is attempted.
  - `runDirectory` calls in order: (1) `labelDirectories(dirs, rootLabel)`, (2) `summary.SortDirs(dirs, summary.SortKey(flags.sort), flags.sortAsc)`, (3) `renderer.RenderTree(...)`. The inline lexical sort from Unit 7.2 is removed from `walkAndCount` at this point (Decision 3.3, F39).
  - `root_test.go` table-driven tests cover:
    - Default (no flags) → `lines desc`
    - `--sort lines` (no `--sort-asc`) → desc
    - `--sort lines --sort-asc` → asc
    - `--sort files` (no `--sort-asc`) → desc
    - `--sort files --sort-asc` → asc
    - `--sort bytes` (no `--sort-asc`) → desc
    - `--sort bytes --sort-asc` → asc
    - `--sort path` (no `--sort-asc`) → asc (key-specific default per C1)
    - `--sort path --sort-asc` → desc (flipped from default)
    - `TestRootCmd_SortTokens_Errors`: `--sort tokens` returns an error (Decision 3.4)
    Tests use a synthetic `[]summary.Directory` slice with distinct values per sort key to verify ordering.
  - `mage build` and `mage test` pass for `./cmd/rak/...`. `mage ci` passes clean on the full tree.
- **Blocked by:** 7.2

## Notes

**F-pin range for this drop: F35–F44.**
- F35: `summary.Directory` struct shape (Path, Counts, ByLang, Files — in that order).
- F36: `summary.Summary` struct shape (Dirs, Total).
- F37: `Renderer.RenderTree` signature updated to `[]summary.Directory` parameter.
- F38: `--sort {lines,files,bytes,path}` flag; default "lines". `SortKey` constant is `SortPath` (not `SortName`); `--sort path` is the CLI value.
- F39: `--sort-asc` flag; default false. Direction is KEY-SPECIFIC: numeric keys (lines/files/bytes) default desc; `path` defaults asc. `--sort-asc` flips the key's natural default. Sort runs in `runDirectory` AFTER `labelDirectories`, BEFORE `RenderTree` (Decision 3.3).
- F40: `mage ci` gate passes after all three units.
- F41: `tokens` omitted from `SortKey` per Decision 30; documented on type. Unrecognized `--sort` values return explicit error (Decision 3.4).
- F42: `summary.Directory.Files int64` carries per-directory file count; `walkAndCount` increments it.
- F43: `summary.Directory` and `directoryJSON` declare fields in identical order (Path, Counts, ByLang, Files) so bare struct conversion compiles.
- F44: `Files` field MUST propagate through all `Directory` reconstruction sites. Specifically `cmd/rak/root.go.labelDirectories` and `internal/render/json.go.filterUnknown` BOTH rebuild `Directory` values from existing instances; both MUST carry `Files: d.Files` or `--sort files` produces degenerate order and JSON omits the field via `omitempty`. Test `TestRootCmd_SortFiles_NonDegenerate` in 7.3 covers this end-to-end. (Round 2 falsification C2.1.)

**Dependency note:** the three units are fully serialized. No parallelism is achievable — the type migration in 7.2 is prerequisite to the flag wiring in 7.3, and both require the `internal/summary` package from 7.1.

**Warning (re: Drop 4 carve-out discipline):** Unit 7.2 makes `cmd/rak` import `internal/summary`. If `internal/summary` compiles cleanly (it will, since 7.1 must complete first), this is not a carve-out scenario. Lint cleanliness carries through from 7.1 completion.

**Warning (re: interim sort state):** between unit 7.2 landing and unit 7.3 landing, the output sort order is lexical by path (the interim fallback retained in `walkAndCount`). This is deterministic and matches prior behavior — acceptable for the brief inter-unit window during development.
