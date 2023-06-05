// Package oracle is the advisor for oracle database.
package oracle

import (
	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/plsql-parser"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/db"
)

var (
	_ advisor.Advisor = (*WhereRequireAdvisor)(nil)
)

func init() {
	advisor.Register(db.Oracle, advisor.OracleWhereRequirement, &WhereRequireAdvisor{})
}

// WhereRequireAdvisor is the advisor checking for WHERE clause requirement.
type WhereRequireAdvisor struct {
}

// Check checks for WHERE clause requirement.
func (*WhereRequireAdvisor) Check(ctx advisor.Context, statement string) ([]advisor.Advice, error) {
	tree, errAdvice := parseStatement(statement)
	if errAdvice != nil {
		return errAdvice, nil
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}

	listener := &whereRequireListener{
		level:         level,
		title:         string(ctx.Rule.Type),
		currentSchema: ctx.CurrentSchema,
	}

	antlr.ParseTreeWalkerDefault.Walk(listener, tree)

	return listener.generateAdvice()
}

// whereRequireListener is the listener for WHERE clause requirement.
type whereRequireListener struct {
	*parser.BasePlSqlParserListener

	level         advisor.Status
	title         string
	currentSchema string
	adviceList    []advisor.Advice
}

func (l *whereRequireListener) generateAdvice() ([]advisor.Advice, error) {
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

// EnterUpdate_statement is called when production update_statement is entered.
func (l *whereRequireListener) EnterUpdate_statement(ctx *parser.Update_statementContext) {
	if ctx.Where_clause() == nil {
		l.adviceList = append(l.adviceList, advisor.Advice{
			Status:  l.level,
			Code:    advisor.StatementNoWhere,
			Title:   l.title,
			Content: "WHERE clause is required for UPDATE statement.",
			Line:    ctx.GetStop().GetLine(),
		})
	}
}

// EnterDelete_statement is called when production delete_statement is entered.
func (l *whereRequireListener) EnterDelete_statement(ctx *parser.Delete_statementContext) {
	if ctx.Where_clause() == nil {
		l.adviceList = append(l.adviceList, advisor.Advice{
			Status:  l.level,
			Code:    advisor.StatementNoWhere,
			Title:   l.title,
			Content: "WHERE clause is required for DELETE statement.",
			Line:    ctx.GetStop().GetLine(),
		})
	}
}

// EnterQuery_block is called when production query_block is entered.
func (l *whereRequireListener) EnterQuery_block(ctx *parser.Query_blockContext) {
	if ctx.Where_clause() == nil {
		l.adviceList = append(l.adviceList, advisor.Advice{
			Status:  l.level,
			Code:    advisor.StatementNoWhere,
			Title:   l.title,
			Content: "WHERE clause is required for SELECT statement.",
			Line:    ctx.GetStop().GetLine(),
		})
	}
}
