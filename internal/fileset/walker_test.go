package fileset

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"slices"
	"strings"
	"testing"
	"testing/fstest"
)

// collect drains a walk into slices for assertion. It deliberately
// stores per-entry errors alongside paths rather than failing the test
// so tests can assert on the yielded error sequence.
type walkResult struct {
	paths []string
	errs  []error
}

func collect(t *testing.T, w *Walker) walkResult {
	t.Helper()
	return collectCtx(t, w, context.Background())
}

func collectCtx(t *testing.T, w *Walker, ctx context.Context) walkResult {
	t.Helper()
	var res walkResult
	for f, err := range w.Walk(ctx) {
		if err != nil {
			res.errs = append(res.errs, err)
			continue
		}
		res.paths = append(res.paths, f.RelPath)
	}
	return res
}

func TestWalker_EmptyRoot(t *testing.T) {
	t.Parallel()

	// An empty MapFS with the root explicitly marked as a directory so
	// fs.WalkDir has somewhere to start. No entries beneath it.
	fsys := fstest.MapFS{
		".": &fstest.MapFile{Mode: fs.ModeDir},
	}

	w := NewWalker(fsys, ".", WalkOptions{})
	got := collect(t, w)

	if len(got.errs) != 0 {
		t.Fatalf("errs = %v, want none", got.errs)
	}
	if len(got.paths) != 0 {
		t.Errorf("paths = %v, want none", got.paths)
	}
}

func TestWalker_SingleFile(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		"a.txt": &fstest.MapFile{Data: []byte("a")},
	}

	w := NewWalker(fsys, ".", WalkOptions{})
	got := collect(t, w)

	if len(got.errs) != 0 {
		t.Fatalf("errs = %v, want none", got.errs)
	}
	if len(got.paths) != 1 || got.paths[0] != "a.txt" {
		t.Errorf("paths = %v, want [a.txt]", got.paths)
	}
}

func TestWalker_NestedTree(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		"a.txt":             &fstest.MapFile{Data: []byte("a")},
		"sub/b.txt":         &fstest.MapFile{Data: []byte("b")},
		"sub/deep/c.txt":    &fstest.MapFile{Data: []byte("c")},
		"sub/deep/inner.md": &fstest.MapFile{Data: []byte("inner")},
	}

	w := NewWalker(fsys, ".", WalkOptions{})
	got := collect(t, w)

	if len(got.errs) != 0 {
		t.Fatalf("errs = %v, want none", got.errs)
	}

	want := []string{
		"a.txt",
		"sub/b.txt",
		"sub/deep/c.txt",
		"sub/deep/inner.md",
	}
	slices.Sort(got.paths)
	slices.Sort(want)
	if !slices.Equal(got.paths, want) {
		t.Errorf("paths = %v, want %v", got.paths, want)
	}
}

func TestWalker_DepthLimit(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		"root.txt":           &fstest.MapFile{Data: []byte("root")},
		"sub/mid.txt":        &fstest.MapFile{Data: []byte("mid")},
		"sub/deep/inner.txt": &fstest.MapFile{Data: []byte("inner")},
	}

	tests := []struct {
		name  string
		depth int
		want  []string
	}{
		{
			name:  "depth_zero_unlimited",
			depth: 0,
			want:  []string{"root.txt", "sub/mid.txt", "sub/deep/inner.txt"},
		},
		{
			name:  "depth_one_root_only",
			depth: 1,
			want:  []string{"root.txt"},
		},
		{
			name:  "depth_two_root_plus_one_level",
			depth: 2,
			want:  []string{"root.txt", "sub/mid.txt"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			w := NewWalker(fsys, ".", WalkOptions{Depth: tc.depth})
			got := collect(t, w)

			if len(got.errs) != 0 {
				t.Fatalf("errs = %v, want none", got.errs)
			}

			gotPaths := append([]string(nil), got.paths...)
			slices.Sort(gotPaths)
			wantPaths := append([]string(nil), tc.want...)
			slices.Sort(wantPaths)
			if !slices.Equal(gotPaths, wantPaths) {
				t.Errorf("depth=%d paths = %v, want %v", tc.depth, gotPaths, wantPaths)
			}
		})
	}
}

