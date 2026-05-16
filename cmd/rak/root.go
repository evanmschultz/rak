package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/spf13/cobra"

	"github.com/evanmschultz/rak/internal/counting"
	"github.com/evanmschultz/rak/internal/fileset"
	"github.com/evanmschultz/rak/internal/lang"
	"github.com/evanmschultz/rak/internal/lister"
	"github.com/evanmschultz/rak/internal/render"
	"github.com/evanmschultz/rak/internal/summary"
)

// rootFlags bundles every flag bound to the root command so runRoot and the
// cobra factory stay decoupled from the flag-variable lifetime. The bundle
// is declared inside newRootCmd (closure-local) so each test Execute owns
// an isolated flag-state binding.
type rootFlags struct {
	human       bool
	json        bool
	toon        bool
	depth       int
	hidden      bool
	noGitignore bool
	binary      bool
	includes    []string
	excludes    []string
	langs       []string
	sort        string // sort key: lines, files, bytes, path (default: lines)
	sortAsc     bool   // flip sort direction from the key-specific default
	maxFiles    int    // abort the walk when accepted file count reaches this value (0 = no limit)
}

// ErrMaxFilesExceeded is returned (wrapped) by walkAndCount when the accepted
// file count reaches the --max-files limit. Callers use errors.Is to branch;
// never string-match (F45).
var ErrMaxFilesExceeded = errors.New("rak: file count exceeded --max-files limit")

// validSortKeys is the set of accepted --sort values in v0.1.0. "tokens" is
// intentionally absent (Decision 30 / F41 — deferred to v0.2).
var validSortKeys = map[string]struct{}{
	"lines": {},
	"files": {},
	"bytes": {},
	"path":  {},
}

// newRootCmd builds the root Cobra command for rak. The factory returns a
// fresh *cobra.Command per call so tests can own an isolated flag-state
// binding via the closure-local flags bundle declared inside the factory.
func newRootCmd() *cobra.Command {
	flags := &rootFlags{}

	cmd := &cobra.Command{
		Use:   "rak [path]",
		Short: "Summarize code in a directory: per-directory and per-language counts",
		Long: "rak walks a path, detects languages, and reports byte, line, " +
			"word, character, and file counts rolled up by " +
			"directory and language. Default output is TOON for LLM-first " +
			"consumption; pass --human or --json for other formats.\n\n" +
			"With no positional argument rak reads stdin and reports totals " +
			"for the stream. With a single path argument rak walks the " +
			"directory and reports per-directory rollups plus a grand total.",
		Args: cobra.MaximumNArgs(1),
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			if _, ok := validSortKeys[flags.sort]; !ok {
				return fmt.Errorf("%q is not a valid sort key; valid keys: lines, files, bytes, path", flags.sort)
			}
			return nil
		},
		RunE: func(c *cobra.Command, args []string) error {
			return runRoot(c, args, flags)
		},
	}

	cmd.Flags().BoolVar(
		&flags.human,
		"human",
		false,
		"render output in human-readable format (laslig)",
	)
	cmd.Flags().BoolVar(
		&flags.json,
		"json",
		false,
		"render output as JSON",
	)
	cmd.Flags().BoolVar(
		&flags.toon,
		"toon",
		false,
		"render output as TOON (default if no other flag set)",
	)
	cmd.MarkFlagsMutuallyExclusive("human", "json", "toon")

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
	cmd.Flags().StringSliceVar(
		&flags.langs,
		"lang",
		nil,
		"filter counted files to comma-separated language names (e.g. go,rust); default: count all",
	)
	cmd.Flags().StringVar(
		&flags.sort,
		"sort",
		"lines",
		"sort directories by key: lines, files, bytes, path (default: lines; numeric keys default descending, path defaults ascending)",
	)
	cmd.Flags().BoolVar(
		&flags.sortAsc,
		"sort-asc",
		false,
		"flip sort direction from its key-specific default",
	)
	cmd.Flags().IntVar(
		&flags.maxFiles,
		"max-files",
		0,
		"abort the walk when the file count exceeds N (default 0 = no limit)",
	)

	return cmd
}

