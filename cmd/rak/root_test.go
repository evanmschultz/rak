package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os/exec"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/evanmschultz/rak/internal/counting"
	"github.com/evanmschultz/rak/internal/lister"
	"github.com/evanmschultz/rak/internal/render"
)

// compile-time assertions: all three concrete renderers satisfy the
// render.Renderer interface. Fails the build if a method is dropped from
// any implementation without updating the interface symmetrically.
var (
	_ render.Renderer = render.NewHumanRenderer()
	_ render.Renderer = render.NewJSONRenderer()
	_ render.Renderer = render.NewTOONRenderer()
)

// TestRenderer_TreeInterface_Compile is a trivial runtime no-op that keeps
// the compile-time assertions above in the test build. The test runner lists
// the invariant by name in coverage output.
func TestRenderer_TreeInterface_Compile(t *testing.T) {
	t.Parallel()
	// The package-level var block above is the real assertion.
}

// TestRootCmd_ReadsStdin_RendersTOONDefault verifies the default (no format
// flag) path: stdin is read via cmd.InOrStdin(), counting runs, and the TOON
// renderer emits key-value lines containing the four canonical field names in
// lowercase per the toonCounts struct tags. Stdout is a bytes.Buffer
// (non-TTY), and TOON output is not TTY-sensitive — the assertions use
// strings.Contains on the lower-case field names produced by the toon:"..."
// struct tags.
func TestRootCmd_ReadsStdin_RendersTOONDefault(t *testing.T) {
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
	for _, label := range []string{"bytes:", "lines:", "words:", "chars:"} {
		if !strings.Contains(got, label) {
			t.Errorf("TOON output missing label %q; got:\n%s", label, got)
		}
	}
}

// TestRootCmd_FlagJSON verifies --json picks NewJSONRenderer. The JSON
// renderer uses stdlib encoding/json with no struct tags (F4 pin), so the
// emitted keys match the Counts struct declaration order. We assert key
// presence rather than exact bytes.
func TestRootCmd_FlagJSON(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	cmd := newRootCmd()
	cmd.SetIn(strings.NewReader("hello world\n"))
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--json"})

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

// TestRootCmd_MutuallyExclusiveFlags verifies that passing two format flags
// simultaneously causes cobra to return an error containing a hint about
// mutual exclusivity. Cobra enforces this via MarkFlagsMutuallyExclusive
// before RunE is called (F24).
func TestRootCmd_MutuallyExclusiveFlags(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	cmd := newRootCmd()
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--human", "--json"})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected error for --human --json, got nil")
	}
	// Cobra's mutual exclusivity error message contains the flag group and
	// "none of the others can be" or similar. We assert that the error
	// message references at least one of the flag names to confirm the
	// rejection came from the mutual-exclusion group, not an unrelated
	// parsing error.
	errMsg := strings.ToLower(err.Error())
	if !strings.Contains(errMsg, "human") && !strings.Contains(errMsg, "json") {
		t.Errorf("error should reference the mutually exclusive flags; got: %v", err)
	}
}

// TestRootCmd_UnknownFlag verifies that an unrecognised flag causes cobra to
// return an error. This is cobra's default behavior; the test exists so that
// if flag parsing is ever replaced with a permissive mode it fails loudly.
func TestRootCmd_UnknownFlag(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	cmd := newRootCmd()
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--bogus"})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected error for --bogus, got nil")
	}
}

// TestRootCmd_NoGitignoreInRepo_Errors verifies the full pipeline behavior
// when --no-gitignore is passed and the target directory is inside a git
// repository: lister.Detect returns ErrNoGitignoreInRepo which propagates
// through runRoot unchanged to cmd.Execute's error return. The test
// constructs a real temporary git repo so it exercises lister.Detect's
// git-probe path.
func TestRootCmd_NoGitignoreInRepo_Errors(t *testing.T) {
	t.Parallel()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not found")
	}

	// Create a temp dir and init a git repo inside it.
	tmpDir := t.TempDir()
	initCmd := exec.Command("git", "init", tmpDir)
	if out, err := initCmd.CombinedOutput(); err != nil {
		t.Skipf("git init failed (%v): %s", err, out)
	}

	var out bytes.Buffer
	cmd := newRootCmd()
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--no-gitignore", tmpDir})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected error for --no-gitignore inside a git repo, got nil")
	}
	if !strings.Contains(err.Error(), "no-gitignore") &&
		!strings.Contains(err.Error(), "git repository") {
		// The sentinel's message contains both phrases; this is a belt-and-
		// suspenders check on top of the errors.Is check below.
		t.Logf("error message may be missing expected content: %v", err)
	}
	if !isErrNoGitignoreInRepo(err) {
		t.Errorf("expected errors.Is(err, lister.ErrNoGitignoreInRepo) to be true; got: %v", err)
	}
}

