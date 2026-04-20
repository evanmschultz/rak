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

// Renderer writes a counting.Counts value to the supplied io.Writer in one
// concrete output format. Implementations are obtained via the package-level
// constructors (NewHumanRenderer, NewJSONRenderer) rather than a factory
// taking a format enum, so each call site picks its representation
// explicitly.
type Renderer interface {
	// Render writes counts to w in the implementation's chosen format. It
	// returns any error surfaced by the underlying writer or formatter.
	Render(w io.Writer, counts counting.Counts) error
}
