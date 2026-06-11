package starrocks

import (
	"unicode/utf8"

	"github.com/bytebase/omni/starrocks/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterSplitterFunc(storepb.Engine_STARROCKS, SplitSQL)
}

// SplitSQL splits the input into multiple SQL statements using the omni
// Doris splitter, then converts each Segment into a base.Statement with
// the position fields bytebase expects.
//
// Positions are computed in a single O(n) pass over the input.
func SplitSQL(statement string) ([]base.Statement, error) {
	segs := parser.Split(statement)
	if len(segs) == 0 {
		return nil, nil
	}

	type pos struct{ line, col int }
	positions := make(map[int]pos, len(segs)*3)
	for _, seg := range segs {
		positions[seg.ByteStart] = pos{}
		positions[seg.ByteEnd] = pos{}
		// Also collect position immediately past the trailing ';' for End column.
		if seg.ByteEnd < len(statement) && statement[seg.ByteEnd] == ';' {
			positions[seg.ByteEnd+1] = pos{}
		}
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
		// bytebase historically returns Text including the trailing ';' delimiter
		// (when present). omni's Segment.Text excludes it, so we re-attach when
		// the byte after ByteEnd is ';'.
		text := seg.Text
		end := seg.ByteEnd
		if end < len(statement) && statement[end] == ';' {
			text = statement[seg.ByteStart : end+1]
			end++
		}
		sp := positions[seg.ByteStart]
		ep, ok := positions[end]
		if !ok {
			ep = positions[seg.ByteEnd]
		}
		stmts = append(stmts, base.Statement{
			Text:  text,
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
				End:   int32(end),
			},
		})
	}
	return stmts, nil
}
