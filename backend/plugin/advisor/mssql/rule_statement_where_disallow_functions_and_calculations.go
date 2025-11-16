package mssql

import (
	"context"
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/tsql"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

func init() {
	advisor.Register(storepb.Engine_MSSQL, advisor.SchemaRuleStatementWhereDisallowFunctionsAndCalculations, &DisallowFuncAndCalculationsAdvisor{})
}

type DisallowFuncAndCalculationsAdvisor struct{}

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

	// Create the rule
	rule := NewDisallowFuncAndCalculationsRule(level, string(checkCtx.Rule.Type))

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	antlr.ParseTreeWalkerDefault.Walk(checker, tree)

	return checker.GetAdviceList(), nil
}

// DisallowFuncAndCalculationsRule is the rule for disallowing functions and calculations in WHERE clause.
type DisallowFuncAndCalculationsRule struct {
	BaseRule
	// We only check the 'WHERE' clause in a 'SELECT' statement.
	// Also, the value of 'whereCnt' represents the depth of entering the 'WHERE' clause.  Same below.
	whereCnt      int
	selectStatCnt int
	havingCnt     int
	// Each statement can only trigger the rule once.
	hasTriggeredRule bool
}

// NewDisallowFuncAndCalculationsRule creates a new DisallowFuncAndCalculationsRule.
func NewDisallowFuncAndCalculationsRule(level storepb.Advice_Status, title string) *DisallowFuncAndCalculationsRule {
	return &DisallowFuncAndCalculationsRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
		selectStatCnt:    0,
		whereCnt:         0,
		havingCnt:        0,
		hasTriggeredRule: false,
	}
}

// Name returns the rule name.
func (*DisallowFuncAndCalculationsRule) Name() string {
	return "DisallowFuncAndCalculationsRule"
}

// OnEnter is called when entering a parse tree node.
func (r *DisallowFuncAndCalculationsRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case "Query_specification":
		r.enterQuerySpecification(ctx.(*parser.Query_specificationContext))
	case "Having_clause":
		r.enterHavingClause(ctx.(*parser.Having_clauseContext))
	case NodeTypeSearchCondition:
		r.enterSearchCondition(ctx.(*parser.Search_conditionContext))
	case "BUILT_IN_FUNC":
		r.enterBuiltInFunc(ctx.(*parser.BUILT_IN_FUNCContext))
	case "RANKING_WINDOWED_FUNC":
		r.enterRankingWindowedFunc(ctx.(*parser.RANKING_WINDOWED_FUNCContext))
	case "AGGREGATE_WINDOWED_FUNC":
		r.enterAggregateWindowedFunc(ctx.(*parser.AGGREGATE_WINDOWED_FUNCContext))
	case "ANALYTIC_WINDOWED_FUNC":
		r.enterAnalyticWindowedFunc(ctx.(*parser.ANALYTIC_WINDOWED_FUNCContext))
	case "SCALAR_FUNCTION":
		r.enterScalarFunction(ctx.(*parser.SCALAR_FUNCTIONContext))
	case "FREE_TEXT":
		r.enterFreeText(ctx.(*parser.FREE_TEXTContext))
	case "PARTITION_FUNC":
		r.enterPartitionFunc(ctx.(*parser.PARTITION_FUNCContext))
	case "HIERARCHYID_METHOD":
		r.enterHierarchyIDMethod(ctx.(*parser.HIERARCHYID_METHODContext))
	case "Expression":
		r.enterExpression(ctx.(*parser.ExpressionContext))
	case "Unary_operator_expression":
		r.enterUnaryOperatorExpression(ctx.(*parser.Unary_operator_expressionContext))
	default:
		// Ignore other node types
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (r *DisallowFuncAndCalculationsRule) OnExit(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case "Query_specification":
		r.exitQuerySpecification(ctx.(*parser.Query_specificationContext))
	case "Having_clause":
		r.exitHavingClause(ctx.(*parser.Having_clauseContext))
	case NodeTypeSearchCondition:
		r.exitSearchCondition(ctx.(*parser.Search_conditionContext))
	default:
		// Ignore other node types
	}
	return nil
}

func (r *DisallowFuncAndCalculationsRule) generateAdviceOnFunctionUsing(ctx antlr.BaseParserRuleContext) *storepb.Advice {
	if r.whereCnt == 0 || r.hasTriggeredRule {
		return nil
	}
	r.hasTriggeredRule = true
	return &storepb.Advice{
		Status:        r.level,
		Code:          code.StatementDisallowFunctionsAndCalculations.Int32(),
		Title:         r.title,
		Content:       fmt.Sprintf("Calling function '%s' in 'WHERE' clause is not allowed", ctx.GetText()),
		StartPosition: common.ConvertANTLRLineToPosition(ctx.GetStart().GetLine()),
	}
}

