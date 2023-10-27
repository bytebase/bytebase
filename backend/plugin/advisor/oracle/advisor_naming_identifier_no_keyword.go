// Package oracle is the advisor for oracle database.
package oracle

import (
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/plsql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	plsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/plsql"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*NamingIdentifierNoKeywordAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, advisor.OracleIdentifierNamingNoKeyword, &NamingIdentifierNoKeywordAdvisor{})
	advisor.Register(storepb.Engine_DM, advisor.OracleIdentifierNamingNoKeyword, &NamingIdentifierNoKeywordAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE_ORACLE, advisor.OracleIdentifierNamingNoKeyword, &NamingIdentifierNoKeywordAdvisor{})
}

// NamingIdentifierNoKeywordAdvisor is the advisor checking for identifier naming convention without keyword.
type NamingIdentifierNoKeywordAdvisor struct {
}

// Check checks for identifier naming convention without keyword.
func (*NamingIdentifierNoKeywordAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	tree, ok := ctx.AST.(antlr.Tree)
	if !ok {
		return nil, errors.Errorf("failed to convert to Tree")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}

	listener := &namingIdentifierNoKeywordListener{
		level:         level,
		title:         string(ctx.Rule.Type),
		currentSchema: ctx.CurrentSchema,
	}

	antlr.ParseTreeWalkerDefault.Walk(listener, tree)

	return listener.generateAdvice()
}

// namingIdentifierNoKeywordListener is the listener for identifier naming convention without keyword.
type namingIdentifierNoKeywordListener struct {
	*parser.BasePlSqlParserListener

	level         advisor.Status
	title         string
	currentSchema string
	adviceList    []advisor.Advice
}

func (l *namingIdentifierNoKeywordListener) generateAdvice() ([]advisor.Advice, error) {
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
func (l *namingIdentifierNoKeywordListener) EnterId_expression(ctx *parser.Id_expressionContext) {
	identifier := normalizeIDExpression(ctx)
	if plsqlparser.IsOracleKeyword(identifier) {
		l.adviceList = append(l.adviceList, advisor.Advice{
			Status:  l.level,
			Code:    advisor.NameIsKeywordIdentifier,
			Title:   l.title,
			Content: fmt.Sprintf("Identifier %q is a keyword and should be avoided", identifier),
			Line:    ctx.GetStart().GetLine(),
		})
	}
}
