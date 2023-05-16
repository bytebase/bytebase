// Package oracle is the advisor for oracle database.
package oracle

import (
	"fmt"
	"strings"

	"github.com/antlr/antlr4/runtime/Go/antlr/v4"
	parser "github.com/bytebase/plsql-parser"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/db"
)

var (
	_ advisor.Advisor = (*NamingIdentifierCaseAdvisor)(nil)
)

func init() {
	advisor.Register(db.Oracle, advisor.OracleIdentifierCase, &NamingIdentifierCaseAdvisor{})
}

// NamingIdentifierCaseAdvisor is the advisor checking for identifier case.
type NamingIdentifierCaseAdvisor struct {
}

// Check checks for identifier case.
func (*NamingIdentifierCaseAdvisor) Check(ctx advisor.Context, statement string) ([]advisor.Advice, error) {
	tree, errAdvice := parseStatement(statement)
	if errAdvice != nil {
		return errAdvice, nil
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	payload, err := advisor.UnmarshalNamingCaseRulePayload(ctx.Rule.Payload)
	if err != nil {
		return nil, err
	}

	listener := &namingIdentifierCaseListener{
		level:         level,
		title:         string(ctx.Rule.Type),
		currentSchema: ctx.CurrentSchema,
		namingCase:    payload.Case,
	}

	antlr.ParseTreeWalkerDefault.Walk(listener, tree)

	return listener.generateAdvice()
}

// namingIdentifierCaseListener is the listener for identifier case.
type namingIdentifierCaseListener struct {
	*parser.BasePlSqlParserListener

	level         advisor.Status
	title         string
	currentSchema string
	adviceList    []advisor.Advice
	namingCase    advisor.NamingCase
}

func (l *namingIdentifierCaseListener) generateAdvice() ([]advisor.Advice, error) {
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

// EnterId_expression is called when production id_expression is entered.
func (l *namingIdentifierCaseListener) EnterId_expression(ctx *parser.Id_expressionContext) {
	identifier := normalizeIDExpression(ctx)
	switch l.namingCase {
	case advisor.NamingCaseLower:
		if identifier != strings.ToLower(identifier) {
			l.adviceList = append(l.adviceList, advisor.Advice{
				Status:  l.level,
				Code:    advisor.NamingCaseMismatch,
				Title:   l.title,
				Content: fmt.Sprintf("Identifier %q should be lower case", identifier),
				Line:    ctx.GetStart().GetLine(),
			})
		}
	case advisor.NamingCaseUpper:
		if identifier != strings.ToUpper(identifier) {
			l.adviceList = append(l.adviceList, advisor.Advice{
				Status:  l.level,
				Code:    advisor.NamingCaseMismatch,
				Title:   l.title,
				Content: fmt.Sprintf("Identifier %q should be upper case", identifier),
				Line:    ctx.GetStart().GetLine(),
			})
		}
	}
}
