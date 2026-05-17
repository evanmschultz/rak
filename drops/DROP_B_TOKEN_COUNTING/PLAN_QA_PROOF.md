# DROP_B — PLAN_QA_PROOF

## Round 1

**Verdict:** PASS WITH FINDINGS

The decomposition is sound, the dependency graph is correct, all core technical claims verify against the current source and the tiktoken-go Context7 docs. Findings below are mostly polish — one concern (F2) is worth resolving before build because it touches a load-bearing struct conversion that no test currently guards.

## Proof targets verified

| Claim | Evidence | Verdict |
|---|---|---|
| `internal/counting` has zero `internal/` imports | `counting.go` imports only `bufio`, `errors`, `io`, `unicode` | PASS |
| `Counts` field order is load-bearing, no struct tags today | `counting.go` lines 19–31 (struct decl) + lines 14–18 (doc comment confirms) | PASS |
| `int64` zero + `json:",omitempty"` suppresses field | stdlib `encoding/json` behavior — `omitempty` omits zero values of numeric types | PASS |
| `summary.Directory` embeds `counting.Counts` reachable as `d.Counts.Tokens` | `summary.go` lines 28–29 (`Counts counting.Counts`) | PASS |
| `TestSortDirs_UnknownKey_Panics` currently uses `SortKey("tokens")` | `summary_test.go` line 174 | PASS |
| `addCounts` sums only four fields today | `root.go` lines 501–508 | PASS |
| Stdin path calls `counting.Count(c.InOrStdin())` unconditionally | `root.go` line 248 | PASS |
| `validSortKeys` lives in `cmd/rak/root.go`, not in `internal/summary` | `root.go` lines 46–53 | PASS |
| `PersistentPreRunE` sort-key error message lists keys verbatim | `root.go` line 97 | PASS |
| `github.com/tiktoken-go/tokenizer` not in `go.mod` | `go.mod` lines 5–13 — no tiktoken-go entry | PASS |
| `tokenizer.Get(tokenizer.Cl100kBase)` → `(Codec, error)` | Context7 `/tiktoken-go/tokenizer` "Get Tokenizer by Encoding Format" snippet | PASS |
| `Codec.Count(string) (int, error)` exists, cheaper than Encode | Context7 "Count Tokens in Text using tiktoken-go" snippet — `enc.Count(text)` returns `(int, error)` | PASS |
| No `io.Reader` API on the library | Context7 snippets all use `string` input | PASS |
| `Cl100kBase` constant for GPT-3.5/4 encoding | Context7 explicitly names cl100k_base as GPT-4/GPT-3.5-turbo encoding | PASS |
| Tapes layout: `docs/tapes/<name>.tape` → `docs/<name>.gif` | Existing repo layout: `docs/tapes/default-toon.tape` → `docs/default-toon.gif` (7 tape/gif pairs) | PASS |
| B.3 + B.4 file footprints are disjoint (parallel-safe) | B.3: `internal/summary/sort.go` + `summary_test.go`; B.4: `internal/render/{toon,human,json}.go` + `render_test.go` — no overlap | PASS |
| Unit dependency graph B.1 → B.2 → (B.3 ‖ B.4) → B.5 → B.6 | B.2's `Blocked by` lists B.1; B.3 and B.4 each list B.2 only; B.5 lists B.3, B.4; B.6 lists B.5. No cycle. | PASS |
| Feature trio (VHS + README + cobra Example) in one unit | B.6 lists all three; cobra Example deliberately scoped to B.5 (same file as flag wiring) with cross-ref in B.6 acceptance | PASS |

## Findings

### Finding 1 — `tokens.Codec` interface declaration vs `Get()` return type is incoherent

- **Severity:** concern
- **Where:** Unit B.1, "Design decisions" + "Key API facts"
- **Issue:** B.1 scope says "Create `internal/tokens/tokens.go` exporting a `Codec` interface and a package-level `Get()` function." Design says "`Codec` interface lives in `internal/tokens`" but then specifies `Get() (tokenizer.Codec, error)` — i.e., returns the upstream tiktoken-go type, not rak's own `Codec`. Two readings:
  - (a) rak's `Codec` is purely a type alias/re-export (e.g. `type Codec = tokenizer.Codec`) and `Get` returns the alias.
  - (b) rak's `Codec` is a new interface and `Get` returns it (typed as `Codec`, not `tokenizer.Codec`).
  The plan as written specifies neither cleanly — `Get() (tokenizer.Codec, error)` suggests (a) but B.1 also says "exporting a `Codec` interface" which suggests (b) does work to do.
  Note that B.2 separately defines `counting.TokenCounter` interface for the dep-isolation contract; whether `internal/tokens` ALSO exports its own `Codec` interface is duplicative if (a) holds.
