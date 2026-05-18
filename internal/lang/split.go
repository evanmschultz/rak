// Package lang — split.go provides blank/comment/code line classification for
// the languages rak detects. Classification uses Policy α (F28, Decision C4):
// any line containing a block-comment marker is classified as Comment,
// regardless of adjacent code. See LineCounts and Split for details.
package lang

import (
	"bufio"
	"io"
	"strings"

	"github.com/evanmschultz/rak/internal/counting"
)

// LineCounts holds a three-way line classification for a single file or an
// aggregate of files. Each line is counted exactly once in one of the three
// buckets.
type LineCounts struct {
	// Blank is the number of lines whose trimmed content is empty.
	Blank int
	// Comment is the number of lines classified as comments per Policy α (F28).
	Comment int
	// Code is the number of non-blank, non-comment lines.
	Code int
}

// LangCounts combines the three-way line split with the raw byte/line/word/char
// counts for a language bucket. It is the per-language accumulation unit used
// by the 5.3 rollup. LangCounts lives in internal/lang per the import DAG
// decision (F30): lang → fileset, counting; render → lang, counting.
type LangCounts struct {
	// Lines is the blank/comment/code line classification.
	Lines LineCounts
	// Counts is the raw byte/line/word/char totals for this language bucket.
	Counts counting.Counts
}

// Add accumulates other into lc field-by-field. It is used by the per-dir
// rollup accumulator in cmd/rak/root.go (Unit 5.3) to aggregate across files
// of the same language.
func (lc *LangCounts) Add(other LangCounts) {
	lc.Lines.Blank += other.Lines.Blank
	lc.Lines.Comment += other.Lines.Comment
	lc.Lines.Code += other.Lines.Code
	lc.Counts.Bytes += other.Counts.Bytes
	lc.Counts.Lines += other.Counts.Lines
	lc.Counts.Words += other.Counts.Words
	lc.Counts.Chars += other.Counts.Chars
}

// grammar holds the comment markers for a single language. Empty strings mean
// the language does not support that comment form.
type grammar struct {
	// linePrefix is the primary single-line comment prefix, e.g. "//" or "#".
	// Empty means no line-comment form.
	linePrefix string
	// linePrefix2 is an optional secondary single-line comment prefix. Used for
	// languages with two valid line-comment starters (e.g. PHP accepts both "//"
	// and "#"). Empty means no secondary form.
	linePrefix2 string
	// blockOpen is the block-comment open marker, e.g. "/*" or "<!--".
	// Empty means no block-comment form.
	blockOpen string
	// blockClose is the block-comment close marker, e.g. "*/" or "-->".
	// Empty means no block-comment form.
	blockClose string
}

