package fileset

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"iter"
	"path"
	"strings"

	"github.com/evanmschultz/rak/internal/ignore"
)

// WalkOptions configures a Walker. The zero value walks every non-hidden
// file under the root with gitignore enabled and no include / exclude
// globs — matching rak's defaults per main/CLAUDE.md.
//
// Depth counts directory edges from the walk root: files directly under
// the root sit at depth 0; files one directory deeper sit at depth 1; and
// so on. A Depth of 0 disables the limit (unlimited descent). A Depth of
// 1 walks only the root directory itself, skipping every subdirectory.
// See C7 in DROP_3's PLAN.md for the pin.
//
// IncludeHidden controls whether entries whose Name() starts with a dot
// are yielded; when false (the default) hidden files are skipped and
// hidden directories are pruned via fs.SkipDir. The check uses the
// package-level IsHidden helper against fs.DirEntry.Name().
//
// DisableGitignore defaults to false so .gitignore handling is ENABLED
// by default. The --no-gitignore CLI flag (wired in Unit 3.5) sets it
// true. See C2.
//
// Includes and Excludes are forwarded verbatim to ignore.New. Empty
// Includes means "allow every path that survives the earlier filters";
// empty Excludes means "deny nothing at the exclude stage". See F2 / F3
// for precedence.
type WalkOptions struct {
	// Depth is the maximum directory edge count from the walk root. A
	// value of 0 disables the limit.
	Depth int

	// IncludeHidden enables emission of hidden files and descent into
	// hidden directories. Defaults to false.
	IncludeHidden bool

	// DisableGitignore suppresses .gitignore handling when true. Zero
	// value (false) keeps gitignore ENABLED, per C2.
	DisableGitignore bool

	// Includes is the --include glob allow-list. Empty allows every
	// path that survives earlier filters.
	Includes []string

	// Excludes is the --exclude glob deny-list. Empty denies nothing at
	// the exclude stage.
	Excludes []string
}

// Walker emits regular files under a walk root as an iter.Seq2[*File,
// error]. It wraps fs.WalkDir so callers can range over a streaming
// sequence without paying the cost of materializing every file in
// memory first.
//
// Construct via NewWalker; zero-value Walker is not useful. The Walker
// is safe to call Walk on multiple times — each call starts a fresh
// traversal, though the underlying fs.FS must still be safe for
// concurrent Open calls if the caller walks in parallel goroutines
// (the current implementation is single-goroutine; parallel walking
// lands in Drop 8.1).
type Walker struct {
	fsys fs.FS
	root string
	opts WalkOptions
}

// NewWalker returns a Walker rooted at root on fsys. root follows the
// io/fs path convention (forward-slash separators, "." for the current
// directory); see io/fs.ValidPath for the full rules.
//
// NewWalker does no validation of root beyond what fs.WalkDir will do
// at walk time — passing a missing root surfaces as the first yielded
// error when Walk is iterated.
func NewWalker(fsys fs.FS, root string, opts WalkOptions) *Walker {
	return &Walker{fsys: fsys, root: root, opts: opts}
}

