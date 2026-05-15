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
	"github.com/evanmschultz/rak/internal/summary"
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
// internal/; see F15 in DROP_3's PLAN.md for the pin. The RenderTree
// parameter type was updated from the provisional render.Directory to
// summary.Directory in Drop 7 Unit 7.2 (F37). In Drop 9 Unit 9.0 the
// signature was updated to accept a summary.Summary value, collapsing the
// separate dirs/total/TotalByLang params into one coherent type (F25/F32
// authorized — no external implementers).
type Renderer interface {
	// Render writes counts to w in the implementation's chosen format. It
	// returns any error surfaced by the underlying writer or formatter.
	Render(w io.Writer, counts counting.Counts) error

	// RenderTree writes a per-directory rollup (s.Dirs) plus a grand total
	// (s.Total), an optional per-language grand total (s.TotalByLang), and
	// an optional slice of aggregated walker-level errors (errs) that the
	// caller collected while walking. Implementations must emit directories
	// in the order the caller supplied inside s.Dirs; sorting is the
	// caller's responsibility before constructing s. Passing a nil or empty
	// errs slice suppresses any error-summary section in the rendered output.
	// s.TotalByLang nil or empty suppresses any per-language totals block;
	// F33 LangUnknown suppression is each implementation's responsibility.
	RenderTree(w io.Writer, s summary.Summary, errs []error) error
}
