package partiql

import (
	"unicode/utf8"

	"github.com/bytebase/omni/partiql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterSplitterFunc(storepb.Engine_DYNAMODB, SplitSQL)
}

// SplitSQL splits the input into multiple SQL statements using the omni
// splitter, then converts each Segment into a base.Statement with the
// position fields that bytebase expects.
//
// Positions are computed in a single O(n) pass over the input, not by
// rescanning from byte 0 for each segment.
func SplitSQL(statement string) ([]base.Statement, error) {
	segs := partiql.Split(statement)
	if len(segs) == 0 {
		return nil, nil
	}

	// Collect all boundary offsets we need positions for.
	type pos struct{ line, col int }
	positions := make(map[int]pos, len(segs)*2)
	for _, seg := range segs {
		positions[seg.ByteStart] = pos{}
		positions[seg.ByteEnd] = pos{}
	}

	// Single O(n) scan: track line/col and record at each boundary offset.
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

	// Build result from pre-computed positions.
	stmts := make([]base.Statement, 0, len(segs))
	for _, seg := range segs {
		sp := positions[seg.ByteStart]
		ep := positions[seg.ByteEnd]
		stmts = append(stmts, base.Statement{
			Text:  seg.Text,
			Empty: seg.Empty(),
			Start: &storepb.Position{
				Line:   int32(sp.line),
				Column: int32(sp.col),
			},
			End: &storepb.Position{
				Line:   int32(ep.line),
				Column: int32(ep.col),
			},
			Range: &storepb.Range{
				Start: int32(seg.ByteStart),
				End:   int32(seg.ByteEnd),
			},
		})
	}
	return stmts, nil
}
