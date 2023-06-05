// Package oracle is the advisor for oracle database.
package oracle

import (
	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/plsql-parser"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/db"
)

var (
	_ advisor.Advisor = (*InsertMustSpecifyColumnAdvisor)(nil)
)

func init() {
	advisor.Register(db.Oracle, advisor.OracleInsertMustSpecifyColumn, &InsertMustSpecifyColumnAdvisor{})
}

// InsertMustSpecifyColumnAdvisor is the advisor checking for to enforce column specified.
type InsertMustSpecifyColumnAdvisor struct {
}

// Check checks for to enforce column specified.
func (*InsertMustSpecifyColumnAdvisor) Check(ctx advisor.Context, statement string) ([]advisor.Advice, error) {
	tree, errAdvice := parseStatement(statement)
	if errAdvice != nil {
		return errAdvice, nil
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}

	listener := &insertMustSpecifyColumnListener{
		level:         level,
		title:         string(ctx.Rule.Type),
		currentSchema: ctx.CurrentSchema,
	}

	antlr.ParseTreeWalkerDefault.Walk(listener, tree)

	return listener.generateAdvice()
}

// insertMustSpecifyColumnListener is the listener for to enforce column specified.
type insertMustSpecifyColumnListener struct {
	*parser.BasePlSqlParserListener

	level         advisor.Status
	title         string
	currentSchema string
	adviceList    []advisor.Advice
}

func (l *insertMustSpecifyColumnListener) generateAdvice() ([]advisor.Advice, error) {
	if len(l.adviceList) == 0 {
		l.adviceList = append(l.adviceList, advisor.Advice{
			Status:  advisor.Success,
			Code:    advisor.Ok,
			Title:   "OK",
			Content: "",
		})
	}
	return l.adviceList, nil
}

// EnterInsert_into_clause is called when production insert_into_clause is entered.
func (l *insertMustSpecifyColumnListener) EnterInsert_into_clause(ctx *parser.Insert_into_clauseContext) {
	if ctx.Paren_column_list() == nil {
		l.adviceList = append(l.adviceList, advisor.Advice{
			Status:  l.level,
			Code:    advisor.InsertNotSpecifyColumn,
			Title:   l.title,
			Content: "INSERT statement should specify column name.",
			Line:    ctx.GetStart().GetLine(),
		})
	}
}
