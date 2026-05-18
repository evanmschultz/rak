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