- **Recommendation:** Resolve at plan revision: pick (a) or (b) explicitly. Suggested resolution = (a) re-export only, since `counting.TokenCounter` already covers the consumer-side interface. Then B.1 becomes "exports `Get() (tokenizer.Codec, error)` — no rak-local interface needed; consumers depend on `counting.TokenCounter`." This shrinks B.1's surface area and keeps the dep-isolation argument clean.

### Finding 2 — Add a defensive "DO NOT add Tokens to summary.Directory" guardrail

- **Severity:** concern
- **Where:** Unit B.4 design decisions, and B.3 design decisions
- **Issue:** Both B.3 ("no new field is needed on `summary.Directory`") and B.4 ("No change needed to `counting.Counts`") implicitly rely on the bare struct conversion `directoryJSON(filterUnknown(d))` at `json.go` line 145, which compiles ONLY when `summary.Directory` and `directoryJSON` declare the same fields in the same order with the same types. `summary.go` line 18 and `json.go` line 54 both call this out, but neither plan unit pre-empts the trap. A well-intentioned reader of B.3 could decide "also add `Tokens int64` to `summary.Directory` for parallel structure" — that would silently break the JSON renderer because `directoryJSON` does NOT declare a top-level `Tokens` field (and shouldn't — the field lives in the nested `Counts`).
- **Recommendation:** Add an explicit "DO NOT add a top-level `Tokens` field to `summary.Directory`. The token count is reachable via `d.Counts.Tokens` through the embedded `counting.Counts`. Adding a top-level field would break the bare struct conversion `directoryJSON(filterUnknown(d))` at `internal/render/json.go:145`." sentence to either B.3 or B.4 design decisions.

### Finding 3 — `tokens.Get()` error path through stdin run is unspecified

- **Severity:** concern
- **Where:** Unit B.5, step 5 ("stdin path")
- **Issue:** Plan says: when `flags.tokens` is true, call `counting.CountWithTokens(c.InOrStdin(), tc)` instead of `counting.Count(c.InOrStdin())`, "Requires obtaining the `tokens.Codec` singleton via `tokens.Get()`." Two unspecified behaviors:
  - Where is `tokens.Get()` called for the stdin branch — inside `runRoot` before the `if len(args) == 1` check, or inside the `len(args) == 0` else branch?
  - If `tokens.Get()` returns a non-nil error (e.g. BPE table load failure), what does `runRoot` do? Wrap and return? Bare return?
  Similar ambiguity applies to the directory walk path — does `runRoot` resolve the codec once and pass it down via `runDirectoryOpts`, or does `runDirectory` itself call `tokens.Get()`?
- **Recommendation:** Add to B.5 design decisions: "`tokens.Get()` is called once in `runRoot` immediately after `resolveRenderer(flags)`, gated on `flags.tokens`. The result is wrapped into the new `runDirectoryOpts.tokenCounter counting.TokenCounter` field (nil when `--tokens` is unset). For the stdin branch, the same `tokenCounter` local is used directly. Any error from `tokens.Get()` is wrapped as `fmt.Errorf(\"init token counter: %w\", err)` and returned from `runRoot`."

### Finding 4 — TOON `omitempty` uncertainty deferred to builder is acceptable but could be planner-resolved

- **Severity:** nit
- **Where:** Unit B.4, "TOON renderer" + RiskNotes
- **Issue:** Plan defers the toon-go `omitempty` behavior on int64 to builder verification at build time. This is acceptable (it's a small spike), but a stronger plan would have resolved it at plan-QA time so the build unit ships with no open question. Existing TOON snapshot test impact (will snapshots break or not?) is conditional on this answer.
- **Recommendation:** Either (a) the planner runs a 30-second `go doc github.com/toon-format/toon-go` or Context7 check now and writes the resolved behavior into B.4 design, or (b) flag this explicitly as "Builder MUST run the spike in round 1 BEFORE editing snapshots; update RiskNotes with finding." Current text already does (b) loosely; tightening to "in round 1, before any snapshot edits" would help.

### Finding 5 — `counting.CountWithTokens` "implementation choice" left to builder may risk Count behavior drift

- **Severity:** nit
- **Where:** Unit B.2, design decisions
- **Issue:** Plan offers builder a choice: "Implementation may either call `Count` on the same bytes (reusing a `bytes.Reader`) or inline both paths in one pass." Acceptance does pin "`Count` behavior is unchanged — all existing tests still pass," which is the right guardrail, but the table-driven `TestCount` cases at `counting_test.go:11–51` test only six small fixtures. A subtle change in `Count` (e.g. switching from `bufio.NewReader` to reading all bytes first) could pass these tests yet differ on a large file or one that triggers `bufio.Reader.ReadRune` partial-rune edge cases.
- **Recommendation:** Constrain the choice: "B.2 MUST NOT modify the existing `Count` function body. `CountWithTokens` is a NEW function that may either (a) call `Count` internally then run tokenizer over the buffered bytes (requires reading into a buffer first), or (b) duplicate the byte/rune loop while running tokenizer separately. The existing `Count` is preserved verbatim." This eliminates any chance of subtle regression in the existing happy path.

### Finding 6 — `TestGet_KnownFixture` token-count regression-anchor needs source attribution

- **Severity:** nit
- **Where:** Unit B.1, acceptance + RiskNotes
- **Issue:** Plan says `"hello world"` → 2 tokens in cl100k_base as the fixed test value. Context7 docs example shows `"The quick brown fox jumps over the lazy dog."` → 10 tokens. The "2 tokens" claim for `"hello world"` is plausible but not directly cited from Context7 or any pinned source. If the actual count differs (e.g., it's 3 because of a leading-space convention), B.1's first test fails and the unit goes round 2 for a trivial reason.
- **Recommendation:** Either (a) pin a Context7-verified fixture: `"The quick brown fox jumps over the lazy dog."` → 10 tokens (directly from the Context7 snippet), or (b) leave the specific count TBD and let the builder pin it from a one-shot `go run` spike in B.1 round 1, with the test comment explicitly documenting the source ("captured 2026-05-16 against tiktoken-go vX.Y.Z").

### Finding 7 — Cobra Example update split across B.5 and B.6 — acceptance phrasing slightly ambiguous

- **Severity:** nit
- **Where:** Unit B.6 acceptance + Unit B.5 step 8
- **Issue:** B.5 step 8 owns the cobra `Example:` string edit. B.6 acceptance lists "`rak --help` (via cobra `Example:` set in B.5) shows `rak --tokens .` — this criterion is a carry-over verification, not a new change in this unit." This is correct, but the phrasing risks a build-QA reviewer marking B.6 as "incomplete" if `--help` doesn't show it (because B.5 forgot to add it). The dependency `B.6 blocked_by B.5` already prevents B.6 from starting before B.5 finishes, so the issue is detected — just the language is slightly off.
- **Recommendation:** Reword B.6 acceptance to "Cobra `Example:` (added in B.5) is verified present via `mage run -- --help | grep -- '--tokens'` — failure indicates B.5 was incomplete." Makes the verification mechanical.

### Finding 8 — TOON column ordering claim in scope vs actual struct order

- **Severity:** nit
- **Where:** Unit B.4, "TOON renderer" — `toonDirectory` column order
- **Issue:** Plan says "Add `Tokens int64 \`toon:\"tokens\"\`` field to `toonCounts` (after `Chars`) and to `toonDirectory` (after `Chars`)." Scope at the top of PLAN.md says "New column in TOON `directories` tabular block: `tokens` after `chars`." Both consistent. Verified `toonDirectory` current order at `toon.go:44–51`: `path|files|bytes|lines|words|chars`. New column at end yields `path|files|bytes|lines|words|chars|tokens` — matches scope. Also `toonCounts` order at `toon.go:32–37`: `bytes|lines|words|chars`. New field at end yields `bytes|lines|words|chars|tokens`. Both correct.
- **Recommendation:** No change needed. Listed here as a positive verification, not a finding to fix.

## Closing notes

- Dependency graph is acyclic and parallel-safe at B.3 ‖ B.4.
- All Context7-checkable tiktoken-go API claims verify against `/tiktoken-go/tokenizer` (`Cl100kBase` constant, `Codec.Count(string) (int, error)`, no `io.Reader` API).
- The `internal/counting` dep-isolation pattern via `TokenCounter` interface is sound — `counting` stays zero-internal-dep, and `tokenizer.Codec` satisfies `TokenCounter` by duck typing (`Count(string) (int, error)` signature matches exactly).
- The JSON `omitempty` strategy on `Tokens int64` is correct: zero value omitted, non-zero present. Existing JSON snapshot tests will remain green for non-`--tokens` runs.
- The TOON `omitempty` open question is the only externally-blocked risk; resolving it at plan time would be the cleanest path forward.

## Hylla Feedback

N/A — action item touched non-Go files in part (PLAN.md, README.md, .tape, .gif planning) but the Go-source verifications worked from direct file reads + Context7 (the standard rak evidence ladder when scoped per-file). Hylla was not queried because the working set was small enough (one file per touched package) that direct `Read` was the lower-overhead path. No Hylla miss to report.
