package lsp

import (
	"bytes"
	"net/url"
	"strings"
	"unicode"
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

	// Validate line number, notes that the Position in LSP is 0-based.
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

func getSQLStatementRangesUTF16Position(content []byte) []lsp.Range {
	s := strings.TrimRightFunc(string(content), unicode.IsSpace)
	// Assuming the content is UTF-8 encoded.
	statements := strings.Split(s, ";")

	var ranges []lsp.Range
	// 0-based UTF-16 encoded line and character.
	line, character := 0, 0

	for i, statement := range statements {
		// Trim left space to provide more accurate range.
		statement = strings.TrimLeftFunc(statement, func(r rune) bool {
			if !unicode.IsSpace(r) {
				return false
			}
			if r == '\n' {
				line++
				character = 0
			} else {
				// Check rune utf16 length by BMP.
				if r <= 0xFFFF {
					character++
				} else {
					character += 2
				}
			}
			return true
		})

		// If the statement is empty, skip it.
		if statement == "" {
			continue
		}

		begin := lsp.Position{Line: line, Character: character}
		for _, r := range statement {
			if r == '\n' {
				line++
				character = 0
			} else {
				// Check rune utf16 length by BMP.
				if r <= 0xFFFF {
					character++
				} else {
					character += 2
				}
			}
		}

		endLine, endCharacter := line, character
		// End is exclusive, so we check the next byte.
		if i == len(statements)-1 {
			// End of the content.
			endLine++
			endCharacter = 0
		} else {
			// Next byte is ';', include it.
			character++
			endCharacter++
			if nextStatement := statements[i+1]; len(nextStatement) > 0 && nextStatement[0] == '\n' {
				endLine++
				endCharacter = 0
			} else {
				endCharacter++
			}
		}
		end := lsp.Position{Line: endLine, Character: endCharacter}
		ranges = append(ranges, lsp.Range{Start: begin, End: end})
	}
	return ranges
}
