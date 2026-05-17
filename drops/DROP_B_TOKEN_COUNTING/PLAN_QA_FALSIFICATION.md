# DROP_B — PLAN_QA_FALSIFICATION

## Round 1

**Verdict:** PASS WITH FINDINGS

The plan is structurally sound and the high-risk surfaces (toon-go `omitempty`, JSON
snapshot drift, encoder lifecycle) are explicitly named in RiskNotes rather than swept
under the rug. But four findings rise to blocker, several more to concern, and the
B/C/D root.go-serialization handshake is under-specified. None of the blockers
require rethinking the decomposition — they are surgical sharpenings to acceptance
criteria + one design correction.

## Counterexamples / Attacks

### Attack 1 — Existing JSON snapshot tests will break under B.2 + B.4

- **Severity:** blocker
- **Where:** Unit B.2 (json:",omitempty" tag claim) + Unit B.4 (JSON renderer "no
  change needed" claim)
- **Counterexample:** The plan asserts (B.2) that adding `Tokens int64 \`json:",omitempty"\``
  to `counting.Counts` preserves snapshot test compatibility. This is false for at least
  one existing test — and likely a second.

  Evidence: `internal/render/render_test.go:93-106` (`TestJSONRenderer_Snapshot`) pins
  byte-exact:
  ```
  want := `{"Bytes":12,"Lines":1,"Words":2,"Chars":12}` + "\n"
  ```
  Note the field names are **capitalized** — `counting.Counts` carries NO json struct
  tags today (see `internal/counting/counting.go:19-31` — pure declaration). Adding
  ONE tagged field (`Tokens int64 \`json:",omitempty"\``) does not break this specific
  snapshot for `Counts{Bytes:12,Lines:1,Words:2,Chars:12}` because `Tokens=0` is
  suppressed by omitempty.

  HOWEVER: `TestJSONRenderer_Table` (line 110-148) includes a `zero` case with
  `counting.Counts{}` (all zeros) where the snapshot is `{"Bytes":0,"Lines":0,"Words":0,"Chars":0}`.
  With `omitempty` on `Tokens` and `Tokens=0`, this snapshot still passes. So far so good.

  The real failure: `TestJSONRenderer_RenderTree_Snapshot` (line 274-297) byte-exactly
  pins:
  ```
  want := `{"directories":[` +
      `{"path":".","counts":{"Bytes":12,"Lines":1,"Words":2,"Chars":12}},` +
      `{"path":"sub","counts":{"Bytes":4,"Lines":1,"Words":1,"Chars":4}}` +
      `],"total":{"Bytes":16,"Lines":2,"Words":3,"Chars":16}}` + "\n"
  ```
  Same — `Tokens=0` omitted, passes. Likewise `TestJSONRenderer_RenderTree_Empty`
  (line 303-316) and `TestJSONRenderer_RenderTree_WithErrors` (line 327-367).

  **So the apparent blocker turns out to be REFUTED for the JSON-zero case** —
  `omitempty` on `int64` works as expected for `Tokens=0`. The plan is correct that
  zero-token JSON output preserves snapshots.

  BUT: the integration test in `cmd/rak/integration_test.go:46+` uses fixed expected
  byte/line/word/char constants. Verify the integration test does not pin a JSON
  snapshot that would now include `"Tokens":N` when `--tokens` IS set in a future
  test added by B.5. (The plan adds a B.5 integration test asserting non-zero
  Tokens, which is correct — it just must not duplicate or conflict with
  existing snapshots.)

  **Real blocker remaining:** The B.2 acceptance criterion mis-claims:
  > `counting.Count` behavior is unchanged — returns `Tokens: 0` in all cases (the
  > zero value due to `omitempty` keeps JSON output clean for non-`--tokens` runs).

  This is correct ONLY if the renderer continues to embed `counting.Counts` directly
  via the `Counts counting.Counts \`json:"counts"\`` field path. If a builder
  "helpfully" denormalizes (lifts Tokens up into `directoryJSON` as a sibling field
  for symmetry with TOON columns), the omitempty contract changes shape silently.

  **Mitigation:** Add an explicit acceptance bullet to B.4: *"Builder MUST NOT lift
  `Tokens` out of `counting.Counts` into a separate `directoryJSON.Tokens` field;
  the existing pass-through via the embedded `counts` object is the contract. The
  Tokens key MUST appear inside the nested `"counts":{}` object in JSON output,
  never as a sibling of `"path"`."* This pins the embedding shape so future
  refactors don't drift.

### Attack 2 — TOON snapshot tests WILL break (the plan acknowledges this risk but acceptance is too soft)

- **Severity:** blocker
- **Where:** Unit B.4 (TOON renderer changes + snapshot test handling)
- **Counterexample:** `toon-go` tabular array format is documented (Context7
  `/toon-format/toon-go`) as emitting a header followed by every row with every
  column:
  ```
  items[2|]{sku|qty|price}:
    A-1|5|9.99
    B-2|2|14.5
  ```
  There is NO per-field omitempty mechanism documented for individual columns in
  tabular arrays. The plan's hedge — "If toon-go does not support omitempty on
  numeric types, always emit the column" — is correct that the column will always
  appear, but the acceptance criteria does not say which tests will then need
  updating.

  Concretely, these existing tests pin TOON column shape:
  - `TestRenderer_DirectoriesFilesColumn_TOON` (line 697-736) asserts
    `idxPath < idxFiles < idxBytes` and matches `"alpha|3|"`, `"beta|5|"`,
    `"gamma|0|"`. After B.4 adds `Tokens` AFTER `Chars`, this test still passes
    (the column added is at the END, not between path/files/bytes).
  - `TestTOONRenderer_RenderTree_PerLang` (line 469-504) uses `strings.Contains`
    only — robust.
  - `TestTOONRenderer_Render` (line 373-390) checks `"bytes: 12"`, `"lines: 2"`,
    `"words: 2"`, `"chars: 12"` substrings — robust against a new `tokens: 0`
    line being added.
  - `TestTOONRenderer_RenderTree` (line 399-423) — robust substring assertions.

  So the existing TOON snapshot break risk is LOWER than the plan implies — most
  TOON tests use `strings.Contains` not byte-exact. **HOWEVER**, every TOON
  document emitted in v0.2.0 will now carry a `tokens|0` column for non-`--tokens`
  invocations because toon-go can't suppress per-row columns. That's a user-facing
  cost: every existing tape/gif regenerated will show a `tokens` column populated
  with zeros.

  **Mitigation:** Add explicit acceptance bullets to B.4:
  1. *"Builder MUST verify toon-go behavior on `Tokens=0` by writing a no-op
     `--tokens` test fixture and inspecting the output. If toon-go emits
     `tokens|0` columns unconditionally, the builder MUST document this in
     BUILDER_WORKLOG.md and the user-facing implication (non-`--tokens` TOON output
     gains a `tokens` column) must be raised as a Round 1 finding before
     proceeding — not silently shipped."*
  2. *"All existing tape outputs in `docs/tapes/*.tape` must be re-recorded and
     diffed as part of B.6 to capture any unintended TOON column changes."* (The
     plan owns regeneration of `tokens.tape` only; the other 7 tapes may now
     show a `tokens` column that was not there before.)

