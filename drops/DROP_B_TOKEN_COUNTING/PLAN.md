# DROP_B — TOKEN_COUNTING

**State:** planning
**Tier:** A
**Blocked by:** —
**Paths (expected):** NEW internal/tokens/tokens.go + tokens_test.go, internal/counting/counting.go, internal/summary/summary.go, internal/render/{toon,human,json}.go, cmd/rak/root.go, magefile.go (mage addDep call), README.md, main/docs/tapes/tokens.tape (NEW), main/docs/tokens.gif (NEW)
**Packages (expected):** NEW internal/tokens, internal/counting, internal/summary, internal/render, cmd/rak
**PLAN.md ref:** — (top-level PLAN.md removed at v0.1.0 ship; see memory `session_handoff_2026_05_16_v020_planning.md`)
**Workflow:** main/drops/WORKFLOW.md
**Started:** 2026-05-16
**Closed:** —

## Scope

Add token counting — the **highest-value v0.2.0 feature** for rak's "wc++ for LLMs" positioning. Per dev confirmation (Decision 11 in deleted PLAN.md, reaffirmed 2026-05-16):

- **Library**: `github.com/tiktoken-go/tokenizer`. Add via `mage addDep`.
- **Default encoder**: `cl100k_base` (GPT-3.5 / GPT-4 family). Document the Claude-approximation caveat in README + `--help`.
- **New flag**: `--tokens` opts into token counting (counting is slow vs byte-counting, so off by default; if dev later wants always-on, re-evaluate).
- **New flag**: `--tokens-encoding` selects the tokenizer vocabulary: `cl100k` (default, GPT-3.5/4) or `o200k` (GPT-4o; closer Claude approximation). Validated in `PersistentPreRunE`. No-op without `--tokens`.
- **New column** in TOON `directories` tabular block: `tokens` after `chars`.
- **New JSON field**: `directoryJSON.Tokens int64` with `json:",omitempty"` so older consumers don't choke.
- **New `--human` row**: `Tokens` line in each per-directory block + overall total.
- **New `total.tokens`** in TOON + JSON + human.
- **New `--sort tokens`** key (numeric, defaults desc).

**Feature trio (mandatory per memory `feedback_rak_docs_and_gifs_before_pr.md`):**

1. VHS demo: `main/docs/tapes/tokens.tape` + generated `main/docs/tokens.gif`. Embed near the README narrative section that introduces `--tokens`.
2. README example: add `rak --tokens .` to "Common invocations" + a dedicated "Token counting" narrative section.
3. Cobra `Example:` entries in `cmd/rak/root.go` so `rak --help` surfaces both `rak --tokens .` and `rak --tokens --tokens-encoding o200k .`.

## Planner

### Unit B.1 — Add dep + internal/tokens package

**State:** todo
**Paths:** `go.mod`, `go.sum`, `internal/tokens/tokens.go`, `internal/tokens/tokens_test.go`
**Packages:** `internal/tokens` (new), module-level go.mod/go.sum
**Blocked by:** —

**Scope.** Run `mage addDep github.com/tiktoken-go/tokenizer` to add the dep. Create
`internal/tokens/tokens.go` exporting a `Get(encoding string) (counting.TokenCounter, error)`
function backed by two per-encoding singletons (`cl100k_base` and `o200k`), each initialized
via its own `sync.Once`.

**Design decisions (do not relitigate):**

- No rak-local `Codec` interface. `internal/tokens.Get(encoding string)` returns
  `counting.TokenCounter` directly. `internal/tokens` imports `internal/counting` (a downward
  import into a leaf — no cycle). Callers (`cmd/rak`) import both packages but hold the result
  typed as `counting.TokenCounter`.
- Dual-encoding singletons: two `sync.Once` + `tokenizer.Codec` pairs, one for `cl100k_base`
  and one for `o200k`. `Get(encoding string) (counting.TokenCounter, error)` switches on the
  encoding string to select the correct pair. Supported encoding strings: `"cl100k"` and
  `"o200k"`. Any other value returns `fmt.Errorf("unsupported encoding: %q", encoding)`.
- `Get` returns an error rather than panicking on unsupported encoding. The flag's PreRunE
  validation (B.5) ensures only valid strings reach `Get`, so runtime errors here are
  programmer errors — still, return the error rather than panic.
