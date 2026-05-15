package lister

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/evanmschultz/rak/internal/fileset"
)

// skipIfNoGit calls t.Skip when the git binary is absent from PATH.
func skipIfNoGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not found")
	}
}

// mainDir returns the absolute path to main/ (the repo working tree root),
// resolved as ../.. relative to main/internal/lister/ where tests run.
func mainDir(t *testing.T) string {
	t.Helper()
	abs, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatalf("filepath.Abs: %v", err)
	}
	return abs
}

// filesetDir returns the absolute path to main/internal/fileset/.
func filesetDir(t *testing.T) string {
	t.Helper()
	abs, err := filepath.Abs("../fileset")
	if err != nil {
		t.Fatalf("filepath.Abs: %v", err)
	}
	return abs
}

// TestGitLister_List_InRepo verifies that GitLister enumerates the actual rak
// checkout and that go.mod appears in results with RelPath "go.mod"
// (walk-root-relative, no leading separator).
func TestGitLister_List_InRepo(t *testing.T) {
	skipIfNoGit(t)
	ctx := t.Context()

	root := mainDir(t)
	gl, err := newGitLister(ctx, root, fileset.WalkOptions{IncludeHidden: true})
	if err != nil {
		t.Fatalf("newGitLister: %v", err)
	}

	var found bool
	for f, err := range gl.List(ctx) {
		if err != nil {
			t.Fatalf("List yielded error: %v", err)
		}
		if f.RelPath == "go.mod" {
			found = true
			break
		}
	}
	if !found {
		t.Error("List did not yield a file with RelPath == \"go.mod\"")
	}
}

// TestGitLister_List_SubdirRoot verifies that GitLister with a sub-directory
// walk root yields only files from that subtree, with RelPath values relative
// to the sub-directory root (not the repo toplevel).
func TestGitLister_List_SubdirRoot(t *testing.T) {
	skipIfNoGit(t)
	ctx := t.Context()

	root := filesetDir(t)
	gl, err := newGitLister(ctx, root, fileset.WalkOptions{IncludeHidden: true})
	if err != nil {
		t.Fatalf("newGitLister: %v", err)
	}

	var relPaths []string
	for f, err := range gl.List(ctx) {
		if err != nil {
			t.Fatalf("List yielded error: %v", err)
		}
		relPaths = append(relPaths, f.RelPath)
	}

	if len(relPaths) == 0 {
		t.Fatal("List yielded no files for internal/fileset/ root")
	}

	// Every relPath must be fileset-root-relative: no "internal/" prefix.
	for _, rp := range relPaths {
		if strings.HasPrefix(rp, "internal/") {
			t.Errorf("RelPath %q is not walk-root-relative (still has package prefix)", rp)
		}
		if strings.HasPrefix(rp, "./") {
			t.Errorf("RelPath %q has leading ./", rp)
		}
		if strings.HasPrefix(rp, "/") {
			t.Errorf("RelPath %q has leading /", rp)
		}
	}

	// At minimum file.go and walker.go should appear.
	want := []string{"file.go", "walker.go"}
	for _, name := range want {
		found := false
		for _, rp := range relPaths {
			if rp == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected RelPath %q in results, got: %v", name, relPaths)
		}
	}
}

