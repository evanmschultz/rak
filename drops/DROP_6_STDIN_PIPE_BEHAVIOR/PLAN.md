# DROP_6 — STDIN_PIPE_BEHAVIOR

**State:** done
**Tier:** C
**Blocked by:** DROP_5
**Paths (expected):** — (no new code; verification only)
**Packages (expected):** — (no new packages)
**PLAN.md ref:** main/PLAN.md → `DROP_6_STDIN_PIPE_BEHAVIOR` row
**Workflow:** main/drops/WORKFLOW.md
**Started:** 2026-05-15
**Closed:** 2026-05-15 (no-op tier C — ratifies existing Drop 2 / decision 9 / decision 30 cuts; no new code shipped)

## Scope

Drop 6 is a **tier C no-op close**. Its purpose is to formally ratify that:

1. **Stdin wc-parity counting already shipped in Drop 2.** `counting.Count(c.InOrStdin())` is the no-args path in `cmd/rak.runRoot`; behavior matches `wc`'s read-until-EOF semantics. Verified by existing tests `TestRootCmd_ReadsStdin_RendersTOONDefault` (Drop 4 4.4 TOON-default rewrite of Drop 2's stdin-render test) and the format-flag variants.
2. **Pipe-vs-TTY auto-detection for OUTPUT was originally Drop 2's job** via laslig. Drop 4 superseded this: TOON is the default renderer regardless of TTY (per decision 33 + LLM-first framing). `--human` / `--json` / `--toon` are explicit boolean flags. No further work needed.
3. **TTY-hang on no-path-no-stdin and `--as <lang>` stream-type assertion are CUT** per decision 30 (rak is wc++; UX polish + code-aware stdin moves to v0.2). No code change required to "cut" — the planned features simply never landed.

## Planner

One unit, ratifying-only. No code work, no QA subagents (per WORKFLOW.md § "Cascade Tiering" Tier C mechanics).

### Unit 6.1 — Ratify Drop 2/4 stdin behavior; no new code

- **State:** done (no code work)
- **Paths:** — (none touched)
- **Packages:** — (none touched)
- **Acceptance:**
  - `mage ci` green from `main/` (re-verified during Drop 5 close at commit `4fde076`; no code touched in Drop 6, so the same green state holds).
  - Existing tests covering stdin behavior pass: `TestRootCmd_ReadsStdin_RendersTOONDefault`, `TestRootCmd_FlagJSON`, `TestRootCmd_FlagHuman` (Drop 4 4.4 surface).
  - No new Go file added; no flag surface change; no behavior change.
- **Blocked by:** —

## Notes

### Ratified cuts (decision 30 + decision 9 amendment)

- **Cut: TTY-hang detection on no-path-no-stdin.** Original Drop 5 plan (pre-decision-30) had unit 5.1 explicitly detect TTY-stdin and hang reading until EOF, matching `wc`'s interactive behavior. Decision 30 cut this UX feature: rak's no-args + TTY-stdin path simply reads `os.Stdin` via cobra's `c.InOrStdin()` and blocks until EOF (Ctrl-D from user). This is byte-identical to `wc`'s actual behavior; the cut is a cut of EXPLICIT detection code, not a behavior change.
- **Cut: `--as <lang>` stream-type assertion.** Original Drop 5 plan unit 5.3 (now part of Drop 5 lang decomposition under a different number) added `--as go` to enable code-aware counting on a stdin stream. Decision 30 cut this for v0.1.0; only `--lang` (walk filter for path-arg mode) lands. Stdin remains a wc-parity stream — bytes/lines/words/chars, no language detection, no split.

### Why no code change

Drop 2's stdin path + Drop 4's TOON-default renderer already produce the desired v0.1.0 stdin behavior:
- Piped stdin → reads to EOF → emits TOON (or `--human` / `--json`) with `counting.Counts`.
- TTY stdin + no args → blocks reading stdin (matches `wc`); user Ctrl-Ds to send EOF.
- No `--as` flag → no language-aware stdin counting.

The drop closes with no Go file touched and no commit-level code diff vs Drop 5 close.

### Open Unknowns

None. All Drop 6 design questions are pre-decided by decisions 9 + 30.
