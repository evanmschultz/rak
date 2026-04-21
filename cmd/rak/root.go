package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/evanmschultz/rak/internal/counting"
	"github.com/evanmschultz/rak/internal/render"
)

// newRootCmd builds the root Cobra command for rak. The factory returns a
// fresh *cobra.Command per call so tests can own an isolated flag-state
// binding via the closure-local format var declared inside the factory.
func newRootCmd() *cobra.Command {
	// format is the --format flag target. Declared inside the factory rather
	// than at package scope so each newRootCmd() call gets its own binding —
	// test cases can SetArgs independently without leaking flag state across
	// subtests.
	var format string

	cmd := &cobra.Command{
		Use:   "rak [path]",
		Short: "Summarize code in a directory: line, word, and token counts by language",
		Long: "rak walks a path, detects languages, and reports byte, line, " +
			"word, character, and (eventually) token counts rolled up by " +
			"directory and language.\n\n" +
			"Drop 2 wires stdin → counting → render. Path arguments land in " +
			"Drop 3 when the walker arrives; until then pipe input via stdin.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return runRoot(c, args, format)
		},
	}

	cmd.Flags().StringVarP(
		&format,
		"format",
		"f",
		"auto",
		"output format: auto | human | json",
	)

	return cmd
}

// runRoot is the real RunE body. Split out of newRootCmd so the closure is
// a thin shim around a testable, argument-explicit function.
func runRoot(c *cobra.Command, args []string, format string) error {
	// c.Context() is threaded here so future work (walker, concurrency,
	// signal cancellation) can honor cmd-scoped cancellation without
	// inventing a fresh context.Background().
	_ = c.Context()

	if len(args) == 1 {
		return fmt.Errorf(
			"positional path argument not supported yet — walker lands in Drop 3; "+
				"pipe input via stdin for now (got %q)",
			args[0],
		)
	}

	counts, err := counting.Count(c.InOrStdin())
	if err != nil {
		return fmt.Errorf("count input: %w", err)
	}

	renderer, err := selectRenderer(format)
	if err != nil {
		return err
	}

	if err := renderer.Render(c.OutOrStdout(), counts); err != nil {
		return fmt.Errorf("render counts: %w", err)
	}
	return nil
}

// selectRenderer maps the --format flag value to a render.Renderer. "auto"
// and "human" both pick NewHumanRenderer — laslig's per-call printer
// construction inside Render auto-selects plain non-styled output when
// cmd.OutOrStdout() is not a TTY, so "auto" does not need its own
// TTY-detection path in rak. Any other value returns a wrapped error.
func selectRenderer(format string) (render.Renderer, error) {
	switch format {
	case "auto", "human":
		return render.NewHumanRenderer(), nil
	case "json":
		return render.NewJSONRenderer(), nil
	default:
		return nil, fmt.Errorf("invalid --format %q: want auto | human | json", format)
	}
}
