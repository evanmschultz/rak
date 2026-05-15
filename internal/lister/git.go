// Package lister — git-backed file enumeration (Unit 4.2).
package lister

import (
	"context"
	"fmt"
	"io/fs"
	"iter"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/evanmschultz/rak/internal/fileset"
	"github.com/evanmschultz/rak/internal/ignore"
)

// GitLister enumerates files tracked by git under a walk root. It runs
// "git ls-files --full-name -z" to obtain the list of tracked paths, then
// applies the same per-path filters that fileset.Walker applies (hidden
// check, depth limit, include/exclude globs). The opts.DisableGitignore flag
// is unreachable for GitLister: lister.Detect returns ErrNoGitignoreInRepo
// before newGitLister is ever called when DisableGitignore is true (F19 /
// Decision A).
//
// GitLister is exported so callers (e.g. lister_test.go) can perform type
// assertions on the value returned by lister.Detect.
type GitLister struct {
	absRoot  string
	toplevel string
	prefix   string
	fsys     fs.FS
	opts     fileset.WalkOptions
}

// newGitLister constructs a GitLister for root. root should already be an
// absolute path (Detect resolves it before calling newGitLister), but
// filepath.Abs is called defensively.
//
// It runs "git rev-parse --show-toplevel" with cmd.Dir = absRoot to obtain the
// repository toplevel, then computes the walk-root-relative prefix needed to
// scope git ls-files output to the requested subtree (F17).
func newGitLister(ctx context.Context, root string, opts fileset.WalkOptions) (*GitLister, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("lister: new git lister: %w", err)
	}

	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--show-toplevel")
	cmd.Dir = absRoot
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("lister: new git lister: %w", err)
	}
	toplevel := strings.TrimRight(string(out), "\n\r")

	// Compute the forward-slash prefix that separates the walk root from the
	// repository toplevel (F17). When absRoot == toplevel, prefix is empty and
	// no prefix filtering is needed during List.
	prefix := filepath.ToSlash(strings.TrimPrefix(absRoot, toplevel))
	prefix = strings.TrimPrefix(prefix, "/")

	return &GitLister{
		absRoot:  absRoot,
		toplevel: toplevel,
		prefix:   prefix,
		fsys:     os.DirFS(absRoot),
		opts:     opts,
	}, nil
}

// anySegmentHidden reports whether any forward-slash-delimited segment of
// relPath is a hidden name (starts with a dot, excluding "." and ".."). It is
// used by List to implement F21: hidden files and files inside hidden
// directories are excluded when opts.IncludeHidden is false.
func anySegmentHidden(relPath string) bool {
	for _, seg := range strings.Split(relPath, "/") {
		if fileset.IsHidden(seg) {
			return true
		}
	}
	return false
}

// List returns an iterator that yields every tracked file under the walk root
// that survives the configured filters. The iterator is a range-over-func
// value matching the FileLister contract: per-entry errors are yielded as
// (nil, err) pairs; context cancellation terminates iteration with
// (nil, ctx.Err()); the iterator does not panic if the caller's yield returns
// false (F14 carry-over).
//
// List runs "git ls-files --full-name -z" once to collect the full tracked
// list, then iterates over the NUL-separated output. The "git ls-files"
// subprocess is run with cmd.Dir = g.absRoot so git scopes output to the
// subtree, but per Decision E the emitted paths are toplevel-relative
// regardless of CWD. The prefix-strip step (F17) is therefore always active
// when g.prefix != "".
func (g *GitLister) List(ctx context.Context) iter.Seq2[*fileset.File, error] {
	return func(yield func(*fileset.File, error) bool) {
		// Run git ls-files to collect all tracked paths.
		cmd := exec.CommandContext(ctx, "git", "ls-files", "--full-name", "-z")
		cmd.Dir = g.absRoot
		stdout, err := cmd.Output()
		if err != nil {
			// Distinguish context cancellation from a git failure.
			if ctxErr := ctx.Err(); ctxErr != nil {
				yield(nil, ctxErr)
				return
			}
			yield(nil, fmt.Errorf("lister: git ls-files: %w", err))
			return
		}

		// Build the ignore.Matcher once before the per-path loop (F18).
		matcher, err := ignore.New(nil, g.opts.Includes, g.opts.Excludes)
		if err != nil {
			yield(nil, fmt.Errorf("lister: git lister build matcher: %w", err))
			return
		}

		// Split on NUL; discard the trailing empty entry from NUL-terminated output.
		raw := string(stdout)
		parts := strings.Split(raw, "\x00")
		if len(parts) > 0 && parts[len(parts)-1] == "" {
			parts = parts[:len(parts)-1]
		}

		for _, rawPath := range parts {
			// Context cancellation: stop iteration cleanly.
			if ctx.Err() != nil {
				yield(nil, ctx.Err())
				return
			}

			// Prefix handling (F17, Decision E empirical): git emits
			// toplevel-relative paths regardless of CWD. When g.prefix is
			// non-empty, paths that don't start with that prefix belong to other
			// subtrees and must be skipped; paths that do have their prefix
			// stripped to obtain walk-root-relative relPath.
			var relPath string
			if g.prefix != "" {
				want := g.prefix + "/"
				if !strings.HasPrefix(rawPath, want) {
					continue
				}
				relPath = strings.TrimPrefix(rawPath, want)
			} else {
				relPath = rawPath
			}

			// Forward-slash normalize (safety guard for any platform divergence).
			relPath = filepath.ToSlash(relPath)

			// Hidden check (F21 / Decision B).
			if !g.opts.IncludeHidden && anySegmentHidden(relPath) {
				continue
			}

			// Depth check (F18, C15): matches Walker's "depth >= w.opts.Depth"
			// using >= and guarding on Depth > 0 (zero means unlimited).
			if g.opts.Depth > 0 && strings.Count(relPath, "/") >= g.opts.Depth {
				continue
			}

			// Include/exclude matcher check (F18): true means "drop this path".
			if matcher.Match(relPath, false) {
				continue
			}

			// Emit. Honour F14: if yield returns false, stop.
			if !yield(fileset.NewFile(g.fsys, relPath, relPath), nil) {
				return
			}
		}
	}
}
