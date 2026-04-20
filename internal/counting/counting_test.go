package counting

import (
	"strings"
	"testing"
)

func TestCount(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input string
		want  Counts
	}{
		{
			name:  "empty",
			input: "",
			want:  Counts{Bytes: 0, Lines: 0, Words: 0, Chars: 0},
		},
		{
			name:  "single_word_no_trailing_newline",
			input: "hello",
			want:  Counts{Bytes: 5, Lines: 0, Words: 1, Chars: 5},
		},
		{
			name:  "single_word_trailing_newline",
			input: "hello\n",
			want:  Counts{Bytes: 6, Lines: 1, Words: 1, Chars: 6},
		},
		{
			name:  "two_words_space_no_newline",
			input: "hello world",
			want:  Counts{Bytes: 11, Lines: 0, Words: 2, Chars: 11},
		},
		{
			name:  "two_lines_four_words",
			input: "hello world\nfoo bar\n",
			want:  Counts{Bytes: 20, Lines: 2, Words: 4, Chars: 20},
		},
		{
			name:  "utf8_multibyte_rune",
			input: "héllo\n",
			want:  Counts{Bytes: 7, Lines: 1, Words: 1, Chars: 6},
		},
		{
			name:  "crlf_line_endings",
			input: "a\r\nb\r\n",
			want:  Counts{Bytes: 6, Lines: 2, Words: 2, Chars: 6},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := Count(strings.NewReader(tc.input))
			if err != nil {
				t.Fatalf("Count(%q) returned unexpected error: %v", tc.input, err)
			}
			if got != tc.want {
				t.Errorf("Count(%q) = %+v, want %+v", tc.input, got, tc.want)
			}
		})
	}
}
