# DROP_3 — Builder QA Proof

## Unit 3.0 — Round 1

- **QA proof:** go-qa-proof-agent
- **Reviewed:** 2026-04-21
- **Verdict:** pass
- **Commit under review:** `be08d20 feat(deps): add go-gitignore and doublestar for drop-3`

### Acceptance-criterion verification

**AC1 — Deps added via `mage addDep`, not raw `go get`:**
- `BUILDER_WORKLOG.md` lines 11–12 document the two invocations verbatim:
  - `mage addDep github.com/sabhiram/go-gitignore` → `go: added github.com/sabhiram/go-gitignore v0.0.0-20210923224102-525f6e181f06`
  - `mage addDep github.com/bmatcuk/doublestar/v4` → `go: added github.com/bmatcuk/doublestar/v4 v4.10.0`
- Commit `be08d20` touches exactly `go.mod` + `go.sum` + the two drop mds; no scratch command log inconsistency; the worklog is the only record of the invocation, and its command output strings match the mage target signature (`go: added <module> <version>`). No sign of bypass.
- `mage -l` confirms the `addDep` target is resolvable (Drop 2.0 landed it as required).
- **Pass.**

**AC2 — `go.mod` has `require (...)` entries for both modules at latest stable tags:**
- `main/go.mod` line 17: `github.com/bmatcuk/doublestar/v4 v4.10.0 // indirect` — tagged release.
- `main/go.mod` line 40: `github.com/sabhiram/go-gitignore v0.0.0-20210923224102-525f6e181f06 // indirect` — Go pseudo-version because sabhiram has never cut a git tag (worklog line 17 documents this; the pseudo-version IS the latest stable resolver choice). The PLAN.md "latest stable tags" phrasing is satisfied by the resolver's latest-stable selection where no tag exists; worklog flagged this nuance to orch as Phase-3 discussion material, not a blocker.
- Both land in the secondary `require (...)` block (lines 12–49) because no rak source imports them yet; they will promote to the primary block in 3.1/3.2.
- **Pass.**

**AC3 — `go.sum` populated for both modules; no surprise compiled transitive deps:**
- `main/go.sum` lines 15–16: doublestar `h1:` + `/go.mod` pair.
- `main/go.sum` lines 78–79: sabhiram `h1:` + `/go.mod` pair.
- Commit diff shows four additional `/go.mod`-only entries: `davecgh/go-spew v1.1.0`, `stretchr/objx v0.1.0`, `stretchr/testify v1.6.1`, `gopkg.in/yaml.v3 v3.0.0-20200313102051`. These are `/go.mod`-only lines (no matching `h1:` hash), which is Go's way of recording **module-graph closure** rather than compiled dependencies — they are hash-verified only for the `go.mod` files themselves, never downloaded as source nor linked into any binary. This is consistent with sabhiram's own test-suite pulling in testify (an `_test.go`-only import), which Go's MVS algorithm records for reproducibility.
- No new `h1:` entries appear for any module other than the two target modules. Neither target contributes a compiled transitive dep.
- Worklog line 18 documents this clearly and correctly.
- **Pass with observation** — see § "Observations" for the surfaced-to-orch note.

**AC4 — `mage build` + `mage test` pass clean:**
- Re-ran both targets locally at review time (not trusting builder's claim alone):
  - `mage build` → exit 0, no stdout/stderr.
  - `mage test` → `ok  github.com/evanmschultz/rak/cmd/rak (cached)` / `ok  github.com/evanmschultz/rak/internal/counting (cached)` / `ok  github.com/evanmschultz/rak/internal/render (cached)` — all three existing test packages green. Cached is expected: no Go source changed, so the test binary is unchanged; `mage test` always runs with `-race` per `magefile.go` / CLAUDE.md.
- No compile errors despite the unused `// indirect` entries — Go permits indirect deps without importers, exactly the workflow Drop 2's `mage addDep` was designed for.
- **Pass.**

### Observations (non-blocking, surfaced to orchestrator)

- **O1 — `/go.mod`-only transitive-dep entries:** The AC's "zero transitive deps" expectation (PLAN.md line 31) is stricter than the actual module resolver outcome; sabhiram's own test suite depends on testify/go-spew/objx/yaml.v3, recorded as `/go.mod`-only closure entries. No compiled dependency is pulled in, so the intent of the AC ("no surprise runtime deps") is satisfied. Builder already flagged this in the worklog. If the dev wants a strict "no new lines under any circumstance" bar, the AC text needs tightening for future drops — but for Unit 3.0 as written, this is AC-compliant (builder followed the "flag and return to orch" path).
- **O2 — Pseudo-version vs tag for sabhiram:** AC line 30 says "latest stable tags"; sabhiram has no tags, so Go picked a pseudo-version. Worklog line 17 calls this out explicitly as Phase-3 discussion material. Non-blocking for Unit 3.0 since the resolver's choice is deterministic and hash-pinned.

### Evidence trail

- `git log --oneline -10` — last commit is `be08d20 feat(deps): add go-gitignore and doublestar for drop-3`.
- `git show HEAD --stat` — exactly four files changed: `go.mod` (+2), `go.sum` (+8), plus worklog and plan state flip.
- `git show HEAD -- go.mod go.sum` — diff exactly matches the worklog's claimed deltas.
- `main/go.mod` lines 17 + 40 — both target modules pinned.
- `main/go.sum` lines 15–16 + 78–79 — `h1:` hashes for both targets; lines 42, 84–85, 106 — `/go.mod`-only closure entries flagged in O1.
- Re-ran `mage build` + `mage test` from `main/` at review time; both green.
- `mage -l` shows `addDep` target present; Drop 2.0's landing is corroborated.

### Hylla Feedback

None — no Hylla queries were needed for this review. Unit 3.0 touched only `go.mod` / `go.sum` (non-Go dependency metadata) and drop mds, which are out of Hylla's Go-source scope by design. Ground truth was entirely in git + filesystem + `mage` output, per CLAUDE.md § "Code Understanding Rules" steps 2 and 3.
