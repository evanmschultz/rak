package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"sort"

	"github.com/spf13/cobra"

	"github.com/evanmschultz/rak/internal/counting"
	"github.com/evanmschultz/rak/internal/fileset"
	"github.com/evanmschultz/rak/internal/render"
)

// rootFlags bundles every flag bound to the root command so runRoot and the
// cobra factory stay decoupled from the flag-variable lifetime. The bundle
// is declared inside newRootCmd (closure-local) so each test Execute owns
// an isolated flag-state binding.
type rootFlags struct {
	format      string
	depth       int
	hidden      bool
	noGitignore bool
	binary      bool
	includes    []string
	excludes    []string
}

// newRootCmd builds the root Cobra command for rak. The factory returns a
// fresh *cobra.Command per call so tests can own an isolated flag-state
// binding via the closure-local flags bundle declared inside the factory.
func newRootCmd() *cobra.Command {
	flags := &rootFlags{}

	cmd := &cobra.Command{
		Use:   "rak [path]",
		Short: "Summarize code in a directory: line, word, and token counts by language",
		Long: "rak walks a path, detects languages, and reports byte, line, " +
			"word, character, and (eventually) token counts rolled up by " +
			"directory and language.\n\n" +
			"With no positional argument rak reads stdin and reports totals " +
			"for the stream. With a single path argument rak walks the " +
			"directory and reports per-directory rollups plus a grand total.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return runRoot(c, args, flags)
		},
	}

	cmd.Flags().StringVarP(
		&flags.format,
		"format",
		"f",
		"auto",
		"output format: auto | human | json",
	)
	cmd.Flags().IntVar(
		&flags.depth,
		"depth",
		0,
		"maximum directory edges to descend from the walk root (0 = no limit)",
	)
	cmd.Flags().BoolVar(
		&flags.hidden,
		"hidden",
		false,
		"include hidden files and directories (names starting with '.')",
	)
	cmd.Flags().BoolVar(
		&flags.noGitignore,
		"no-gitignore",
		false,
		"disable .gitignore-based filtering during the walk",
	)
	cmd.Flags().BoolVar(
		&flags.binary,
		"binary",
		false,
		"include binary files (NUL byte in first 512 bytes) instead of skipping them",
	)
	cmd.Flags().StringSliceVar(
		&flags.includes,
		"include",
		nil,
		"glob pattern that walked files must match (repeatable; doublestar '**' supported)",
	)
	cmd.Flags().StringSliceVar(
		&flags.excludes,
		"exclude",
		nil,
		"glob pattern that dropped files must match (repeatable; exclude wins over include)",
	)

	return cmd
}

// runRoot is the real RunE body. Split out of newRootCmd so the closure is
// a thin shim around a testable, argument-explicit function.
func runRoot(c *cobra.Command, args []string, flags *rootFlags) error {
	ctx := c.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	renderer, err := selectRenderer(flags.format)
	if err != nil {
		return err
	}

	if len(args) == 1 {
		return runDirectory(ctx, c.OutOrStdout(), args[0], os.DirFS(args[0]), flags, renderer)
	}

	counts, err := counting.Count(c.InOrStdin())
	if err != nil {
		return fmt.Errorf("count input: %w", err)
	}

	if err := renderer.Render(c.OutOrStdout(), counts); err != nil {
		return fmt.Errorf("render counts: %w", err)
	}
	return nil
}

// runDirectory performs the len(args)==1 walk case. fsys is passed as a
// parameter rather than built inside so tests can inject a stub fs.FS that
// induces specific per-file errors (e.g. fs.ErrPermission on Open) without
// staging fixtures on disk. rootLabel is the user-facing root path string
// that appears in the rendered "dir: <path>" titles; in production this is
// args[0], in tests it is whatever label makes the assertion readable.
func runDirectory(
	ctx context.Context,
	w io.Writer,
	rootLabel string,
	fsys fs.FS,
	flags *rootFlags,
	renderer render.Renderer,
) error {
	dirs, total, aggErrs, err := walkAndCount(ctx, fsys, flags)
	if err != nil {
		return err
	}

	// The walker speaks in io/fs paths rooted at "." ; rewrite the leading
	// "." segment to the user-facing rootLabel so the rendered output
	// reads naturally (for example "dir: ./testdata/tree" rather than
	// "dir: ."). Empty rootLabel keeps the io/fs convention intact for
	// callers (tests) that prefer it.
	labeled := labelDirectories(dirs, rootLabel)

	if err := renderer.RenderTree(w, labeled, total, aggErrs); err != nil {
		return fmt.Errorf("render tree: %w", err)
	}
	return nil
}