### Attack 3 — `--sort tokens` without `--tokens` is undefined behavior

- **Severity:** concern
- **Where:** Unit B.3 / B.5 (no acceptance criterion covers this case)
- **Counterexample:** Per B.5 the user can pass `rak --sort tokens .` without
  `--tokens`. In that case `Counts.Tokens` is 0 for every directory. `SortDirs`
  with `SortTokens` will then degenerate to: all comparisons return 0, slice
  remains in walk-discovery order, output silently sorted-but-not-actually-sorted.

  This is not a crash — `cmp.Compare(0, 0) == 0` and `slices.SortFunc` is
  *stable-but-not-guaranteed-stable* per Go stdlib documentation. The user sees
  arbitrary-order directories with no error or warning. Worst case: `--sort tokens`
  silently degrades to "directory walk-discovery order" on a tree where someone
  forgot the `--tokens` flag, and the dev wonders why their LLM context budget
  isn't where they expected the heavy directories to be.

  The plan does not address this case at all. Options:
  1. **Warn/error**: `PersistentPreRunE` rejects `--sort tokens` when
     `!flags.tokens` ("--sort tokens requires --tokens").
  2. **Silently accept**: document in `--help` that `--sort tokens` without
     `--tokens` produces zero-keyed ordering.
  3. **Auto-imply**: passing `--sort tokens` implies `--tokens` (slow walk).

  Option 1 is the least surprising (explicit + early-fail). Option 3 is the most
  ergonomic but violates the principle of least magic. Option 2 ships sharp edges.

  **Mitigation:** Add a B.5 acceptance bullet: *"`rak --sort tokens .` without
  `--tokens` MUST error in PersistentPreRunE with: `--sort tokens requires
  --tokens`. New test: `TestRoot_SortTokensRequiresTokensFlag` covers this case."*

