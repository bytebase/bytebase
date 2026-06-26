package tidb

import (
	tidbparser "github.com/bytebase/omni/tidb/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterSplitterFunc(storepb.Engine_TIDB, SplitSQL)
}

// SplitSQL splits the given SQL statement into multiple SQL statements.
func SplitSQL(statement string) ([]base.Statement, error) {
	segments := tidbparser.Split(statement)

	result := make([]base.Statement, 0, len(segments))
	positionMapper := base.NewByteOffsetPositionMapper(statement)
	for _, seg := range segments {
		result = append(result, base.NewStatementFromRange(statement, positionMapper, seg.ByteStart, seg.ByteEnd, seg.Empty()))
	}
	return result, nil
}
