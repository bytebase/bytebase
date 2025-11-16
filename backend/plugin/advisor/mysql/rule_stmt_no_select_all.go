package mysql

import (
	"context"
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	"github.com/bytebase/parser/mysql"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

var (
	_ advisor.Advisor = (*NoSelectAllAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.SchemaRuleStatementNoSelectAll, &NoSelectAllAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, advisor.SchemaRuleStatementNoSelectAll, &NoSelectAllAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, advisor.SchemaRuleStatementNoSelectAll, &NoSelectAllAdvisor{})
}

// NoSelectAllAdvisor is the advisor checking for no "select *".
type NoSelectAllAdvisor struct {
}

// Check checks for no "select *".
func (*NoSelectAllAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	root, ok := checkCtx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parse result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	// Create the rule
	rule := NewNoSelectAllRule(level, string(checkCtx.Rule.Type))

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	for _, stmtNode := range root {
		rule.SetBaseLine(stmtNode.BaseLine)
		checker.SetBaseLine(stmtNode.BaseLine)
		antlr.ParseTreeWalkerDefault.Walk(checker, stmtNode.Tree)
	}

	return checker.GetAdviceList(), nil
}

// NoSelectAllRule checks for no "select *".
type NoSelectAllRule struct {
	BaseRule
	text string
}

// NewNoSelectAllRule creates a new NoSelectAllRule.
func NewNoSelectAllRule(level storepb.Advice_Status, title string) *NoSelectAllRule {
	return &NoSelectAllRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
	}
}

// Name returns the rule name.
func (*NoSelectAllRule) Name() string {
	return "NoSelectAllRule"
}

// OnEnter is called when entering a parse tree node.
func (r *NoSelectAllRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case NodeTypeQuery:
		queryCtx, ok := ctx.(*mysql.QueryContext)
		if !ok {
			return nil
		}
		r.text = queryCtx.GetParser().GetTokenStream().GetTextFromRuleContext(queryCtx)
	case NodeTypeSelectItemList:
		r.checkSelectItemList(ctx.(*mysql.SelectItemListContext))
	default:
		// Other node types
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*NoSelectAllRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *NoSelectAllRule) checkSelectItemList(ctx *mysql.SelectItemListContext) {
	if ctx.MULT_OPERATOR() != nil {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          code.StatementSelectAll.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf("\"%s\" uses SELECT all", r.text),
			StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + ctx.GetStart().GetLine()),
		})
	}
}
