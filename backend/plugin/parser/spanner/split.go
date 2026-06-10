package spanner

import (
	"github.com/bytebase/omni/googlesql/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterSplitterFunc(storepb.Engine_SPANNER, SplitSQL)
}

// SplitSQL splits the input into multiple SQL statements using the omni
// GoogleSQL splitter (block-aware: BEGIN/END, IF, LOOP, CASE bodies stay whole),
// then converts each Segment into a base.Statement with the position fields the
// legacy parse-tree splitter produced: each statement's Text runs CONTIGUOUSLY
// from the end of the previous statement (so inter-statement whitespace and
// comments attach to the FOLLOWING statement) through its own trailing ';'.
// Positions are computed in a single O(n) pass via the shared
// base.ByteOffsetPositionMapper (offsets are queried in increasing order).
func SplitSQL(statement string) ([]base.Statement, error) {
	segs := parser.Split(statement)
	if len(segs) == 0 {
		return nil, nil
	}

	mapper := base.NewByteOffsetPositionMapper(statement)
	stmts := make([]base.Statement, 0, len(segs))
	prevEnd := 0
	for _, seg := range segs {
		// omni's Segment.Text excludes the trailing ';' (ByteEnd points AT it);
		// the legacy spanner splitter included it, so re-attach when present.
		end := seg.ByteEnd
		if end < len(statement) && statement[end] == ';' {
			end++
		}
		stmts = append(stmts, base.Statement{
			Text:  statement[prevEnd:end],
			Empty: seg.Empty(),
			Start: mapper.Position(prevEnd),
			End:   mapper.Position(end),
			Range: &storepb.Range{
				Start: int32(prevEnd),
				End:   int32(end),
			},
		})
		prevEnd = end
	}
	return stmts, nil
}