- No `io.Reader` API exists in the library — callers must pass a `string`. The buffer-to-string
  read is the caller's responsibility (done in `internal/counting`, B.2).

**Key API facts (Context7 /tiktoken-go/tokenizer):**

- `tokenizer.Get(tokenizer.Cl100kBase) (tokenizer.Codec, error)` — returns shared encoder for
  cl100k. `tokenizer.Get(tokenizer.O200kBase)` — returns shared encoder for o200k.
- `codec.Count(text string) (int, error)` — returns token count without allocating the full
  token ID slice; cheaper than `Encode` for count-only use.
- No streaming / `io.Reader` API. Entire file content must be read into a buffer first.
- `tokenizer.Cl100kBase` is the constant for the GPT-3.5/4 encoding (approximation for Claude).
- `tokenizer.O200kBase` is the constant for the GPT-4o encoding (closer Claude approximation).
- At build time, builder confirms `tiktoken-go/tokenizer` last commit + open-issue count and
  records the finding in `BUILDER_WORKLOG.md`. Also records first-call `tokenizer.Get` latency
  measurement (expected: embedded BPE asset load; subsequent calls are cached).

**Acceptance:**

- `mage build` passes (entire module compiles including `internal/tokens`).
- `mage test-pkg internal/tokens` passes with:
  - `TestGet_ReturnsSingleton` (two `Get("cl100k")` calls return the same underlying pointer).
  - `TestGet_Empty` (empty string, cl100k → 0 tokens, no error).
  - `TestGet_KnownFixture_Cl100k` (fixed ASCII string, pinned expected token count —
    e.g. `"hello world"` → 2 tokens in cl100k_base; record expected count in test comment).
  - `TestGet_KnownFixture_O200k` (same or similar fixture, pinned count for o200k; counts may
    differ from cl100k — record both).
  - `TestGet_Multibyte` (UTF-8 emoji string, cl100k and o200k both do not error).
  - `TestGet_UnsupportedEncoding` (`Get("gpt2")` returns non-nil error containing "unsupported").
  - `TestGet_ConcurrentCount` (100 goroutines × 1000 iterations each calling
    `Get("cl100k")` then `counter.Count("hello")` — must pass under `-race` flag). If the
    test fails under race detector, B.1 wraps the `Count` call with a mutex before delivery.
- The `internal/tokens` package imports `internal/counting` (for `counting.TokenCounter`) and
  no other `internal/` package.

**RiskNotes:**

- `tokenizer.Get` goroutine safety for `Count`: BPE encoders are read-only after init; the
  returned `Codec` is assumed goroutine-safe for concurrent `Count` calls. This is verified
  in B.1 by `TestGet_ConcurrentCount` (100 goroutines × 1000 iters under `-race`). If the
  test fails, B.1 wraps `Count` with a mutex rather than deferring the finding to Drop 8.1.
- `tokenizer.Get` may do file I/O on first call (loading the BPE merge table from an embedded
  asset). Subsequent calls via our singletons bypass this. Builder measures first-call latency
  and records it in `BUILDER_WORKLOG.md`.
- Known fixture token counts (`TestGet_KnownFixture_Cl100k`, `TestGet_KnownFixture_O200k`) are
  regression anchors: if tiktoken-go updates its BPE tables, these tests will alert. Record the
  expected count in a comment so future maintainers can update intentionally.

---

### Unit B.2 — Extend internal/counting with token-aware counter

**State:** todo
**Paths:** `internal/counting/counting.go`, `internal/counting/counting_test.go`
**Packages:** `internal/counting`
**Blocked by:** B.1

**Scope.** Add `Tokens int64` to `counting.Counts` (must be the last field — see constraint
below). Add `TokenCounter` interface and `CountWithTokens(io.Reader, TokenCounter) (Counts, error)`
function to `internal/counting`. The existing `Count(io.Reader)` function is unchanged.

**Design decisions (do not relitigate):**

- `Tokens int64` goes at the END of `Counts` — after `Chars` — to avoid breaking the
  `encoding/json` field-order dependency in `internal/render/json.go`. The `Render` path
  encodes `counting.Counts` directly; field order is the JSON key order.
- `Tokens` carries `json:",omitempty"` via a struct tag so zero-value (non-`--tokens` runs)
  does not add a `"Tokens":0` key to JSON output. This preserves snapshot test compatibility
  for existing tests that pin `--json` output.
