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
- **New column** in TOON `directories` tabular block: `tokens` after `chars`.
- **New JSON field**: `directoryJSON.Tokens int64` with `json:",omitempty"` so older consumers don't choke.
- **New `--human` row**: `Tokens` line in each per-directory block + overall total.
- **New `total.tokens`** in TOON + JSON + human.
- **New `--sort tokens`** key (numeric, defaults desc).

**Feature trio (mandatory per memory `feedback_rak_docs_and_gifs_before_pr.md`):**

1. VHS demo: `main/docs/tapes/tokens.tape` + generated `main/docs/tokens.gif`. Embed near the README narrative section that introduces `--tokens`.
2. README example: add `rak --tokens .` to "Common invocations" + a dedicated "Token counting" narrative section.
3. Cobra `Example:` entry in `cmd/rak/root.go` so `rak --help` surfaces the invocation pattern.

## Planner

### Unit B.1 — Add dep + internal/tokens package

**State:** todo
**Paths:** `go.mod`, `go.sum`, `internal/tokens/tokens.go`, `internal/tokens/tokens_test.go`
**Packages:** `internal/tokens` (new), module-level go.mod/go.sum
**Blocked by:** —

**Scope.** Run `mage addDep github.com/tiktoken-go/tokenizer` to add the dep. Create
`internal/tokens/tokens.go` exporting a `Codec` interface and a package-level `Get()` function
that returns the shared `cl100k_base` encoder, cached via `sync.Once`.

**Design decisions (do not relitigate):**

- `Codec` interface lives in `internal/tokens`, not in `internal/counting`, to avoid circular
  import confusion. `internal/counting` defines its own minimal `TokenCounter` interface
  (see B.2); `internal/tokens.Codec` satisfies it at the call site.
- Package-level singleton: `var codec tokenizer.Codec` initialized once via `sync.Once`.
  `Get() (tokenizer.Codec, error)` returns `(singleton, nil)` after init or the init error if
  the first `tokenizer.Get(tokenizer.Cl100kBase)` call failed.
- No `io.Reader` API exists in the library — callers must pass a `string`. The buffer-to-string
  read is the caller's responsibility (done in `internal/counting`, B.2).

**Key API facts (Context7 /tiktoken-go/tokenizer):**

- `tokenizer.Get(tokenizer.Cl100kBase) (tokenizer.Codec, error)` — returns shared encoder.
- `codec.Count(text string) (int, error)` — returns token count without allocating the full
  token ID slice; cheaper than `Encode` for count-only use.
- No streaming / `io.Reader` API. Entire file content must be read into a buffer first.
- `tokenizer.Cl100kBase` is the constant for the GPT-3.5/4 encoding (approximation for Claude).

**Acceptance:**

- `mage build` passes (entire module compiles including `internal/tokens`).
- `mage test-pkg internal/tokens` (or `go test -race ./internal/tokens/...` via mage target)
  passes with: `TestGet_ReturnsSingleton` (two calls return identical pointer), `TestGet_Empty`
  (empty string → 0 tokens, no error), `TestGet_KnownFixture` (fixed ASCII string with a
  pinned expected token count to detect library behavior change — e.g. `"hello world"` →
  2 tokens in cl100k_base), `TestGet_Multibyte` (UTF-8 emoji string does not error).
- The `internal/tokens` package has zero imports from other `internal/` packages (leaf node).

**RiskNotes:**

- `tokenizer.Get` goroutine safety for `Count`: BPE encoders are read-only after init; the
  returned `Codec` is assumed goroutine-safe for concurrent `Count` calls. This assumption is
  acceptable for v0.2.0 (single-goroutine walk). Parallel walk (Drop 8.1) should re-verify.
- `tokenizer.Get` may do file I/O on first call (loading the BPE merge table from an embedded
  asset). Subsequent calls via our singleton bypass this. First-call latency is acceptable.
- Known fixture token counts (`TestGet_KnownFixture`) are regression anchors: if
  tiktoken-go updates its BPE tables, these tests will alert. Record the expected count in a
  comment so future maintainers can update intentionally.

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
- `TokenCounter` interface is defined in `internal/counting` to keep the package dep-free
  from `internal/tokens`:
  ```
  type TokenCounter interface {
      Count(text string) (int, error)
  }
  ```
  `internal/tokens.Codec` satisfies this interface automatically (duck typing).
- `CountWithTokens` reads the reader to completion via `io.ReadAll`, then calls
  `tc.Count(string(buf))`. It also runs the same byte/line/word/char logic from `Count`.
  Implementation may either call `Count` on the same bytes (reusing a `bytes.Reader`) or
  inline both paths in one pass. Either is acceptable; the builder chooses based on
  implementation simplicity.
- `Count`'s existing comment "Field declaration order ... is load-bearing" should be updated
  to reflect the `Tokens` addition and the `omitempty` tag.

**Acceptance:**

- `mage build` passes.
- `mage test-pkg internal/counting` passes with: all existing tests still pass; new tests for
  `CountWithTokens`: `TestCountWithTokens_Empty` (0 tokens), `TestCountWithTokens_KnownText`
  (pinned fixture), `TestCountWithTokens_Multibyte` (UTF-8 emoji). A `stubCounter` type
  implementing `TokenCounter` is used in tests to avoid importing `internal/tokens` (keeps
  counting tests dep-free).