### Attack 4 — Tokenizer thread-safety claim is unverified and pre-fails Drop C

- **Severity:** concern
- **Where:** Unit B.1 (RiskNote on goroutine safety)
- **Counterexample:** The plan's RiskNote says:
  > BPE encoders are read-only after init; the returned `Codec` is assumed
  > goroutine-safe for concurrent `Count` calls. This assumption is acceptable for
  > v0.2.0 (single-goroutine walk). Parallel walk (Drop 8.1) should re-verify.

  Two problems:
  1. **"Drop 8.1" reference is stale** — per session-handoff memory the parallel
     walk is now Drop C (planned for v0.2.0, not deferred). The same v0.2.0 release
     ships both token counting AND parallel walk, so the "single-goroutine walk
     is fine" defense evaporates within the same release cycle.
  2. **The Context7 docs for tiktoken-go say NOTHING about thread safety** — there
     is no `// Codec is safe for concurrent use` note in the API surface. The
     planner is assuming; the library may or may not be safe. If `Codec.Count`
     internally maintains scratch state (BPE merging uses a priority queue / byte
     buffer; some implementations reuse buffers across calls), `-race` may or may
     not catch a data race depending on whether the racing writes hit overlapping
     memory.

  Concrete counterexample for Drop C: parallel walk dispatches N goroutines, each
  calling `codec.Count(string(buf))` against the shared singleton. If the codec
  reuses a scratch buffer internally, two goroutines mutating that buffer
  simultaneously produces silent token-count corruption. `-race` MAY catch this
  if the writes are to a shared `[]byte`; may NOT catch it if the corruption
  manifests as wrong-but-deterministic ints (no race-detected write, just two
  reads interleaved with a write).

  **Mitigation:** Add a NEW test to B.1 acceptance:
  *"`TestGet_ConcurrentCount`: 100 goroutines call `codec.Count(text)` 1000
  times each against the singleton; verify all return identical counts and the
  test passes under `-race`. If this test fails or races, B.1 escalates to
  require wrapping `codec.Count` with a `sync.Mutex` in `internal/tokens` —
  defending Drop C against the parallel-walk race in B's own deliverable rather
  than punting to a downstream drop that doesn't yet exist."*

  This is "pay $5 of testing today to skip $50 of cross-drop race-debugging
  tomorrow."

### Attack 5 — B.5 plumbing breaks `tokens.Codec` interface conformance silently

