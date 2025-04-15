package mssql

import (
	"context"
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/tsql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	advisor.Register(storepb.Engine_MSSQL, advisor.MSSQLStatementWhereDisallowFunctionsAndCalculations, &DisallowFuncAndCalculationsAdvisor{})
}

type DisallowFuncAndCalculationsAdvisor struct{}

type DisallowFuncAndCalculationsChecker struct {
	*parser.BaseTSqlParserListener
	level storepb.Advice_Status
	title string

	// We only check the 'WHERE' clause in a 'SELECT' statement.
	// Also, the value of 'whereCnt' represents the depth of entering the 'WHERE' clause.  Same below.
	whereCnt      int
	selectStatCnt int
	havingCnt     int
	// Each statement can only trigger the rule once.
	hasTriggeredRule bool
	adviceList       []*storepb.Advice
}

var _ advisor.Advisor = (*DisallowFuncAndCalculationsAdvisor)(nil)

func (*DisallowFuncAndCalculationsAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, ok := checkCtx.AST.(antlr.Tree)
	if !ok {
		return nil, errors.Errorf("failed to convert to AST tree")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	checker := &DisallowFuncAndCalculationsChecker{
		level:            level,
		title:            checkCtx.Rule.Type,
		selectStatCnt:    0,
		whereCnt:         0,
		havingCnt:        0,
		hasTriggeredRule: false,
	}

	antlr.ParseTreeWalkerDefault.Walk(checker, tree)

	return checker.adviceList, nil
}

func generateAdviceOnFunctionUsing(c *DisallowFuncAndCalculationsChecker, ctx antlr.BaseParserRuleContext) *storepb.Advice {
	if c.whereCnt == 0 || c.hasTriggeredRule {
		return nil
	}
	c.hasTriggeredRule = true
	return &storepb.Advice{
		Status:        c.level,
		Code:          advisor.StatementDisallowFunctionsAndCalculations.Int32(),
		Title:         c.title,
		Content:       fmt.Sprintf("Calling function '%s' in 'WHERE' clause is not allowed", ctx.GetText()),
		StartPosition: common.ConvertANTLRLineToPosition(ctx.GetStart().GetLine()),
	}
}

func generateAdviceOnPerformingCalculations(c *DisallowFuncAndCalculationsChecker, ctx antlr.BaseParserRuleContext) *storepb.Advice {
	if c.whereCnt == 0 || c.hasTriggeredRule {
		return nil
	}
	c.hasTriggeredRule = true
	return &storepb.Advice{
		Status:        c.level,
		Code:          advisor.StatementDisallowFunctionsAndCalculations.Int32(),
		Title:         c.title,
		Content:       "Performing calculations in 'WHERE' clause is not allowed",
		StartPosition: common.ConvertANTLRLineToPosition(ctx.GetStart().GetLine()),
	}
}

func (c *DisallowFuncAndCalculationsChecker) EnterQuery_specification(_ *parser.Query_specificationContext) {
	c.selectStatCnt++
}

func (c *DisallowFuncAndCalculationsChecker) ExitQuery_specification(_ *parser.Query_specificationContext) {
	c.selectStatCnt--
}

func (c *DisallowFuncAndCalculationsChecker) EnterHaving_clause(_ *parser.Having_clauseContext) {
	c.havingCnt++
}

func (c *DisallowFuncAndCalculationsChecker) ExitHaving_clause(_ *parser.Having_clauseContext) {
	c.havingCnt--
}

func (c *DisallowFuncAndCalculationsChecker) EnterSearch_condition(_ *parser.Search_conditionContext) {
	if c.selectStatCnt != 0 && c.havingCnt == 0 {
		c.whereCnt++
	}
}

func (c *DisallowFuncAndCalculationsChecker) ExitSearch_condition(_ *parser.Search_conditionContext) {
	c.whereCnt--
	c.hasTriggeredRule = false
}

// Calling functions.
func (c *DisallowFuncAndCalculationsChecker) EnterBUILT_IN_FUNC(ctx *parser.BUILT_IN_FUNCContext) {
	if advice := generateAdviceOnFunctionUsing(c, ctx.BaseParserRuleContext); advice != nil {
		c.adviceList = append(c.adviceList, advice)
	}
}

func (c *DisallowFuncAndCalculationsChecker) EnterRANKING_WINDOWED_FUNC(ctx *parser.RANKING_WINDOWED_FUNCContext) {
	if advice := generateAdviceOnFunctionUsing(c, ctx.BaseParserRuleContext); advice != nil {
		c.adviceList = append(c.adviceList, advice)
	}
}

func (c *DisallowFuncAndCalculationsChecker) EnterAGGREGATE_WINDOWED_FUNC(ctx *parser.AGGREGATE_WINDOWED_FUNCContext) {
	if advice := generateAdviceOnFunctionUsing(c, ctx.BaseParserRuleContext); advice != nil {
		c.adviceList = append(c.adviceList, advice)
	}
}

func (c *DisallowFuncAndCalculationsChecker) EnterANALYTIC_WINDOWED_FUNC(ctx *parser.ANALYTIC_WINDOWED_FUNCContext) {
	if advice := generateAdviceOnFunctionUsing(c, ctx.BaseParserRuleContext); advice != nil {
		c.adviceList = append(c.adviceList, advice)
	}
}

func (c *DisallowFuncAndCalculationsChecker) EnterSCALAR_FUNCTION(ctx *parser.SCALAR_FUNCTIONContext) {
	if advice := generateAdviceOnFunctionUsing(c, ctx.BaseParserRuleContext); advice != nil {
		c.adviceList = append(c.adviceList, advice)
	}
}

func (c *DisallowFuncAndCalculationsChecker) EnterFREE_TEXT(ctx *parser.FREE_TEXTContext) {
	if advice := generateAdviceOnFunctionUsing(c, ctx.BaseParserRuleContext); advice != nil {
		c.adviceList = append(c.adviceList, advice)
	}
}

func (c *DisallowFuncAndCalculationsChecker) EnterPARTITION_FUNC(ctx *parser.PARTITION_FUNCContext) {
	if advice := generateAdviceOnFunctionUsing(c, ctx.BaseParserRuleContext); advice != nil {
		c.adviceList = append(c.adviceList, advice)
	}
}

func (c *DisallowFuncAndCalculationsChecker) EnterHIERARCHYID_METHOD(ctx *parser.HIERARCHYID_METHODContext) {
	if advice := generateAdviceOnFunctionUsing(c, ctx.BaseParserRuleContext); advice != nil {
		c.adviceList = append(c.adviceList, advice)
	}
}

// Performing calculations.
func (c *DisallowFuncAndCalculationsChecker) EnterExpression(ctx *parser.ExpressionContext) {
	if ctx.GetOp() != nil {
		if advice := generateAdviceOnPerformingCalculations(c, ctx.BaseParserRuleContext); advice != nil {
			c.adviceList = append(c.adviceList, advice)
		}
	}
}

func (c *DisallowFuncAndCalculationsChecker) EnterUnary_operator_expression(ctx *parser.Unary_operator_expressionContext) {
	if advice := generateAdviceOnPerformingCalculations(c, ctx.BaseParserRuleContext); advice != nil {
		c.adviceList = append(c.adviceList, advice)
	}
}
