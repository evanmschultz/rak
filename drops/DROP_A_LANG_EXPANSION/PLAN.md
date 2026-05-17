# DROP_A — LANG_EXPANSION

**State:** planning
**Tier:** A
**Blocked by:** —
**Paths (expected):** internal/lang/lang.go, internal/lang/split.go, internal/lang/*_test.go, README.md
**Packages (expected):** internal/lang
**PLAN.md ref:** — (top-level PLAN.md removed at v0.1.0 ship; see memory `session_handoff_2026_05_16_v020_planning.md` for v0.2.0 scope)
**Workflow:** main/drops/WORKFLOW.md
**Started:** 2026-05-16
**Closed:** —

## Scope

Add ~30 new languages plus a long-overdue XML-from-HTML split. Coverage targets:

- **Programming**: C#, Scala, Lua, SQL, Dart, Elixir, Zig, R, F#, Haskell.
- **Templating + frontend variants**: templ (`.templ`), JSX (`.jsx`), TSX (`.tsx`), Sass/SCSS (`.scss`, `.sass`), LESS (`.less`), Vue (`.vue`), Svelte (`.svelte`), ERB (`.erb`), Jinja (`.j2`, `.jinja`, `.jinja2`), Liquid (`.liquid`), Mustache (`.mustache`), Handlebars (`.hbs`).
- **Config**: INI (`.ini`), `.env`, `.editorconfig`, `.properties`, HCL/Terraform (`.tf`, `.tfvars`, `.hcl`), Nix (`.nix`).
- **Data/schema**: `.proto`, `.graphql`/`.gql`, `.csv`, `.tsv`, `.jsonl`/`.ndjson`.
- **Build/task files**: Bazel (`BUILD`, `BUILD.bazel`, `WORKSPACE`, `*.bzl`), Justfile / `justfile`, Earthfile, Jenkinsfile (Groovy), Vagrantfile (Ruby), Brewfile (Ruby), Procfile, Caddyfile.
- **XML split** from HTML into its own `LangXML` constant.

Locked design principles (from dev 2026-05-16):

1. Extension-first; content-sniff only as last-resort disambiguator (e.g. `.m` MATLAB-vs-ObjC).
2. One file = one language — no Vue/Svelte sub-parsing, no notebook split.
3. Group only when distinction doesn't matter (`Shell` already groups sh/bash/zsh/fish). Do **not** group CSS preprocessors.
4. Each lang gets: `Language` constant + extension/filename/shebang table entry + comment-split rule + detection test + split test + README "Languages detected" entry.
5. Skip MATLAB, Fortran, VHDL, Verilog — let community add via PR.

## Planner

<Filled by go-planning-agent in Phase 1. Atomic units of work below. Each unit's state is mutated in place by the builder during Phase 4.>

## Notes

**Cross-stream coordination**: this is one of four v0.2.0 streams (A=langs, B=tokens, C=parallel-walk+follow, D=files-from). Stream A is isolated to `internal/lang/*` plus README — it does NOT touch `cmd/rak/root.go`, so no flag-wiring contention with B/C/D.
