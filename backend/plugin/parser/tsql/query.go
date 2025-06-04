package tsql

import (
	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/tsql-parser"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	base.RegisterQueryValidator(storepb.Engine_MSSQL, ValidateSQLForEditor)
}

func ValidateSQLForEditor(statement string) (bool, bool, error) {
	parseResult, err := ParseTSQL(statement)
	if err != nil {
		return false, false, err
	}
	if parseResult == nil {
		return false, false, nil
	}

	l := &queryValidateListener{
		valid: true,
	}
	antlr.ParseTreeWalkerDefault.Walk(l, parseResult.Tree)
	return l.valid, l.valid, nil
}

type queryValidateListener struct {
	*parser.BaseTSqlParserListener

	valid bool
}

func (q *queryValidateListener) EnterBatch_without_go(ctx *parser.Batch_without_goContext) {
	if !q.valid {
		return
	}
	if ctx.Batch_level_statement() != nil {
		q.valid = false
		return
	}
}

func (q *queryValidateListener) EnterSql_clauses(ctx *parser.Sql_clausesContext) {
	if !q.valid {
		return
	}
	if ctx.Dml_clause() == nil {
		q.valid = false
		return
	}
}

func (q *queryValidateListener) EnterDml_clause(ctx *parser.Dml_clauseContext) {
	if !q.valid {
		return
	}
	_, ok := ctx.GetParent().(*parser.Sql_clausesContext)
	if !ok {
		return
	}
	if ctx.Select_statement_standalone() == nil {
		q.valid = false
		return
	}
}

func (q *queryValidateListener) EnterSelect_statement_standalone(ctx *parser.Select_statement_standaloneContext) {
	if !q.valid {
		return
	}
	_, ok := ctx.GetParent().(*parser.Dml_clauseContext)
	if !ok {
		return
	}
	if ctx.Select_statement() == nil {
		q.valid = false
		return
	}
}

func (q *queryValidateListener) EnterQuery_specification(ctx *parser.Query_specificationContext) {
	if !q.valid {
		return
	}
	if ctx.INTO() != nil {
		// For Into clause, we only select into temporary table, likes "SELECT ... INTO #temp FROM ...".
		isValid := false
		// NOTE: normal mode is not in single session mode, so temporary table is meaningless.
		// if tableName := ctx.Table_name(); tableName != nil {
		// 	if allID := tableName.AllId_(); len(allID) == 1 {
		// 		if id := allID[0].TEMP_ID(); id != nil {
		// 			isValid = true
		// 		}
		// 	}
		// }
		q.valid = isValid
		return
	}
}
