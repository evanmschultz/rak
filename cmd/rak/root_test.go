package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/evanmschultz/rak/internal/counting"
	"github.com/evanmschultz/rak/internal/render"
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

// runTreeFS is a lightweight test helper that runs the per-dir aggregation
// loop against an injected fs.FS without going through cobra. Tests use it
// to:
//   - exercise the walkAndCount + runDirectory chain with a known-shape
//     stub fs.FS (for induced error paths)
//   - return JSON-formatted output so assertions can parse structured
//     data rather than string-match laslig output
//
// The emitted JSON envelope is the same shape produced by
// jsonRenderer.RenderTree (Unit 3.5 F15): `{"directories":[...],"total":{...},"errors"?:[...]}`.
func runTreeFS(t *testing.T, fsys fs.FS, flags *rootFlags) (treeResult, []byte) {
	t.Helper()

	var out bytes.Buffer
	renderer, err := selectRenderer("json")
	if err != nil {
		t.Fatalf("selectRenderer: %v", err)
	}
	// rootLabel = "" keeps the walker's io/fs "." convention so assertions
	// compare against the raw relative paths.
	if err := runDirectory(context.Background(), &out, "", fsys, flags, renderer); err != nil {
		t.Fatalf("runDirectory: %v", err)
	}
	raw := out.Bytes()
	var decoded treeResult
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(%s): %v", raw, err)
	}
	return decoded, raw
}

// treeResult mirrors the RenderTree JSON envelope for structured
// assertions. Only fields tests care about are exported; keeping a local
// copy of the shape means a renderer-internal refactor of directoryJSON
// does not ripple into every test.
type treeResult struct {
	Directories []dirResult     `json:"directories"`
	Total       counting.Counts `json:"total"`
	Errors      []string        `json:"errors,omitempty"`
}

type dirResult struct {
	Path   string          `json:"path"`
	Counts counting.Counts `json:"counts"`
}

func (r treeResult) paths() []string {
	out := make([]string, 0, len(r.Directories))
	for _, d := range r.Directories {
		out = append(out, d.Path)
	}
	return out
}

// TestRootCmd_PathArg_EmptyDir: an empty directory produces zero totals and
// renders cleanly with no directories emitted.
func TestRootCmd_PathArg_EmptyDir(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{} // truly empty — no entries at all
	res, _ := runTreeFS(t, fsys, &rootFlags{})

	if len(res.Directories) != 0 {
		t.Errorf("expected zero directories, got %d: %v", len(res.Directories), res.paths())
	}
	if res.Total != (counting.Counts{}) {
		t.Errorf("expected zero total, got %+v", res.Total)
	}
	if len(res.Errors) != 0 {
		t.Errorf("expected zero errors, got %d: %v", len(res.Errors), res.Errors)
	}
}

// TestRootCmd_PathArg_FlatDir: a directory with two text files produces a
// single "." directory rollup whose counts equal the sum of per-file
// counts, and a grand total matching the directory rollup.
func TestRootCmd_PathArg_FlatDir(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		"a.txt": {Data: []byte("hello\n")},
		"b.txt": {Data: []byte("world\n")},
	}
	res, _ := runTreeFS(t, fsys, &rootFlags{})

	if len(res.Directories) != 1 {
		t.Fatalf("expected 1 directory, got %d: %v", len(res.Directories), res.paths())
	}
	if res.Directories[0].Path != "." {
		t.Errorf("expected root directory '.', got %q", res.Directories[0].Path)
	}
	// "hello\n" = 6 bytes 1 line 1 word 6 chars; "world\n" same.
	// Totals: 12 bytes, 2 lines, 2 words, 12 chars.
	want := counting.Counts{Bytes: 12, Lines: 2, Words: 2, Chars: 12}
	if res.Total != want {
		t.Errorf("total mismatch: want %+v, got %+v", want, res.Total)
	}
	if res.Directories[0].Counts != want {
		t.Errorf("root dir counts should equal total; got %+v vs total %+v", res.Directories[0].Counts, res.Total)
	}
}

