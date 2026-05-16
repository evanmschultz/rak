package render

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/evanmschultz/laslig"

	"github.com/evanmschultz/rak/internal/counting"
	"github.com/evanmschultz/rak/internal/lang"
	"github.com/evanmschultz/rak/internal/summary"
)

// testHumanMode is the explicit laslig.Mode used for snapshot determinism.
// FormatPlain + Styled:false + Width:80 bypasses laslig.ResolveMode's
// environment inspection entirely, so test output is independent of
// $COLUMNS, $TERM, $NO_COLOR, $CI. Pinning these three fields is a
// Unit 2.2 acceptance invariant (F3 pin).
var testHumanMode = laslig.Mode{
	Format: laslig.FormatPlain,
	Styled: false,
	Width:  80,
}

// TestHumanRenderer_SnapshotPlain pins the laslig KV block shape for the
// mid-size Counts case. The captured string is observed output from the
// v0.2.4 plain-mode KV formatter; a bump past v0.2.4 that changes this
// string is a deliberate snapshot break.
func TestHumanRenderer_SnapshotPlain(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	r := newHumanRendererWithMode(testHumanMode)
	if err := r.Render(&buf, counting.Counts{Bytes: 12, Lines: 1, Words: 2, Chars: 12}); err != nil {
		t.Fatalf("render: %v", err)
	}
	got := buf.String()
	want := "\n  Bytes  12\n  Lines  1\n  Words  2\n  Chars  12\n"
	if got != want {
		t.Fatalf("snapshot mismatch\nwant: %q\ngot:  %q", want, got)
	}
}

// TestHumanRenderer_TablePlain walks the zero / small / large Counts cases
// against the plain-mode laslig KV block.
func TestHumanRenderer_TablePlain(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		counts counting.Counts
		want   string
	}{
		{
			name:   "zero",
			counts: counting.Counts{},
			want:   "\n  Bytes  0\n  Lines  0\n  Words  0\n  Chars  0\n",
		},
		{
			name:   "small",
			counts: counting.Counts{Bytes: 12, Lines: 1, Words: 2, Chars: 12},
			want:   "\n  Bytes  12\n  Lines  1\n  Words  2\n  Chars  12\n",
		},
		{
			name:   "large",
			counts: counting.Counts{Bytes: 1_000_000_000, Lines: 42_000, Words: 190_000, Chars: 950_000_000},
			want:   "\n  Bytes  1000000000\n  Lines  42000\n  Words  190000\n  Chars  950000000\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			r := newHumanRendererWithMode(testHumanMode)
			if err := r.Render(&buf, tc.counts); err != nil {
				t.Fatalf("render: %v", err)
			}
			if got := buf.String(); got != tc.want {
				t.Fatalf("snapshot mismatch\nwant: %q\ngot:  %q", tc.want, got)
			}
		})
	}
}

// TestJSONRenderer_Snapshot pins the stdlib encoding/json output shape for
// the mid-size Counts case. Field order (Bytes, Lines, Words, Chars) mirrors
// the counting.Counts declaration order per Unit 2.1's F4 contract (no
// json struct tags). json.Encoder.Encode trails with '\n'.
func TestJSONRenderer_Snapshot(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	r := NewJSONRenderer()
	if err := r.Render(&buf, counting.Counts{Bytes: 12, Lines: 1, Words: 2, Chars: 12}); err != nil {
		t.Fatalf("render: %v", err)
	}
	got := buf.String()
	want := `{"Bytes":12,"Lines":1,"Words":2,"Chars":12}` + "\n"
	if got != want {
		t.Fatalf("snapshot mismatch\nwant: %q\ngot:  %q", want, got)
	}
}

// TestJSONRenderer_Table walks the zero / small / large Counts cases
// against the stdlib encoding/json output.
func TestJSONRenderer_Table(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		counts counting.Counts
		want   string
	}{
		{
			name:   "zero",
			counts: counting.Counts{},
			want:   `{"Bytes":0,"Lines":0,"Words":0,"Chars":0}` + "\n",
		},
		{
			name:   "small",
			counts: counting.Counts{Bytes: 12, Lines: 1, Words: 2, Chars: 12},
			want:   `{"Bytes":12,"Lines":1,"Words":2,"Chars":12}` + "\n",
		},
		{
			name:   "large",
			counts: counting.Counts{Bytes: 1_000_000_000, Lines: 42_000, Words: 190_000, Chars: 950_000_000},
			want:   `{"Bytes":1000000000,"Lines":42000,"Words":190000,"Chars":950000000}` + "\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			r := NewJSONRenderer()
			if err := r.Render(&buf, tc.counts); err != nil {
				t.Fatalf("render: %v", err)
			}
			if got := buf.String(); got != tc.want {
				t.Fatalf("snapshot mismatch\nwant: %q\ngot:  %q", tc.want, got)
			}
		})
	}
}

