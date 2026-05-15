# DROP_7 — Builder Worklog

Append a `## Unit N.M — Round K` section per build attempt. See `main/drops/WORKFLOW.md` § "Phase 4 — Build (per unit)" for what each section should contain.

## Unit 7.2 — Round 1

- **Builder:** go-builder-agent
- **Started:** 2026-05-15
- **Files touched:**
  - `internal/render/render.go` — deleted `render.Directory` struct; updated `Renderer.RenderTree` signature to `[]summary.Directory`; swapped `lang` import for `summary`
  - `internal/render/toon.go` — added `summary` import; `RenderTree` param type → `[]summary.Directory`
  - `internal/render/human.go` — added `summary` import; `RenderTree` param type → `[]summary.Directory`
  - `internal/render/json.go` — added `summary` import; `directoryJSON` gains `Files int64 \`json:"files,omitempty"\`` (F43); `filterUnknown` takes/returns `summary.Directory` with `Files: d.Files` (F44); `RenderTree` param type → `[]summary.Directory`
  - `internal/render/render_test.go` — added `summary` import; all `[]Directory{` → `[]summary.Directory{`
  - `cmd/rak/root.go` — added `summary` import; `walkAndCount` return type → `[]summary.Directory`; `byDirFiles map[string]int64` accumulator added; `byDirFiles[dir]++` per accepted file; `summary.Directory{Files: byDirFiles[p]}` in construction; `labelDirectories` updated to `[]summary.Directory` with `Files: d.Files` propagation (F44 site 2)
  - `cmd/rak/root_test.go` — added `TestRootCmd_FilesField_SurvivesLabelDirectories` (F44 test)
- **Mage targets run:** `mage build` (pass), `mage test` (pass — all 8 packages green), `mage ci` (pass — gofumpt clean, lint 0 issues, test -race green)
- **Notes:**
  - F43 satisfied: `directoryJSON` field order (Path, Counts, ByLang, Files) matches `summary.Directory` exactly; bare struct conversion `directoryJSON(filterUnknown(d))` compiles.
  - F44 covered at both reconstruction sites: `labelDirectories` (root.go) and `filterUnknown` (json.go) both carry `Files: d.Files`.
  - `omitempty` on `directoryJSON.Files` keeps all existing snapshot tests unchanged — zero-Files dirs remain invisible in JSON output.
  - Interim path-sort in `walkAndCount` retained per Decision (7.3 will replace with `summary.SortDirs`).
  - Decision: omitted `Files` from toon and human renderer output for v0.1.0 — the per-language ByLang section already carries the LLM-interesting detail; adding a raw file count would clutter the human format. `Files` is available in JSON output and via `--sort files` (Unit 7.3).
  - `lang` import removed from `render.go` (was only needed for the now-deleted `Directory.ByLang` field); `human.go` and `toon.go` retain their own `lang` imports for `sortedKnownLangs`.

## Unit 7.1 — Round 1

- **Builder:** go-builder-agent
- **Started:** 2026-05-15
- **Files touched:**
  - `internal/summary/summary.go` (new, ~50 LOC)
  - `internal/summary/sort.go` (new, ~80 LOC)
  - `internal/summary/summary_test.go` (new, ~130 LOC)
- **Mage targets run:** `mage build` (pass), `mage test` (pass, all 8 packages green), `mage ci` (pass, gofumpt clean + lint clean + test -race green)
- **Notes:**
  - `effectiveAsc` helper resolves key-specific direction: numeric keys pass `asc` straight through (default false → descending); `SortPath` returns `!asc` (default false → effective ascending, matching `ls` convention).
  - Used `slices.SortFunc` + `cmp.Compare` + `strings.Compare` from stdlib (Go 1.21+; project on 1.26.1).
  - Unknown `SortKey` panics via a `switch default` branch with `fmt.Sprintf` message; the CLI layer validates before calling (Unit 7.3).
  - `Directory` field order (Path, Counts, ByLang, Files) matches F43 requirement for bare struct conversion in `directoryJSON` (Unit 7.2 scope).
  - `mage test` does not accept a package-path argument — target runs `go test -race ./...`; summary package green in the full run.
