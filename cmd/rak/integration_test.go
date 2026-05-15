package main

import (
	"bytes"
	"encoding/json"
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
