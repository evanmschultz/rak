package main

import (
	"bytes"
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
	cmd.SetArgs([]string{"--format=human"})

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
	cmd.SetArgs([]string{"--format=json"})

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
