// Package counting computes byte, line, word, and character totals over an
// io.Reader. It is the pure-stream-math leaf package of rak's internal
// dependency DAG: zero internal deps, no filesystem access, no language
// awareness.
package counting

import (
	"bufio"
	"errors"
	"io"
	"unicode"
)

// Counts stores the computed results for a single input. The field
// declaration order — Bytes, Lines, Words, Chars — is load-bearing: the
// internal/render JSON renderer emits fields in this order via
// encoding/json without struct tags, and downstream snapshot tests pin to
// that exact shape.
type Counts struct {
	// Bytes is the total number of UTF-8 bytes read from the input.
	Bytes int64
	// Lines is the number of line-feed ('\n') runes observed. '\r' on its
	// own does not increment Lines.
	Lines int64
	// Words is the number of whitespace-delimited tokens per
	// unicode.IsSpace, where any run of whitespace separates words.
	Words int64
	// Chars is the total number of runes read from the input (may differ
	// from Bytes when the input contains multi-byte UTF-8 runes).
	Chars int64
}

// Count reads r to completion and returns the byte, line, word, and
// character totals. A clean io.EOF is treated as successful termination;
// any other read error is returned (with a zero Counts) unwrapped so
// callers can wrap with their own context.
func Count(r io.Reader) (Counts, error) {
	reader := bufio.NewReader(r)
	var counts Counts
	inWord := false

	for {
		// ReadRune is useful here because it gives us both:
		//   - the rune itself, which we use for whitespace/newline logic
		//   - the byte width of that rune in UTF-8, which we use for byte counts
		ch, size, err := reader.ReadRune()
		if errors.Is(err, io.EOF) {
			return counts, nil
		}
		if err != nil {
			return Counts{}, err
		}

		counts.Bytes += int64(size)
		counts.Chars++

		if ch == '\n' {
			counts.Lines++
		}

		// unicode.IsSpace lets the word-counting logic work across more than
		// just ASCII spaces.
		if unicode.IsSpace(ch) {
			inWord = false
			continue
		}

		if !inWord {
			inWord = true
			counts.Words++
		}
	}
}
