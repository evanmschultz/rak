# DROP_5 — Builder QA Falsification

Append a `## Unit N.M — Round K` section per build-QA falsification attempt. See `main/drops/WORKFLOW.md` § "Phase 5 — Build-QA (per unit)" for the round shape.

## Unit 5.1 — Round 1

- **Reviewer:** go-qa-falsification-agent
- **Verdict:** PASS — no CONFIRMED counterexample after 13 attack families.
- **Diff under attack:** `f159596 feat(lang): add detect pipeline, wire into walkandcount` — `internal/lang/lang.go` (new, 230 LOC), `internal/lang/lang_test.go` (new, 168 LOC), `cmd/rak/root.go` (+8 LOC; one import + 6-line wire).

### Attack outcomes

| # | Attack | Outcome | Notes |
|---|---|---|---|
| 1 | `Makefile.go` mis-classified as Makefile | REFUTED | `strings.ToLower(filepath.Base("Makefile.go"))` = `"makefile.go"`, not a key in `specialFilenames`; falls through to `.go` → LangGo. Covered by `TestDetect_SpecialFilename` row `{"Makefile.go", LangGo}`. |
| 2 | Nested-path basename (`sub/Makefile`) | REFUTED | `filepath.Base("sub/Makefile")` = `"Makefile"` → lowercase `"makefile"` → LangMakefile. Covered by test row `{"sub/Makefile", LangMakefile}`. |
| 3 | `#!/usr/bin/env python3` env-indirection | REFUTED | `detectShebang` (lang.go:163–207) detects `interp == "env"`, skips `-`-flagged args, uses `python3` as lookup key. Covered by `TestDetect_Shebang_Python`. Also handles `env -S python3` (flag-skip loop). |
| 4 | F27: Peek error must yield LangUnknown silently | REFUTED | lang.go:145–149: `buf, err := f.Peek(512); if err != nil { return LangUnknown }`. No `log`, no `fmt.Errorf`, no panic. Covered by `TestDetect_PeekError_ReturnsUnknown` using an empty `fstest.MapFS` so `Open` returns `fs.ErrNotExist`. |
| 5 | F26 RelPath invariant — no mutation | REFUTED | `f.RelPath` is read at lang.go:131 (`filepath.Base`) and :137 (`filepath.Ext`) only. No assignment anywhere in the package. The `*fileset.File` struct field is set only by `fileset.newFile` (file.go:52–58); no external write surface. |
| 6 | `_ = detectedLang` code smell | NOT A COUNTEREXAMPLE — accepted | Spec line 60 explicitly mandates "Store the resulting `Language` value in a per-file local (e.g. `detectedLang := lang.Detect(f)`)" for 5.2/5.4 to build on. The `_ = detectedLang` line with comment `// consumed by 5.3 (Split) and 5.4 (--lang filter)` suppresses the unused-variable compile error while preserving the call-site hook. Documented in BUILDER_WORKLOG.md. Acceptable. |
| 7 | Language-constant count | REFUTED | 18 constants present (LangUnknown + 17 named). Spec minimum: 15 from the named list + LangDocker + LangCMake = 17 named. All present. ".xml" → LangHTML is a builder-choice content-heuristic alias, documented in BUILDER_WORKLOG.md; spec line 43 says "Additional entries are welcome but not required." Acceptable. |
| 8 | Drop 3/4 spine preserved | REFUTED | `git diff HEAD~1 -- internal/fileset/ internal/lister/ internal/ignore/ internal/counting/ internal/render/` is empty. Only `cmd/rak/root.go` and new `internal/lang/` files were touched. |
| 9 | Case sensitivity — `MAKEFILE` (all caps) | EXHAUSTED, no counterexample | `strings.ToLower("MAKEFILE")` = `"makefile"` → LangMakefile via std-lib. The test table covers `Makefile`, `makefile`, `GNUmakefile`, but not `MAKEFILE`. This is a test-table gap, not a code defect — std-lib `strings.ToLower` is trusted. Worth widening the table in a future drop; not a falsification. |
| 10 | `mage ci` regression | REFUTED | Re-ran `mage ci` from `main/`. 0 issues; all 7 packages green (counting, fileset, ignore, lang, lister, render, cmd/rak). |
| 11 | `mage install` invocation | REFUTED | Worklog shows builder ran `mage test` and `mage ci` only. No `mage install`, no raw `go build` / `go test` / `go vet`. |
| 12 | Concurrency / goroutine leaks | REFUTED | `Detect` is pure synchronous — no goroutines, no channels, no shared mutable state. `Peek` is stateless per F4 (open-read-close per call). Safe under the future parallel walker. |
| 13 | Error swallowing | REFUTED | The single error-discard site (Peek failure at lang.go:145–149) is the F27-mandated silent fallthrough, explicitly commented. No `fmt.Errorf` without `%w` (none are emitted at all from `Detect`). |