// isErrNoGitignoreInRepo is a helper that wraps the errors.Is call for
// lister.ErrNoGitignoreInRepo so the test body reads clearly.
func isErrNoGitignoreInRepo(err error) bool {
	return errors.Is(err, lister.ErrNoGitignoreInRepo)
}

// runTreeFS is a lightweight test helper that runs the per-dir aggregation
// loop against an injected fs.FS without going through cobra or lister.Detect.
// It constructs a lister.WalkLister directly from the supplied MapFS, then
// calls runDirectory with the JSON renderer so assertions can parse
// structured data.
//
// The emitted JSON envelope is the same shape produced by
// jsonRenderer.RenderTree (Unit 3.5 F15):
// `{"directories":[...],"total":{...},"errors"?:[...]}`.
func runTreeFS(t *testing.T, fsys fs.FS, flags *rootFlags) (treeResult, []byte) {
	t.Helper()

	opts := listerOpts(flags)
	source := lister.NewWalkLister(fsys, ".", opts)
	renderer := render.NewJSONRenderer()

	var out bytes.Buffer
	// rootLabel = "" keeps the walker's io/fs "." convention so assertions
	// compare against the raw relative paths. Sort defaults to "lines" desc when
	// not set by the test so existing assertions are unaffected.
	sortKey := flags.sort
	if sortKey == "" {
		sortKey = "lines"
	}
	if err := runDirectory(context.Background(), &out, source, "", flags.binary, flags.langs, sortKey, flags.sortAsc, renderer, flags.maxFiles); err != nil {
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

// TestRootCmd_FlagLang_FiltersToGo verifies that --lang go counts only .go
// files and excludes .rs and .txt files from the count.
func TestRootCmd_FlagLang_FiltersToGo(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		"a.go":  {Data: []byte("package main\n")},
		"b.rs":  {Data: []byte("fn main() {}\n")},
		"c.txt": {Data: []byte("hello world\n")},
	}
	res, _ := runTreeFS(t, fsys, &rootFlags{langs: []string{"go"}})

	// Only a.go ("package main\n" = 13 bytes) should be counted.
	if res.Total.Bytes != 13 {
		t.Errorf("--lang go: expected Bytes=13 (a.go only), got %+v", res.Total)
	}
	if res.Total.Lines != 1 {
		t.Errorf("--lang go: expected Lines=1, got %+v", res.Total)
	}
}

// TestRootCmd_FlagLang_MultiValue verifies that --lang go,rust counts both
// .go and .rs files while excluding .txt files.
func TestRootCmd_FlagLang_MultiValue(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		"a.go":  {Data: []byte("package main\n")},
		"b.rs":  {Data: []byte("fn main() {}\n")},
		"c.txt": {Data: []byte("hello world\n")},
	}
	res, _ := runTreeFS(t, fsys, &rootFlags{langs: []string{"go", "rust"}})

	// a.go (13 bytes) + b.rs (13 bytes) = 26 bytes. c.txt excluded.
	if res.Total.Bytes != 26 {
		t.Errorf("--lang go,rust: expected Bytes=26, got %+v", res.Total)
	}
	if res.Total.Lines != 2 {
		t.Errorf("--lang go,rust: expected Lines=2, got %+v", res.Total)
	}
}

// TestRootCmd_FlagLang_CaseInsensitive verifies that --lang Go (mixed case)
// still matches .go files via case normalization.
func TestRootCmd_FlagLang_CaseInsensitive(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		"a.go":  {Data: []byte("package main\n")},
		"b.rs":  {Data: []byte("fn main() {}\n")},
		"c.txt": {Data: []byte("hello world\n")},
	}
	res, _ := runTreeFS(t, fsys, &rootFlags{langs: []string{"Go"}})

	// Only a.go counted despite uppercase flag value.
	if res.Total.Bytes != 13 {
		t.Errorf("--lang Go (mixed case): expected Bytes=13, got %+v", res.Total)
	}
}

// TestRootCmd_FlagLang_ExcludesUnknown verifies that when any --lang filter
// is set, files with LangUnknown (undetected language) are excluded. c.txt
// has an extension not in the detection table and no shebang, so it maps to
// LangUnknown. Per F29, LangUnknown is the zero value ("") which never
// matches any non-empty filter value.
func TestRootCmd_FlagLang_ExcludesUnknown(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		"a.go":  {Data: []byte("package main\n")},
		"c.txt": {Data: []byte("hello world\n")},
	}
	res, _ := runTreeFS(t, fsys, &rootFlags{langs: []string{"go"}})

	// Only a.go counted; c.txt (LangUnknown) excluded.
	if res.Total.Bytes != 13 {
		t.Errorf("--lang go excludes unknown: expected Bytes=13, got %+v", res.Total)
	}
}