// resolveRenderer maps the rootFlags format booleans to a concrete
// render.Renderer. Cobra's MarkFlagsMutuallyExclusive guarantee fires before
// RunE, so at most one of human/json/toon is true when resolveRenderer is
// called. The default (no flag set) and --toon both return NewTOONRenderer —
// TOON is rak's default output format (decision 33).
func resolveRenderer(flags *rootFlags) render.Renderer {
	switch {
	case flags.human:
		return render.NewHumanRenderer()
	case flags.json:
		return render.NewJSONRenderer()
	default:
		return render.NewTOONRenderer()
	}
}

// listerOpts translates rootFlags fields into a fileset.WalkOptions value
// for lister.Detect and lister.NewWalkLister callers.
func listerOpts(flags *rootFlags) fileset.WalkOptions {
	return fileset.WalkOptions{
		Depth:            flags.depth,
		IncludeHidden:    flags.hidden,
		DisableGitignore: flags.noGitignore,
		Includes:         flags.includes,
		Excludes:         flags.excludes,
	}
}

// runRoot is the real RunE body. Split out of newRootCmd so the closure is
// a thin shim around a testable, argument-explicit function.
func runRoot(c *cobra.Command, args []string, flags *rootFlags) error {
	ctx := c.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	renderer := resolveRenderer(flags)

	if len(args) == 1 {
		source, err := lister.Detect(ctx, args[0], listerOpts(flags))
		if err != nil {
			// Surface the ErrNoGitignoreInRepo sentinel (and any other
			// lister.Detect error) directly to cobra. The sentinel's
			// Error() carries the full user-visible message; no extra
			// wrapping needed here (F19 contract).
			return err
		}
		return runDirectory(ctx, c.OutOrStdout(), source, args[0], flags.binary, flags.langs, flags.sort, flags.sortAsc, renderer, flags.maxFiles)
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

// runDirectory performs the len(args)==1 walk case. source is the FileLister
// whose List method yields files to count. rootLabel is the user-facing root
// path string that appears in the rendered "dir: <path>" titles; in
// production this is args[0], in tests it is whatever label makes the
// assertion readable. binary controls whether binary files are counted or
// skipped (F23). langs is the raw --lang filter values from rootFlags; an
// empty slice means no filtering (count all languages). sortKey is the raw
// --sort flag value (e.g. "lines", "files", "bytes", "path"); sortAsc is the
// raw --sort-asc flag value. maxFiles is the --max-files limit (0 = no
// limit); walkAndCount aborts with a wrapped ErrMaxFilesExceeded when the
// accepted file count reaches this value. The call order is (F39 / Decision
// 3.3): labelDirectories → SortDirs → RenderTree.
func runDirectory(
	ctx context.Context,
	w io.Writer,
	source lister.FileLister,
	rootLabel string,
	binary bool,
	langs []string,
	sortKey string,
	sortAsc bool,
	renderer render.Renderer,
	maxFiles int,
) error {
	dirs, total, totalByLang, aggErrs, err := walkAndCount(ctx, source, binary, langs, maxFiles)
	if err != nil {
		return err
	}

	// The lister speaks in walk-root-relative paths; rewrite the leading
	// "." segment to the user-facing rootLabel so the rendered output
	// reads naturally (for example "dir: ./testdata/tree" rather than
	// "dir: ."). Empty rootLabel keeps the io/fs convention intact for
	// callers (tests) that prefer it.
	labeled := labelDirectories(dirs, rootLabel)

	// Apply user-controlled sort AFTER labelDirectories so SortDirs
	// operates on the final user-facing paths (Decision 3.3, F39).
	summary.SortDirs(labeled, summary.SortKey(sortKey), sortAsc)

	s := summary.Summary{
		Dirs:        labeled,
		Total:       total,
		TotalByLang: totalByLang,
	}
	if err := renderer.RenderTree(w, s, aggErrs); err != nil {
		return fmt.Errorf("render tree: %w", err)
	}
	return nil
}

// walkAndCount iterates source.List(ctx), aggregates per-directory counts,
// and returns the directory list (in deterministic lexical order), the
// grand total, a per-language grand total collapsed across all directories
// (F46), and any per-entry errors the caller should surface via the
// renderer's error summary.
//
// Only ctx.Err() and the --max-files limit abort the walk; all other error
// conditions — per-entry errors, IsBinary open failures, per-file count
// failures — are aggregated into the returned errs slice and the walk
// continues so one broken entry does not kill the whole count. This mirrors
// F6 (walker continues past per-entry errors) at the aggregation boundary
// and matches C10 (IsBinary open failures are aggregated, not fatal). The
// binary-check policy is preserved verbatim (F23).
//
// langs is the --lang filter value set. When non-empty, only files whose
// detected language is in the set are counted; all others (including
// LangUnknown files) are silently skipped (F29, Decision 24).
//
// maxFiles is the --max-files safety rail (0 = no limit). When positive,
// walkAndCount increments an accepted-files counter at the same gating point
// as byDirFiles (post binary-skip, post lang-filter, post successful count).
// When acceptedFiles reaches maxFiles, the function returns immediately with
// a wrapped ErrMaxFilesExceeded (F45). Results accumulated so far are
// discarded to avoid partial output that could mislead callers.
//
// Per-language line split (Unit 5.3): for each counted file, lang.Split is
// called on a second open of the file to classify lines as blank/comment/code.
// This is a two-open-per-file design (Double-IO trade-off, PLAN.md Notes P4)
// accepted for v0.1.0. Split errors are aggregated into aggErrs but do not
// prevent the file's byte/line/word/char counts from being included. The
// per-dir/per-lang LangCounts are accumulated into byDirLang and surfaced via
// Directory.ByLang. The walk-level totalByLang map accumulates the same
// LangCounts across all directories (F46). LangUnknown suppression (F33) is
// the renderer's responsibility; walkAndCount includes LangUnknown in both
// byDirLang and totalByLang.
func walkAndCount(ctx context.Context, source lister.FileLister, binary bool, langs []string, maxFiles int) ([]summary.Directory, counting.Counts, map[lang.Language]lang.LangCounts, []error, error) {
	byDir := map[string]counting.Counts{}
	byDirLang := map[string]map[lang.Language]lang.LangCounts{}
	byDirFiles := map[string]int64{}
	totalByLang := map[lang.Language]lang.LangCounts{}
	var total counting.Counts
	var aggErrs []error
	var acceptedFiles int

	// Build the lang-filter lookup set once before the per-file loop.
	// Case-insensitive normalization: user values are lowercased to match
	// the lowercase Language constant convention (C6, F29).
	var wantedLangs map[lang.Language]struct{}
	if len(langs) > 0 {
		wantedLangs = make(map[lang.Language]struct{}, len(langs))
		for _, v := range langs {
			wantedLangs[lang.Language(strings.ToLower(v))] = struct{}{}
		}
	}

	for f, walkErr := range source.List(ctx) {
		if walkErr != nil {
			// Context cancellation terminates the run; wrap and return.
			// The lister yields ctx.Err() once and then stops; treat it
			// as fatal here because the user asked to cancel.
			if errors.Is(walkErr, context.Canceled) || errors.Is(walkErr, context.DeadlineExceeded) {
				return nil, counting.Counts{}, nil, nil, fmt.Errorf("walk: %w", walkErr)
			}
			// Any other per-entry error goes into the error summary and
			// the walk continues (F6).
			aggErrs = append(aggErrs, walkErr)
			continue
		}

		// Binary detection policy (C10 / F12): decided here, not in the
		// lister. IsBinary errors are aggregated into the summary and the
		// file is skipped from counting. (F23 — binary check unchanged.)
		if !binary {
			isBin, err := f.IsBinary()
			if err != nil {
				aggErrs = append(aggErrs, fmt.Errorf("binary check %q: %w", f.RelPath, err))
				continue
			}
			if isBin {
				continue
			}
		}

		// Detect language once per file. The value is stored in a
		// per-iteration local so downstream consumers (Split call and the
		// filter gate below) can read it without a second Detect invocation.
		detectedLang := lang.Detect(f)

		// Lang-filter gate (F29, Decision 24): when --lang is set, skip
		// any file whose detected language is not in the wanted set.
		// LangUnknown ("") is implicitly excluded because it never matches
		// any non-empty filter value unless the user explicitly passes
		// --lang "" (which cobra's StringSliceVar rejects).
		if wantedLangs != nil {
			if _, ok := wantedLangs[detectedLang]; !ok {
				continue
			}
		}

		// Per-language split (Unit 5.3): open the file a second time to
		// classify its lines as blank/comment/code. Split errors are
		// aggregated but do not prevent byte/line/word/char counting (P4).
		var lineCounts lang.LineCounts
		if rc, openErr := f.Open(); openErr != nil {
			aggErrs = append(aggErrs, fmt.Errorf("split open %q: %w", f.RelPath, openErr))
		} else {
			var splitErr error
			lineCounts, splitErr = lang.Split(rc, detectedLang)
			_ = rc.Close()
			if splitErr != nil {
				aggErrs = append(aggErrs, fmt.Errorf("split %q: %w", f.RelPath, splitErr))
				// lineCounts is zero-value; counting continues below.
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
		byDirFiles[dir]++
		acceptedFiles++

		// --max-files safety rail (F45): abort when the accepted file count
		// reaches the limit. Partial results are discarded to avoid misleading
		// callers with an incomplete view of the tree.
		if maxFiles > 0 && acceptedFiles >= maxFiles {
			return nil, counting.Counts{}, nil, nil, fmt.Errorf("rak: file count exceeded --max-files %d: %w", maxFiles, ErrMaxFilesExceeded)
		}

		// Accumulate per-lang LangCounts for this directory (F30).
		if byDirLang[dir] == nil {
			byDirLang[dir] = map[lang.Language]lang.LangCounts{}
		}
		lc := byDirLang[dir][detectedLang]
		lc.Add(lang.LangCounts{Lines: lineCounts, Counts: fileCounts})
		byDirLang[dir][detectedLang] = lc

		// Accumulate per-lang LangCounts across all directories (F46).
		tlc := totalByLang[detectedLang]
		tlc.Add(lang.LangCounts{Lines: lineCounts, Counts: fileCounts})
		totalByLang[detectedLang] = tlc
	}

	dirs := make([]summary.Directory, 0, len(byDir))
	for p, c := range byDir {
		dirs = append(dirs, summary.Directory{Path: p, Counts: c, ByLang: byDirLang[p], Files: byDirFiles[p]})
	}

	return dirs, total, totalByLang, aggErrs, nil
}

// countFile opens f via the lister-reported handle, streams it through
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

// labelDirectories rewrites the leading "." path used by the lister into a
// user-facing root label so rendered titles read naturally when the user
// passed a positional path argument. A "." becomes exactly rootLabel; any
// "sub", "sub/nested" etc. becomes "<rootLabel>/<relative>". Passing an
// empty rootLabel returns the input unchanged, preserving the io/fs
// convention for test callers that want it.
//
// Files is propagated verbatim through the reconstruction (F44): failing to
// carry d.Files would cause --sort files to produce degenerate ordering
// (all zeros) and would silently omit the JSON "files" field via omitempty.
func labelDirectories(dirs []summary.Directory, rootLabel string) []summary.Directory {
	if rootLabel == "" {
		return dirs
	}
	out := make([]summary.Directory, len(dirs))
	for i, d := range dirs {
		if d.Path == "." {
			out[i] = summary.Directory{Path: rootLabel, Counts: d.Counts, ByLang: d.ByLang, Files: d.Files}
			continue
		}
		out[i] = summary.Directory{
			Path:   rootLabel + "/" + d.Path,
			Counts: d.Counts,
			ByLang: d.ByLang,
			Files:  d.Files,
		}
	}
	return out
}
