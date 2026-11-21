package partiql

import (
	"github.com/bytebase/omni/partiql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterSplitterFunc(storepb.Engine_DYNAMODB, SplitSQL)
}

// SplitSQL splits the input into multiple SQL statements using the omni splitter.
func SplitSQL(statement string) ([]base.Statement, error) {
	return base.BuildStatements(statement, partiql.Split(statement), func(s partiql.Segment) (int, int, bool) {
		return s.ByteStart, s.ByteEnd, s.Empty()
	}, false), nil
}