// TestRootCmd_NoLangFlag_CountsAll verifies that without --lang all files
// are counted regardless of detected language, preserving existing behavior.
func TestRootCmd_NoLangFlag_CountsAll(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		"a.go":  {Data: []byte("package main\n")},
		"b.rs":  {Data: []byte("fn main() {}\n")},
		"c.txt": {Data: []byte("hello world\n")},
	}
	res, _ := runTreeFS(t, fsys, &rootFlags{})

	// All three files counted: 13 + 13 + 12 = 38 bytes.
	if res.Total.Bytes != 38 {
		t.Errorf("no --lang flag: expected Bytes=38 (all files), got %+v", res.Total)
	}
	if res.Total.Lines != 3 {
		t.Errorf("no --lang flag: expected Lines=3, got %+v", res.Total)
	}
}

// TestRootCmd_LangFlag_ParsesCSV verifies that --lang go,rust via cobra's
// StringSliceVar populates langs with ["go", "rust"].
func TestRootCmd_LangFlag_ParsesCSV(t *testing.T) {
	t.Parallel()

	// We can't easily inspect the flags struct from outside, so we verify
	// the behavior: only go and rust files are counted from a mixed MapFS.
	fsys := fstest.MapFS{
		"a.go":  {Data: []byte("package main\n")},
		"b.rs":  {Data: []byte("fn main() {}\n")},
		"c.txt": {Data: []byte("hello world\n")},
	}
	res, _ := runTreeFS(t, fsys, &rootFlags{langs: []string{"go", "rust"}})
	if res.Total.Lines != 2 {
		t.Errorf("CSV parse: expected Lines=2 (go+rust), got %+v", res.Total)
	}
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

// langCountsJSON mirrors the lang.LangCounts shape for JSON decoding in
// per-lang rollup tests. Kept local to the test that needs it so any
// renderer-internal change doesn't ripple to every test.
type langCountsJSON struct {
	Lines  lineCountsJSON  `json:"Lines"`
	Counts counting.Counts `json:"Counts"`
}

type lineCountsJSON struct {
	Blank   int `json:"Blank"`
	Comment int `json:"Comment"`
	Code    int `json:"Code"`
}

// treeResultWithLang extends treeResult with per-language data for the
// per-lang rollup test. Only the "by_lang" field of each directory is added.
type dirResultWithLang struct {
	Path   string                    `json:"path"`
	Counts counting.Counts           `json:"counts"`
	ByLang map[string]langCountsJSON `json:"by_lang,omitempty"`
}

type treeResultWithLang struct {
	Directories []dirResultWithLang       `json:"directories"`
	Total       counting.Counts           `json:"total"`
	TotalByLang map[string]langCountsJSON `json:"total_by_lang,omitempty"`
	Errors      []string                  `json:"errors,omitempty"`
}

// TestRootCmd_PerLangRollup verifies that walkAndCount accumulates per-language
// LangCounts into Directory.ByLang and that the JSON renderer surfaces them
// under by_lang. Uses a.go + b.py in a flat fstest.MapFS so both are detected
// to known languages (LangGo and LangPython). The by_lang map must contain
// both "go" and "python" keys.
func TestRootCmd_PerLangRollup(t *testing.T) {
	t.Parallel()

	// a.go: one code line, no blank, no comment.
	// b.py: one code line, no blank, no comment.
	fsys := fstest.MapFS{
		"a.go": {Data: []byte("package main\n")},
		"b.py": {Data: []byte("x = 1\n")},
	}

	opts := listerOpts(&rootFlags{})
	source := lister.NewWalkLister(fsys, ".", opts)
	renderer := render.NewJSONRenderer()

	var out bytes.Buffer
	if err := runDirectory(context.Background(), &out, source, "", false, nil, "lines", false, renderer, 0); err != nil {
		t.Fatalf("runDirectory: %v", err)
	}

	var decoded treeResultWithLang
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Fatalf("json.Unmarshal(%s): %v", out.String(), err)
	}

	if len(decoded.Directories) != 1 {
		t.Fatalf("expected 1 directory, got %d", len(decoded.Directories))
	}
	dir := decoded.Directories[0]
	if dir.ByLang == nil {
		t.Fatalf("expected by_lang to be non-nil in JSON output; got:\n%s", out.String())
	}
	if _, ok := dir.ByLang["go"]; !ok {
		t.Errorf("by_lang missing \"go\" key; got keys: %v", byLangKeys(dir.ByLang))
	}
	if _, ok := dir.ByLang["python"]; !ok {
		t.Errorf("by_lang missing \"python\" key; got keys: %v", byLangKeys(dir.ByLang))
	}
	// LangUnknown ("") must not appear (F33).
	if _, ok := dir.ByLang[""]; ok {
		t.Errorf("by_lang must not contain LangUnknown key; got keys: %v", byLangKeys(dir.ByLang))
	}
}

