package doris

import (
	"unicode/utf8"

	"github.com/bytebase/omni/doris/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterSplitterFunc(storepb.Engine_DORIS, SplitSQL)
}

// SplitSQL splits the input into multiple SQL statements using omni's analysis
// package, then attaches 1-based line and rune column positions.
func SplitSQL(statement string) ([]base.Statement, error) {
	segs := analysis.SplitStatements(statement)
	if len(segs) == 0 {
		return nil, nil
	}

	type pos struct{ line, col int }
	positions := make(map[int]pos, len(segs)*2)
	for _, seg := range segs {
		positions[seg.Start] = pos{}
		positions[seg.End] = pos{}
	}

	line, col := 1, 1
	for i := 0; i <= len(statement); {
		if _, need := positions[i]; need {
			positions[i] = pos{line, col}
		}
		if i == len(statement) {
			break
		}
		r, size := utf8.DecodeRuneInString(statement[i:])
		if r == '\n' {
			line++
			col = 1
		} else {
			col++
		}
		i += size
	}

	stmts := make([]base.Statement, 0, len(segs))
	for _, seg := range segs {
		sp := positions[seg.Start]
		ep := positions[seg.End]
		stmts = append(stmts, base.Statement{
			Text:  seg.Text,
			Empty: seg.Empty,
			Start: &storepb.Position{
				Line:   int32(sp.line),
				Column: int32(sp.col),
			},
			End: &storepb.Position{
				Line:   int32(ep.line),
				Column: int32(ep.col),
			},
			Range: &storepb.Range{
				Start: int32(seg.Start),
				End:   int32(seg.End),
			},
		})
	}
	return stmts, nil
}
