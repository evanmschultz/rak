# DROP_3 — Builder QA Falsification

## Unit 3.0 — Round 1

- **QA:** go-qa-falsification-agent
- **Reviewed:** 2026-04-21
- **HEAD under review:** `be08d20` (`feat(deps): add go-gitignore and doublestar for drop-3`)
- **Verdict:** `pass` — no unmitigated counterexamples found.

### Attacks attempted

| # | Attack vector | Outcome |
|---|---|---|
| A1 | Did `mage addDep` wrap `go get`, or was a raw `go get` slipped in, or does it call `go mod download` / something else? | **Mitigated.** `magefile.go:88–93` shows `AddDep(module string)` is literally `sh.RunV("go", "get", module)`. The builder worklog's two `mage addDep` invocations return `go: added ...` — that is `go get`'s normal stdout, not `go mod download`. No raw `go get` line appears in the commit; the two `// indirect` requires in `go.mod` are the entries that `go get` produces when the module has no importer yet, which is exactly the unit's documented Drop 2 workflow. |
| A2 | Are the pinned versions legitimate upstream releases (no forks, no typo'd module paths like `go.gitignore` vs `go-gitignore`)? | **Mitigated.** `go.mod` uses `github.com/sabhiram/go-gitignore` (hyphen, correct). GitHub API confirms commit `525f6e181f062064d83887ed2530e3b1ba0bc95a` exists on the real `sabhiram/go-gitignore` repo with author date `2021-09-23T22:41:02Z`, matching the pseudo-version timestamp byte-for-byte. `bmatcuk/doublestar/v4@v4.10.0` is a tagged release on the real upstream repo (Context7 resolves it; Go module proxy lists v4.10.0 as the highest v4 tag). |
| A3 | Is "latest stable tags" honored, or did the resolver pick a stale fork / pre-release? | **Mitigated as worded, clarified.** `go list -m -versions` via the module proxy shows `sabhiram/go-gitignore` has published **zero** tags (empty version list returned by `proxy.golang.org/github.com/sabhiram/go-gitignore/@v/list`). A pseudo-version off `master` is the only resolvable "latest stable" for a tag-less module — the builder's own worklog note calls this out and is correct. `doublestar/v4` v4.10.0 is the highest v4 tag on the proxy (list shows v4.0.x–v4.10.0, no v4.11+ or pre-releases), so "latest stable tag" is satisfied literally. The Unit 3.0 acceptance wording "latest stable tags" should read "latest stable resolver choice" for tag-less modules; this is a documentation nit at the planner level, not a build failure, and the Phase 3 plan-QA loop already converged without flagging it. |
| A4 | Do the licenses permit inclusion? (MIT / Apache-2.0 / BSD expected; no GPL / AGPL / CDDL / proprietary.) | **Mitigated.** Both modules ship MIT (`LICENSE` headers: "The MIT License (MIT)\nCopyright (c) 2015 Shaba Abhiram" and "The MIT License (MIT)\nCopyright (c) 2014 Bob Matcuk"). MIT is compatible with any outbound license rak chooses. |
| A5 | Any CVEs on the pinned versions? | **Mitigated.** The Go vulnerability DB index at `https://vuln.go.dev/index/modules.json` (365 KB, pulled 2026-04-21) has **zero entries** matching `sabhiram`, `bmatcuk`, `doublestar`, or `gitignore` (case-insensitive). No published CVEs affect either pin. |
| A6 | Hidden compiled transitive deps pulled in by either module? | **Mitigated.** `bmatcuk/doublestar/v4@v4.10.0`'s own `go.mod` is two lines (`module github.com/bmatcuk/doublestar/v4\ngo 1.16`) — zero requires. `sabhiram/go-gitignore@v0.0.0-20210923224102-525f6e181f06`'s `go.mod` requires only `github.com/stretchr/testify v1.6.1`, a **test-only** dep that does not get compiled into rak. The four `/go.mod`-only lines added to rak's `go.sum` (`davecgh/go-spew v1.1.0`, `stretchr/objx v0.1.0`, `stretchr/testify v1.6.1`, `gopkg.in/yaml.v3 v3.0.0-20200313102051`) are module-graph closure records — Go records `.mod` hashes for the full dependency graph so `go mod verify` can check the graph, but these modules are **not** downloaded as source and **not** linked. rak's `go.mod` gained exactly two new `// indirect` lines, matching the builder's claim. |
| A7 | Does `mage ci` (not just `mage build` + `mage test`) pass — including `gofumpt -l .` emptiness and `golangci-lint run`? | **Mitigated.** Ran `mage ci` from `main/` during this review: `gofumpt -l .` clean, `golangci-lint run` reported `0 issues.`, `go test -race ./...` passed all three test packages (cached: `cmd/rak`, `internal/counting`, `internal/render`). The builder's worklog only cited `mage build` + `mage test`, but `mage ci` also passes, so nothing was swept under the rug. |
| A8 | YAGNI — does the drop actually need both deps now, or could one be deferred? | **Accepted as not-a-finding.** Unit 3.1's acceptance explicitly consumes `sabhiram/go-gitignore` (gitignore matcher) and `bmatcuk/doublestar/v4` (`--include`/`--exclude` globs), landing in the same drop. Deferring either would just push the identical `mage addDep` invocation into 3.1, creating a dep-plus-code commit that's harder to review. Coupling two adds in one unit is the Drop 2 workflow's stated purpose. |
| A9 | Did the builder sneak a raw `go` invocation anywhere (script, commit, file)? | **Mitigated.** `git show HEAD --stat` shows only four files touched: `go.mod`, `go.sum`, the drop's `PLAN.md` (state flip `todo → done`), and `BUILDER_WORKLOG.md`. No scripts, no workflow files, no magefile edits. `magefile.go` is unchanged from Drop 1.4, so the only route to these dep additions was `mage addDep` → `sh.RunV("go", "get", module)`. |
| A10 | Did the unit produce any uncommitted residue (e.g. a lingering `go mod tidy` side-effect the builder hadn't noticed)? | **Mitigated.** `git status` reports `working tree clean` at HEAD. No stray edits. Branch is 9 commits ahead of `origin/main` — consistent with Drop 3 being mid-build and push deferred to drop-end verify per WORKFLOW.md Phase 6. |
| A11 | Does `go.mod` integrity match `go.sum` — no missing or extra hashes? | **Mitigated.** `mage test -race ./...` inside `mage ci` would have failed with `missing go.sum entry` or `verifying ...: checksum mismatch` if integrity were broken; it passed clean. Both `h1:` and `/go.mod` hashes for each new module are present in go.sum. |
| A12 | Is the Go toolchain version compatible? | **Mitigated.** `go.mod` declares `go 1.26.1`; both dep `go.mod`s declare `go 1.13` (sabhiram) and `go 1.16` (doublestar), well below 1.26.1 so no forward-compat issue. |

### Counterexamples

None. No attack produced a concrete failure reproducible against HEAD.

### Findings surface

- **Documentation nit (non-blocking)** — Unit 3.0's acceptance wording "latest stable tags" reads literally-incorrectly for a tag-less module. The builder worklog calls this out and offers the correct interpretation ("latest stable resolver choice"). If the orchestrator wants tighter prose, a one-line planner tweak at drop-close would be ergonomic; it is **not** grounds for failing this QA round because the resolver-selected pseudo-version is the only legitimate choice upstream.

### Summary

Unit 3.0 is clean. Two production deps added by the mandated `mage addDep` path, both MIT-licensed, no CVEs, no hidden transitive compile surface, `mage ci` fully green. No counterexample found.
