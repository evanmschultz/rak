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
		{"foo.xml", LangXML},
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

// TestDetect_Ruby verifies extension (.rb, .rake, .gemspec), special-filename
// (Rakefile, Gemfile), and shebang detection for Ruby.
func TestDetect_Ruby(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		path    string
		content []byte
		want    Language
	}{
		{"rb extension", "foo.rb", []byte("puts 'hi'"), LangRuby},
		{"rake extension", "foo.rake", []byte("task :default"), LangRuby},
		{"gemspec extension", "my.gemspec", []byte("Gem::Specification.new"), LangRuby},
		{"Rakefile filename", "Rakefile", []byte("task :build"), LangRuby},
		{"Gemfile filename", "Gemfile", []byte("source 'https://rubygems.org'"), LangRuby},
		{"nested Rakefile", "sub/Rakefile", []byte("task :test"), LangRuby},
		{"shebang env ruby", "script", []byte("#!/usr/bin/env ruby\nputs 'hi'\n"), LangRuby},
		{"shebang usr bin ruby", "script2", []byte("#!/usr/bin/ruby\nputs 'hi'\n"), LangRuby},
		{"shebang bin ruby", "script3", []byte("#!/bin/ruby\nputs 'hi'\n"), LangRuby},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fsys := fstest.MapFS{tc.path: &fstest.MapFile{Data: tc.content}}
			f := newTestFile(fsys, tc.path)
			got := Detect(f)
			if got != tc.want {
				t.Errorf("Detect(%q) = %q; want %q", tc.path, got, tc.want)
			}
		})
	}
}

// TestDetect_Java verifies extension-based detection for Java (.java).
func TestDetect_Java(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{"Hello.java": &fstest.MapFile{Data: []byte("public class Hello {}")}}
	f := newTestFile(fsys, "Hello.java")
	got := Detect(f)
	if got != LangJava {
		t.Errorf("Detect(%q) = %q; want %q", "Hello.java", got, LangJava)
	}
}

// TestDetect_PHP verifies extension-based detection for PHP (.php, .phtml).
func TestDetect_PHP(t *testing.T) {
	t.Parallel()

	cases := []struct {
		path string
		want Language
	}{
		{"index.php", LangPHP},
		{"template.phtml", LangPHP},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.path, func(t *testing.T) {
			t.Parallel()
			fsys := fstest.MapFS{tc.path: &fstest.MapFile{Data: []byte("<?php echo 'hi'; ?>")}}
			f := newTestFile(fsys, tc.path)
			got := Detect(f)
			if got != tc.want {
				t.Errorf("Detect(%q) = %q; want %q", tc.path, got, tc.want)
			}
		})
	}
}

// TestDetect_Kotlin verifies extension-based detection for Kotlin (.kt, .kts).
func TestDetect_Kotlin(t *testing.T) {
	t.Parallel()

	cases := []struct {
		path string
		want Language
	}{
		{"Main.kt", LangKotlin},
		{"build.gradle.kts", LangKotlin},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.path, func(t *testing.T) {
			t.Parallel()
			fsys := fstest.MapFS{tc.path: &fstest.MapFile{Data: []byte("fun main() {}")}}
			f := newTestFile(fsys, tc.path)
			got := Detect(f)
			if got != tc.want {
				t.Errorf("Detect(%q) = %q; want %q", tc.path, got, tc.want)
			}
		})
	}
}

// TestDetect_Swift verifies extension-based detection for Swift (.swift).
func TestDetect_Swift(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{"main.swift": &fstest.MapFile{Data: []byte("import Foundation")}}
	f := newTestFile(fsys, "main.swift")
	got := Detect(f)
	if got != LangSwift {
		t.Errorf("Detect(%q) = %q; want %q", "main.swift", got, LangSwift)
	}
}

// TestDetect_NewLanguages_UnknownNegative verifies that an unrecognized
// extension still returns LangUnknown (regression guard for the new entries).
func TestDetect_NewLanguages_UnknownNegative(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{"foo.unknown": &fstest.MapFile{Data: []byte("no shebang\n")}}
	f := newTestFile(fsys, "foo.unknown")
	got := Detect(f)
	if got != LangUnknown {
		t.Errorf("Detect(%q) = %q; want LangUnknown", "foo.unknown", got)
	}
}

// TestDetect_XML_ExtensionAndContentSniff verifies Unit A.1: .xml extension
// maps to LangXML (not LangHTML), and content-sniff on a <?xml declaration
// also returns LangXML.
func TestDetect_XML_ExtensionAndContentSniff(t *testing.T) {
	t.Parallel()

	t.Run("extension .xml", func(t *testing.T) {
		t.Parallel()
		fsys := fstest.MapFS{"data.xml": &fstest.MapFile{Data: []byte(`<?xml version="1.0"?>`)}}
		f := newTestFile(fsys, "data.xml")
		got := Detect(f)
		if got != LangXML {
			t.Errorf("Detect(%q) = %q; want %q", "data.xml", got, LangXML)
		}
	})

	t.Run("content sniff <?xml extensionless", func(t *testing.T) {
		t.Parallel()
		// File has no extension → extension lookup returns LangUnknown.
		// Content heuristic sees <?xml prefix → must return LangXML.
		const relPath = "feed"
		fsys := fstest.MapFS{relPath: &fstest.MapFile{Data: []byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<rss/>\n")}}
		f := newTestFile(fsys, relPath)
		got := Detect(f)
		if got != LangXML {
			t.Errorf("Detect(%q) = %q; want %q", relPath, got, LangXML)
		}
	})
}

// TestDetect_HTML_Regression verifies that the XML split (Unit A.1) did not
// break HTML detection: .html and .htm extensions still return LangHTML.
func TestDetect_HTML_Regression(t *testing.T) {
	t.Parallel()

	cases := []struct {
		path string
		want Language
	}{
		{"index.html", LangHTML},
		{"page.htm", LangHTML},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.path, func(t *testing.T) {
			t.Parallel()
			fsys := fstest.MapFS{tc.path: &fstest.MapFile{Data: []byte("<!DOCTYPE html>")}}
			f := newTestFile(fsys, tc.path)
			got := Detect(f)
			if got != tc.want {
				t.Errorf("Detect(%q) = %q; want %q", tc.path, got, tc.want)
			}
		})
	}
}
