package render

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/evanmschultz/rak/internal/counting"
	"github.com/evanmschultz/rak/internal/lang"
	"github.com/evanmschultz/rak/internal/summary"
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
//
// ByLang mirrors summary.Directory.ByLang byte-for-byte (F34): the same
// field name and same Go type are required for the Go struct-type conversion
// directoryJSON(d) to compile. The json:"by_lang,omitempty" tag serializes
// to lowercase and omits the field entirely when the map is nil or empty
// (after LangUnknown filtering — see filterUnknown).
//
// Field declaration order — Path, Counts, ByLang, Files — must exactly
// match summary.Directory (F43); the bare struct conversion
// directoryJSON(filterUnknown(d)) requires both types to declare fields
// in the same order. Do not reorder. Files carries the per-directory
// accepted-file count; json:"files,omitempty" keeps zero-count directories
// invisible in existing snapshot tests.
type directoryJSON struct {
	Path   string                            `json:"path"`
	Counts counting.Counts                   `json:"counts"`
	ByLang map[lang.Language]lang.LangCounts `json:"by_lang,omitempty"`
	Files  int64                             `json:"files,omitempty"`
}

// filterUnknown returns a copy of d with lang.LangUnknown removed from ByLang.
// If the result is empty, ByLang is set to nil so omitempty suppresses the
// field in JSON output (F33 — LangUnknown suppression uniform across all
// renderers). The returned summary.Directory is a shallow copy; Counts and
// Files are not mutated.
//
// Files is propagated verbatim through the reconstruction (F44): failing to
// carry d.Files would cause the JSON "files" field to be silently omitted via
// omitempty and would degrade --sort files ordering in Unit 7.3.
func filterUnknown(d summary.Directory) summary.Directory {
	if len(d.ByLang) == 0 {
		return d
	}
	filtered := make(map[lang.Language]lang.LangCounts, len(d.ByLang))
	for k, v := range d.ByLang {
		if k != lang.LangUnknown {
			filtered[k] = v
		}
	}
	if len(filtered) == 0 {
		filtered = nil
	}
	return summary.Directory{
		Path:   d.Path,
		Counts: d.Counts,
		ByLang: filtered,
		Files:  d.Files,
	}
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
func (jsonRenderer) RenderTree(w io.Writer, dirs []summary.Directory, total counting.Counts, errs []error) error {
	payload := treeJSON{
		Directories: make([]directoryJSON, 0, len(dirs)),
		Total:       total,
	}
	for _, d := range dirs {
		payload.Directories = append(payload.Directories, directoryJSON(filterUnknown(d)))
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