// TestRootCmd_PathArg_Gitignore: a .gitignore entry drops matching files
// from the count; the same tree with --no-gitignore includes them.
func TestRootCmd_PathArg_Gitignore(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		".gitignore":         {Data: []byte("vendor/\n")},
		"a.txt":              {Data: []byte("keep\n")},
		"vendor/ignored.txt": {Data: []byte("drop\n")},
	}

	t.Run("default_drops_vendor", func(t *testing.T) {
		t.Parallel()
		res, _ := runTreeFS(t, fsys, &rootFlags{})
		for _, d := range res.Directories {
			if strings.Contains(d.Path, "vendor") {
				t.Errorf("vendor should be filtered by gitignore; got path %q", d.Path)
			}
		}
		// total counts only "keep\n" = 5 bytes.
		if res.Total.Bytes != 5 {
			t.Errorf("expected Bytes=5 (keep only), got %+v", res.Total)
		}
	})

	t.Run("no_gitignore_includes_vendor", func(t *testing.T) {
		t.Parallel()
		res, _ := runTreeFS(t, fsys, &rootFlags{noGitignore: true})
		foundVendor := false
		for _, d := range res.Directories {
			if d.Path == "vendor" {
				foundVendor = true
			}
		}
		if !foundVendor {
			t.Errorf("--no-gitignore should include vendor/; got paths %v", res.paths())
		}
	})
}

// TestRootCmd_PathArg_IncludeExclude: --include limits to a pattern and
// --exclude wins over --include on conflict (F2).
func TestRootCmd_PathArg_IncludeExclude(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		"a.go":      {Data: []byte("go\n")},
		"a_test.go": {Data: []byte("test\n")},
		"readme.md": {Data: []byte("md\n")},
	}

	res, _ := runTreeFS(t, fsys, &rootFlags{
		includes: []string{"*.go"},
		excludes: []string{"*_test.go"},
	})

	// Expect only a.go to contribute. readme.md excluded by include-only
	// filter, a_test.go excluded by explicit exclude-wins rule.
	want := counting.Counts{Bytes: 3, Lines: 1, Words: 1, Chars: 3}
	if res.Total != want {
		t.Errorf("total mismatch: want %+v, got %+v", want, res.Total)
	}
}

// TestRootCmd_PathArg_Depth: a nested tree walked with Depth=1 counts only
// root-level files; Depth=0 (default) counts everything.
func TestRootCmd_PathArg_Depth(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		"a.txt":       {Data: []byte("root\n")},
		"sub/b.txt":   {Data: []byte("one\n")},
		"sub/c/d.txt": {Data: []byte("two\n")},
	}

	t.Run("unlimited", func(t *testing.T) {
		t.Parallel()
		res, _ := runTreeFS(t, fsys, &rootFlags{depth: 0})
		// All three files counted.
		if res.Total.Bytes != 13 { // 5 + 4 + 4
			t.Errorf("unlimited depth: expected Bytes=13, got %+v", res.Total)
		}
	})

	t.Run("depth_1", func(t *testing.T) {
		t.Parallel()
		res, _ := runTreeFS(t, fsys, &rootFlags{depth: 1})
		// Only a.txt at root.
		if res.Total.Bytes != 5 {
			t.Errorf("depth=1: expected Bytes=5 (root only), got %+v (paths %v)", res.Total, res.paths())
		}
	})
}

