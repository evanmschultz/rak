package render

import (
	"fmt"
	"io"
	"sort"

	"github.com/toon-format/toon-go"

	"github.com/evanmschultz/rak/internal/counting"
	"github.com/evanmschultz/rak/internal/summary"
)

// toonRenderer renders counting.Counts values as TOON (Token-Oriented Object
// Notation) output using github.com/toon-format/toon-go. It is a zero-allocation
// value type whose marshaling options are fixed at construction: pipe delimiter
// for both document and array fields (F20). Production callers obtain one via
// NewTOONRenderer.
type toonRenderer struct{}

// NewTOONRenderer returns a Renderer that writes counts in TOON format using
// pipe as the field delimiter. TOON is the default output format for rak
// (decision 33) — compact, human-readable, and token-efficient for LLM
// consumption.
func NewTOONRenderer() Renderer {
	return toonRenderer{}
}

// toonCounts is the TOON document shape for a single counting.Counts value.
// Field names are lowercase per TOON convention; field types are int64 to
// match counting.Counts exactly.
type toonCounts struct {
	Bytes int64 `toon:"bytes"`
	Lines int64 `toon:"lines"`
	Words int64 `toon:"words"`
	Chars int64 `toon:"chars"`
}

// toonDirectory is a single row in the directories array for RenderTree
// output. It combines the walk-relative path with the four count fields so
// the tabular array has all five columns per row.
type toonDirectory struct {
	Path  string `toon:"path"`
	Bytes int64  `toon:"bytes"`
	Lines int64  `toon:"lines"`
	Words int64  `toon:"words"`
	Chars int64  `toon:"chars"`
}

// toonLangRow is a single per-language detail row emitted under a directory
// in the per-lang section of TOON RenderTree output. It carries the language
// name plus the three-way line split and raw counts for that language bucket.
// LangUnknown rows are suppressed before construction (F33).
type toonLangRow struct {
	Dir     string `toon:"dir"`
	Lang    string `toon:"lang"`
	Blank   int    `toon:"blank"`
	Comment int    `toon:"comment"`
	Code    int    `toon:"code"`
	Bytes   int64  `toon:"bytes"`
	Lines   int64  `toon:"lines"`
}

// toonTree is the top-level envelope for RenderTree. The directories field is
// a tabular TOON array; the total field is a nested toonCounts block carrying
// the grand total (spike-confirmed: toon-go emits struct-in-struct as an
// indented nested block — F20 nested-total contract satisfied); by_lang is a
// tabular TOON array of per-directory/per-language rows, omitted when empty
// (F33 — LangUnknown entries never appear); errors is omitted entirely (via
// omitempty) when the caller passes a nil or empty errs slice — spike-confirmed:
// toon-go omitempty drops zero/empty fields from output (C7).
type toonTree struct {
	Directories []toonDirectory `toon:"directories"`
	Total       toonCounts      `toon:"total"`
	ByLang      []toonLangRow   `toon:"by_lang,omitempty"`
	Errors      []string        `toon:"errors,omitempty"`
}

// Render marshals a single counting.Counts value as a TOON document to w.
// The output uses pipe as the document delimiter (F20). Any marshal or write
// error is wrapped with context so cmd/rak can add its own layer.
func (toonRenderer) Render(w io.Writer, counts counting.Counts) error {
	v := toonCounts{
		Bytes: counts.Bytes,
		Lines: counts.Lines,
		Words: counts.Words,
		Chars: counts.Chars,
	}
	b, err := toon.Marshal(
		v,
		toon.WithDocumentDelimiter(toon.DelimiterPipe),
	)
	if err != nil {
		return fmt.Errorf("render counts as toon: %w", err)
	}
	if _, err := w.Write(b); err != nil {
		return fmt.Errorf("render counts as toon: %w", err)
	}
	return nil
}

// RenderTree marshals a per-directory rollup plus a grand total and optional
// errors as a TOON document to w. The directories slice is emitted as a
// tabular TOON array (pipe-delimited columns — F20); the grand total is
// emitted as a nested "total" block (toonCounts — F20 nested-total contract);
// by_lang is emitted as a tabular array of per-directory/per-language rows
// when any directory carries non-unknown language data (F33); errors are
// omitted when the caller passes nil or an empty slice. The emitted directory
// order exactly matches the caller-supplied dirs slice; sorting is the
// caller's responsibility.
func (toonRenderer) RenderTree(w io.Writer, dirs []summary.Directory, total counting.Counts, errs []error) error {
	rows := make([]toonDirectory, 0, len(dirs))
	for _, d := range dirs {
		rows = append(rows, toonDirectory{
			Path:  d.Path,
			Bytes: d.Counts.Bytes,
			Lines: d.Counts.Lines,
			Words: d.Counts.Words,
			Chars: d.Counts.Chars,
		})
	}

	// Build per-lang rows across all directories, sorted by dir then lang
	// (F33: LangUnknown suppressed). The by_lang field is omitted entirely
	// via omitempty when the resulting slice is empty.
	var langRows []toonLangRow
	for _, d := range dirs {
		knownLangs := sortedKnownLangs(d.ByLang)
		for _, l := range knownLangs {
			lc := d.ByLang[l]
			langRows = append(langRows, toonLangRow{
				Dir:     d.Path,
				Lang:    string(l),
				Blank:   lc.Lines.Blank,
				Comment: lc.Lines.Comment,
				Code:    lc.Lines.Code,
				Bytes:   lc.Counts.Bytes,
				Lines:   lc.Counts.Lines,
			})
		}
	}

	// Sort lang rows by dir then lang for deterministic output.
	sort.Slice(langRows, func(i, j int) bool {
		if langRows[i].Dir != langRows[j].Dir {
			return langRows[i].Dir < langRows[j].Dir
		}
		return langRows[i].Lang < langRows[j].Lang
	})

	payload := toonTree{
		Directories: rows,
		Total: toonCounts{
			Bytes: total.Bytes,
			Lines: total.Lines,
			Words: total.Words,
			Chars: total.Chars,
		},
		ByLang: langRows,
	}
	if len(errs) > 0 {
		msgs := make([]string, 0, len(errs))
		for _, e := range errs {
			msgs = append(msgs, e.Error())
		}
		payload.Errors = msgs
	}
	b, err := toon.Marshal(
		payload,
		toon.WithDocumentDelimiter(toon.DelimiterPipe),
		toon.WithArrayDelimiter(toon.DelimiterPipe),
	)
	if err != nil {
		return fmt.Errorf("render tree as toon: %w", err)
	}
	if _, err := w.Write(b); err != nil {
		return fmt.Errorf("render tree as toon: %w", err)
	}
	return nil
}
