// Package oracle is the advisor for oracle database.
package oracle

import (
	"context"
	"fmt"

	"github.com/bytebase/bytebase/backend/plugin/advisor/code"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/plsql"
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
	advisor.Register(storepb.Engine_ORACLE, advisor.SchemaRuleIdentifierNoKeyword, &NamingIdentifierNoKeywordAdvisor{})
}

// NamingIdentifierNoKeywordAdvisor is the advisor checking for identifier naming convention without keyword.
type NamingIdentifierNoKeywordAdvisor struct {
}

// Check checks for identifier naming convention without keyword.
func (*NamingIdentifierNoKeywordAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmtList, ok := checkCtx.AST.([]*plsqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to ParseResult")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := NewNamingIdentifierNoKeywordRule(level, string(checkCtx.Rule.Type), checkCtx.CurrentDatabase)
	checker := NewGenericChecker([]Rule{rule})

	for _, stmtNode := range stmtList {
		rule.SetBaseLine(stmtNode.BaseLine)
		checker.SetBaseLine(stmtNode.BaseLine)
		antlr.ParseTreeWalkerDefault.Walk(checker, stmtNode.Tree)
	}

	return checker.GetAdviceList()
}

// NamingIdentifierNoKeywordRule is the rule implementation for identifier naming convention without keyword.
type NamingIdentifierNoKeywordRule struct {
	BaseRule

	currentDatabase string
}

// NewNamingIdentifierNoKeywordRule creates a new NamingIdentifierNoKeywordRule.
func NewNamingIdentifierNoKeywordRule(level storepb.Advice_Status, title string, currentDatabase string) *NamingIdentifierNoKeywordRule {
	return &NamingIdentifierNoKeywordRule{
		BaseRule:        NewBaseRule(level, title, 0),
		currentDatabase: currentDatabase,
	}
}

// Name returns the rule name.
func (*NamingIdentifierNoKeywordRule) Name() string {
	return "naming.identifier-no-keyword"
}

// OnEnter is called when the parser enters a rule context.
func (r *NamingIdentifierNoKeywordRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	if nodeType == "Id_expression" {
		r.handleIDExpression(ctx.(*parser.Id_expressionContext))
	}
	return nil
}

// OnExit is called when the parser exits a rule context.
func (*NamingIdentifierNoKeywordRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *NamingIdentifierNoKeywordRule) handleIDExpression(ctx *parser.Id_expressionContext) {
	identifier := normalizeIDExpression(ctx)
	if plsqlparser.IsOracleKeyword(identifier) {
		r.AddAdvice(
			r.level,
			code.NameIsKeywordIdentifier.Int32(),
			fmt.Sprintf("Identifier %q is a keyword and should be avoided", identifier),
			common.ConvertANTLRLineToPosition(r.baseLine+ctx.GetStart().GetLine()),
		)
	}
}