- `TokenCounter` interface is defined in `internal/counting` (zero internal deps). The
  returned value from `internal/tokens.Get()` satisfies this interface at the call site.
  `internal/counting` itself does NOT import `internal/tokens` — dependency flows downward
  only (tokens → counting). Interface definition:
  ```
  type TokenCounter interface {
      Count(text string) (int, error)
  }
  ```
- `CountWithTokens` reads the reader to completion via `io.ReadAll`, then calls
  `tc.Count(string(buf))`. The `io.ReadAll + string(buf)` pattern allocates the entire file
  content twice (once as `[]byte`, once as `string`). Document this 2× peak-memory behavior
  in the function's godoc comment as a known v0.2.0 characteristic; v0.2.1+ may chunk if a
  user reports issues. It also runs the same byte/line/word/char logic from `Count`.
  Implementation may either call `Count` on the same bytes (reusing a `bytes.Reader`) or
  inline both paths in one pass. Either is acceptable; the builder chooses based on
  implementation simplicity.
- `Count`'s existing comment "Field declaration order ... is load-bearing" should be updated
  to reflect the `Tokens` addition and the `omitempty` tag.
- **Constraint: the builder MUST NOT modify the existing `Count` function body.** Any
  refactoring that changes `Count`'s behavior (even byte-for-byte-equivalent rewrites) is
  out of scope for B.2. All existing `Count` tests must pass byte-for-byte.

**Acceptance:**

- `mage build` passes.
- `mage test-pkg internal/counting` passes with: all existing tests still pass (zero modification
  to existing `Count` behavior); new tests for `CountWithTokens`:
  - `TestCountWithTokens_Empty` (0 tokens, no error).
  - `TestCountWithTokens_KnownText` (pinned fixture with known token count).
  - `TestCountWithTokens_Multibyte` (UTF-8 emoji string — no error, count ≥ 1).
  - `TestCount_InvalidUTF8` (Latin-1 byte sequence passed as an `io.Reader` — must not panic;
    either returns a count or a non-panic error from the tokenizer; document the behavior in
    a test comment).
  - A `stubCounter` type implementing `TokenCounter` is used in all new tests to avoid
    importing `internal/tokens` (keeps counting tests dep-free).
- `counting.Count` behavior is byte-for-byte unchanged — returns `Tokens: 0` in all cases.
- JSON encoding of a `Counts{Bytes:5}` value produces `{"Bytes":5,"Lines":0,"Words":0,"Chars":0}`
  (Tokens omitted because zero) — existing JSON snapshot tests continue to pass without
  modification.

**RiskNotes:**

- `io.ReadAll` into a `[]byte` then `string(buf)` copies the entire file into memory. For very
  large files (>100 MB) this is a significant allocation. Acceptable for v0.2.0 — document
  in the function's godoc comment as a known behavior.
- The existing `Count` function's buffer-scanning loop and `CountWithTokens`'s `io.ReadAll`
  approach differ in memory allocation profile. If the builder chooses to unify them (reading
  all bytes first, then scanning), that is acceptable as long as `Count` behavior is
  bit-for-bit identical to existing behavior (all existing tests still pass).

---

### Unit B.3 — Add SortTokens to internal/summary

**State:** todo
**Paths:** `internal/summary/sort.go`, `internal/summary/summary_test.go`
**Packages:** `internal/summary`
**Blocked by:** B.2

**Scope.** Add `SortTokens SortKey = "tokens"` constant to `sort.go`. Extend `SortDirs`
switch to handle `SortTokens` (sorts on `a.Counts.Tokens`). Extend `effectiveAsc` to treat
`SortTokens` like the other numeric keys (default descending). Update `validSortKeys` — wait,
that map is in `cmd/rak/root.go`, not in summary. No change needed there in this unit.
Update `summary_test.go` to: (a) replace `SortKey("tokens")` in `TestSortDirs_UnknownKey_Panics`
with a genuinely unknown key (e.g. `SortKey("chars")`), and (b) add `TestSortDirs_Tokens_Default`
and `TestSortDirs_Tokens_Asc` covering the new sort key.

**Design decisions (do not relitigate):**

- `SortTokens` sorts on `a.Counts.Tokens`, which is available on `summary.Directory` via the
  embedded `counting.Counts` field — no new field is needed on `summary.Directory`.
