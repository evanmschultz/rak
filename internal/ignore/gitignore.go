package ignore

import (
	"strings"

	gitignore "github.com/sabhiram/go-gitignore"
)

// gitignoreMatcher holds one compiled per-directory ruleset per
// GitignoreRoot the caller supplied to New. Matching is hierarchical:
// a pattern in foo/.gitignore applies only to paths under foo/, never
// to siblings of foo/. This mirrors git's actual per-directory rule
// scoping (F8).
//
// The struct is private; callers construct it via New and interact
// through the Matcher interface.
type gitignoreMatcher struct {
	rules []gitignoreRule
}

// gitignoreRule pairs a precompiled GitIgnore with its owning
// directory so Match can decide whether a path is in-scope before
// delegating to the library.
type gitignoreRule struct {
	dir      string // forward-slash, empty string == walk root
	compiled *gitignore.GitIgnore
}

// newGitignoreMatcher precompiles every GitignoreRoot.Patterns into a
// *gitignore.GitIgnore keyed by the root's Dir. Empty GitignoreRoot
// slices produce a matcher that ignores nothing.
func newGitignoreMatcher(roots []GitignoreRoot) *gitignoreMatcher {
	if len(roots) == 0 {
		return &gitignoreMatcher{}
	}
	rules := make([]gitignoreRule, 0, len(roots))
	for _, r := range roots {
		// CompileIgnoreLines never returns an error; it stores the
		// compiled regexps on the returned GitIgnore and the library
		// exposes no parse-failure surface.
		rules = append(rules, gitignoreRule{
			dir:      r.Dir,
			compiled: gitignore.CompileIgnoreLines(r.Patterns...),
		})
	}
	return &gitignoreMatcher{rules: rules}
}

// match reports whether relPath is ignored by any in-scope gitignore
// ruleset. In-scope means the rule's directory prefixes relPath (the
// walk root matches everything). The path passed into the library
// gets its dir prefix stripped so patterns like /foo (anchored in the
// rule's own directory) resolve correctly.
//
// isDir triggers the dir-only trailing-slash suffix: go-gitignore
// matches a pattern like "node_modules/" against "node_modules/X"
// but not against the bare string "node_modules", so appending "/"
// for directories lets the library handle dir-only patterns without
// the caller reimplementing the rule.
func (g *gitignoreMatcher) match(relPath string, isDir bool) bool {
	if len(g.rules) == 0 {
		return false
	}
	for _, rule := range g.rules {
		scoped, ok := scopePath(relPath, rule.dir)
		if !ok {
			continue
		}
		probe := scoped
		if isDir {
			probe += "/"
		}
		if rule.compiled.MatchesPath(probe) {
			return true
		}
	}
	return false
}

// scopePath strips the rule's dir prefix off relPath. It returns the
// remainder and a boolean signaling whether the path was in-scope. An
// empty dir matches everything (walk-root .gitignore). A dir equal to
// relPath itself is in-scope with an empty remainder.
func scopePath(relPath, dir string) (string, bool) {
	if dir == "" {
		return relPath, true
	}
	if relPath == dir {
		return "", true
	}
	prefix := dir + "/"
	if strings.HasPrefix(relPath, prefix) {
		return strings.TrimPrefix(relPath, prefix), true
	}
	return "", false
}
