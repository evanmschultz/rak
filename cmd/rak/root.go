package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

// newRootCmd builds the root Cobra command for rak.
func newRootCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rak [path]",
		Short: "Summarize code in a directory: line, word, and token counts by language",
		Long: "rak walks a path, detects languages, and reports byte, line, " +
			"word, character, and (eventually) token counts rolled up by " +
			"directory and language.\n\n" +
			"Drop 1 shipped only the command surface; Drop 2 lifts counting " +
			"into internal/counting and lands render — wiring arrives in " +
			"Unit 2.3.",
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