- No change to `summary.Directory` field order or `Summary` struct — they are unaffected.
- `effectiveAsc` for `SortTokens`: same as numeric keys (default descending, `asc` flag
  flips to ascending). No special casing needed.
- The sort.go comment block "Valid keys in v0.1.0 are: SortLines, SortFiles, SortBytes,
  SortPath. 'tokens' is intentionally absent" must be updated to reflect v0.2.0 addition.

**Acceptance:**

- `mage build` passes.
- `mage test-pkg internal/summary` passes with all existing tests plus new token-sort tests.
- `TestSortDirs_UnknownKey_Panics` uses a non-"tokens" unknown key and still panics.
- `TestSortDirs_Tokens_Default`: dirs with `Counts.Tokens` 10/5/20 → descending [20, 10, 5].
- `TestSortDirs_Tokens_Asc`: same dirs, `asc=true` → ascending [5, 10, 20].

---

### Unit B.4 — Add Tokens field to all three renderers

**State:** todo
**Paths:** `internal/render/toon.go`, `internal/render/human.go`, `internal/render/json.go`, `internal/render/render_test.go`
**Packages:** `internal/render`
**Blocked by:** B.2

**Scope.** Update all three renderers to emit `Tokens` when non-zero. All changes are in
one package — one unit is appropriate given the mechanical nature of the additions.

**TOON renderer (`toon.go`):**
- Add `Tokens int64 \`toon:"tokens"\`` field to `toonCounts` (after `Chars`) and to
  `toonDirectory` (after `Chars`).
- `RenderTree`: populate `Tokens` from `d.Counts.Tokens` in each row and from `s.Total.Tokens`
  in the `toonCounts` total block.
- `Render`: populate `Tokens` from `counts.Tokens`.
- TOON `omitempty` behavior: check whether toon-go respects an `omitempty` tag on int64 fields.
  If it does, add `toon:"tokens,omitempty"` so zero-token runs suppress the column. If toon-go
  does not support `omitempty` on numeric types, always emit the column. **Builder must verify
  via `mage build` + a quick manual `mage run`; note the finding in BUILDER_WORKLOG.md.**

**Human renderer (`human.go`):**
- Extend `countsKV` to append a `{Label: "Tokens", Value: strconv.FormatInt(counts.Tokens, 10)}`
  pair when `counts.Tokens != 0`. Same for `dirKV`.
- Grand-total block via `countsKV("total", s.Total)` will automatically pick up Tokens.

**JSON renderer (`json.go`):**
- No change needed to `counting.Counts` (already handled in B.2 with `json:",omitempty"`).
  The `directoryJSON` embeds `counting.Counts` via `Counts counting.Counts \`json:"counts"\`` —
  the `Tokens` field is automatically present in the nested `counts` object when non-zero.
  `json.go` itself may need no change if the `Counts` field in `directoryJSON` round-trips
  the omitempty behavior correctly. **Builder must verify.**
- `treeJSON.Total` is also `counting.Counts` — same automatic behavior.

**Snapshot tests (`render_test.go`):**
- Existing snapshot tests must continue to pass (Tokens = 0 → omitted from JSON; toon-go
  behavior TBD — if zero columns are emitted, existing TOON snapshots will break and must
  be updated to include the zero-token column).
- Add new snapshot tests for `--tokens` output in each renderer with a known fixture
  (non-zero Tokens count) to lock the output format.

**RiskNotes:**

- The toon-go `omitempty` question is the primary risk. If toon-go always emits zero-value
  int64 columns, existing TOON snapshot tests will need updating to include a `tokens|0`
  column. The builder must handle this in the round-1 implementation.
- Human renderer's `countsKV` conditional emission (`if counts.Tokens != 0`) is straightforward
  but must be consistent: the grand-total block uses `countsKV("total", s.Total)` — if Total
  Tokens is 0 (no `--tokens` flag), the Tokens row is suppressed. Correct behavior.

**Acceptance:**

