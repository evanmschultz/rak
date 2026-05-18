package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Integration tests drive newRootCmd end-to-end against the cmd/rak/testdata/
// fixture tree per CLAUDE.md § "Tests" → "Two-tier testdata rule". The fixture
// content is chosen to exercise the F12 fixture-coverage invariant from
// Unit 2.4's plan: multi-line, multi-word, and at least one multi-byte UTF-8
// rune, with expected Bytes > Chars. "hello world\nrak café naïve\n" gives:
//
//   - Bytes = 29  (ASCII bytes for "hello world\nrak caf" [19] + "é"=2 + " na"=3
//     + "ï"=2 + "ve\n"=3 = 29; confirmed via `wc -c`)
//   - Lines = 2   (two '\n' runes)
//   - Words = 5   ("hello", "world", "rak", "café", "naïve")
//   - Chars = 27  (rune count: 29 bytes - 2 for é - 2 for ï + the two runes
//     themselves = 29 - 4 + 2 = 27)
//
// Bytes > Chars holds (29 > 27) because é and ï are each 2-byte / 1-rune. If
// the fixture content changes, update these expectations and the snapshot
// strings below.

const (
	integrationExpectedBytes = 29
	integrationExpectedLines = 2
	integrationExpectedWords = 5
	integrationExpectedChars = 27
)

// TestRootCmd_Integration_HumanFormat drives the full CLI from fixture file
// through stdin piping to human-renderer output. Assertions are
// tolerance-based (strings.Contains) rather than byte-exact because
// NewHumanRenderer's production laslig.Policy resolves mode against the
// writer at Render time. Even though the bytes.Buffer target is a non-TTY (so
// laslig resolves to plain + unstyled), pinning exact bytes here would
// double-couple cmd/rak tests to laslig's internal formatter output — the
// exact-snapshot coverage already lives in internal/render/render_test.go via
// newHumanRendererWithMode. This test's job is to prove the wiring composes:
// fixture → stdin → counting → renderer → stdout.
func TestRootCmd_Integration_HumanFormat(t *testing.T) {
	t.Parallel()

	file, err := os.Open(filepath.Join("testdata", "hello.txt"))
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	t.Cleanup(func() { _ = file.Close() })

	var out bytes.Buffer
	cmd := newRootCmd()
	cmd.SetIn(file)
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--human"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("cmd.Execute: %v", err)
	}

	got := out.String()
	for _, want := range []string{
		"Bytes", "Lines", "Words", "Chars",
		"29", // Bytes
		"2",  // Lines (also substring of 29 — but labelled Bytes too; the check is loose by design)
		"5",  // Words
		"27", // Chars
	} {
		if !strings.Contains(got, want) {
			t.Errorf("human output missing %q; got:\n%s", want, got)
		}
	}
}

// TestRootCmd_Integration_JSONFormat drives the full CLI from fixture file
// through stdin piping to JSON output. Unlike the human path, the JSON
// encoder's output is deterministic across environments — stdlib
// encoding/json.NewEncoder with the counting.Counts declaration-order field
// layout (F4 pin: no json struct tags) emits a single stable line. We assert
// byte-exact here because the entire chain is deterministic: no laslig, no
// TTY resolution, no locale-sensitive formatting.
func TestRootCmd_Integration_JSONFormat(t *testing.T) {
	t.Parallel()

	file, err := os.Open(filepath.Join("testdata", "hello.txt"))
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	t.Cleanup(func() { _ = file.Close() })

	var out bytes.Buffer
	cmd := newRootCmd()
	cmd.SetIn(file)
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("cmd.Execute: %v", err)
	}

	got := out.String()
	want := `{"Bytes":29,"Lines":2,"Words":5,"Chars":27}` + "\n"
	if got != want {
		t.Fatalf("json output mismatch\nwant: %q\ngot:  %q", want, got)
	}

	// Belt-and-suspenders: also verify the constants match the literal to
	// catch drift between the fixture content and the documented expectations
	// at the top of this file.
	if integrationExpectedBytes != 29 ||
		integrationExpectedLines != 2 ||
		integrationExpectedWords != 5 ||
		integrationExpectedChars != 27 {
		t.Fatalf("fixture expectation constants drifted from snapshot literal")
	}
}

