package mysql

import (
	"context"
	"fmt"
	"strconv"

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
	_ advisor.Advisor = (*StatementMaximumLimitValueAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.SchemaRuleStatementMaximumLimitValue, &StatementMaximumLimitValueAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, advisor.SchemaRuleStatementMaximumLimitValue, &StatementMaximumLimitValueAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, advisor.SchemaRuleStatementMaximumLimitValue, &StatementMaximumLimitValueAdvisor{})
}

type StatementMaximumLimitValueAdvisor struct {
}

func (*StatementMaximumLimitValueAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
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
	rule := NewStatementMaximumLimitValueRule(level, string(checkCtx.Rule.Type), payload.Number)

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	for _, stmt := range stmtList {
		rule.SetBaseLine(stmt.BaseLine)
		checker.SetBaseLine(stmt.BaseLine)
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
	}

	return checker.GetAdviceList(), nil
}

// StatementMaximumLimitValueRule checks for maximum limit value.
type StatementMaximumLimitValueRule struct {
	BaseRule
	text          string
	isSelect      bool
	limitMaxValue int
}

// NewStatementMaximumLimitValueRule creates a new StatementMaximumLimitValueRule.
func NewStatementMaximumLimitValueRule(level storepb.Advice_Status, title string, limitMaxValue int) *StatementMaximumLimitValueRule {
	return &StatementMaximumLimitValueRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
		limitMaxValue: limitMaxValue,
	}
}

// Name returns the rule name.
func (*StatementMaximumLimitValueRule) Name() string {
	return "StatementMaximumLimitValueRule"
}

// OnEnter is called when entering a parse tree node.
func (r *StatementMaximumLimitValueRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case NodeTypeQuery:
		queryCtx, ok := ctx.(*mysql.QueryContext)
		if !ok {
			return nil
		}
		r.text = queryCtx.GetParser().GetTokenStream().GetTextFromRuleContext(queryCtx)
	case NodeTypeSelectStatement:
		if mysqlparser.IsTopMySQLRule(&ctx.(*mysql.SelectStatementContext).BaseParserRuleContext) {
			r.isSelect = true
		}
	case NodeTypeLimitClause:
		r.checkLimitClause(ctx.(*mysql.LimitClauseContext))
	default:
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (r *StatementMaximumLimitValueRule) OnExit(ctx antlr.ParserRuleContext, nodeType string) error {
	if nodeType == NodeTypeSelectStatement {
		if mysqlparser.IsTopMySQLRule(&ctx.(*mysql.SelectStatementContext).BaseParserRuleContext) {
			r.isSelect = false
		}
	}
	return nil
}

func (r *StatementMaximumLimitValueRule) checkLimitClause(ctx *mysql.LimitClauseContext) {
	if !r.isSelect {
		return
	}
	if ctx.LIMIT_SYMBOL() == nil {
		return
	}

	limitOptions := ctx.LimitOptions()
	for _, limitOption := range limitOptions.AllLimitOption() {
		limitValue, err := strconv.Atoi(limitOption.GetText())
		if err != nil {
			// Ignore invalid limit value and continue.
			continue
		}

		if limitValue > r.limitMaxValue {
			r.AddAdvice(&storepb.Advice{
				Status:        r.level,
				Code:          code.StatementExceedMaximumLimitValue.Int32(),
				Title:         r.title,
				Content:       fmt.Sprintf("The limit value %d exceeds the maximum allowed value %d", limitValue, r.limitMaxValue),
				StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + ctx.GetStart().GetLine()),
			})
		}
	}
}
