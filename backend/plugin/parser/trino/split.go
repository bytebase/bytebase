package trino

import (
	"github.com/bytebase/omni/trino/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterSplitterFunc(storepb.Engine_TRINO, SplitSQL)
}

// SplitSQL splits the input into multiple SQL statements using the omni Trino
// splitter, then converts each Segment into a base.Statement with the position
// fields bytebase expects. Positions are computed in a single O(n) pass via the
// shared base.ByteOffsetPositionMapper (segment offsets are queried in
// increasing order).
func SplitSQL(statement string) ([]base.Statement, error) {
	segs := parser.Split(statement)
	if len(segs) == 0 {
		return nil, nil
	}

	mapper := base.NewByteOffsetPositionMapper(statement)
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
		stmts = append(stmts, base.Statement{
			Text:  text,
			Empty: seg.Empty(),
			Start: mapper.Position(seg.ByteStart),
			End:   mapper.Position(end),
			Range: &storepb.Range{
				Start: int32(seg.ByteStart),
				End:   int32(end),
			},
		})
	}
	return stmts, nil
}
