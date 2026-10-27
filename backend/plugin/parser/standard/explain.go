package standard

import (
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterExplainStatementFunc(storepb.Engine_CLICKHOUSE, clickhouseExplainStatement)
	base.RegisterExplainStatementFunc(storepb.Engine_DATABRICKS, databricksExplainStatement)
	base.RegisterExplainStatementFunc(storepb.Engine_HIVE, hiveExplainStatement)
}

// hiveExplainStatement asks Hive for the structured plan the query-plan viewer
// draws. FORMATTED answers with one JSON object, whose CBOPlan is the plan
// Calcite built; every other format of Hive's EXPLAIN is a tree of lines.
//
// With no AST to ask which statement an EXPLAIN plans, a statement that already
// asks for a plan runs as the user wrote it; that keeps a request from planning
// a plan, which is what the rest of this package can promise.
func hiveExplainStatement(statement string, format base.ExplainFormat) (string, error) {
	if base.StartsWithExplain(statement) {
		return statement, nil
	}
	if format == base.ExplainFormatJSON {
		return "EXPLAIN FORMATTED " + statement, nil
	}
	return "EXPLAIN " + statement, nil
}

// databricksExplainStatement asks for Spark's FORMATTED plan, which is a tree
// of numbered operators followed by a block of detail for each, where a plain
// EXPLAIN prints one tree with every operator's arguments inline. Databricks
// SQL is Spark SQL, and FORMATTED has been there since Spark 3.0.
//
// Databricks prints this one plan whatever the caller asks for. A statement
// that already asks for a plan runs as the user wrote it, keeping the mode it
// named; EXPLAIN never executes what it plans, so the planned form of a write
// is a plan of that write.
func databricksExplainStatement(statement string, _ base.ExplainFormat) (string, error) {
	if base.StartsWithExplain(statement) {
		return statement, nil
	}
	return "EXPLAIN FORMATTED " + statement, nil
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
