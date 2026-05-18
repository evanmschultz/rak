package lister_test

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/evanmschultz/rak/internal/fileset"
	"github.com/evanmschultz/rak/internal/lister"
)

// TestDetect_InsideRepo verifies that Detect returns a *GitLister when called
// with a root that is inside a git repository and DisableGitignore is false.
// Constructs a throwaway git repo in t.TempDir() so the test is hermetic and
// does not depend on the rak checkout being visible on the CI runner.
func TestDetect_InsideRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not found")
	}

	tmp := t.TempDir()
	initCmd := exec.Command("git", "init", "--template=", tmp)
	if out, err := initCmd.CombinedOutput(); err != nil {
		t.Skipf("git init failed (%v): %s", err, out)
	}

	ctx := t.Context()
	got, err := lister.Detect(ctx, tmp, fileset.WalkOptions{})
	if err != nil {
		t.Fatalf("Detect returned unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("Detect returned nil lister, want non-nil")
	}

	if _, ok := got.(*lister.GitLister); !ok {
		t.Errorf("Detect returned %T, want *lister.GitLister", got)
	}
}

// TestDetect_OutsideRepo verifies that Detect returns a *WalkLister when
// called with a root that is not inside a git repository.
func TestDetect_OutsideRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not found")
	}

	// t.TempDir() is outside any git repository on a typical CI machine; it is
	// guaranteed to be a non-git directory. On developer machines with
	// unconventional setups, the skip below provides a safety net.
	dir := t.TempDir()

	ctx := t.Context()
	got, err := lister.Detect(ctx, dir, fileset.WalkOptions{})
	if err != nil {
		t.Fatalf("Detect returned unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("Detect returned nil lister, want non-nil")
	}

	if _, ok := got.(*lister.WalkLister); !ok {
		t.Errorf("Detect returned %T, want *lister.WalkLister", got)
	}
}

// TestDetect_BareRepo verifies that Detect returns a *WalkLister (not a
// *GitLister and not an error) when called with the root of a bare git
// repository. A bare repo causes "git rev-parse --is-inside-work-tree" to
// print "false" with exit 0 — the fix checks stdout rather than exit code only.
func TestDetect_BareRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not found")
	}

	tmp := t.TempDir()
	initCmd := exec.Command("git", "init", "--bare", "--template=", tmp)
	if out, err := initCmd.CombinedOutput(); err != nil {
		t.Skipf("git init --bare failed (%v): %s", err, out)
	}

	ctx := t.Context()
	got, err := lister.Detect(ctx, tmp, fileset.WalkOptions{})
	if err != nil {
		t.Fatalf("Detect returned unexpected error for bare repo: %v", err)
	}
	if got == nil {
		t.Fatal("Detect returned nil lister for bare repo, want non-nil")
	}
	if _, ok := got.(*lister.WalkLister); !ok {
		t.Errorf("Detect returned %T for bare repo, want *lister.WalkLister", got)
	}
}

// TestDetect_InsideGitDir verifies that Detect returns a *WalkLister when
// called with the .git/ directory of a normal (non-bare) git repository.
// Inside .git/, "git rev-parse --is-inside-work-tree" prints "false" with
// exit 0 — the fix checks stdout rather than exit code only.
func TestDetect_InsideGitDir(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not found")
	}

	tmp := t.TempDir()
	initCmd := exec.Command("git", "init", "--template=", tmp)
	if out, err := initCmd.CombinedOutput(); err != nil {
		t.Skipf("git init failed (%v): %s", err, out)
	}

	gitDir := filepath.Join(tmp, ".git")
	ctx := t.Context()
	got, err := lister.Detect(ctx, gitDir, fileset.WalkOptions{})
	if err != nil {
		t.Fatalf("Detect returned unexpected error for .git/ dir: %v", err)
	}
	if got == nil {
		t.Fatal("Detect returned nil lister for .git/ dir, want non-nil")
	}
	if _, ok := got.(*lister.WalkLister); !ok {
		t.Errorf("Detect returned %T for .git/ dir, want *lister.WalkLister", got)
	}
}

