# Rak — Ledger

One-paragraph summary per closed drop. Appended at Phase 7 closeout. Reading top-to-bottom gives the project narrative.

---

## DROP_1_CODE_SCAFFOLD_MAGE_CI — closed 2026-04-19

Drop 1 landed the Go scaffold + mage-first build gates + first CI workflow for rak in 6 atomic units. Final shape: `cmd/rak/{main.go,root.go}` (fang entry + cobra root with `rak [path]` + `WithNotifySignal`), `magefile.go` with 9 targets (`build`/`test`/`format`/`lint`/`ci`/`install`/`run`/`coverage`/`planCheck`), `.golangci.yml` v2 schema with one narrow exclusion, `.github/workflows/ci.yml` running `mage ci` with `gofumpt` + `golangci-lint` (v2.11.4 via upstream install script) on Go 1.26.x Ubuntu. Module `github.com/evanmschultz/rak`. Zero `internal/*` packages — `count(io.Reader) (Counts, error)` stays unexported in `cmd/rak/root.go` awaiting Drop 2.1 lift. First green CI run: `24646161643` @ `e92bf70`, 1m14s. One Round 2 defect (v1/v2 golangci-lint module path) discovered in Phase 6 and fixed before this closeout — lesson captured in REFINEMENTS entry 4.