- `mage build` passes.
- **toon-go zero-column behavior must be verified BEFORE any other renderer change.** Builder
  runs `mage run -- --help` then a quick manual `mage run -- .` (no flags) to observe whether
  toon-go emits a zero-value `tokens` column. Finding is recorded in `BUILDER_WORKLOG.md` as
  the first entry in this unit. If toon-go emits zero columns unconditionally, all TOON
  snapshot tests plus ALL 7 existing VHS tapes must be re-recorded before this unit can close:
  `default-toon.tape`, `human.tape`, `json.tape`, `lang-filter.tape`, `sort-files.tape`,
  `max-files.tape`, `version.tape`.
- `mage test-pkg internal/render` passes with all existing and new tests.
- TOON output with `Tokens=1234` includes `tokens | 1234` column in `directories` array and
  `total.tokens | 1234` in the nested total block.
- Human output with `Tokens=1234` includes a `Tokens: 1234` KV pair.
- JSON output with `Tokens=1234` includes `"tokens":1234` inside the nested `"counts":{}` object,
  NOT as a sibling of `"path"`. Verify via `internal/render/json.go` `directoryJSON` struct
  shape: `Tokens` must be a field of `counting.Counts` (embedded in `directoryJSON.Counts`),
  never a top-level field directly on `directoryJSON`. Adding a top-level `Tokens` field to
  `summary.Directory` would silently break existing JSON snapshot tests — this is explicitly
  prohibited; the B.2 `Counts` field-extension approach is the only sanctioned path.
- All three renderers suppress the Tokens field/column when `Tokens=0` (or, for TOON, emit it
  at zero if toon-go doesn't support omitempty — consistency with toon-go behavior is
  the acceptance bar, not a specific value).
- Builder records the toon-go `omitempty` finding in `BUILDER_WORKLOG.md` before proceeding.

---

### Unit B.5 — CLI flag wiring and walk integration

**State:** todo
**Paths:** `cmd/rak/root.go`, `cmd/rak/root_test.go`, `cmd/rak/integration_test.go`
**Packages:** `cmd/rak`
**Blocked by:** B.3, B.4

**Scope.** Wire `--tokens` and `--tokens-encoding` flags, update `validSortKeys`, update
`addCounts`, plumb token counting through `walkAndCount` and `countFile`, handle stdin path,
update cobra `Example:`.

**Changes in `root.go`:**

1. Add `tokens bool` and `tokensEncoding string` fields to `rootFlags`.
2. Register `--tokens` flag:
   ```
   cmd.Flags().BoolVar(&flags.tokens, "tokens", false,
       "count tokens (approximation; cl100k is GPT-3.5/4 vocabulary, o200k is GPT-4o vocabulary,
       both approximate Claude tokenization; slower than byte counting)")
   ```
3. Register `--tokens-encoding` flag:
   ```
   cmd.Flags().StringVar(&flags.tokensEncoding, "tokens-encoding", "cl100k",
       "tokenizer encoding for --tokens: cl100k (default, GPT-3.5/4) or o200k (GPT-4o)")
   ```
4. Add `--tokens-encoding` validation to `PersistentPreRunE` (alongside existing sort-key
   validation): if `flags.tokens` is true and `flags.tokensEncoding` is not in
   `{"cl100k", "o200k"}`, return `fmt.Errorf("--tokens-encoding %q is not valid; use cl100k
   or o200k", flags.tokensEncoding)`.
5. Add `--sort tokens` requires `--tokens` check to `PersistentPreRunE`: if `flags.sort ==
   "tokens"` and `!flags.tokens`, return `fmt.Errorf("--sort tokens requires --tokens")`.
6. Add `"tokens"` to `validSortKeys` map.
7. Pass `flags.tokens` and `flags.tokensEncoding` into `runDirectoryOpts` (new fields
   `countTokens bool`, `tokensEncoding string`).
8. Call `tokens.Get(flags.tokensEncoding)` exactly once in `runRoot`, immediately after
   `resolveRenderer`, when `flags.tokens` is true. Wrap error:
   `fmt.Errorf("init token counter: %w", err)`. Store result as `runDirectoryOpts.tokenCounter`
   (`counting.TokenCounter`; nil when `--tokens` is false).
9. Pass `flags.tokens` into the stdin path: when `flags.tokens` is true and `tokenCounter`
   is non-nil, call `counting.CountWithTokens(c.InOrStdin(), tokenCounter)` instead of
   `counting.Count(c.InOrStdin())`.