// Walk returns an iter.Seq2[*File, error] that yields every regular file
// surviving the configured filters. The returned iterator is single-use
// by convention — callers iterate with range-over-func:
//
//	for f, err := range w.Walk(ctx) {
//	    if err != nil { /* per-entry error, walk continues */ }
//	    // ...
//	}
//
// Per-entry errors (ReadDir failures, per-file opens) are yielded as
// (nil, err) pairs and the walk continues past them so one broken
// directory does not abort the whole traversal. Context cancellation is
// the only condition that terminates iteration: on cancel the iterator
// yields (nil, ctx.Err()) once and returns fs.SkipAll to wind the walk
// down cleanly.
//
// Breaking out of the range loop (via break, return, panic, or a false
// yield anywhere in the caller's pipeline) is supported: the walker
// tracks whether yield has ever returned false and returns fs.SkipAll
// from the WalkDirFunc on the next invocation. Returning nil after a
// false yield would re-invoke yield and panic per the iter package
// contract. See F14.
//
// Symbolic links are yielded as ordinary entries; fs.WalkDir does not
// follow them (stdlib-documented behavior) and rak defers the
// --follow flag to Drop 8.5. A broken symlink manifests as a File whose
// Open call returns an error that unwraps to fs.ErrNotExist.
func (w *Walker) Walk(ctx context.Context) iter.Seq2[*File, error] {
	return func(yield func(*File, error) bool) {
		// yieldOK is flipped to false the first time yield returns false.
		// After that, every WalkDirFunc invocation returns fs.SkipAll so
		// fs.WalkDir winds down without re-invoking yield (which would
		// panic per the iter package contract). See F14.
		yieldOK := true

		// roots accumulates the per-directory gitignore rulesets we have
		// discovered so far. Each entry carries its owning directory so
		// ignore.New can scope pattern matching per F8. We build a fresh
		// ignore.Matcher whenever a new .gitignore is seen, which is
		// cheap relative to the walk itself (compilation is one linear
		// pass per rule set).
		var roots []ignore.GitignoreRoot

		// matcher is rebuilt every time we pick up a new .gitignore.
		// ignore.New never fails for our inputs (includes/excludes are
		// validated up front below), so we cache the construction error
		// in matcherErr and treat any subsequent rebuild failure the
		// same way.
		matcher, matcherErr := ignore.New(roots, w.opts.Includes, w.opts.Excludes)

		// rootDepth is the number of path separators in the walk root.
		// Every path fs.WalkDir reports carries the root as a prefix, so
		// subtracting rootDepth from the visited path's separator count
		// yields "edges from the walk root" (C7). The walk root itself
		// has depth 0.
		rootDepth := slashCount(w.root)

		err := fs.WalkDir(w.fsys, w.root, func(p string, d fs.DirEntry, entryErr error) error {
			// F14 guard #1: once yield has returned false we must not
			// invoke yield again. Bail out of the walk as soon as
			// fs.WalkDir hands control back to us.
			if !yieldOK {
				return fs.SkipAll
			}

			// Context cancellation takes precedence over anything else.
			// Yield the error once and stop — the contract is "context
			// cancel terminates iteration" regardless of what the caller
			// returned from yield. Either way we must never invoke yield
			// again, so flip the guard and return fs.SkipAll.
			if err := ctx.Err(); err != nil {
				yield(nil, err)
				yieldOK = false
				return fs.SkipAll
			}

			// entryErr is set by fs.WalkDir when the initial Stat on
			// root fails or when a directory's ReadDir fails. Yield the
			// wrapped error and continue so a single unreadable subtree
			// does not kill the whole walk (F6). The special values
			// SkipDir / SkipAll are never set here — they are caller
			// returns, not WalkDir inputs.
			if entryErr != nil {
				wrapped := fmt.Errorf("walk %q: %w", p, entryErr)
				if !yield(nil, wrapped) {
					yieldOK = false
					return fs.SkipAll
				}
				// Tell fs.WalkDir to skip the rest of this subtree: we
				// can't descend into a directory whose ReadDir failed,
				// and the root-Stat failure case is terminal anyway.
				if d != nil && d.IsDir() {
					return fs.SkipDir
				}
				return nil
			}

			// Surface the earlier ignore.New failure (invalid include/
			// exclude pattern) as a single yielded error on the first
			// entry and stop. The walker itself has nothing useful to
			// do without a matcher, so we flip the yield guard
			// regardless of the caller's return value.
			if matcherErr != nil {
				yield(nil, fmt.Errorf("walker: %w", matcherErr))
				yieldOK = false
				return fs.SkipAll
			}

			relPath := relFrom(w.root, p)
			isDir := d.IsDir()

			// Hidden-entry filter: the walk root itself is never
			// skipped by the hidden check (its Name may be "." from
			// fs.DirEntry, which IsHidden excludes anyway).
			if !w.opts.IncludeHidden && p != w.root && IsHidden(d.Name()) {
				if isDir {
					return fs.SkipDir
				}
				return nil
			}

			// Depth enforcement: only meaningful when Depth != 0.
			// relPath slash count equals the edge count from the walk
			// root. For a directory, we compare its own depth against
			// the limit and prune once it exceeds. Files use the same
			// rule because a file at depth N lives inside a directory
			// at depth N (the containing dir passed the same check).
			if w.opts.Depth != 0 {
				depth := slashCount(p) - rootDepth
				if isDir {
					// Depth == 1 means "walk the root only, no
					// subdirectories". The root itself is at depth 0
					// and must always be walked; any directory at
					// depth >= Depth is pruned.
					if p != w.root && depth >= w.opts.Depth {
						return fs.SkipDir
					}
				} else if depth >= w.opts.Depth {
					// File living beyond the depth limit. This normally
					// cannot happen because the containing dir would
					// have been pruned; guard anyway for robustness.
					return nil
				}
			}

			// Ingest any .gitignore in this directory before filtering
			// its entries. Do this before the matcher.Match check so
			// the directory's own rules apply to its children.
			if isDir && !w.opts.DisableGitignore {
				if addedRoot := readGitignore(w.fsys, p, relPath); addedRoot != nil {
					roots = append(roots, *addedRoot)
					m, err := ignore.New(roots, w.opts.Includes, w.opts.Excludes)
					if err != nil {
						// Same treatment as a boot-time matcher
						// error: yield once and stop unconditionally.
						yield(nil, fmt.Errorf("walker: %w", err))
						yieldOK = false
						return fs.SkipAll
					}
					matcher = m
				}
			}

			// Matcher check — skip the walk root itself (relPath == "")
			// because the user explicitly asked to walk it.
			if relPath != "" && matcher.Match(relPath, isDir) {
				if isDir {
					return fs.SkipDir
				}
				return nil
			}

			// Directories produce no emission; descent continues.
			if isDir {
				return nil
			}

			// Regular file (or symlink — yielded as a plain entry per
			// F7; File.Open surfaces broken-target errors).
			file := newFile(w.fsys, p, relPath)
			if !yield(file, nil) {
				yieldOK = false
				return fs.SkipAll
			}
			return nil
		})
		// fs.WalkDir returns nil on SkipAll / normal completion and an
		// error only when the WalkDirFunc itself returned a non-sentinel
		// error (we never do). A non-nil err here would indicate a bug
		// in the walker; yield it defensively so callers can observe it
		// rather than silently losing the failure.
		if err != nil && yieldOK {
			_ = yield(nil, fmt.Errorf("walker: %w", err))
		}
	}
}

