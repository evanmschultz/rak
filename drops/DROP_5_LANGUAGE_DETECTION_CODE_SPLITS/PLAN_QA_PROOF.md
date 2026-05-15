# Drop 5 — Plan QA Proof, Round 1

**Verdict:** pass-with-findings

## Summary

The plan is well-grounded and consistent with Drop 4's surface (F19 / F23 / F25 / F26
preserved; cobra flag mutual-exclusion gate intact; `Renderer` interface signature
unchanged). The four-unit decomposition is internally coherent, each unit has
explicit `Paths` / `Packages` / `Acceptance` / `Blocked by`, and the F-pin
register (F27–F32) genuinely captures load-bearing invariants for v0.1.0.

Six findings — none blocker. Two are major (P1 chain-rationale and P2 unknown-only
`ByLang`), two are medium (P3 priority-mode contract and P4 single-pass tee), two
are minor (P5 cross-platform extension edge case and P6 `addCounts` analog for
LangCounts). The parallelism question P1 surfaces is not a defect in the plan's
correctness — the planner may have made a defensible serialization choice — but
the rationale needs sharpening so dev review in Phase 3 can accept it knowingly
rather than by default.

External API claims in the plan check out: `bufio.Scanner` default split is
`\r?\n` (handles LF + CRLF, strips both, returns last unterminated line) per
`go doc bufio.ScanLines`. `filepath.Ext` returns the suffix beginning at the
final dot in the final element per `go doc path/filepath.Ext` — that is the
contract Unit 5.1 needs.

## Findings

### Finding P1 — Strict-linear-chain rationale conflates "same package" with "must serialize"

- **Severity:** major
- **Unit/F-pin affected:** Units 5.1 + 5.2; § "Parallel eligibility"
- **Claim:** The plan asserts "5.1 and 5.2 both live in `internal/lang` (same
  package — serialized by rule)." There is no documented rak rule that
  same-package work must serialize. The documented criterion from
  `feedback_parallelize_aggressively` is "different *files*, no shared mutation
  surface, no dependency edge." 5.1 writes `lang.go` + `lang_test.go`; 5.2
  writes `split.go` + `split_test.go` (+ optionally `grammar.go`) — different
  files, no shared mutation surface inside the files themselves.
