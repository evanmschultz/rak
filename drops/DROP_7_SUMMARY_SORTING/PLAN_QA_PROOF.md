# DROP_7 — Plan QA Proof

## Drop 7 — Plan QA Proof Round 2

**Verdict:** PASS.

**Scope of review:** delta-only — Round 1 produced 2 CEs (C1 sort-direction defaults, C2 `SortName`→`SortPath`) plus 3 sub-pins (3.2 field order, 3.3 sort-after-label call order, 3.4 explicit error for `--sort tokens`). Dev/orch decisions: key-specific defaults (numeric desc, path asc), full sweep of the `SortName` rename including the orch-edited Scope paragraph, explicit cobra-level validation error for unrecognized `--sort` values. Round 2 audit verifies each delta lands in the revised PLAN.md.

**Audit:**

- **C1 — key-specific sort-direction defaults.** Pinned at Scope ¶17 (numeric desc, path asc), AcceptanceCriteria line 32 (KEY-SPECIFIC default direction with Decision 19 / C1 reference), ContextBlocks decision line 55 (explicit `effectiveAsc(key, asc)` helper — numeric returns `asc`, `SortPath` returns `!asc`), Unit 7.1 acceptance line 99 (helper resolved inside `SortDirs`), Unit 7.1 tests line 102 (all four keys in both directions including SortPath asc/desc), F39 (KEY-SPECIFIC default direction recorded), Unit 7.3 tests lines 147–148 (`--sort path` → asc; `--sort path --sort-asc` → desc). Five-plus independent pins. PASS.

- **C2 — `SortName`/`--sort name` retired in favor of `SortPath`/`--sort path`.** Full sweep verified: zero residual `SortName` or `--sort name` references in PLAN.md. Explicit retirement noted at Unit 7.1 acceptance line 98 ("`SortName` is retired; the constant is `SortPath`") and F38 line 160 ("`SortKey` constant is `SortPath` (not `SortName`); `--sort path` is the CLI value"). Orch-edited Scope ¶17 reads `--sort {lines,files,bytes,path}` with path-key default-asc — clean. All flag bindings, usage strings, and tests use `path`. PASS.

- **3.2 — field order pinned in `summary.Directory` and `directoryJSON`.** Pinned at AcceptanceCriteria line 31 (identical declaration order required for bare struct conversion), Unit 7.1 acceptance line 100 (exact field order: `Path`, `Counts`, `ByLang`, `Files`), Unit 7.2 acceptance line 121 (`directoryJSON` must match exactly), and F43 (dedicated F-pin). Compile-level enforcement via `directoryJSON(filterUnknown(d))` bare conversion — `mage build` catches drift. PASS.

- **3.3 — sort applies AFTER `labelDirectories`.** Pinned at ContextBlocks critical-constraint line 58 (explicit three-step `runDirectory` call order: `labelDirectories` → `summary.SortDirs` → `RenderTree`; explicit "sort must NOT run on raw walk-root-relative paths"), Unit 7.3 acceptance line 138 (same three-step order numbered), and F39 line 161 (sort runs AFTER `labelDirectories`, BEFORE `RenderTree`). The interim 7.2→7.3 window keeps the lexical sort in `walkAndCount` as fallback; 7.3 removes it and installs the configurable sort in `runDirectory`. PASS.

- **3.4 — explicit error for unrecognized `--sort` values.** Pinned at AcceptanceCriteria line 34 (explicit error, not silent fallback), ContextBlocks decision line 56 (full error string with valid-key list, cobra-level validator), Unit 7.3 acceptance line 137 (`PersistentPreRunE` validator fires before `RunE`), Unit 7.3 test row line 149 (`TestRootCmd_SortTokens_Errors` named — `--sort tokens` returns an error), and F41 (unrecognized values return explicit error). PASS.

**Findings:** none. No new CEs. No regressions from Round 1's already-mitigated territory. Plan is ready for build.

**Advisory (not a finding):** the error-message phrasing differs slightly between AcceptanceCriteria line 34 (`"X is not a valid sort key"`) and ContextBlocks line 56 / Unit 7.3 line 137 (`"\"X\" is not a valid sort key; valid keys: lines, files, bytes, path"`). Unit 7.3 acceptance is operative (it's where the test asserts); the builder should follow it. Noise, not a blocker.

**Hylla feedback:** N/A — plan-QA review touched markdown only.
