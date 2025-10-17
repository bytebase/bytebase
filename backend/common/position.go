//nolint:revive
package common

import (
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
// ANTLRPosition uses 1-based line and 0-based character column.
// Returns a Position with 1-based line and 1-based character column.
// Returns the end Position if the ANTLRPosition is out of the end of text,
// returns the previous Position if the ANTLRPosition is out of the end of a line.
func ConvertANTLRPositionToPosition(a *ANTLRPosition, text string) *storepb.Position {
	runes := []rune(text)
	line := int32(1) // Start at 1 for 1-based line numbering
	charOffsetInLine := int32(0)
	globalCharOffset := int32(0)

	for globalCharOffset < int32(len(runes)) {
		// Skip lines before the target line
		if line < a.Line {
			if runes[globalCharOffset] == '\n' {
				line++
			}
			globalCharOffset++
			continue
		}

		// Stop if we've reached the target column or hit a newline
		if charOffsetInLine >= a.Column || runes[globalCharOffset] == '\n' {
			break
		}

		charOffsetInLine++
		globalCharOffset++
	}

	// Convert from 0-based to 1-based column
	return &storepb.Position{
		Line:   line,
		Column: charOffsetInLine + 1,
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

func ConvertTiDBParserErrorPositionToPosition(line, column int) *storepb.Position {
	if line < 1 {
		line = 1
	}
	if column < 1 {
		column = 1
	}

	// TiDB parser provides 1-based line and column
	// Store them as 1-based in Position
	return &storepb.Position{
		Line:   int32(line),
		Column: int32(column),
	}
}

func ConvertPGParserErrorCursorPosToPosition(cursorPos int, text string) *storepb.Position {
	// PostgreSQL cursorPos is 1-based character (rune) position.
	// Cursorpos points to the character WHERE the error is.
	// We need to count characters BEFORE the error position.
	if cursorPos >= 1 {
		cursorPos--
	}
	line := 1 // Start at 1 for 1-based line numbering
	column := 0
	rText := []rune(text)
	for i, r := range rText {
		// Stop when we reach the error position
		if i == cursorPos {
			break
		}
		if r == '\n' {
			line++
			column = 0
			continue
		}
		// Count characters (not bytes)
		column++
	}
	// Convert from 0-based to 1-based column
	return &storepb.Position{
		Line:   int32(line),
		Column: int32(column + 1),
	}
}
