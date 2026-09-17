package standard

import (
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterExplainStatementFunc(storepb.Engine_CLICKHOUSE, explainStatement)
	base.RegisterExplainStatementFunc(storepb.Engine_HIVE, explainStatement)
}

// explainStatement implements base.ExplainStatement for the engines parsed only
// as text here. With no AST to ask which statement an EXPLAIN plans, a statement
// that already asks for a plan runs as the user wrote it; that keeps a request
// from planning a plan, which is what the rest of this package can promise.
// Both engines return one plan format whatever the caller asked for
// (supportedExplainFormats in the API offers them the default plan only).
func explainStatement(statement string, _ base.ExplainFormat) (string, error) {
	if base.StartsWithExplain(statement) {
		return statement, nil
	}
	return "EXPLAIN " + statement, nil
}