// TestHumanRenderer_RenderTree_Labels verifies the laslig KV output for a
// multi-directory rollup carries:
//   - one "dir: <path>" title per supplied Directory, in caller order
//   - a final "total" block
//   - the four canonical count labels (Bytes, Lines, Words, Chars)
//   - the numeric values reported for each directory plus the grand total
//
// Substring assertions (rather than byte-exact snapshot) because KV Title
// layout with multiple blocks is more sensitive to laslig formatter changes
// than the titleless single-block case; the existing TablePlain snapshot
// already pins the inner KV shape. This test's job is to prove RenderTree
// composes the blocks in the right order with the right labels.
func TestHumanRenderer_RenderTree_Labels(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	r := newHumanRendererWithMode(testHumanMode)
	s := summary.Summary{
		Dirs: []summary.Directory{
			{Path: ".", Counts: counting.Counts{Bytes: 12, Lines: 1, Words: 2, Chars: 12}},
			{Path: "sub", Counts: counting.Counts{Bytes: 4, Lines: 1, Words: 1, Chars: 4}},
		},
		Total: counting.Counts{Bytes: 16, Lines: 2, Words: 3, Chars: 16},
	}
	if err := r.RenderTree(&buf, s, nil); err != nil {
		t.Fatalf("RenderTree: %v", err)
	}

	got := buf.String()
	for _, want := range []string{
		"dir: .",
		"dir: sub",
		"total",
		"Bytes", "Lines", "Words", "Chars",
		"12", "4", "16",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("RenderTree output missing %q; got:\n%s", want, got)
		}
	}

	// Block ordering: "dir: ." must precede "dir: sub" must precede "total".
	idxRoot := strings.Index(got, "dir: .")
	idxSub := strings.Index(got, "dir: sub")
	idxTotal := strings.Index(got, "total")
	if idxRoot < 0 || idxSub < 0 || idxTotal < 0 {
		t.Fatalf("missing required titles; got:\n%s", got)
	}
	if idxRoot >= idxSub || idxSub >= idxTotal {
		t.Errorf("block order wrong: root=%d sub=%d total=%d; got:\n%s", idxRoot, idxSub, idxTotal, got)
	}
}

// TestHumanRenderer_RenderTree_NoErrors verifies an empty errs slice does
// NOT emit a warning section.
func TestHumanRenderer_RenderTree_NoErrors(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	r := newHumanRendererWithMode(testHumanMode)
	s := summary.Summary{
		Dirs:  []summary.Directory{{Path: ".", Counts: counting.Counts{Bytes: 1, Lines: 0, Words: 0, Chars: 1}}},
		Total: counting.Counts{Bytes: 1, Lines: 0, Words: 0, Chars: 1},
	}
	if err := r.RenderTree(&buf, s, nil); err != nil {
		t.Fatalf("RenderTree: %v", err)
	}
	got := buf.String()
	if strings.Contains(got, "WARNING") {
		t.Errorf("no-errors case should not emit a WARNING notice; got:\n%s", got)
	}
	if strings.Contains(strings.ToLower(got), "errors") {
		t.Errorf("no-errors case should not mention errors; got:\n%s", got)
	}
}

// TestHumanRenderer_RenderTree_WithErrors verifies a non-empty errs slice
// emits a WARNING-level Notice whose detail includes each error's message.
func TestHumanRenderer_RenderTree_WithErrors(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	r := newHumanRendererWithMode(testHumanMode)
	s := summary.Summary{
		Dirs:  []summary.Directory{{Path: ".", Counts: counting.Counts{Bytes: 1, Lines: 0, Words: 0, Chars: 1}}},
		Total: counting.Counts{Bytes: 1, Lines: 0, Words: 0, Chars: 1},
	}
	errs := []error{errors.New("walk \"foo\": permission denied"), errors.New("walk \"bar\": not a directory")}
	if err := r.RenderTree(&buf, s, errs); err != nil {
		t.Fatalf("RenderTree: %v", err)
	}
	got := buf.String()
	for _, want := range []string{"WARNING", "Errors", "permission denied", "not a directory"} {
		if !strings.Contains(got, want) {
			t.Errorf("RenderTree output missing %q; got:\n%s", want, got)
		}
	}
}

// TestHumanRenderer_RenderTree_EmptyDirs verifies passing zero directories
// still emits a "total" block — the empty-directory user case should report
// zeroed totals rather than blank output.
func TestHumanRenderer_RenderTree_EmptyDirs(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	r := newHumanRendererWithMode(testHumanMode)
	if err := r.RenderTree(&buf, summary.Summary{}, nil); err != nil {
		t.Fatalf("RenderTree: %v", err)
	}
	got := buf.String()
	for _, want := range []string{"total", "Bytes", "Lines", "Words", "Chars", "0"} {
		if !strings.Contains(got, want) {
			t.Errorf("empty-dirs RenderTree missing %q; got:\n%s", want, got)
		}
	}
	if strings.Contains(got, "dir:") {
		t.Errorf("empty-dirs RenderTree should not emit any dir: blocks; got:\n%s", got)
	}
}

// TestJSONRenderer_RenderTree_Snapshot pins the JSON envelope shape for a
// multi-directory rollup. Unlike the human path, JSON output is fully
// deterministic so byte-exact matching is appropriate.
func TestJSONRenderer_RenderTree_Snapshot(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	r := NewJSONRenderer()
	s := summary.Summary{
		Dirs: []summary.Directory{
			{Path: ".", Counts: counting.Counts{Bytes: 12, Lines: 1, Words: 2, Chars: 12}},
			{Path: "sub", Counts: counting.Counts{Bytes: 4, Lines: 1, Words: 1, Chars: 4}},
		},
		Total: counting.Counts{Bytes: 16, Lines: 2, Words: 3, Chars: 16},
	}
	if err := r.RenderTree(&buf, s, nil); err != nil {
		t.Fatalf("RenderTree: %v", err)
	}
	got := buf.String()
	want := `{"directories":[` +
		`{"path":".","counts":{"Bytes":12,"Lines":1,"Words":2,"Chars":12}},` +
		`{"path":"sub","counts":{"Bytes":4,"Lines":1,"Words":1,"Chars":4}}` +
		`],"total":{"Bytes":16,"Lines":2,"Words":3,"Chars":16}}` + "\n"
	if got != want {
		t.Fatalf("snapshot mismatch\nwant: %q\ngot:  %q", want, got)
	}
}

