package mysql

import (
	"context"
	"fmt"

	"github.com/bytebase/bytebase/backend/plugin/advisor/code"

	"github.com/antlr4-go/antlr/v4"
	"github.com/bytebase/parser/mysql"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

var (
	_ advisor.Advisor = (*NamingIdentifierNoKeywordAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_NAMING_IDENTIFIER_NO_KEYWORD, &NamingIdentifierNoKeywordAdvisor{})
}

// NamingIdentifierNoKeywordAdvisor is the advisor checking for identifier naming convention without keyword.
type NamingIdentifierNoKeywordAdvisor struct {
}

// Check checks for identifier naming convention without keyword.
func (*NamingIdentifierNoKeywordAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	// Create the rule
	rule := NewNamingIdentifierNoKeywordRule(level, checkCtx.Rule.Type.String())

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	for _, stmt := range checkCtx.ParsedStatements {
		rule.SetBaseLine(stmt.BaseLine())
		checker.SetBaseLine(stmt.BaseLine())
		if stmt.AST == nil {
			continue
		}
		antlrAST, ok := base.GetANTLRAST(stmt.AST)
		if !ok {
			continue
		}
		antlr.ParseTreeWalkerDefault.Walk(checker, antlrAST.Tree)
	}

	return checker.GetAdviceList(), nil
}

// NamingIdentifierNoKeywordRule checks for identifier naming convention without keyword.
type NamingIdentifierNoKeywordRule struct {
	BaseRule
}

// NewNamingIdentifierNoKeywordRule creates a new NamingIdentifierNoKeywordRule.
func NewNamingIdentifierNoKeywordRule(level storepb.Advice_Status, title string) *NamingIdentifierNoKeywordRule {
	return &NamingIdentifierNoKeywordRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
	}
}

// Name returns the rule name.
func (*NamingIdentifierNoKeywordRule) Name() string {
	return "NamingIdentifierNoKeywordRule"
}

// OnEnter is called when entering a parse tree node.
func (r *NamingIdentifierNoKeywordRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case NodeTypePureIdentifier:
		r.checkPureIdentifier(ctx.(*mysql.PureIdentifierContext))
	case NodeTypeIdentifierKeyword:
		r.checkIdentifierKeyword(ctx.(*mysql.IdentifierKeywordContext))
	default:
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*NamingIdentifierNoKeywordRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *NamingIdentifierNoKeywordRule) checkPureIdentifier(ctx *mysql.PureIdentifierContext) {
	// The suspect identifier should be always wrapped in backquotes, otherwise a syntax error will be thrown before entering this checker.
	textNode := ctx.BACK_TICK_QUOTED_ID()
	if textNode == nil {
		return
	}

	// Remove backticks as possible.
	identifier := trimBackTicks(textNode.GetText())
	advice := r.checkIdentifier(identifier)
	if advice != nil {
		advice.StartPosition = common.ConvertANTLRLineToPosition(r.baseLine + ctx.GetStart().GetLine())
		r.adviceList = append(r.adviceList, advice)
	}
}

func (r *NamingIdentifierNoKeywordRule) checkIdentifierKeyword(ctx *mysql.IdentifierKeywordContext) {
	identifier := ctx.GetText()
	advice := r.checkIdentifier(identifier)
	if advice != nil {
		advice.StartPosition = common.ConvertANTLRLineToPosition(r.baseLine + ctx.GetStart().GetLine())
		r.adviceList = append(r.adviceList, advice)
	}
}

func (r *NamingIdentifierNoKeywordRule) checkIdentifier(identifier string) *storepb.Advice {
	if isKeyword(identifier) {
		return &storepb.Advice{
			Status:  r.level,
			Code:    code.NameIsKeywordIdentifier.Int32(),
			Title:   r.title,
			Content: fmt.Sprintf("Identifier %q is a keyword and should be avoided", identifier),
		}
	}

	return nil
}

func trimBackTicks(s string) string {
	if len(s) < 2 {
		return s
	}
	return s[1 : len(s)-1]
}
