package tsql

import (
	omnimssql "github.com/bytebase/omni/mssql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterSplitterFunc(storepb.Engine_MSSQL, SplitSQL)
}

// SplitSQL splits the given T-SQL script into contiguous statements via omni.
func SplitSQL(statement string) ([]base.Statement, error) {
	segs := omnimssql.Split(statement)
	if len(segs) == 0 {
		return nil, nil
	}
	result := make([]base.Statement, len(segs))
	for i, seg := range segs {
		result[i] = base.Statement{
			Text: seg.Text,
			Range: &storepb.Range{
				Start: int32(seg.ByteStart),
				End:   int32(seg.ByteEnd),
			},
			Start: &storepb.Position{
				Line:   int32(seg.Start.Line),
				Column: int32(seg.Start.Column),
			},
			End: &storepb.Position{
				Line:   int32(seg.End.Line),
				Column: int32(seg.End.Column),
			},
			Empty: seg.Empty,
		}
	}
	return result, nil
}
