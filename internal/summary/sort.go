package summary

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
)

// SortKey identifies the field by which a []Directory slice is sorted.
// Valid keys in v0.1.0 are: SortLines, SortFiles, SortBytes, SortPath.
//
// "tokens" is intentionally absent: token counting is scoped to v0.2.0
// (Decision 30 / F41). Future drops re-add it as SortTokens = "tokens".
// Passing an unrecognized SortKey to SortDirs panics — the CLI layer
// validates the key before calling SortDirs (Unit 7.3 PersistentPreRunE).
type SortKey string

const (
	// SortLines sorts directories by their aggregate line count.
	SortLines SortKey = "lines"

	// SortFiles sorts directories by their per-directory accepted-file count
	// (F42).
	SortFiles SortKey = "files"

	// SortBytes sorts directories by their aggregate byte count.
	SortBytes SortKey = "bytes"

	// SortPath sorts directories lexicographically by their Path field.
	// Note: the constant name is SortPath, NOT SortName.
	SortPath SortKey = "path"
)

// effectiveAsc resolves the effective sort direction from the raw --sort-asc
// flag value (asc) and the key's natural default direction:
//
//   - Numeric keys (SortLines, SortFiles, SortBytes) default descending.
//     effectiveAsc returns asc unchanged: false → desc, true → asc.
//   - SortPath defaults ascending (A→Z, matching ls convention).
//     effectiveAsc returns !asc: false → asc (ascending), true → desc.
//
// Unknown keys are not handled here; SortDirs panics before calling this.
func effectiveAsc(key SortKey, asc bool) bool {
	if key == SortPath {
		return !asc
	}
	return asc
}

// SortDirs sorts dirs in place by key, honoring key-specific default
// directions. It does not return a copy — the slice is modified directly.
//
// Direction semantics (asc is the raw --sort-asc flag value):
//   - Numeric keys (lines, files, bytes) default descending; asc=true flips
//     to ascending.
//   - path defaults ascending (A→Z); asc=true flips to descending.
//
// If key is not one of the four recognised SortKey constants, SortDirs panics
// with a descriptive message. The CLI layer (Unit 7.3 PersistentPreRunE) is
// expected to reject unrecognized keys before this point, so a panic here
// surfaces programming errors only.
//
// SortDirs uses slices.SortFunc (stdlib, Go 1.21+) for the in-place sort.
func SortDirs(dirs []Directory, key SortKey, asc bool) {
	switch key {
	case SortLines, SortFiles, SortBytes, SortPath:
		// Recognised key — fall through to sort below.
	default:
		panic(fmt.Sprintf("summary: SortDirs called with unrecognized SortKey %q", key))
	}

	eff := effectiveAsc(key, asc)

	slices.SortFunc(dirs, func(a, b Directory) int {
		var result int
		switch key {
		case SortLines:
			result = cmp.Compare(a.Counts.Lines, b.Counts.Lines)
		case SortFiles:
			result = cmp.Compare(a.Files, b.Files)
		case SortBytes:
			result = cmp.Compare(a.Counts.Bytes, b.Counts.Bytes)
		case SortPath:
			result = strings.Compare(a.Path, b.Path)
		}
		if !eff {
			// Descending: negate the comparator result.
			return -result
		}
		return result
	})
}
