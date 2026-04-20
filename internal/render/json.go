package render

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/evanmschultz/rak/internal/counting"
)

// jsonRenderer renders a counting.Counts value as one JSON object followed
// by a trailing newline, using stdlib encoding/json only — no laslig. Field
// order follows the Counts struct declaration (Bytes, Lines, Words, Chars)
// because Counts carries no json struct tags; Unit 2.1's struct shape is
// load-bearing for downstream snapshot tests.
type jsonRenderer struct{}

// NewJSONRenderer returns a Renderer that writes counts as stdlib
// encoding/json output. The output is one single-line JSON object terminated
// by a newline (json.Encoder.Encode trails with '\n'). Callers should not
// depend on any indentation or key ordering beyond the Counts struct's
// declared field order.
func NewJSONRenderer() Renderer {
	return jsonRenderer{}
}

// Render encodes counts as JSON to w. Any encoding error is wrapped with
// context so callers at cmd/rak can wrap once more at their boundary.
func (jsonRenderer) Render(w io.Writer, counts counting.Counts) error {
	if err := json.NewEncoder(w).Encode(counts); err != nil {
		return fmt.Errorf("render counts as json: %w", err)
	}
	return nil
}