func (r *DisallowFuncAndCalculationsRule) generateAdviceOnPerformingCalculations(ctx antlr.BaseParserRuleContext) *storepb.Advice {
	if r.whereCnt == 0 || r.hasTriggeredRule {
		return nil
	}
	r.hasTriggeredRule = true
	return &storepb.Advice{
		Status:        r.level,
		Code:          code.StatementDisallowFunctionsAndCalculations.Int32(),
		Title:         r.title,
		Content:       "Performing calculations in 'WHERE' clause is not allowed",
		StartPosition: common.ConvertANTLRLineToPosition(ctx.GetStart().GetLine()),
	}
}

func (r *DisallowFuncAndCalculationsRule) enterQuerySpecification(_ *parser.Query_specificationContext) {
	r.selectStatCnt++
}

func (r *DisallowFuncAndCalculationsRule) exitQuerySpecification(_ *parser.Query_specificationContext) {
	r.selectStatCnt--
}

func (r *DisallowFuncAndCalculationsRule) enterHavingClause(_ *parser.Having_clauseContext) {
	r.havingCnt++
}

func (r *DisallowFuncAndCalculationsRule) exitHavingClause(_ *parser.Having_clauseContext) {
	r.havingCnt--
}

func (r *DisallowFuncAndCalculationsRule) enterSearchCondition(_ *parser.Search_conditionContext) {
	if r.selectStatCnt != 0 && r.havingCnt == 0 {
		r.whereCnt++
	}
}

func (r *DisallowFuncAndCalculationsRule) exitSearchCondition(_ *parser.Search_conditionContext) {
	r.whereCnt--
	r.hasTriggeredRule = false
}

// Calling functions.
func (r *DisallowFuncAndCalculationsRule) enterBuiltInFunc(ctx *parser.BUILT_IN_FUNCContext) {
	if advice := r.generateAdviceOnFunctionUsing(ctx.BaseParserRuleContext); advice != nil {
		r.AddAdvice(advice)
	}
}

func (r *DisallowFuncAndCalculationsRule) enterRankingWindowedFunc(ctx *parser.RANKING_WINDOWED_FUNCContext) {
	if advice := r.generateAdviceOnFunctionUsing(ctx.BaseParserRuleContext); advice != nil {
		r.AddAdvice(advice)
	}
}

func (r *DisallowFuncAndCalculationsRule) enterAggregateWindowedFunc(ctx *parser.AGGREGATE_WINDOWED_FUNCContext) {
	if advice := r.generateAdviceOnFunctionUsing(ctx.BaseParserRuleContext); advice != nil {
		r.AddAdvice(advice)
	}
}

func (r *DisallowFuncAndCalculationsRule) enterAnalyticWindowedFunc(ctx *parser.ANALYTIC_WINDOWED_FUNCContext) {
	if advice := r.generateAdviceOnFunctionUsing(ctx.BaseParserRuleContext); advice != nil {
		r.AddAdvice(advice)
	}
}

func (r *DisallowFuncAndCalculationsRule) enterScalarFunction(ctx *parser.SCALAR_FUNCTIONContext) {
	if advice := r.generateAdviceOnFunctionUsing(ctx.BaseParserRuleContext); advice != nil {
		r.AddAdvice(advice)
	}
}

func (r *DisallowFuncAndCalculationsRule) enterFreeText(ctx *parser.FREE_TEXTContext) {
	if advice := r.generateAdviceOnFunctionUsing(ctx.BaseParserRuleContext); advice != nil {
		r.AddAdvice(advice)
	}
}

func (r *DisallowFuncAndCalculationsRule) enterPartitionFunc(ctx *parser.PARTITION_FUNCContext) {
	if advice := r.generateAdviceOnFunctionUsing(ctx.BaseParserRuleContext); advice != nil {
		r.AddAdvice(advice)
	}
}

func (r *DisallowFuncAndCalculationsRule) enterHierarchyIDMethod(ctx *parser.HIERARCHYID_METHODContext) {
	if advice := r.generateAdviceOnFunctionUsing(ctx.BaseParserRuleContext); advice != nil {
		r.AddAdvice(advice)
	}
}

// Performing calculations.
func (r *DisallowFuncAndCalculationsRule) enterExpression(ctx *parser.ExpressionContext) {
	if ctx.GetOp() != nil {
		if advice := r.generateAdviceOnPerformingCalculations(ctx.BaseParserRuleContext); advice != nil {
			r.AddAdvice(advice)
		}
	}
}

func (r *DisallowFuncAndCalculationsRule) enterUnaryOperatorExpression(ctx *parser.Unary_operator_expressionContext) {
	if advice := r.generateAdviceOnPerformingCalculations(ctx.BaseParserRuleContext); advice != nil {
		r.AddAdvice(advice)
	}
}
