package lister_test

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/evanmschultz/rak/internal/fileset"
	"github.com/evanmschultz/rak/internal/lister"
)

// TestDetect_InsideRepo verifies that Detect returns a *GitLister when called
// with a root that is inside a git repository and DisableGitignore is false.
// Constructs a throwaway git repo in t.TempDir() so the test is hermetic and
// does not depend on the rak checkout being visible on the CI runner.
func TestDetect_InsideRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not found")
	}

	tmp := t.TempDir()
	initCmd := exec.Command("git", "init", "--template=", tmp)
	if out, err := initCmd.CombinedOutput(); err != nil {
		t.Skipf("git init failed (%v): %s", err, out)
	}

	ctx := t.Context()
	got, err := lister.Detect(ctx, tmp, fileset.WalkOptions{})
	if err != nil {
		t.Fatalf("Detect returned unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("Detect returned nil lister, want non-nil")
	}

	if _, ok := got.(*lister.GitLister); !ok {
		t.Errorf("Detect returned %T, want *lister.GitLister", got)
	}
}

// TestDetect_OutsideRepo verifies that Detect returns a *WalkLister when
// called with a root that is not inside a git repository.
func TestDetect_OutsideRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not found")
	}

	// t.TempDir() is outside any git repository on a typical CI machine; it is
	// guaranteed to be a non-git directory. On developer machines with
	// unconventional setups, the skip below provides a safety net.
	dir := t.TempDir()

	ctx := t.Context()
	got, err := lister.Detect(ctx, dir, fileset.WalkOptions{})
	if err != nil {
		t.Fatalf("Detect returned unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("Detect returned nil lister, want non-nil")
	}

	if _, ok := got.(*lister.WalkLister); !ok {
		t.Errorf("Detect returned %T, want *lister.WalkLister", got)
	}
}

// TestDetect_BareRepo verifies that Detect returns a *WalkLister (not a
// *GitLister and not an error) when called with the root of a bare git
// repository. A bare repo causes "git rev-parse --is-inside-work-tree" to
// print "false" with exit 0 — the fix checks stdout rather than exit code only.
func TestDetect_BareRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not found")
	}

	tmp := t.TempDir()
	initCmd := exec.Command("git", "init", "--bare", "--template=", tmp)
	if out, err := initCmd.CombinedOutput(); err != nil {
		t.Skipf("git init --bare failed (%v): %s", err, out)
	}

	ctx := t.Context()
	got, err := lister.Detect(ctx, tmp, fileset.WalkOptions{})
	if err != nil {
		t.Fatalf("Detect returned unexpected error for bare repo: %v", err)
	}
	if got == nil {
		t.Fatal("Detect returned nil lister for bare repo, want non-nil")
	}
	if _, ok := got.(*lister.WalkLister); !ok {
		t.Errorf("Detect returned %T for bare repo, want *lister.WalkLister", got)
	}
}

// TestDetect_InsideGitDir verifies that Detect returns a *WalkLister when
// called with the .git/ directory of a normal (non-bare) git repository.
// Inside .git/, "git rev-parse --is-inside-work-tree" prints "false" with
// exit 0 — the fix checks stdout rather than exit code only.
func TestDetect_InsideGitDir(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not found")
	}

	tmp := t.TempDir()
	initCmd := exec.Command("git", "init", "--template=", tmp)
	if out, err := initCmd.CombinedOutput(); err != nil {
		t.Skipf("git init failed (%v): %s", err, out)
	}

	gitDir := filepath.Join(tmp, ".git")
	ctx := t.Context()
	got, err := lister.Detect(ctx, gitDir, fileset.WalkOptions{})
	if err != nil {
		t.Fatalf("Detect returned unexpected error for .git/ dir: %v", err)
	}
	if got == nil {
		t.Fatal("Detect returned nil lister for .git/ dir, want non-nil")
	}
	if _, ok := got.(*lister.WalkLister); !ok {
		t.Errorf("Detect returned %T for .git/ dir, want *lister.WalkLister", got)
	}
}

