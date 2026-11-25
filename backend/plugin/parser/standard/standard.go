package standard

import (
	chanalysis "github.com/bytebase/omni/clickhouse/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.Register(base.Engine{
		Split:         SplitSQL,
		ValidateQuery: chanalysis.ValidateQuery,
		Explain:       clickhouseExplainStatement,
		ResultLimit:   statementWithResultLimit,
	}, storepb.Engine_CLICKHOUSE)
}

// SplitSQL splits the given SQL statement into multiple SQL statements.
func SplitSQL(statement string) ([]base.Statement, error) {
	stmts, err := chanalysis.Split(statement)
	if err != nil {
		return nil, err
	}
	return base.OmniStatements(statement, stmts), nil
}

func clickhouseExplainStatement(statement string, format base.ExplainFormat) (string, error) {
	formatName := ""
	if format == base.ExplainFormatJSON {
		formatName = "JSON"
	}
	return chanalysis.ExplainStatement(statement, formatName), nil
}

// statementWithResultLimit implements base.ResultLimitFunc for ClickHouse,
// which takes no engineVersion.
func statementWithResultLimit(statement string, limit int, _ string) string {
	return chanalysis.WrapResultLimit(statement, limit)
}
