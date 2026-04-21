package render

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/evanmschultz/rak/internal/counting"
)

// jsonRenderer renders counting.Counts values as stdlib encoding/json output
// — no laslig. For the single-stream Render path, field order follows the
// Counts struct declaration (Bytes, Lines, Words, Chars) because Counts
// carries no json struct tags; Unit 2.1's struct shape is load-bearing for
// downstream snapshot tests. For the RenderTree path the outer envelope has
// fields directories, total, and optionally errors.
type jsonRenderer struct{}

// NewJSONRenderer returns a Renderer that writes counts as stdlib
// encoding/json output. The output is one single-line JSON object
// terminated by a newline (json.Encoder.Encode trails with '\n'). Callers
// should not depend on any indentation or key ordering beyond the Counts
// struct's declared field order.
func NewJSONRenderer() Renderer {
	return jsonRenderer{}
}

// Render encodes a single counting.Counts value as JSON to w. Any encoding
// error is wrapped with context so callers at cmd/rak can wrap once more at
// their boundary.
func (jsonRenderer) Render(w io.Writer, counts counting.Counts) error {
	if err := json.NewEncoder(w).Encode(counts); err != nil {
		return fmt.Errorf("render counts as json: %w", err)
	}
	return nil
}

// directoryJSON is the JSON shape for a single per-directory rollup. The
// field tags pin lowercase keys so the wire format does not leak Go's
// exported-identifier capitalization. "counts" embeds the unmodified
// counting.Counts shape so the per-dir block matches the single-stream
// Render output byte-for-byte at the counts boundary.
type directoryJSON struct {
	Path   string          `json:"path"`
	Counts counting.Counts `json:"counts"`
}

// treeJSON is the top-level envelope for RenderTree. Errors is omitted
// entirely (via omitempty) when the caller passes a nil / empty slice so
// the common no-errors case emits a clean two-field object.
type treeJSON struct {
	Directories []directoryJSON `json:"directories"`
	Total       counting.Counts `json:"total"`
	Errors      []string        `json:"errors,omitempty"`
}

// RenderTree encodes the per-directory rollup plus grand total as a JSON
// object with keys "directories", "total", and (when errs is non-empty)
// "errors". The emitted directories slice preserves the caller-supplied
// order; callers are responsible for sorting.
func (jsonRenderer) RenderTree(w io.Writer, dirs []Directory, total counting.Counts, errs []error) error {
	payload := treeJSON{
		Directories: make([]directoryJSON, 0, len(dirs)),
		Total:       total,
	}
	for _, d := range dirs {
		payload.Directories = append(payload.Directories, directoryJSON(d))
	}
	if len(errs) > 0 {
		payload.Errors = make([]string, 0, len(errs))
		for _, e := range errs {
			payload.Errors = append(payload.Errors, e.Error())
		}
	}
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		return fmt.Errorf("render tree as json: %w", err)
	}
	return nil
}
