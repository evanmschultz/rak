package fileset

import (
	"bytes"
	"errors"
)

// ErrBinaryFile is the sentinel returned by callers that want to signal a file
// was classified as binary and skipped. IsBinary itself does not return this
// sentinel — it reports the classification via its bool result — but callers
// (for example the aggregation loop in cmd/rak that skips binaries by default)
// use ErrBinaryFile when translating an IsBinary == true decision into an
// error value for their own error summary. Inspect via errors.Is, never via
// string-match. See F9 in DROP_3's PLAN.md.
var ErrBinaryFile = errors.New("binary file")

// IsBinary reports whether the file is classified as binary by the single
// NUL-byte-in-the-first-512-bytes heuristic that git and ripgrep use. It calls
// Peek(512) on the receiver and scans the returned slice for 0x00.
//
// Semantics:
//   - An empty file (Peek returns a zero-length slice) is not binary.
//   - A file whose first 512 bytes contain at least one 0x00 byte is binary.
//   - A file whose first 512 bytes contain no 0x00 byte is not binary, even
//     if a 0x00 byte appears later in the file — only the peek window is
//     sniffed. This matches git's behavior and is pinned by F10.
//   - Errors from Peek's open-read-close chain are returned verbatim; they
//     already carry the `open %q: %w` prefix from File.Open. The NUL scan
//     itself is a pure byte search and cannot fail.
//
// UTF-16 and other encodings that legitimately contain 0x00 bytes are
// misclassified as binary by this heuristic. That matches git's own behavior;
// rak intentionally inherits the limitation to keep detection O(512) and
// dependency-free (F10).
//
// Callers that want to skip binaries wrap a positive result as ErrBinaryFile
// in their own error summary; inside this package IsBinary only reports the
// classification. See F12 in DROP_3's PLAN.md — fileset stays CLI-free and
// does not decide walk policy.
func (f *File) IsBinary() (bool, error) {
	peek, err := f.Peek(512)
	if err != nil {
		return false, err
	}
	if len(peek) == 0 {
		return false, nil
	}
	return bytes.IndexByte(peek, 0x00) >= 0, nil
}
