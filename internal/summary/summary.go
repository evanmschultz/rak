// Package summary provides the canonical per-directory rollup types for rak.
// It is the home of Directory, Summary, SortKey, and SortDirs — the data layer
// that the render package consumes and cmd/rak populates during a walk.
//
// Import graph: summary imports internal/counting and internal/lang only.
// No other internal packages are imported here.
package summary

import (
	"github.com/evanmschultz/rak/internal/counting"
	"github.com/evanmschultz/rak/internal/lang"
)

// Directory holds the aggregated counts for a single directory observed during
// a walk. Field declaration order — Path, Counts, ByLang, Files — is
// load-bearing: internal/render/json.go uses a bare struct conversion
// (directoryJSON(d)) that requires both types to declare fields in the same
// order. Do not reorder.
type Directory struct {
	// Path is the directory path, using forward slashes (F26 RelPath
	// invariant). The root directory is rendered as a labelled path by
	// cmd/rak/root.go's labelDirectories helper; this field stores the raw
	// walk-relative value before labelling.
	Path string

	// Counts holds the aggregate byte/line/word/char totals across all
	// accepted files in this directory.
	Counts counting.Counts

	// ByLang maps each detected Language to its accumulated line-split and
	// byte/line/word/char counts. LangUnknown may be present; renderers are
	// responsible for suppressing it on output (F33).
	ByLang map[lang.Language]lang.LangCounts

	// Files is the count of accepted (non-skipped) files in this directory.
	// "Accepted" means the file passed the binary-skip, --lang, --include,
	// and --exclude filters. Used by the "files" sort key (F42).
	Files int64
}

// Summary is the top-level rollup container produced by a completed walk. It
// holds the per-directory breakdown, a grand-total Counts aggregated across
// all accepted files in all directories, and a per-language grand-total map
// collapsed across all directories.
type Summary struct {
	// Dirs holds one Directory entry per distinct directory observed during
	// the walk. The ordering of Dirs is determined by the caller (typically
	// via SortDirs before handing to a Renderer).
	Dirs []Directory

	// Total is the aggregate byte/line/word/char totals across every accepted
	// file in every directory.
	Total counting.Counts

	// TotalByLang is the per-language aggregate collapsed across all accepted
	// files in all directories. It mirrors the per-directory Directory.ByLang
	// field but rolled up to the walk level. A nil map means no language
	// detection data was collected (e.g. all files were LangUnknown). Renderers
	// apply F33 LangUnknown suppression before emitting this field; it is
	// their responsibility, not the walk's.
	TotalByLang map[lang.Language]lang.LangCounts
}
