# Rak — Hylla Feedback Ledger

Drop-by-drop aggregation of every `### Hylla Feedback` subsection from each drop's `BUILDER_WORKLOG.md`. Appended at Phase 7 closeout. Reading top-to-bottom gives the full history of where Hylla helped, where it missed, and where it was not the right tool.

---

## DROP_1_CODE_SCAFFOLD_MAGE_CI — closed 2026-04-19

Aggregate across all 8 `### Hylla Feedback` subsections in the drop's `BUILDER_WORKLOG.md` (Units 1.1 through 1.6, including 1.2 Round 2 and 1.6 Round 2): **zero Hylla misses, zero forced fallbacks.**

Every subsection reports the same shape: the unit's file set was non-Go (go.mod / go.sum / magefile.go as tool config / `.golangci.yml` YAML / `.github/workflows/ci.yml` YAML / markdown) or a single-file local rewrite inside `cmd/rak/root.go` with no cross-package callers yet. Hylla is Go-only by design (main/CLAUDE.md § "Code Understanding Rules" rule 3), so none of Drop 1's work touched indexable surface area. The Unit 1.3 note explicitly calls out that Hylla would be the right tool from Drop 2.1 onward once `internal/counting` introduces the first cross-package caller of the `count` / `Counts` primitive — the miss is structural (no Go ingest yet on this drop's diff), not a Hylla shortfall. The Unit 1.6 Round 2 Context7 lookup for `/golangci/golangci-lint` install guidance was an external-semantics query (third evidence tier), not a Hylla fallback.

Net: Hylla was correctly excluded from every Drop 1 query path. Drop 2 is the first drop whose feedback aggregation will carry signal — it lands the first `internal/*` package with real Go surface area.
