package mysql

import (
	"context"
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	"github.com/bytebase/parser/mysql"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	advisorcode "github.com/bytebase/bytebase/backend/plugin/advisor/code"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

var (
	_ advisor.Advisor = (*ViewDisallowCreateAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.SchemaRuleViewDisallowCreate, &ViewDisallowCreateAdvisor{})
}

// ViewDisallowCreateAdvisor is the advisor checking for disallow creating view.
type ViewDisallowCreateAdvisor struct {
}

// Check checks for disallow creating view.
func (*ViewDisallowCreateAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmtList, ok := checkCtx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parser result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	// Create the rule
	rule := NewViewDisallowCreateRule(level, string(checkCtx.Rule.Type))

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	for _, stmt := range stmtList {
		rule.SetBaseLine(stmt.BaseLine)
		checker.SetBaseLine(stmt.BaseLine)
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
	}

	return checker.GetAdviceList(), nil
}

// ViewDisallowCreateRule checks for disallow creating view.
type ViewDisallowCreateRule struct {
	BaseRule
	text string
}

// NewViewDisallowCreateRule creates a new ViewDisallowCreateRule.
func NewViewDisallowCreateRule(level storepb.Advice_Status, title string) *ViewDisallowCreateRule {
	return &ViewDisallowCreateRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
	}
}

// Name returns the rule name.
func (*ViewDisallowCreateRule) Name() string {
	return "ViewDisallowCreateRule"
}

// OnEnter is called when entering a parse tree node.
func (r *ViewDisallowCreateRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case NodeTypeQuery:
		if queryCtx, ok := ctx.(*mysql.QueryContext); ok {
			r.text = queryCtx.GetParser().GetTokenStream().GetTextFromRuleContext(queryCtx)
		}
	case NodeTypeCreateView:
		r.checkCreateView(ctx.(*mysql.CreateViewContext))
	default:
		// Ignore other node types
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*ViewDisallowCreateRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *ViewDisallowCreateRule) checkCreateView(ctx *mysql.CreateViewContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	code := advisorcode.Ok
	if ctx.ViewName() != nil {
		code = advisorcode.DisallowCreateView
	}

	if code != advisorcode.Ok {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          code.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf("View is forbidden, but \"%s\" creates", r.text),
			StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + ctx.GetStart().GetLine()),
		})
	}
}
