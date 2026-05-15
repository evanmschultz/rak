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