// TestRootCmd_PathArg_SkipsBinary exercises both halves of the C10 /
// F9 / F10 contract:
//
//   - Clean NUL-detected path: a file whose first byte is 0x00 is
//     classified as binary and skipped when --binary=false. --binary=true
//     includes it.
//   - Induced Peek error: a stub fs.FS whose Open returns fs.ErrPermission
//     causes IsBinary to fail. The resulting error is aggregated into the
//     render error summary and the file is skipped from counting; the
//     walk itself does NOT abort.
func TestRootCmd_PathArg_SkipsBinary(t *testing.T) {
	t.Parallel()

	t.Run("nul_detected_skipped_by_default", func(t *testing.T) {
		t.Parallel()
		fsys := fstest.MapFS{
			"a.txt":   {Data: []byte("text\n")},
			"bin.dat": {Data: []byte{0x00}},
		}
		res, _ := runTreeFS(t, fsys, &rootFlags{})
		// Only "text\n" = 5 bytes counted.
		if res.Total.Bytes != 5 {
			t.Errorf("binary default: expected Bytes=5 (text only), got %+v", res.Total)
		}
		if len(res.Errors) != 0 {
			t.Errorf("binary skip should not emit errors; got %v", res.Errors)
		}
	})

	t.Run("nul_detected_included_with_flag", func(t *testing.T) {
		t.Parallel()
		fsys := fstest.MapFS{
			"a.txt":   {Data: []byte("text\n")},
			"bin.dat": {Data: []byte{0x00}},
		}
		res, _ := runTreeFS(t, fsys, &rootFlags{binary: true})
		// Both counted: 5 + 1 = 6 bytes.
		if res.Total.Bytes != 6 {
			t.Errorf("--binary: expected Bytes=6, got %+v", res.Total)
		}
	})

	t.Run("induced_peek_error_aggregated", func(t *testing.T) {
		t.Parallel()
		// Stub fs: "a.txt" is normal, "bad.txt" returns fs.ErrPermission
		// on any Open (so both IsBinary's Peek and any count attempt fail).
		inner := fstest.MapFS{
			"a.txt":   {Data: []byte("ok\n")},
			"bad.txt": {Data: []byte("hidden\n")}, // content irrelevant; Open always fails.
		}
		fsys := &failingOpenFS{inner: inner, failPath: "bad.txt"}

		res, _ := runTreeFS(t, fsys, &rootFlags{})
		// a.txt should still be counted — the bad file does not abort
		// the walk.
		if res.Total.Bytes != 3 { // "ok\n"
			t.Errorf("induced error: a.txt should still contribute Bytes=3; got %+v", res.Total)
		}
		// At least one aggregated error must reference bad.txt.
		if len(res.Errors) == 0 {
			t.Fatalf("induced error: expected aggregated errors, got none")
		}
		var found bool
		for _, e := range res.Errors {
			if strings.Contains(e, "bad.txt") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("induced error: no aggregated error mentions bad.txt; got %v", res.Errors)
		}
	})
}

// TestRootCmd_PathArg_Hidden: hidden files are excluded by default,
// included with --hidden.
func TestRootCmd_PathArg_Hidden(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		"a.txt":       {Data: []byte("keep\n")},
		".hidden.txt": {Data: []byte("drop\n")},
	}

	t.Run("default_excludes_hidden", func(t *testing.T) {
		t.Parallel()
		res, _ := runTreeFS(t, fsys, &rootFlags{})
		if res.Total.Bytes != 5 { // "keep\n"
			t.Errorf("default: hidden excluded, expected Bytes=5, got %+v", res.Total)
		}
	})

	t.Run("hidden_flag_includes_hidden", func(t *testing.T) {
		t.Parallel()
		res, _ := runTreeFS(t, fsys, &rootFlags{hidden: true})
		if res.Total.Bytes != 10 { // both
			t.Errorf("--hidden: expected Bytes=10, got %+v", res.Total)
		}
	})
}

// TestRenderer_TreeInterface_Compile is a compile-time assertion that the
// two concrete renderers satisfy the render.Renderer interface (F15
// interface growth). The intent is to fail the build if someone drops
// RenderTree from the interface without also pruning a renderer
// implementation — we want a symmetric interface.
//
// _ = &... rather than typed-var declarations because staticcheck's QF1011
// insists the types could be inferred; the explicit interface-typed
// assertion goes in the package-level var block below.
var (
	_ render.Renderer = render.NewHumanRenderer()
	_ render.Renderer = render.NewJSONRenderer()
)

// TestRenderer_TreeInterface_Compile is a trivial runtime no-op that keeps
// the compile-time assertions above in the test build; a Go package with no
// *_test.go references beyond _ = ... still compiles, but wrapping the
// assertions in a test that the test runner reports makes the invariant
// visible in coverage output.
func TestRenderer_TreeInterface_Compile(t *testing.T) {
	t.Parallel()
	// The package-level var block above is the real assertion. This body
	// exists only so `go test` lists the invariant by name.
}

// failingOpenFS is an fs.FS test stub that delegates to an inner MapFS for
// every path except failPath, which returns fs.ErrPermission on Open. Used
// by TestRootCmd_PathArg_SkipsBinary to prove the aggregation loop surfaces
// IsBinary-Peek errors into the render summary rather than aborting the
// whole walk.
type failingOpenFS struct {
	inner    fstest.MapFS
	failPath string
}

// Open delegates to inner MapFS for directory listings (so fs.WalkDir can
// still traverse the tree) and for any file whose name is not failPath;
// opens of failPath return fs.ErrPermission.
func (f *failingOpenFS) Open(name string) (fs.File, error) {
	if name == f.failPath {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrPermission}
	}
	return f.inner.Open(name)
}
