package lockfiles_test

import (
	"testing"

	"github.com/evanmschultz/rak/internal/lockfiles"
)

// TestIsLockfile verifies that all 10 denylist entries are recognized in
// lowercase, mixed-case, and uppercase forms, and that well-known non-lockfile
// names (including names that contain "lock" as a substring but are not in the
// denylist) are correctly rejected.
func TestIsLockfile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path string
		want bool
	}{
		// Denylist entries — canonical (lowercase) forms.
		{"go.sum", true},
		{"package-lock.json", true},
		{"yarn.lock", true},
		{"pnpm-lock.yaml", true},
		{"Cargo.lock", true},
		{"Gemfile.lock", true},
		{"Pipfile.lock", true},
		{"poetry.lock", true},
		{"composer.lock", true},
		{"mix.lock", true},

		// Denylist entries — UPPERCASE forms.
		{"GO.SUM", true},
		{"PACKAGE-LOCK.JSON", true},
		{"YARN.LOCK", true},
		{"PNPM-LOCK.YAML", true},
		{"CARGO.LOCK", true},
		{"GEMFILE.LOCK", true},
		{"PIPFILE.LOCK", true},
		{"POETRY.LOCK", true},
		{"COMPOSER.LOCK", true},
		{"MIX.LOCK", true},

		// Denylist entries — mixed-case forms.
		{"Go.Sum", true},
		{"Package-Lock.Json", true},
		{"Yarn.Lock", true},
		{"Pnpm-Lock.Yaml", true},

		// Basename match — directory prefix must be ignored.
		{"/path/to/sub/Cargo.lock", true},
		{"some/nested/dir/go.sum", true},
		{"a/b/c/package-lock.json", true},

		// Non-lockfile examples.
		{"main.go", false},
		{"README.md", false},
		{"lockfiles.txt", false}, // contains "lock" substring but not in denylist
		{"go.mod", false},
		{"package.json", false},
		{".gitignore", false},
		{"Makefile", false},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			t.Parallel()
			got := lockfiles.IsLockfile(tc.path)
			if got != tc.want {
				t.Errorf("IsLockfile(%q) = %v, want %v", tc.path, got, tc.want)
			}
		})
	}
}
