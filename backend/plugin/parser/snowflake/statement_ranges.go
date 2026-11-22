package snowflake

import (
	"context"
	"unicode/utf8"

	protocol "github.com/bytebase/lsp-protocol"
	"github.com/bytebase/omni/snowflake/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterStatementRangesFunc(storepb.Engine_SNOWFLAKE, GetStatementRanges)
}

// GetStatementRanges returns the UTF-16 line/character ranges of statements in
// the input, formatted for the LSP protocol.
func GetStatementRanges(_ context.Context, _ base.StatementRangeContext, statement string) ([]base.Range, error) {
	byteRanges := analysis.StatementRanges(statement)
	if byteRanges == nil {
		return nil, nil
	}
	mapper := newUTF16Mapper(statement)
	ranges := make([]base.Range, 0, len(byteRanges))
	for _, r := range byteRanges {
		ranges = append(ranges, base.Range{
			Start: mapper.position(r.Start),
			End:   mapper.position(r.End),
		})
	}
	return ranges, nil
}

// utf16Mapper maps byte offsets in a source string to UTF-16 LSP positions
// (0-based line, 0-based UTF-16 code-unit column) in a single O(n) precompute.
type utf16Mapper struct {
	positions []protocol.Position
}

func newUTF16Mapper(s string) *utf16Mapper {
	positions := make([]protocol.Position, len(s)+1)
	var line, character uint32
	for i := 0; i < len(s); {
		positions[i] = protocol.Position{Line: line, Character: character}
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == '\n' {
			line++
			character = 0
		} else if r > 0xFFFF {
			character += 2 // UTF-16 surrogate pair
		} else {
			character++
		}
		i += size
	}
	positions[len(s)] = protocol.Position{Line: line, Character: character}
	return &utf16Mapper{positions: positions}
}

// position returns the UTF-16 position at byteOffset. byteOffset is clamped to
// [0, len(s)]; out-of-range offsets resolve to the nearest boundary.
func (m *utf16Mapper) position(byteOffset int) protocol.Position {
	if byteOffset < 0 {
		byteOffset = 0
	}
	if byteOffset >= len(m.positions) {
		byteOffset = len(m.positions) - 1
	}
	return m.positions[byteOffset]
}
