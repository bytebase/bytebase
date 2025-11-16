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
	_ advisor.Advisor = (*StatementMaximumJoinTableCountAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.SchemaRuleStatementMaximumJoinTableCount, &StatementMaximumJoinTableCountAdvisor{})
}

type StatementMaximumJoinTableCountAdvisor struct {
}

func (*StatementMaximumJoinTableCountAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmtList, ok := checkCtx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parse result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	payload, err := advisor.UnmarshalNumberTypeRulePayload(checkCtx.Rule.Payload)
	if err != nil {
		return nil, err
	}

	// Create the rule
	rule := NewStatementMaximumJoinTableCountRule(level, string(checkCtx.Rule.Type), payload.Number)

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	for _, stmt := range stmtList {
		rule.SetBaseLine(stmt.BaseLine)
		checker.SetBaseLine(stmt.BaseLine)
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
	}

	return checker.GetAdviceList(), nil
}

// StatementMaximumJoinTableCountRule checks for maximum join table count.
type StatementMaximumJoinTableCountRule struct {
	BaseRule
	text          string
	limitMaxValue int
	count         int
}

// NewStatementMaximumJoinTableCountRule creates a new StatementMaximumJoinTableCountRule.
func NewStatementMaximumJoinTableCountRule(level storepb.Advice_Status, title string, limitMaxValue int) *StatementMaximumJoinTableCountRule {
	return &StatementMaximumJoinTableCountRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
		limitMaxValue: limitMaxValue,
	}
}

// Name returns the rule name.
func (*StatementMaximumJoinTableCountRule) Name() string {
	return "StatementMaximumJoinTableCountRule"
}

// OnEnter is called when entering a parse tree node.
func (r *StatementMaximumJoinTableCountRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case NodeTypeQuery:
		queryCtx, ok := ctx.(*mysql.QueryContext)
		if !ok {
			return nil
		}
		r.text = queryCtx.GetParser().GetTokenStream().GetTextFromRuleContext(queryCtx)
	case NodeTypeJoinedTable:
		r.checkJoinedTable(ctx.(*mysql.JoinedTableContext))
	default:
		// No action required for other node types
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*StatementMaximumJoinTableCountRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *StatementMaximumJoinTableCountRule) checkJoinedTable(ctx *mysql.JoinedTableContext) {
	r.count++
	// The count starts from 0. We count the number of tables in the joins.
	if r.count == r.limitMaxValue {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          code.StatementMaximumJoinTableCount.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf("\"%s\" exceeds the maximum number of joins %d.", r.text, r.limitMaxValue),
			StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + ctx.GetStart().GetLine()),
		})
	}
}
