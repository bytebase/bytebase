// Package oracle is the advisor for oracle database.
package oracle

import (
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/plsql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*NamingIdentifierCaseAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, advisor.OracleIdentifierCase, &NamingIdentifierCaseAdvisor{})
	advisor.Register(storepb.Engine_DM, advisor.OracleIdentifierCase, &NamingIdentifierCaseAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE_ORACLE, advisor.OracleIdentifierCase, &NamingIdentifierCaseAdvisor{})
}

// NamingIdentifierCaseAdvisor is the advisor checking for identifier case.
type NamingIdentifierCaseAdvisor struct {
}

// Check checks for identifier case.
func (*NamingIdentifierCaseAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	tree, ok := ctx.AST.(antlr.Tree)
	if !ok {
		return nil, errors.Errorf("failed to convert to Tree")
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
		upper:         payload.Upper,
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
	upper         bool
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
	if l.upper {
		if identifier != strings.ToUpper(identifier) {
			l.adviceList = append(l.adviceList, advisor.Advice{
				Status:  l.level,
				Code:    advisor.NamingCaseMismatch,
				Title:   l.title,
				Content: fmt.Sprintf("Identifier %q should be upper case", identifier),
				Line:    ctx.GetStart().GetLine(),
			})
		}
	} else {
		if identifier != strings.ToLower(identifier) {
			l.adviceList = append(l.adviceList, advisor.Advice{
				Status:  l.level,
				Code:    advisor.NamingCaseMismatch,
				Title:   l.title,
				Content: fmt.Sprintf("Identifier %q should be lower case", identifier),
				Line:    ctx.GetStart().GetLine(),
			})
		}
	}
}