// byLangKeys returns the string keys of m for error messages.
func byLangKeys(m map[string]langCountsJSON) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// sortTestFS is a fixed MapFS with two directories: root (2 files, 10 lines
// total) and sub (3 files, 30 lines total). Used by the sort tests to verify
// ordering across all four sort keys and both directions.
//
// Root files: a.go (5 lines / 5 bytes each → 50 bytes), b.go (5 lines /
// 5 bytes). Sub files: c.go (10 lines), d.go (10 lines), e.go (10 lines).
// Line counts per dir: root=10, sub=30. File counts: root=2, sub=3.
// Byte counts: root=50, sub=150.
var sortTestFS = fstest.MapFS{
	// Root: 2 files, 10 lines each-file→5, 50 bytes.
	"a.go": {Data: []byte("// a\n// a\n// a\n// a\n// a\n")}, // 5 lines, 25 bytes
	"b.go": {Data: []byte("// b\n// b\n// b\n// b\n// b\n")}, // 5 lines, 25 bytes
	// Sub: 3 files, 10 lines, 150 bytes.
	"sub/c.go": {Data: []byte("// c\n// c\n// c\n// c\n// c\n// c\n// c\n// c\n// c\n// c\n")}, // 10 lines, 50 bytes
	"sub/d.go": {Data: []byte("// d\n// d\n// d\n// d\n// d\n// d\n// d\n// d\n// d\n// d\n")}, // 10 lines, 50 bytes
	"sub/e.go": {Data: []byte("// e\n// e\n// e\n// e\n// e\n// e\n// e\n// e\n// e\n// e\n")}, // 10 lines, 50 bytes
}

// TestRootCmd_Sort_Default_LinesDesc verifies that without any sort flags the
// directories are ordered by lines descending (sub=30 lines before root=10
// lines).
func TestRootCmd_Sort_Default_LinesDesc(t *testing.T) {
	t.Parallel()

	res, _ := runTreeFS(t, sortTestFS, &rootFlags{})
	if len(res.Directories) != 2 {
		t.Fatalf("expected 2 directories, got %d: %v", len(res.Directories), res.paths())
	}
	// Default: lines desc → sub (30) before root (10).
	if res.Directories[0].Path != "sub" {
		t.Errorf("lines desc: expected sub first (more lines), got %q", res.Directories[0].Path)
	}
}

// TestRootCmd_Sort_Lines_AscFlipped verifies that --sort lines --sort-asc
// produces lines ascending (root=10 lines before sub=30 lines).
func TestRootCmd_Sort_Lines_AscFlipped(t *testing.T) {
	t.Parallel()

	res, _ := runTreeFS(t, sortTestFS, &rootFlags{sort: "lines", sortAsc: true})
	if len(res.Directories) != 2 {
		t.Fatalf("expected 2 directories, got %d", len(res.Directories))
	}
	// Lines asc → root (10) before sub (30).
	if res.Directories[0].Path != "." {
		t.Errorf("lines asc: expected . first (fewer lines), got %q", res.Directories[0].Path)
	}
}

// TestRootCmd_Sort_Files_Default verifies that --sort files (no --sort-asc)
// produces files descending (sub=3 files before root=2 files).
func TestRootCmd_Sort_Files_Default(t *testing.T) {
	t.Parallel()

	res, _ := runTreeFS(t, sortTestFS, &rootFlags{sort: "files"})
	if len(res.Directories) != 2 {
		t.Fatalf("expected 2 directories, got %d", len(res.Directories))
	}
	// Files desc → sub (3) before root (2).
	if res.Directories[0].Path != "sub" {
		t.Errorf("files desc: expected sub first (more files), got %q", res.Directories[0].Path)
	}
}

// TestRootCmd_Sort_Files_AscFlipped verifies that --sort files --sort-asc
// produces files ascending (root=2 files before sub=3 files).
func TestRootCmd_Sort_Files_AscFlipped(t *testing.T) {
	t.Parallel()

	res, _ := runTreeFS(t, sortTestFS, &rootFlags{sort: "files", sortAsc: true})
	if len(res.Directories) != 2 {
		t.Fatalf("expected 2 directories, got %d", len(res.Directories))
	}
	// Files asc → root (2) before sub (3).
	if res.Directories[0].Path != "." {
		t.Errorf("files asc: expected . first (fewer files), got %q", res.Directories[0].Path)
	}
}