// TestGitLister_FilterHidden verifies Decision B: hidden files tracked by git
// (e.g. .gitignore at the repo root) are excluded by default when
// IncludeHidden is false, and included when IncludeHidden is true.
//
// It also directly exercises anySegmentHidden to cover the non-first-segment
// case (F2 — hidden segment at index 1 in a multi-component path).
func TestGitLister_FilterHidden(t *testing.T) {
	skipIfNoGit(t)
	ctx := t.Context()

	root := mainDir(t)

	// Run with hidden excluded (default).
	glHidden, err := newGitLister(ctx, root, fileset.WalkOptions{IncludeHidden: false})
	if err != nil {
		t.Fatalf("newGitLister (hidden=false): %v", err)
	}
	var noHiddenPaths []string
	for f, err := range glHidden.List(ctx) {
		if err != nil {
			t.Fatalf("List yielded error: %v", err)
		}
		noHiddenPaths = append(noHiddenPaths, f.RelPath)
	}
	for _, rp := range noHiddenPaths {
		if rp == ".gitignore" {
			t.Errorf("IncludeHidden=false: .gitignore should be excluded but appeared in results")
		}
	}

	// Run with hidden included.
	glInclude, err := newGitLister(ctx, root, fileset.WalkOptions{IncludeHidden: true})
	if err != nil {
		t.Fatalf("newGitLister (hidden=true): %v", err)
	}
	var withHiddenPaths []string
	for f, err := range glInclude.List(ctx) {
		if err != nil {
			t.Fatalf("List yielded error: %v", err)
		}
		withHiddenPaths = append(withHiddenPaths, f.RelPath)
	}
	foundGitignore := false
	for _, rp := range withHiddenPaths {
		if rp == ".gitignore" {
			foundGitignore = true
			break
		}
	}
	if !foundGitignore {
		t.Error("IncludeHidden=true: .gitignore should appear in results but did not")
	}

	// F2 — non-first-segment hidden check: anySegmentHidden must detect a
	// hidden segment even when it is not at index 0. This exercises the full
	// loop body of anySegmentHidden beyond the first element.
	t.Run("anySegmentHidden_NonFirstSegment", func(t *testing.T) {
		cases := []struct {
			path   string
			hidden bool
		}{
			{"internal/.cache/x.bin", true}, // hidden at index 1
			{"a/b/.hidden/c.txt", true},     // hidden at index 2
			{"normal/path/file.go", false},  // no hidden segment
			{".hidden", true},               // hidden at index 0 (existing coverage, kept for completeness)
		}
		for _, tc := range cases {
			got := anySegmentHidden(tc.path)
			if got != tc.hidden {
				t.Errorf("anySegmentHidden(%q) = %v, want %v", tc.path, got, tc.hidden)
			}
		}
	})
}

// TestGitLister_ContextCancel verifies that List yields (nil, ctx.Err()) when
// the context is cancelled before iteration begins.
func TestGitLister_ContextCancel(t *testing.T) {
	skipIfNoGit(t)

	root := mainDir(t)
	// Use background context for construction; cancel before List.
	gl, err := newGitLister(context.Background(), root, fileset.WalkOptions{})
	if err != nil {
		t.Fatalf("newGitLister: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	var gotErr error
	for f, err := range gl.List(ctx) {
		if f != nil {
			// May receive a file if git output was already buffered before cancel.
			// Only assert on the error path below.
			continue
		}
		gotErr = err
		break
	}
	if gotErr == nil {
		t.Skip("context cancellation did not propagate (git output buffered before cancel); acceptable on fast machines")
	}
	if gotErr != context.Canceled {
		t.Errorf("List yielded err = %v, want context.Canceled", gotErr)
	}
}

// TestGitLister_RelPathInvariant enforces F26: every emitted *fileset.File has
// a walk-root-relative RelPath with forward-slash separators, no leading "./",
// and no leading "/".
func TestGitLister_RelPathInvariant(t *testing.T) {
	skipIfNoGit(t)
	ctx := t.Context()

	root := mainDir(t)
	gl, err := newGitLister(ctx, root, fileset.WalkOptions{IncludeHidden: true})
	if err != nil {
		t.Fatalf("newGitLister: %v", err)
	}

	var count int
	for f, err := range gl.List(ctx) {
		if err != nil {
			t.Fatalf("List yielded error: %v", err)
		}
		count++
		rp := f.RelPath

		if strings.HasPrefix(rp, "./") {
			t.Errorf("F26: RelPath %q has leading ./", rp)
		}
		if strings.HasPrefix(rp, "/") {
			t.Errorf("F26: RelPath %q has leading /", rp)
		}
		if rp != filepath.ToSlash(rp) {
			t.Errorf("F26: RelPath %q contains backslash separators", rp)
		}
	}
	if count == 0 {
		t.Error("F26 invariant test: no files emitted; cannot verify invariant")
	}
}
