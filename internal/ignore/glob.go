package ignore

import (
	"fmt"

	"github.com/bmatcuk/doublestar/v4"
)

// globMatcher wraps a validated pattern slice. Empty slice matches
// nothing; any non-empty slice matches when at least one pattern hits.
//
// The patterns are consumed by doublestar.Match, which splits both
// pattern and name on forward slash ('/') on every platform. This is
// the correct choice for relPath values supplied by the walker (C6 /
// io/fs convention) and keeps behavior portable. doublestar.PathMatch
// is deliberately rejected because it uses the OS path separator and
// would mis-match forward-slash relPath values on non-Unix hosts.
type globMatcher struct {
	patterns []string
}

// newGlobMatcher validates each pattern via doublestar.ValidatePattern
// so a malformed pattern fails construction in New rather than at
// first use. Returning nil matcher (rather than an error) when the
// slice is empty lets the composite collapse the empty-include
// fast-path cleanly.
func newGlobMatcher(patterns []string) (*globMatcher, error) {
	if len(patterns) == 0 {
		return &globMatcher{}, nil
	}
	for _, p := range patterns {
		if !doublestar.ValidatePattern(p) {
			return nil, fmt.Errorf("ignore: invalid glob pattern %q: %w", p, doublestar.ErrBadPattern)
		}
	}
	return &globMatcher{patterns: patterns}, nil
}

// match reports whether name matches any configured pattern. An empty
// matcher (no patterns) always returns false — the composite handles
// the "empty include allows everything" rule via a separate
// hasIncludes flag, so this method stays semantically uniform.
//
// The error from doublestar.Match is intentionally dropped: patterns
// were validated at construction, so Match can only fail on a
// malformed input that already would not have made it past
// newGlobMatcher.
func (g *globMatcher) match(name string) bool {
	for _, p := range g.patterns {
		ok, _ := doublestar.Match(p, name)
		if ok {
			return true
		}
	}
	return false
}
