// Package oracle is the advisor for oracle database.
package oracle

import (
	"context"
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/plsql"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

var (
	_ advisor.Advisor = (*NamingIdentifierCaseAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, storepb.SQLReviewRule_NAMING_IDENTIFIER_CASE, &NamingIdentifierCaseAdvisor{})
}

// NamingIdentifierCaseAdvisor is the advisor checking for identifier case.
type NamingIdentifierCaseAdvisor struct {
}

// Check checks for identifier case.
func (*NamingIdentifierCaseAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	namingCasePayload := checkCtx.Rule.GetNamingCasePayload()

	rule := NewNamingIdentifierCaseRule(level, checkCtx.Rule.Type.String(), checkCtx.CurrentDatabase, namingCasePayload.Upper)
	checker := NewGenericChecker([]Rule{rule})

	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		antlrAST, ok := base.GetANTLRAST(stmt.AST)
		if !ok {
			continue
		}
		rule.SetBaseLine(stmt.BaseLine())
		checker.SetBaseLine(stmt.BaseLine())
		antlr.ParseTreeWalkerDefault.Walk(checker, antlrAST.Tree)
	}

	return checker.GetAdviceList()
}

// NamingIdentifierCaseRule is the rule implementation for identifier case.
type NamingIdentifierCaseRule struct {
	BaseRule

	currentDatabase string
	upper           bool
}

// NewNamingIdentifierCaseRule creates a new NamingIdentifierCaseRule.
func NewNamingIdentifierCaseRule(level storepb.Advice_Status, title string, currentDatabase string, upper bool) *NamingIdentifierCaseRule {
	return &NamingIdentifierCaseRule{
		BaseRule:        NewBaseRule(level, title, 0),
		currentDatabase: currentDatabase,
		upper:           upper,
	}
}

// Name returns the rule name.
func (*NamingIdentifierCaseRule) Name() string {
	return "naming.identifier-case"
}

// OnEnter is called when the parser enters a rule context.
func (r *NamingIdentifierCaseRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	if nodeType == "Id_expression" {
		r.handleIDExpression(ctx.(*parser.Id_expressionContext))
	}
	return nil
}

// OnExit is called when the parser exits a rule context.
func (*NamingIdentifierCaseRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *NamingIdentifierCaseRule) handleIDExpression(ctx *parser.Id_expressionContext) {
	identifier := normalizeIDExpression(ctx)
	if r.upper {
		if identifier != strings.ToUpper(identifier) {
			r.AddAdvice(
				r.level,
				code.NamingCaseMismatch.Int32(),
				fmt.Sprintf("Identifier %q should be upper case", identifier),
				common.ConvertANTLRLineToPosition(r.baseLine+ctx.GetStart().GetLine()),
			)
		}
	} else {
		if identifier != strings.ToLower(identifier) {
			r.AddAdvice(
				r.level,
				code.NamingCaseMismatch.Int32(),
				fmt.Sprintf("Identifier %q should be lower case", identifier),
				common.ConvertANTLRLineToPosition(r.baseLine+ctx.GetStart().GetLine()),
			)
		}
	}
}
