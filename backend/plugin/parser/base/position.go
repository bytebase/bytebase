package base

import (
	"unicode/utf8"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// ByteOffsetPositionMapper converts monotonically increasing byte offsets in a
// single SQL string to 1-based line:column positions in one pass.
type ByteOffsetPositionMapper struct {
	sql        string
	byteOffset int
	line       int32
	runeCol    int32
}

// NewByteOffsetPositionMapper creates a mapper for byte offsets in sql.
func NewByteOffsetPositionMapper(sql string) *ByteOffsetPositionMapper {
	return &ByteOffsetPositionMapper{
		sql:  sql,
		line: 1,
	}
}

// Position returns the 1-based line:column position for byteOffset.
func (m *ByteOffsetPositionMapper) Position(byteOffset int) *storepb.Position {
	if byteOffset < 0 {
		byteOffset = 0
	}
	if byteOffset > len(m.sql) {
		byteOffset = len(m.sql)
	}
	if byteOffset < m.byteOffset {
		return byteOffsetToRunePosition(m.sql, byteOffset)
	}

	for m.byteOffset < byteOffset {
		r, size := utf8.DecodeRuneInString(m.sql[m.byteOffset:])
		switch r {
		case '\r':
			m.line++
			m.runeCol = 0
		case '\n':
			if m.byteOffset == 0 || m.sql[m.byteOffset-1] != '\r' {
				m.line++
				m.runeCol = 0
			}
		default:
			m.runeCol++
		}
		m.byteOffset += size
	}

	return &storepb.Position{
		Line:   m.line,
		Column: m.runeCol + 1,
	}
}

func byteOffsetToRunePosition(sql string, byteOffset int) *storepb.Position {
	line := int32(1)
	runeCol := int32(0)
	i := 0
	for i < byteOffset {
		r, size := utf8.DecodeRuneInString(sql[i:])
		switch r {
		case '\r':
			line++
			runeCol = 0
		case '\n':
			if i == 0 || sql[i-1] != '\r' {
				line++
				runeCol = 0
			}
		default:
			runeCol++
		}
		i += size
	}

	return &storepb.Position{
		Line:   line,
		Column: runeCol + 1,
	}
}