### Edge-case probes (extra adversarial passes that produced no counterexample)

- **Empty Peek buffer**: `detectContent` guards `len(buf) == 0` → LangUnknown (lang.go:212–214). Safe.
- **`"#!"` with no interpreter body**: `firstLine[2:]` after TrimSpace yields `""`; `detectShebang` returns LangUnknown (lang.go:176–178). Safe.
- **`#! /bin/bash` (space after `#!`)**: TrimSpace + `strings.Fields` recovers `["/bin/bash"]` → basename `bash` → LangShell. Safe.
- **`#!/usr/bin/env -S python3` (env with flag)**: flag-skip loop at lang.go:194–199 skips `-S`, uses `python3`. Safe.
- **Generic-state contraband (C5)**: `Detect` returns the first concrete match OR LangUnknown. No "generic" intermediate. REFUTED.
- **Block-comment language fields in shebang table**: shebangsTable correctly excludes file-ext-only languages (Go, Rust, C, etc.) — shebang is only consulted when steps 1+2 returned LangUnknown.

### Hidden-attack residual

None survived. The single non-counterexample observation is the all-caps `MAKEFILE` test-table gap (Attack 9) — flagged as a future test-widening opportunity but explicitly NOT a falsification because `strings.ToLower` is std-lib trusted.

### Hylla Feedback

N/A — review touched the new `internal/lang/` package which would not be in the prior Hylla ingest (drop-end-only reingest per main/CLAUDE.md). Evidence-gathering went through `Read` on lang.go / lang_test.go and `git diff HEAD~1` for the deltas; no Hylla fallback miss to log because Hylla was not the right tool for an under-review uncommitted-since-ingest package.

### Certificate

- **Premises:** Unit 5.1 must (a) define `Language` + 17 named constants, (b) implement 4-step Detect pipeline per F27, (c) preserve F26 RelPath invariant, (d) silently swallow Peek errors per F27/P3, (e) wire `lang.Detect(f)` once per file into `walkAndCount` without consuming the result yet, (f) keep Drop 3/4 spine untouched, (g) pass `mage ci`.
- **Evidence:** `internal/lang/lang.go` (230 LOC, read in full); `internal/lang/lang_test.go` (168 LOC, read in full); `cmd/rak/root.go` diff (+8 LOC); `git diff HEAD~1 -- <spine>` empty; `mage ci` green on all 7 packages; BUILDER_WORKLOG.md confirms `mage test` + `mage ci` only.
- **Trace or cases:** 13 attack families + 6 edge-case probes = 19 distinct adversarial probes. All REFUTED, EXHAUSTED, or NOT-A-COUNTEREXAMPLE. Zero CONFIRMED.
- **Conclusion:** PASS. The unit survives Round 1 falsification with no unmitigated counterexample.
- **Unknowns:** None routed. One test-widening suggestion (add `MAKEFILE` all-caps to `TestDetect_SpecialFilename` table) is a minor future polish item, not a gate.
