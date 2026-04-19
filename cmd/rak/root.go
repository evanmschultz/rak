package main

import (
	"bufio"
	"fmt"
	"io"
	"unicode"

	"github.com/spf13/cobra"
)

// Counts stores the computed results for a single input.
type Counts struct {
	Bytes int64
	Lines int64
	Words int64
	Chars int64
}

// newRootCmd builds the root Cobra command for rak.
func newRootCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rak [path]",
		Short: "Summarize code in a directory: line, word, and token counts by language",
		Long: "rak walks a path, detects languages, and reports byte, line, " +
			"word, character, and (eventually) token counts rolled up by " +
			"directory and language.\n\n" +
			"Drop 1 ships only the command surface; real counting lands in " +
			"subsequent drops.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			// c.Context() is threaded here so future work (walker,
			// concurrency, signal cancellation) can honor cmd-scoped
			// cancellation without inventing a fresh context.Background().
			_ = c.Context()
			return fmt.Errorf("not implemented — see drop 2")
		},
	}
}

// count reads from r and computes byte, line, word, and character totals.
func count(r io.Reader) (Counts, error) {
	reader := bufio.NewReader(r)
	var counts Counts
	inWord := false

	for {
		// ReadRune is useful here because it gives us both:
		//   - the rune itself, which we use for whitespace/newline logic
		//   - the byte width of that rune in UTF-8, which we use for byte counts
		ch, size, err := reader.ReadRune()
		if err == io.EOF {
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
