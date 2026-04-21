// Package ignore unifies gitignore parsing and --include / --exclude glob
// matching behind a single Matcher interface. It is a leaf of rak's
// internal dependency DAG: zero internal deps, no filesystem access.
//
// Matchers compose three sub-filters (exclude globs, gitignore roots,
// include globs) in the fixed precedence order documented on New:
//
//	--exclude  →  .gitignore  →  --include
//
// Every call to Match returns true when the path should be **ignored /
// filtered out**. This is the opposite polarity of fs.WalkDirFunc's
// fs.SkipDir, so callers must read the return value carefully. (See F1
// in DROP_3's PLAN.md for the convention pin.)
//
// All paths handed to Match use forward-slash separators regardless of
// host OS, matching the io/fs convention. The Walker (internal/fileset)
// is responsible for normalizing OS-native separators before calling
// into this package.
package ignore

// Matcher decides whether a path is filtered out of the walk.
//
// Match returns true when relPath should be **ignored** (skipped,
// dropped, not emitted). Returning false means the path survives all
// configured filters. relPath is the path relative to the walk root
// using forward-slash separators on every platform. isDir signals that
// the path names a directory so dir-only gitignore patterns (trailing
// slash) can match correctly.
type Matcher interface {
	// Match reports whether relPath should be ignored. See the package
	// doc for the F1 convention: true means "drop this path".
	Match(relPath string, isDir bool) bool
}

// GitignoreRoot is a pre-parsed .gitignore payload scoped to a single
// directory. Dir is the walk-relative directory that owns Patterns
// (forward-slash separators, empty string for the walk root). Patterns
// is the raw .gitignore file content, one entry per line; comments,
// blanks, negations, and dir-only patterns are handled by the gitignore
// sub-matcher per the gitignore spec.
//
// The Walker constructs one GitignoreRoot per directory it enters that
// contains a .gitignore file (landing in Unit 3.3). This package does
// no disk IO — it consumes the pre-read Patterns only.
type GitignoreRoot struct {
	// Dir is the walk-relative directory the patterns apply to. Empty
	// string denotes the walk root itself.
	Dir string
	// Patterns are the raw .gitignore lines in the order they were read.
	Patterns []string
}

// New returns a Matcher composing the three sub-matchers in the fixed
// precedence order: --exclude beats .gitignore beats --include.
//
// The concrete behavior for a single call:
//
//  1. If any exclude pattern matches relPath, return true (drop).
//  2. Else if any gitignore root (whose Dir prefixes relPath) matches,
//     return true (drop).
//  3. Else if includes is non-empty and no include pattern matches,
//     return true (drop).
//  4. Else return false (keep).
//
// An empty includes slice means "allow everything that survives the
// earlier filters"; an empty excludes slice means "deny nothing at the
// exclude stage". An empty roots slice (or a caller passing nil)
// disables the gitignore stage entirely — matching the walker's
// --no-gitignore escape hatch wired in Unit 3.5.
//
// New validates each include and exclude pattern via doublestar. A
// malformed pattern is reported as an error and no Matcher is
// returned.
func New(roots []GitignoreRoot, includes, excludes []string) (Matcher, error) {
	excludeM, err := newGlobMatcher(excludes)
	if err != nil {
		return nil, err
	}
	includeM, err := newGlobMatcher(includes)
	if err != nil {
		return nil, err
	}
	gi := newGitignoreMatcher(roots)
	return &compositeMatcher{
		exclude:     excludeM,
		gitignore:   gi,
		include:     includeM,
		hasIncludes: len(includes) > 0,
	}, nil
}

// compositeMatcher chains the three sub-matchers per the New contract.
type compositeMatcher struct {
	exclude     *globMatcher
	gitignore   *gitignoreMatcher
	include     *globMatcher
	hasIncludes bool
}

// Match applies exclude → gitignore → include in that order. See the
// Matcher interface docs and the F1 pin for the return convention.
func (c *compositeMatcher) Match(relPath string, isDir bool) bool {
	if c.exclude.match(relPath) {
		return true
	}
	if c.gitignore.match(relPath, isDir) {
		return true
	}
	if c.hasIncludes && !c.include.match(relPath) {
		return true
	}
	return false
}