func TestWalker_SkipsHidden(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		"visible.txt":  &fstest.MapFile{Data: []byte("v")},
		".hidden.txt":  &fstest.MapFile{Data: []byte("h")},
		".git/a.txt":   &fstest.MapFile{Data: []byte("g")},
		"sub/leaf.txt": &fstest.MapFile{Data: []byte("l")},
	}

	t.Run("hidden_excluded_by_default", func(t *testing.T) {
		t.Parallel()
		w := NewWalker(fsys, ".", WalkOptions{})
		got := collect(t, w)
		if len(got.errs) != 0 {
			t.Fatalf("errs = %v, want none", got.errs)
		}
		want := []string{"visible.txt", "sub/leaf.txt"}
		gp := append([]string(nil), got.paths...)
		slices.Sort(gp)
		slices.Sort(want)
		if !slices.Equal(gp, want) {
			t.Errorf("paths = %v, want %v", gp, want)
		}
	})

	t.Run("hidden_included_on_flag", func(t *testing.T) {
		t.Parallel()
		w := NewWalker(fsys, ".", WalkOptions{IncludeHidden: true})
		got := collect(t, w)
		if len(got.errs) != 0 {
			t.Fatalf("errs = %v, want none", got.errs)
		}
		want := []string{"visible.txt", ".hidden.txt", ".git/a.txt", "sub/leaf.txt"}
		gp := append([]string(nil), got.paths...)
		slices.Sort(gp)
		slices.Sort(want)
		if !slices.Equal(gp, want) {
			t.Errorf("paths = %v, want %v", gp, want)
		}
	})
}

func TestWalker_Gitignore(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		".gitignore":       &fstest.MapFile{Data: []byte("vendor/\n")},
		"keep.go":          &fstest.MapFile{Data: []byte("keep")},
		"vendor/foo.go":    &fstest.MapFile{Data: []byte("ignored")},
		"vendor/deep/b.go": &fstest.MapFile{Data: []byte("ignored")},
	}

	t.Run("gitignore_enabled_skips_vendor", func(t *testing.T) {
		t.Parallel()
		w := NewWalker(fsys, ".", WalkOptions{})
		got := collect(t, w)
		if len(got.errs) != 0 {
			t.Fatalf("errs = %v, want none", got.errs)
		}
		// keep.go yielded. vendor/... dropped.
		want := []string{"keep.go"}
		gp := append([]string(nil), got.paths...)
		slices.Sort(gp)
		if !slices.Equal(gp, want) {
			t.Errorf("paths = %v, want %v", gp, want)
		}
	})

	t.Run("gitignore_disabled_includes_vendor", func(t *testing.T) {
		t.Parallel()
		w := NewWalker(fsys, ".", WalkOptions{DisableGitignore: true})
		got := collect(t, w)
		if len(got.errs) != 0 {
			t.Fatalf("errs = %v, want none", got.errs)
		}
		want := []string{"keep.go", "vendor/foo.go", "vendor/deep/b.go"}
		gp := append([]string(nil), got.paths...)
		slices.Sort(gp)
		slices.Sort(want)
		if !slices.Equal(gp, want) {
			t.Errorf("paths = %v, want %v", gp, want)
		}
	})
}

func TestWalker_NestedGitignore(t *testing.T) {
	t.Parallel()

	// sub/.gitignore says "secret.txt" — scoped to sub/ only. A sibling
	// secret.txt at the root must NOT be dropped (F8).
	fsys := fstest.MapFS{
		"secret.txt":       &fstest.MapFile{Data: []byte("top")},
		"sub/.gitignore":   &fstest.MapFile{Data: []byte("secret.txt\n")},
		"sub/secret.txt":   &fstest.MapFile{Data: []byte("nested")},
		"sub/other.txt":    &fstest.MapFile{Data: []byte("sibling")},
		"other/secret.txt": &fstest.MapFile{Data: []byte("unrelated")},
	}

	w := NewWalker(fsys, ".", WalkOptions{})
	got := collect(t, w)
	if len(got.errs) != 0 {
		t.Fatalf("errs = %v, want none", got.errs)
	}

	// Root secret.txt kept (nested gitignore doesn't apply to siblings).
	// sub/secret.txt dropped. sub/other.txt kept. other/secret.txt kept.
	want := []string{"secret.txt", "sub/other.txt", "other/secret.txt"}
	gp := append([]string(nil), got.paths...)
	slices.Sort(gp)
	slices.Sort(want)
	if !slices.Equal(gp, want) {
		t.Errorf("paths = %v, want %v", gp, want)
	}
}

