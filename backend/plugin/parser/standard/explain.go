package standard

import (
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterExplainStatementFunc(storepb.Engine_CLICKHOUSE, clickhouseExplainStatement)
	base.RegisterExplainStatementFunc(storepb.Engine_HIVE, explainStatement)
}

// explainStatement implements base.ExplainStatement for the engines parsed only
// as text here. With no AST to ask which statement an EXPLAIN plans, a statement
// that already asks for a plan runs as the user wrote it; that keeps a request
// from planning a plan, which is what the rest of this package can promise.
// Hive returns one plan format whatever the caller asked for; the driver
// capability table offers it the default plan only.
func explainStatement(statement string, _ base.ExplainFormat) (string, error) {
	if base.StartsWithExplain(statement) {
		return statement, nil
	}
	return "EXPLAIN " + statement, nil
}

// clickhouseExplainStatement asks ClickHouse for the structured plan the
// query-plan viewer draws. `indexes = 1` is what makes each read report how
// many parts and granules its indexes let it skip, which is what a ClickHouse
// plan is read for; without it the plan is shape alone.
//
// A statement that already asks for a plan still runs as the user wrote it, so
// a typed EXPLAIN keeps whichever of ClickHouse's plan kinds it named rather
// than being rewritten by a parser that cannot read it.
func clickhouseExplainStatement(statement string, format base.ExplainFormat) (string, error) {
	if base.StartsWithExplain(statement) {
		return statement, nil
	}
	if format == base.ExplainFormatJSON {
		return "EXPLAIN json = 1, indexes = 1 " + statement, nil
	}
	return "EXPLAIN " + statement, nil
}
