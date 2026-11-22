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
	return base.BuildStatements(statement, analysis.SplitStatements(statement), func(s analysis.Statement) (int, int, bool) {
		return s.Start, s.End, s.Empty
	}, false), nil
}