- **Severity:** concern
- **Where:** Unit B.5 (cmd/rak imports both `internal/tokens` and `internal/counting`)
- **Counterexample:** The design says:
  > `cmd/rak` imports both `internal/tokens` (for `tokens.Get()`) and
  > `internal/counting` (for `CountWithTokens` and the `TokenCounter` interface),
  > and passes the concrete `tokens.Codec` as the interface.

  But `internal/tokens.Get()` returns `tokenizer.Codec` (the upstream library's
  interface from Context7), not `*tokens.Codec`. The "`tokens.Codec`" name in the
  plan refers to the **library's** `tokenizer.Codec` interface. The plan slightly
  over-claims by calling it "tokens.Codec" — there isn't necessarily a re-export.

  Two possibilities:
  1. `internal/tokens.Codec = tokenizer.Codec` (type alias re-export).
  2. `internal/tokens` exposes a wrapping struct whose `Count(string) (int, error)`
     method satisfies the upstream interface AND `counting.TokenCounter`.

  The plan leaves this ambiguous. If the builder picks (2) but forgets to re-export
  the underlying `tokenizer.Codec` interface, downstream callers in cmd/rak that
  reference `tokens.Codec` won't compile. If the builder picks (1) and the library
  ever changes its interface signature, every rak call site breaks transitively.

  **Mitigation:** Add a B.1 design decision:
  *"`internal/tokens.Get() (counting.TokenCounter, error)` — return the
  satisfied-interface type directly rather than re-exposing the upstream
  `tokenizer.Codec`. This isolates rak from upstream interface drift and avoids
  cmd/rak having to import both `internal/tokens` AND `internal/counting` to wire
  the call. cmd/rak gets `tc, err := tokens.Get()` and passes `tc` straight to
  `counting.CountWithTokens(rc, tc)`."*

  This violates the plan's stated "leaf node, zero `internal/` imports" property
  for `internal/tokens` — but the violation is **`internal/tokens` imports
  `internal/counting`**, which is fine: `internal/counting` is itself a true leaf
  and the dependency is acyclic (tokens → counting; no reverse path). The current
  plan inverts this: `cmd/rak` carries both imports and the duck-typing happens
  implicitly at the call boundary. That works but is more fragile and creates
  three places to look when something goes wrong.

### Attack 6 — `cl100k_base` "Claude approximation" caveat in `--help` won't fit

- **Severity:** concern
- **Where:** Unit B.5 + B.6 (cobra `Example:` + flag description)
- **Counterexample:** Plan proposes:
  ```
  "count tokens using cl100k_base (GPT-3.5/4 approximation; slower than byte counting)"
  ```
  Two issues:
  1. Cobra flag descriptions are single-line in `--help` output. The string above
     is 75 characters — fits, but loses the **critical caveat** that this is an
     approximation for Claude and counts will differ from Claude's actual billing.
     The README narrative will spell this out, but `--help` is the contract.
  2. A user piping `rak --tokens .` into a Claude billing workflow will discover
     the discrepancy after the fact. The shorter the `--help` text, the higher
     the chance the user trusts the count for purposes the planner did not intend.

  **Counterexample to the current README + `--help` strategy:** Many users only
  read `--help`. The README narrative caveat is necessary but not sufficient.

  **Mitigation:** Extend the `--help` flag description to:
  ```
  "count tokens using cl100k_base (GPT-3.5/4 vocab; approximation for Claude, " +
      "off-by-roughly-20% vs Claude billing — slower than byte counting)"
  ```
  Or add a second-line caveat in `Long:` of the root command that always renders
  in `--help`. The plan currently has zero acceptance criterion for the `--help`
  surface — only README. Add: *"`rak --tokens --help`-equivalent (the flag
  description shown in `rak --help`) MUST include the word `approximation` and a
  reference to Claude billing differing."*

### Attack 7 — `o200k_base` is closer to current Claude than `cl100k_base`; cl100k default is shipping yesterday's model