// TestJSONRenderer_RenderTree_Empty verifies the no-directories case emits
// a well-formed envelope with an empty directories array and zero totals.
// The directories field is always present (not omitted) so JSON consumers
// can rely on its presence.
func TestJSONRenderer_RenderTree_Empty(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	r := NewJSONRenderer()
	if err := r.RenderTree(&buf, summary.Summary{}, nil); err != nil {
		t.Fatalf("RenderTree: %v", err)
	}
	got := buf.String()
	want := `{"directories":[],"total":{"Bytes":0,"Lines":0,"Words":0,"Chars":0}}` + "\n"
	if got != want {
		t.Fatalf("snapshot mismatch\nwant: %q\ngot:  %q", want, got)
	}
}

// Compile-time interface assertions — verify all three renderers satisfy Renderer.
var (
	_ Renderer = humanRenderer{}
	_ Renderer = jsonRenderer{}
	_ Renderer = toonRenderer{}
)

// TestJSONRenderer_RenderTree_WithErrors verifies the errors key is emitted
// only when errs is non-empty, and carries each error's Error() string.
func TestJSONRenderer_RenderTree_WithErrors(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	r := NewJSONRenderer()
	s := summary.Summary{
		Dirs:  []summary.Directory{{Path: ".", Counts: counting.Counts{Bytes: 1, Lines: 0, Words: 0, Chars: 1}}},
		Total: counting.Counts{Bytes: 1, Lines: 0, Words: 0, Chars: 1},
	}
	errs := []error{errors.New("walk \"foo\": permission denied")}
	if err := r.RenderTree(&buf, s, errs); err != nil {
		t.Fatalf("RenderTree: %v", err)
	}
	got := buf.String()

	// Parse back and verify structure: the exact field order of encoding/json
	// is declaration-order, so a byte-match is safe, but we also want to
	// verify the errors array content.
	var parsed struct {
		Directories []directoryJSON `json:"directories"`
		Total       counting.Counts `json:"total"`
		Errors      []string        `json:"errors"`
	}
	if err := json.Unmarshal([]byte(got), &parsed); err != nil {
		t.Fatalf("json.Unmarshal: %v (body: %s)", err, got)
	}
	if len(parsed.Errors) != 1 {
		t.Fatalf("expected 1 error, got %d: %v", len(parsed.Errors), parsed.Errors)
	}
	if !strings.Contains(parsed.Errors[0], "permission denied") {
		t.Errorf("error message missing expected text; got: %q", parsed.Errors[0])
	}

	// Byte-exact to pin field order (directories → total → errors).
	want := `{"directories":[{"path":".","counts":{"Bytes":1,"Lines":0,"Words":0,"Chars":1}}],` +
		`"total":{"Bytes":1,"Lines":0,"Words":0,"Chars":1},` +
		`"errors":["walk \"foo\": permission denied"]}` + "\n"
	if got != want {
		t.Fatalf("snapshot mismatch\nwant: %q\ngot:  %q", want, got)
	}
}

// TestTOONRenderer_Render verifies a single counting.Counts value marshals to
// TOON output containing the expected key-value lines. Substring assertions
// are used (not byte-exact snapshot) so the test is robust against toon-go
// formatting tweaks across patch versions.
func TestTOONRenderer_Render(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	r := NewTOONRenderer()
	if err := r.Render(&buf, counting.Counts{Bytes: 12, Lines: 2, Words: 2, Chars: 12}); err != nil {
		t.Fatalf("Render: %v", err)
	}
	got := buf.String()
	for _, want := range []string{"bytes: 12", "lines: 2", "words: 2", "chars: 12"} {
		if !strings.Contains(got, want) {
			t.Errorf("Render output missing %q; got:\n%s", want, got)
		}
	}
	if !strings.HasSuffix(got, "\n") {
		t.Errorf("TOON Render output should end with newline, got %q", got)
	}
}

// TestTOONRenderer_RenderTree verifies a multi-directory rollup contains the
// "directories" array key, the "total" nested block, and both directory paths
// in their pipe-delimited tabular column context.
//
// ".|" pins the "." path as the first column of a pipe-delimited tabular row
// (not just any incidental dot in the output). "total" verifies the nested
// grand-total block is present (F20 nested-total contract).
func TestTOONRenderer_RenderTree(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	r := NewTOONRenderer()
	s := summary.Summary{
		Dirs: []summary.Directory{
			{Path: ".", Counts: counting.Counts{Bytes: 5, Lines: 1, Words: 1, Chars: 5}},
			{Path: "sub", Counts: counting.Counts{Bytes: 3, Lines: 1, Words: 1, Chars: 3}},
		},
		Total: counting.Counts{Bytes: 8, Lines: 2, Words: 2, Chars: 8},
	}
	if err := r.RenderTree(&buf, s, nil); err != nil {
		t.Fatalf("RenderTree: %v", err)
	}
	got := buf.String()
	for _, want := range []string{"directories", ".|", "sub", "total"} {
		if !strings.Contains(got, want) {
			t.Errorf("RenderTree output missing %q; got:\n%s", want, got)
		}
	}
	if !strings.HasSuffix(got, "\n") {
		t.Errorf("TOON RenderTree output should end with newline, got %q", got)
	}
}

