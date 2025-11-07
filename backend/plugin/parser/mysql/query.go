package mysql

import (
	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/mysql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterQueryValidator(storepb.Engine_MYSQL, validateQuery)
	base.RegisterQueryValidator(storepb.Engine_MARIADB, validateQuery)
	base.RegisterQueryValidator(storepb.Engine_OCEANBASE, validateQuery)
}

// validateQuery validates the SQL statement for SQL editor.
// We validate the statement by following steps:
// 1. Remove all quoted text(quoted identifier, string literal) and comments from the statement.
// 2. Use regexp to check if the statement is a normal SELECT statement and EXPLAIN statement.
// 3. For CTE, use regexp to check if the statement has UPDATE, DELETE and INSERT statements.
func validateQuery(statement string) (bool, bool, error) {
	trees, err := ParseMySQL(statement)
	if err != nil {
		return false, false, err
	}
	hasExecute := false
	readOnly := true
	for _, item := range trees {
		l := &queryValidateListener{
			valid:      true,
			hasExecute: false,
		}
		antlr.ParseTreeWalkerDefault.Walk(l, item.Tree)
		if !l.valid {
			return false, false, nil
		}
		if l.explainAnalyze {
			readOnly = false
		}
		if l.hasExecute {
			hasExecute = true
		}
	}
	return readOnly, !hasExecute, nil
}

type queryValidateListener struct {
	*parser.BaseMySQLParserListener

	valid          bool
	explainAnalyze bool
	hasExecute     bool
}

// EnterQuery is called when production query is entered.
func (l *queryValidateListener) EnterQuery(ctx *parser.QueryContext) {
	if !l.valid {
		return
	}
	if ctx.BeginWork() != nil {
		l.valid = false
	}
}

// EnterSimpleStatement is called when production simpleStatement is entered.
func (l *queryValidateListener) EnterSimpleStatement(ctx *parser.SimpleStatementContext) {
	if !l.valid {
		return
	}
	if ctx.SetStatement() != nil {
		l.hasExecute = true
	}
	if ctx.SelectStatement() == nil && ctx.UtilityStatement() == nil && ctx.SetStatement() == nil && ctx.ShowStatement() == nil {
		l.valid = false
	}
}

// EnterUtilityStatement is called when production utilityStatement is entered.
func (l *queryValidateListener) EnterUtilityStatement(ctx *parser.UtilityStatementContext) {
	if !l.valid {
		return
	}
	if ctx.ExplainStatement() == nil && ctx.DescribeStatement() == nil {
		l.valid = false
		return
	}
	if ctx.ExplainStatement() != nil {
		if ctx.ExplainStatement().ANALYZE_SYMBOL() != nil {
			l.explainAnalyze = true
			return
		}
	}
}