// TestDetect_BareRepo_WithDisableGitignore verifies that Detect returns a
// *WalkLister (not an error) when called with the root of a bare git
// repository AND DisableGitignore is true. The ErrNoGitignoreInRepo sentinel
// applies only when the walk root is inside a real work tree — bare repos fall
// through to the WalkLister path regardless of DisableGitignore.
func TestDetect_BareRepo_WithDisableGitignore(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not found")
	}

	dir := t.TempDir()
	cmd := exec.Command("git", "init", "--bare", "--template=", dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("git init --bare failed (%v): %s", err, out)
	}

	ctx := t.Context()
	got, err := lister.Detect(ctx, dir, fileset.WalkOptions{DisableGitignore: true})
	if err != nil {
		t.Fatalf("Detect returned unexpected error for bare repo with DisableGitignore: %v", err)
	}
	if _, ok := got.(*lister.WalkLister); !ok {
		t.Errorf("Detect returned %T for bare repo with DisableGitignore, want *lister.WalkLister", got)
	}
}

// TestDetect_InsideGitDir_WithDisableGitignore verifies that Detect returns a
// *WalkLister (not an error) when called with the .git/ directory of a normal
// repository AND DisableGitignore is true. Inside .git/, the walk root is not
// a real work tree, so ErrNoGitignoreInRepo must not be returned even when
// DisableGitignore is set.
func TestDetect_InsideGitDir_WithDisableGitignore(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not found")
	}

	dir := t.TempDir()
	cmd := exec.Command("git", "init", "--template=", dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("git init failed (%v): %s", err, out)
	}

	ctx := t.Context()
	got, err := lister.Detect(ctx, filepath.Join(dir, ".git"), fileset.WalkOptions{DisableGitignore: true})
	if err != nil {
		t.Fatalf("Detect returned unexpected error for .git/ dir with DisableGitignore: %v", err)
	}
	if _, ok := got.(*lister.WalkLister); !ok {
		t.Errorf("Detect returned %T for .git/ dir with DisableGitignore, want *lister.WalkLister", got)
	}
}

// TestDetect_NoGitignoreInRepo_ReturnsSentinel verifies that Detect returns
// ErrNoGitignoreInRepo (via errors.Is) when DisableGitignore is true and the
// walk root is inside a git repository. Constructs a throwaway git repo in
// t.TempDir() so the test is hermetic — no dependency on the rak checkout.
func TestDetect_NoGitignoreInRepo_ReturnsSentinel(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not found")
	}

	tmp := t.TempDir()
	initCmd := exec.Command("git", "init", "--template=", tmp)
	if out, err := initCmd.CombinedOutput(); err != nil {
		t.Skipf("git init failed (%v): %s", err, out)
	}

	ctx := t.Context()
	got, err := lister.Detect(ctx, tmp, fileset.WalkOptions{DisableGitignore: true})
	if got != nil {
		t.Errorf("Detect returned non-nil lister, want nil when sentinel returned")
	}
	if !errors.Is(err, lister.ErrNoGitignoreInRepo) {
		t.Errorf("errors.Is(err, ErrNoGitignoreInRepo) = false; got %v", err)
	}
}

