package ignore_test

import (
	"errors"
	"testing"

	"github.com/bmatcuk/doublestar/v4"

	"github.com/evanmschultz/rak/internal/ignore"
)

// TestMatcher_Empty confirms a matcher with no patterns and no
// gitignore roots lets every path through. This is the degenerate
// baseline: `rak` without any filter flags and without an encountered
// .gitignore must not drop files.
func TestMatcher_Empty(t *testing.T) {
	t.Parallel()
	m, err := ignore.New(nil, nil, nil)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	cases := []struct {
		path  string
		isDir bool
	}{
		{"foo.go", false},
		{"sub/bar.txt", false},
		{"deeply/nested/thing", false},
		{"some/dir", true},
	}
	for _, c := range cases {
		if m.Match(c.path, c.isDir) {
			t.Errorf("empty matcher dropped %q (isDir=%v); want kept", c.path, c.isDir)
		}
	}
}

// TestMatcher_GitignoreOnly exercises the gitignore branch without
// any include/exclude flags. Covers the spec features listed in the
// unit acceptance: negation, dir-only, double-star, character class,
// and hierarchical scoping (F8).
func TestMatcher_GitignoreOnly(t *testing.T) {
	t.Parallel()

	roots := []ignore.GitignoreRoot{
		{Dir: "", Patterns: []string{
			"*.log",
			"!keep.log",
			"node_modules/",
			"**/vendor",
			"[abc].txt",
			"# this is a comment, should parse cleanly",
			"",
		}},
		{Dir: "nested", Patterns: []string{
			"secret.conf",
		}},
	}

	m, err := ignore.New(roots, nil, nil)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	cases := []struct {
		name   string
		path   string
		isDir  bool
		ignore bool
	}{
		{"wildcard_hits", "debug.log", false, true},
		{"negation_reincludes", "keep.log", false, false},
		{"dir_only_pattern_matches_dir", "node_modules", true, true},
		{"dir_only_pattern_skips_file_of_same_name", "node_modules", false, false},
		{"dir_only_pattern_matches_child", "node_modules/foo.js", false, true},
		{"double_star_at_root", "vendor", true, true},
		{"double_star_subdir", "src/vendor", true, true},
		// go-gitignore matches a "**/vendor" directory pattern against
		// paths *inside* the matched vendor dir too — git's standard
		// behavior for implicit-directory ignores. This test pins
		// that real semantics rather than an idealized spec.
		{"double_star_matches_children", "src/vendor/x.go", false, true},
		{"char_class_hit", "a.txt", false, true},
		{"char_class_miss", "d.txt", false, false},
		{"unmatched_plain_file", "README.md", false, false},

		// Hierarchical scoping (F8): secret.conf is only ignored under
		// nested/, not at the walk root or under sibling dirs.
		{"scoped_rule_hits_inside_scope", "nested/secret.conf", false, true},
		{"scoped_rule_misses_outside_scope", "secret.conf", false, false},
		{"scoped_rule_misses_in_sibling", "other/secret.conf", false, false},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := m.Match(c.path, c.isDir)
			if got != c.ignore {
				t.Errorf("Match(%q, isDir=%v) = %v; want %v",
					c.path, c.isDir, got, c.ignore)
			}
		})
	}
}

// TestMatcher_IncludeOnly pins the --include semantics: empty slice
// allows everything, non-empty slice is an allow-list.
func TestMatcher_IncludeOnly(t *testing.T) {
	t.Parallel()

	m, err := ignore.New(nil, []string{"*.go", "docs/**/*.md"}, nil)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	cases := []struct {
		name   string
		path   string
		ignore bool
	}{
		{"go_survives", "main.go", false},
		{"md_under_docs_survives", "docs/intro/getting-started.md", false},
		{"root_md_dropped", "README.md", true},
		{"txt_dropped", "notes.txt", true},
		{"go_in_subdir_dropped", "cmd/rak/root.go", true}, // "*.go" does not cross '/'
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := m.Match(c.path, false)
			if got != c.ignore {
				t.Errorf("Match(%q) = %v; want %v", c.path, got, c.ignore)
			}
		})
	}
}

// TestMatcher_ExcludeOnly pins the --exclude semantics: empty slice
// denies nothing, non-empty slice is a deny-list.
func TestMatcher_ExcludeOnly(t *testing.T) {
	t.Parallel()

	m, err := ignore.New(nil, nil, []string{"*_test.go", "**/vendor/**"})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	cases := []struct {
		name   string
		path   string
		ignore bool
	}{
		{"test_file_dropped", "foo_test.go", true},
		{"non_test_go_kept", "foo.go", false},
		{"vendor_leaf_dropped", "cmd/rak/vendor/x.go", true},
		// doublestar.Match treats a leading '**/' as allowing zero
		// path components, so "**/vendor/**" also matches the bare
		// "vendor" string. Callers who want "match only under vendor,
		// not vendor itself" should use "**/vendor/*" instead.
		{"vendor_dir_also_matched", "vendor", true},
		{"unmatched_plain", "README.md", false},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := m.Match(c.path, false)
			if got != c.ignore {
				t.Errorf("Match(%q) = %v; want %v", c.path, got, c.ignore)
			}
		})
	}
}

