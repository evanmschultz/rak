package lang

import (
	"strings"
	"testing"

	"github.com/evanmschultz/rak/internal/counting"
)

// TestSplit_GoSimple verifies the three-way split for a minimal Go snippet:
// one line comment, one blank line, one code line.
func TestSplit_GoSimple(t *testing.T) {
	t.Parallel()

	input := "// comment\n\nfunc main() {}\n"
	got, err := Split(strings.NewReader(input), LangGo)
	if err != nil {
		t.Fatalf("Split: unexpected error: %v", err)
	}
	want := LineCounts{Blank: 1, Comment: 1, Code: 1}
	if got != want {
		t.Errorf("got %+v; want %+v", got, want)
	}
}

// TestSplit_BlockCommentOpenClosePerLine verifies Policy α: a line containing
// both a block-open and a block-close marker is classified as Comment, even
// though code appears between them.
func TestSplit_BlockCommentOpenClosePerLine(t *testing.T) {
	t.Parallel()

	input := "/* a */ b /* c */\n"
	got, err := Split(strings.NewReader(input), LangGo)
	if err != nil {
		t.Fatalf("Split: unexpected error: %v", err)
	}
	want := LineCounts{Blank: 0, Comment: 1, Code: 0}
	if got != want {
		t.Errorf("got %+v; want %+v", got, want)
	}
}

// TestSplit_TrailingComment verifies Policy α: a trailing block-comment marker
// anywhere on the line promotes the entire line to Comment.
func TestSplit_TrailingComment(t *testing.T) {
	t.Parallel()

	input := "x := 1 /* note */\n"
	got, err := Split(strings.NewReader(input), LangGo)
	if err != nil {
		t.Fatalf("Split: unexpected error: %v", err)
	}
	want := LineCounts{Blank: 0, Comment: 1, Code: 0}
	if got != want {
		t.Errorf("got %+v; want %+v", got, want)
	}
}

// TestSplit_StringContainsMarker_KnownLimitation documents the known YAGNI
// limitation of Policy α: a string literal containing "/*" or "*/" is
// mis-classified as a Comment line. This is intentional for v0.1.0 (F28).
func TestSplit_StringContainsMarker_KnownLimitation(t *testing.T) {
	t.Parallel()

	// s := "/*" contains "/*" → classified as Comment (known limitation).
	input := "s := \"/*\"\n"
	got, err := Split(strings.NewReader(input), LangGo)
	if err != nil {
		t.Fatalf("Split: unexpected error: %v", err)
	}
	want := LineCounts{Blank: 0, Comment: 1, Code: 0}
	if got != want {
		t.Errorf("got %+v; want %+v (Policy α YAGNI known limitation)", got, want)
	}
}

// TestSplit_BlockSpansMultipleLines verifies that the block-comment state
// machine correctly tracks inBlockComment across multiple lines.
func TestSplit_BlockSpansMultipleLines(t *testing.T) {
	t.Parallel()

	input := "/* line one\nline two */\n"
	got, err := Split(strings.NewReader(input), LangGo)
	if err != nil {
		t.Fatalf("Split: unexpected error: %v", err)
	}
	want := LineCounts{Blank: 0, Comment: 2, Code: 0}
	if got != want {
		t.Errorf("got %+v; want %+v", got, want)
	}
}

// TestSplit_PythonHash verifies that Python uses "#" as the line-comment prefix.
func TestSplit_PythonHash(t *testing.T) {
	t.Parallel()

	input := "# comment\nx = 1\n"
	got, err := Split(strings.NewReader(input), LangPython)
	if err != nil {
		t.Fatalf("Split: unexpected error: %v", err)
	}
	want := LineCounts{Blank: 0, Comment: 1, Code: 1}
	if got != want {
		t.Errorf("got %+v; want %+v", got, want)
	}
}

// TestSplit_PythonDocstring_IsCode verifies Policy C7: Python triple-quoted
// strings are strings at the language level, not comments. Split classifies
// them as Code lines (matching cloc behavior). Python grammar uses "#" only;
// triple-quote detection is out of scope for v0.1.0 (F28).
func TestSplit_PythonDocstring_IsCode(t *testing.T) {
	t.Parallel()

	input := "def f():\n    \"\"\"docstring\"\"\"\n"
	got, err := Split(strings.NewReader(input), LangPython)
	if err != nil {
		t.Fatalf("Split: unexpected error: %v", err)
	}
	// Both lines are non-blank and have no "#" prefix → Code.
	want := LineCounts{Blank: 0, Comment: 0, Code: 2}
	if got != want {
		t.Errorf("got %+v; want %+v (Python docstrings are Code per C7)", got, want)
	}
}

