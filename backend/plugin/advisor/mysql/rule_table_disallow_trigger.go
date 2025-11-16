package mysql

import (
	"context"
	"fmt"

	advisorcode "github.com/bytebase/bytebase/backend/plugin/advisor/code"

	"github.com/antlr4-go/antlr/v4"
	"github.com/bytebase/parser/mysql"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

var (
	_ advisor.Advisor = (*TableDisallowTriggerAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.SchemaRuleTableDisallowTrigger, &TableDisallowTriggerAdvisor{})
}

// TableDisallowTriggerAdvisor is the advisor checking for disallow table trigger.
type TableDisallowTriggerAdvisor struct {
}

// Check checks for disallow table trigger.
func (*TableDisallowTriggerAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmtList, ok := checkCtx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parser result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	// Create the rule
	rule := NewTableDisallowTriggerRule(level, string(checkCtx.Rule.Type))

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	for _, stmt := range stmtList {
		rule.SetBaseLine(stmt.BaseLine)
		checker.SetBaseLine(stmt.BaseLine)
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
	}

	return checker.GetAdviceList(), nil
}

// TableDisallowTriggerRule checks for disallow table trigger.
type TableDisallowTriggerRule struct {
	BaseRule
	text string
}

// NewTableDisallowTriggerRule creates a new TableDisallowTriggerRule.
func NewTableDisallowTriggerRule(level storepb.Advice_Status, title string) *TableDisallowTriggerRule {
	return &TableDisallowTriggerRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
	}
}

// Name returns the rule name.
func (*TableDisallowTriggerRule) Name() string {
	return "TableDisallowTriggerRule"
}

// OnEnter is called when entering a parse tree node.
func (r *TableDisallowTriggerRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case NodeTypeQuery:
		queryCtx, ok := ctx.(*mysql.QueryContext)
		if !ok {
			return nil
		}
		r.text = queryCtx.GetParser().GetTokenStream().GetTextFromRuleContext(queryCtx)
	case NodeTypeCreateTrigger:
		r.checkCreateTrigger(ctx.(*mysql.CreateTriggerContext))
	default:
		// Ignore other node types
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*TableDisallowTriggerRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *TableDisallowTriggerRule) checkCreateTrigger(ctx *mysql.CreateTriggerContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	code := advisorcode.Ok
	if ctx.TriggerName() != nil {
		code = advisorcode.CreateTableTrigger
	}

	if code != advisorcode.Ok {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          code.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf("Trigger is forbidden, but \"%s\" creates", r.text),
			StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + ctx.GetStart().GetLine()),
		})
	}
}