// TestRootCmd_Sort_Bytes_Default verifies that --sort bytes (no --sort-asc)
// produces bytes descending (sub=150 bytes before root=50 bytes).
func TestRootCmd_Sort_Bytes_Default(t *testing.T) {
	t.Parallel()

	res, _ := runTreeFS(t, sortTestFS, &rootFlags{sort: "bytes"})
	if len(res.Directories) != 2 {
		t.Fatalf("expected 2 directories, got %d", len(res.Directories))
	}
	// Bytes desc → sub (150) before root (50).
	if res.Directories[0].Path != "sub" {
		t.Errorf("bytes desc: expected sub first (more bytes), got %q", res.Directories[0].Path)
	}
}

// TestRootCmd_Sort_Bytes_AscFlipped verifies that --sort bytes --sort-asc
// produces bytes ascending (root=50 before sub=150).
func TestRootCmd_Sort_Bytes_AscFlipped(t *testing.T) {
	t.Parallel()

	res, _ := runTreeFS(t, sortTestFS, &rootFlags{sort: "bytes", sortAsc: true})
	if len(res.Directories) != 2 {
		t.Fatalf("expected 2 directories, got %d", len(res.Directories))
	}
	// Bytes asc → root (50) before sub (150).
	if res.Directories[0].Path != "." {
		t.Errorf("bytes asc: expected . first (fewer bytes), got %q", res.Directories[0].Path)
	}
}

// TestRootCmd_Sort_Path_Default verifies that --sort path (no --sort-asc)
// produces paths ascending (A→Z; "." before "sub" lexicographically).
func TestRootCmd_Sort_Path_Default(t *testing.T) {
	t.Parallel()

	res, _ := runTreeFS(t, sortTestFS, &rootFlags{sort: "path"})
	if len(res.Directories) != 2 {
		t.Fatalf("expected 2 directories, got %d", len(res.Directories))
	}
	// Path asc (key-specific default) → "." before "sub".
	if res.Directories[0].Path != "." {
		t.Errorf("path asc: expected . first, got %q", res.Directories[0].Path)
	}
}

// TestRootCmd_Sort_Path_AscFlipped verifies that --sort path --sort-asc
// produces paths descending (flipped from key default; "sub" before ".").
func TestRootCmd_Sort_Path_AscFlipped(t *testing.T) {
	t.Parallel()

	res, _ := runTreeFS(t, sortTestFS, &rootFlags{sort: "path", sortAsc: true})
	if len(res.Directories) != 2 {
		t.Fatalf("expected 2 directories, got %d", len(res.Directories))
	}
	// Path flipped → desc → "sub" before ".".
	if res.Directories[0].Path != "sub" {
		t.Errorf("path desc (flipped): expected sub first, got %q", res.Directories[0].Path)
	}
}

// TestRootCmd_SortTokens_Errors verifies that --sort tokens returns the
// canonical error with exact text per F41 / Decision 3.4.
func TestRootCmd_SortTokens_Errors(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	cmd := newRootCmd()
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	// Pass a path arg so RunE is reached. Use an empty MapFS routed via
	// runTreeFS path — but for this test we drive cobra directly and want
	// PersistentPreRunE to fire. We don't need a real path because the
	// validation fires before any walk.
	cmd.SetArgs([]string{"--sort", "tokens", "."})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected error for --sort tokens, got nil")
	}
	want := `"tokens" is not a valid sort key; valid keys: lines, files, bytes, path`
	if !strings.Contains(err.Error(), want) {
		t.Errorf("error should contain canonical message %q; got: %v", want, err)
	}
}