// TestSplit_JSON_NoComments verifies that JSON has no comment syntax: every
// non-blank line is classified as Code, even lines that resemble comments in
// other languages.
func TestSplit_JSON_NoComments(t *testing.T) {
	t.Parallel()

	// JSON technically cannot have comments; this tests the grammar absence.
	input := "{\n  \"key\": \"// not a comment\",\n  \"val\": 1\n}\n"
	got, err := Split(strings.NewReader(input), LangJSON)
	if err != nil {
		t.Fatalf("Split: unexpected error: %v", err)
	}
	want := LineCounts{Blank: 0, Comment: 0, Code: 4}
	if got != want {
		t.Errorf("got %+v; want %+v (JSON has no comment syntax)", got, want)
	}
}

// TestSplit_LangCounts_Add verifies that LangCounts.Add accumulates all fields
// from other into lc correctly.
func TestSplit_LangCounts_Add(t *testing.T) {
	t.Parallel()

	base := LangCounts{
		Lines:  LineCounts{Blank: 1, Comment: 2, Code: 3},
		Counts: counting.Counts{Bytes: 10, Lines: 6, Words: 4, Chars: 12},
	}
	other := LangCounts{
		Lines:  LineCounts{Blank: 4, Comment: 5, Code: 6},
		Counts: counting.Counts{Bytes: 20, Lines: 15, Words: 8, Chars: 24},
	}
	base.Add(other)

	wantLines := LineCounts{Blank: 5, Comment: 7, Code: 9}
	wantCounts := counting.Counts{Bytes: 30, Lines: 21, Words: 12, Chars: 36}
	if base.Lines != wantLines {
		t.Errorf("Lines: got %+v; want %+v", base.Lines, wantLines)
	}
	if base.Counts != wantCounts {
		t.Errorf("Counts: got %+v; want %+v", base.Counts, wantCounts)
	}
}

// TestSplit_EmptyInput verifies that an empty reader returns zero LineCounts
// and nil error.
func TestSplit_EmptyInput(t *testing.T) {
	t.Parallel()

	got, err := Split(strings.NewReader(""), LangGo)
	if err != nil {
		t.Fatalf("Split: unexpected error: %v", err)
	}
	want := LineCounts{}
	if got != want {
		t.Errorf("got %+v; want zero LineCounts", got)
	}
}

// TestSplit_CRLF verifies that bufio.Scanner's default ScanLines correctly
// strips the "\r" from CRLF line endings, so line classification works the
// same as with LF-only input.
func TestSplit_CRLF(t *testing.T) {
	t.Parallel()

	input := "line1\r\nline2\r\n"
	got, err := Split(strings.NewReader(input), LangGo)
	if err != nil {
		t.Fatalf("Split: unexpected error: %v", err)
	}
	want := LineCounts{Blank: 0, Comment: 0, Code: 2}
	if got != want {
		t.Errorf("got %+v; want %+v", got, want)
	}
}

// TestSplit_LangUnknown_AllCode verifies that when lang is LangUnknown (no
// grammar entry), all non-blank lines are classified as Code. No comment
// detection is applied.
func TestSplit_LangUnknown_AllCode(t *testing.T) {
	t.Parallel()

	input := "a\nb\n"
	got, err := Split(strings.NewReader(input), LangUnknown)
	if err != nil {
		t.Fatalf("Split: unexpected error: %v", err)
	}
	want := LineCounts{Blank: 0, Comment: 0, Code: 2}
	if got != want {
		t.Errorf("got %+v; want %+v (LangUnknown = no grammar = all Code)", got, want)
	}
}

