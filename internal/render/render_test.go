package render

import (
	"bytes"
	"testing"

	"github.com/evanmschultz/laslig"

	"github.com/evanmschultz/rak/internal/counting"
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