- **Severity:** concern (design call for dev, not blocker)
- **Where:** Unit B.1 (default encoder choice)
- **Counterexample:** Context7 `/tiktoken-go/tokenizer` shows `O200kBase` is the
  encoding used by GPT-4o and the O-series (current OpenAI frontier). Anthropic
  has not published an official BPE tokenizer for Claude 3 / 3.5 / 4, but
  empirically (community measurements) `o200k_base` is closer to Claude's actual
  tokenization than `cl100k_base` for English-heavy code corpora — `cl100k_base`
  is the 2022-vintage GPT-3.5/4 encoding.

  Picking `cl100k_base` as the default in 2026 ships an encoder optimized for
  models more than two model generations old. "Approximation for Claude" is more
  accurate with `o200k_base` than `cl100k_base`.

  Counterargument: `cl100k_base` is the most widely-known tiktoken encoding,
  community familiarity is highest, and the dev has already approved this in
  Decision 11. The plan correctly defers `o200k_base` exposure to a future drop
  (presumably via `--tokens-encoding=o200k_base`).

  **Mitigation:** Two paths — pick one:
  1. **Keep cl100k_base default + add `--tokens-encoding` flag in B.5 now**
     (option set: `cl100k_base`, `o200k_base`). Cost: one extra flag, ~20 LOC
     in B.1 (`Get(encoding tokenizer.Encoding)` accepts a parameter; default
     stays cl100k_base). Benefit: users who know they want closer-to-Claude can
     opt into `--tokens-encoding=o200k_base` without waiting for v0.3.
  2. **Defer to v0.3** as the plan does today — but then the README "Claude
     approximation" caveat MUST cite a measured token-count delta (e.g.
     "cl100k_base over-counts vs Claude by ~10–25% for typical code") rather
     than vague "approximation" language, so users can do back-of-envelope
     math. Without a measurement, "approximation" is unfalsifiable marketing.

  This is a dev decision, not a builder one. Surface during Phase 3 discuss.

### Attack 8 — `tokenizer.Get` first-call latency is masked but not eliminated

- **Severity:** nit
- **Where:** Unit B.1 (RiskNote on first-call I/O)
- **Counterexample:** The plan says:
  > `tokenizer.Get` may do file I/O on first call (loading the BPE merge table
  > from an embedded asset). Subsequent calls via our singleton bypass this.
  > First-call latency is acceptable.

  Verify "embedded asset" claim — Context7 doesn't explicitly state where
  tiktoken-go stores its BPE tables. If it's `embed.FS`, no real I/O. If it's
  a Go-source-baked `[]byte` literal, no I/O either, just heap allocation.
  Most likely one of these — the planner is probably right.

  But if it's loaded lazily from disk (e.g. a `.tiktoken` file in a default
  location) the first `--tokens` invocation pays a latency hit that doesn't show
  up in any test (tests use a tiny fixture; latency is dominated by tokenizer
  init, not by `Count` calls). A user running `rak --tokens .` for the first
  time after install might see a 200ms+ stall they don't see on subsequent runs.

  **Mitigation:** Add a B.1 spike acceptance: *"Builder MUST verify by reading
  the tiktoken-go source or running a smoke benchmark that `tokenizer.Get`
  first-call latency is bounded (< 100ms) on a clean filesystem. Document the
  observed latency in BUILDER_WORKLOG.md. If > 500ms, escalate as a finding."*

### Attack 9 — `io.ReadAll` then `string(buf)` doubles memory; no streaming alternative explored

- **Severity:** nit (planner already accepts; flagging for documentation)
- **Where:** Unit B.2 RiskNotes
- **Counterexample:** Plan says:
  > `io.ReadAll` into a `[]byte` then `string(buf)` copies the entire file into
  > memory. For very large files (>100 MB) this is a significant allocation.

  Math: 100 MB file → 100 MB for `buf` (`[]byte`) + 100 MB for `string(buf)`
  (Go strings are immutable, conversion copies). Peak ~200 MB per file. A walk
  over a 1 GB git repo with several large vendored JS bundles can momentarily
  spike to several hundred MB.

  Two mitigations available without rewriting:
  1. `unsafe.String(unsafe.SliceData(buf), len(buf))` to alias `buf` as a
     string without copying — but the resulting string is read-only-but-still-
     pointing-at-the-`[]byte`, so the underlying `buf` must not be mutated or
     freed early. The library doesn't document whether `Count` keeps a reference.
     Probably safe but requires verification. Adds ~30 LOC for the unsafe path
     + a comment block; saves 50% peak memory. Worth it? Marginal at v0.2.0
     scale.
  2. Chunked tokenization with a known-safe split point — fundamentally wrong
     for BPE (merges can cross arbitrary byte boundaries). Skip.

  **Mitigation:** Acknowledge in B.2 RiskNote that `unsafe.String` is a
  v0.3 optimization path. Add an acceptance bullet: *"Document the 2x peak
  memory profile in the function's godoc comment so users running rak on
  very large files (>100 MB single-file) know the cost."*