// grammarTable maps each language to its comment grammar. Languages absent
// from this table (including LangUnknown) receive an empty grammar — all
// non-blank lines are classified as Code (no comment detection).
//
// Policy α (F28, Decision C4): Split classifies as Comment any line that
// contains blockOpen or blockClose anywhere in the line, regardless of code
// preceding or following the marker. Known limitation: string literals
// containing markers (e.g. s := "/*") are mis-classified as Comment lines.
// This is a deliberate YAGNI trade-off for v0.1.0, matching cloc defaults.
//
// Python docstrings (C7): triple-quoted strings are strings at the language
// level, not comments. Python grammar uses only "#" for line comments; triple-
// quote detection is not implemented. """docstring""" lines are classified as
// Code. This matches cloc behavior.
var grammarTable = map[Language]grammar{
	// C-family and JVM-adjacent languages: "//" line + "/* */" block.
	LangGo:     {linePrefix: "//", blockOpen: "/*", blockClose: "*/"},
	LangRust:   {linePrefix: "//", blockOpen: "/*", blockClose: "*/"},
	LangC:      {linePrefix: "//", blockOpen: "/*", blockClose: "*/"},
	LangCPP:    {linePrefix: "//", blockOpen: "/*", blockClose: "*/"},
	LangJS:     {linePrefix: "//", blockOpen: "/*", blockClose: "*/"},
	LangTS:     {linePrefix: "//", blockOpen: "/*", blockClose: "*/"},
	LangCSS:    {linePrefix: "", blockOpen: "/*", blockClose: "*/"},
	LangJava:   {linePrefix: "//", blockOpen: "/*", blockClose: "*/"},
	LangKotlin: {linePrefix: "//", blockOpen: "/*", blockClose: "*/"},
	// Swift permits nested block comments; rak's flat-detection is acceptable
	// for v0.1.0 — nested "/* /* */ */" will be classified correctly at the
	// outer open/close but inner nesting is not tracked (Policy α, YAGNI).
	LangSwift: {linePrefix: "//", blockOpen: "/*", blockClose: "*/"},
	// PHP accepts both "//" and "#" as line-comment prefixes. linePrefix2 holds
	// the secondary form so both are detected by Split.
	LangPHP: {linePrefix: "//", linePrefix2: "#", blockOpen: "/*", blockClose: "*/"},

	// Hash-comment languages: "#" line only, no block-comment form.
	// Python: triple-quoted docstrings are strings, not comments (C7).
	LangPython:   {linePrefix: "#"},
	LangShell:    {linePrefix: "#"},
	LangTOML:     {linePrefix: "#"},
	LangYAML:     {linePrefix: "#"},
	LangMakefile: {linePrefix: "#"},
	LangDocker:   {linePrefix: "#"},
	LangCMake:    {linePrefix: "#", blockOpen: "#[[", blockClose: "]]"},
	// Ruby uses "#" for line comments. Block comments via =begin/=end are
	// supported: Policy α classifies any line containing "=begin" or "=end"
	// as Comment (known limitation: string literals containing these markers
	// will be mis-classified, consistent with F28 YAGNI for v0.1.0).
	LangRuby: {linePrefix: "#", blockOpen: "=begin", blockClose: "=end"},

	// HTML/XML-family: "<!-- -->" block, no line-comment form.
	LangHTML:     {blockOpen: "<!--", blockClose: "-->"},
	LangXML:      {blockOpen: "<!--", blockClose: "-->"},
	LangMarkdown: {blockOpen: "<!--", blockClose: "-->"},

	// JSON has no comment syntax per spec; all non-blank lines are Code.
	// LangJSON intentionally absent (zero grammar).

	// Unit A.2 — Programming languages.

	// C-family: "//" line + "/* */" block (same as Go/Rust/Java/etc.).
	LangCSharp: {linePrefix: "//", blockOpen: "/*", blockClose: "*/"},
	LangDart:   {linePrefix: "//", blockOpen: "/*", blockClose: "*/"},
	LangScala:  {linePrefix: "//", blockOpen: "/*", blockClose: "*/"},

	// SQL (ANSI): "--" line + "/* */" block.
	LangSQL: {linePrefix: "--", blockOpen: "/*", blockClose: "*/"},

	// Lua: "--" line + "--[[" / "]]" long-bracket block.
	// Known limitation (Policy α YAGNI): "]]" also appears as a table-index
	// operator in Lua code. Lines containing "]]" in code context are
	// mis-classified as Comment. Additionally, "]]" inside string literals can
	// corrupt multi-line block-comment state. Accepted under F28 YAGNI.
	LangLua: {linePrefix: "--", blockOpen: "--[[", blockClose: "]]"},

	// Elixir: "#" line only — Elixir has no block-comment form.
	LangElixir: {linePrefix: "#"},

	// Zig: "//" line only — Zig has no block-comment form.
	// "////" doc comments use the same "//" prefix and are detected correctly.
	LangZig: {linePrefix: "//"},

	// R: "#" line only — R has no block-comment form.
	LangR: {linePrefix: "#"},

	// F# (ML-style): "//" line + "(* *)" block.
	LangFSharp: {linePrefix: "//", blockOpen: "(*", blockClose: "*)"},

	// Haskell: "--" line + "{- -}" block.
	LangHaskell: {linePrefix: "--", blockOpen: "{-", blockClose: "-}"},

	// Unit A.3 — Templating and frontend variants.

	// Go-superset templ: "//" line + "/* */" block (same as Go).
	// HTML-like <!-- --> comments inside .templ files classify as Code
	// (single-grammar policy, design principle 2, out of scope v0.2.0).
	LangTempl: {linePrefix: "//", blockOpen: "/*", blockClose: "*/"},

	// JSX/TSX: JS/TS family "//" line + "/* */" block.
	LangJSX: {linePrefix: "//", blockOpen: "/*", blockClose: "*/"},
	LangTSX: {linePrefix: "//", blockOpen: "/*", blockClose: "*/"},

	// CSS preprocessors: "//" line + "/* */" block.
	// SCSS supports both forms per spec. Sass indented syntax uses // for
	// line comments; /* */ exists but is less common (Policy α YAGNI).
	// LESS supports both forms per spec.
	LangSCSS: {linePrefix: "//", blockOpen: "/*", blockClose: "*/"},
	LangSass: {linePrefix: "//", blockOpen: "/*", blockClose: "*/"},
	LangLESS: {linePrefix: "//", blockOpen: "/*", blockClose: "*/"},

	// Vue/Svelte: HTML-level <!-- --> block, no line-comment form.
	// JS/TS comments inside <script> blocks use JS/TS syntax invisible to
	// rak's grammar → those lines classify as Code (design principle 2,
	// one file = one grammar, sub-parsing out of scope for v0.2.0).
	LangVue:    {blockOpen: "<!--", blockClose: "-->"},
	LangSvelte: {blockOpen: "<!--", blockClose: "-->"},

	// ERB: block form <%# ... %> rather than linePrefix "<%#".
	// The linePrefix form (strings.HasPrefix) only matches when "<%#" is at
	// trimmed-line start. Real ERB files commonly have mid-line comments like
	// "<%= val %> <%# note %>" where "<%#" is not at line start. The block
	// form (strings.Contains) catches those. Trade-off: blockClose "%>" also
	// appears on expression-output lines like "<%= value %>" — those lines are
	// mis-classified as Comment under Policy α (known limitation, YAGNI F28).
	// HTML <!-- --> comments inside .erb files are HTML output written to the
	// browser, not ERB-level comments; they classify as Code (intentional).
	LangERB: {blockOpen: "<%#", blockClose: "%>"},

	// Jinja2: {# ... #} block comment syntax.
	LangJinja: {blockOpen: "{#", blockClose: "#}"},

	// Liquid: {% comment %} / {% endcomment %} block tags.
	LangLiquid: {blockOpen: "{% comment %}", blockClose: "{% endcomment %}"},

	// Mustache/Handlebars: "{{!" line prefix + "{{!--" / "--}}" block.
	// linePrefix "{{!" catches single-line {{! comment }} style.
	// blockOpen "{{!--" + blockClose "--}}" catches the multi-line form.
	// Note: "{{!" is a prefix of "{{!--" so the linePrefix fires first on
	// single-line {{! comments; the blockOpen fires on {{!-- block opens.
	LangMustache: {linePrefix: "{{!", blockOpen: "{{!--", blockClose: "--}}"},
}