// Directory-walk integration fixture constants. The tree under
// testdata/tree/ (created for Unit 3.5) has:
//
//   - a.txt         = "hello world\n"            (12 bytes, 1 line, 2 words, 12 chars)
//   - sub/nested.txt = "one two\n"               (8 bytes, 1 line, 2 words, 8 chars)
//   - .gitignore     = "vendor/\n"               (not walked; gitignore content itself not counted by default — .gitignore lives at root, and hidden filter does not drop it since the name doesn't start with "." — but this file IS counted!)
//   - .hidden.txt    = excluded by default (hidden)
//   - bin.dat        = excluded by default (NUL byte → binary)
//   - vendor/ignored.txt = excluded (gitignore)
//
// NOTE: .gitignore's name does NOT start with a dot when IsHidden is
// considered (wait — it does: ".gitignore" starts with ".", so IsHidden
// returns true and it is excluded by default). Both hidden files are
// skipped under the defaults.
//
// Default-walk totals (no flags): a.txt + sub/nested.txt
//
//   - Bytes: 12 + 8 = 20
//   - Lines: 1 + 1 = 2
//   - Words: 2 + 2 = 4
//   - Chars: 12 + 8 = 20
const (
	treeExpectedTotalBytes = 20
	treeExpectedTotalLines = 2
	treeExpectedTotalWords = 4
	treeExpectedTotalChars = 20
)

// TestRootCmd_Integration_PathArg_HumanFormat drives rak with a positional
// directory argument end-to-end against testdata/tree/. Assertions are
// tolerance-based for the same reason as the stdin human test: laslig's
// exact layout is pinned in render_test.go, not here.
func TestRootCmd_Integration_PathArg_HumanFormat(t *testing.T) {
	t.Parallel()

	root := filepath.Join("testdata", "tree")
	var out bytes.Buffer
	cmd := newRootCmd()
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--human", root})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("cmd.Execute: %v", err)
	}

	got := out.String()
	for _, want := range []string{
		"Bytes", "Lines", "Words", "Chars",
		"total",
		"dir: ",
		// Root directory label is "dir: <root>" where root is the CLI
		// positional arg. The path includes testdata/tree and is
		// platform-separator-aware (filepath.Join), so we assert the
		// suffix rather than the whole string.
		"testdata",
		"tree",
		"20", // Total bytes = 20 AND per-dir root bytes = 12 (substring-match only checks "20")
	} {
		if !strings.Contains(got, want) {
			t.Errorf("human path-arg output missing %q; got:\n%s", want, got)
		}
	}
}

// TestRootCmd_Integration_PathArg_JSONFormat drives rak with a positional
// directory argument end-to-end against testdata/tree/. JSON output is
// deterministic so we parse and structurally assert on the envelope.
func TestRootCmd_Integration_PathArg_JSONFormat(t *testing.T) {
	t.Parallel()

	root := filepath.Join("testdata", "tree")
	var out bytes.Buffer
	cmd := newRootCmd()
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--json", root})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("cmd.Execute: %v", err)
	}

	var parsed treeResult
	if err := json.Unmarshal(out.Bytes(), &parsed); err != nil {
		t.Fatalf("json.Unmarshal(%s): %v", out.String(), err)
	}

	if parsed.Total.Bytes != treeExpectedTotalBytes ||
		parsed.Total.Lines != treeExpectedTotalLines ||
		parsed.Total.Words != treeExpectedTotalWords ||
		parsed.Total.Chars != treeExpectedTotalChars {
		t.Errorf("total mismatch: want B=%d L=%d W=%d C=%d, got %+v",
			treeExpectedTotalBytes, treeExpectedTotalLines,
			treeExpectedTotalWords, treeExpectedTotalChars, parsed.Total)
	}

	// Expect exactly two directory buckets: the root and sub. vendor/ is
	// gitignored, bin.dat and .hidden.txt are filtered out of the root.
	if len(parsed.Directories) != 2 {
		t.Fatalf("expected 2 directories (root + sub), got %d: %+v", len(parsed.Directories), parsed.Directories)
	}

	// Find root + sub entries — order is lexical on walk-relative path,
	// which after labelDirectories rewrites becomes "testdata/tree" <
	// "testdata/tree/sub" (on Unix; the test uses filepath.Join which
	// produces forward-slash paths on Unix — on Windows this would be
	// "testdata\tree" and the comparison below would need adjustment,
	// but rak's target platforms are Unix-like).
	var rootDir, subDir *dirResult
	for i := range parsed.Directories {
		d := &parsed.Directories[i]
		switch {
		case d.Path == root:
			rootDir = d
		case strings.HasSuffix(d.Path, "sub"):
			subDir = d
		}
	}
	if rootDir == nil || subDir == nil {
		t.Fatalf("missing expected dirs: rootDir=%v subDir=%v; all=%+v", rootDir, subDir, parsed.Directories)
	}
	if rootDir.Counts.Bytes != 12 { // a.txt
		t.Errorf("root counts mismatch: want Bytes=12, got %+v", rootDir.Counts)
	}
	if subDir.Counts.Bytes != 8 { // sub/nested.txt
		t.Errorf("sub counts mismatch: want Bytes=8, got %+v", subDir.Counts)
	}
	if len(parsed.Errors) != 0 {
		t.Errorf("expected no errors walking the fixture tree; got %v", parsed.Errors)
	}
}