// TestSplit_Ruby verifies that Ruby uses "#" for line comments and
// "=begin"/"=end" for block comments (Policy α applies).
func TestSplit_Ruby(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input string
		want  LineCounts
	}{
		{
			name:  "hash line comment",
			input: "# comment\nputs 'hi'\n",
			want:  LineCounts{Blank: 0, Comment: 1, Code: 1},
		},
		{
			// Policy α: =begin (blockOpen) marks line 1 as Comment and sets
			// inBlockComment=true. Line 2 ("This is a block comment.") is inside
			// the block → Comment. Line 3 (=end) is inside block at start →
			// Comment; closes block. Line 4 (puts 'code') → Code.
			name:  "block comment =begin/=end",
			input: "=begin\nThis is a block comment.\n=end\nputs 'code'\n",
			want:  LineCounts{Blank: 0, Comment: 3, Code: 1},
		},
		{
			name:  "blank and code only",
			input: "puts 'a'\n\nputs 'b'\n",
			want:  LineCounts{Blank: 1, Comment: 0, Code: 2},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := Split(strings.NewReader(tc.input), LangRuby)
			if err != nil {
				t.Fatalf("Split: unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %+v; want %+v", got, tc.want)
			}
		})
	}
}

// TestSplit_Java verifies that Java uses "//" for line comments and
// "/* */" for block comments, consistent with C-family Policy α.
func TestSplit_Java(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input string
		want  LineCounts
	}{
		{
			name:  "line comment",
			input: "// comment\nint x = 1;\n",
			want:  LineCounts{Blank: 0, Comment: 1, Code: 1},
		},
		{
			name:  "block comment",
			input: "/* open\n * middle\n */\nint y = 2;\n",
			want:  LineCounts{Blank: 0, Comment: 3, Code: 1},
		},
		{
			name:  "inline block comment (Policy α)",
			input: "int z = /* value */ 3;\n",
			want:  LineCounts{Blank: 0, Comment: 1, Code: 0},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := Split(strings.NewReader(tc.input), LangJava)
			if err != nil {
				t.Fatalf("Split: unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %+v; want %+v", got, tc.want)
			}
		})
	}
}

// TestSplit_PHP verifies that PHP uses both "//" and "#" as line-comment
// prefixes and "/* */" for block comments.
func TestSplit_PHP(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input string
		want  LineCounts
	}{
		{
			name:  "slashslash line comment",
			input: "// comment\n$x = 1;\n",
			want:  LineCounts{Blank: 0, Comment: 1, Code: 1},
		},
		{
			name:  "hash line comment",
			input: "# comment\n$y = 2;\n",
			want:  LineCounts{Blank: 0, Comment: 1, Code: 1},
		},
		{
			name:  "block comment",
			input: "/* open\n * middle\n */\n$z = 3;\n",
			want:  LineCounts{Blank: 0, Comment: 3, Code: 1},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := Split(strings.NewReader(tc.input), LangPHP)
			if err != nil {
				t.Fatalf("Split: unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %+v; want %+v", got, tc.want)
			}
		})
	}
}

// TestSplit_Kotlin verifies that Kotlin uses "//" for line comments and
// "/* */" for block comments.
func TestSplit_Kotlin(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input string
		want  LineCounts
	}{
		{
			name:  "line comment",
			input: "// comment\nfun main() {}\n",
			want:  LineCounts{Blank: 0, Comment: 1, Code: 1},
		},
		{
			name:  "block comment",
			input: "/* open\n * doc\n */\nval x = 1\n",
			want:  LineCounts{Blank: 0, Comment: 3, Code: 1},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := Split(strings.NewReader(tc.input), LangKotlin)
			if err != nil {
				t.Fatalf("Split: unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %+v; want %+v", got, tc.want)
			}
		})
	}
}

// TestSplit_XML verifies Unit A.1: LangXML uses the same "<!-- -->" grammar
// as LangHTML. A line containing "<!-- comment -->" is classified as Comment;
// a plain element line is Code. This confirms the grammar entry is correct
// and that XML and HTML share the same comment delimiter.
func TestSplit_XML(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input string
		want  LineCounts
	}{
		{
			name:  "xml comment line",
			input: "<!-- comment -->\n",
			want:  LineCounts{Blank: 0, Comment: 1, Code: 0},
		},
		{
			name:  "xml element is Code",
			input: "<root>\n  <child/>\n</root>\n",
			want:  LineCounts{Blank: 0, Comment: 0, Code: 3},
		},
		{
			name:  "mixed comment and element",
			input: "<!-- note -->\n<item/>\n\n",
			want:  LineCounts{Blank: 1, Comment: 1, Code: 1},
		},
		{
			name:  "multiline xml block comment",
			input: "<!-- open\n     body\n-->\n<end/>\n",
			want:  LineCounts{Blank: 0, Comment: 3, Code: 1},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := Split(strings.NewReader(tc.input), LangXML)
			if err != nil {
				t.Fatalf("Split: unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %+v; want %+v", got, tc.want)
			}
		})
	}
}

