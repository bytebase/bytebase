package base

import (
	"math"
	"unicode/utf8"

	lsp "github.com/bytebase/lsp-protocol"
	omnianalysis "github.com/bytebase/omni/analysis"

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
		return ByteOffsetToRunePosition(m.sql, byteOffset)
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

// ByteOffsetToRunePosition converts a 0-based byte offset in sql to a 1-based line and rune-column Position.
func ByteOffsetToRunePosition(sql string, byteOffset int) *storepb.Position {
	if byteOffset < 0 {
		byteOffset = 0
	}
	if byteOffset > len(sql) {
		byteOffset = len(sql)
	}
	line := int32(1)
	runeCol := int32(0)
	i := 0
	for i < byteOffset {
		r, size := utf8.DecodeRuneInString(sql[i:])
		if i+size > byteOffset {
			break
		}
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

// shiftPosition translates pos relative to basePos (1-based line and column).
func shiftPosition(pos, basePos *storepb.Position) *storepb.Position {
	if pos == nil {
		return nil
	}
	out := &storepb.Position{
		Line:   pos.Line,
		Column: pos.Column,
	}
	if basePos != nil {
		if pos.Line == 1 {
			out.Column += basePos.Column - 1
		}
		out.Line += basePos.Line - 1
	}
	return out
}

// MapByteRangesToLSP converts 0-based byte-offset ranges into LSP ranges (0-based line, UTF-16 code-unit column)
// using O(1) auxiliary memory via a streaming position cursor.
func MapByteRangesToLSP(sql string, byteRanges []omnianalysis.ByteRange) []Range {
	if byteRanges == nil {
		return nil
	}

	mapper := &utf16PositionMapper{sql: sql}
	ranges := make([]Range, 0, len(byteRanges))
	for _, r := range byteRanges {
		if r.Start < 0 || r.End > len(sql) || r.Start > r.End {
			continue
		}
		ranges = append(ranges, Range{
			Start: mapper.position(r.Start),
			End:   mapper.position(r.End),
		})
	}
	return ranges
}

type utf16PositionMapper struct {
	sql        string
	byteOffset int
	line       uint32
	character  uint32
}

func (m *utf16PositionMapper) position(target int) lsp.Position {
	if target < m.byteOffset {
		m.byteOffset = 0
		m.line = 0
		m.character = 0
	}
	for m.byteOffset < target {
		r, size := utf8.DecodeRuneInString(m.sql[m.byteOffset:])
		if m.byteOffset+size > target {
			return lsp.Position{}
		}
		if r == '\n' {
			m.line++
			m.character = 0
		} else if r > 0xFFFF {
			m.character += 2
		} else {
			m.character++
		}
		m.byteOffset += size
	}
	return lsp.Position{Line: m.line, Character: m.character}
}

func safeIntToInt32(v int) int32 {
	if v > math.MaxInt32 {
		return math.MaxInt32
	}
	if v < math.MinInt32 {
		return math.MinInt32
	}
	return int32(v)
}

// LinePosition returns a 1-based Position for line, clamping non-positive lines to 1.
func LinePosition(line int) *storepb.Position {
	return &storepb.Position{
		Line: safeIntToInt32(max(line, 1)),
	}
}