// TestRootCmd_Integration_FilesFrom_StdinList verifies that --files-from -
// reads a newline-separated list of paths from stdin and counts the files it
// names. The fixture list contains both files from testdata/tree/ — a.txt (12
// bytes) and sub/nested.txt (8 bytes) — so totals must match the tree fixture
// constants (Bytes=20, Lines=2, Words=4, Chars=20). Paths are fed as
// relative-to-CWD strings; FilesFromLister resolves them via os.Getwd() at
// list time, which returns cmd/rak/ during the test run.
func TestRootCmd_Integration_FilesFrom_StdinList(t *testing.T) {
	t.Parallel()

	list := "testdata/tree/a.txt\ntestdata/tree/sub/nested.txt\n"
	var out bytes.Buffer
	cmd := newRootCmd()
	cmd.SetIn(strings.NewReader(list))
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--json", "--files-from", "-"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("cmd.Execute: %v", err)
	}

	var parsed treeResult
	if err := json.Unmarshal(out.Bytes(), &parsed); err != nil {
		t.Fatalf("json.Unmarshal(%s): %v", out.String(), err)
	}

	if parsed.Total.Bytes != treeExpectedTotalBytes ||
		parsed.Total.Lines != treeExpectedTotalLines ||
		parsed.Total.Words != treeExpectedTotalWords ||
		parsed.Total.Chars != treeExpectedTotalChars {
		t.Errorf("total mismatch: want B=%d L=%d W=%d C=%d, got %+v",
			treeExpectedTotalBytes, treeExpectedTotalLines,
			treeExpectedTotalWords, treeExpectedTotalChars, parsed.Total)
	}
}

// TestRootCmd_Integration_FilesFrom_EmptyStdin verifies that --files-from -
// with an empty stdin (equivalent to `echo -n | rak --files-from -`) produces
// valid JSON with zero totals and does not panic or error. The rendered output
// must parse cleanly and Total.Bytes must be zero.
func TestRootCmd_Integration_FilesFrom_EmptyStdin(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	cmd := newRootCmd()
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--json", "--files-from", "-"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("cmd.Execute: %v", err)
	}

	var parsed treeResult
	if err := json.Unmarshal(out.Bytes(), &parsed); err != nil {
		t.Fatalf("json.Unmarshal(%s): %v", out.String(), err)
	}

	if parsed.Total.Bytes != 0 {
		t.Errorf("empty stdin: expected Total.Bytes=0, got %+v", parsed.Total)
	}
}

