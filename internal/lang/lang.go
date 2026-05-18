// Package lang provides language detection for files walked by rak. It
// classifies each file into a Language value using a four-step pipeline:
// (1) exact-basename lookup for special filenames (Makefile, Dockerfile, etc.),
// (2) file-extension lookup, (3) shebang sniff for extensionless executables,
// and (4) a best-effort content-marker heuristic. Detection never propagates
// I/O errors; failures return LangUnknown silently. See F27 in DROP_5's
// PLAN.md for the canonical pipeline pin.
package lang

import (
	"bytes"
	"path/filepath"
	"strings"

	"github.com/evanmschultz/rak/internal/fileset"
)

// Language is a named string type that identifies a programming or markup
// language. Language values are stored lowercase by convention (e.g. "go",
// "rust"). LangUnknown is the zero value and is returned by Detect when no
// rule matches.
type Language string

// Language constants used throughout rak. Values are lowercase strings.
// LangUnknown is the zero value: an empty string, returned by Detect when no
// detection rule matches the file.
const (
	LangUnknown  Language = ""
	LangC        Language = "c"
	LangCPP      Language = "cpp"
	LangCMake    Language = "cmake"
	LangCSS      Language = "css"
	LangDocker   Language = "docker"
	LangGo       Language = "go"
	LangHTML     Language = "html"
	LangJava     Language = "java"
	LangJS       Language = "javascript"
	LangJSON     Language = "json"
	LangKotlin   Language = "kotlin"
	LangMakefile Language = "makefile"
	LangMarkdown Language = "markdown"
	LangPHP      Language = "php"
	LangPython   Language = "python"
	LangRuby     Language = "ruby"
	LangRust     Language = "rust"
	LangShell    Language = "shell"
	LangSwift    Language = "swift"
	LangTS       Language = "typescript"
	LangTOML     Language = "toml"
	LangXML      Language = "xml"
	LangYAML     Language = "yaml"

	// Unit A.2 — Programming languages.

	// LangCSharp is the Language constant for C# source files (.cs).
	LangCSharp Language = "csharp"
	// LangDart is the Language constant for Dart source files (.dart).
	LangDart Language = "dart"
	// LangElixir is the Language constant for Elixir source files (.ex, .exs).
	LangElixir Language = "elixir"
	// LangFSharp is the Language constant for F# source files (.fs, .fsi, .fsx).
	LangFSharp Language = "fsharp"
	// LangHaskell is the Language constant for Haskell source files (.hs, .lhs).
	LangHaskell Language = "haskell"
	// LangLua is the Language constant for Lua source files (.lua).
	LangLua Language = "lua"
	// LangR is the Language constant for R source files (.r — filepath.Ext
	// lowercases, so both .r and .R files map here via strings.ToLower in Detect).
	LangR Language = "r"
	// LangScala is the Language constant for Scala source files (.scala).
	LangScala Language = "scala"
	// LangSQL is the Language constant for SQL source files (.sql).
	LangSQL Language = "sql"
	// LangZig is the Language constant for Zig source files (.zig).
	LangZig Language = "zig"

	// Unit A.3 — Templating and frontend variants.

	// LangTempl is the Language constant for Go-superset templ files (.templ).
	// Templ uses Go-style comment syntax (// and /* */).
	LangTempl Language = "templ"
	// LangJSX is the Language constant for React JSX files (.jsx).
	LangJSX Language = "jsx"
	// LangTSX is the Language constant for TypeScript JSX files (.tsx).
	// Distinct from .ts → LangTS.
	LangTSX Language = "tsx"
	// LangSCSS is the Language constant for SCSS stylesheets (.scss).
	// SCSS supports both // line comments and /* */ block comments.
	LangSCSS Language = "scss"
	// LangSass is the Language constant for indented Sass stylesheets (.sass).
	// Uses // for line comments; /* */ block comments exist but are less common
	// (Policy α YAGNI — some non-comment lines may be over-classified).
	LangSass Language = "sass"
	// LangLESS is the Language constant for LESS stylesheets (.less).
	LangLESS Language = "less"
	// LangVue is the Language constant for Vue single-file components (.vue).
	// Grammar covers HTML-level <!-- --> comments; JS/TS inside <script> blocks
	// uses JS/TS comment syntax not detected here (one file = one grammar,
	// design principle 2, out of scope for v0.2.0).
	LangVue Language = "vue"
	// LangSvelte is the Language constant for Svelte components (.svelte).
	// Same single-grammar HTML-level policy as LangVue.
	LangSvelte Language = "svelte"
	// LangERB is the Language constant for Ruby ERB templates (.erb).
	// Grammar uses block form <%# ... %> to catch mid-line ERB comments.
	// Known limitation: %> also appears on expression-output lines like
	// <%= value %> — those lines are mis-classified as Comment (Policy α YAGNI).
	LangERB Language = "erb"
	// LangJinja is the Language constant for Jinja2 templates
	// (.j2, .jinja, .jinja2).
	LangJinja Language = "jinja"
	// LangLiquid is the Language constant for Liquid templates (.liquid).
	LangLiquid Language = "liquid"
	// LangMustache is the Language constant for Mustache and Handlebars templates
	// (.mustache, .hbs). Handlebars is a Mustache superset sharing the same
	// comment syntax; one constant follows the existing pattern of grouping
	// closely-related variants (Shell groups sh/bash/zsh/fish).
	LangMustache Language = "mustache"

	// Unit A.4 — Config and data formats.

	// LangINI is the Language constant for INI configuration files (.ini).
	// INI uses ";" as the primary line-comment prefix and "#" as a secondary.
	LangINI Language = "ini"
	// LangEnv is the Language constant for dotenv environment files (.env).
	// filepath.Ext(".env") returns ".env" so standalone dotfiles match correctly.
	LangEnv Language = "env"
	// LangEditorConfig is the Language constant for EditorConfig files
	// (.editorconfig). Uses "#" for line comments per the EditorConfig spec.
	LangEditorConfig Language = "editorconfig"
	// LangProperties is the Language constant for Java .properties files
	// (.properties). Uses "#" as primary and "!" as secondary line-comment prefix.
	LangProperties Language = "properties"
	// LangHCL is the Language constant for HashiCorp Configuration Language files
	// (.tf, .tfvars, .hcl). HCL supports "#", "//", and "/* */" comment forms.
	LangHCL Language = "hcl"
	// LangNix is the Language constant for Nix expression language files (.nix).
	// Uses "#" for line comments and "/* */" for block comments.
	LangNix Language = "nix"
	// LangProto is the Language constant for Protocol Buffer definition files
	// (.proto). Uses "//" for line comments and "/* */" for block comments.
	LangProto Language = "proto"
	// LangGraphQL is the Language constant for GraphQL schema definition files
	// (.graphql, .gql). "#" is the only comment form in GraphQL SDL.
	LangGraphQL Language = "graphql"
	// LangCSV is the Language constant for Comma-Separated Values files (.csv).
	// CSV has no comment syntax; all non-blank lines are classified as Code.
	LangCSV Language = "csv"
	// LangTSV is the Language constant for Tab-Separated Values files (.tsv).
	// TSV has no comment syntax; all non-blank lines are classified as Code.
	LangTSV Language = "tsv"
	// LangJSONL is the Language constant for JSON Lines files (.jsonl, .ndjson).
	// JSON Lines has no comment syntax; all non-blank lines are classified as Code.
	LangJSONL Language = "jsonl"

	// Unit A.5 — Build and task files.

	// LangBazel is the Language constant for Bazel build files (BUILD, BUILD.bazel,
	// WORKSPACE special filenames and .bzl extension). Bazel uses Starlark
	// (Python-like) syntax with "#" for line comments.
	LangBazel Language = "bazel"
	// LangGroovy is the Language constant for Groovy source files, including
	// Jenkinsfile (special filename). Groovy is a Java-family language with "//"
	// line comments and "/* */" block comments.
	LangGroovy Language = "groovy"
	// LangJust is the Language constant for Justfile task runner files (Justfile
	// and justfile special filenames). Uses "#" for line comments.
	LangJust Language = "just"
	// LangEarth is the Language constant for Earthly build files (Earthfile special
	// filename). Uses "#" for line comments per Earthly syntax.
	LangEarth Language = "earth"
	// LangCaddy is the Language constant for Caddyfile web server configuration
	// files (Caddyfile special filename). Uses "#" for line comments.
	LangCaddy Language = "caddy"
)

