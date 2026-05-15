package lister_test

import (
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/evanmschultz/rak/internal/fileset"
	"github.com/evanmschultz/rak/internal/lister"
)

// TestDetect_InsideRepo verifies that Detect returns a *GitLister when called
// with a root that is inside the rak git repository and DisableGitignore is
// false. Uses the actual rak checkout so no hermetic fixture is needed.
//
// TODO unit 4.2: enable GitLister type assertion (currently inert until
// GitLister is defined in git.go).
func TestDetect_InsideRepo(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not found")
	}

	// main/internal/lister/ is two levels below main/internal/ and three
	// levels below main/ — ../../.. resolves to the main/ checkout root.
	absRoot, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatalf("filepath.Abs: %v", err)
	}

	ctx := t.Context()
	got, err := lister.Detect(ctx, absRoot, fileset.WalkOptions{})
	// Exit 128 from git means git environment is broken in this test subprocess
	// (e.g. GIT_DIR or similar set by a parent process in a way that conflicts
	// with cmd.Dir-based repo discovery). Skip rather than fail in that case.
	if err != nil && strings.Contains(err.Error(), "exit status 128") {
		t.Skipf("git env broken in test subprocess (exit 128); skipping: %v", err)
	}
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
//
// TODO unit 4.3: enable WalkLister type assertion (currently inert until
// WalkLister is defined in walk.go).
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

// TestDetect_NoGitignoreInRepo_ReturnsSentinel verifies that Detect returns
// ErrNoGitignoreInRepo (via errors.Is) when DisableGitignore is true and the
// walk root is inside a git repository. This test can pass fully at the 4.1
// commit boundary: it exercises only the Detect branch that returns the
// sentinel, which does not require GitLister or WalkLister.
func TestDetect_NoGitignoreInRepo_ReturnsSentinel(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not found")
	}

	absRoot, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatalf("filepath.Abs: %v", err)
	}

	ctx := t.Context()
	got, err := lister.Detect(ctx, absRoot, fileset.WalkOptions{DisableGitignore: true})
	if got != nil {
		t.Errorf("Detect returned non-nil lister, want nil when sentinel returned")
	}
	if !errors.Is(err, lister.ErrNoGitignoreInRepo) {
		t.Errorf("errors.Is(err, ErrNoGitignoreInRepo) = false; got %v", err)
	}
}
