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
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

var (
	_ advisor.Advisor = (*StatementMaximumJoinTableCountAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_STATEMENT_MAXIMUM_JOIN_TABLE_COUNT, &StatementMaximumJoinTableCountAdvisor{})
}

type StatementMaximumJoinTableCountAdvisor struct {
}

func (*StatementMaximumJoinTableCountAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	numberPayload := checkCtx.Rule.GetNumberPayload()
	if numberPayload == nil {
		return nil, errors.New("number_payload is required for this rule")
	}

	// Create the rule
	rule := NewStatementMaximumJoinTableCountRule(level, checkCtx.Rule.Type.String(), int(numberPayload.Number))

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