func TestWalker_IncludeExclude(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		"main.go":      &fstest.MapFile{Data: []byte("m")},
		"main_test.go": &fstest.MapFile{Data: []byte("t")},
		"README.md":    &fstest.MapFile{Data: []byte("r")},
		"notes.txt":    &fstest.MapFile{Data: []byte("n")},
	}

	t.Run("include_only_go_files", func(t *testing.T) {
		t.Parallel()
		w := NewWalker(fsys, ".", WalkOptions{Includes: []string{"*.go"}})
		got := collect(t, w)
		if len(got.errs) != 0 {
			t.Fatalf("errs = %v, want none", got.errs)
		}
		want := []string{"main.go", "main_test.go"}
		gp := append([]string(nil), got.paths...)
		slices.Sort(gp)
		slices.Sort(want)
		if !slices.Equal(gp, want) {
			t.Errorf("paths = %v, want %v", gp, want)
		}
	})

	t.Run("exclude_test_files", func(t *testing.T) {
		t.Parallel()
		w := NewWalker(fsys, ".", WalkOptions{
			Includes: []string{"*.go"},
			Excludes: []string{"*_test.go"},
		})
		got := collect(t, w)
		if len(got.errs) != 0 {
			t.Fatalf("errs = %v, want none", got.errs)
		}
		want := []string{"main.go"}
		gp := append([]string(nil), got.paths...)
		slices.Sort(gp)
		if !slices.Equal(gp, want) {
			t.Errorf("paths = %v, want %v", gp, want)
		}
	})
}

func TestWalker_ContextCancelled(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		"a.txt": &fstest.MapFile{Data: []byte("a")},
		"b.txt": &fstest.MapFile{Data: []byte("b")},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	w := NewWalker(fsys, ".", WalkOptions{})
	got := collectCtx(t, w, ctx)

	// Expect one yielded error wrapping context.Canceled, no paths
	// emitted.
	if len(got.paths) != 0 {
		t.Errorf("paths = %v, want none", got.paths)
	}
	if len(got.errs) == 0 {
		t.Fatalf("errs = none, want at least one")
	}
	if !errors.Is(got.errs[0], context.Canceled) {
		t.Errorf("first err = %v, want errors.Is(_, context.Canceled)", got.errs[0])
	}
}

// errFS wraps a MapFS and returns an error from Open on a configured
// directory path so we can exercise the per-entry error surface
// (fs.WalkDir calls WalkDirFunc a second time with err != nil when
// ReadDir fails; MapFS's Open → ReadDirFile chain is the lever).
type errFS struct {
	inner fstest.MapFS
	// failDir is the directory path whose ReadDir should fail.
	failDir string
}

func (e *errFS) Open(name string) (fs.File, error) {
	if name == e.failDir {
		// Return a fake directory handle whose ReadDir always errors.
		info, err := e.inner.Stat(name)
		if err != nil {
			return nil, err
		}
		return &errDir{info: info}, nil
	}
	return e.inner.Open(name)
}

// errDir is an fs.ReadDirFile that fails every ReadDir call.
type errDir struct {
	info fs.FileInfo
}

func (e *errDir) Stat() (fs.FileInfo, error) { return e.info, nil }
func (e *errDir) Read(p []byte) (int, error) { return 0, fmt.Errorf("is a directory") }
func (e *errDir) Close() error               { return nil }
func (e *errDir) ReadDir(n int) ([]fs.DirEntry, error) {
	return nil, errors.New("induced ReadDir failure")
}