// Split reads r line by line and classifies each line as Blank, Comment, or
// Code according to lang's comment grammar and Policy α (F28). It returns the
// aggregated LineCounts and any scanner error.
//
// Policy α: a line is classified as Comment if any of the following holds:
//   - The scanner is inside a block comment at the start of the line.
//   - The line contains the block-comment open OR close marker anywhere.
//   - The trimmed line begins with the language's line-comment prefix.
//
// This is intentionally coarse — lines where a comment marker appears only
// inside a string literal are still classified as Comment (known limitation,
// F28, YAGNI v0.1.0).
//
// For languages with no grammar entry (including LangUnknown), all non-blank
// lines are classified as Code.
func Split(r io.Reader, lang Language) (LineCounts, error) {
	g := grammarTable[lang] // zero grammar if absent

	scanner := bufio.NewScanner(r)
	var lc LineCounts
	inBlockComment := false

	for scanner.Scan() {
		line := scanner.Text() // CRLF already stripped by ScanLines
		trimmed := strings.TrimSpace(line)

		// Blank line.
		if trimmed == "" {
			lc.Blank++
			continue
		}

		// Determine if this line is a Comment or Code.
		isComment := false

		// (a) Inside a block comment at line start.
		if inBlockComment {
			isComment = true
		}

		// (b) Block marker anywhere on the line (Policy α).
		if !isComment && g.blockOpen != "" && strings.Contains(line, g.blockOpen) {
			isComment = true
		}
		if !isComment && g.blockClose != "" && strings.Contains(line, g.blockClose) {
			isComment = true
		}

		// (c) Line-comment prefix at trimmed start (primary and optional secondary).
		if !isComment && g.linePrefix != "" && strings.HasPrefix(trimmed, g.linePrefix) {
			isComment = true
		}
		if !isComment && g.linePrefix2 != "" && strings.HasPrefix(trimmed, g.linePrefix2) {
			isComment = true
		}

		if isComment {
			lc.Comment++
		} else {
			lc.Code++
		}

		// Update block-comment state: scan for open/close markers; last one wins.
		if g.blockOpen != "" {
			idx := 0
			for {
				openIdx := strings.Index(line[idx:], g.blockOpen)
				closeIdx := strings.Index(line[idx:], g.blockClose)
				if openIdx == -1 && closeIdx == -1 {
					break
				}
				if openIdx != -1 && (closeIdx == -1 || openIdx < closeIdx) {
					inBlockComment = true
					idx += openIdx + len(g.blockOpen)
				} else {
					inBlockComment = false
					idx += closeIdx + len(g.blockClose)
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return LineCounts{}, err
	}
	return lc, nil
}
