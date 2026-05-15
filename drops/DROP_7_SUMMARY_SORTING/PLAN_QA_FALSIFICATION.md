# DROP_7 — PLAN_QA_FALSIFICATION (Round 1)

**Reviewer:** go-qa-falsification-agent
**Round:** 1
**Verdict:** FAIL (one CONFIRMED counterexample; one HIGH-severity flag-semantics ambiguity)

## Summary

Decomposition is sound at the structural level: three serialized units, correct `blocked_by` chain, mechanical type migration in 7.2 is bounded, JSON snapshot survival is correctly handled by `directoryJSON.Files json:"files,omitempty"`. Most attack families REFUTE cleanly.

Two attacks land. The first is a CONFIRMED counterexample on flag semantics that will produce a confusing CLI surface and a brittle test plan. The second is a HIGH-severity ambiguity on the `name` sort key that conflates a path with a name and should be renamed or its semantics nailed before 7.3.

## Counterexamples

### C1 — `--sort-asc bool` semantics are inverted and accept `=false` to mean ascending (CONFIRMED, HIGH)

**Claim under attack.** Plan unit 7.3 spec says: `sortAsc bool` "default `false`"; flag usage `"reverse sort direction (default: descending)"`.

**Repro.** Cobra's `BoolVar` accepts long-form `--sort-asc=true` / `--sort-asc=false`. The plan's mental model is:
- `--sort-asc` (bare flag) → `sortAsc=true` → ascending
- no flag → `sortAsc=false` → descending (the default per Decision 19)
- `--sort-asc=false` → `sortAsc=false` → descending

That much is consistent. The break is in the **usage string**: `"reverse sort direction (default: descending)"`. With `sortAsc` default `false`, passing `--sort-asc` (no value, cobra parses as `true`) flips to ascending. So the flag does NOT "reverse" direction in the verb sense — it specifies ascending direction explicitly. The usage string says "reverse sort direction" which implies it inverts whatever the current direction is — but it doesn't; it pins ascending. A user who passes `--sort-asc` while reading the usage string will be confused if they later pair it with a future `--sort tokens` (or any key whose "natural" default differs).

More concretely: passing `--sort-asc=false` is a valid cobra invocation. The plan never specifies what `--sort-asc=false` means. By default direction (descending), `--sort-asc=false` is descending — so the flag is a no-op. This is fine but undocumented and untested.

The deeper issue: the parameter to `summary.SortDirs(dirs, key, asc bool)` is named `asc` per the plan, but the plan's prose says `asc=false means descending; asc=true means ascending`. That is correct. But Unit 7.1 acceptance line 97 also says "For `SortName`, ascending is lexical A→Z" — which implies `asc=true` is A→Z. So `--sort name --sort-asc` (boolean is true) = A→Z. Default (no `--sort-asc`) = `asc=false` = descending name = Z→A. Is **descending lexical name** what the user wants by default? Decision 19 says default is `lines desc`, but it does NOT specify a default direction for `name` — and the plan inherits "desc" globally. Descending name (Z→A) is non-idiomatic; most CLI tools sort names A→Z by default.

**Severity.** HIGH. This is a UX cliff for one of the four documented sort keys.

**Repro test that will fail or surprise:**
```go
// Expected by most users:
// rak --sort name → A→Z lexical
// Actual per plan:
// rak --sort name → Z→A (because sortAsc default is false = desc, applied uniformly)
```

The plan's test plan in 7.3 line 136 says: "table-driven tests cover: ... `--sort name`, `--sort-asc` with each key". It does NOT assert which direction is the default for `name`. A reasonable test author will pick one (most likely A→Z), the implementation will produce the other, the test will fail mid-7.3.

**Resolution paths (planner picks one):**
- **(A) Key-specific defaults**: numeric keys (lines/files/bytes) default desc; name defaults asc. `--sort-asc` only meaningfully toggles the numerics; for name it's a no-op or it pins asc.
- **(B) Document the uniform-desc default**: pin "all keys default desc; --sort-asc flips" in the SortKey doc comment, in `--sort-asc` usage, and in a test case `--sort name` → Z→A. Most consistent with the current plan, ugliest UX.
- **(C) Rename `--sort-asc` to `--sort-dir {asc,desc}`**: explicit string enum, no boolean ambiguity. Most idiomatic; biggest API surface.

The plan must pick one in Round 2 and pin it in 7.1's SortKey doc comment + 7.3's test plan. Without that pin, 7.3 cannot author its tests deterministically.

### C2 — `--sort name` semantics conflate "name" with "path" (CONFIRMED, MEDIUM)

**Claim under attack.** Plan line 17 says: `--sort {lines,files,bytes,name}`. Unit 7.1 acceptance line 97 says: "For `SortName`, ascending is lexical A→Z".