// TestMatcher_AllThreeCombined runs every stage simultaneously to
// confirm the New composition wires correctly when no precedence
// conflict is in play.
func TestMatcher_AllThreeCombined(t *testing.T) {
	t.Parallel()

	roots := []ignore.GitignoreRoot{{
		Dir:      "",
		Patterns: []string{"*.log"},
	}}
	m, err := ignore.New(roots, []string{"*.go"}, []string{"*_test.go"})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	cases := []struct {
		name   string
		path   string
		ignore bool
	}{
		{"go_file_survives_all_stages", "main.go", false},
		{"test_file_dropped_by_exclude", "main_test.go", true},
		{"log_file_dropped_by_gitignore", "debug.log", true},
		{"txt_dropped_because_not_in_include", "README.txt", true},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := m.Match(c.path, false)
			if got != c.ignore {
				t.Errorf("Match(%q) = %v; want %v", c.path, got, c.ignore)
			}
		})
	}
}

// TestMatcher_Precedence_ExcludeBeatsGitignoreNegate locks the F2 / F3
// pin: if a user's --exclude catches a path that gitignore would have
// re-included via '!' negation, --exclude still wins because it runs
// before the gitignore stage.
func TestMatcher_Precedence_ExcludeBeatsGitignoreNegate(t *testing.T) {
	t.Parallel()

	roots := []ignore.GitignoreRoot{{
		Dir:      "",
		Patterns: []string{"*.go", "!important.go"},
	}}
	m, err := ignore.New(roots, nil, []string{"important.go"})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	// Even though .gitignore re-includes important.go via '!', the
	// --exclude pattern runs first and drops it.
	if !m.Match("important.go", false) {
		t.Errorf("expected --exclude to beat gitignore negation; got kept")
	}
	// A file the exclude doesn't touch but gitignore negates stays.
	// (nothing in .gitignore actually negates a different file here,
	// so we double-check the baseline: other.go is plain *.go, dropped
	// by gitignore.)
	if !m.Match("other.go", false) {
		t.Errorf("expected gitignore *.go rule to drop other.go; got kept")
	}
}

// TestMatcher_Precedence_IncludeDoesNotOverrideExclude locks the
// inverse direction: --include cannot rescue a path that --exclude
// already condemned.
func TestMatcher_Precedence_IncludeDoesNotOverrideExclude(t *testing.T) {
	t.Parallel()

	m, err := ignore.New(nil, []string{"*.go"}, []string{"main.go"})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	if !m.Match("main.go", false) {
		t.Errorf("expected --exclude to beat --include; main.go was kept")
	}
	if m.Match("other.go", false) {
		t.Errorf("expected --include to admit other.go; got dropped")
	}
}

// TestMatcher_Precedence_IncludeAfterGitignore confirms that a file
// already dropped by .gitignore remains dropped even when it matches
// a broad --include. This nails the F3 order: gitignore decides
// before include looks at the path.
func TestMatcher_Precedence_IncludeAfterGitignore(t *testing.T) {
	t.Parallel()

	roots := []ignore.GitignoreRoot{{
		Dir:      "",
		Patterns: []string{"generated.go"},
	}}
	m, err := ignore.New(roots, []string{"*.go"}, nil)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	if !m.Match("generated.go", false) {
		t.Errorf("expected gitignore to drop generated.go; got kept")
	}
	if m.Match("real.go", false) {
		t.Errorf("expected --include '*.go' to admit real.go; got dropped")
	}
}

// TestMatcher_NoGitignore_EquivalentToEmptyRoots confirms the
// --no-gitignore escape hatch semantics: passing an empty roots slice
// (what Unit 3.5 does when --no-gitignore is set) disables the
// gitignore stage entirely.
func TestMatcher_NoGitignore_EquivalentToEmptyRoots(t *testing.T) {
	t.Parallel()

	// Matcher A: has a root .gitignore that would drop *.log.
	roots := []ignore.GitignoreRoot{{
		Dir:      "",
		Patterns: []string{"*.log"},
	}}
	withGI, err := ignore.New(roots, nil, nil)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if !withGI.Match("debug.log", false) {
		t.Fatalf("baseline: gitignore should drop debug.log")
	}

	// Matcher B: same situation minus the gitignore roots. The walker
	// passes nil here when --no-gitignore is set.
	withoutGI, err := ignore.New(nil, nil, nil)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if withoutGI.Match("debug.log", false) {
		t.Errorf("--no-gitignore equivalent (nil roots) should let debug.log through")
	}
}

// TestMatcher_InvalidGlob_Include ensures New surfaces a malformed
// include pattern as an error and refuses to return a partial matcher.
func TestMatcher_InvalidGlob_Include(t *testing.T) {
	t.Parallel()

	// doublestar treats '[' with no closing ']' as ErrBadPattern.
	m, err := ignore.New(nil, []string{"src/[broken"}, nil)
	if err == nil {
		t.Fatalf("expected error for malformed include; got matcher %#v", m)
	}
	if !errors.Is(err, doublestar.ErrBadPattern) {
		t.Errorf("expected error to wrap doublestar.ErrBadPattern; got %v", err)
	}
}

// TestMatcher_InvalidGlob_Exclude mirrors the above for the exclude
// channel.
func TestMatcher_InvalidGlob_Exclude(t *testing.T) {
	t.Parallel()

	m, err := ignore.New(nil, nil, []string{"src/[broken"})
	if err == nil {
		t.Fatalf("expected error for malformed exclude; got matcher %#v", m)
	}
	if !errors.Is(err, doublestar.ErrBadPattern) {
		t.Errorf("expected error to wrap doublestar.ErrBadPattern; got %v", err)
	}
}
