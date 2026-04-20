# DROP_1 — Closeout

Written once at drop close. See `main/drops/WORKFLOW.md` § "Phase 7 — Closeout" for the full step list.

- **Closed:** 2026-04-19
- **Final build/QA commit:** `e92bf70` (`docs(drop-1): unit 1.6 round 2 qa green`) — the closeout commit itself advances state flips + this file.
- **CI run:** https://github.com/evanmschultz/rak/actions/runs/24646161643 — green, 1m14s on `e92bf70`.

## Hylla Feedback Aggregation

**Aggregate across all 8 `### Hylla Feedback` subsections in `BUILDER_WORKLOG.md` (Units 1.1 through 1.6, including 1.2 Round 2 and 1.6 Round 2): zero Hylla misses, zero forced fallbacks.**

Every subsection reports the same shape: the unit's file set was non-Go (go.mod / go.sum / magefile.go as tool config / `.golangci.yml` YAML / `.github/workflows/ci.yml` YAML / markdown) or a single-file local rewrite inside `cmd/rak/root.go` with no cross-package callers yet. Hylla is Go-only by design (main/CLAUDE.md § "Code Understanding Rules" rule 3), so none of Drop 1's work touched indexable surface area. The Unit 1.3 note explicitly calls out that Hylla would be the right tool from Drop 2.1 onward once `internal/counting` introduces the first cross-package caller of the `count` / `Counts` primitive — the miss is structural (no Go ingest yet on this drop's diff), not a Hylla shortfall. The Unit 1.6 Round 2 Context7 lookup for `/golangci/golangci-lint` install guidance was an external-semantics query (third evidence tier), not a Hylla fallback.

Net: Hylla was correctly excluded from every Drop 1 query path. Drop 2 is the first drop whose feedback aggregation will carry signal — it lands the first `internal/*` package with real Go surface area.

(Same entry appended to `main/HYLLA_FEEDBACK.md`.)

## Refinements

Six entries land in `main/REFINEMENTS.md` from Drop 1 — two ergonomic wins, one pre-filed follow-up win, one QA-loop lesson, one CI-install lesson, one advisory.

1. **Planner-locked Drop 3 contract prevents scope crack (ergonomic win).** Decision 25 (`fileset.File` struct exposes `Open()` + `Peek(n)`) was committed into `main/PLAN.md` before Drop 1 started, closing the scope crack QA falsification flagged during the planning phase of this project. Drop 4's shebang sniff and binary-detection work (Drop 3.3) both share `Peek(512)` rather than duplicating file-open logic.
2. **Drop 1.3 boundary-pin prevents Drop-1/Drop-2 double work (ergonomic win).** Locking `count(io.Reader)` as *unexported + in-file in `cmd/rak/root.go`* during Drop 1.3 — with Drop 2.1 explicitly owning the move into `internal/counting` — saved a cross-drop refactor. Drop 1 never pretended to have `internal/counting`, Drop 2 doesn't have to undo a premature export.
3. **Plan-QA discovered the Drop 9 tool-pinning follow-up (ergonomic win).** Drop 1 plan-QA falsification C4 surfaced that CI's `gofumpt` / `golangci-lint` install steps would drift vs local. The finding was filed into `main/PLAN.md` § "Follow-Ups" as a Drop 9 item rather than bloating Drop 1. Round 2 later validated this concern partially (v1/v2 module-path bug) and partially extended it (action-version bumps).
4. **Both Round 1 QA passes missed the v1/v2 install defect (QA loop lesson).** Phase 6 CI fail at commit `c230000` — `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest` resolves to v1 (v2 migrated to `/v2` module path); v1 rejects the v2-schema `.golangci.yml`. **Root cause of the miss:** both QA agents evaluated locally where the dev's binary is already v2.11.4 via the install-script path. The proof reviewer cleared acceptance bullets against the working tree; the falsification reviewer flagged F3 as a version-float concern rather than a module-path bug. **Lesson for future drops touching CI install steps:** QA falsification must explicitly construct the runner-side install resolution (what does `@latest` on a specific module path actually fetch today?) independent of the dev's local state. Adding a "CI-install defect class" to the Drop 9 plan-QA falsification brief captures this.
5. **Upstream install-script pinned to version beats `go install` for golangci-lint (CI-install lesson).** Context7 `/golangci/golangci-lint` states maintainers "strongly recommend against 'go install', 'go get', 'go tool' directives" for golangci-lint. Round 2 fix (`a87dda5`) switches to the upstream `install.sh` one-liner pinned to `v2.11.4`. `actions/setup-go@v5` guarantees `$(go env GOPATH)/bin` is writable and on PATH. This pattern generalizes: any CI install step that depends on a Go-published binary whose upstream explicitly disrecommends `go install` should use the upstream install path; version pinning applies to both.
6. **Advisory tracked but unresolved: `actions/checkout@v4` + `actions/setup-go@v5` are Node.js 20 runners (CI tooling).** GitHub's first green run on `e92bf70` emitted a deprecation annotation — Node.js 20 actions forced to Node.js 24 by 2026-06-02, removed from runners 2026-09-16. Already covered by the Drop 9 tool-pinning + action-version follow-up; no Drop 1 reopen.

(Same 6 entries appended to `main/REFINEMENTS.md`.)

## Ledger Entry

Drop 1 landed the Go scaffold + mage-first build gates + first CI workflow for rak in 6 atomic units. Final shape: `cmd/rak/{main.go,root.go}` (fang entry + cobra root with `rak [path]` + `WithNotifySignal`), `magefile.go` with 9 targets (`build`/`test`/`format`/`lint`/`ci`/`install`/`run`/`coverage`/`planCheck`), `.golangci.yml` v2 schema with one narrow exclusion, `.github/workflows/ci.yml` running `mage ci` with `gofumpt` + `golangci-lint` (v2.11.4 via upstream install script) on Go 1.26.x Ubuntu. Module `github.com/evanmschultz/rak`. Zero `internal/*` packages — `count(io.Reader) (Counts, error)` stays unexported in `cmd/rak/root.go` awaiting Drop 2.1 lift. First green CI run: `24646161643` @ `e92bf70`, 1m14s. One Round 2 defect (v1/v2 golangci-lint module path) discovered in Phase 6 and fixed before this closeout — lesson captured in REFINEMENTS entry 4.

(Same entry appended to `main/LEDGER.md`.)

## Wiki Changelog

`2026-04-19 — DROP_1_CODE_SCAFFOLD_MAGE_CI closed. Go scaffold + mage (9 targets) + CI workflow (golangci-lint v2.11.4 via upstream install script) green on e92bf70; no best-practice WIKI changes.`

(Same one-liner appended to `main/WIKI_CHANGELOG.md`.)

## Hylla Ingest

- **Triggered:** 2026-04-19 (after CI green on `e92bf70`)
- **Mode:** full_enrichment
- **Source:** `https://github.com/evanmschultz/rak.git` @ commit `e92bf708f83fd4f1755b35ea6e0b60bf303efa3a` (branch `main`)
- **Result:** task `task-9a296802d780f823` — status `completed`, progress 100/100, message `ingest complete`. Started 2026-04-20T03:07:23Z, finished 2026-04-20T03:08:43Z (~80s). Explicit commit hash required because rak has no semver tags yet (v0.1.0 tag lands in Drop 9.6).

## WIKI.md Updates

**None — no best-practice change.** Drop 1 implemented already-documented best practices (mage discipline, drop-lifecycle workflow, QA discipline). The one pattern that is a genuine new-best-practice — *use the upstream `install.sh` pinned to a version for CI golangci-lint install, never `go install`* — is preserved in three durable places already: `.github/workflows/ci.yml:38` (the actual install step), drop `PLAN.md:22` (with the Context7 rationale citation), and `main/REFINEMENTS.md` entry 5 (lesson form). Promoting it into WIKI would duplicate, not clarify. If a later drop adds a second CI-install defect of the same class, reconsider — a second instance earns a WIKI section.
