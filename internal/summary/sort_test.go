package summary

import (
	"testing"

	"github.com/evanmschultz/rak/internal/counting"
)

// tieRichDirs returns a slice of three Directory values that are fully tied on
// all three numeric axes (Lines=100, Files=5, Bytes=2048). The only
// distinguishing field is Path, which lets callers verify stable ordering by
// checking the original slice index sequence is preserved after sorting by a
// numeric key.
//
// The tie-richness across ALL three numeric axes is MANDATORY: a tie-free
// fixture would make the stability assertion silently vacuous.
func tieRichDirs() []Directory {
	counts := counting.Counts{Lines: 100, Bytes: 2048}
	return []Directory{
		{Path: "gamma/", Counts: counts, Files: 5},
		{Path: "alpha/", Counts: counts, Files: 5},
		{Path: "beta/", Counts: counts, Files: 5},
	}
}

// TestSortDirs_StableTieBreak verifies that SortDirs uses a stable sort:
// when multiple Directory entries compare equal on the chosen key, their
// original relative order is preserved. A tie-rich fixture (all three entries
// share identical Lines=100, Files=5, Bytes=2048) is mandatory — a tie-free
// fixture would make the stability assertion silently vacuous.
func TestSortDirs_StableTieBreak(t *testing.T) {
	t.Parallel()

	// numericKeys are the keys where all three fixture dirs are fully tied.
	numericKeys := []SortKey{SortLines, SortFiles, SortBytes}

	for _, key := range numericKeys {
		key := key // capture range variable for parallel subtest
		t.Run(string(key), func(t *testing.T) {
			t.Parallel()

			dirs := tieRichDirs()
			// Record the original order by Path before sorting.
			want := make([]string, len(dirs))
			for i, d := range dirs {
				want[i] = d.Path
			}

			SortDirs(dirs, key, false) // asc=false → descending (default for numeric keys)

			// All entries tie on the numeric key, so the comparator always
			// returns 0. A stable sort must preserve the original input order.
			for i, d := range dirs {
				if d.Path != want[i] {
					t.Errorf("key=%s: index %d has path %q, want %q (stable order broken)",
						key, i, d.Path, want[i])
				}
			}
		})
	}

	// path sort: entries have distinct paths, so the output is fully determined
	// by lexicographic order (ascending, since SortPath defaults ascending when
	// asc=false, because effectiveAsc returns !asc = true for SortPath).
	t.Run("path", func(t *testing.T) {
		t.Parallel()

		dirs := tieRichDirs()
		SortDirs(dirs, SortPath, false) // asc=false → effectiveAsc true → ascending A→Z

		wantPaths := []string{"alpha/", "beta/", "gamma/"}
		for i, d := range dirs {
			if d.Path != wantPaths[i] {
				t.Errorf("key=path: index %d got %q, want %q", i, d.Path, wantPaths[i])
			}
		}
	})
}