// TestSplit_ProgrammingLanguages verifies Unit A.2: blank/comment/code
// classification for each of the 10 new language grammars. One representative
// snippet per language confirms the comment markers are registered correctly.
func TestSplit_ProgrammingLanguages(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		lang  Language
		input string
		want  LineCounts
	}{
		// C# — C-family "//" line + "/* */" block.
		{
			name:  "csharp line comment",
			lang:  LangCSharp,
			input: "// comment\nint x = 1;\n",
			want:  LineCounts{Blank: 0, Comment: 1, Code: 1},
		},
		{
			name:  "csharp block comment",
			lang:  LangCSharp,
			input: "/* open\n * doc\n */\nclass Foo {}\n",
			want:  LineCounts{Blank: 0, Comment: 3, Code: 1},
		},
		// Scala — C-family "//" line + "/* */" block.
		{
			name:  "scala line comment",
			lang:  LangScala,
			input: "// comment\nval x = 1\n",
			want:  LineCounts{Blank: 0, Comment: 1, Code: 1},
		},
		{
			name:  "scala block comment",
			lang:  LangScala,
			input: "/* open\n * body\n */\nval y = 2\n",
			want:  LineCounts{Blank: 0, Comment: 3, Code: 1},
		},
		// Dart — C-family "//" line + "/* */" block.
		{
			name:  "dart line comment",
			lang:  LangDart,
			input: "// comment\nint x = 1;\n",
			want:  LineCounts{Blank: 0, Comment: 1, Code: 1},
		},
		{
			name:  "dart block comment",
			lang:  LangDart,
			input: "/* open\n * doc\n */\nvoid f() {}\n",
			want:  LineCounts{Blank: 0, Comment: 3, Code: 1},
		},
		// SQL — ANSI "--" line + "/* */" block.
		{
			name:  "sql line comment",
			lang:  LangSQL,
			input: "-- comment\nSELECT 1;\n",
			want:  LineCounts{Blank: 0, Comment: 1, Code: 1},
		},
		{
			name:  "sql block comment",
			lang:  LangSQL,
			input: "/* open\n * multi\n */\nSELECT 2;\n",
			want:  LineCounts{Blank: 0, Comment: 3, Code: 1},
		},
		// Lua — "--" line + "--[[" / "]]" long-bracket block.
		// Acceptance #5: a line "--[[ comment ]]" is classified as Comment.
		// Known limitation (Policy α, YAGNI): "]]" also appears as a table-index
		// operator; such lines are mis-classified as Comment.
		{
			name:  "lua line comment",
			lang:  LangLua,
			input: "-- comment\nlocal x = 1\n",
			want:  LineCounts{Blank: 0, Comment: 1, Code: 1},
		},
		{
			name:  "lua block comment single-line (Acceptance #5)",
			lang:  LangLua,
			input: "--[[ comment ]]\nlocal y = 2\n",
			want:  LineCounts{Blank: 0, Comment: 1, Code: 1},
		},
		{
			name:  "lua block comment multi-line",
			lang:  LangLua,
			input: "--[[\nline two\n]]\nlocal z = 3\n",
			want:  LineCounts{Blank: 0, Comment: 3, Code: 1},
		},
		// Elixir — "#" line only (no block-comment form).
		{
			name:  "elixir line comment",
			lang:  LangElixir,
			input: "# comment\nx = 1\n",
			want:  LineCounts{Blank: 0, Comment: 1, Code: 1},
		},
		{
			name:  "elixir code only",
			lang:  LangElixir,
			input: "defmodule Foo do\nend\n",
			want:  LineCounts{Blank: 0, Comment: 0, Code: 2},
		},
		// Zig — "//" line only (no block-comment form).
		// "////" doc comments use the same "//" prefix and are detected.
		{
			name:  "zig line comment",
			lang:  LangZig,
			input: "// comment\nconst x = 1;\n",
			want:  LineCounts{Blank: 0, Comment: 1, Code: 1},
		},
		{
			name:  "zig doc comment uses same prefix",
			lang:  LangZig,
			input: "/// doc comment\npub fn f() void {}\n",
			want:  LineCounts{Blank: 0, Comment: 1, Code: 1},
		},
		// R — "#" line only (no block-comment form).
		{
			name:  "r line comment",
			lang:  LangR,
			input: "# comment\nx <- 1\n",
			want:  LineCounts{Blank: 0, Comment: 1, Code: 1},
		},
		// F# — "//" line + "(* *)" ML-style block.
		{
			name:  "fsharp line comment",
			lang:  LangFSharp,
			input: "// comment\nlet x = 1\n",
			want:  LineCounts{Blank: 0, Comment: 1, Code: 1},
		},
		{
			name:  "fsharp block comment",
			lang:  LangFSharp,
			input: "(* open\n * body\n *)\nlet y = 2\n",
			want:  LineCounts{Blank: 0, Comment: 3, Code: 1},
		},
		// Haskell — "--" line + "{- -}" block.
		{
			name:  "haskell line comment",
			lang:  LangHaskell,
			input: "-- comment\nx = 1\n",
			want:  LineCounts{Blank: 0, Comment: 1, Code: 1},
		},
		{
			name:  "haskell block comment",
			lang:  LangHaskell,
			input: "{- open\n   body\n-}\nx = 2\n",
			want:  LineCounts{Blank: 0, Comment: 3, Code: 1},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := Split(strings.NewReader(tc.input), tc.lang)
			if err != nil {
				t.Fatalf("Split: unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %+v; want %+v", got, tc.want)
			}
		})
	}
}