### Attack 10 — B/C/D root.go serialization plan is missing

- **Severity:** concern (orchestrator hazard, not planner fault per se)
- **Where:** PLAN.md § Notes (cross-stream coordination paragraph)
- **Counterexample:** The Notes paragraph says:
  > Streams B, C, D all add new flags to `cmd/rak/root.go`. The planner should
  > make the cmd/rak flag-wiring unit explicit and self-contained so the
  > orchestrator can serialize it against C and D at build time.

  B.5 is the cmd/rak unit for stream B. Stream C presumably has a C.N unit that
  adds `--workers`. Stream D presumably has D.N that adds `--files-from`. All
  three touch root.go's `rootFlags`, the validSortKeys map, the cobra command
  factory, the Example string, and the `runRoot` body.

  Without a declared "first to land" stream, the second and third streams will
  rebase against a moving target. Three-way merge conflicts in root.go are not
  insurmountable but are an orchestrator burden.

  Counterexample: If B.5 lands first, the builder for C's `--workers` flag has
  to add to the validSortKeys-style structures plus the new `tokens` flag's
  PreRunE check (`--sort tokens requires --tokens` from Attack 3). If C lands
  first instead, C's `--workers` doesn't touch validSortKeys but DOES touch
  walkAndCount's signature (adds a workerCount parameter). B.5 then has to
  rebase its walkAndCount signature additions against C's. Either ordering
  works but the order matters for who-rebases-whom.

  **Mitigation:** Add to PLAN.md Notes:
  *"Cross-stream ordering: orch dispatches B.5 → C.cmd → D.cmd serially against
  cmd/rak/root.go. Internal-package work (B.1-B.4, C.internal-pkg, D.internal-pkg)
  can run in parallel; only the cmd/rak units serialize. If C or D's plan
  changes the walkAndCount signature, that change MUST land before B.5's
  walkAndCount edit."*

  Or: orchestrator declares this in the build dispatch prompt; either works,
  but a planner-level note prevents the orchestrator from forgetting.

### Attack 11 — Invalid UTF-8 file behavior under `Count` is undefined

- **Severity:** nit
- **Where:** Unit B.1 / B.2 (no test covers it)
- **Counterexample:** `counting.Count` (existing code) uses `bufio.NewReader.ReadRune`
  which returns `RuneError` (U+FFFD) for invalid UTF-8 sequences and silently
  proceeds. `Counts.Chars` is incremented for each `RuneError`.

  `tokenizer.Codec.Count(string(buf))` on an invalid-UTF-8 `[]byte` reinterpreted
  as `string` — Go's `string([]byte)` conversion doesn't validate UTF-8; it just
  reinterprets bytes. The BPE encoder may panic, return an error, or silently
  produce garbage counts depending on tiktoken-go's internal handling of invalid
  byte sequences.

  Most real-world files are valid UTF-8 (binary files are NUL-detected and skipped
  upstream). But Latin-1 / Windows-1252 source files (common in legacy codebases)
  contain bytes 0x80-0xFF that don't form valid UTF-8 — these will reach
  `CountWithTokens`.

  **Mitigation:** Add a B.1 test: *"`TestCount_InvalidUTF8`: input contains the
  byte sequence `0xC3 0x28` (invalid 2-byte UTF-8). Verify `codec.Count` does
  not panic and either returns an error (acceptable) or a finite int (also
  acceptable). If it panics, B.1 must add a `utf8.Valid(buf)` guard with a
  fallback to a `Tokens: 0, error nil` return path."*

### Attack 12 — `tiktoken-go/tokenizer` supply-chain stability is unverified

