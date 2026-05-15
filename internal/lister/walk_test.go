package lister

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/evanmschultz/rak/internal/fileset"
)

// compile-time assertion: WalkLister must satisfy FileLister.
var _ FileLister = (*WalkLister)(nil)

// collectWalk iterates all entries from wl.List(ctx) and returns the RelPaths
// of successfully emitted files and any errors encountered. It is a test helper
// only — not part of the production API.
func collectWalk(t *testing.T, ctx context.Context, wl *WalkLister) ([]string, []error) {
	t.Helper()
	var paths []string
	var errs []error
	for f, err := range wl.List(ctx) {
		if err != nil {
			errs = append(errs, err)
			continue
		}
		paths = append(paths, f.RelPath)
	}
	return paths, errs
}

// TestWalkLister_EmptyFS verifies that WalkLister emits nothing for an empty
// MapFS.
func TestWalkLister_EmptyFS(t *testing.T) {
	fsys := fstest.MapFS{}
	wl := newWalkLister(fsys, ".", fileset.WalkOptions{})
	paths, errs := collectWalk(t, context.Background(), wl)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(paths) != 0 {
		t.Errorf("expected no files, got %v", paths)
	}
}

// TestWalkLister_FlatFiles verifies that WalkLister yields exactly two text
// files at the root with their correct RelPaths.
func TestWalkLister_FlatFiles(t *testing.T) {
	fsys := fstest.MapFS{
		"alpha.txt": {Data: []byte("hello")},
		"beta.txt":  {Data: []byte("world")},
	}
	wl := newWalkLister(fsys, ".", fileset.WalkOptions{})
	paths, errs := collectWalk(t, context.Background(), wl)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	got := make(map[string]bool, len(paths))
	for _, p := range paths {
		got[p] = true
	}
	for _, want := range []string{"alpha.txt", "beta.txt"} {
		if !got[want] {
			t.Errorf("expected %q in results, got %v", want, paths)
		}
	}
	if len(paths) != 2 {
		t.Errorf("expected exactly 2 files, got %d: %v", len(paths), paths)
	}
}

// TestWalkLister_DepthFilter verifies that WalkOptions.Depth=1 yields only the
// file at depth 0 (directly in the root), skipping files at depth 1 and 2.
func TestWalkLister_DepthFilter(t *testing.T) {
	fsys := fstest.MapFS{
		"root.txt":         {Data: []byte("depth0")},
		"sub/one.txt":      {Data: []byte("depth1")},
		"sub/deep/two.txt": {Data: []byte("depth2")},
	}
	opts := fileset.WalkOptions{Depth: 1}
	wl := newWalkLister(fsys, ".", opts)
	paths, errs := collectWalk(t, context.Background(), wl)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(paths) != 1 {
		t.Errorf("expected 1 file with Depth=1, got %d: %v", len(paths), paths)
	}
	if len(paths) == 1 && paths[0] != "root.txt" {
		t.Errorf("expected \"root.txt\", got %q", paths[0])
	}
}

// TestWalkLister_HiddenFilter verifies that hidden files are excluded by
// default and included when IncludeHidden is true.
func TestWalkLister_HiddenFilter(t *testing.T) {
	fsys := fstest.MapFS{
		"visible.txt": {Data: []byte("visible")},
		".hidden.txt": {Data: []byte("hidden")},
	}

	t.Run("default_excludes_hidden", func(t *testing.T) {
		wl := newWalkLister(fsys, ".", fileset.WalkOptions{})
		paths, errs := collectWalk(t, context.Background(), wl)
		if len(errs) != 0 {
			t.Fatalf("unexpected errors: %v", errs)
		}
		for _, p := range paths {
			if strings.HasPrefix(filepath.Base(p), ".") {
				t.Errorf("hidden file %q should not be emitted by default", p)
			}
		}
		if len(paths) != 1 || paths[0] != "visible.txt" {
			t.Errorf("expected only [visible.txt], got %v", paths)
		}
	})

	t.Run("include_hidden", func(t *testing.T) {
		wl := newWalkLister(fsys, ".", fileset.WalkOptions{IncludeHidden: true})
		paths, errs := collectWalk(t, context.Background(), wl)
		if len(errs) != 0 {
			t.Fatalf("unexpected errors: %v", errs)
		}
		got := make(map[string]bool, len(paths))
		for _, p := range paths {
			got[p] = true
		}
		if !got[".hidden.txt"] {
			t.Errorf("expected .hidden.txt with IncludeHidden=true, got %v", paths)
		}
	})
}

// TestWalkLister_ImplementsFileLister is a compile-time assertion that
// *WalkLister satisfies the FileLister interface. The package-level var above
// provides the same check; this test makes it explicit and named for QA.
func TestWalkLister_ImplementsFileLister(t *testing.T) {
	var _ FileLister = (*WalkLister)(nil)
}

// TestWalkLister_RelPathInvariant enforces F26: every emitted *fileset.File
// has a walk-root-relative RelPath that contains no leading "./" or "/",
// and uses forward-slash separators on all platforms.
func TestWalkLister_RelPathInvariant(t *testing.T) {
	fsys := fstest.MapFS{
		"a.txt":          {Data: []byte("a")},
		"sub/b.txt":      {Data: []byte("b")},
		"sub/deep/c.txt": {Data: []byte("c")},
	}
	wl := NewWalkLister(fsys, ".", fileset.WalkOptions{})
	for f, err := range wl.List(context.Background()) {
		if err != nil {
			t.Fatalf("unexpected error during iteration: %v", err)
		}
		rp := f.RelPath
		if strings.HasPrefix(rp, "./") {
			t.Errorf("RelPath %q has leading \"./\" (F26 violation)", rp)
		}
		if strings.HasPrefix(rp, "/") {
			t.Errorf("RelPath %q has leading \"/\" (F26 violation)", rp)
		}
		if rp != filepath.ToSlash(rp) {
			t.Errorf("RelPath %q contains non-forward-slash separators (F26 violation)", rp)
		}
	}
}