// TestTOONRenderer_RenderTree_WithErrors verifies that a non-empty errs slice
// causes the output to contain an "errors" field.
func TestTOONRenderer_RenderTree_WithErrors(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	r := NewTOONRenderer()
	s := summary.Summary{
		Dirs:  []summary.Directory{{Path: ".", Counts: counting.Counts{Bytes: 1, Lines: 0, Words: 0, Chars: 1}}},
		Total: counting.Counts{Bytes: 1, Lines: 0, Words: 0, Chars: 1},
	}
	errs := []error{errors.New("walk \"foo\": permission denied")}
	if err := r.RenderTree(&buf, s, errs); err != nil {
		t.Fatalf("RenderTree: %v", err)
	}
	got := buf.String()
	if !strings.Contains(got, "errors") {
		t.Errorf("RenderTree with errors missing \"errors\" key; got:\n%s", got)
	}
}

// TestTOONRenderer_RenderTree_NoErrors verifies that a nil/empty errs slice
// does NOT emit an "errors" key in the output.
func TestTOONRenderer_RenderTree_NoErrors(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	r := NewTOONRenderer()
	s := summary.Summary{
		Dirs:  []summary.Directory{{Path: ".", Counts: counting.Counts{Bytes: 1, Lines: 0, Words: 0, Chars: 1}}},
		Total: counting.Counts{Bytes: 1, Lines: 0, Words: 0, Chars: 1},
	}
	if err := r.RenderTree(&buf, s, nil); err != nil {
		t.Fatalf("RenderTree: %v", err)
	}
	got := buf.String()
	if strings.Contains(got, "errors") {
		t.Errorf("RenderTree no-errors case must not emit \"errors\" key; got:\n%s", got)
	}
}

// TestTOONRenderer_RenderTree_PerLang verifies that a non-empty ByLang map
// causes per-language detail to appear in TOON output (F33 compliant — only
// known languages, LangUnknown suppressed).
func TestTOONRenderer_RenderTree_PerLang(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	r := NewTOONRenderer()
	s := summary.Summary{
		Dirs: []summary.Directory{
			{
				Path:   ".",
				Counts: counting.Counts{Bytes: 26, Lines: 2, Words: 2, Chars: 26},
				ByLang: map[lang.Language]lang.LangCounts{
					lang.LangGo: {
						Lines:  lang.LineCounts{Blank: 0, Comment: 0, Code: 1},
						Counts: counting.Counts{Bytes: 13, Lines: 1, Words: 1, Chars: 13},
					},
					lang.LangRust: {
						Lines:  lang.LineCounts{Blank: 0, Comment: 0, Code: 1},
						Counts: counting.Counts{Bytes: 13, Lines: 1, Words: 1, Chars: 13},
					},
				},
			},
		},
		Total: counting.Counts{Bytes: 26, Lines: 2, Words: 2, Chars: 26},
	}
	if err := r.RenderTree(&buf, s, nil); err != nil {
		t.Fatalf("RenderTree: %v", err)
	}
	got := buf.String()
	// Per-language detail must appear somewhere in the output.
	if !strings.Contains(got, "go") {
		t.Errorf("TOON per-lang output missing \"go\"; got:\n%s", got)
	}
	if !strings.Contains(got, "rust") {
		t.Errorf("TOON per-lang output missing \"rust\"; got:\n%s", got)
	}
}

// TestTOONRenderer_RenderTree_AllUnknown verifies that when ByLang contains
// only LangUnknown, the output does NOT contain an "unknown" or "" language
// key (F33 suppression).
func TestTOONRenderer_RenderTree_AllUnknown(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	r := NewTOONRenderer()
	s := summary.Summary{
		Dirs: []summary.Directory{
			{
				Path:   ".",
				Counts: counting.Counts{Bytes: 12, Lines: 1, Words: 2, Chars: 12},
				ByLang: map[lang.Language]lang.LangCounts{
					lang.LangUnknown: {
						Lines:  lang.LineCounts{Blank: 0, Comment: 0, Code: 1},
						Counts: counting.Counts{Bytes: 12, Lines: 1, Words: 2, Chars: 12},
					},
				},
			},
		},
		Total: counting.Counts{Bytes: 12, Lines: 1, Words: 2, Chars: 12},
	}
	if err := r.RenderTree(&buf, s, nil); err != nil {
		t.Fatalf("RenderTree: %v", err)
	}
	got := buf.String()
	// "unknown" and "" as language keys must not appear after F33 suppression.
	if strings.Contains(got, "by_lang") && strings.Contains(got, "unknown") {
		t.Errorf("TOON all-unknown ByLang should suppress unknown key; got:\n%s", got)
	}
	// The by_lang block should be absent entirely when all entries are unknown.
	if strings.Contains(got, "by_lang") {
		// Acceptable only if no language keys appear. Verify no language key present.
		if strings.Contains(got, ": ") {
			// by_lang is present with content — check it's not unknown
			if strings.Contains(got, `""`+":") || strings.Contains(got, "unknown:") {
				t.Errorf("TOON all-unknown: language key for unknown must be suppressed; got:\n%s", got)
			}
		}
	}
}