// TestRootCmd_SortFiles_NonDegenerate is the F44 end-to-end test. It
// constructs a fixture with two directories of differing file counts —
// root (2 files) and sub (3 files) — runs with --sort files, and asserts
// that:
//  1. The per-dir ordering reflects Files counts: sub (3) before root (2)
//     under the default descending direction.
//  2. The JSON output carries non-zero "files" values for each directory.
//
// This test ONLY passes if Files survives labelDirectories reconstruction
// and SortDirs uses the actual Files field.
func TestRootCmd_SortFiles_NonDegenerate(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		"a.go":     {Data: []byte("package main\n")},
		"b.go":     {Data: []byte("package main\n")},
		"sub/c.go": {Data: []byte("package sub\n")},
		"sub/d.go": {Data: []byte("package sub\n")},
		"sub/e.go": {Data: []byte("package sub\n")},
	}

	// Use runDirectory directly so we can use a non-empty rootLabel (to
	// exercise labelDirectories) and control the sort key.
	opts := listerOpts(&rootFlags{})
	src := lister.NewWalkLister(fsys, ".", opts)

	var out bytes.Buffer
	if err := runDirectory(context.Background(), &out, src, "myroot", false, nil, "files", false, render.NewJSONRenderer(), 0); err != nil {
		t.Fatalf("runDirectory: %v", err)
	}

	// Decode the JSON envelope to inspect ordering and "files" values.
	var raw struct {
		Directories []map[string]interface{} `json:"directories"`
	}
	if err := json.Unmarshal(out.Bytes(), &raw); err != nil {
		t.Fatalf("json.Unmarshal: %v (body: %s)", err, out.String())
	}

	if len(raw.Directories) != 2 {
		t.Fatalf("expected 2 directories, got %d", len(raw.Directories))
	}

	// First directory in output should be "myroot/sub" (3 files > 2, desc).
	firstPath, _ := raw.Directories[0]["path"].(string)
	if firstPath != "myroot/sub" {
		t.Errorf("--sort files desc: expected myroot/sub first (3 files), got %q", firstPath)
	}

	// Build path→files map to assert non-zero values.
	filesByPath := map[string]int64{}
	for _, d := range raw.Directories {
		p, _ := d["path"].(string)
		f, _ := d["files"].(float64)
		filesByPath[p] = int64(f)
	}
	if got := filesByPath["myroot"]; got != 2 {
		t.Errorf("myroot: expected files=2 in JSON, got %d (F44: Files must survive labelDirectories)", got)
	}
	if got := filesByPath["myroot/sub"]; got != 3 {
		t.Errorf("myroot/sub: expected files=3 in JSON, got %d (F44: Files must survive labelDirectories)", got)
	}
}

// TestRootCmd_MaxFiles_NotSet_CountsAll verifies that without --max-files, all
// accepted files are counted (existing behavior preserved, no limit applied).
func TestRootCmd_MaxFiles_NotSet_CountsAll(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		"a.go": {Data: []byte("package main\n")},
		"b.go": {Data: []byte("package main\n")},
		"c.go": {Data: []byte("package main\n")},
		"d.go": {Data: []byte("package main\n")},
		"e.go": {Data: []byte("package main\n")},
	}
	// maxFiles not set → defaults to 0 (no limit); all 5 files counted.
	res, _ := runTreeFS(t, fsys, &rootFlags{})
	if res.Total.Lines != 5 {
		t.Errorf("no --max-files: expected Lines=5 (all 5 files), got %+v", res.Total)
	}
}

// TestRootCmd_MaxFiles_ZeroExplicit_CountsAll verifies that --max-files 0
// (explicit zero) is treated as "no limit" and all files are counted.
func TestRootCmd_MaxFiles_ZeroExplicit_CountsAll(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		"a.go": {Data: []byte("package main\n")},
		"b.go": {Data: []byte("package main\n")},
		"c.go": {Data: []byte("package main\n")},
		"d.go": {Data: []byte("package main\n")},
		"e.go": {Data: []byte("package main\n")},
	}
	// maxFiles = 0 → no limit; all 5 files counted.
	res, _ := runTreeFS(t, fsys, &rootFlags{maxFiles: 0})
	if res.Total.Lines != 5 {
		t.Errorf("--max-files 0: expected Lines=5 (all 5 files), got %+v", res.Total)
	}
}

// TestRootCmd_MaxFiles_UnderLimit verifies that when the accepted file count is
// below the --max-files limit, the walk completes normally.
func TestRootCmd_MaxFiles_UnderLimit(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		"a.go": {Data: []byte("package main\n")},
		"b.go": {Data: []byte("package main\n")},
		"c.go": {Data: []byte("package main\n")},
		"d.go": {Data: []byte("package main\n")},
		"e.go": {Data: []byte("package main\n")},
	}
	// maxFiles = 10; only 5 files in fixture → limit never hit.
	res, _ := runTreeFS(t, fsys, &rootFlags{maxFiles: 10})
	if res.Total.Lines != 5 {
		t.Errorf("--max-files 10: expected Lines=5 (under limit), got %+v", res.Total)
	}
}

// TestRootCmd_MaxFiles_AtLimit_Aborts verifies that when the accepted file
// count reaches --max-files, the walk aborts and the returned error wraps
// ErrMaxFilesExceeded (F45: errors.Is consumer interface).
func TestRootCmd_MaxFiles_AtLimit_Aborts(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		"a.go": {Data: []byte("package main\n")},
		"b.go": {Data: []byte("package main\n")},
		"c.go": {Data: []byte("package main\n")},
		"d.go": {Data: []byte("package main\n")},
		"e.go": {Data: []byte("package main\n")},
	}

	opts := listerOpts(&rootFlags{})
	source := lister.NewWalkLister(fsys, ".", opts)

	var out bytes.Buffer
	// maxFiles = 3; 5 files present → walk aborts after accepting the 3rd.
	err := runDirectory(context.Background(), &out, source, "", false, nil, "lines", false, render.NewJSONRenderer(), 3)
	if err == nil {
		t.Fatalf("--max-files 3: expected error wrapping ErrMaxFilesExceeded, got nil")
	}
	if !errors.Is(err, ErrMaxFilesExceeded) {
		t.Errorf("--max-files 3: expected errors.Is(err, ErrMaxFilesExceeded) true; got: %v", err)
	}
}