10. Update `addCounts` to sum `Tokens` field:
    ```
    return counting.Counts{
        Bytes:  a.Bytes + b.Bytes,
        Lines:  a.Lines + b.Lines,
        Words:  a.Words + b.Words,
        Chars:  a.Chars + b.Chars,
        Tokens: a.Tokens + b.Tokens,
    }
    ```
11. Update `walkAndCount` / `countFile`: `countFile` accepts an optional `counting.TokenCounter`
    (nil = no token counting) so the `internal/counting` interface drives the boundary. When
    non-nil, call `counting.CountWithTokens(rc, tokenCounter)`; otherwise call `counting.Count(rc)`.
12. Update cobra `Example:` to add:
    ```
    # Count tokens (cl100k_base; Claude/GPT-3.5/4 approximation)
    rak --tokens .

    # Count tokens using GPT-4o vocabulary (o200k; closer Claude approximation)
    rak --tokens --tokens-encoding o200k .
    ```
13. Update `PersistentPreRunE` sort-key validation error message to include "tokens" in the
    valid-key list.

**Design decisions (do not relitigate):**

- `tokens.Get(flags.tokensEncoding)` is called once in `runRoot` when `flags.tokens` is true;
  the returned `counting.TokenCounter` is stored on `runDirectoryOpts.tokenCounter` and
  shared across the whole walk. No per-file re-init.
- `countFile` accepts an optional `counting.TokenCounter` (nil = no token counting) so the
  `internal/counting` package interface drives the boundary — `cmd/rak` imports both
  `internal/tokens` (for `tokens.Get()`) and `internal/counting` (for `CountWithTokens`
  and the `TokenCounter` interface).
- `--tokens-encoding` PreRunE validation runs only when `--tokens` is true; the flag is a
  no-op without `--tokens` and must not error in that case.
- No goroutine safety concern in v0.2.0 (single-threaded walk; B.1 verified concurrency
  safety regardless).
- Cross-stream sequencing: B.5 lands first in `cmd/rak/root.go`. Stream C's `--workers`/
  `--follow` flag-registration block rebases against B.5. Stream D's `--files-from` rebases
  against C. Each subsequent stream's flag-registration block is appended below the prior;
  PreRunE checks chained in declaration order.

**Acceptance:**

- `mage build` passes.
- `mage test-pkg cmd/rak` passes with: all existing tests; and:
  - `TestFlags_Tokens_Parse` — `--tokens` flag parsed to `rootFlags.tokens == true`.
  - `TestFlags_TokensEncoding_Default` — omitting `--tokens-encoding` yields `"cl100k"`.
  - `TestFlags_TokensEncoding_O200k` — `--tokens-encoding o200k` yields `"o200k"`.
  - `TestFlags_TokensEncoding_Invalid` — `--tokens-encoding gpt2` with `--tokens` returns
    error containing `"--tokens-encoding"` from `PersistentPreRunE`.
  - `TestFlags_TokensEncodingWithoutTokens` — `--tokens-encoding o200k` alone (no `--tokens`)
    does not error (the encoding flag is a no-op without `--tokens`).
  - `TestFlags_SortTokensRequiresTokens` — `rak --sort tokens .` without `--tokens` returns
    error `--sort tokens requires --tokens` from `PersistentPreRunE`.
  - New integration test: `rak --tokens <fixture>` produces non-zero token counts in output.
- `rak --help` shows the `--tokens` flag with the approximation caveat including both
  vocabulary descriptions: "cl100k is GPT-3.5/4 vocabulary, o200k is GPT-4o vocabulary,
  both approximate Claude tokenization."
- `rak --help` shows the `--tokens-encoding` flag listing valid values.
- Verified via `mage run -- --help | grep -- '--tokens'` and
  `mage run -- --help | grep -- '--tokens-encoding'` in worklog.
- `rak --tokens --sort tokens .` sorts by token count descending.
- `rak --tokens --sort tokens --sort-asc .` sorts ascending.
- `rak --tokens <stdin_fixture>` (pipe path) includes Tokens in the rendered output.
- `rak --sort badkey .` error message mentions "tokens" as a valid key.
- `addCounts` correctly accumulates Tokens (verified by integration test showing total equals
  sum of per-directory token counts).

---

### Unit B.6 — Feature trio: VHS tape, README, gif

**State:** todo
**Paths:** `docs/tapes/tokens.tape`, `docs/tokens.gif`, `README.md`
**Packages:** — (no Go packages)
**Blocked by:** B.5

