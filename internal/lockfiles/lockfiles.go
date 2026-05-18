// Package lockfiles provides a denylist of well-known machine-generated
// lockfile basenames and a helper to test whether a file path matches one.
// rak excludes lockfiles from counts by default (v0.2.0 behavior change) so
// that output reflects code your team wrote rather than machine-generated
// dependency manifests. Pass --include-lockfiles to count them.
package lockfiles

import (
	"path/filepath"
	"strings"
)

// denied is the set of well-known lockfile basenames that rak skips by
// default. Match is case-insensitive (keys are stored lower-case; inputs are
// lowercased before lookup). Entries cover the major package ecosystems:
// Go, Node (npm/yarn/pnpm), Rust, Ruby, Python (pip/poetry), PHP, Elixir.
var denied = map[string]struct{}{
	"go.sum":            {},
	"package-lock.json": {},
	"yarn.lock":         {},
	"pnpm-lock.yaml":    {},
	"cargo.lock":        {},
	"gemfile.lock":      {},
	"pipfile.lock":      {},
	"poetry.lock":       {},
	"composer.lock":     {},
	"mix.lock":          {},
}

// IsLockfile reports whether the basename of path matches a well-known
// lockfile name. Match is case-insensitive on the basename only; directory
// components are ignored.
//
//	IsLockfile("go.sum")                     // true
//	IsLockfile("Cargo.lock")                 // true (case-insensitive)
//	IsLockfile("/path/to/sub/Cargo.lock")    // true (basename match)
//	IsLockfile("main.go")                    // false
func IsLockfile(path string) bool {
	base := strings.ToLower(filepath.Base(path))
	_, ok := denied[base]
	return ok
}