- **Evidence missing or weak:** The plan does not enumerate the actual coupling
  edges. There IS a real coupling: 5.2's `Split(r io.Reader, lang Language)
  (LineCounts, error)` consumes `Language` (type) and `LangGo` /
  `LangPython` / etc. (constants) defined in 5.1. That is a dependency edge
  — not a "same package" rule. The chain should be justified as
  "5.2 blocked by 5.1 because 5.2 imports the `Language` type and language
  constants 5.1 defines," not by package co-location. Similarly, 5.3 ←
  5.4 share `cmd/rak/root.go` — that one IS a true shared-mutation
  serialization (same file, same `rootFlags` struct, same `walkAndCount`
  function). 5.3 ← 5.2 is a real dep edge (`LangCounts`, `LineCounts`).
- **Suggested fix:** Rewrite the § "Parallel eligibility" paragraph to cite
  *type/symbol dependencies* instead of "same package":
  - 5.2 blocked by 5.1: needs `type Language`, `LangXxx` constants — dep edge.
  - 5.3 blocked by 5.2: needs `LangCounts`, `LineCounts`, `Detect`, `Split` — dep edge.
  - 5.4 blocked by 5.3: needs `lang.Detect` already wired into `walkAndCount`
    AND shares `cmd/rak/root.go` mutation — both dep edge and shared file.
  No outcome change; the chain stays linear. The rationale becomes
  audit-correct rather than rule-of-thumb.

### Finding P2 — `ByLang` semantics for "every file unknown" is unspecified

- **Severity:** major
- **Unit/F-pin affected:** Unit 5.3, F29, F31
- **Claim:** The plan says renderers emit per-lang detail "when `d.ByLang` is
  non-empty" and that `ByLang` is nil "when language detection was not run."
  But after 5.3, language detection is ALWAYS run in `walkAndCount` (5.4's
  `--lang` filter pre-supposes detection has run). For a directory whose every
  surviving file is `LangUnknown` (e.g., the existing
  `cmd/rak/testdata/tree` fixture — `a.txt`, `sub/nested.txt` both detect to
  `LangUnknown`), `ByLang` will be `{LangUnknown: <counts>}` — non-empty,
  containing only the zero-value Language. The plan does not specify whether
  this should:
  - (a) render a `lang/: ...` row with empty-string key (renders as awkward in
    human / TOON and confusing in JSON);
  - (b) be suppressed in renderers so unknown-only dirs look like pre-5.3 output;
  - (c) render as `lang/unknown: ...` with a user-friendly label.
- **Evidence missing or weak:** F29 says "the zero value" but doesn't tie that
  to renderer display behavior. The integration fixture `testdata/tree` is the
  concrete case — Drop 4 snapshot tests assert byte-exact JSON for that
  fixture; without an explicit decision, 5.3 silently breaks the Drop 4
  integration snapshot.
- **Suggested fix:** Add a per-unit acceptance bullet in 5.3 (or an F-pin —
  e.g., F33) deciding one of (a)/(b)/(c). Recommend (b): renderers iterate
  `ByLang` but skip entries with `LangUnknown` keys, OR `walkAndCount` omits
  `LangUnknown` from the `ByLang` map at populate time. Either way, declare
  whether existing Drop 4 integration snapshots stay byte-exact (they should,
  for backward compatibility) — and add that assertion to 5.3's integration
  test bullet.

### Finding P3 — `Detect`'s "shebang only when extension is unknown OR generic" is undefined

- **Severity:** medium
- **Unit/F-pin affected:** Unit 5.1, F27
- **Claim:** 5.1 acceptance step (2) says shebang sniff runs "when extension
  lookup returns `LangUnknown` OR yields a generic language." The set of
  "generic" languages is never enumerated. Without it, the builder cannot
  know whether `.sh` (already maps to `LangShell` in step 1) should fall
  through to shebang to upgrade to `LangBash`, or whether `.js` (→ `LangJS`)
  should fall through to detect Node-vs-deno-vs-bun, etc.
- **Evidence missing or weak:** `TestDetect_ExtensionBeatsShebang` (line 48 of
  PLAN.md) asserts extension wins, which is consistent with "generic = none"
  (i.e., shebang only fires on `LangUnknown`). But the prose says "OR yields
  a generic language" which contradicts the test.
- **Suggested fix:** Resolve by deleting "OR yields a generic language" from
  the prose — make the rule a clean "shebang fires only when extension
  returns `LangUnknown`." Update the corresponding doc comment requirement.
  If the planner intended the "generic" fall-through (e.g., `.sh` →
  `LangShell`, then shebang refines to `LangBash`), enumerate the generic
  set explicitly (e.g., `{LangShell}`) so the builder + QA can verify.

### Finding P4 — Single-pass tee vs double-open trade-off is asymmetric

- **Severity:** medium
- **Unit/F-pin affected:** Unit 5.3, "Ordering note"
- **Claim:** The plan offers builder's choice between (a) opening the file
  twice (once for `counting.Count`, once for `lang.Split`) or (b)
  `io.TeeReader` single-pass. Both are described as acceptable. But (a)
  doubles the I/O for every file walked — on a 10k-file repo this is a
  measurable wall-clock regression (Drop 8.1 will add parallel walking,
  amplifying the cost). The plan does not call this out.
- **Evidence missing or weak:** No reference to Drop 8.1's parallel-walker
  cost-pressure, no measurement guidance ("if double-open, document the
  wall-clock cost in BUILDER_WORKLOG"). The "two-open approach is simpler
  and acceptable for v0.1.0" sentence reads as a green-light without an
  exit ramp.
- **Suggested fix:** Make the recommended path explicit. Either:
  (a) Plan defaults to `io.TeeReader` single-pass; double-open allowed only
      with a builder-supplied wall-clock justification in BUILDER_WORKLOG.
  (b) Plan defaults to double-open; flags it as a Drop 8.1 follow-up to
      revisit (and adds a U4 unknown for dev confirmation).
  Either is fine, but the trade-off should be a planner decision, not
  builder's-choice — because the choice has measurable performance
  consequences across the entire drop's product feature.

### Finding P5 — Extension table key normalization needs cross-platform pin

- **Severity:** minor
- **Unit/F-pin affected:** Unit 5.1
- **Claim:** The acceptance bullet says
  `strings.ToLower(filepath.Ext(f.RelPath))`. `filepath.Ext` is defined to
  return the suffix beginning at the final dot in the *final element of
  path* — which is correct, but the table key choice ("without the leading
  dot as the key — or with dot, builder's choice, but document it") leaves
  a small foot-gun: if the table is keyed *without* dot but the lookup uses
  `filepath.Ext`'s output (which always includes the dot when one exists),
  the builder must strip the leading dot before lookup. Builder freedom
  with a strip-mismatch is a likely round-2 finding.
- **Evidence missing or weak:** `go doc path/filepath.Ext` confirms the
  return value includes the leading dot ("`.go`" not "`go`"). No test in
  the plan asserts the key-shape choice — the only ext tests assert
  Language results, not key shape.
- **Suggested fix:** Pin the table key shape in 5.1's acceptance: "keys
  include the leading dot, e.g., `".go"` → `LangGo`. Lookup uses
  `strings.ToLower(filepath.Ext(f.RelPath))` directly, no strip." This
  matches the lowercase-extension example already given on line 41
  (`".go"` → `LangGo`) — the explicit pin just kills the
  builder's-choice ambiguity.

### Finding P6 — `addCounts` analog for `LangCounts` is implied but unnamed

- **Severity:** nit
- **Unit/F-pin affected:** Unit 5.3
- **Claim:** `cmd/rak/root.go`'s existing `addCounts` helper sums two
  `counting.Counts` field-wise. After 5.3, `walkAndCount` needs an
  analogous field-wise sum for `lang.LangCounts` (which contains both
  `LineCounts` and `counting.Counts`). The plan describes the
  accumulation behavior ("Accumulate `LangCounts` per dir+lang key") but
  doesn't name the helper, so the builder may put it in `cmd/rak` or
  inside `internal/lang` (closer to the type).
- **Evidence missing or weak:** No explicit symbol named. Minor —
  builder will figure it out. But for symmetry with `addCounts` (already
  in `cmd/rak`), a one-line acceptance pin would speed QA.
- **Suggested fix:** Add to 5.3 acceptance: "introduce
  `lang.AddLangCounts(a, b lang.LangCounts) lang.LangCounts` (or a
  package-local helper in `cmd/rak`, builder's choice) for the
  accumulation. Match `addCounts`'s field-wise pattern."
