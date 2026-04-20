package render

import (
	"fmt"
	"io"
	"strconv"

	"github.com/evanmschultz/laslig"

	"github.com/evanmschultz/rak/internal/counting"
)

// humanRenderer renders a counting.Counts value as a laslig key-value block
// aligned for human reading. Production callers obtain one via
// NewHumanRenderer; tests obtain a mode-pinned variant via
// newHumanRendererWithMode to keep snapshot output independent of the
// ambient TTY / $COLUMNS / $TERM / $NO_COLOR / $CI environment.
type humanRenderer struct {
	// useExplicitMode toggles the printer construction path. When false,
	// Render uses laslig.New(w, policy) so laslig.ResolveMode auto-detects
	// TTY-vs-pipe against the actual writer at call time. When true, Render
	// uses laslig.NewWithMode(w, mode) which bypasses ResolveMode entirely.
	useExplicitMode bool
	// policy is used when useExplicitMode is false.
	policy laslig.Policy
	// mode is used when useExplicitMode is true.
	mode laslig.Mode
}

// NewHumanRenderer returns a Renderer that writes counts as a laslig
// key-value block. The laslig printer is constructed per Render call against
// the supplied writer, so laslig's ResolveMode auto-selects styled human
// output on a TTY and plain non-styled output on a pipe. Callers that need
// deterministic output across environments should use encoding/json via
// NewJSONRenderer instead.
func NewHumanRenderer() Renderer {
	return humanRenderer{
		useExplicitMode: false,
		policy: laslig.Policy{
			Format: laslig.FormatAuto,
			Style:  laslig.StyleAuto,
		},
	}
}

// newHumanRendererWithMode returns a Renderer that pins the laslig output
// mode explicitly, bypassing ResolveMode's environment inspection. It exists
// solely for snapshot tests in this package; the production code path is
// NewHumanRenderer.
func newHumanRendererWithMode(mode laslig.Mode) Renderer {
	return humanRenderer{
		useExplicitMode: true,
		mode:            mode,
	}
}

// Render writes counts as a laslig KV block with labels Bytes, Lines,
// Words, Chars. A fresh laslig printer is constructed per call bound to w so
// TTY detection runs against the real writer.
func (h humanRenderer) Render(w io.Writer, counts counting.Counts) error {
	var printer *laslig.Printer
	if h.useExplicitMode {
		printer = laslig.NewWithMode(w, h.mode)
	} else {
		printer = laslig.New(w, h.policy)
	}

	kv := laslig.KV{
		Pairs: []laslig.Field{
			{Label: "Bytes", Value: strconv.FormatInt(counts.Bytes, 10)},
			{Label: "Lines", Value: strconv.FormatInt(counts.Lines, 10)},
			{Label: "Words", Value: strconv.FormatInt(counts.Words, 10)},
			{Label: "Chars", Value: strconv.FormatInt(counts.Chars, 10)},
		},
	}
	if err := printer.KV(kv); err != nil {
		return fmt.Errorf("render counts as human kv block: %w", err)
	}
	return nil
}
