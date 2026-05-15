# DROP_4 — Builder QA Proof

Append a `## Unit N.M — Round K` section per build-QA proof pass. See `main/drops/WORKFLOW.md` § "Phase 5 — Build-QA (per unit)" for the section contract.

## Unit 4.0 — Round 1

**Verdict:** pass-with-findings

### Acceptance audit

- "mage addDep used (not raw go get)" — **pass**. `BUILDER_WORKLOG.md` § "Unit 4.0 — Round 1" line 10 records `mage addDep github.com/toon-format/toon-go` as the invocation; `mage -l` confirms `addDep` is the canonical target wrapping `go get`. No raw `go get` trace.
- "go.mod has require entry" — **pass**. `go.mod` line 42: `github.com/toon-format/toon-go v0.0.0-20251202084852-7ca0e27c4e8c // indirect` inside the second `require` block. `// indirect` is the expected placement at this stage because no Go source file imports it yet (import lands in Unit 4.5).
- "go.sum populated" — **pass**. `git diff HEAD~1 -- go.sum` shows exactly two added lines for the new module: `h1:D8lDFovBMZywze1eh9iwMLcYor5f11mHBocLhO7cBe8=` (module zip) and `/go.mod h1:j/BOnpF2ihnz4lELs99h9mwGJBx/zdleOUCnLLRPCsc=` (go.mod). Both standard required entries.
- "no unexpected transitive deps" — **pass**. `go mod graph` shows toon-go's only outgoing edge is `go@1.23` (Go language version, not a module). Zero transitive module deps. Confirmed by `git diff HEAD~1 -- go.mod`: exactly one `+` line, no other indirect deltas. Matches builder's claim.
- "no Go source files changed" — **pass**. `git show HEAD --stat` lists only `BUILDER_WORKLOG.md`, `PLAN.md`, `go.mod`, `go.sum`. No `.go` files.
- "mage build passes" — **pass**. Re-ran `mage build` from `main/`: exit 0, no output (clean build).
- "mage test passes" — **pass**. Re-ran `mage test` from `main/`: all five packages report `ok ... (cached)` (`cmd/rak`, `internal/counting`, `internal/fileset`, `internal/ignore`, `internal/render`). Cache is sound because no Go source changed; `-race` is enabled per mage target definition.

### Findings

- **F1 (low) — "latest tagged version" wording does not apply to an untagged module.** PLAN.md line 32 reads: *"`main/go.mod` gains a `require` entry for `github.com/toon-format/toon-go` at its latest tagged version."* Verified upstream via `gh api repos/toon-format/toon-go/tags` → `[]` (zero tags exist on the repo). The pseudo-version `v0.0.0-20251202084852-7ca0e27c4e8c` is the correct resolution for a tagless module per Go modules semantics — `go get` cannot resolve to a tag that does not exist. Builder's flag in `BUILDER_WORKLOG.md` line 11 is accurate. Action for orchestrator: optionally tighten the PLAN.md wording (e.g. "at its latest tagged release, or pseudo-version if no tag exists") in any future dep-add unit; no remediation needed in Unit 4.0 itself. This is informational, not a build defect.

- **F2 (low) — Builder worklog notes pseudo-version risk for Unit 4.5.** `BUILDER_WORKLOG.md` line 13 flags: *"Pseudo-version flagged for orchestrator awareness — not a blocker per acceptance criteria, but worth noting for 4.5 if the library API surface is unstable."* This is good hygiene — surfacing for orchestrator visibility when Unit 4.5 starts. No action required at Unit 4.0 closure.

### Evidence summary

- `git show HEAD --stat`: 4 files changed (`BUILDER_WORKLOG.md`, `PLAN.md`, `go.mod`, `go.sum`); no `.go` files.
- `git diff HEAD~1 -- go.mod go.sum`: +1 line in go.mod (`// indirect` block), +2 lines in go.sum (h1: + /go.mod h1:).
- `go mod graph | line 224`: `github.com/toon-format/toon-go@v0.0.0-... go@1.23` — only `go@1.23` edge, no module deps.
- `gh api repos/toon-format/toon-go/tags` → `[]` — confirms no tagged release exists upstream.
- `mage build` → exit 0, clean.
- `mage test` → all 5 packages `ok (cached)`.
- `mage -l` → `addDep` target is canonical.