**Repro.** `summary.Directory.Path` is the walk-relative directory path with forward-slash separators (per `render.Directory.Path` doc and F26). For the tree fixture in integration_test.go, dirs are `"."`, `"sub"`, `"sub/nested"`, etc. The "name" of a directory is unambiguous to a user — it's the leaf — but the field being sorted is the **full path**, not the leaf. Sorting `["./a/z", "./b/a"]` by "name":
- Leaf-sort: `["./b/a", "./a/z"]` (leaves: "a" < "z")
- Path-sort: `["./a/z", "./b/a"]` (paths: "./a/z" < "./b/a")

The plan implicitly picks path-sort (since `Path` is what `Directory` carries) but calls it `SortName`. A user reading `--help` "sort directories by ... name" will reasonably assume leaf-sort, get path-sort, and report a bug.

**Severity.** MEDIUM. Documentation drift, but the implementation is fine; just rename or clarify.

**Resolution paths:**
- **(A) Rename**: `name` → `path` in the SortKey constants, flag value enum, usage string, and tests. Most consistent with the field actually being sorted. Smallest behavior change.
- **(B) Implement leaf-sort**: `SortName` extracts `filepath.Base(d.Path)` for comparison. Adds branch but matches intuitive UX. Edge: leaf-collisions across different parents (`"foo/lib"` vs `"bar/lib"`) need a deterministic tiebreaker (full path).
- **(C) Document the path-sort meaning**: doc comment on `SortName` + flag usage clarifies "sorts by full relative path string". Cheapest but UX-fragile.

**Recommendation:** (A). The user surface is small, "path" is the field, and renaming costs four token swaps.

## Findings (REFUTED attacks, recorded for the dev's audit trail)

### 1.1 Renderer.RenderTree signature change — REFUTED

F37 changes `RenderTree(w io.Writer, dirs []render.Directory, ...)` to `RenderTree(w io.Writer, dirs []summary.Directory, ...)`. Plan correctly cites F15 (no external implementers under `internal/`; pre-v1.0). All three concrete implementers (`humanRenderer`, `jsonRenderer`, `toonRenderer`) are in the same package as the interface; the only outside-package caller is `cmd/rak/root.go`. The compile-time interface assertions in `internal/render/render_test.go` lines 312-316 and `cmd/rak/root_test.go` lines 22-26 will catch any drift. The signature change is mechanically safe.

### 1.2 `directoryJSON(filterUnknown(d))` conversion survival — REFUTED

`directoryJSON` is `Path string / Counts counting.Counts / ByLang map[...]...`. `summary.Directory` will be `Path string / Counts counting.Counts / ByLang map[...]... / Files int64`. Go struct conversion `directoryJSON(d)` requires identical underlying field types AND identical field names AND order. The plan acceptance (7.2 line 118) explicitly adds `Files int64 \`json:"files,omitempty"\`` to `directoryJSON` so the conversion compiles. Field order in both structs must match — but the plan does not pin field order. **Sub-finding:** Builder must ensure `Files` is the LAST field in both `summary.Directory` and `directoryJSON`, in the same position, or the bare conversion fails to compile. Add explicit "fields in this order: Path, Counts, ByLang, Files" to 7.1's acceptance to avoid a builder-time gotcha.

### 1.3 Interim path-sort retained in 7.2, removed in 7.3 — REFUTED with caveat

Plan line 57 explicitly retains the lexical sort in 7.2 and pins removal to 7.3. 7.3's acceptance (line 134) says "The inline lexical sort from Unit 7.2 is removed from `walkAndCount` at this point". So the plan correctly forecloses the "sort runs twice" trap. **Caveat for builder:** when 7.3 lands, the sort moves from `walkAndCount` to `runDirectory` AFTER `labelDirectories`. If sort happens BEFORE labeling, the `"./testdata/tree" < "./testdata/tree/sub"` ordering used by integration_test.go (line 230 comment) flips because the relative paths `"."` and `"sub"` sort differently from the labeled `"testdata/tree"` and `"testdata/tree/sub"`. The plan should pin: "sort applies to the labeled slice after `labelDirectories`, so the rendered order matches the user's mental model of the labeled paths" — particularly for `SortName` (see C2 above).

### 1.4 `tokens` rejection (F41) — REFUTED with caveat

Plan acceptance line 135 says: "An unrecognized value falls back to `SortLines` desc (no error; documented in `SortDirs` doc comment)." So `--sort tokens` does NOT error; it silently sorts by lines. This is a **soft rejection**. Two issues:
- F41 says "`tokens` is NOT a valid `--sort` key" — but the fall-through doesn't reject; it falls back. A user who types `--sort tokens` will see lines-sorted output and assume tokens-sort worked.
- Test coverage: plan line 136 lists tests for the four valid keys but does NOT pin a test for `--sort tokens` or `--sort bogus` fall-through behavior. Builder should add one.

Not a counterexample — the plan is internally consistent — but the planner should pick a stance: silent fallback (current) OR explicit error ("unknown sort key 'tokens'; v0.1.0 supports: lines, files, bytes, name").

### 1.5 `SortDirs` modifies in place — REFUTED

