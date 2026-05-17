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

<Filled by go-planning-agent in Phase 1.>

## Notes

**Cross-stream coordination**: Streams B, C, D all add new flags to `cmd/rak/root.go`. The planner should make the cmd/rak flag-wiring unit explicit and self-contained so the orchestrator can serialize it against C and D at build time. Internal-package work (`internal/tokens/`, `internal/counting`, `internal/summary`, `internal/render/*`) is parallel-safe with the other streams.

**Performance note**: tokenization is meaningfully slower than byte/line counting. Cache the encoder across files (`tokenizer.Get(encoding)` returns a shared instance); do not re-instantiate per file. Tests must cover empty file, ASCII, multibyte (UTF-8 emoji), and a known-token-count fixture to detect regression if tiktoken-go updates change behavior.
