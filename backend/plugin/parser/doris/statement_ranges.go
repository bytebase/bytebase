package doris

import (
	"context"
	"unicode/utf8"

	lsp "github.com/bytebase/lsp-protocol"
	"github.com/bytebase/omni/doris/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterStatementRangesFunc(storepb.Engine_DORIS, GetStatementRanges)
}

func GetStatementRanges(_ context.Context, _ base.StatementRangeContext, statement string) ([]base.Range, error) {
	rawRanges := analysis.StatementRanges(statement)
	if len(rawRanges) == 0 {
		return nil, nil
	}
	mapper := newUTF16PositionMapper(statement)
	ranges := make([]base.Range, 0, len(rawRanges))
	for _, r := range rawRanges {
		if r.Start < 0 || r.End > len(statement) || r.Start > r.End {
			continue
		}
		ranges = append(ranges, base.Range{
			Start: mapper.position(r.Start),
			End:   mapper.position(r.End),
		})
	}
	return ranges, nil
}

// utf16PositionMapper converts monotonically non-decreasing byte offsets to
// 0-based UTF-16 LSP positions in a single pass.
type utf16PositionMapper struct {
	sql        string
	byteOffset int
	line       uint32
	char       uint32
}

func newUTF16PositionMapper(sql string) *utf16PositionMapper {
	return &utf16PositionMapper{sql: sql}
}

func (m *utf16PositionMapper) position(byteOffset int) lsp.Position {
	for m.byteOffset < byteOffset {
		r, size := utf8.DecodeRuneInString(m.sql[m.byteOffset:])
		if r == '\n' {
			m.line++
			m.char = 0
		} else if r > 0xFFFF {
			m.char += 2
		} else {
			m.char++
		}
		m.byteOffset += size
	}
	return lsp.Position{Line: m.line, Character: m.char}
}
