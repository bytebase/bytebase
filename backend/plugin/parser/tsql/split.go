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
	return base.BuildStatements(statement, omnimssql.Split(statement), func(s omnimssql.Segment) (int, int, bool) {
		return s.ByteStart, s.ByteEnd, s.Empty
	}, false), nil
}
