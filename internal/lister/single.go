// Package lister — single-file source (v0.1.4).
package lister

import (
	"context"
	"iter"
	"os"
	"path/filepath"

	"github.com/evanmschultz/rak/internal/fileset"
)

// SingleFileLister is a FileLister that yields exactly one file. It is
// selected by Detect when the walk root resolves to a regular file (not a
// directory). The yielded file uses os.DirFS(parent-dir) so Open and Peek
// work identically to paths emitted by GitLister and WalkLister.
//
// SingleFileLister is exported so callers (e.g. lister_test.go) can perform
// type assertions on the value returned by lister.Detect.
type SingleFileLister struct {
	absPath string
}

// newSingleFileLister constructs a SingleFileLister for absPath. absPath
// must already be an absolute path pointing to a regular file (Detect
// resolves and stats the path before calling newSingleFileLister).
func newSingleFileLister(absPath string) *SingleFileLister {
	return &SingleFileLister{absPath: absPath}
}

// List returns an iterator that yields exactly one file. The file is opened
// via os.DirFS on the parent directory so that Open and Peek operate on the
// real filesystem without any git or walker machinery. The iterator honours
// the FileLister contract: context cancellation yields (nil, ctx.Err()) and
// terminates; if the caller's yield returns false the iterator stops.
func (s *SingleFileLister) List(ctx context.Context) iter.Seq2[*fileset.File, error] {
	return func(yield func(*fileset.File, error) bool) {
		if ctx.Err() != nil {
			yield(nil, ctx.Err())
			return
		}
		dir := filepath.Dir(s.absPath)
		base := filepath.Base(s.absPath)
		fsys := os.DirFS(dir)
		// path and relPath are both the bare basename so that dirKey(relPath)
		// returns "." and labelDirectories later rewrites "." to the user-
		// supplied filename, giving the correct user-facing directory label.
		f := fileset.NewFile(fsys, base, base)
		yield(f, nil)
	}
}

// compile-time assertion: SingleFileLister must satisfy FileLister.
var _ FileLister = (*SingleFileLister)(nil)