`slices.SortFunc` is in-place. Plan line 97 explicitly says "sorts `dirs` in place". Plan RiskNotes line 47 calls this out. Doc comment requirement is on the planner; builder will inherit. Callers in `runDirectory` consume `dirs` exactly once before passing to `RenderTree`, so in-place is fine.

### 1.6 `Files int64` counting semantics — REFUTED

Plan line 119 says: "accumulation loop adds `dir.Files++` per accepted file". "Accepted" in `walkAndCount` means: (a) survived ctx-cancellation check, (b) survived binary-skip gate when `binary=false`, (c) survived `--lang` filter when set, (d) `countFile` succeeded. The split-error path (line 316-319 of root.go) does NOT skip the file — it aggregates the error but continues to `countFile` and adds counts. So `Files` should increment in lockstep with the existing `byDir[dir] = addCounts(...)` line (root.go:329) — same place, same condition. Plan does not pin the EXACT site but the logic is unambiguous. Builder will land it on the right line.

### 1.7 JSON snapshot stability — REFUTED

`internal/render/render_test.go:269-289` (TestJSONRenderer_RenderTree_Snapshot) pins:
```json
{"directories":[{"path":".","counts":{...}},{"path":"sub","counts":{...}}],"total":{...}}
```

With `Files int64 \`json:"files,omitempty"\``, zero-Files cases emit no `"files"` key, so this snapshot survives. The test constructs `Directory{Path: ..., Counts: ...}` literals with zero Files. ✓

`cmd/rak/integration_test.go:227` `dirResult` struct has `Path` and `Counts` fields only. `json.Unmarshal` silently ignores unknown fields by default. ✓

`cmd/rak/root_test.go:227-230` same: `dirResult` ignores unknown `"files"` key on the wire. ✓

**However:** for tests that DO assert byte-exact JSON output of the integration walk (the one that calls `walkAndCount`, which increments `Files`), the snapshot WILL drift. Audit the test plan: `TestRootCmd_Integration_PathArg_JSONFormat` (integration_test.go:193-256) does NOT assert byte-exact JSON — it parses and structurally compares. ✓

`TestJSONRenderer_RenderTree_WithErrors` (render_test.go:320-359) DOES assert byte-exact:
```go
want := `{"directories":[{"path":".","counts":{...}}],"total":{...},"errors":[...]}` + "\n"
```
That test constructs a synthetic `Directory{Path: ".", Counts: ...}` literal with zero Files — survives via `omitempty`. ✓

All snapshot tests survive. The plan's `omitempty` decision (line 118) is load-bearing and correct.

### 1.8 `mage ci` order — REFUTED

Plan 7.2 acceptance (line 121) explicitly says `mage build` AND `mage test` pass after the unit. Since 7.2 changes `render.Directory` → `summary.Directory` across `render.go`, `human.go`, `json.go`, `toon.go`, `render_test.go`, `root.go`, and `root_test.go` in one unit, the unit's atomic commit will build clean. No intermediate broken state visible to `mage ci`. ✓

**Sub-finding:** Plan 7.2 paths list (line 105-111) does NOT include `cmd/rak/root_test.go`, but root_test.go has two compile-time interface assertions on `render.Renderer` (lines 22-26) and uses `treeResult`/`dirResult` types that only care about JSON-on-the-wire (not Go type identity), so the test file does NOT need editing for 7.2. ✓ — confirmed by reading the test. But if `dirResult` ever grows a Files field for test assertions, 7.2 will need to touch root_test.go. The plan should explicitly note "root_test.go is NOT modified in 7.2 because `dirResult` ignores the wire-level Files key".

### 1.9 Cascade vocabulary attacks — REFUTED (N/A for rak)

Rak does not use cascade vocabulary (drop/segment/confluence/droplet from Tillsyn). Drop 7 is a flat three-unit decomposition with serial `blocked_by`. No cascade-shape attack lands.

### 1.10 Sibling overlap without `blocked_by` — REFUTED

7.2 and 7.3 both touch `cmd/rak/root.go`. They are correctly chained `blocked_by` 7.2 → 7.3 in the plan tree. No parallelism is possible per planner's own analysis (line 23). ✓

## Hylla Feedback

N/A — review touched plan markdown + existing Go source via `Read`; Hylla query was not necessary because the action item is plan-QA on a flat tree and all evidence is in-file. No fallback occurred for the Go references because the surface was small enough to read directly.

## Verdict

**FAIL.** Two findings:
- **C1 (HIGH):** `--sort-asc` semantics + default direction for `SortName` are ambiguous. Plan must pin one of (A) key-specific defaults, (B) uniform-desc-document-it, or (C) `--sort-dir {asc,desc}` enum, BEFORE 7.3's test plan can be authored deterministically.
- **C2 (MEDIUM):** `SortName` semantics conflate path with leaf. Recommend rename to `SortPath` (and `--sort path`) since `Path` is the field actually being compared.

Minor sub-findings on builder ergonomics (struct field order pin, sort-after-label pin, root_test.go non-touch pin) recorded above; planner should fold them into 7.1/7.2 acceptance criteria in Round 2.

Round 2: route back to `go-planning-agent` for revision.
