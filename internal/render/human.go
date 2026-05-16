package render

import (
	"fmt"
	"io"
	"sort"
	"strconv"

	"github.com/evanmschultz/laslig"

	"github.com/evanmschultz/rak/internal/counting"
	"github.com/evanmschultz/rak/internal/lang"
	"github.com/evanmschultz/rak/internal/summary"
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
// "total" KV block, an optional "total by language" section (when
// s.TotalByLang is non-empty after F33 LangUnknown suppression), and, when
// errs is non-empty, a laslig Notice summarizing the aggregated walker-level
// errors. The emitted block order exactly matches the caller-supplied s.Dirs
// slice — sorting is the caller's job before constructing s.
//
// When a directory's ByLang map is non-empty (after F33 LangUnknown
// suppression), each language gets one additional KV row appended under the
// directory block, sorted by language string for deterministic output.
func (h humanRenderer) RenderTree(w io.Writer, s summary.Summary, errs []error) error {
	printer := h.newPrinter(w)
	for _, d := range s.Dirs {
		if err := printer.KV(dirKV("dir: "+d.Path, d.Files, d.Counts)); err != nil {
			return fmt.Errorf("render directory %q as human kv block: %w", d.Path, err)
		}
		// Per F33: filter LangUnknown before emitting per-lang rows.
		if len(d.ByLang) > 0 {
			langs := sortedKnownLangs(d.ByLang)
			for _, l := range langs {
				lc := d.ByLang[l]
				if err := printer.KV(langKV(string(l), lc)); err != nil {
					return fmt.Errorf("render lang %q under dir %q: %w", l, d.Path, err)
				}
			}
		}
	}
	// Emit per-language grand totals before the grand total block (F33:
	// LangUnknown suppressed via sortedKnownLangs). total comes last so the
	// most-summary value is the final block in the output.
	knownTotalLangs := sortedKnownLangs(s.TotalByLang)
	for _, l := range knownTotalLangs {
		lc := s.TotalByLang[l]
		if err := printer.KV(totalLangKV(string(l), lc)); err != nil {
			return fmt.Errorf("render total lang %q as human kv block: %w", l, err)
		}
	}
	if err := printer.KV(countsKV("total", s.Total)); err != nil {
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

// countsKV builds the shared KV body used by both Render and the grand-total
// block in RenderTree. Title is empty for the single-stream case and carries a
// label ("total") for the tree case. It does NOT include a Files row because
// counting.Counts has no Files field; grand-total file counts are not in scope
// for v0.1.0 (a separate Summary.TotalFiles field is deferred to v0.2).
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

// dirKV builds a KV block for a single per-directory rollup. It prepends a
// "Files" row before the four count fields so the displayed order is:
// Files / Bytes / Lines / Words / Chars. This helper is distinct from
// countsKV so the grand-total block (which uses countsKV) never emits a
// Files row — summary.Summary.Total is counting.Counts and has no Files field.
func dirKV(title string, files int64, counts counting.Counts) laslig.KV {
	return laslig.KV{
		Title: title,
		Pairs: []laslig.Field{
			{Label: "Files", Value: strconv.FormatInt(files, 10)},
			{Label: "Bytes", Value: strconv.FormatInt(counts.Bytes, 10)},
			{Label: "Lines", Value: strconv.FormatInt(counts.Lines, 10)},
			{Label: "Words", Value: strconv.FormatInt(counts.Words, 10)},
			{Label: "Chars", Value: strconv.FormatInt(counts.Chars, 10)},
		},
	}
}

// langKV builds a per-language KV block showing blank/comment/code line
// counts plus raw byte/line/word/char totals. Title is "lang: <name>".
func langKV(name string, lc lang.LangCounts) laslig.KV {
	return laslig.KV{
		Title: "lang: " + name,
		Pairs: []laslig.Field{
			{Label: "Blank", Value: strconv.Itoa(lc.Lines.Blank)},
			{Label: "Comment", Value: strconv.Itoa(lc.Lines.Comment)},
			{Label: "Code", Value: strconv.Itoa(lc.Lines.Code)},
			{Label: "Bytes", Value: strconv.FormatInt(lc.Counts.Bytes, 10)},
			{Label: "Lines", Value: strconv.FormatInt(lc.Counts.Lines, 10)},
		},
	}
}

// totalLangKV builds a per-language KV block for the top-level
// "total by language" section. Title is "total lang: <name>" to distinguish
// it from the per-directory "lang: <name>" rows.
func totalLangKV(name string, lc lang.LangCounts) laslig.KV {
	return laslig.KV{
		Title: "total lang: " + name,
		Pairs: []laslig.Field{
			{Label: "Blank", Value: strconv.Itoa(lc.Lines.Blank)},
			{Label: "Comment", Value: strconv.Itoa(lc.Lines.Comment)},
			{Label: "Code", Value: strconv.Itoa(lc.Lines.Code)},
			{Label: "Bytes", Value: strconv.FormatInt(lc.Counts.Bytes, 10)},
			{Label: "Lines", Value: strconv.FormatInt(lc.Counts.Lines, 10)},
		},
	}
}

// sortedKnownLangs returns the keys of byLang in ascending string order,
// excluding lang.LangUnknown (F33 suppression).
func sortedKnownLangs(byLang map[lang.Language]lang.LangCounts) []lang.Language {
	out := make([]lang.Language, 0, len(byLang))
	for l := range byLang {
		if l != lang.LangUnknown {
			out = append(out, l)
		}
	}
	sort.Slice(out, func(i, j int) bool { return string(out[i]) < string(out[j]) })
	return out
}