// TestJSONRenderer_RenderTree_PerLang verifies that a non-empty ByLang map
// causes a by_lang field to appear in JSON output with per-language detail
// (F33 compliant — LangUnknown suppressed via omitempty).
func TestJSONRenderer_RenderTree_PerLang(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	r := NewJSONRenderer()
	s := summary.Summary{
		Dirs: []summary.Directory{
			{
				Path:   ".",
				Counts: counting.Counts{Bytes: 26, Lines: 2, Words: 2, Chars: 26},
				ByLang: map[lang.Language]lang.LangCounts{
					lang.LangGo: {
						Lines:  lang.LineCounts{Blank: 0, Comment: 0, Code: 1},
						Counts: counting.Counts{Bytes: 13, Lines: 1, Words: 1, Chars: 13},
					},
					lang.LangRust: {
						Lines:  lang.LineCounts{Blank: 0, Comment: 0, Code: 1},
						Counts: counting.Counts{Bytes: 13, Lines: 1, Words: 1, Chars: 13},
					},
				},
			},
		},
		Total: counting.Counts{Bytes: 26, Lines: 2, Words: 2, Chars: 26},
	}
	if err := r.RenderTree(&buf, s, nil); err != nil {
		t.Fatalf("RenderTree: %v", err)
	}
	got := buf.String()
	if !strings.Contains(got, "by_lang") {
		t.Errorf("JSON per-lang output missing \"by_lang\"; got:\n%s", got)
	}
	if !strings.Contains(got, `"go"`) {
		t.Errorf("JSON per-lang output missing go key; got:\n%s", got)
	}
	if !strings.Contains(got, `"rust"`) {
		t.Errorf("JSON per-lang output missing rust key; got:\n%s", got)
	}
}

// TestJSONRenderer_RenderTree_AllUnknown verifies that when ByLang contains
// only LangUnknown, the by_lang field is absent from JSON output (F33).
func TestJSONRenderer_RenderTree_AllUnknown(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	r := NewJSONRenderer()
	s := summary.Summary{
		Dirs: []summary.Directory{
			{
				Path:   ".",
				Counts: counting.Counts{Bytes: 12, Lines: 1, Words: 2, Chars: 12},
				ByLang: map[lang.Language]lang.LangCounts{
					lang.LangUnknown: {
						Lines:  lang.LineCounts{Blank: 0, Comment: 0, Code: 1},
						Counts: counting.Counts{Bytes: 12, Lines: 1, Words: 2, Chars: 12},
					},
				},
			},
		},
		Total: counting.Counts{Bytes: 12, Lines: 1, Words: 2, Chars: 12},
	}
	if err := r.RenderTree(&buf, s, nil); err != nil {
		t.Fatalf("RenderTree: %v", err)
	}
	got := buf.String()
	// by_lang must be absent (omitempty on empty map after LangUnknown filter).
	if strings.Contains(got, "by_lang") {
		t.Errorf("JSON all-unknown ByLang should suppress by_lang field entirely; got:\n%s", got)
	}
	// Explicit unknown key must not appear.
	if strings.Contains(got, `""`) {
		t.Errorf("JSON all-unknown ByLang must not emit empty-string language key; got:\n%s", got)
	}
}

// TestHumanRenderer_RenderTree_PerLang verifies that a non-empty ByLang map
// causes per-language KV rows to appear in human output (F33 compliant).
func TestHumanRenderer_RenderTree_PerLang(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	r := newHumanRendererWithMode(testHumanMode)
	s := summary.Summary{
		Dirs: []summary.Directory{
			{
				Path:   ".",
				Counts: counting.Counts{Bytes: 26, Lines: 2, Words: 2, Chars: 26},
				ByLang: map[lang.Language]lang.LangCounts{
					lang.LangGo: {
						Lines:  lang.LineCounts{Blank: 0, Comment: 0, Code: 1},
						Counts: counting.Counts{Bytes: 13, Lines: 1, Words: 1, Chars: 13},
					},
					lang.LangPython: {
						Lines:  lang.LineCounts{Blank: 1, Comment: 0, Code: 0},
						Counts: counting.Counts{Bytes: 13, Lines: 1, Words: 1, Chars: 13},
					},
				},
			},
		},
		Total: counting.Counts{Bytes: 26, Lines: 2, Words: 2, Chars: 26},
	}
	if err := r.RenderTree(&buf, s, nil); err != nil {
		t.Fatalf("RenderTree: %v", err)
	}
	got := buf.String()
	// Both language names must appear somewhere in the per-lang section.
	if !strings.Contains(got, "go") {
		t.Errorf("human per-lang output missing \"go\"; got:\n%s", got)
	}
	if !strings.Contains(got, "python") {
		t.Errorf("human per-lang output missing \"python\"; got:\n%s", got)
	}
}

// --- Unit 9.6 DirectoriesFilesColumn tests ---