// TestSplit_Templating verifies Unit A.3: blank/comment/code classification
// for the 12 new templating and frontend language grammars. Tests cover at
// minimum: one Vue <!-- --> comment, one Jinja {# #} comment, one Mustache
// {{!-- --}} block comment, one JSX /* */ block comment, one ERB <%# note %>
// mid-line occurrence (verifies block form catches it), and one ERB <%= value %>
// line (verifies the Policy α known limitation: %> on expression lines is
// treated as a block-close, mis-classifying code lines as Comment).
//
// Vue/Svelte sub-parsing limitation: LangVue and LangSvelte use HTML-level
// <!-- --> grammar. JS/TS comments inside <script> blocks are invisible to
// rak's grammar and classify as Code. One file = one grammar per design
// principle 2 (out of scope for v0.2.0).
//
// Templ HTML-comment limitation: LangTempl uses Go-style // and /* */ grammar.
// HTML-like <!-- --> comments inside .templ files classify as Code. Same
// single-grammar policy.
//
// Sass Policy α YAGNI: LangSass is assigned // + /* */ grammar. Indented Sass
// uses // for real; /* */ block comments exist but are less common. Some
// non-comment lines may be over-classified. Acceptable for v0.2.0.
func TestSplit_Templating(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		lang  Language
		input string
		want  LineCounts
	}{
		// LangTempl — Go-style // line + /* */ block.
		{
			name:  "templ line comment",
			lang:  LangTempl,
			input: "// comment\nfunc Foo() templ.Component {\n",
			want:  LineCounts{Blank: 0, Comment: 1, Code: 1},
		},
		{
			name:  "templ block comment",
			lang:  LangTempl,
			input: "/* open\n * body\n */\nvar x = 1\n",
			want:  LineCounts{Blank: 0, Comment: 3, Code: 1},
		},
		// LangJSX — JS-family // line + /* */ block.
		{
			name:  "jsx line comment",
			lang:  LangJSX,
			input: "// comment\nconst x = <div/>\n",
			want:  LineCounts{Blank: 0, Comment: 1, Code: 1},
		},
		{
			name:  "jsx block comment (Acceptance #5 analog — /* */)",
			lang:  LangJSX,
			input: "/* open\n * note\n */\nreturn <App/>\n",
			want:  LineCounts{Blank: 0, Comment: 3, Code: 1},
		},
		// LangTSX — TS-family // line + /* */ block.
		{
			name:  "tsx line comment",
			lang:  LangTSX,
			input: "// comment\nexport const App = () => <div/>\n",
			want:  LineCounts{Blank: 0, Comment: 1, Code: 1},
		},
		// LangSCSS — // line + /* */ block.
		{
			name:  "scss line comment",
			lang:  LangSCSS,
			input: "// comment\n.foo { color: red; }\n",
			want:  LineCounts{Blank: 0, Comment: 1, Code: 1},
		},
		{
			name:  "scss block comment",
			lang:  LangSCSS,
			input: "/* open\n * body\n */\n.bar { margin: 0; }\n",
			want:  LineCounts{Blank: 0, Comment: 3, Code: 1},
		},
		// LangSass — // line + /* */ block (Policy α YAGNI; see func comment).
		{
			name:  "sass line comment",
			lang:  LangSass,
			input: "// comment\n.foo\n  color: red\n",
			want:  LineCounts{Blank: 0, Comment: 1, Code: 2},
		},
		// LangLESS — // line + /* */ block.
		{
			name:  "less line comment",
			lang:  LangLESS,
			input: "// comment\n.foo { color: red; }\n",
			want:  LineCounts{Blank: 0, Comment: 1, Code: 1},
		},
		// LangVue — HTML-level <!-- --> block (Acceptance #5).
		// JS/TS comments inside <script> blocks are Code per design principle 2.
		{
			name:  "vue html comment (Acceptance #5)",
			lang:  LangVue,
			input: "<!-- comment -->\n<template>\n  <div/>\n</template>\n",
			want:  LineCounts{Blank: 0, Comment: 1, Code: 3},
		},
		{
			name:  "vue multiline html comment",
			lang:  LangVue,
			input: "<!-- open\n     body\n-->\n<div/>\n",
			want:  LineCounts{Blank: 0, Comment: 3, Code: 1},
		},
		{
			name:  "vue script js comment is Code (sub-parsing out of scope)",
			lang:  LangVue,
			input: "<script>\n// this js comment classifies as Code — single grammar policy\nconst x = 1\n</script>\n",
			want:  LineCounts{Blank: 0, Comment: 0, Code: 4},
		},
		// LangSvelte — HTML-level <!-- --> block (same policy as Vue).
		{
			name:  "svelte html comment",
			lang:  LangSvelte,
			input: "<!-- comment -->\n<script>\nlet x = 1\n</script>\n",
			want:  LineCounts{Blank: 0, Comment: 1, Code: 3},
		},
		// LangERB — block form <%# ... %> (Acceptance #8).
		// blockOpen="<%#", blockClose="%>": Policy α uses strings.Contains so
		// mid-line <%# note %> is detected.
		// Known limitation: %> on expression-output lines like <%= value %> is also
		// treated as blockClose → whole line classified as Comment (Policy α YAGNI).
		{
			name:  "erb comment at line start",
			lang:  LangERB,
			input: "<%# comment %>\n<%= @user.name %>\n",
			// Line 1: contains "<%#" (blockOpen) → Comment.
			// Line 2: contains "%>" (blockClose) → Comment (Policy α known limitation).
			want: LineCounts{Blank: 0, Comment: 2, Code: 0},
		},
		{
			name:  "erb mid-line comment (Acceptance #8 — block form catches it)",
			lang:  LangERB,
			input: "<%= val %> <%# note %>\n",
			// Contains "<%#" (blockOpen) → Comment. Also contains "%>" twice but
			// blockOpen is found first → Comment on the first pass.
			want: LineCounts{Blank: 0, Comment: 1, Code: 0},
		},
		{
			name: "erb expression-output line is Comment (Policy α known limitation)",
			// <%= value %> contains "%>" which is the blockClose for ERB grammar.
			// Under Policy α this line is mis-classified as Comment. This is the
			// accepted trade-off (see PLAN.md ERB grammar trade-off note and Notes §
			// "ERB grammar trade-off"). Document here to lock in the known behavior.
			lang:  LangERB,
			input: "<%= @title %>\n<p>plain html</p>\n",
			// Line 1: "<%= @title %>" contains "%>" → Comment (known limitation).
			// Line 2: "<p>plain html</p>" — no markers → Code.
			want: LineCounts{Blank: 0, Comment: 1, Code: 1},
		},
		// LangJinja — {# ... #} block (Acceptance #6).
		{
			name:  "jinja comment (Acceptance #6)",
			lang:  LangJinja,
			input: "{# comment #}\n{{ variable }}\n",
			// Line 1: contains "{#" (blockOpen) → Comment.
			// Line 2: "{{ variable }}" — no "#}" marker → Code.
			want: LineCounts{Blank: 0, Comment: 1, Code: 1},
		},
		{
			name:  "jinja multiline comment",
			lang:  LangJinja,
			input: "{# open\n   body\n#}\n{{ var }}\n",
			want:  LineCounts{Blank: 0, Comment: 3, Code: 1},
		},
		// LangLiquid — {% comment %} / {% endcomment %} block.
		{
			name:  "liquid comment block",
			lang:  LangLiquid,
			input: "{% comment %}\nThis is hidden.\n{% endcomment %}\n{{ title }}\n",
			// Line 1: contains "{% comment %}" (blockOpen) → Comment; sets inBlockComment.
			// Line 2: "This is hidden." — inBlockComment=true → Comment.
			// Line 3: "{% endcomment %}" — inBlockComment=true at start → Comment; closes block.
			// Line 4: "{{ title }}" — no markers → Code.
			want: LineCounts{Blank: 0, Comment: 3, Code: 1},
		},
		// LangMustache — {{! linePrefix + {{!-- --}} block (Acceptance #7).
		{
			name:  "mustache line comment via linePrefix",
			lang:  LangMustache,
			input: "{{! inline comment }}\n{{name}}\n",
			// Line 1: trimmed starts with "{{!" (linePrefix) → Comment.
			// Line 2: "{{name}}" — no "{{!" prefix, no blockOpen "{{!--" → Code.
			want: LineCounts{Blank: 0, Comment: 1, Code: 1},
		},
		{
			name:  "mustache block comment {{!-- --}} (Acceptance #7)",
			lang:  LangMustache,
			input: "{{!-- comment --}}\n{{name}}\n",
			// Line 1: contains "{{!--" (blockOpen) → Comment.
			// Line 2: "{{name}}" → Code.
			want: LineCounts{Blank: 0, Comment: 1, Code: 1},
		},
		{
			name:  "mustache multiline block comment",
			lang:  LangMustache,
			input: "{{!--\n  multiline\n--}}\n{{name}}\n",
			want:  LineCounts{Blank: 0, Comment: 3, Code: 1},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := Split(strings.NewReader(tc.input), tc.lang)
			if err != nil {
				t.Fatalf("Split: unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %+v; want %+v", got, tc.want)
			}
		})
	}
}