- `counting.Count` behavior is unchanged — returns `Tokens: 0` in all cases (the zero value
  due to `omitempty` keeps JSON output clean for non-`--tokens` runs).
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
- `mage test-pkg internal/render` passes with all existing and new tests.
- TOON output with `Tokens=1234` includes `tokens | 1234` column in `directories` array and
  `total.tokens | 1234` in the nested total block.
- Human output with `Tokens=1234` includes a `Tokens: 1234` KV pair.
- JSON output with `Tokens=1234` includes `"Tokens":1234` in the nested `"counts"` object.
- All three renderers suppress the Tokens field/column when `Tokens=0` (or, for TOON, emit it
  at zero if toon-go doesn't support omitempty — consistency with toon-go behavior is
  the acceptance bar, not a specific value).

---

### Unit B.5 — CLI flag wiring and walk integration

**State:** todo
**Paths:** `cmd/rak/root.go`, `cmd/rak/root_test.go`, `cmd/rak/integration_test.go`
**Packages:** `cmd/rak`
**Blocked by:** B.3, B.4

**Scope.** Wire `--tokens` flag, update `validSortKeys`, update `addCounts`, plumb token
counting through `walkAndCount` and `countFile`, handle stdin path, update cobra `Example:`.

**Changes in `root.go`:**

1. Add `tokens bool` field to `rootFlags`.
2. Register `--tokens` flag:
   ```
   cmd.Flags().BoolVar(&flags.tokens, "tokens", false,
       "count tokens using cl100k_base (GPT-3.5/4 approximation; slower than byte counting)")
   ```
3. Add `"tokens"` to `validSortKeys` map.
4. Pass `flags.tokens` into `runDirectoryOpts` (new field `countTokens bool`).
5. Pass `flags.tokens` into the stdin path: when `flags.tokens` is true, call
   `counting.CountWithTokens(c.InOrStdin(), tc)` instead of `counting.Count(c.InOrStdin())`.
   Requires obtaining the `tokens.Codec` singleton via `tokens.Get()`.
6. Update `addCounts` to sum `Tokens` field:
   ```
   return counting.Counts{
       Bytes:  a.Bytes + b.Bytes,
       Lines:  a.Lines + b.Lines,
       Words:  a.Words + b.Words,
       Chars:  a.Chars + b.Chars,
       Tokens: a.Tokens + b.Tokens,
   }
   ```
7. Update `walkAndCount` signature or internal plumbing: when `countTokens` is true, call
   `counting.CountWithTokens(rc, tc)` inside `countFile` (or pass the codec into `countFile`).
   Since `countFile` is a local function, updating its signature to accept an optional
   `tokens.Codec` (nil = no token counting) is the cleanest approach.
8. Update cobra `Example:` to add:
   ```
   # Count tokens (cl100k_base; Claude approximation)
   rak --tokens .
   ```
9. Update `PersistentPreRunE` sort-key validation error message to include "tokens" in the
   valid-key list.

**Design decisions (do not relitigate):**

- `tokens.Get()` is called once in `runRoot`/`runDirectory` when `flags.tokens` is true;
  the returned `Codec` is passed down rather than re-called per file.
- `countFile` accepts an optional `counting.TokenCounter` (nil = no token counting) so the
  `internal/counting` package interface drives the boundary — `cmd/rak` imports both
  `internal/tokens` (for `tokens.Get()`) and `internal/counting` (for `CountWithTokens`
  and the `TokenCounter` interface), and passes the concrete `tokens.Codec` as the interface.
- No goroutine safety concern in v0.2.0 (single-threaded walk).

**Acceptance:**

- `mage build` passes.
- `mage test-pkg cmd/rak` passes with: all existing tests; new tests for `--tokens` flag
  parsing; new integration test asserting that `rak --tokens <fixture>` produces non-zero
  token counts in the output; test that `rak --sort tokens .` does not error.
- `rak --help` shows the `--tokens` flag with the approximation caveat in its usage line.
- `rak --sort tokens .` sorts by token count descending.
- `rak --sort tokens --sort-asc .` sorts ascending.
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
  `--tokens`, note the `cl100k_base` encoder, document the **Claude-approximation caveat**
  verbatim: "Token counts use the `cl100k_base` (GPT-3.5/4) vocabulary. This is an
  approximation for Claude — counts will differ from Claude's actual billing token count
  but serve as a useful ballpark."
- Embed `docs/tokens.gif` near the narrative section.

**Acceptance:**

- `docs/tokens.tape` exists and is syntactically valid (runnable with `vhs docs/tapes/tokens.tape`).
- `docs/tokens.gif` exists and is a valid GIF file (non-zero size).
- `README.md` contains the "Token counting" section with the Claude-approximation caveat.
- `README.md` "Common invocations" includes `rak --tokens .`.
- `rak --help` (via cobra `Example:` set in B.5) shows `rak --tokens .` — this criterion
  is a carry-over verification, not a new change in this unit.

## Notes

**Cross-stream coordination**: Streams B, C, D all add new flags to `cmd/rak/root.go`. The planner should make the cmd/rak flag-wiring unit explicit and self-contained so the orchestrator can serialize it against C and D at build time. Internal-package work (`internal/tokens/`, `internal/counting`, `internal/summary`, `internal/render/*`) is parallel-safe with the other streams.

**Performance note**: tokenization is meaningfully slower than byte/line counting. Cache the encoder across files (`tokenizer.Get(encoding)` returns a shared instance); do not re-instantiate per file. Tests must cover empty file, ASCII, multibyte (UTF-8 emoji), and a known-token-count fixture to detect regression if tiktoken-go updates change behavior.
