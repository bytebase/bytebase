package cassandra

import (
	omnicassandra "github.com/bytebase/omni/cassandra"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterSplitterFunc(storepb.Engine_CASSANDRA, SplitSQL)
}

func SplitSQL(statement string) ([]base.Statement, error) {
	return base.BuildStatements(statement, omnicassandra.Split(statement), func(s omnicassandra.Segment) (int, int, bool) {
		return s.ByteStart, s.ByteEnd, s.Empty
	}, false), nil
}