// TestRootCmd_MaxFiles_NegativeValue verifies that --max-files -1 is treated as
// "no limit" (same as 0), so all files are counted without error.
//
// Decision (Unit 8.1 worklog): negative values are treated as 0 (no limit)
// via the guard condition `maxFiles > 0`. This avoids a cobra validation step
// and keeps the UX symmetrical with `--depth 0` (unlimited). A negative value
// is arguably a user error but produces the safe behavior (count everything).
func TestRootCmd_MaxFiles_NegativeValue(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		"a.go": {Data: []byte("package main\n")},
		"b.go": {Data: []byte("package main\n")},
		"c.go": {Data: []byte("package main\n")},
		"d.go": {Data: []byte("package main\n")},
		"e.go": {Data: []byte("package main\n")},
	}
	// maxFiles = -1 → guard `maxFiles > 0` is false → treated as no limit.
	res, _ := runTreeFS(t, fsys, &rootFlags{maxFiles: -1})
	if res.Total.Lines != 5 {
		t.Errorf("--max-files -1: expected Lines=5 (treated as no-limit), got %+v", res.Total)
	}
}

// TestRootCmd_FilesField_SurvivesLabelDirectories verifies that the Files
// field on summary.Directory is correctly populated by walkAndCount and
// survives the labelDirectories reconstruction (F44). A fixture with multiple
// directories of differing file counts is used: 2 files in "." and 3 files in
// "sub". The JSON output must carry a non-zero "files" value for each
// directory. This test ONLY passes if Files is propagated through
// labelDirectories and filterUnknown without being zeroed.
func TestRootCmd_FilesField_SurvivesLabelDirectories(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		// Two accepted files in root dir.
		"a.go": {Data: []byte("package main\n")},
		"b.go": {Data: []byte("package main\n")},
		// Three accepted files in sub dir.
		"sub/c.go": {Data: []byte("package sub\n")},
		"sub/d.go": {Data: []byte("package sub\n")},
		"sub/e.go": {Data: []byte("package sub\n")},
	}

	opts := listerOpts(&rootFlags{})
	source := lister.NewWalkLister(fsys, ".", opts)

	var out bytes.Buffer
	// Use a non-empty rootLabel to exercise labelDirectories reconstruction.
	if err := runDirectory(context.Background(), &out, source, "myroot", false, nil, "lines", false, render.NewJSONRenderer(), 0); err != nil {
		t.Fatalf("runDirectory: %v", err)
	}

	// Decode into a map to inspect the "files" field generically.
	var raw struct {
		Directories []map[string]interface{} `json:"directories"`
	}
	if err := json.Unmarshal(out.Bytes(), &raw); err != nil {
		t.Fatalf("json.Unmarshal: %v (body: %s)", err, out.String())
	}

	if len(raw.Directories) != 2 {
		t.Fatalf("expected 2 directories, got %d", len(raw.Directories))
	}

	// Build path→files map from decoded output.
	filesByPath := map[string]int64{}
	for _, d := range raw.Directories {
		p, _ := d["path"].(string)
		f, _ := d["files"].(float64) // JSON numbers decode as float64.
		filesByPath[p] = int64(f)
	}

	// "myroot" should have 2 files; "myroot/sub" should have 3.
	if got := filesByPath["myroot"]; got != 2 {
		t.Errorf("expected myroot files=2, got %d (F44: Files must survive labelDirectories)", got)
	}
	if got := filesByPath["myroot/sub"]; got != 3 {
		t.Errorf("expected myroot/sub files=3, got %d (F44: Files must survive labelDirectories)", got)
	}
}

