# Drop 5 — Plan QA Falsification, Round 2

**Verdict:** pass-with-findings (1 confirmed counterexample, 2 minor findings).

## Summary

Round 2 successfully resolves Round 1's blocker (C1 parallelism) and the three Round 1 major findings (C2 special-filename pipeline, C3 ByLang JSON nil handling, C4 Policy α). The revised chain `5.1 → {5.2 || 5.4} → 5.3` is internally consistent; 5.4's dependency edge correctly lists `Blocked by: 5.1` only (not 5.2), and the parallel-eligibility note (lines 218–227) walks the dep-edge reasoning honestly. The new F33 LangUnknown-suppression pin is well-scoped and ships with three tests.

**One CONFIRMED counterexample remains (C1 below): the JSON renderer's existing `directoryJSON(d)` type-conversion idiom at `internal/render/json.go:67` blocks the additive `ByLang` field growth as the plan currently describes it.** This is a real "code won't compile" trap that Round 2 inherited from Round 1 without addressing.

Two minor findings (C2 special-filename basename scope, C3 RelPath wired-storage strategy) round it out.

## Counterexamples

### Counterexample C1 — `directoryJSON(d)` type-conversion at `json.go:67` blocks `Directory.ByLang` additive growth

- **Severity:** confirmed counterexample (blocker for Unit 5.3 as currently written)
- **Attack target:** Unit 5.3 acceptance (lines 173–184 of PLAN.md) — "`directoryJSON` struct grows an optional `by_lang` field"; F32 pin (lines 215–216 of PLAN.md) — "additive struct field is safe pre-v1.0".
- **Construction:** the current JSON renderer at `main/internal/render/json.go:61–79` populates the JSON envelope via the line

  ```go
  payload.Directories = append(payload.Directories, directoryJSON(d))
  ```

  This is a **Go type-conversion expression**, not a copy-by-field. Go's spec § "Conversions" requires that two named struct types `T1`, `T2` be **identical in field structure** (same field names, same types, same declaration order, ignoring struct tags) before `T2(v1)` compiles. Currently both `Directory` (in `render.go:51–60`) and `directoryJSON` (in `json.go:43–46`) have the same shape `{Path string; Counts counting.Counts}` and the conversion works.

  When 5.3 grows `Directory` to add `ByLang map[lang.Language]lang.LangCounts` (PLAN.md lines 173–175), **two paths can compile**:

  1. **Grow `directoryJSON` to mirror `Directory` exactly** — i.e., add `ByLang map[lang.Language]lang.LangCounts` with a `json:"by_lang,omitempty"` struct tag. The type conversion `directoryJSON(d)` keeps working. But this means the JSON wire shape encodes the map as `map[string]LangCounts` (json marshals `lang.Language` to its underlying `string` type), AND the F33 `LangUnknown`-suppression filter MUST run before the conversion, otherwise the suppression has no place to land — `directoryJSON(d)` blindly copies the map. So the conversion-keeper path forces an "iterate-and-filter into a fresh `directoryJSON` value, not a conversion" pattern.

  2. **Drop the conversion and build `directoryJSON` field-by-field** — replace line 67 with an explicit constructor that walks `d.ByLang`, skips `LangUnknown`, and builds a fresh map. Then `directoryJSON`'s `by_lang` field can have a different Go type from `Directory.ByLang` (e.g., the plan's stated `map[string]struct{Lines struct{Code,Comment,Blank int}; Counts counting.Counts}`).

  PLAN.md line 183 describes the `by_lang` Go type as `map[string]struct{Lines struct{Code,Comment,Blank int}; Counts counting.Counts}` — that's a **different type** from `Directory.ByLang`'s `map[lang.Language]lang.LangCounts`. **As written, the plan demands path (2) but Round 2 left the existing `directoryJSON(d)` conversion idiom intact in the codebase and does NOT call out that the conversion line at `json.go:67` must be rewritten.** A builder reading 5.3 will either:

  - Notice the conversion fails after adding `ByLang` to `Directory`, improvise a fix (most likely path 1 — grow `directoryJSON` symmetrically — because it's the minimal-diff resolution), and silently end up with a JSON wire shape DIFFERENT from the plan's stated `by_lang` type (the plan says nested anonymous struct; path 1 emits `lang.LangCounts` directly).

  - OR, less likely, read the plan's `by_lang` type more carefully, realize the conversion is broken, and rewrite the loop. But then the plan still hasn't pinned which approach to take, and the LangUnknown-suppression filter (F33) is silent on whether it runs before or after the conversion.

  Concrete consequence: two builders given this plan would produce two different JSON wire shapes. The F33 suppression tests (lines 195–197) would pass under both (they only check the negative — `LangUnknown` key absent), but the **`by_lang` value type** for non-Unknown entries would differ. Existing `dirResult` test struct in `root_test.go:227–230` would need updates either way, but the update target is undefined.

- **Mitigation status:** UNMITIGATED — Round 1 did not surface this (existing-code-interaction angle wasn't on Round 1's attack surface), and Round 2's "additive struct field is safe" framing in F32 assumed no other site cares about the struct shape. The `directoryJSON(d)` conversion line is exactly such a site.
- **Suggested fix (planner brief):** in PLAN.md Unit 5.3 acceptance, add:

  > **JSON renderer growth path (pin):** the existing `directoryJSON(d)` type-conversion at `internal/render/json.go:67` MUST be rewritten in this unit. Replace it with an explicit per-field constructor that:
  > 1. Copies `Path` and `Counts` directly.
  > 2. Walks `d.ByLang`, skips the `LangUnknown` key (F33), and copies surviving entries into a fresh `directoryJSON.ByLang` field.
  >
  > `directoryJSON` grows a `ByLang map[lang.Language]lang.LangCounts` field with `json:"by_lang,omitempty"` (same Go type as `Directory.ByLang` — the plan's earlier hint at a nested anonymous struct in `by_lang` is wrong; use the existing `lang.LangCounts` directly so the wire shape mirrors the in-memory shape and zero hand-rolled marshalling logic is needed). F33's `LangUnknown` filter runs INSIDE the conversion loop, not after marshalling, so the field-tag's `omitempty` cleanly omits the key when no recognized language survived. Lock this in F33 or a new F34 pin.

  Adjust PLAN.md line 183 accordingly. Add `TestJSONRenderer_RenderTree_ByLang_OmitsLangUnknown_AndPreservesGo` to the new tests list with explicit byte assertions on the `by_lang` JSON shape.

## Findings (minor, no counterexample produced)

### Finding F1 — Special-filename lookup target not pinned (basename vs RelPath)

- **Severity:** minor
- **Attack target:** Unit 5.1 acceptance, line 45: "Special-filename lookup: consult inline `specialFilenames map[string]Language` (case-insensitive key lookup; normalize with `strings.ToLower`). Keys include at minimum: `"makefile"`, `"gnumakefile"`, `"dockerfile"`, `"cmakelists.txt"`. If match → return immediately."
- **What's vague:** the plan says "case-insensitive key lookup; normalize with `strings.ToLower`" but does NOT pin **what string is being looked up**. Two readings:
  1. `strings.ToLower(filepath.Base(f.RelPath))` — basename only. Then `sub/Makefile` → `"makefile"` → match.
  2. `strings.ToLower(f.RelPath)` — full RelPath. Then `sub/Makefile` → `"sub/makefile"` → no match.

  The naming `specialFilenames` (plural) and the example keys (`"makefile"`, `"dockerfile"`) imply basename, and the `TestDetect_SpecialFilename` test cases at lines 52–53 use paths with no directory component, so they don't actually distinguish the two readings. A `Makefile` deep in a subdirectory would silently fail special-filename detection under reading (2).
- **Mitigation status:** partially mitigated by convention but the test table doesn't pin it.
- **Suggested fix:** add one line to PLAN.md line 45: "Lookup key is `strings.ToLower(filepath.Base(f.RelPath))`." Add one new test case to `TestDetect_SpecialFilename`: `"sub/Makefile"` → `LangMakefile` (probe the subdir case).

### Finding F2 — `detectedLang` storage strategy ("walk context") under-specified

- **Severity:** minor
- **Attack target:** Unit 5.1 acceptance, line 33: "extend `walkAndCount` — wire `lang.Detect(f)` call per file, attach result to walk context"; line 60: "Store the resulting `Language` value in a per-file local (e.g. `detectedLang := lang.Detect(f)`). This wired value is what 5.2's `Split` call (added in 5.3) and 5.4's filter gate (added in 5.4) will each consume."
- **What's vague:** "per-file local" works fine WITHIN a single iteration of the `for f, walkErr := range source.List(ctx)` loop, but the plan mixes that with "attach result to walk context" (line 33), which suggests longer-lived state. Three readings:

  1. **Per-iteration local** — `detectedLang := lang.Detect(f)` inside the loop body. 5.4 reads it inline (same loop iteration). 5.3 reads it inline (same loop iteration). Trivial. No shared state.
  2. **Map keyed by RelPath** — `detectedByPath map[string]lang.Language` built before the loop or alongside. 5.4 looks up by `f.RelPath`. Plausible if 5.4 wants to pre-filter into a slice for a second pass.
  3. **New field on `*fileset.File`** — `f.DetectedLang lang.Language`. Plan paths don't include `internal/fileset/file.go`, so this is OUT of 5.1's scope and would be a scope violation.

  Reading (3) is ruled out by the paths list. The plan implies reading (1) via "per-file local" but the language "attach result to walk context" creates ambiguity. 5.4's acceptance (line 146) — "Consumes the `detectedLang` value already wired by Unit 5.1 (no second `lang.Detect` call)" — assumes inline access in the same loop iteration, which forces reading (1).
- **Mitigation status:** the plan implicitly converges on (1) but the "walk context" language at line 33 invites confusion.
- **Suggested fix:** drop the "attach result to walk context" phrase from line 33. Replace with: "wire `lang.Detect(f)` call per file as a per-iteration local variable (`detectedLang := lang.Detect(f)`) inside the `for f, walkErr := range source.List(ctx)` loop body. Subsequent units (5.4's filter gate, 5.3's `Split` call) read the same per-iteration local; no map or struct-field storage required."

## Attack families exhausted (no counterexample found)

- **Dep-chain integrity** (`5.1 → {5.2 || 5.4} → 5.3`): walked. 5.4's `Blocked by: 5.1` is correct (5.4 needs `Language` + the wired `detectedLang`; no `LangCounts` / `Split` dependency on 5.2). 5.3's `Blocked by: 5.2, 5.4` is correct (needs `LangCounts.Add` from 5.2; touches `cmd/rak/root.go` which 5.4 last touched). REFUTED — no counterexample.
- **5.1 scope creep across `internal/lang` + `cmd/rak`**: borderline-but-acceptable. The plan's carve-out note (line 241) confirms each unit compiles independently. The blast radius is bounded — 5.1 adds one local-variable assignment + one import in `cmd/rak/root.go`. Splitting 5.1 into 5.1a (lang package) + 5.1b (wiring) would add ceremony without parallelism gain (the two would still serialize, same builder, same drop). EXHAUSTED — no counterexample.
- **Special-filename matching ambiguity (Makefile.go case)**: `specialFilenames` is a Go `map[string]Language`. Map-key lookup is by definition exact match on the key, not prefix or substring. `"makefile.go"` (lowercased basename of `Makefile.go`) is not in the key set `{"makefile", "gnumakefile", "dockerfile", "cmakelists.txt"}` → step 1 returns LangUnknown → step 2 (extension `.go`) → LangGo. REFUTED — Go map semantics rule out the attack.
- **Policy α F28 false positive on line-comment markers**: F28 wording (line 87) explicitly scopes Policy α to "any block-comment marker (`/*` or `*/`)". A line like `x := 1 // some text with no block markers` contains no `/*` or `*/` substring → Policy α does not fire → classified Code per default. F28's known limitation only covers `/*`/`*/` inside string literals, which is documented. REFUTED — wording is precise.
- **F33 LangUnknown suppression vs existing integration test fixtures**: the existing fixture `cmd/rak/testdata/tree/` holds `a.txt` (12 B) and `sub/nested.txt` (8 B). The plan's extension table does NOT list `.txt`, so both files detect as `LangUnknown`. After F33 suppression, `Directory.ByLang` is empty (or nil) for both directories. Existing tests at `cmd/rak/integration_test.go:193–256` use a local `treeResult` / `dirResult` shape that does NOT include `by_lang`, and JSON `omitempty` on the new field means the wire output stays backward-compatible for the unmarshal target. The `len(parsed.Directories) != 2` assertion (line 224) still holds. REFUTED — existing tests survive Drop 5 unchanged.
- **`LangCounts.Add` signature**: pointer-receiver pinned in PLAN.md line 79: `func (lc *LangCounts) Add(other LangCounts)`. 5.3's accumulator initializes a `LangCounts` value, then mutates via `(&lc).Add(other)` or via `lc := byDirLang[dir][lang]; lc.Add(other); byDirLang[dir][lang] = lc`. Workable; plan pins it. REFUTED.
- **Double-IO trade-off (P4)**: documented at lines 230–231; builder has explicit choice + documentation requirement. EXHAUSTED — known and accepted, not a counterexample.
- **5.4 `LangUnknown` filter behavior** (F29 says `--lang` filter rejects `LangUnknown` files): a user running `rak --lang go .` on a tree where some files have unknown extension correctly excludes them per F29 plain reading. The minimum-coverage list at line 43 names enough languages that real Go-only filtering works. REFUTED.
- **Renderer interface signature preserved (F32)**: confirmed against `internal/render/render.go:27–39` — `Renderer.RenderTree(w io.Writer, dirs []Directory, total counting.Counts, errs []error) error` is the exact signature 5.3 leaves untouched (only `Directory` grows). F32 holds. REFUTED.

## What rounds 1+2 dev-routed findings look like in the document

The Decision-C1 chain revision is reflected accurately at line 23 and lines 220–227. The Decision-C2 special-filename pipeline addition is at lines 44–45. The Decision-C3 LangUnknown suppression is at F33 (lines 216) + Unit 5.3 acceptance (lines 177, 194–197). The Decision-C4 Policy α pin is at lines 85–91 and F28 (line 211). The Decision-C5/C6/C7/C8 sweeps are reflected. No drift between dev decisions and plan text.

## Verdict

**1 confirmed counterexample (C1 — JSON renderer type-conversion break).** This is a real "code won't compile as planned" blocker for Unit 5.3 — the planner needs one more sweep to pin the `directoryJSON(d)` rewrite + the exact `by_lang` Go type.

**2 minor findings (F1 — basename pin, F2 — walk-context language).** Both are pin-tightening; neither blocks dispatch but both are quick fixes.

Pass with one revise-required item.
