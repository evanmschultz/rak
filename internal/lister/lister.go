// Package lister resolves the concrete file source for a given walk root and
// exposes it behind a uniform FileLister interface. The package selects
// between a git-backed source (GitLister, Unit 4.2) and a filesystem-walk
// source (WalkLister, Unit 4.3) based on whether the walk root lives inside a
// git repository.
package lister

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/evanmschultz/rak/internal/fileset"
)

// FileLister enumerates files under a walk root as a streaming iterator. The
// iterator contract mirrors fileset.Walker.Walk: per-entry errors are yielded
// as (nil, err) pairs and iteration continues; context cancellation terminates
// iteration with (nil, ctx.Err()); implementations must not panic when the
// caller's yield returns false (F14 carry-over).
type FileLister interface {
	List(ctx context.Context) iter.Seq2[*fileset.File, error]
}

// ErrNoGitignoreInRepo is returned by Detect when the caller passes
// opts.DisableGitignore = true and the walk root is inside a git repository.
// In git mode, rak enumerates only tracked files — .gitignore rules have
// already been applied by git itself, so --no-gitignore has no effect.
// Callers branch on this condition via errors.Is(err, lister.ErrNoGitignoreInRepo);
// never string-match the message.
var ErrNoGitignoreInRepo = errors.New("rak: --no-gitignore has no effect when run inside a git repository. rak counts git-tracked files in this mode. To count untracked files, run rak outside the repository.")

// Detect resolves the concrete FileLister for root. It resolves root to an
// absolute path, then probes whether root sits inside a git repository by
// running "git rev-parse --is-inside-work-tree". The probe result drives the
// selection:
//
//   - Exit 0 (in repo) + opts.DisableGitignore true  → (nil, ErrNoGitignoreInRepo).
//   - Exit 0 (in repo) + opts.DisableGitignore false → newGitLister (Unit 4.2).
//   - Non-zero exit (not in repo) or git binary absent → newWalkLister (Unit 4.3).
//   - Unexpected OS-level command failure             → wrapped error.
//
// Errors are wrapped with the "lister: detect: %w" prefix.
func Detect(ctx context.Context, root string, opts fileset.WalkOptions) (FileLister, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}

	// Fast-path: if git is not installed, skip the probe and fall back to the
	// filesystem walker immediately.
	if _, lookErr := exec.LookPath("git"); lookErr != nil {
		return newWalkLister(os.DirFS(absRoot), ".", opts), nil
	}

	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--is-inside-work-tree")
	cmd.Dir = absRoot
	runErr := cmd.Run()

	if runErr == nil {
		// Exit 0: we are inside a git repository.
		if opts.DisableGitignore {
			return nil, fmt.Errorf("lister: detect: %w", ErrNoGitignoreInRepo)
		}
		return newGitLister(ctx, absRoot, opts)
	}

	// Distinguish a non-zero git exit (not in a repo) from an OS-level failure
	// that prevented the command from running at all.
	var exitErr *exec.ExitError
	if errors.As(runErr, &exitErr) {
		// Non-zero exit: not inside a git repository. Use the walk-based source.
		return newWalkLister(os.DirFS(absRoot), ".", opts), nil
	}

	// Unexpected OS-level command failure (e.g. permission denied on the
	// process spawn itself, not from git).
	return nil, fmt.Errorf("lister: detect: %w", runErr)
}