// dirFilesFixture returns a summary.Summary with two directories at distinct
// Files counts (3 and 5) so tests can assert presence and correct values. The
// second directory has Files=0 so JSON omitempty behavior is testable too.
func dirFilesFixture() summary.Summary {
	return summary.Summary{
		Dirs: []summary.Directory{
			{
				Path:   "alpha",
				Counts: counting.Counts{Bytes: 300, Lines: 10, Words: 20, Chars: 300},
				Files:  3,
			},
			{
				Path:   "beta",
				Counts: counting.Counts{Bytes: 500, Lines: 20, Words: 40, Chars: 500},
				Files:  5,
			},
			{
				Path:   "gamma",
				Counts: counting.Counts{Bytes: 100, Lines: 5, Words: 10, Chars: 100},
				Files:  0, // zero — JSON omitempty suppresses "files" key for this dir
			},
		},
		Total: counting.Counts{Bytes: 900, Lines: 35, Words: 70, Chars: 900},
	}
}

// TestRenderer_DirectoriesFilesColumn_TOON verifies that the TOON directories
// array header contains "files" between "path" and "bytes", and that each row
// carries the expected file count as the second pipe-delimited column.
func TestRenderer_DirectoriesFilesColumn_TOON(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	r := NewTOONRenderer()
	if err := r.RenderTree(&buf, dirFilesFixture(), nil); err != nil {
		t.Fatalf("RenderTree: %v", err)
	}
	got := buf.String()

	// Header must contain "files" between "path" and "bytes".
	// toon-go emits tabular headers as {path|files|bytes|lines|words|chars}:
	if !strings.Contains(got, "files") {
		t.Errorf("TOON directories: output missing \"files\" column; got:\n%s", got)
	}
	// Verify column ordering: "path" appears before "files" before "bytes" in the header.
	idxPath := strings.Index(got, "path")
	idxFiles := strings.Index(got, "files")
	idxBytes := strings.Index(got, "bytes")
	if idxPath < 0 || idxFiles < 0 || idxBytes < 0 {
		t.Fatalf("TOON header missing path/files/bytes; got:\n%s", got)
	}
	if idxPath >= idxFiles || idxFiles >= idxBytes {
		t.Errorf("TOON column order wrong: path=%d files=%d bytes=%d; got:\n%s", idxPath, idxFiles, idxBytes, got)
	}

	// Each row with Files>0 must contain the expected value as a pipe-delimited field.
	// alpha has Files=3: row starts "alpha|3|..."
	if !strings.Contains(got, "alpha|3|") {
		t.Errorf("TOON row for alpha: expected Files=3 column, missing \"alpha|3|\"; got:\n%s", got)
	}
	// beta has Files=5: row starts "beta|5|..."
	if !strings.Contains(got, "beta|5|") {
		t.Errorf("TOON row for beta: expected Files=5 column, missing \"beta|5|\"; got:\n%s", got)
	}
	// gamma has Files=0: row starts "gamma|0|..."
	if !strings.Contains(got, "gamma|0|") {
		t.Errorf("TOON row for gamma: expected Files=0 column, missing \"gamma|0|\"; got:\n%s", got)
	}
}

// TestRenderer_DirectoriesFilesColumn_JSON verifies that JSON output carries
// "files" for directories where Files>0, and omits it (omitempty) where Files==0.
func TestRenderer_DirectoriesFilesColumn_JSON(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	r := NewJSONRenderer()
	if err := r.RenderTree(&buf, dirFilesFixture(), nil); err != nil {
		t.Fatalf("RenderTree: %v", err)
	}
	got := buf.String()

	// Parse top-level to inspect per-directory files fields.
	var parsed struct {
		Directories []struct {
			Path   string          `json:"path"`
			Files  *int64          `json:"files"`
			Counts counting.Counts `json:"counts"`
		} `json:"directories"`
		Total counting.Counts `json:"total"`
	}
	if err := json.Unmarshal([]byte(got), &parsed); err != nil {
		t.Fatalf("json.Unmarshal: %v (body: %s)", err, got)
	}
	if len(parsed.Directories) != 3 {
		t.Fatalf("expected 3 directories, got %d", len(parsed.Directories))
	}

	// alpha: Files=3 — must be present with value 3.
	if parsed.Directories[0].Path != "alpha" {
		t.Fatalf("directories[0].path: got %q, want %q", parsed.Directories[0].Path, "alpha")
	}
	if parsed.Directories[0].Files == nil {
		t.Errorf("directories[alpha].files: expected 3, got nil (omitted)")
	} else if *parsed.Directories[0].Files != 3 {
		t.Errorf("directories[alpha].files: got %d, want 3", *parsed.Directories[0].Files)
	}

	// beta: Files=5 — must be present with value 5.
	if parsed.Directories[1].Path != "beta" {
		t.Fatalf("directories[1].path: got %q, want %q", parsed.Directories[1].Path, "beta")
	}
	if parsed.Directories[1].Files == nil {
		t.Errorf("directories[beta].files: expected 5, got nil (omitted)")
	} else if *parsed.Directories[1].Files != 5 {
		t.Errorf("directories[beta].files: got %d, want 5", *parsed.Directories[1].Files)
	}

	// gamma: Files=0 — must be absent per omitempty.
	if parsed.Directories[2].Path != "gamma" {
		t.Fatalf("directories[2].path: got %q, want %q", parsed.Directories[2].Path, "gamma")
	}
	if parsed.Directories[2].Files != nil {
		t.Errorf("directories[gamma].files: expected nil (omitted by omitempty for Files=0), got %d", *parsed.Directories[2].Files)
	}
}