// TestRootCmd_Integration_FilesFrom_SkipsEmptyLines verifies that blank lines
// in the --files-from input are silently skipped and do not affect the count.
// The input has blank lines before, between, and after the two fixture paths.
// Totals must match the tree fixture constants (Bytes=20, Lines=2, Words=4,
// Chars=20) — identical to TestRootCmd_Integration_FilesFrom_StdinList.
func TestRootCmd_Integration_FilesFrom_SkipsEmptyLines(t *testing.T) {
	t.Parallel()

	list := "\ntestdata/tree/a.txt\n\ntestdata/tree/sub/nested.txt\n\n"
	var out bytes.Buffer
	cmd := newRootCmd()
	cmd.SetIn(strings.NewReader(list))
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--json", "--files-from", "-"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("cmd.Execute: %v", err)
	}

	var parsed treeResult
	if err := json.Unmarshal(out.Bytes(), &parsed); err != nil {
		t.Fatalf("json.Unmarshal(%s): %v", out.String(), err)
	}

	if parsed.Total.Bytes != treeExpectedTotalBytes ||
		parsed.Total.Lines != treeExpectedTotalLines ||
		parsed.Total.Words != treeExpectedTotalWords ||
		parsed.Total.Chars != treeExpectedTotalChars {
		t.Errorf("total mismatch (empty lines should be skipped): want B=%d L=%d W=%d C=%d, got %+v",
			treeExpectedTotalBytes, treeExpectedTotalLines,
			treeExpectedTotalWords, treeExpectedTotalChars, parsed.Total)
	}
}

// TestRootCmd_Integration_FilesFrom_HashFileWorks verifies that a file whose
// name starts with '#' is counted normally and not silently dropped as a
// comment. FilesFromLister explicitly does not implement comment syntax (per
// PLAN.md D.1 design). The file is created in t.TempDir() with a small content
// payload; its absolute path is fed through --files-from stdin. The total byte
// count must equal the length of the written content.
func TestRootCmd_Integration_FilesFrom_HashFileWorks(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	hashFile := filepath.Join(tmp, "#draft.md")
	content := []byte("# draft\n")
	if err := os.WriteFile(hashFile, content, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Feed the absolute path through stdin so CWD resolution is irrelevant.
	list := hashFile + "\n"
	var out bytes.Buffer
	cmd := newRootCmd()
	cmd.SetIn(strings.NewReader(list))
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--json", "--files-from", "-"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("cmd.Execute: %v", err)
	}

	var parsed treeResult
	if err := json.Unmarshal(out.Bytes(), &parsed); err != nil {
		t.Fatalf("json.Unmarshal(%s): %v", out.String(), err)
	}

	// The content "# draft\n" is 8 bytes; a non-zero count proves the file
	// was not silently dropped due to the '#' prefix.
	wantBytes := int64(len(content))
	if parsed.Total.Bytes != wantBytes {
		t.Errorf("hash-prefixed file: expected Total.Bytes=%d, got %+v", wantBytes, parsed.Total)
	}
}

// TestRootCmd_Integration_FilesFrom_PositionalArgConflict verifies that
// combining --files-from with a positional path argument produces a hard error
// (Guard A in PersistentPreRunE). The error message must contain "cannot
// combine".
func TestRootCmd_Integration_FilesFrom_PositionalArgConflict(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	cmd := newRootCmd()
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--files-from", "-", "."})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected error combining --files-from with positional arg, got nil")
	}
	if !strings.Contains(err.Error(), "cannot combine") {
		t.Errorf("error must contain %q; got: %v", "cannot combine", err)
	}
}

// TestFlags_FilesFromNoGitignoreHardErrors verifies that combining --files-from
// with --no-gitignore produces a hard error (Guard B in PersistentPreRunE).
// The error message must reference "--no-gitignore".
func TestFlags_FilesFromNoGitignoreHardErrors(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	cmd := newRootCmd()
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--files-from", "-", "--no-gitignore"})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected error combining --files-from with --no-gitignore, got nil")
	}
	if !strings.Contains(err.Error(), "--no-gitignore") {
		t.Errorf("error must contain %q; got: %v", "--no-gitignore", err)
	}
}

// ---------------------------------------------------------------------------
// Lockfile filter integration tests
// ---------------------------------------------------------------------------