// TestSplit_ConfigDataFormats verifies Unit A.4: blank/comment/code
// classification for the 8 grammar-bearing config/data languages and the
// grammar-absent CSV/TSV/JSONL formats (all non-blank lines = Code).
//
// Grammar-less formats (LangCSV, LangTSV, LangJSONL): absent from grammarTable,
// so Split uses the zero grammar — all non-blank lines classify as Code.
// Acceptance #9 requires one assertion per grammar-less lang.
//
// INI uses ";" as primary line-comment prefix and "#" as secondary.
// Properties uses "#" as primary and "!" as secondary.
// HCL accepts "#", "//", and "/* */".
func TestSplit_ConfigDataFormats(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		lang  Language
		input string
		want  LineCounts
	}{
		// LangINI — ";" primary + "#" secondary (Acceptance #6).
		{
			name:  "ini semicolon comment (Acceptance #6)",
			lang:  LangINI,
			input: "; comment\n[section]\nkey=value\n",
			want:  LineCounts{Blank: 0, Comment: 1, Code: 2},
		},
		{
			name:  "ini hash secondary comment",
			lang:  LangINI,
			input: "# comment\nkey=value\n",
			want:  LineCounts{Blank: 0, Comment: 1, Code: 1},
		},
		// LangEnv — "#" line only.
		{
			name:  "env hash comment",
			lang:  LangEnv,
			input: "# comment\nFOO=bar\n",
			want:  LineCounts{Blank: 0, Comment: 1, Code: 1},
		},
		// LangEditorConfig — "#" line only.
		{
			name:  "editorconfig hash comment",
			lang:  LangEditorConfig,
			input: "# comment\n[*.go]\nindent_size = 4\n",
			want:  LineCounts{Blank: 0, Comment: 1, Code: 2},
		},
		// LangProperties — "#" primary + "!" secondary (Acceptance #8).
		{
			name:  "properties hash comment",
			lang:  LangProperties,
			input: "# comment\nkey=value\n",
			want:  LineCounts{Blank: 0, Comment: 1, Code: 1},
		},
		{
			name:  "properties exclamation secondary comment (Acceptance #8)",
			lang:  LangProperties,
			input: "! comment\nkey=value\n",
			want:  LineCounts{Blank: 0, Comment: 1, Code: 1},
		},
		// LangHCL — "#" primary, "//" secondary, "/* */" block (Acceptance #7).
		{
			name:  "hcl hash comment (Acceptance #7)",
			lang:  LangHCL,
			input: "# comment\nresource \"aws_s3_bucket\" \"b\" {}\n",
			want:  LineCounts{Blank: 0, Comment: 1, Code: 1},
		},
		{
			name:  "hcl slashslash comment (Acceptance #7)",
			lang:  LangHCL,
			input: "// comment\nresource \"aws_s3_bucket\" \"b\" {}\n",
			want:  LineCounts{Blank: 0, Comment: 1, Code: 1},
		},
		{
			name:  "hcl block comment (Acceptance #7)",
			lang:  LangHCL,
			input: "/* open\n * body\n */\nresource \"aws_s3_bucket\" \"b\" {}\n",
			want:  LineCounts{Blank: 0, Comment: 3, Code: 1},
		},
		// LangNix — "#" line + "/* */" block.
		{
			name:  "nix hash comment",
			lang:  LangNix,
			input: "# comment\nlet x = 1; in x\n",
			want:  LineCounts{Blank: 0, Comment: 1, Code: 1},
		},
		{
			name:  "nix block comment",
			lang:  LangNix,
			input: "/* open\n   body\n*/\nlet x = 1; in x\n",
			want:  LineCounts{Blank: 0, Comment: 3, Code: 1},
		},
		// LangProto — "//" line + "/* */" block.
		{
			name:  "proto line comment",
			lang:  LangProto,
			input: "// comment\nmessage Foo {}\n",
			want:  LineCounts{Blank: 0, Comment: 1, Code: 1},
		},
		{
			name:  "proto block comment",
			lang:  LangProto,
			input: "/* open\n * doc\n */\nmessage Foo {}\n",
			want:  LineCounts{Blank: 0, Comment: 3, Code: 1},
		},
		// LangGraphQL — "#" line only.
		{
			name:  "graphql hash comment",
			lang:  LangGraphQL,
			input: "# comment\ntype Query { hello: String }\n",
			want:  LineCounts{Blank: 0, Comment: 1, Code: 1},
		},
		// Grammar-absent formats: all non-blank lines = Code (Acceptance #9).
		// LangCSV: no comment syntax; all lines are Code.
		{
			name:  "csv all code (Acceptance #9)",
			lang:  LangCSV,
			input: "a,b,c\n1,2,3\n\n",
			want:  LineCounts{Blank: 1, Comment: 0, Code: 2},
		},
		// LangTSV: no comment syntax; all lines are Code.
		{
			name:  "tsv all code (Acceptance #9)",
			lang:  LangTSV,
			input: "a\tb\tc\n1\t2\t3\n",
			want:  LineCounts{Blank: 0, Comment: 0, Code: 2},
		},
		// LangJSONL: no comment syntax (JSON Lines); all lines are Code.
		{
			name:  `jsonl all code (Acceptance #9)`,
			lang:  LangJSONL,
			input: "{\"key\":\"value\"}\n{\"a\":1}\n",
			want:  LineCounts{Blank: 0, Comment: 0, Code: 2},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := Split(strings.NewReader(tc.input), tc.lang)
			if err != nil {
				t.Fatalf("Split: unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %+v; want %+v", got, tc.want)
			}
		})
	}
}

// TestSplit_Swift verifies that Swift uses "//" for line comments and
// "/* */" for block comments. Nested block comments are not tracked (Policy α,
// YAGNI v0.1.0): the flat open/close scan is acceptable.
func TestSplit_Swift(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input string
		want  LineCounts
	}{
		{
			name:  "line comment",
			input: "// comment\nlet x = 1\n",
			want:  LineCounts{Blank: 0, Comment: 1, Code: 1},
		},
		{
			name:  "block comment",
			input: "/* open\n * doc\n */\nvar y = 2\n",
			want:  LineCounts{Blank: 0, Comment: 3, Code: 1},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := Split(strings.NewReader(tc.input), LangSwift)
			if err != nil {
				t.Fatalf("Split: unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %+v; want %+v", got, tc.want)
			}
		})
	}
}