// walkAndCount runs the walker over fsys, aggregates per-directory counts,
// and returns the directory list (in deterministic lexical order), the
// grand total, and any walker-level errors the caller should surface via
// the renderer's error summary.
//
// Only ctx.Err() aborts the walk. All other error conditions — walker
// per-entry errors, IsBinary open failures, per-file count failures — are
// aggregated into the returned errs slice and the walk continues so one
// broken directory does not kill the whole count. This mirrors F6 (walker
// continues past per-entry errors) at the aggregation boundary and matches
// C10 (IsBinary open failures are aggregated, not fatal).
func walkAndCount(ctx context.Context, fsys fs.FS, flags *rootFlags) ([]render.Directory, counting.Counts, []error, error) {
	walker := fileset.NewWalker(fsys, ".", fileset.WalkOptions{
		Depth:            flags.depth,
		IncludeHidden:    flags.hidden,
		DisableGitignore: flags.noGitignore,
		Includes:         flags.includes,
		Excludes:         flags.excludes,
	})

	byDir := map[string]counting.Counts{}
	var total counting.Counts
	var aggErrs []error

	for f, walkErr := range walker.Walk(ctx) {
		if walkErr != nil {
			// Context cancellation terminates the run; wrap and return.
			// Walker yields ctx.Err() once and then stops; treat it as
			// fatal here because the user asked to cancel.
			if errors.Is(walkErr, context.Canceled) || errors.Is(walkErr, context.DeadlineExceeded) {
				return nil, counting.Counts{}, nil, fmt.Errorf("walk: %w", walkErr)
			}
			// Any other walker-level error goes into the error summary
			// and the walk continues (F6).
			aggErrs = append(aggErrs, walkErr)
			continue
		}

		// Binary detection policy (C10 / F12): decided here, not in the
		// walker. IsBinary errors are aggregated into the summary and the
		// file is skipped from counting.
		if !flags.binary {
			isBin, err := f.IsBinary()
			if err != nil {
				aggErrs = append(aggErrs, fmt.Errorf("binary check %q: %w", f.RelPath, err))
				continue
			}
			if isBin {
				continue
			}
		}

		fileCounts, err := countFile(f)
		if err != nil {
			aggErrs = append(aggErrs, err)
			continue
		}

		dir := dirKey(f.RelPath)
		byDir[dir] = addCounts(byDir[dir], fileCounts)
		total = addCounts(total, fileCounts)
	}

	dirs := make([]render.Directory, 0, len(byDir))
	for p, c := range byDir {
		dirs = append(dirs, render.Directory{Path: p, Counts: c})
	}
	sort.Slice(dirs, func(i, j int) bool { return dirs[i].Path < dirs[j].Path })

	return dirs, total, aggErrs, nil
}

// countFile opens f via the walker-reported handle, streams it through
// counting.Count, and wraps any error with the RelPath so the aggregated
// error summary identifies which file failed.
func countFile(f *fileset.File) (counting.Counts, error) {
	rc, err := f.Open()
	if err != nil {
		return counting.Counts{}, fmt.Errorf("count %q: %w", f.RelPath, err)
	}
	defer func() { _ = rc.Close() }()

	counts, err := counting.Count(rc)
	if err != nil {
		return counting.Counts{}, fmt.Errorf("count %q: %w", f.RelPath, err)
	}
	return counts, nil
}

// dirKey returns the walk-relative directory containing the given
// walk-relative file path. Files at the walk root report "." to match the
// io/fs root convention used elsewhere in rak.
func dirKey(relPath string) string {
	if relPath == "" {
		return "."
	}
	dir := path.Dir(relPath)
	if dir == "" {
		return "."
	}
	return dir
}

// addCounts sums two counting.Counts field-wise.
func addCounts(a, b counting.Counts) counting.Counts {
	return counting.Counts{
		Bytes: a.Bytes + b.Bytes,
		Lines: a.Lines + b.Lines,
		Words: a.Words + b.Words,
		Chars: a.Chars + b.Chars,
	}
}

// labelDirectories rewrites the leading "." path used by the walker into a
// user-facing root label so rendered titles read naturally when the user
// passed a positional path argument. A "." becomes exactly rootLabel; any
// "sub", "sub/nested" etc. becomes "<rootLabel>/<relative>". Passing an
// empty rootLabel returns the input unchanged, preserving the io/fs
// convention for test callers that want it.
func labelDirectories(dirs []render.Directory, rootLabel string) []render.Directory {
	if rootLabel == "" {
		return dirs
	}
	out := make([]render.Directory, len(dirs))
	for i, d := range dirs {
		if d.Path == "." {
			out[i] = render.Directory{Path: rootLabel, Counts: d.Counts}
			continue
		}
		out[i] = render.Directory{
			Path:   rootLabel + "/" + d.Path,
			Counts: d.Counts,
		}
	}
	return out
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
