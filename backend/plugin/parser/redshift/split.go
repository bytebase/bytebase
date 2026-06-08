package redshift

import (
	omniredshift "github.com/bytebase/omni/redshift"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterSplitterFunc(storepb.Engine_REDSHIFT, SplitSQL)
}

// SplitSQL splits the given SQL statement into multiple SQL statements.
func SplitSQL(statement string) ([]base.Statement, error) {
	segments := omniredshift.Split(statement)
	list := make([]base.Statement, 0, len(segments))
	for _, segment := range segments {
		startLine, startColumn := base.CalculateLineAndColumn(statement, segment.ByteStart)
		endLine, endColumn := base.CalculateLineAndColumn(statement, segment.ByteEnd)
		list = append(list, base.Statement{
			Text:  segment.Text,
			Empty: segment.Empty(),
			Start: &storepb.Position{
				Line:   int32(startLine + 1),
				Column: int32(startColumn + 1),
			},
			End: &storepb.Position{
				Line:   int32(endLine + 1),
				Column: int32(endColumn + 1),
			},
			Range: &storepb.Range{
				Start: int32(segment.ByteStart),
				End:   int32(segment.ByteEnd),
			},
		})
	}
	return list, nil
}
