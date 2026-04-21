package fileset

import (
	"bytes"
	"testing"
	"testing/fstest"
)

// buildASCII returns a slice of n ASCII 'A' bytes. Used to build fixtures that
// push content past the 512-byte peek window without introducing any NUL
// bytes in the window itself.
func buildASCII(n int) []byte {
	out := bytes.Repeat([]byte{'A'}, n)
	return out
}

// buildASCIIThenNULAt returns a slice of total length totalLen, filled with
// 'A' bytes, with a single NUL byte at index nulPos. nulPos must be < totalLen
// and >= 0; no bounds-checking beyond that — fixture construction only.
func buildASCIIThenNULAt(totalLen, nulPos int) []byte {
	out := buildASCII(totalLen)
	out[nulPos] = 0x00
	return out
}

func TestFile_IsBinary(t *testing.T) {
	t.Parallel()

	// 521-byte fixture: 520 bytes of 'A' + NUL at index 520. Peek(512) only
	// returns the first 512 bytes, which contain no NUL, so the classifier
	// must return false. This is the F10 regression guard — NUL bytes past
	// byte 512 do not trigger binary classification.
	tailNULFixture := buildASCIIThenNULAt(521, 520)

	tests := []struct {
		name    string
		content []byte
		want    bool
	}{
		{
			name:    "empty_file_is_not_binary",
			content: []byte{},
			want:    false,
		},
		{
			name:    "pure_ascii_hello_world_is_not_binary",
			content: []byte("hello world"),
			want:    false,
		},
		{
			name:    "utf8_cafe_is_not_binary",
			content: []byte("café"),
			want:    false,
		},
		{
			name:    "nul_prefixed_buffer_is_binary",
			content: []byte{0x00, 0x01, 0x02, 0x03},
			want:    true,
		},
		{
			name:    "five_hundred_twelve_ascii_bytes_is_not_binary",
			content: buildASCII(512),
			want:    false,
		},
		{
			name:    "nul_past_peek_window_is_not_binary",
			content: tailNULFixture,
			want:    false,
		},
		{
			name:    "png_magic_bytes_is_binary",
			content: []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 0x00, 0x00, 0x00, 0x0d},
			want:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			fsys := fstest.MapFS{
				"data.bin": &fstest.MapFile{Data: tc.content},
			}
			f := newFile(fsys, "data.bin", "data.bin")

			got, err := f.IsBinary()
			if err != nil {
				t.Fatalf("IsBinary() error = %v, want nil", err)
			}
			if got != tc.want {
				t.Errorf("IsBinary() = %v, want %v (content len = %d)", got, tc.want, len(tc.content))
			}
		})
	}
}
