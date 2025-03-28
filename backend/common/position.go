package common

import (
	"unicode/utf8"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// ANTLRPosition is a position in a text expressed as one-based line and
// zero-based column character (code point) offset integrated with ANTLR4.
type ANTLRPosition struct {
	// Line position in a text (one-based).
	Line int32
	// Column position in a text (zero-based), equivalent to character offset.
	Column int32
}

// ConvertPositionToANTLRPosition converts a Position to an ANTLRPosition in a given text.
// Returns the end ANTLRPosition if the Position is out of the end of text,
// returns the previous ANTLRPosition if the Position is in the middle of a character,
// returns the previous ANTLRPosition if the Position is out of the end of a line.
func ConvertPositionToANTLRPosition(p *storepb.Position, text string) *ANTLRPosition {
	bs := []byte(text)
	line := int32(0)
	byteOffset := int32(0)
	characterOffset := int32(0)

	for bi := 0; bi < len(bs); {
		if line < p.Line {
			if bs[bi] == '\n' {
				line++
			}
			bi++
			continue
		}

		if byteOffset >= p.Column || bs[bi] == '\n' {
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

	return &ANTLRPosition{
		Line:   line + 1,
		Column: characterOffset,
	}
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