**Scope.** Author the VHS tape, regenerate the gif, and update README. The cobra `Example:`
string update lives in B.5 (it is in `cmd/rak/root.go`). This unit covers only the
documentation and recording artifacts.

**VHS tape (`docs/tapes/tokens.tape`):**
- Record `rak --tokens .` run against a fixture or the repo itself.
- Follows the existing tape conventions in `docs/tapes/` (check existing tapes for Output,
  Set, Type, Screenshot preamble).
- Output: `docs/tokens.gif`.

**README (`README.md`):**
- Add `rak --tokens .` to the "Common invocations" table or list.
- Add a "Token counting" narrative section (after the existing counting section): explain
  `--tokens` and `--tokens-encoding`, note both encoders, document the **Claude-approximation
  caveat** verbatim: "Token counts use the `cl100k_base` (GPT-3.5/4) vocabulary by default.
  Use `--tokens-encoding o200k` for the GPT-4o vocabulary (a closer approximation for Claude).
  These are approximations — counts will differ from Claude's actual billing token count but
  serve as a useful ballpark."
- Embed `docs/tokens.gif` near the narrative section.

**Acceptance:**

- `docs/tokens.tape` exists and is syntactically valid (runnable with `vhs docs/tapes/tokens.tape`).
- `docs/tokens.gif` exists and is a valid GIF file (non-zero size).
- `README.md` contains the "Token counting" section with the Claude-approximation caveat
  covering both `cl100k` and `o200k` encodings.
- `README.md` "Common invocations" includes `rak --tokens .`.
- `rak --help` (via cobra `Example:` set in B.5) shows both `rak --tokens .` and
  `rak --tokens --tokens-encoding o200k .`. Verified via
  `mage run -- --help | grep -- '--tokens'` in worklog — not just assumed from B.5 passing.
- **If B.4 determined that toon-go emits zero-value columns unconditionally**, ALL 7 existing
  VHS tapes must be re-recorded in this unit before close (the toon column shape has changed):
  `default-toon.tape`, `human.tape`, `json.tape`, `lang-filter.tape`, `sort-files.tape`,
  `max-files.tape`, `version.tape`. Builder checks the `BUILDER_WORKLOG.md` toon-go finding
  from B.4 before deciding whether re-recording is required.

## Notes

**Cross-stream root.go sequencing**: Streams B, C, D all add new flags to `cmd/rak/root.go`.
Ordering: B.5 (token flags) lands first. Stream C's `--workers`/`--follow` flag-registration
block rebases against B.5. Stream D's `--files-from` flag-registration block rebases against
C. Each subsequent stream's flag-registration block is appended below the prior; PreRunE
checks are chained in declaration order. Internal-package work (`internal/tokens/`,
`internal/counting`, `internal/summary`, `internal/render/*`) is parallel-safe with the other
streams.

**Performance note**: tokenization is meaningfully slower than byte/line counting. Cache the
encoder across files (`tokenizer.Get(encoding)` returns a shared instance via our singletons);
do not re-instantiate per file. Tests must cover empty file, ASCII, multibyte (UTF-8 emoji),
known-token-count fixtures (one per encoding), and invalid UTF-8 (Latin-1 byte sequences) to
detect panics or regressions if tiktoken-go updates change behavior.

**Dual encoder**: `--tokens-encoding` defaults to `cl100k` (GPT-3.5/4 vocabulary). `o200k`
(GPT-4o vocabulary) is an opt-in for users who want a closer Claude approximation. Both are
approximations. PreRunE rejects any other value. The flag is a no-op without `--tokens`.

**toon-go zero-column finding**: If B.4's builder confirms toon-go emits zero-value int64
columns for all numeric fields regardless of value, then the toon output shape changes for
ALL non-`--tokens` runs (a `tokens | 0` column appears). In that case ALL 7 existing VHS
tapes must be re-recorded in B.6 and all existing TOON snapshot tests in `internal/render`
must be updated in B.4. This is the highest-risk finding in the drop — builder prioritizes
the verification in B.4 Round 1 before any other B.4 work.

**`TestGet_KnownFixture` pinned counts**: record the expected integer in a test comment with
the date verified. If tiktoken-go updates its BPE tables, the test will fail with the old
count — update intentionally, not silently.
