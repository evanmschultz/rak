package main

import (
	"bytes"
	"strings"
	"testing"
)

// TestRootCmd_ReadsStdin_RendersHumanDefault verifies the default (no
// --format flag) path: stdin is read via cmd.InOrStdin() (NOT os.Stdin —
// F9 pin), counting runs, and the human renderer emits a laslig KV block
// containing the four canonical labels. Stdout is a bytes.Buffer (non-TTY),
// so laslig auto-selects plain non-styled output — we assert on labels, not
// exact bytes, to keep the test robust against laslig layout tweaks.
func TestRootCmd_ReadsStdin_RendersHumanDefault(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	cmd := newRootCmd()
	cmd.SetIn(strings.NewReader("hello world\n"))
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("cmd.Execute: %v", err)
	}

	got := out.String()
	for _, label := range []string{"Bytes", "Lines", "Words", "Chars"} {
		if !strings.Contains(got, label) {
			t.Errorf("output missing label %q; got:\n%s", label, got)
		}
	}
}

// TestRootCmd_FormatJSON verifies --format=json picks NewJSONRenderer. The
// JSON renderer uses stdlib encoding/json with no struct tags (F4 pin), so
// the emitted keys match the Counts struct declaration order. We assert key
// presence rather than exact bytes so the test does not couple to
// json.Encoder's trailing-newline convention beyond what the render package
// already snapshots.
func TestRootCmd_FormatJSON(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	cmd := newRootCmd()
	cmd.SetIn(strings.NewReader("hello world\n"))
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--format=json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("cmd.Execute: %v", err)
	}

	got := out.String()
	for _, key := range []string{`"Bytes"`, `"Lines"`, `"Words"`, `"Chars"`} {
		if !strings.Contains(got, key) {
			t.Errorf("json output missing key %s; got: %s", key, got)
		}
	}
}

// TestRootCmd_InvalidFormat verifies an unknown --format value returns a
// non-nil error whose message mentions "format", so future CLI users get a
// useful hint. cobra returns the error from Execute when RunE returns one.
func TestRootCmd_InvalidFormat(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	cmd := newRootCmd()
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--format=xml"})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected error for --format=xml, got nil")
	}
	if !strings.Contains(err.Error(), "format") {
		t.Errorf("error should mention %q; got: %v", "format", err)
	}
}

// TestRootCmd_RejectsPathArg verifies Decision A1: a single positional arg
// returns a hard error pointing to Drop 3, rather than silently falling back
// to stdin. The error message must mention "Drop 3" so the user knows when
// the feature is expected to land.
func TestRootCmd_RejectsPathArg(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	cmd := newRootCmd()
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"./somepath"})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected error for positional path arg, got nil")
	}
	if !strings.Contains(err.Error(), "Drop 3") {
		t.Errorf("error should mention %q; got: %v", "Drop 3", err)
	}
}
