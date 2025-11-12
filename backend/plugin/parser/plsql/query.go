package plsql

import (
	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/plsql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterQueryValidator(storepb.Engine_ORACLE, validateQuery)
}

// validateQuery validates the SQL statement for SQL editor.
func validateQuery(statement string) (bool, bool, error) {
	results, err := ParsePLSQL(statement)
	if err != nil {
		return false, false, err
	}
	if len(results) == 0 {
		return false, false, nil
	}

	// Validate all statements, not just the first one
	l := &queryValidateListener{
		validate: true,
	}
	for _, result := range results {
		antlr.ParseTreeWalkerDefault.Walk(l, result.Tree)
		if !l.validate {
			return false, false, nil
		}
	}

	return true, true, nil
}

type queryValidateListener struct {
	*parser.BasePlSqlParserListener

	validate bool
}

// EnterSql_script is called when production sql_script is entered.
func (l *queryValidateListener) EnterSql_script(ctx *parser.Sql_scriptContext) {
	if len(ctx.AllSql_plus_command()) > 0 {
		l.validate = false
	}
}

// EnterUnit_statement is called when production unit_statement is entered.
func (l *queryValidateListener) EnterUnit_statement(ctx *parser.Unit_statementContext) {
	if ctx.Data_manipulation_language_statements() == nil {
		l.validate = false
	}
}

// EnterData_manipulation_language_statements is called when production data_manipulation_language_statements is entered.
func (l *queryValidateListener) EnterData_manipulation_language_statements(ctx *parser.Data_manipulation_language_statementsContext) {
	if ctx.Select_statement() == nil && ctx.Explain_statement() == nil {
		l.validate = false
	}
}
