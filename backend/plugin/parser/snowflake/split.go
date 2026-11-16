package snowflake

import (
	"github.com/bytebase/omni/snowflake/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterSplitterFunc(storepb.Engine_SNOWFLAKE, SplitSQL)
}

// SplitSQL splits the input into statements, each with its trailing ';'.
func SplitSQL(statement string) ([]base.Statement, error) {
	segs := analysis.SplitStatements(statement)
	if len(segs) == 0 {
		return nil, nil
	}

	// Segment offsets increase, which the mapper's single pass relies on.
	mapper := base.NewByteOffsetPositionMapper(statement)
	stmts := make([]base.Statement, 0, len(segs))
	for _, seg := range segs {
		stmts = append(stmts, base.Statement{
			Text:  seg.Text,
			Empty: seg.Empty,
			Start: mapper.Position(seg.Start),
			End:   mapper.Position(seg.End),
			Range: &storepb.Range{
				Start: int32(seg.Start),
				End:   int32(seg.End),
			},
		})
	}
	return stmts, nil
}
