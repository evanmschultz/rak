package lister_test

import (
	"errors"
	"os/exec"
	"path/filepath"
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
