package fileset

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"
)

func TestFile_Open(t *testing.T) {
	t.Parallel()

	want := []byte("hello world\n")
	fsys := fstest.MapFS{
		"greet.txt": &fstest.MapFile{Data: want},
	}

	f := newFile(fsys, "greet.txt", "greet.txt")
	rc, err := f.Open()
	if err != nil {
		t.Fatalf("Open() error = %v, want nil", err)
	}

	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("ReadAll() error = %v, want nil", err)
	}
	if err := rc.Close(); err != nil {
		t.Fatalf("Close() error = %v, want nil", err)
	}

	if !bytes.Equal(got, want) {
		t.Errorf("Open() bytes = %q, want %q", got, want)
	}
}

func TestFile_Open_NotFound(t *testing.T) {
	t.Parallel()

	// Empty MapFS — any Open call fails with fs.ErrNotExist.
	fsys := fstest.MapFS{}
	f := newFile(fsys, "missing.txt", "missing.txt")

	_, err := f.Open()
	if err == nil {
		t.Fatal("Open() error = nil, want non-nil")
	}

	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("errors.Is(err, fs.ErrNotExist) = false, want true (err = %v)", err)
	}

	// Error wrapping prefix: open "missing.txt": ...
	if !strings.Contains(err.Error(), `open "missing.txt":`) {
		t.Errorf("error text = %q, want prefix %q", err.Error(), `open "missing.txt":`)
	}
}

func TestFile_Peek(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content []byte
		n       int
		want    []byte
	}{
		{
			name:    "empty_file_returns_empty",
			content: []byte{},
			n:       512,
			want:    []byte{},
		},
		{
			name:    "short_file_returns_all_bytes",
			content: []byte("hi"),
			n:       512,
			want:    []byte("hi"),
		},
		{
			name:    "exact_match_returns_all_bytes",
			content: []byte("exactly8"),
			n:       8,
			want:    []byte("exactly8"),
		},
		{
			name:    "long_file_returns_first_n_bytes",
			content: []byte("0123456789abcdef"),
			n:       8,
			want:    []byte("01234567"),
		},
		{
			name:    "one_byte_file_n_one",
			content: []byte{0x41},
			n:       1,
			want:    []byte{0x41},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			fsys := fstest.MapFS{
				"data.bin": &fstest.MapFile{Data: tc.content},
			}
			f := newFile(fsys, "data.bin", "data.bin")

			got, err := f.Peek(tc.n)
			if err != nil {
				t.Fatalf("Peek(%d) error = %v, want nil", tc.n, err)
			}
			if !bytes.Equal(got, tc.want) {
				t.Errorf("Peek(%d) = %q, want %q", tc.n, got, tc.want)
			}
		})
	}
}

func TestFile_Peek_MultipleCalls(t *testing.T) {
	t.Parallel()

	content := []byte("deterministic peek payload")
	fsys := fstest.MapFS{
		"data.bin": &fstest.MapFile{Data: content},
	}
	f := newFile(fsys, "data.bin", "data.bin")

	first, err := f.Peek(10)
	if err != nil {
		t.Fatalf("Peek #1 error = %v, want nil", err)
	}
	second, err := f.Peek(10)
	if err != nil {
		t.Fatalf("Peek #2 error = %v, want nil", err)
	}

	if !bytes.Equal(first, second) {
		t.Errorf("Peek #1 = %q, Peek #2 = %q, want identical", first, second)
	}
	if !bytes.Equal(first, []byte("determinis")) {
		t.Errorf("Peek(10) = %q, want %q", first, "determinis")
	}
}

func TestIsHidden(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want bool
	}{
		{name: "dot", in: ".", want: false},
		{name: "dotdot", in: "..", want: false},
		{name: "dotgit", in: ".git", want: true},
		{name: "dothidden_file", in: ".hidden.txt", want: true},
		{name: "normal_file", in: "normal.txt", want: false},
		{name: "empty", in: "", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := IsHidden(tc.in)
			if got != tc.want {
				t.Errorf("IsHidden(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}
