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
	return base.BuildStatements(statement, tidbparser.Split(statement), func(s tidbparser.Segment) (int, int, bool) {
		return s.ByteStart, s.ByteEnd, s.Empty()
	}, true), nil
}