// slashCount returns the number of forward slashes in p, used for
// depth math. "." (io/fs root convention) counts as zero slashes,
// matching the "root is at depth 0" rule.
func slashCount(p string) int {
	if p == "." || p == "" {
		return 0
	}
	return strings.Count(p, "/")
}

// relFrom strips the walk-root prefix off path, returning the
// walk-relative form Walker uses when talking to the ignore.Matcher.
// The walk root itself (equal to the configured root, including the
// "." convention for the fs root) maps to the empty string.
func relFrom(root, p string) string {
	if p == root {
		return ""
	}
	if root == "." || root == "" {
		return p
	}
	return strings.TrimPrefix(p, root+"/")
}

// readGitignore opens dir/.gitignore on fsys and returns a
// GitignoreRoot scoped to relPath. Missing files (fs.ErrNotExist) and
// any read error are swallowed silently — a missing .gitignore is
// expected, and a permission error on one .gitignore should not abort
// the walk. Returns nil when no .gitignore was ingested.
//
// relPath is the walk-relative directory (empty string for the walk
// root). Patterns keep their raw form; ignore.newGitignoreMatcher's
// wrapped library handles comments, blank lines, negations, and
// dir-only trailing slashes.
func readGitignore(fsys fs.FS, dir, relPath string) *ignore.GitignoreRoot {
	p := path.Join(dir, ".gitignore")
	f, err := fsys.Open(p)
	if err != nil {
		return nil
	}
	defer func() { _ = f.Close() }()

	// Read the whole file; .gitignore files are small in practice (even
	// the Linux kernel's root .gitignore is under 2KB).
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(f); err != nil {
		return nil
	}

	// Split on newline; preserve empty lines and comments verbatim so
	// the downstream library handles them per the gitignore spec.
	scanner := bufio.NewScanner(&buf)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil
	}
	if len(lines) == 0 {
		return nil
	}
	return &ignore.GitignoreRoot{Dir: relPath, Patterns: lines}
}