- **Severity:** nit
- **Where:** Unit B.1 (library choice)
- **Counterexample:** The plan picks `github.com/tiktoken-go/tokenizer`.
  Alternative `github.com/pkoukk/tiktoken-go` exists and has historically had
  more community uptake. Neither is officially maintained by OpenAI; both are
  community ports of the Python `tiktoken` library.

  If `tiktoken-go/tokenizer` goes unmaintained mid-v0.2.0 lifecycle:
  - Security: BPE merge-table data is static, low risk.
  - Bug-fix path: upstream may need a fork.
  - Migration cost: swapping libraries means re-pinning all known-fixture
    token counts (different libraries may tokenize edge cases slightly
    differently due to BPE merge-rule corner cases).

  The plan does not consider library reputation or last-release date.

  **Mitigation:** Add to B.1 RiskNotes: *"Library health: verify last release
  date and open-issue count for `tiktoken-go/tokenizer` before pinning. If
  >12 months stale or >50 unresolved issues, escalate to dev as a Phase 3
  discuss item — alternative `pkoukk/tiktoken-go` may be preferable."*

---

## Summary of mitigations to fold into PLAN.md (planner brief)

1. **B.4 — Pin embedded JSON shape**: `Tokens` MUST appear inside the nested
   `"counts":{}` object, never as a sibling of `"path"` in `directoryJSON`. Add
   acceptance bullet.
2. **B.4 — TOON snapshot regeneration**: builder verifies toon-go zero-value
   behavior on `Tokens=0` in Round 1; all 7 existing tape outputs in
   `docs/tapes/*.tape` must be re-recorded as part of B.6. Add acceptance
   bullets.
3. **B.5 — `--sort tokens` requires `--tokens`**: PreRunE rejects the
   combination with a clear error. Add acceptance criterion + test.
4. **B.1 — Concurrent-Count test**: `TestGet_ConcurrentCount` (100 goroutines ×
   1000 iters under `-race`). If it fails, B.1 wraps `Count` with a mutex.
5. **B.1 — Return type clarity**: `internal/tokens.Get() (counting.TokenCounter,
   error)` — return the interface-satisfied type, let `internal/tokens` import
   `internal/counting`. Simpler than implicit duck-typing at cmd/rak.
6. **B.5 — `--help` caveat strength**: flag description includes the word
   "approximation" + a Claude-billing-difference cue.
7. **B.5 + B.6 — Claude approximation measurement**: either ship
   `--tokens-encoding=o200k_base` now OR cite measured cl100k_vs_Claude delta in
   the README caveat. Dev decision — surface in Phase 3.
8. **B.1 — First-call latency check**: builder verifies tokenizer.Get latency
   < 100ms; documents observed value.
9. **B.2 — Document 2x peak memory profile** in godoc.
10. **PLAN.md Notes — cross-stream serialization**: explicitly declare B.5
    serializes against C.cmd and D.cmd; internal-pkg work is parallel.
11. **B.1 — Invalid UTF-8 test**: `TestCount_InvalidUTF8` to verify no panic.
12. **B.1 — Library health check**: verify last-release date and open-issue
    count for tiktoken-go/tokenizer.

## Attacks attempted, NOT landed (REFUTED / EXHAUSTED)

- **JSON `omitempty` on `int64`**: REFUTED — stdlib `encoding/json` does omit
  zero-value `int64` when tagged `omitempty`. Verified via existing F33-pattern
  in `directoryJSON.Files` (`json:"files,omitempty"` on `int64` — see
  `internal/render/json.go:63`).
- **Mage-bypass scan**: EXHAUSTED — plan correctly uses `mage` targets
  throughout; no raw `go test` / `go build` references.
- **`init()` side-effect attack**: EXHAUSTED — no `init()` functions introduced
  in any B unit per plan text.
- **Cycle attack on import DAG**: EXHAUSTED — proposed `internal/tokens` is a
  leaf (or, per Attack 5 mitigation, depends only on `internal/counting`, also
  a leaf). No cycle possible.
- **`fmt.Errorf` without `%w`**: REFUTED for surfaces named in plan — design
  decisions consistently say "wrap with context"; no inspection of error
  bodies via string match.