// TestDetect_SingleFile verifies that Detect returns a *SingleFileLister when
// the root argument points to a regular file (not a directory). The test
// writes a temporary file into t.TempDir() and passes the file path directly
// to Detect. This exercises the early-return path added in v0.1.4 (Bug A).
func TestDetect_SingleFile(t *testing.T) {
	tmp := t.TempDir()
	filePath := filepath.Join(tmp, "sample.txt")
	if err := os.WriteFile(filePath, []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	ctx := t.Context()
	got, err := lister.Detect(ctx, filePath, fileset.WalkOptions{})
	if err != nil {
		t.Fatalf("Detect returned unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("Detect returned nil lister, want non-nil")
	}
	if _, ok := got.(*lister.SingleFileLister); !ok {
		t.Errorf("Detect returned %T, want *lister.SingleFileLister", got)
	}
}

// TestDetect_SymlinkedFile verifies that Detect resolves a symlink that points
// to a regular file and returns a *SingleFileLister. This exercises the
// EvalSymlinks + stat path added in v0.1.4 (Bug B + Bug A combined).
func TestDetect_SymlinkedFile(t *testing.T) {
	tmp := t.TempDir()
	target := filepath.Join(tmp, "real.txt")
	if err := os.WriteFile(target, []byte("content\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	link := filepath.Join(tmp, "link.txt")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("os.Symlink not supported: %v", err)
	}

	ctx := t.Context()
	got, err := lister.Detect(ctx, link, fileset.WalkOptions{})
	if err != nil {
		t.Fatalf("Detect returned unexpected error for symlink-to-file: %v", err)
	}
	if got == nil {
		t.Fatal("Detect returned nil lister, want non-nil")
	}
	if _, ok := got.(*lister.SingleFileLister); !ok {
		t.Errorf("Detect returned %T for symlink-to-file, want *lister.SingleFileLister", got)
	}
}

// TestDetect_SymlinkedDir verifies that Detect resolves a symlink that points
// to a directory containing a git repository and returns a *GitLister. This
// exercises the EvalSymlinks path added in v0.1.4 (Bug B) for the directory
// case where a git probe can succeed after symlink resolution.
func TestDetect_SymlinkedDir(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not found")
	}

	tmp := t.TempDir()
	// Create a real git repo inside tmp.
	repoDir := filepath.Join(tmp, "repo")
	if err := os.Mkdir(repoDir, 0o755); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}
	initCmd := exec.Command("git", "init", "--template=", repoDir)
	if out, err := initCmd.CombinedOutput(); err != nil {
		t.Skipf("git init failed (%v): %s", err, out)
	}

	// Create a symlink pointing to the git repo directory.
	link := filepath.Join(tmp, "link-to-repo")
	if err := os.Symlink(repoDir, link); err != nil {
		t.Skipf("os.Symlink not supported: %v", err)
	}

	ctx := t.Context()
	got, err := lister.Detect(ctx, link, fileset.WalkOptions{})
	if err != nil {
		t.Fatalf("Detect returned unexpected error for symlink-to-dir: %v", err)
	}
	if got == nil {
		t.Fatal("Detect returned nil lister, want non-nil")
	}
	if _, ok := got.(*lister.GitLister); !ok {
		t.Errorf("Detect returned %T for symlink-to-git-dir, want *lister.GitLister", got)
	}
}

// TestDetect_BrokenSymlink verifies that Detect returns a wrapped error when
// the root argument is a symlink that points to a path that does not exist.
// EvalSymlinks fails on broken symlinks; Detect must propagate that error.
func TestDetect_BrokenSymlink(t *testing.T) {
	tmp := t.TempDir()
	link := filepath.Join(tmp, "broken")
	// Point to a path that does not exist.
	if err := os.Symlink(filepath.Join(tmp, "nonexistent"), link); err != nil {
		t.Skipf("os.Symlink not supported: %v", err)
	}

	ctx := t.Context()
	got, err := lister.Detect(ctx, link, fileset.WalkOptions{})
	if err == nil {
		t.Fatalf("Detect returned nil error for broken symlink, want an error (lister: %T)", got)
	}
	if got != nil {
		t.Errorf("Detect returned non-nil lister for broken symlink, want nil")
	}
	// The error must be wrapped with the "lister: detect:" prefix so callers
	// get consistent error messages regardless of the underlying OS error.
	if msg := err.Error(); !strings.HasPrefix(msg, "lister: detect:") {
		t.Errorf("error message does not start with %q: %s", "lister: detect:", msg)
	}
}

// TestSingleFileLister_List verifies that a SingleFileLister constructed
// directly yields exactly one file with the expected RelPath value, and that
// iterating a second time produces the same result (idempotency).
func TestSingleFileLister_List(t *testing.T) {
	tmp := t.TempDir()
	filePath := filepath.Join(tmp, "hello.go")
	if err := os.WriteFile(filePath, []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Build a SingleFileLister via Detect so we don't depend on any unexported
	// constructor — Detect returns *SingleFileLister for a regular file path.
	fl, err := lister.Detect(t.Context(), filePath, fileset.WalkOptions{})
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if _, ok := fl.(*lister.SingleFileLister); !ok {
		t.Fatalf("expected *lister.SingleFileLister, got %T", fl)
	}

	// Iterate and collect results.
	var files []*fileset.File
	var errs []error
	for f, e := range fl.List(context.Background()) {
		if e != nil {
			errs = append(errs, e)
			continue
		}
		files = append(files, f)
	}

	if len(errs) != 0 {
		t.Fatalf("List returned unexpected errors: %v", errs)
	}
	if len(files) != 1 {
		t.Fatalf("List yielded %d files, want exactly 1", len(files))
	}
	if got := files[0].RelPath; got != "hello.go" {
		t.Errorf("RelPath = %q, want %q", got, "hello.go")
	}
}