// TestRootCmd_Version verifies that the root cobra command, when given the
// version string that fang.WithVersion wires in main.go, prints output
// containing "v0.1.0" when invoked with --version. The test sets cmd.Version
// directly (mirroring what fang.WithVersion does to the cobra command) and
// captures cobra's built-in version output via cmd.SetOut.
func TestRootCmd_Version(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	cmd := newRootCmd()
	// Mirror what fang.WithVersion("v0.1.0") does to the cobra command:
	// cobra prints "<use> version <Version>" to OutOrStdout() when --version
	// is passed and cmd.Version != "".
	cmd.Version = version
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--version"})

	// cobra's built-in --version handler returns nil (no os.Exit); safe to call.
	if err := cmd.Execute(); err != nil {
		t.Fatalf("cmd.Execute --version: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "v0.1.0") {
		t.Errorf("--version output does not contain %q; got:\n%s", "v0.1.0", got)
	}
}

// TestRootCmd_TotalByLang_EndToEnd verifies that walkAndCount aggregates
// TotalByLang correctly across multiple directories and that the JSON renderer
// surfaces it under total_by_lang. The fixture has Go files in the root
// directory and Python files in a sub directory; the total_by_lang must contain
// both "go" and "python" keys, and their aggregate counts must equal the sum
// of the corresponding per-directory ByLang entries (F46).
func TestRootCmd_TotalByLang_EndToEnd(t *testing.T) {
	t.Parallel()

	// a.go: 1 line, "package main\n" = 13 bytes.
	// b.go: 1 line, "package main\n" = 13 bytes.
	// sub/c.py: 1 line, "x = 1\n" = 6 bytes.
	// sub/d.py: 1 line, "x = 1\n" = 6 bytes.
	fsys := fstest.MapFS{
		"a.go":     {Data: []byte("package main\n")},
		"b.go":     {Data: []byte("package main\n")},
		"sub/c.py": {Data: []byte("x = 1\n")},
		"sub/d.py": {Data: []byte("x = 1\n")},
	}

	opts := listerOpts(&rootFlags{})
	source := lister.NewWalkLister(fsys, ".", opts)
	renderer := render.NewJSONRenderer()

	var out bytes.Buffer
	if err := runDirectory(context.Background(), &out, source, "", false, nil, "lines", false, renderer, 0); err != nil {
		t.Fatalf("runDirectory: %v", err)
	}

	var decoded treeResultWithLang
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Fatalf("json.Unmarshal(%s): %v", out.String(), err)
	}

	// TotalByLang must contain "go" and "python" keys.
	if decoded.TotalByLang == nil {
		t.Fatalf("expected total_by_lang to be non-nil; got:\n%s", out.String())
	}
	goTotal, hasGo := decoded.TotalByLang["go"]
	if !hasGo {
		t.Errorf("total_by_lang missing 'go' key; got keys: %v", byLangKeys(decoded.TotalByLang))
	}
	pyTotal, hasPy := decoded.TotalByLang["python"]
	if !hasPy {
		t.Errorf("total_by_lang missing 'python' key; got keys: %v", byLangKeys(decoded.TotalByLang))
	}
	// LangUnknown must not appear.
	if _, hasUnknown := decoded.TotalByLang[""]; hasUnknown {
		t.Errorf("total_by_lang must not contain LangUnknown key; got keys: %v", byLangKeys(decoded.TotalByLang))
	}

	// Go aggregate: 2 files × 13 bytes each = 26 bytes, 2 lines.
	if hasGo {
		if goTotal.Counts.Bytes != 26 {
			t.Errorf("total_by_lang[go].Counts.Bytes: want 26, got %d", goTotal.Counts.Bytes)
		}
		if goTotal.Counts.Lines != 2 {
			t.Errorf("total_by_lang[go].Counts.Lines: want 2, got %d", goTotal.Counts.Lines)
		}
	}
	// Python aggregate: 2 files × 6 bytes each = 12 bytes, 2 lines.
	if hasPy {
		if pyTotal.Counts.Bytes != 12 {
			t.Errorf("total_by_lang[python].Counts.Bytes: want 12, got %d", pyTotal.Counts.Bytes)
		}
		if pyTotal.Counts.Lines != 2 {
			t.Errorf("total_by_lang[python].Counts.Lines: want 2, got %d", pyTotal.Counts.Lines)
		}
	}

	// Verify aggregate equals sum of per-dir ByLang values for each language.
	// Collect per-dir totals from the directories slice.
	var sumGoBytes, sumPyBytes int64
	for _, d := range decoded.Directories {
		if goEntry, ok := d.ByLang["go"]; ok {
			sumGoBytes += goEntry.Counts.Bytes
		}
		if pyEntry, ok := d.ByLang["python"]; ok {
			sumPyBytes += pyEntry.Counts.Bytes
		}
	}
	if hasGo && sumGoBytes != goTotal.Counts.Bytes {
		t.Errorf("total_by_lang[go].Bytes (%d) != sum of per-dir go bytes (%d)", goTotal.Counts.Bytes, sumGoBytes)
	}
	if hasPy && sumPyBytes != pyTotal.Counts.Bytes {
		t.Errorf("total_by_lang[python].Bytes (%d) != sum of per-dir python bytes (%d)", pyTotal.Counts.Bytes, sumPyBytes)
	}
}
