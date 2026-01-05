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
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

var (
	_ advisor.Advisor = (*StatementMaximumLimitValueAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_STATEMENT_MAXIMUM_LIMIT_VALUE, &StatementMaximumLimitValueAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_STATEMENT_MAXIMUM_LIMIT_VALUE, &StatementMaximumLimitValueAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_STATEMENT_MAXIMUM_LIMIT_VALUE, &StatementMaximumLimitValueAdvisor{})
}

type StatementMaximumLimitValueAdvisor struct {
}

func (*StatementMaximumLimitValueAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	numberPayload := checkCtx.Rule.GetNumberPayload()
	if numberPayload == nil {
		return nil, errors.New("number_payload is required for this rule")
	}

	// Create the rule
	rule := NewStatementMaximumLimitValueRule(level, checkCtx.Rule.Type.String(), int(numberPayload.Number))

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