// specialFilenames maps exact lowercased basenames to languages. Lookup is
// performed before extension lookup so that files like "Makefile" (which have
// no extension) are detected correctly regardless of containing directory.
// Keys must already be lowercase; Detect normalizes the basename before lookup.
var specialFilenames = map[string]Language{
	"cmakelists.txt": LangCMake,
	"dockerfile":     LangDocker,
	"gemfile":        LangRuby,
	"gnumakefile":    LangMakefile,
	"makefile":       LangMakefile,
	"rakefile":       LangRuby,

	// Unit A.5 — Build and task files.
	// Bazel: BUILD, BUILD.bazel, and WORKSPACE are well-known Bazel entry points.
	// Keys are pre-lowercased; Detect lowercases the basename before lookup.
	"build":       LangBazel,
	"build.bazel": LangBazel,
	"workspace":   LangBazel,
	// Groovy: Jenkinsfile is the standard CI pipeline file. The constant is
	// LangGroovy (not LangJenkinsfile) because Groovy is the actual language.
	"jenkinsfile": LangGroovy,
	// Just: both Justfile (conventional) and justfile (lowercase) are valid.
	"justfile": LangJust,
	// Earth: Earthfile is the standard Earthly build definition file.
	"earthfile": LangEarth,
	// Caddy: Caddyfile is the standard Caddy web server configuration file.
	"caddyfile": LangCaddy,
	// Ruby DSLs: Vagrantfile and Brewfile are Ruby DSLs; they reuse LangRuby
	// (same as existing Gemfile/Rakefile pattern). No new constant needed.
	"vagrantfile": LangRuby,
	"brewfile":    LangRuby,
	// Procfile is intentionally NOT listed here. Files named "Procfile" count
	// as bytes/lines/words but return LangUnknown from Detect (YAGNI cut,
	// 2026-05-16: nobody asked to filter by Procfile specifically). If a user
	// requests Procfile detection, add in v0.2.1+ with a LangProcfile constant.
}