// TestRenderer_DirectoriesFilesColumn_Human verifies that human per-directory
// KV blocks include a "Files" row before "Bytes", and that the grand-total
// block does NOT include a "Files" row (s.Total is counting.Counts, no Files).
func TestRenderer_DirectoriesFilesColumn_Human(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	r := newHumanRendererWithMode(testHumanMode)
	if err := r.RenderTree(&buf, dirFilesFixture(), nil); err != nil {
		t.Fatalf("RenderTree: %v", err)
	}
	got := buf.String()

	// Per-dir blocks must contain "Files".
	if !strings.Contains(got, "Files") {
		t.Errorf("human per-dir blocks: output missing \"Files\" label; got:\n%s", got)
	}

	// "Files" must appear before "Bytes" within the per-dir section.
	// Since there are multiple blocks, verify first occurrence of "Files" is
	// before first occurrence of "Bytes".
	idxFiles := strings.Index(got, "Files")
	idxBytes := strings.Index(got, "Bytes")
	if idxFiles < 0 || idxBytes < 0 {
		t.Fatalf("human output missing Files or Bytes; got:\n%s", got)
	}
	if idxFiles >= idxBytes {
		t.Errorf("human: \"Files\" must appear before \"Bytes\"; Files=%d Bytes=%d; got:\n%s", idxFiles, idxBytes, got)
	}

	// Grand-total block must NOT contain a "Files" row. The total block is
	// identified by the "total" title. We find the start of the total block and
	// verify no "Files" label appears between it and end-of-output.
	idxTotal := strings.LastIndex(got, "total")
	if idxTotal < 0 {
		t.Fatalf("human output missing 'total' block; got:\n%s", got)
	}
	totalSection := got[idxTotal:]
	// "Files" must not appear in the total section (only Bytes/Lines/Words/Chars).
	if strings.Contains(totalSection, "Files") {
		t.Errorf("human grand-total block must NOT contain 'Files' row; got total section:\n%s", totalSection)
	}
}

// TestHumanRenderer_RenderTree_AllUnknown verifies that when ByLang contains
// only LangUnknown, no language row appears in the human output (F33).
func TestHumanRenderer_RenderTree_AllUnknown(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	r := newHumanRendererWithMode(testHumanMode)
	s := summary.Summary{
		Dirs: []summary.Directory{
			{
				Path:   ".",
				Counts: counting.Counts{Bytes: 12, Lines: 1, Words: 2, Chars: 12},
				ByLang: map[lang.Language]lang.LangCounts{
					lang.LangUnknown: {
						Lines:  lang.LineCounts{Blank: 0, Comment: 0, Code: 1},
						Counts: counting.Counts{Bytes: 12, Lines: 1, Words: 2, Chars: 12},
					},
				},
			},
		},
		Total: counting.Counts{Bytes: 12, Lines: 1, Words: 2, Chars: 12},
	}
	if err := r.RenderTree(&buf, s, nil); err != nil {
		t.Fatalf("RenderTree: %v", err)
	}
	got := buf.String()
	// "unknown" must not appear as a language identifier in output after F33.
	if strings.Contains(strings.ToLower(got), "unknown") {
		t.Errorf("human all-unknown ByLang must not emit unknown language row; got:\n%s", got)
	}
}

// --- Unit 9.0 TotalByLang tests ---

// totalByLangFixture builds a summary.Summary with two directories and a
// populated TotalByLang map containing Go and Markdown entries. Used by
// TestRenderer_TotalByLang_TOON, TestRenderer_TotalByLang_JSON, and
// TestRenderer_TotalByLang_Human.
func totalByLangFixture() summary.Summary {
	goLang := lang.LangCounts{
		Lines:  lang.LineCounts{Blank: 1, Comment: 2, Code: 5},
		Counts: counting.Counts{Bytes: 100, Lines: 8, Words: 20, Chars: 100},
	}
	mdLang := lang.LangCounts{
		Lines:  lang.LineCounts{Blank: 3, Comment: 0, Code: 10},
		Counts: counting.Counts{Bytes: 200, Lines: 13, Words: 50, Chars: 200},
	}
	return summary.Summary{
		Dirs: []summary.Directory{
			{
				Path:   ".",
				Counts: counting.Counts{Bytes: 100, Lines: 8, Words: 20, Chars: 100},
				ByLang: map[lang.Language]lang.LangCounts{lang.LangGo: goLang},
			},
			{
				Path:   "docs",
				Counts: counting.Counts{Bytes: 200, Lines: 13, Words: 50, Chars: 200},
				ByLang: map[lang.Language]lang.LangCounts{lang.LangMarkdown: mdLang},
			},
		},
		Total: counting.Counts{Bytes: 300, Lines: 21, Words: 70, Chars: 300},
		TotalByLang: map[lang.Language]lang.LangCounts{
			lang.LangGo:       goLang,
			lang.LangMarkdown: mdLang,
		},
	}
}

