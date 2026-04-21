// Package render is the rendering boundary for rak. It exposes a single
// Renderer interface with constructors for each supported output format —
// NewHumanRenderer for laslig-backed TTY-aware human output, NewJSONRenderer
// for stdlib encoding/json machine output — and no enum-keyed factory. The
// package imports internal/counting because every renderer ships a
// Render(w, counts) signature; the explicit constructor pattern avoids a
// premature Format enum and keeps the dependency DAG simple.
package render

import (
	"io"

	"github.com/evanmschultz/rak/internal/counting"
)

// Renderer writes counting.Counts values to the supplied io.Writer in one
// concrete output format. Implementations are obtained via the package-level
// constructors (NewHumanRenderer, NewJSONRenderer) rather than a factory
// taking a format enum, so each call site picks its representation
// explicitly.
//
// The interface exposes two methods: Render for a single counting.Counts
// value (the Drop 2 stdin path) and RenderTree for a per-directory rollup
// plus a grand total (the Drop 3 directory walk path). Growing the interface
// is acceptable pre-v1.0 because rak has no external implementers under
// internal/; see F15 in DROP_3's PLAN.md for the pin.
type Renderer interface {
	// Render writes counts to w in the implementation's chosen format. It
	// returns any error surfaced by the underlying writer or formatter.
	Render(w io.Writer, counts counting.Counts) error

	// RenderTree writes a per-directory rollup (dirs) plus a grand total
	// (total) and an optional slice of aggregated walker-level errors
	// (errs) that the caller collected while walking. Implementations must
	// emit the directories in the order the caller supplied; sorting is
	// the caller's responsibility. Passing a nil or empty errs slice
	// suppresses any error-summary section in the rendered output.
	RenderTree(w io.Writer, dirs []Directory, total counting.Counts, errs []error) error
}

// Directory pairs a walk-relative directory path with the accumulated
// counting.Counts for every non-skipped file under that directory. It is
// the minimal shape RenderTree needs today.
//
// Directory is PROVISIONAL: Drop 6.1 introduces the canonical
// internal/summary.Summary type and refactors both renderer implementations
// to consume it. Treat the Drop 3 shape as a stand-in, not a stable
// contract — no code outside this package plus cmd/rak should grow a
// dependency on Directory's field layout beyond what Unit 3.5 itself
// needs. See the C8 breadcrumb in DROP_3's PLAN.md.
type Directory struct {
	// Path is the walk-relative directory path using forward-slash
	// separators. The walk root itself is represented by the string "."
	// (matching the io/fs root convention used by cmd/rak's walker).
	Path string

	// Counts is the aggregated counting.Counts for every file under this
	// directory that survived the configured walk filters.
	Counts counting.Counts
}