// extensionTable maps lowercased file extensions (with the leading dot, e.g.
// ".go") to languages. Keys match filepath.Ext output directly (F27 / P5).
var extensionTable = map[string]Language{
	".bash":    LangShell,
	".c":       LangC,
	".cc":      LangCPP,
	".cpp":     LangCPP,
	".css":     LangCSS,
	".cxx":     LangCPP,
	".fish":    LangShell,
	".gemspec": LangRuby,
	".go":      LangGo,
	".h":       LangC,
	".hpp":     LangCPP,
	".htm":     LangHTML,
	".html":    LangHTML,
	".java":    LangJava,
	".js":      LangJS,
	".json":    LangJSON,
	".kt":      LangKotlin,
	".kts":     LangKotlin,
	".md":      LangMarkdown,
	".php":     LangPHP,
	".phtml":   LangPHP,
	".py":      LangPython,
	".rake":    LangRuby,
	".rb":      LangRuby,
	".rs":      LangRust,
	".sh":      LangShell,
	".swift":   LangSwift,
	".toml":    LangTOML,
	".ts":      LangTS,
	".xml":     LangXML,
	".yaml":    LangYAML,
	".yml":     LangYAML,
	".zsh":     LangShell,

	// Unit A.2 — Programming languages.
	".cs":    LangCSharp,
	".dart":  LangDart,
	".ex":    LangElixir,
	".exs":   LangElixir,
	".fs":    LangFSharp,
	".fsi":   LangFSharp,
	".fsx":   LangFSharp,
	".hs":    LangHaskell,
	".lhs":   LangHaskell,
	".lua":   LangLua,
	".r":     LangR,
	".scala": LangScala,
	".sql":   LangSQL,
	".zig":   LangZig,

	// Unit A.3 — Templating and frontend variants.
	".templ":    LangTempl,
	".jsx":      LangJSX,
	".tsx":      LangTSX,
	".scss":     LangSCSS,
	".sass":     LangSass,
	".less":     LangLESS,
	".vue":      LangVue,
	".svelte":   LangSvelte,
	".erb":      LangERB,
	".j2":       LangJinja,
	".jinja":    LangJinja,
	".jinja2":   LangJinja,
	".liquid":   LangLiquid,
	".mustache": LangMustache,
	".hbs":      LangMustache,

	// Unit A.4 — Config and data formats.
	".ini":          LangINI,
	".env":          LangEnv,
	".editorconfig": LangEditorConfig,
	".properties":   LangProperties,
	".tf":           LangHCL,
	".tfvars":       LangHCL,
	".hcl":          LangHCL,
	".nix":          LangNix,
	".proto":        LangProto,
	".graphql":      LangGraphQL,
	".gql":          LangGraphQL,
	".csv":          LangCSV,
	".tsv":          LangTSV,
	".jsonl":        LangJSONL,
	".ndjson":       LangJSONL,

	// Unit A.5 — Build and task files.
	// .bzl is the Starlark extension used for Bazel macro and rule files.
	".bzl": LangBazel,
}

// shebangsTable maps interpreter basenames to languages. For
// "#!/usr/bin/env python3" the interpreter path is "/usr/bin/env" and the
// real interpreter argument is "python3" — Detect handles the env-indirection
// case explicitly before consulting this table.
var shebangsTable = map[string]Language{
	"bash":    LangShell,
	"fish":    LangShell,
	"node":    LangJS,
	"nodejs":  LangJS,
	"python":  LangPython,
	"python2": LangPython,
	"python3": LangPython,
	"ruby":    LangRuby,
	"sh":      LangShell,
	"zsh":     LangShell,
}

