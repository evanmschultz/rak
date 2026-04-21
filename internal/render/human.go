package render

import (
	"fmt"
	"io"
	"strconv"

	"github.com/evanmschultz/laslig"

	"github.com/evanmschultz/rak/internal/counting"
)

// humanRenderer renders counting.Counts values as laslig key-value blocks
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
	printer := h.newPrinter(w)
	if err := printer.KV(countsKV("", counts)); err != nil {
		return fmt.Errorf("render counts as human kv block: %w", err)
	}
	return nil
}

// RenderTree writes one laslig KV block per directory followed by a final
// "total" KV block and, when errs is non-empty, a laslig Notice summarizing
// the aggregated walker-level errors. The emitted block order exactly
// matches the caller-supplied dirs slice — sorting is the caller's job.
func (h humanRenderer) RenderTree(w io.Writer, dirs []Directory, total counting.Counts, errs []error) error {
	printer := h.newPrinter(w)
	for _, d := range dirs {
		if err := printer.KV(countsKV("dir: "+d.Path, d.Counts)); err != nil {
			return fmt.Errorf("render directory %q as human kv block: %w", d.Path, err)
		}
	}
	if err := printer.KV(countsKV("total", total)); err != nil {
		return fmt.Errorf("render total as human kv block: %w", err)
	}
	if len(errs) > 0 {
		detail := make([]string, 0, len(errs))
		for _, e := range errs {
			detail = append(detail, e.Error())
		}
		notice := laslig.Notice{
			Level:  laslig.NoticeWarningLevel,
			Title:  "Errors",
			Detail: detail,
		}
		if err := printer.Notice(notice); err != nil {
			return fmt.Errorf("render walk errors as human notice: %w", err)
		}
	}
	return nil
}

// newPrinter builds a laslig Printer bound to w using whichever construction
// path the renderer was configured for (auto-policy for production,
// explicit-mode for snapshot tests).
func (h humanRenderer) newPrinter(w io.Writer) *laslig.Printer {
	if h.useExplicitMode {
		return laslig.NewWithMode(w, h.mode)
	}
	return laslig.New(w, h.policy)
}

// countsKV builds the shared KV body used by both Render and RenderTree.
// Title is empty for the single-stream case and carries a label (dir: ...
// or "total") for the tree case.
func countsKV(title string, counts counting.Counts) laslig.KV {
	return laslig.KV{
		Title: title,
		Pairs: []laslig.Field{
			{Label: "Bytes", Value: strconv.FormatInt(counts.Bytes, 10)},
			{Label: "Lines", Value: strconv.FormatInt(counts.Lines, 10)},
			{Label: "Words", Value: strconv.FormatInt(counts.Words, 10)},
			{Label: "Chars", Value: strconv.FormatInt(counts.Chars, 10)},
		},
	}
}
