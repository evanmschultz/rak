// Package fileset exposes the File type and (in Unit 3.3) the Walker that
// emits files over an io/fs.FS tree.
//
// File carries the walk-relative path plus the underlying fs.FS so the same
// type works uniformly over real disk (os.DirFS) and in-memory test trees
// (testing/fstest.MapFS). Open and Peek both go through fs.FS.Open; Peek is
// stateless (open-read-close per call) so binary detection (Unit 3.4) and
// shebang sniff (Drop 4.1) can call it independently without coordinating a
// shared cursor. See F4 in DROP_3's PLAN.md for the pin.
//
// All paths carried by File use forward-slash separators regardless of host
// OS, matching the io/fs convention. The Walker (Unit 3.3) is responsible
// for normalizing OS-native separators before constructing File values.
package fileset

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"strings"
)

// File names a single regular file in a walked tree. Path is the full
// walk-relative path as the Walker saw it (the argument fs.WalkDir passed to
// the WalkDirFunc); RelPath is the path relative to the walk root. For a
// walk rooted at ".", Path and RelPath are identical; for a walk rooted at a
// subdirectory the Walker strips the root prefix when building RelPath so
// downstream code (matchers, renderers) sees paths anchored at the user's
// chosen root.
//
// Zero-value File is not useful; construct via newFile (unexported). Tests
// use newFile directly; the Walker in Unit 3.3 calls it while emitting.
// Fields are exported so callers can read metadata without a getter — see
// main/CLAUDE.md § "Go-Idiomatic Naming Rules" rule 4.
type File struct {
	// Path is the walk-relative path as fs.WalkDir passed it to the
	// WalkDirFunc (includes the walk root prefix).
	Path string
	// RelPath is the path relative to the walk root (no root prefix).
	RelPath string

	// fs is the underlying file system used to open the file. Unexported
	// so callers cannot bypass Open/Peek and read via fs directly.
	fs fs.FS
}

// newFile constructs a File value. It is the only sanctioned constructor;
// Walker (Unit 3.3) and the package tests both call it. Keeping the
// constructor unexported makes it explicit that callers outside the package
// have no business fabricating File values — they receive them from Walker.
func newFile(fsys fs.FS, path, relPath string) *File {
	return &File{
		Path:    path,
		RelPath: relPath,
		fs:      fsys,
	}
}

// NewFile constructs a File for the given path. Callers outside
// internal/fileset use this to create File handles when they have obtained a
// path from a non-Walker source (e.g. GitLister).
func NewFile(fsys fs.FS, path, relPath string) *File {
	return newFile(fsys, path, relPath)
}

// Open opens the file for reading. The returned io.ReadCloser is the
// fs.File returned by the underlying fs.FS; callers must Close it.
//
// On error the returned error wraps the underlying cause with the prefix
// open %q: ... so callers can both identify the failing path and inspect
// the root cause via errors.Is / errors.As. In particular,
// errors.Is(err, fs.ErrNotExist) returns true when the path is missing.
func (f *File) Open() (io.ReadCloser, error) {
	rc, err := f.fs.Open(f.Path)
	if err != nil {
		return nil, fmt.Errorf("open %q: %w", f.Path, err)
	}
	return rc, nil
}

// Peek opens the file, reads up to n bytes into a fresh buffer, closes the
// file, and returns the bytes actually read. Short files return what they
// have with a nil error; a file with at least n bytes returns exactly n.
//
// Peek is stateless: each call opens and closes the file independently, so
// repeated Peek calls on the same *File return identical bytes. This is the
// F4 contract that binary detection (Unit 3.4) and shebang sniff (Drop 4.1)
// both rely on.
//
// For n <= 0 Peek returns a nil slice with no error; it does not open the
// file. Errors from Open, Read, or Close (when the read itself succeeded)
// are wrapped with the same open %q: ... prefix Open uses.
func (f *File) Peek(n int) ([]byte, error) {
	if n <= 0 {
		return nil, nil
	}
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	// Discard Close error: Peek has already consumed the bytes it needed, and
	// fs.File Close failures on a read-only handle do not invalidate the bytes
	// already returned. Read errors are surfaced above via io.ReadFull.
	defer func() { _ = rc.Close() }()

	buf := make([]byte, n)
	k, err := io.ReadFull(rc, buf)
	if err == nil || errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
		return buf[:k], nil
	}
	return nil, fmt.Errorf("open %q: %w", f.Path, err)
}

// IsHidden reports whether a single path element (a basename, not a full
// path) names a hidden entry. A name is hidden when it starts with a dot,
// excluding the special directory names "." and ".." which denote the
// current and parent directory and are not hidden entries in any shell's
// sense. An empty name is not hidden.
//
// The Walker (Unit 3.3) calls IsHidden on fs.DirEntry.Name() to decide
// whether to skip an entry when IncludeHidden is false. See C3 in DROP_3's
// PLAN.md.
func IsHidden(name string) bool {
	if name == "" || name == "." || name == ".." {
		return false
	}
	return strings.HasPrefix(name, ".")
}
