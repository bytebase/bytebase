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
	_ advisor.Advisor = (*EventDisallowCreateAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.SchemaRuleEventDisallowCreate, &EventDisallowCreateAdvisor{})
}

// EventDisallowCreateAdvisor is the advisor checking for disallow creating event.
type EventDisallowCreateAdvisor struct {
}

// Check checks for disallow creating event.
func (*EventDisallowCreateAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmtList, ok := checkCtx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parser result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	// Create the rule
	rule := NewEventDisallowCreateRule(level, string(checkCtx.Rule.Type))

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	for _, stmt := range stmtList {
		rule.SetBaseLine(stmt.BaseLine)
		checker.SetBaseLine(stmt.BaseLine)
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
	}

	return checker.GetAdviceList(), nil
}

// EventDisallowCreateRule checks for disallow creating event.
type EventDisallowCreateRule struct {
	BaseRule
	text string
}

// NewEventDisallowCreateRule creates a new EventDisallowCreateRule.
func NewEventDisallowCreateRule(level storepb.Advice_Status, title string) *EventDisallowCreateRule {
	return &EventDisallowCreateRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
	}
}

// Name returns the rule name.
func (*EventDisallowCreateRule) Name() string {
	return "EventDisallowCreateRule"
}

// OnEnter is called when entering a parse tree node.
func (r *EventDisallowCreateRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case NodeTypeQuery:
		if queryCtx, ok := ctx.(*mysql.QueryContext); ok {
			r.text = queryCtx.GetParser().GetTokenStream().GetTextFromRuleContext(queryCtx)
		}
	case NodeTypeCreateEvent:
		r.checkCreateEvent(ctx.(*mysql.CreateEventContext))
	default:
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*EventDisallowCreateRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *EventDisallowCreateRule) checkCreateEvent(ctx *mysql.CreateEventContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	code := advisorcode.Ok
	if ctx.EventName() != nil {
		code = advisorcode.DisallowCreateEvent
	}

	if code != advisorcode.Ok {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          code.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf("Event is forbidden, but \"%s\" creates", r.text),
			StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + ctx.GetStart().GetLine()),
		})
	}
}