// TestLockfileFilter_ExcludedByDefault verifies that rak skips lockfiles
// (go.sum, package-lock.json, etc.) when --include-lockfiles is NOT passed.
// Uses --files-from so the test is hermetic and does not depend on git
// enumerating from within the rak checkout.
func TestLockfileFilter_ExcludedByDefault(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()

	// Regular source file — should be counted.
	src := filepath.Join(tmp, "main.go")
	srcContent := []byte("package main\n")
	if err := os.WriteFile(src, srcContent, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Lockfile — should be excluded by default.
	lock := filepath.Join(tmp, "go.sum")
	lockContent := []byte("github.com/example/pkg v1.0.0 h1:abc\n")
	if err := os.WriteFile(lock, lockContent, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Feed both paths via --files-from.
	list := src + "\n" + lock + "\n"
	var out bytes.Buffer
	cmd := newRootCmd()
	cmd.SetIn(strings.NewReader(list))
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--json", "--files-from", "-"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("cmd.Execute: %v", err)
	}

	var parsed treeResult
	if err := json.Unmarshal(out.Bytes(), &parsed); err != nil {
		t.Fatalf("json.Unmarshal(%s): %v", out.String(), err)
	}

	// Only main.go should be counted — go.sum must be excluded.
	wantBytes := int64(len(srcContent))
	if parsed.Total.Bytes != wantBytes {
		t.Errorf("lockfile excluded by default: Total.Bytes = %d, want %d (only main.go counted)", parsed.Total.Bytes, wantBytes)
	}
}

// TestLockfileFilter_IncludeWhenFlagSet verifies that rak counts lockfiles
// when --include-lockfiles is passed. Uses the same hermetic --files-from
// approach as the exclusion test; both files must appear in the total.
func TestLockfileFilter_IncludeWhenFlagSet(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()

	// Regular source file.
	src := filepath.Join(tmp, "main.go")
	srcContent := []byte("package main\n")
	if err := os.WriteFile(src, srcContent, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Lockfile.
	lock := filepath.Join(tmp, "go.sum")
	lockContent := []byte("github.com/example/pkg v1.0.0 h1:abc\n")
	if err := os.WriteFile(lock, lockContent, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Feed both paths via --files-from WITH --include-lockfiles.
	list := src + "\n" + lock + "\n"
	var out bytes.Buffer
	cmd := newRootCmd()
	cmd.SetIn(strings.NewReader(list))
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--json", "--files-from", "-", "--include-lockfiles"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("cmd.Execute: %v", err)
	}

	var parsed treeResult
	if err := json.Unmarshal(out.Bytes(), &parsed); err != nil {
		t.Fatalf("json.Unmarshal(%s): %v", out.String(), err)
	}

	// Both files must be counted.
	wantBytes := int64(len(srcContent) + len(lockContent))
	if parsed.Total.Bytes != wantBytes {
		t.Errorf("lockfile included with flag: Total.Bytes = %d, want %d (both files counted)", parsed.Total.Bytes, wantBytes)
	}
}

// TestFilesFrom_MaxFiles verifies that --max-files fires correctly in
// --files-from mode: when the number of accepted files reaches the limit,
// cmd.Execute returns a non-nil error wrapping ErrMaxFilesExceeded. Three
// real files are created in t.TempDir(); their absolute paths are fed via
// stdin with --max-files 1 so the limit is hit after the first file.
func TestFilesFrom_MaxFiles(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()

	// Create three real files so FilesFromLister can os.Stat them.
	var paths []string
	for i := range 3 {
		p := filepath.Join(tmp, fmt.Sprintf("file%d.txt", i))
		if err := os.WriteFile(p, []byte("content\n"), 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
		paths = append(paths, p)
	}

	// Build stdin list with all three absolute paths.
	list := strings.Join(paths, "\n") + "\n"

	var out bytes.Buffer
	cmd := newRootCmd()
	cmd.SetIn(strings.NewReader(list))
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--json", "--files-from", "-", "--max-files", "1"})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("--max-files 1 with 3 files: expected error wrapping ErrMaxFilesExceeded, got nil")
	}
	if !errors.Is(err, ErrMaxFilesExceeded) {
		t.Errorf("expected errors.Is(err, ErrMaxFilesExceeded) true; got: %v", err)
	}
}
