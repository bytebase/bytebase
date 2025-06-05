package plsql

import (
	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/plsql-parser"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	base.RegisterQueryValidator(storepb.Engine_ORACLE, validateQuery)
	base.RegisterQueryValidator(storepb.Engine_DM, validateQuery)
	base.RegisterQueryValidator(storepb.Engine_OCEANBASE_ORACLE, validateQuery)
}

// validateQuery validates the SQL statement for SQL editor.
func validateQuery(statement string) (bool, bool, error) {
	tree, _, err := ParsePLSQL(statement)
	if err != nil {
		return false, false, err
	}
	l := &queryValidateListener{
		validate: true,
	}
	antlr.ParseTreeWalkerDefault.Walk(l, tree)
	if !l.validate {
		return false, false, nil
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
