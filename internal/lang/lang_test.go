package lang

import (
	"testing"
	"testing/fstest"

	"github.com/evanmschultz/rak/internal/fileset"
)

// newTestFile constructs a *fileset.File from a MapFS for use in Detect tests.
// path and relPath are set to the same value (walk root "." convention).
func newTestFile(fsys fstest.MapFS, relPath string) *fileset.File {
	return fileset.NewFile(fsys, relPath, relPath)
}

// TestDetect_ByExtension verifies that extension-based lookup returns the
// expected Language for every entry in extensionTable.
func TestDetect_ByExtension(t *testing.T) {
	t.Parallel()

	cases := []struct {
		path string
		want Language
	}{
		{"foo.go", LangGo},
		{"foo.rs", LangRust},
		{"foo.py", LangPython},
		{"foo.js", LangJS},
		{"foo.ts", LangTS},
		{"foo.sh", LangShell},
		{"foo.md", LangMarkdown},
		{"foo.toml", LangTOML},
		{"foo.yaml", LangYAML},
		{"foo.yml", LangYAML},
		{"foo.json", LangJSON},
		{"foo.c", LangC},
		{"foo.cpp", LangCPP},
		{"foo.cc", LangCPP},
		{"foo.html", LangHTML},
		{"foo.css", LangCSS},
		{"foo.xyzzy", LangUnknown},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.path, func(t *testing.T) {
			t.Parallel()
			fsys := fstest.MapFS{tc.path: &fstest.MapFile{Data: []byte("content")}}
			f := newTestFile(fsys, tc.path)
			got := Detect(f)
			if got != tc.want {
				t.Errorf("Detect(%q) = %q; want %q", tc.path, got, tc.want)
			}
		})
	}
}

// TestDetect_SpecialFilename verifies exact-basename matching for special
// filenames (Makefile, Dockerfile, CMakeLists.txt), including nested paths and
// the Makefile.go fallthrough case.
func TestDetect_SpecialFilename(t *testing.T) {
	t.Parallel()

	cases := []struct {
		path string
		want Language
	}{
		{"Makefile", LangMakefile},
		{"makefile", LangMakefile},
		{"GNUmakefile", LangMakefile},
		{"Dockerfile", LangDocker},
		{"CMakeLists.txt", LangCMake},
		// Nested path: basename match still works.
		{"sub/Makefile", LangMakefile},
		// Basename "Makefile.go" does NOT match "makefile" in special table;
		// falls through to extension lookup → ".go" → LangGo.
		{"Makefile.go", LangGo},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.path, func(t *testing.T) {
			t.Parallel()
			fsys := fstest.MapFS{tc.path: &fstest.MapFile{Data: []byte("content")}}
			f := newTestFile(fsys, tc.path)
			got := Detect(f)
			if got != tc.want {
				t.Errorf("Detect(%q) = %q; want %q", tc.path, got, tc.want)
			}
		})
	}
}

// TestDetect_Shebang_Shell verifies that a file with no extension whose first
// line is "#!/bin/bash" is detected as LangShell.
func TestDetect_Shebang_Shell(t *testing.T) {
	t.Parallel()

	const relPath = "script_no_ext"
	fsys := fstest.MapFS{relPath: &fstest.MapFile{Data: []byte("#!/bin/bash\necho hi\n")}}
	f := newTestFile(fsys, relPath)
	got := Detect(f)
	// Builder choice: bash maps to LangShell (not a separate LangBash constant).
	// Documented in BUILDER_WORKLOG.md.
	if got != LangShell {
		t.Errorf("Detect(%q) = %q; want %q", relPath, got, LangShell)
	}
}

// TestDetect_Shebang_Python verifies that "#!/usr/bin/env python3" resolves to
// LangPython via the env-interpreter lookup path.
func TestDetect_Shebang_Python(t *testing.T) {
	t.Parallel()

	const relPath = "script"
	fsys := fstest.MapFS{relPath: &fstest.MapFile{Data: []byte("#!/usr/bin/env python3\nprint('hi')\n")}}
	f := newTestFile(fsys, relPath)
	got := Detect(f)
	if got != LangPython {
		t.Errorf("Detect(%q) = %q; want %q", relPath, got, LangPython)
	}
}

// TestDetect_UnknownExtension_NoShebang verifies that a file with an
// unrecognized extension and no shebang returns LangUnknown.
func TestDetect_UnknownExtension_NoShebang(t *testing.T) {
	t.Parallel()

	const relPath = "foo.xyzzy"
	fsys := fstest.MapFS{relPath: &fstest.MapFile{Data: []byte("no shebang here\n")}}
	f := newTestFile(fsys, relPath)
	got := Detect(f)
	if got != LangUnknown {
		t.Errorf("Detect(%q) = %q; want %q (LangUnknown)", relPath, got, LangUnknown)
	}
}

// TestDetect_ExtensionBeatsShebang verifies that extension lookup (step 2)
// takes priority over shebang sniff (step 3): foo.go with a bash shebang
// returns LangGo, not LangShell.
func TestDetect_ExtensionBeatsShebang(t *testing.T) {
	t.Parallel()

	const relPath = "foo.go"
	fsys := fstest.MapFS{relPath: &fstest.MapFile{Data: []byte("#!/usr/bin/env bash\npackage main\n")}}
	f := newTestFile(fsys, relPath)
	got := Detect(f)
	if got != LangGo {
		t.Errorf("Detect(%q) = %q; want %q", relPath, got, LangGo)
	}
}

// TestDetect_PeekError_ReturnsUnknown verifies that when Peek fails (because
// the File's path does not exist in the underlying fs.FS), Detect returns
// LangUnknown without panicking. The file has an unknown extension so steps 1
// and 2 return LangUnknown; step 3 attempts Peek, which fails.
func TestDetect_PeekError_ReturnsUnknown(t *testing.T) {
	t.Parallel()

	// The MapFS is empty — "nonexistent.xyzzy" is not registered, so Open will
	// return fs.ErrNotExist and Peek will propagate that as an error.
	fsys := fstest.MapFS{}
	f := fileset.NewFile(fsys, "nonexistent.xyzzy", "nonexistent.xyzzy")
	got := Detect(f)
	if got != LangUnknown {
		t.Errorf("Detect on missing file = %q; want %q (LangUnknown)", got, LangUnknown)
	}
}
