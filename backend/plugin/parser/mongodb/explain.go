package mongodb

import (
	"github.com/bytebase/omni/mongo/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterExplainStatementFunc(storepb.Engine_MONGODB, explainStatement)
}

// explainStatement implements base.ExplainStatement. MongoDB states a plan as
// one document and has no second format, so the format argument is ignored.
func explainStatement(statement string, _ base.ExplainFormat) (string, error) {
	return analysis.ExplainStatement(statement)
}

// DescribeExplain returns the output format of a statement that asks for its
// own plan, and whether producing that plan runs the statement.
func DescribeExplain(statement string) (format string, executed bool, ok bool) {
	return analysis.DescribeExplain(statement)
}
