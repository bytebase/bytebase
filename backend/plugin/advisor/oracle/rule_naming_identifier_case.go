// Package oracle is the advisor for oracle database.
package oracle

import (
	"context"
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/plsql"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	plsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/plsql"
)

var (
	_ advisor.Advisor = (*NamingIdentifierCaseAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, advisor.SchemaRuleIdentifierCase, &NamingIdentifierCaseAdvisor{})
}

// NamingIdentifierCaseAdvisor is the advisor checking for identifier case.
type NamingIdentifierCaseAdvisor struct {
}

// Check checks for identifier case.
func (*NamingIdentifierCaseAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmtList, ok := checkCtx.AST.([]*plsqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to ParseResult")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	payload, err := advisor.UnmarshalNamingCaseRulePayload(checkCtx.Rule.Payload)
	if err != nil {
		return nil, err
	}

	rule := NewNamingIdentifierCaseRule(level, string(checkCtx.Rule.Type), checkCtx.CurrentDatabase, payload.Upper)
	checker := NewGenericChecker([]Rule{rule})

	for _, stmtNode := range stmtList {
		rule.SetBaseLine(stmtNode.BaseLine)
		checker.SetBaseLine(stmtNode.BaseLine)
		antlr.ParseTreeWalkerDefault.Walk(checker, stmtNode.Tree)
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