// Detect classifies f's language using a four-step priority pipeline (F27):
//
//  1. Special-filename lookup — exact case-insensitive basename match against
//     specialFilenames (e.g. "Makefile", "Dockerfile"). Nested paths like
//     "sub/Makefile" match on the basename only. "Makefile.go" does NOT match
//     because its lowercased basename "makefile.go" is not a key in the table.
//
//  2. Extension lookup — filepath.Ext(f.RelPath) lowercased, looked up in
//     extensionTable (e.g. ".go" → LangGo). Pure; no I/O.
//
//  3. Shebang sniff — runs only when steps 1 and 2 both returned LangUnknown.
//     Calls f.Peek(512). If the first line starts with "#!" the interpreter
//     path is extracted and looked up in shebangsTable. "env"-indirected
//     shebangs (#!/usr/bin/env python3) use the following argument as the
//     lookup key. If Peek returns an error, Detect silently returns LangUnknown
//     without propagating the error (F27 / P3).
//
//  4. Content heuristic — runs only when steps 1–3 all returned LangUnknown.
//     Scans the first 512 bytes for well-known markers (<?xml, <!DOCTYPE,
//     leading { or [ as JSON candidates, --- as YAML front-matter). Best-effort;
//     returns LangUnknown when no marker matches.
//
// Detect never propagates I/O errors. Detection failure always returns
// LangUnknown. There is no intermediate "generic" state — the pipeline returns
// the first concrete match or LangUnknown (C5).
func Detect(f *fileset.File) Language {
	// Step 1 — special-filename lookup (case-insensitive basename match).
	base := strings.ToLower(filepath.Base(f.RelPath))
	if lang, ok := specialFilenames[base]; ok {
		return lang
	}

	// Step 2 — extension lookup.
	ext := strings.ToLower(filepath.Ext(f.RelPath))
	if ext != "" {
		if lang, ok := extensionTable[ext]; ok {
			return lang
		}
	}

	// Steps 3 and 4 both need the peeked bytes; fetch once.
	buf, err := f.Peek(512)
	if err != nil {
		// Peek failure → detection failure → LangUnknown (F27 / P3).
		return LangUnknown
	}

	// Step 3 — shebang sniff (only when 1+2 returned LangUnknown).
	if lang := detectShebang(buf); lang != LangUnknown {
		return lang
	}

	// Step 4 — content heuristic (only when 1+2+3 all returned LangUnknown).
	return detectContent(buf)
}

// detectShebang parses the first line of buf for a "#!" shebang and looks up
// the interpreter in shebangsTable. Returns LangUnknown when the first line is
// not a shebang or the interpreter is unrecognized.
func detectShebang(buf []byte) Language {
	if !bytes.HasPrefix(buf, []byte("#!")) {
		return LangUnknown
	}

	// Extract first line (up to newline or end of buf).
	firstLine := buf
	if idx := bytes.IndexByte(buf, '\n'); idx >= 0 {
		firstLine = buf[:idx]
	}

	// Strip the "#!" prefix and trim whitespace.
	line := strings.TrimSpace(string(firstLine[2:]))
	if line == "" {
		return LangUnknown
	}

	// Split on whitespace to get the interpreter path and (possibly) args.
	// e.g. "/usr/bin/env python3" → ["/usr/bin/env", "python3"]
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return LangUnknown
	}

	// Get the basename of the interpreter.
	interp := filepath.Base(parts[0])

	// env-indirection: "#!/usr/bin/env python3" → use "python3" as lookup key.
	if interp == "env" && len(parts) >= 2 {
		// The next argument may have flags (e.g. "env -S python3"); skip
		// arguments starting with "-".
		for _, arg := range parts[1:] {
			if !strings.HasPrefix(arg, "-") {
				interp = filepath.Base(arg)
				break
			}
		}
	}

	interp = strings.ToLower(interp)
	if lang, ok := shebangsTable[interp]; ok {
		return lang
	}
	return LangUnknown
}

// detectContent scans buf for well-known content markers as a best-effort
// language hint. Returns LangUnknown when no marker matches.
func detectContent(buf []byte) Language {
	if len(buf) == 0 {
		return LangUnknown
	}

	// Trim leading whitespace for marker checks.
	trimmed := bytes.TrimSpace(buf)

	switch {
	case bytes.HasPrefix(trimmed, []byte("<?xml")):
		return LangXML
	case bytes.HasPrefix(trimmed, []byte("<!DOCTYPE")):
		return LangHTML
	case bytes.HasPrefix(trimmed, []byte("---")):
		return LangYAML
	case len(trimmed) > 0 && (trimmed[0] == '{' || trimmed[0] == '['):
		return LangJSON
	}
	return LangUnknown
}
