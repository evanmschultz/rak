package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode"

	"github.com/spf13/cobra"
)

// Config stores the user-selected settings for a single command run.
type Config struct {
	Filename  string
	ShowBytes bool
	ShowLines bool
	ShowWords bool
	ShowChars bool
}

// Counts stores the computed results for a single input.
type Counts struct {
	Bytes int64
	Lines int64
	Words int64
	Chars int64
}

// newRootCmd builds the root Cobra command for the CLI.
func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "fwc [file]",
		Short: "Count bytes, lines, words, and characters in a file",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			cfg, err := configFromCommand(c, args[0])
			if err != nil {
				return err
			}

			// c.OutOrStdout keeps output routed through Cobra's configured writer.
			return run(cfg, c.OutOrStdout())
		},
	}

	root.Flags().BoolP("bytes", "b", false, "count bytes")
	root.Flags().BoolP("lines", "l", false, "count lines")
	root.Flags().BoolP("words", "w", false, "count words")
	root.Flags().BoolP("chars", "c", false, "count characters")

	return root
}

// configFromCommand builds a fresh Config struct from the parsed Cobra command state.
func configFromCommand(cmd *cobra.Command, filename string) (Config, error) {
	flags := cmd.Flags()
	cfg := Config{Filename: filename}

	showBytes, err := flags.GetBool("bytes")
	if err != nil {
		return Config{}, err
	}
	showLines, err := flags.GetBool("lines")
	if err != nil {
		return Config{}, err
	}
	showWords, err := flags.GetBool("words")
	if err != nil {
		return Config{}, err
	}
	showChars, err := flags.GetBool("chars")
	if err != nil {
		return Config{}, err
	}

	// If no flags were passed in at command call, show all computed values.
	if !flags.Changed("bytes") && !flags.Changed("lines") && !flags.Changed("words") && !flags.Changed("chars") {
		cfg.ShowBytes = true
		cfg.ShowLines = true
		cfg.ShowWords = true
		cfg.ShowChars = true
		return cfg, nil
	}

	cfg.ShowBytes = showBytes
	cfg.ShowLines = showLines
	cfg.ShowWords = showWords
	cfg.ShowChars = showChars

	return cfg, nil
}

// run opens the file, computes counts, and prints the selected output fields.
func run(cfg Config, w io.Writer) error {
	file, err := os.Open(cfg.Filename)
	if err != nil {
		return err
	}
	defer func() {
		if err := file.Close(); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}()

	counts, err := count(file)
	if err != nil {
		return err
	}

	return printCounts(w, cfg, counts)
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

// printCounts prints only the fields selected by the config.
func printCounts(w io.Writer, cfg Config, counts Counts) error {
	parts := make([]string, 0, 5)

	if cfg.ShowBytes {
		parts = append(parts, fmt.Sprintf("Bytes: %8d\n", counts.Bytes))
	}
	if cfg.ShowLines {
		parts = append(parts, fmt.Sprintf("Lines: %8d\n", counts.Lines))
	}
	if cfg.ShowWords {
		parts = append(parts, fmt.Sprintf("Words: %8d\n", counts.Words))
	}
	if cfg.ShowChars {
		parts = append(parts, fmt.Sprintf("Chars: %8d\n", counts.Chars))
	}

	// The filename always appears at the end so the output still identifies
	// which input was processed, even when only one count is shown.
	parts = append(parts, fmt.Sprintf("File: %s", cfg.Filename))

	_, err := fmt.Fprintln(w, strings.Join(parts, ""))
	return err
}
