package lsp

import (
	"bytes"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/pkg/errors"
	"github.com/sourcegraph/go-lsp"
)

func trimFilePrefix(s string) string {
	return strings.TrimPrefix(s, "file://")
}

// IsURI returns true if the given URI is a file:// URI.
func IsURI(uri lsp.DocumentURI) bool {
	return strings.HasPrefix(string(uri), "file://")
}

// URIToPath converts a URI to a file path.
func URIToPath(uri lsp.DocumentURI) string {
	u, err := url.Parse(string(uri))
	if err != nil {
		return trimFilePrefix(string(uri))
	}
	return u.Path
}

// offsetForPosition converts a protocol (UTF-16) position to a byte offset. content is utf8 encoded sequence.
func offsetForPosition(content []byte, p lsp.Position) (byteOffset int, whyInvalid error) {
	newLineCount := bytes.Count(content, []byte("\n"))
	lineNumber := newLineCount + 1
	// lineStartOffsets records the byte offset of each line start.
	lineStartOffsets := make([]int, 1, lineNumber) // Init as {0}
	for offset, b := range content {
		if b == '\n' {
			nextLineBeginOffset := offset + 1
			lineStartOffsets = append(lineStartOffsets, nextLineBeginOffset)
		}
	}

	// Validate line number, notes that the Posititon in LSP is 0-based.
	if p.Line >= lineNumber {
		return 0, errors.Errorf("line number %d out of range [0, %d)", p.Line, lineNumber)
	}

	offset := lineStartOffsets[p.Line]
	trimmedContent := content[offset:]

	col8 := 0
	for col16 := 0; col16 < p.Character; col16++ {
		// Unpack the first rune.
		r, sz := utf8.DecodeRune(trimmedContent)
		if sz == 0 {
			return 0, errors.Errorf("column is beyond end of file")
		}
		if r == '\n' {
			return 0, errors.Errorf("column is beyond end of line")
		}
		if sz == 1 && r == utf8.RuneError {
			return 0, errors.Errorf("mem buffer contains invalid UTF-8 text")
		}

		// Step to the first code unit of next rune.
		trimmedContent = trimmedContent[sz:]

		if r > 0xFFFF {
			// Rune was encoded by a pair of surrogate UTF-16 codes.
			col16++

			// Requested position is in the middle of a rune? Valid?
			if col16 == int(p.Character) {
				break
			}
		}
		col8 += sz
	}
	return offset + col8, nil
}
