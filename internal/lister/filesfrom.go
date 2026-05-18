// Package lister — files-from reader source (Unit D.1).
package lister

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"iter"
	"os"
	"path/filepath"
	"strings"

	"github.com/evanmschultz/rak/internal/fileset"
)

// FilesFromLister is a FileLister that reads newline-separated paths from a
// caller-supplied io.Reader and yields each resolved path as a *fileset.File.
// It satisfies the FileLister iterator contract: per-path errors are yielded
// as (nil, err) pairs and iteration continues; context cancellation terminates
// iteration with (nil, ctx.Err()); the iterator stops immediately when the
// caller's yield returns false.
//
// The caller owns the reader and is responsible for closing it after listing.
// FilesFromLister never closes r.
//
// Lines are scanned with the default bufio.Scanner buffer (64 KiB per line).
// Whitespace is trimmed from each line; empty lines are skipped. No comment
// syntax — every non-empty line is treated as a path, including lines that
// start with "#" (e.g. "#draft.md").
//
// Each path is cleaned with filepath.Clean and resolved to an absolute path
// via filepath.Abs relative to the working directory at the time List is
// called. If the resolved path is not a regular file, a per-line error is
// yielded and iteration continues.
type FilesFromLister struct {
	r io.Reader
}

// NewFilesFromLister constructs a FilesFromLister that reads paths from r.
// The caller owns r and is responsible for closing it after listing.
func NewFilesFromLister(r io.Reader) *FilesFromLister {
	return &FilesFromLister{r: r}
}

// List returns an iterator that yields one *fileset.File per non-empty line in
// the reader. CWD is resolved once at the start of List so that test helpers
// that change directory between constructor and List see the correct base.
//
// Per-line pipeline:
//  1. Check ctx.Err() — cancel terminates iteration with (nil, ctx.Err()).
//  2. Scan one line; trim whitespace; skip if empty.
//  3. filepath.Clean the line.
//  4. filepath.Abs relative to CWD (resolved once at List entry).
//  5. os.Stat — must be a regular file; non-regular or missing paths yield
//     (nil, err) and iteration continues.
//  6. Build a *fileset.File via os.DirFS(dir) + filepath.Base(absPath).
//  7. Yield; stop if yield returns false (F14 carry-over).
//
// After the scan loop, scanner.Err() is checked; if non-nil, (nil, wrappedErr)
// is yielded before the iterator returns.
func (fl *FilesFromLister) List(ctx context.Context) iter.Seq2[*fileset.File, error] {
	return func(yield func(*fileset.File, error) bool) {
		// Resolve CWD once at list time so tests that os.Chdir before List
		// see the correct base directory for relative paths.
		cwd, err := os.Getwd()
		if err != nil {
			yield(nil, fmt.Errorf("lister: files-from: getwd: %w", err))
			return
		}

		scanner := bufio.NewScanner(fl.r)
		for {
			// Step 1: check context before starting the next scan.
			if ctx.Err() != nil {
				yield(nil, ctx.Err())
				return
			}

			// Step 2: scan next line.
			if !scanner.Scan() {
				break
			}
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}

			// Steps 3–4: clean + absolutise.
			// filepath.Join does not treat an absolute second argument specially —
			// it would corrupt absolute paths by prepending cwd. Check first.
			cleaned := filepath.Clean(line)
			absPath := cleaned
			if !filepath.IsAbs(absPath) {
				absPath = filepath.Join(cwd, cleaned)
			}

			// Step 5: stat — must be a regular file.
			info, statErr := os.Stat(absPath)
			if statErr != nil {
				if !yield(nil, fmt.Errorf("lister: files-from: %q is not a regular file: %w", line, statErr)) {
					return
				}
				continue
			}
			if !info.Mode().IsRegular() {
				if !yield(nil, fmt.Errorf("lister: files-from: %q is not a regular file: not a regular file", line)) {
					return
				}
				continue
			}

			// Step 6: construct the File value matching the SingleFileLister pattern:
			// both path and relPath are the bare basename so dirKey(relPath) returns
			// "." and the label logic in cmd/rak rewrites "." to the parent directory.
			dir := filepath.Dir(absPath)
			base := filepath.Base(absPath)
			f := fileset.NewFile(os.DirFS(dir), base, base)

			// Step 7: yield; stop if caller signals done.
			if !yield(f, nil) {
				return
			}
		}

		// After scan loop: propagate any scanner error.
		if err := scanner.Err(); err != nil {
			yield(nil, fmt.Errorf("lister: files-from: scanner: %w", err))
		}
	}
}

// compile-time assertion: FilesFromLister must satisfy FileLister.
var _ FileLister = (*FilesFromLister)(nil)
