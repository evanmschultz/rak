package summary

import (
	"testing"

	"github.com/evanmschultz/rak/internal/counting"
)

// makeDir is a test helper that constructs a Directory with the given path,
// Lines count, Files count, and Bytes count. ByLang is left nil — sort tests
// do not need language breakdown.
func makeDir(path string, lines, files, bytes int64) Directory {
	return Directory{
		Path: path,
		Counts: counting.Counts{
			Lines: lines,
			Bytes: bytes,
		},
		Files: files,
	}
}

// TestSortDirs_Lines_Default verifies that three directories with Lines
// 10/5/20 sorted by SortLines with asc=false are returned in descending order
// [20, 10, 5].
func TestSortDirs_Lines_Default(t *testing.T) {
	dirs := []Directory{
		makeDir("a", 10, 0, 0),
		makeDir("b", 5, 0, 0),
		makeDir("c", 20, 0, 0),
	}
	SortDirs(dirs, SortLines, false)
	want := []int64{20, 10, 5}
	for i, w := range want {
		if dirs[i].Counts.Lines != w {
			t.Errorf("index %d: got Lines=%d, want %d", i, dirs[i].Counts.Lines, w)
		}
	}
}

// TestSortDirs_Lines_Asc verifies that SortLines with asc=true returns
// directories in ascending order [5, 10, 20].
func TestSortDirs_Lines_Asc(t *testing.T) {
	dirs := []Directory{
		makeDir("a", 10, 0, 0),
		makeDir("b", 5, 0, 0),
		makeDir("c", 20, 0, 0),
	}
	SortDirs(dirs, SortLines, true)
	want := []int64{5, 10, 20}
	for i, w := range want {
		if dirs[i].Counts.Lines != w {
			t.Errorf("index %d: got Lines=%d, want %d", i, dirs[i].Counts.Lines, w)
		}
	}
}

// TestSortDirs_Files_Default verifies that three directories with Files
// 1/3/2 sorted by SortFiles with asc=false are returned in descending order
// [3, 2, 1].
func TestSortDirs_Files_Default(t *testing.T) {
	dirs := []Directory{
		makeDir("a", 0, 1, 0),
		makeDir("b", 0, 3, 0),
		makeDir("c", 0, 2, 0),
	}
	SortDirs(dirs, SortFiles, false)
	want := []int64{3, 2, 1}
	for i, w := range want {
		if dirs[i].Files != w {
			t.Errorf("index %d: got Files=%d, want %d", i, dirs[i].Files, w)
		}
	}
}

// TestSortDirs_Files_Asc verifies that SortFiles with asc=true returns
// directories in ascending order [1, 2, 3].
func TestSortDirs_Files_Asc(t *testing.T) {
	dirs := []Directory{
		makeDir("a", 0, 1, 0),
		makeDir("b", 0, 3, 0),
		makeDir("c", 0, 2, 0),
	}
	SortDirs(dirs, SortFiles, true)
	want := []int64{1, 2, 3}
	for i, w := range want {
		if dirs[i].Files != w {
			t.Errorf("index %d: got Files=%d, want %d", i, dirs[i].Files, w)
		}
	}
}

// TestSortDirs_Bytes_Default verifies that three directories with Bytes
// 100/300/200 sorted by SortBytes with asc=false are returned in descending
// order [300, 200, 100].
func TestSortDirs_Bytes_Default(t *testing.T) {
	dirs := []Directory{
		makeDir("a", 0, 0, 100),
		makeDir("b", 0, 0, 300),
		makeDir("c", 0, 0, 200),
	}
	SortDirs(dirs, SortBytes, false)
	want := []int64{300, 200, 100}
	for i, w := range want {
		if dirs[i].Counts.Bytes != w {
			t.Errorf("index %d: got Bytes=%d, want %d", i, dirs[i].Counts.Bytes, w)
		}
	}
}

// TestSortDirs_Bytes_Asc verifies that SortBytes with asc=true returns
// directories in ascending order [100, 200, 300].
func TestSortDirs_Bytes_Asc(t *testing.T) {
	dirs := []Directory{
		makeDir("a", 0, 0, 100),
		makeDir("b", 0, 0, 300),
		makeDir("c", 0, 0, 200),
	}
	SortDirs(dirs, SortBytes, true)
	want := []int64{100, 200, 300}
	for i, w := range want {
		if dirs[i].Counts.Bytes != w {
			t.Errorf("index %d: got Bytes=%d, want %d", i, dirs[i].Counts.Bytes, w)
		}
	}
}

// TestSortDirs_Path_Default verifies that three directories with paths "c",
// "a", "b" sorted by SortPath with asc=false return in ascending order
// ["a", "b", "c"] — the natural default for path is ascending (ls convention).
func TestSortDirs_Path_Default(t *testing.T) {
	dirs := []Directory{
		makeDir("c", 0, 0, 0),
		makeDir("a", 0, 0, 0),
		makeDir("b", 0, 0, 0),
	}
	SortDirs(dirs, SortPath, false)
	want := []string{"a", "b", "c"}
	for i, w := range want {
		if dirs[i].Path != w {
			t.Errorf("index %d: got Path=%q, want %q", i, dirs[i].Path, w)
		}
	}
}

// TestSortDirs_Path_Asc verifies that SortPath with asc=true returns
// directories in descending order ["c", "b", "a"] — asc=true flips the
// path key's natural ascending default.
func TestSortDirs_Path_Asc(t *testing.T) {
	dirs := []Directory{
		makeDir("c", 0, 0, 0),
		makeDir("a", 0, 0, 0),
		makeDir("b", 0, 0, 0),
	}
	SortDirs(dirs, SortPath, true)
	want := []string{"c", "b", "a"}
	for i, w := range want {
		if dirs[i].Path != w {
			t.Errorf("index %d: got Path=%q, want %q", i, dirs[i].Path, w)
		}
	}
}

// TestSortDirs_UnknownKey_Panics verifies that SortDirs panics when called
// with an unrecognized SortKey (e.g. "tokens", reserved for v0.2.0).
func TestSortDirs_UnknownKey_Panics(t *testing.T) {
	dirs := []Directory{makeDir("a", 1, 1, 1)}
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected SortDirs to panic on unknown key, but it did not")
		}
	}()
	SortDirs(dirs, SortKey("tokens"), false)
}

// TestSortDirs_EmptySlice verifies that SortDirs does not panic on a
// zero-length slice.
func TestSortDirs_EmptySlice(t *testing.T) {
	var dirs []Directory
	// Must not panic for any valid key.
	SortDirs(dirs, SortLines, false)
	SortDirs(dirs, SortFiles, true)
	SortDirs(dirs, SortBytes, false)
	SortDirs(dirs, SortPath, false)
}

// TestSortDirs_SingleEntry verifies that SortDirs does not panic on a
// single-element slice and leaves the slice unchanged.
func TestSortDirs_SingleEntry(t *testing.T) {
	dirs := []Directory{makeDir("only", 42, 7, 1024)}
	SortDirs(dirs, SortLines, false)
	if dirs[0].Path != "only" {
		t.Errorf("single-entry sort changed path: got %q, want %q", dirs[0].Path, "only")
	}
	if dirs[0].Counts.Lines != 42 {
		t.Errorf("single-entry sort changed Lines: got %d, want 42", dirs[0].Counts.Lines)
	}
}