// TestRenderer_TotalByLang_TOON verifies that a Summary with two languages in
// TotalByLang causes a total_by_lang tabular array to appear in TOON output,
// containing both language names (F33 LangUnknown suppressed).
func TestRenderer_TotalByLang_TOON(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	r := NewTOONRenderer()
	if err := r.RenderTree(&buf, totalByLangFixture(), nil); err != nil {
		t.Fatalf("RenderTree: %v", err)
	}
	got := buf.String()
	if !strings.Contains(got, "total_by_lang") {
		t.Errorf("TOON TotalByLang: output missing total_by_lang key; got:\n%s", got)
	}
	if !strings.Contains(got, "go") {
		t.Errorf("TOON TotalByLang: output missing go language row; got:\n%s", got)
	}
	if !strings.Contains(got, "markdown") {
		t.Errorf("TOON TotalByLang: output missing markdown language row; got:\n%s", got)
	}
}

// TestRenderer_TotalByLang_JSON verifies that a Summary with two languages in
// TotalByLang causes a total_by_lang field to appear in JSON output with both
// language keys (F33 LangUnknown suppressed, omitempty suppresses when empty).
func TestRenderer_TotalByLang_JSON(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	r := NewJSONRenderer()
	if err := r.RenderTree(&buf, totalByLangFixture(), nil); err != nil {
		t.Fatalf("RenderTree: %v", err)
	}
	got := buf.String()
	if !strings.Contains(got, "total_by_lang") {
		t.Errorf("JSON TotalByLang: output missing total_by_lang key; got:\n%s", got)
	}
	if !strings.Contains(got, `"go"`) {
		t.Errorf("JSON TotalByLang: output missing go key; got:\n%s", got)
	}
	if !strings.Contains(got, `"markdown"`) {
		t.Errorf("JSON TotalByLang: output missing markdown key; got:\n%s", got)
	}
}

// TestRenderer_TotalByLang_Human verifies that a Summary with two languages in
// TotalByLang causes a "total lang: <name>" KV block to appear in human output
// for each known language after the "total" block (F33 LangUnknown suppressed).
func TestRenderer_TotalByLang_Human(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	r := newHumanRendererWithMode(testHumanMode)
	if err := r.RenderTree(&buf, totalByLangFixture(), nil); err != nil {
		t.Fatalf("RenderTree: %v", err)
	}
	got := buf.String()
	if !strings.Contains(got, "total lang: go") {
		t.Errorf("human TotalByLang: output missing 'total lang: go'; got:\n%s", got)
	}
	if !strings.Contains(got, "total lang: markdown") {
		t.Errorf("human TotalByLang: output missing 'total lang: markdown'; got:\n%s", got)
	}
	// total lang: blocks must precede the grand total block (total is last).
	// Use the LAST occurrence of "total" to find the grand-total block (it is
	// the final block in the output after reordering), and the FIRST occurrence
	// of "total lang:" for the first per-language row.
	idxTotalLang := strings.Index(got, "total lang:")
	idxTotal := strings.LastIndex(got, "total")
	if idxTotalLang >= 0 && idxTotal >= 0 && idxTotal <= idxTotalLang {
		t.Errorf("human TotalByLang: 'total lang:' blocks must precede 'total' block; got:\n%s", got)
	}
}

// TestRenderer_TotalByLang_LangUnknownSuppressed verifies that when
// TotalByLang contains only LangUnknown, no total_by_lang block is emitted by
// any renderer (F33 uniform suppression).
func TestRenderer_TotalByLang_LangUnknownSuppressed(t *testing.T) {
	t.Parallel()

	s := summary.Summary{
		Dirs: []summary.Directory{
			{
				Path:   ".",
				Counts: counting.Counts{Bytes: 5, Lines: 1, Words: 1, Chars: 5},
				ByLang: map[lang.Language]lang.LangCounts{
					lang.LangUnknown: {
						Lines:  lang.LineCounts{Blank: 0, Comment: 0, Code: 1},
						Counts: counting.Counts{Bytes: 5, Lines: 1, Words: 1, Chars: 5},
					},
				},
			},
		},
		Total: counting.Counts{Bytes: 5, Lines: 1, Words: 1, Chars: 5},
		TotalByLang: map[lang.Language]lang.LangCounts{
			lang.LangUnknown: {
				Lines:  lang.LineCounts{Blank: 0, Comment: 0, Code: 1},
				Counts: counting.Counts{Bytes: 5, Lines: 1, Words: 1, Chars: 5},
			},
		},
	}

	t.Run("toon", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		r := NewTOONRenderer()
		if err := r.RenderTree(&buf, s, nil); err != nil {
			t.Fatalf("RenderTree: %v", err)
		}
		got := buf.String()
		if strings.Contains(got, "total_by_lang") {
			t.Errorf("TOON: all-unknown TotalByLang must not emit total_by_lang; got:\n%s", got)
		}
	})

	t.Run("json", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		r := NewJSONRenderer()
		if err := r.RenderTree(&buf, s, nil); err != nil {
			t.Fatalf("RenderTree: %v", err)
		}
		got := buf.String()
		if strings.Contains(got, "total_by_lang") {
			t.Errorf("JSON: all-unknown TotalByLang must not emit total_by_lang; got:\n%s", got)
		}
	})

	t.Run("human", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		r := newHumanRendererWithMode(testHumanMode)
		if err := r.RenderTree(&buf, s, nil); err != nil {
			t.Fatalf("RenderTree: %v", err)
		}
		got := buf.String()
		if strings.Contains(got, "total lang:") {
			t.Errorf("human: all-unknown TotalByLang must not emit 'total lang:' block; got:\n%s", got)
		}
	})
}
