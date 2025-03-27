package common

import "unicode/utf8"

// Position in a text expressed as zero-based line and zero-based column byte offset.
type Position struct {
	// Line position in a text (zero-based).
	Line int
	// Column position in a text (zero-based), equivalent to byte offset.
	Column int
}

// ANTLRPosition is a position in a text expressed as one-based line and
// zero-based column character (code point) offset integrated with ANTLR4.
type ANTLRPosition struct {
	// Line position in a text (one-based).
	Line int
	// Column position in a text (zero-based), equivalent to character offset.
	Column int
}

// ToANTLRPosition converts a Position to an ANTLRPosition in a given text.
// Returns the end ANTLRPosition if the Position is out of the end of text,
// returns the previous ANTLRPosition if the Position is in the middle of a character,
// returns the previous ANTLRPosition if the Position is out of the end of a line.
func (p Position) ToANTLRPosition(text string) ANTLRPosition {
	bt := []byte(text)
	line := 0
	byteOffset := 0
	characterOffset := 0
	for bi := 0; bi < len(bt); {
		if line < p.Line {
			if bt[bi] == '\n' {
				line++
			}
			bi++
			continue
		}

		if byteOffset >= p.Column || bt[bi] == '\n' {
			break
		}

		r, sz := utf8.DecodeRune(bt[bi:])
		if r == utf8.RuneError {
			break
		}
		// Avoid endless loop.
		if sz == 0 {
			break
		}

		byteOffset += sz
		bi += sz
		characterOffset += 1
	}

	return ANTLRPosition{
		Line:   line + 1,
		Column: characterOffset,
	}
}

// ToPosition converts an ANTLRPosition to a Position in a given text.
// Returns the end Position if the ANTLRPosition is out of the end of text,
// returns the previous Position if the ANTLRPosition is in the middle of a character,
// returns the previous Position if the ANTLRPosition is out of the end of a line.
func (a ANTLRPosition) ToPosition(text string) Position {
	bt := []byte(text)
	line := 0
	byteOffset := 0
	characterOffset := 0
	for bi := 0; bi < len(bt); {
		if line < a.Line-1 {
			if bt[bi] == '\n' {
				line++
			}
			bi++
			continue
		}

		if characterOffset >= a.Column || bt[bi] == '\n' {
			break
		}

		r, sz := utf8.DecodeRune(bt[bi:])
		if r == utf8.RuneError {
			break
		}
		// Avoid endless loop.
		if sz == 0 {
			break
		}

		byteOffset += sz
		bi += sz
		characterOffset += 1
	}

	return Position{
		Line:   line,
		Column: byteOffset,
	}
}
