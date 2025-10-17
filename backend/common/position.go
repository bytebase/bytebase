//nolint:revive
package common

import (
	"unicode/utf8"

	lsp "github.com/bytebase/lsp-protocol"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// ANTLRPosition is a position in a text expressed as one-based line and
// zero-based column character (code point) offset integrated with ANTLR4.
type ANTLRPosition struct {
	// Line position in a text (one-based).
	Line int32
	// Column position in a text (zero-based), equivalent to character offset.
	Column int32
}

// ConvertANTLRPositionToPosition converts an ANTLRPosition to a Position in a given text.
// Returns the end Position if the ANTLRPosition is out of the end of text,
// returns the previous Position if the ANTLRPosition is in the middle of a character,
// returns the previous Position if the ANTLRPosition is out of the end of a line.
func ConvertANTLRPositionToPosition(a *ANTLRPosition, text string) *storepb.Position {
	bs := []byte(text)
	line := int32(0)
	byteOffset := int32(0)
	characterOffset := int32(0)

	for bi := 0; bi < len(bs); {
		if line < a.Line-1 {
			if bs[bi] == '\n' {
				line++
			}
			bi++
			continue
		}

		if characterOffset >= a.Column || bs[bi] == '\n' {
			break
		}

		r, sz := utf8.DecodeRune(bs[bi:])
		if r == utf8.RuneError {
			break
		}
		// Avoid endless loop.
		if sz == 0 {
			break
		}

		byteOffset += int32(sz)
		bi += sz
		characterOffset++
	}

	return &storepb.Position{
		Line:   line,
		Column: byteOffset,
	}
}

func ConvertANTLRLineToPosition(line int) *storepb.Position {
	positionLine := line - 1
	if line == 0 {
		positionLine = 0
	}
	return &storepb.Position{
		Line: int32(positionLine),
	}
}

// UTF16Position is a position in a text expressed as zero-based line and zero-based column counted in UTF-16 code units.
type UTF16Position = lsp.Position

func ConvertTiDBParserErrorPositionToPosition(line, column int) *storepb.Position {
	if line < 1 {
		line = 1
	}
	if column < 1 {
		column = 1
	}

	return &storepb.Position{
		Line:   int32(line) - 1,
		Column: int32(column) - 1,
	}
}

func ConvertPGParserErrorCursorPosToPosition(cursorPos int, text string) *storepb.Position {
	if cursorPos >= 1 {
		cursorPos--
	}
	line := 0
	column := 0
	rText := []rune(text)
	for i, r := range rText {
		if i >= cursorPos {
			break
		}
		if r == '\n' {
			line++
			column = 0
			continue
		}
		column += len(string(r))
	}
	return &storepb.Position{
		Line:   int32(line),
		Column: int32(column),
	}
}
