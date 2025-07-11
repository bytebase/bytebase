// Package oracle is the advisor for oracle database.
package oracle

import (
	"context"
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/plsql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	plsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/plsql"
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
func (*NamingIdentifierNoKeywordAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, ok := checkCtx.AST.(antlr.Tree)
	if !ok {
		return nil, errors.Errorf("failed to convert to Tree")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	listener := &namingIdentifierNoKeywordListener{
		level:           level,
		title:           string(checkCtx.Rule.Type),
		currentDatabase: checkCtx.CurrentDatabase,
	}

	antlr.ParseTreeWalkerDefault.Walk(listener, tree)

	return listener.generateAdvice()
}

// namingIdentifierNoKeywordListener is the listener for identifier naming convention without keyword.
type namingIdentifierNoKeywordListener struct {
	*parser.BasePlSqlParserListener

	level           storepb.Advice_Status
	title           string
	currentDatabase string
	adviceList      []*storepb.Advice
}

func (l *namingIdentifierNoKeywordListener) generateAdvice() ([]*storepb.Advice, error) {
	return l.adviceList, nil
}

// EnterId_expression is called when production id_expression is entered.
func (l *namingIdentifierNoKeywordListener) EnterId_expression(ctx *parser.Id_expressionContext) {
	identifier := normalizeIDExpression(ctx)
	if plsqlparser.IsOracleKeyword(identifier) {
		l.adviceList = append(l.adviceList, &storepb.Advice{
			Status:        l.level,
			Code:          advisor.NameIsKeywordIdentifier.Int32(),
			Title:         l.title,
			Content:       fmt.Sprintf("Identifier %q is a keyword and should be avoided", identifier),
			StartPosition: common.ConvertANTLRLineToPosition(ctx.GetStart().GetLine()),
		})
	}
}
