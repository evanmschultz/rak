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
	dirs := []summary.Directory{
		{Path: ".", Counts: counting.Counts{Bytes: 12, Lines: 1, Words: 2, Chars: 12}},
		{Path: "sub", Counts: counting.Counts{Bytes: 4, Lines: 1, Words: 1, Chars: 4}},
	}
	total := counting.Counts{Bytes: 16, Lines: 2, Words: 3, Chars: 16}
	if err := r.RenderTree(&buf, dirs, total, nil); err != nil {
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
	dirs := []summary.Directory{
		{Path: ".", Counts: counting.Counts{Bytes: 1, Lines: 0, Words: 0, Chars: 1}},
	}
	if err := r.RenderTree(&buf, dirs, dirs[0].Counts, nil); err != nil {
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
	dirs := []summary.Directory{
		{Path: ".", Counts: counting.Counts{Bytes: 1, Lines: 0, Words: 0, Chars: 1}},
	}
	errs := []error{errors.New("walk \"foo\": permission denied"), errors.New("walk \"bar\": not a directory")}
	if err := r.RenderTree(&buf, dirs, dirs[0].Counts, errs); err != nil {
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
	if err := r.RenderTree(&buf, nil, counting.Counts{}, nil); err != nil {
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
	dirs := []summary.Directory{
		{Path: ".", Counts: counting.Counts{Bytes: 12, Lines: 1, Words: 2, Chars: 12}},
		{Path: "sub", Counts: counting.Counts{Bytes: 4, Lines: 1, Words: 1, Chars: 4}},
	}
	total := counting.Counts{Bytes: 16, Lines: 2, Words: 3, Chars: 16}
	if err := r.RenderTree(&buf, dirs, total, nil); err != nil {
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
	if err := r.RenderTree(&buf, nil, counting.Counts{}, nil); err != nil {
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
	dirs := []summary.Directory{
		{Path: ".", Counts: counting.Counts{Bytes: 1, Lines: 0, Words: 0, Chars: 1}},
	}
	errs := []error{errors.New("walk \"foo\": permission denied")}
	if err := r.RenderTree(&buf, dirs, dirs[0].Counts, errs); err != nil {
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
	dirs := []summary.Directory{
		{Path: ".", Counts: counting.Counts{Bytes: 5, Lines: 1, Words: 1, Chars: 5}},
		{Path: "sub", Counts: counting.Counts{Bytes: 3, Lines: 1, Words: 1, Chars: 3}},
	}
	total := counting.Counts{Bytes: 8, Lines: 2, Words: 2, Chars: 8}
	if err := r.RenderTree(&buf, dirs, total, nil); err != nil {
		t.Fatalf("RenderTree: %v", err)
	}
	got := buf.String()
	for _, want := range []string{"directories", ".|", "sub", "total"} {
		if !strings.Contains(got, want) {
			t.Errorf("RenderTree output missing %q; got:\n%s", want, got)
		}
	}
}

// TestTOONRenderer_RenderTree_WithErrors verifies that a non-empty errs slice
// causes the output to contain an "errors" field.
func TestTOONRenderer_RenderTree_WithErrors(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	r := NewTOONRenderer()
	dirs := []summary.Directory{
		{Path: ".", Counts: counting.Counts{Bytes: 1, Lines: 0, Words: 0, Chars: 1}},
	}
	errs := []error{errors.New("walk \"foo\": permission denied")}
	if err := r.RenderTree(&buf, dirs, dirs[0].Counts, errs); err != nil {
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
	dirs := []summary.Directory{
		{Path: ".", Counts: counting.Counts{Bytes: 1, Lines: 0, Words: 0, Chars: 1}},
	}
	if err := r.RenderTree(&buf, dirs, dirs[0].Counts, nil); err != nil {
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
	dirs := []summary.Directory{
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
	}
	total := counting.Counts{Bytes: 26, Lines: 2, Words: 2, Chars: 26}
	if err := r.RenderTree(&buf, dirs, total, nil); err != nil {
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
	dirs := []summary.Directory{
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
	}
	total := counting.Counts{Bytes: 12, Lines: 1, Words: 2, Chars: 12}
	if err := r.RenderTree(&buf, dirs, total, nil); err != nil {
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
	dirs := []summary.Directory{
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
	}
	total := counting.Counts{Bytes: 26, Lines: 2, Words: 2, Chars: 26}
	if err := r.RenderTree(&buf, dirs, total, nil); err != nil {
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
	dirs := []summary.Directory{
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
	}
	total := counting.Counts{Bytes: 12, Lines: 1, Words: 2, Chars: 12}
	if err := r.RenderTree(&buf, dirs, total, nil); err != nil {
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
	dirs := []summary.Directory{
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
	}
	total := counting.Counts{Bytes: 26, Lines: 2, Words: 2, Chars: 26}
	if err := r.RenderTree(&buf, dirs, total, nil); err != nil {
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

// TestHumanRenderer_RenderTree_AllUnknown verifies that when ByLang contains
// only LangUnknown, no language row appears in the human output (F33).
func TestHumanRenderer_RenderTree_AllUnknown(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	r := newHumanRendererWithMode(testHumanMode)
	dirs := []summary.Directory{
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
	}
	total := counting.Counts{Bytes: 12, Lines: 1, Words: 2, Chars: 12}
	if err := r.RenderTree(&buf, dirs, total, nil); err != nil {
		t.Fatalf("RenderTree: %v", err)
	}
	got := buf.String()
	// "unknown" must not appear as a language identifier in output after F33.
	if strings.Contains(strings.ToLower(got), "unknown") {
		t.Errorf("human all-unknown ByLang must not emit unknown language row; got:\n%s", got)
	}
}