func TestWalker_UnreadableEntry(t *testing.T) {
	t.Parallel()

	inner := fstest.MapFS{
		"keep.txt":     &fstest.MapFile{Data: []byte("k")},
		"broken":       &fstest.MapFile{Mode: fs.ModeDir},
		"broken/x.txt": &fstest.MapFile{Data: []byte("x")},
		"other/y.txt":  &fstest.MapFile{Data: []byte("y")},
	}
	fsys := &errFS{inner: inner, failDir: "broken"}

	w := NewWalker(fsys, ".", WalkOptions{})
	got := collect(t, w)

	// The broken dir's ReadDir failure should surface as a yielded
	// error, but the walk must continue and emit keep.txt + other/y.txt.
	if len(got.errs) == 0 {
		t.Fatalf("errs = none, want the induced ReadDir failure")
	}
	foundInducedErr := false
	for _, err := range got.errs {
		if strings.Contains(err.Error(), "induced ReadDir failure") {
			foundInducedErr = true
			break
		}
	}
	if !foundInducedErr {
		t.Errorf("yielded errors = %v, want one containing %q", got.errs, "induced ReadDir failure")
	}

	want := []string{"keep.txt", "other/y.txt"}
	gp := append([]string(nil), got.paths...)
	slices.Sort(gp)
	slices.Sort(want)
	if !slices.Equal(gp, want) {
		t.Errorf("paths = %v, want %v", gp, want)
	}
}

func TestWalker_RangeBreak(t *testing.T) {
	t.Parallel()

	// At least three files so a break after the first emission would
	// panic if the WalkDirFunc returned nil after yield==false (F14).
	fsys := fstest.MapFS{
		"a.txt": &fstest.MapFile{Data: []byte("a")},
		"b.txt": &fstest.MapFile{Data: []byte("b")},
		"c.txt": &fstest.MapFile{Data: []byte("c")},
		"d.txt": &fstest.MapFile{Data: []byte("d")},
	}

	w := NewWalker(fsys, ".", WalkOptions{})

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("walker panicked after range break: %v", r)
		}
	}()

	count := 0
	for f, err := range w.Walk(context.Background()) {
		if err != nil {
			t.Fatalf("unexpected err on iteration %d: %v", count, err)
		}
		if f == nil {
			t.Fatalf("nil file on iteration %d", count)
		}
		count++
		break
	}

	if count != 1 {
		t.Errorf("count = %d, want 1 (F14 guard — only one yield should have returned before break halted iteration)", count)
	}
}

func TestWalker_SymlinkYielded(t *testing.T) {
	t.Parallel()

	// MapFS symlink support: MapFile with Mode fs.ModeSymlink uses Data
	// as the target path. link_ok targets an existing file; link_broken
	// targets a missing file.
	fsys := fstest.MapFS{
		"target.txt":  &fstest.MapFile{Data: []byte("t")},
		"link_ok":     &fstest.MapFile{Mode: fs.ModeSymlink, Data: []byte("target.txt")},
		"link_broken": &fstest.MapFile{Mode: fs.ModeSymlink, Data: []byte("missing.txt")},
	}

	w := NewWalker(fsys, ".", WalkOptions{})
	got := collect(t, w)

	if len(got.errs) != 0 {
		t.Fatalf("errs = %v, want none", got.errs)
	}

	want := []string{"target.txt", "link_ok", "link_broken"}
	gp := append([]string(nil), got.paths...)
	slices.Sort(gp)
	slices.Sort(want)
	if !slices.Equal(gp, want) {
		t.Fatalf("paths = %v, want %v", gp, want)
	}

	// Broken symlink's File.Open must return an error unwrapping to
	// fs.ErrNotExist per the F7 contract and file.go's open %q: %w
	// wrapping.
	var broken *File
	for f, err := range w.Walk(context.Background()) {
		if err != nil {
			t.Fatalf("unexpected err during broken-link lookup: %v", err)
		}
		if f.RelPath == "link_broken" {
			broken = f
			break
		}
	}
	if broken == nil {
		t.Fatal("could not locate link_broken File")
	}

	_, err := broken.Open()
	if err == nil {
		t.Fatal("Open(link_broken) = nil error, want non-nil")
	}
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("errors.Is(err, fs.ErrNotExist) = false, want true (err = %v)", err)
	}
}