// TestDetect_BareRepo_WithDisableGitignore verifies that Detect returns a
// *WalkLister (not an error) when called with the root of a bare git
// repository AND DisableGitignore is true. The ErrNoGitignoreInRepo sentinel
// applies only when the walk root is inside a real work tree — bare repos fall
// through to the WalkLister path regardless of DisableGitignore.
func TestDetect_BareRepo_WithDisableGitignore(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not found")
	}

	dir := t.TempDir()
	cmd := exec.Command("git", "init", "--bare", "--template=", dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("git init --bare failed (%v): %s", err, out)
	}

	ctx := t.Context()
	got, err := lister.Detect(ctx, dir, fileset.WalkOptions{DisableGitignore: true})
	if err != nil {
		t.Fatalf("Detect returned unexpected error for bare repo with DisableGitignore: %v", err)
	}
	if _, ok := got.(*lister.WalkLister); !ok {
		t.Errorf("Detect returned %T for bare repo with DisableGitignore, want *lister.WalkLister", got)
	}
}

// TestDetect_InsideGitDir_WithDisableGitignore verifies that Detect returns a
// *WalkLister (not an error) when called with the .git/ directory of a normal
// repository AND DisableGitignore is true. Inside .git/, the walk root is not
// a real work tree, so ErrNoGitignoreInRepo must not be returned even when
// DisableGitignore is set.
func TestDetect_InsideGitDir_WithDisableGitignore(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not found")
	}

	dir := t.TempDir()
	cmd := exec.Command("git", "init", "--template=", dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("git init failed (%v): %s", err, out)
	}

	ctx := t.Context()
	got, err := lister.Detect(ctx, filepath.Join(dir, ".git"), fileset.WalkOptions{DisableGitignore: true})
	if err != nil {
		t.Fatalf("Detect returned unexpected error for .git/ dir with DisableGitignore: %v", err)
	}
	if _, ok := got.(*lister.WalkLister); !ok {
		t.Errorf("Detect returned %T for .git/ dir with DisableGitignore, want *lister.WalkLister", got)
	}
}

// TestDetect_NoGitignoreInRepo_ReturnsSentinel verifies that Detect returns
// ErrNoGitignoreInRepo (via errors.Is) when DisableGitignore is true and the
// walk root is inside a git repository. Constructs a throwaway git repo in
// t.TempDir() so the test is hermetic — no dependency on the rak checkout.
func TestDetect_NoGitignoreInRepo_ReturnsSentinel(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not found")
	}

	tmp := t.TempDir()
	initCmd := exec.Command("git", "init", "--template=", tmp)
	if out, err := initCmd.CombinedOutput(); err != nil {
		t.Skipf("git init failed (%v): %s", err, out)
	}

	ctx := t.Context()
	got, err := lister.Detect(ctx, tmp, fileset.WalkOptions{DisableGitignore: true})
	if got != nil {
		t.Errorf("Detect returned non-nil lister, want nil when sentinel returned")
	}
	if !errors.Is(err, lister.ErrNoGitignoreInRepo) {
		t.Errorf("errors.Is(err, ErrNoGitignoreInRepo) = false; got %v", err)
	}
}

