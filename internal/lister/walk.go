// Package lister — Walker adapter (Unit 4.3).
package lister

import (
	"context"
	"io/fs"
	"iter"

	"github.com/evanmschultz/rak/internal/fileset"
)

// WalkLister is a thin adapter that wraps fileset.Walker and satisfies the
// FileLister interface. All WalkOptions filters are applied by the underlying
// Walker; WalkLister performs zero additional filtering (F22).
type WalkLister struct {
	walker *fileset.Walker
}

// newWalkLister constructs a WalkLister backed by fileset.NewWalker with the
// supplied fs.FS, root, and options. It is the unexported variant called by
// Detect in the non-git branch.
func newWalkLister(fsys fs.FS, root string, opts fileset.WalkOptions) *WalkLister {
	return &WalkLister{walker: fileset.NewWalker(fsys, root, opts)}
}

// NewWalkLister constructs a WalkLister with an injected fs.FS for tests that
// need to bypass lister.Detect. It is the exported counterpart to newWalkLister
// and has identical behaviour; the split exists so cmd/rak test helpers (in a
// different package) can construct a WalkLister with a testing/fstest.MapFS
// directly without going through the git-detection path.
func NewWalkLister(fsys fs.FS, root string, opts fileset.WalkOptions) *WalkLister {
	return &WalkLister{walker: fileset.NewWalker(fsys, root, opts)}
}

// List returns an iterator that yields every file emitted by the underlying
// Walker. It is a pure delegation to walker.Walk(ctx) — no filtering is
// applied here (F22).
func (wl *WalkLister) List(ctx context.Context) iter.Seq2[*fileset.File, error] {
	return wl.walker.Walk(ctx)
}

// compile-time assertion: WalkLister must satisfy FileLister.
var _ FileLister = (*WalkLister)(nil)
