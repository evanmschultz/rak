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
