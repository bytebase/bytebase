package tidb

import (
	"github.com/bytebase/omni/tidb/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterExplainStatementFunc(storepb.Engine_TIDB, explainStatement)
}

// explainStatement implements base.ExplainStatement. Only the JSON format has
// a reader downstream; any other format gets the engine's default plan.
func explainStatement(statement string, format base.ExplainFormat) (string, error) {
	explain, err := analysis.ExplainStatement(statement, format == base.ExplainFormatJSON)
	return base.ExplainStatementResult(explain, err, func(e error) error {
		return convertParseError(e, base.Statement{Text: statement})
	})
}

// DescribeExplain returns the output format of an EXPLAIN statement and whether
// it executes the statement it plans.
func DescribeExplain(statement string) (format string, executed bool, ok bool) {
	return analysis.DescribeExplain(statement)
}