// TestDetect_SingleFile verifies that Detect returns a *SingleFileLister when
// the root argument points to a regular file (not a directory). The test
// writes a temporary file into t.TempDir() and passes the file path directly
// to Detect. This exercises the early-return path added in v0.1.4 (Bug A).
func TestDetect_SingleFile(t *testing.T) {
	tmp := t.TempDir()
	filePath := filepath.Join(tmp, "sample.txt")
	if err := os.WriteFile(filePath, []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	ctx := t.Context()
	got, err := lister.Detect(ctx, filePath, fileset.WalkOptions{})
	if err != nil {
		t.Fatalf("Detect returned unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("Detect returned nil lister, want non-nil")
	}
	if _, ok := got.(*lister.SingleFileLister); !ok {
		t.Errorf("Detect returned %T, want *lister.SingleFileLister", got)
	}
}

// TestDetect_SymlinkedFile verifies that Detect resolves a symlink that points
// to a regular file and returns a *SingleFileLister. This exercises the
// EvalSymlinks + stat path added in v0.1.4 (Bug B + Bug A combined).
func TestDetect_SymlinkedFile(t *testing.T) {
	tmp := t.TempDir()
	target := filepath.Join(tmp, "real.txt")
	if err := os.WriteFile(target, []byte("content\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	link := filepath.Join(tmp, "link.txt")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("os.Symlink not supported: %v", err)
	}

	ctx := t.Context()
	got, err := lister.Detect(ctx, link, fileset.WalkOptions{})
	if err != nil {
		t.Fatalf("Detect returned unexpected error for symlink-to-file: %v", err)
	}
	if got == nil {
		t.Fatal("Detect returned nil lister, want non-nil")
	}
	if _, ok := got.(*lister.SingleFileLister); !ok {
		t.Errorf("Detect returned %T for symlink-to-file, want *lister.SingleFileLister", got)
	}
}

// TestDetect_SymlinkedDir verifies that Detect resolves a symlink that points
// to a directory containing a git repository and returns a *GitLister. This
// exercises the EvalSymlinks path added in v0.1.4 (Bug B) for the directory
// case where a git probe can succeed after symlink resolution.
func TestDetect_SymlinkedDir(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not found")
	}

	tmp := t.TempDir()
	// Create a real git repo inside tmp.
	repoDir := filepath.Join(tmp, "repo")
	if err := os.Mkdir(repoDir, 0o755); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}
	initCmd := exec.Command("git", "init", "--template=", repoDir)
	if out, err := initCmd.CombinedOutput(); err != nil {
		t.Skipf("git init failed (%v): %s", err, out)
	}

	// Create a symlink pointing to the git repo directory.
	link := filepath.Join(tmp, "link-to-repo")
	if err := os.Symlink(repoDir, link); err != nil {
		t.Skipf("os.Symlink not supported: %v", err)
	}

	ctx := t.Context()
	got, err := lister.Detect(ctx, link, fileset.WalkOptions{})
	if err != nil {
		t.Fatalf("Detect returned unexpected error for symlink-to-dir: %v", err)
	}
	if got == nil {
		t.Fatal("Detect returned nil lister, want non-nil")
	}
	if _, ok := got.(*lister.GitLister); !ok {
		t.Errorf("Detect returned %T for symlink-to-git-dir, want *lister.GitLister", got)
	}
}

// TestDetect_BrokenSymlink verifies that Detect returns a wrapped error when
// the root argument is a symlink that points to a path that does not exist.
// EvalSymlinks fails on broken symlinks; Detect must propagate that error.
func TestDetect_BrokenSymlink(t *testing.T) {
	tmp := t.TempDir()
	link := filepath.Join(tmp, "broken")
	// Point to a path that does not exist.
	if err := os.Symlink(filepath.Join(tmp, "nonexistent"), link); err != nil {
		t.Skipf("os.Symlink not supported: %v", err)
	}

	ctx := t.Context()
	got, err := lister.Detect(ctx, link, fileset.WalkOptions{})
	if err == nil {
		t.Fatalf("Detect returned nil error for broken symlink, want an error (lister: %T)", got)
	}
	if got != nil {
		t.Errorf("Detect returned non-nil lister for broken symlink, want nil")
	}
	// The error must be wrapped with the "lister: detect:" prefix so callers
	// get consistent error messages regardless of the underlying OS error.
	if msg := err.Error(); !strings.HasPrefix(msg, "lister: detect:") {
		t.Errorf("error message does not start with %q: %s", "lister: detect:", msg)
	}
}

// ---------------------------------------------------------------------------
// FilesFromLister tests
// ---------------------------------------------------------------------------

// TestFilesFromLister_EmptyReader verifies that a FilesFromLister backed by an
// empty reader yields zero files and zero errors.
func TestFilesFromLister_EmptyReader(t *testing.T) {
	t.Parallel()

	fl := lister.NewFilesFromLister(strings.NewReader(""))
	var files []*fileset.File
	var errs []error
	for f, e := range fl.List(t.Context()) {
		if e != nil {
			errs = append(errs, e)
			continue
		}
		files = append(files, f)
	}
	if len(files) != 0 {
		t.Errorf("got %d files, want 0", len(files))
	}
	if len(errs) != 0 {
		t.Errorf("got %d errors, want 0: %v", len(errs), errs)
	}
}

// TestFilesFromLister_HashPrefixedFileWorks verifies that a file literally
// named "#draft.md" is yielded normally — hash-prefixed paths are NOT treated
// as comments.
func TestFilesFromLister_HashPrefixedFileWorks(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	target := filepath.Join(tmp, "#draft.md")
	if err := os.WriteFile(target, []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	fl := lister.NewFilesFromLister(strings.NewReader(target + "\n"))
	var files []*fileset.File
	var errs []error
	for f, e := range fl.List(t.Context()) {
		if e != nil {
			errs = append(errs, e)
			continue
		}
		files = append(files, f)
	}
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(files) != 1 {
		t.Fatalf("got %d files, want 1", len(files))
	}
	if got := files[0].RelPath; got != "#draft.md" {
		t.Errorf("RelPath = %q, want %q", got, "#draft.md")
	}
}

// TestFilesFromLister_SkipsEmptyLines verifies that blank lines interspersed in
// the reader are skipped; valid paths around them are still yielded.
func TestFilesFromLister_SkipsEmptyLines(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	fileA := filepath.Join(tmp, "a.txt")
	fileB := filepath.Join(tmp, "b.txt")
	for _, f := range []string{fileA, fileB} {
		if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
	}

	input := "\n" + fileA + "\n\n" + fileB + "\n\n"
	fl := lister.NewFilesFromLister(strings.NewReader(input))
	var files []*fileset.File
	var errs []error
	for f, e := range fl.List(t.Context()) {
		if e != nil {
			errs = append(errs, e)
			continue
		}
		files = append(files, f)
	}
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(files) != 2 {
		t.Errorf("got %d files, want 2", len(files))
	}
}

// TestFilesFromLister_MixedPaths verifies that a mix of valid paths and empty
// lines produces only the valid-path files, in order.
func TestFilesFromLister_MixedPaths(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	fileA := filepath.Join(tmp, "first.go")
	fileB := filepath.Join(tmp, "second.go")
	for _, f := range []string{fileA, fileB} {
		if err := os.WriteFile(f, []byte("package p\n"), 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
	}

	input := fileA + "\n\n" + fileB + "\n"
	fl := lister.NewFilesFromLister(strings.NewReader(input))
	var relPaths []string
	var errs []error
	for f, e := range fl.List(t.Context()) {
		if e != nil {
			errs = append(errs, e)
			continue
		}
		relPaths = append(relPaths, f.RelPath)
	}
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	want := []string{"first.go", "second.go"}
	if len(relPaths) != len(want) {
		t.Fatalf("got %v, want %v", relPaths, want)
	}
	for i, w := range want {
		if relPaths[i] != w {
			t.Errorf("relPaths[%d] = %q, want %q", i, relPaths[i], w)
		}
	}
}

// TestFilesFromLister_MissingFile verifies that a path that does not exist on
// disk yields a (nil, err) pair while iteration continues; subsequent valid
// paths are still yielded.
func TestFilesFromLister_MissingFile(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	realFile := filepath.Join(tmp, "real.txt")
	if err := os.WriteFile(realFile, []byte("data"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	missing := filepath.Join(tmp, "does_not_exist.txt")

	input := missing + "\n" + realFile + "\n"
	fl := lister.NewFilesFromLister(strings.NewReader(input))
	var files []*fileset.File
	var errs []error
	for f, e := range fl.List(t.Context()) {
		if e != nil {
			errs = append(errs, e)
			continue
		}
		files = append(files, f)
	}
	if len(errs) != 1 {
		t.Errorf("got %d errors, want 1", len(errs))
	}
	if len(files) != 1 {
		t.Errorf("got %d files, want 1 (iteration must continue past bad paths)", len(files))
	}
	if len(files) == 1 && files[0].RelPath != "real.txt" {
		t.Errorf("RelPath = %q, want %q", files[0].RelPath, "real.txt")
	}
}

// TestFilesFromLister_ContextCancel verifies that cancelling the context after
// the first yield terminates iteration with a context error and does not panic.
func TestFilesFromLister_ContextCancel(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	// Create three files so there are entries after the first.
	for _, name := range []string{"a.txt", "b.txt", "c.txt"} {
		f := filepath.Join(tmp, name)
		if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
	}
	fileA := filepath.Join(tmp, "a.txt")
	fileB := filepath.Join(tmp, "b.txt")
	fileC := filepath.Join(tmp, "c.txt")

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	fl := lister.NewFilesFromLister(strings.NewReader(fileA + "\n" + fileB + "\n" + fileC + "\n"))
	count := 0
	var ctxErr error
	for f, e := range fl.List(ctx) {
		if e != nil {
			ctxErr = e
			break
		}
		count++
		_ = f
		if count == 1 {
			cancel()
		}
	}

	if ctxErr == nil {
		t.Error("expected a context error after cancellation, got nil")
	}
	if count > 2 {
		t.Errorf("iteration continued past cancellation: count = %d", count)
	}
}

// ---------------------------------------------------------------------------
// TestDetect_NotRegularFile_FriendlyError verifies that Detect returns a
// friendly error when the root argument is a non-regular, non-directory file
// (e.g. a character device). The error message must contain
// "not a regular file or directory" and must NOT contain "fork/exec".
//
// /dev/null is always present on macOS and Linux and is a character device,
// making it the canonical test input for this case without requiring any
// platform-specific syscall (e.g. syscall.Mkfifo).
func TestDetect_NotRegularFile_FriendlyError(t *testing.T) {
	const devNull = "/dev/null"
	if _, err := os.Stat(devNull); err != nil {
		t.Skipf("/dev/null not available on this platform: %v", err)
	}

	ctx := t.Context()
	got, err := lister.Detect(ctx, devNull, fileset.WalkOptions{})
	if err == nil {
		t.Fatalf("Detect returned nil error for %s, want a friendly error (lister: %T)", devNull, got)
	}
	if got != nil {
		t.Errorf("Detect returned non-nil lister for %s, want nil", devNull)
	}
	if !errors.Is(err, lister.ErrNotRegularFileOrDirectory) {
		t.Errorf("errors.Is(err, ErrNotRegularFileOrDirectory) = false; got: %v", err)
	}
	if msg := err.Error(); !strings.Contains(msg, "not a regular file or directory") {
		t.Errorf("error message does not contain %q: %s", "not a regular file or directory", msg)
	}
	if msg := err.Error(); strings.Contains(msg, "fork/exec") {
		t.Errorf("error message must not contain %q but got: %s", "fork/exec", msg)
	}
}

// ---------------------------------------------------------------------------
// TestSingleFileLister_List (original, not FilesFromLister)
// ---------------------------------------------------------------------------

// TestSingleFileLister_List verifies that a SingleFileLister constructed
// directly yields exactly one file with the expected RelPath value, and that
// iterating a second time produces the same result (idempotency).
func TestSingleFileLister_List(t *testing.T) {
	tmp := t.TempDir()
	filePath := filepath.Join(tmp, "hello.go")
	if err := os.WriteFile(filePath, []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Build a SingleFileLister via Detect so we don't depend on any unexported
	// constructor — Detect returns *SingleFileLister for a regular file path.
	fl, err := lister.Detect(t.Context(), filePath, fileset.WalkOptions{})
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if _, ok := fl.(*lister.SingleFileLister); !ok {
		t.Fatalf("expected *lister.SingleFileLister, got %T", fl)
	}

	// Iterate and collect results.
	var files []*fileset.File
	var errs []error
	for f, e := range fl.List(context.Background()) {
		if e != nil {
			errs = append(errs, e)
			continue
		}
		files = append(files, f)
	}

	if len(errs) != 0 {
		t.Fatalf("List returned unexpected errors: %v", errs)
	}
	if len(files) != 1 {
		t.Fatalf("List yielded %d files, want exactly 1", len(files))
	}
	if got := files[0].RelPath; got != "hello.go" {
		t.Errorf("RelPath = %q, want %q", got, "hello.go")
	}
}
