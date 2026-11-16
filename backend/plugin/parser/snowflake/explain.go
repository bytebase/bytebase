package snowflake

import (
	"github.com/bytebase/omni/snowflake/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterExplainStatementFunc(storepb.Engine_SNOWFLAKE, explainStatement)
}

// explainStatement implements base.ExplainStatement. Snowflake spells its
// structured plan EXPLAIN USING JSON; any other format gets the tabular plan.
func explainStatement(statement string, format base.ExplainFormat) (string, error) {
	return analysis.ExplainStatement(statement, format == base.ExplainFormatJSON)
}

// DescribeExplain returns the output format of an EXPLAIN statement and whether
// it executes the statement it plans.
func DescribeExplain(statement string) (format string, executed bool, ok bool) {
	return analysis.DescribeExplain(statement)
}
