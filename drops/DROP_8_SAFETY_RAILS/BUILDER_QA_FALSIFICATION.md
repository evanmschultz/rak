# Drop 8 — Builder QA Falsification

Tier B for every unit unless explicitly upgraded. Each round appends below; older rounds are kept verbatim for audit.

## Unit 8.1 — Round 1

**Verdict: PASS** (no counterexamples; zero blocking findings; one non-blocking smell)

### Attack-by-attack

| # | Attack | Outcome | Evidence |
|---|---|---|---|
| 1 | F45 `%w` sentinel chain | REFUTED (correct) | `root.go:400` — `fmt.Errorf("rak: file count exceeded --max-files %d: %w", maxFiles, ErrMaxFilesExceeded)`; verifier test at `root_test.go:996` asserts `errors.Is(err, ErrMaxFilesExceeded)`. |
| 2 | Abort timing — must be AFTER binary-skip + lang filter | REFUTED (correct) | Filter chain at `root.go:341-366` (binary skip, lang detect, lang filter, split, countFile) gates BEFORE the `acceptedFiles++` at line 394 and the check at 399. Filtered-out files never increment the counter. |
| 3 | Off-by-one `>=` semantics — 3rd accepted file aborts on `--max-files 3`, not 4th | REFUTED (correct) | `root.go:394` pre-increments, then `root.go:399` checks `acceptedFiles >= maxFiles`. Trace: file1 → 1 (false), file2 → 2 (false), file3 → 3 (true → abort). Matches spec verbatim. Test `TestRootCmd_MaxFiles_AtLimit_Aborts` confirms with 5-file fixture + cap=3. |
| 4 | Negative values handled cleanly | REFUTED (correct) | Guard `maxFiles > 0` at `root.go:399`; negatives bypass the abort branch. Test `TestRootCmd_MaxFiles_NegativeValue` (`-1` → no abort, all 5 files counted). |
| 5 | Pre vs post-increment matches comparison operator | REFUTED (correct) | Pre-increment (`acceptedFiles++` then `>=` check) is the off-by-one-safe pairing for "abort at limit, not after." No drift. |
| 6 | Abort is FATAL, not appended to aggErrs | REFUTED (correct) | `root.go:400` returns immediately with the sentinel as the 4th return value (the fatal `err`). `runDirectory` at `root.go:251-253` propagates fatal err via `return err`. Not buffered into `aggErrs`. |
| 7 | `runDirectory` 10-param signature is a smell | NOTED, non-blocking | Signature now `(ctx, w, source, rootLabel, binary, langs, sortKey, sortAsc, renderer, maxFiles)` — 10 params. Eventual refactor candidate (options-struct), but YAGNI for v0.1.0. Not a unit 8.1 blocker. |
| 8 | Drop 5/6/7 surface preservation | REFUTED (correct) | `git diff HEAD~1 -- internal/lang/ internal/summary/ internal/render/ internal/fileset/ internal/lister/ internal/ignore/ internal/counting/` → zero substantive lines. Only `cmd/rak/` (root.go + root_test.go) plus drop dir mds touched. |
| 9 | F44 (Files field) + F42 (per-lang rollup) preservation | REFUTED (correct) | `TestRootCmd_PerLangRollup` (root_test.go:673) and `TestRootCmd_FilesField_SurvivesLabelDirectories` (root_test.go:1058) still present; signatures updated mechanically (added `0` for maxFiles); no logic change in the F42/F44 paths. |
| 10 | `mage ci` green | REFUTED (correct) | Re-ran `mage ci` from `main/`: `0 issues.` from golangci-lint; all 8 packages `ok`; gofumpt-clean; race-detector pass. |

### Concurrency & edge probes (self-audit)

- **Race surface on `acceptedFiles`**: walker is single-goroutine pre-Drop-8.2; the counter is a stack-local int inside `walkAndCount` — no shared-state surface. EXHAUSTED.
- **Context cancellation interaction**: `ctx.Err()` path at `root.go:329-331` runs BEFORE the file-accept block; ctx-cancel and max-files cannot both fire in the same iteration. Clean. REFUTED.
- **`maxFiles=1` boundary**: file1 → `acceptedFiles=1`, check `1 >= 1` → true, abort. Spec-consistent (cap of 1 means "abort on the first accepted file"). Not tested directly, but covered by the off-by-one logic in test 4 (`AtLimit_Aborts` with cap=3). EXHAUSTED.
- **Partial-results discard correctness**: line 400 returns `nil, counting.Counts{}, nil, err` — all four returns zeroed. Caller (`runDirectory` line 251) propagates the err immediately without reading `dirs` / `total` / `aggErrs`. Discard is total. REFUTED.

### Spec-attack family (Tier B build-QA)

- **`KindPayload` vs diff drift**: PLAN.md unit 8.1 spec promises (`paths: cmd/rak/`, sentinel `ErrMaxFilesExceeded`, `--max-files` flag, F45 `errors.Is` consumer interface, abort at filter-gated count, FATAL abort). Diff delivers all six. No drift.
- **Silently dropped acceptance criteria**: every spec invariant (sentinel, wrap-with-%w, post-filter gating, FATAL abort, no-limit semantics for 0/negative) has a corresponding test in `root_test.go`. None dropped.
- **DecisionLog gaps**: builder worklog records the negative-value decision inline (treat-as-no-limit via `> 0` guard) and notes the 10-param signature smell. Adequate for v0.1.0.

### Counterexamples

None. Zero CONFIRMED counterexamples.

### Unknowns

- 10-param `runDirectory` signature is a refactor candidate (options-struct) but YAGNI for v0.1.0. Route to a future drop if/when a 6th flag lands.
- `maxFiles=1` boundary is logically covered by the off-by-one analysis but not exercised by a dedicated test. Non-blocking; the `AtLimit_Aborts` test exercises the same comparison path.

### Hylla Feedback

N/A — this round read only `cmd/rak/root.go` and `cmd/rak/root_test.go` (Go) plus the PLAN.md / WORKLOG markdown. The Go reads were targeted at known line ranges from the diff; no Hylla search was attempted. No misses.
