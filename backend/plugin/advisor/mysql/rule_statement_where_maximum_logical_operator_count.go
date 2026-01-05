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
	_ advisor.Advisor = (*StatementWhereMaximumLogicalOperatorCountAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_STATEMENT_WHERE_MAXIMUM_LOGICAL_OPERATOR_COUNT, &StatementWhereMaximumLogicalOperatorCountAdvisor{})
}

type StatementWhereMaximumLogicalOperatorCountAdvisor struct {
}

func (*StatementWhereMaximumLogicalOperatorCountAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	numberPayload := checkCtx.Rule.GetNumberPayload()
	if numberPayload == nil {
		return nil, errors.New("number_payload is required for this rule")
	}

	var allAdvice []*storepb.Advice
	for _, stmt := range checkCtx.ParsedStatements {
		// Create the rule for each statement
		rule := NewStatementWhereMaximumLogicalOperatorCountRule(level, checkCtx.Rule.Type.String(), int(numberPayload.Number))

		// Create the generic checker with the rule
		checker := NewGenericChecker([]Rule{rule})

		rule.SetBaseLine(stmt.BaseLine())
		checker.SetBaseLine(stmt.BaseLine())
		rule.resetForStatement()
		if stmt.AST == nil {
			continue
		}
		antlrAST, ok := base.GetANTLRAST(stmt.AST)
		if !ok {
			continue
		}
		antlr.ParseTreeWalkerDefault.Walk(checker, antlrAST.Tree)

		// Check OR conditions after walking the tree
		rule.checkOrConditions()

		allAdvice = append(allAdvice, checker.GetAdviceList()...)
	}

	return allAdvice, nil
}

// StatementWhereMaximumLogicalOperatorCountRule checks for maximum logical operators in WHERE.
type StatementWhereMaximumLogicalOperatorCountRule struct {
	BaseRule
	text              string
	maximum           int
	reported          bool
	depth             int
	inPredicateExprIn bool
	maxOrCount        int
	maxOrCountLine    int
}

// NewStatementWhereMaximumLogicalOperatorCountRule creates a new StatementWhereMaximumLogicalOperatorCountRule.
func NewStatementWhereMaximumLogicalOperatorCountRule(level storepb.Advice_Status, title string, maximum int) *StatementWhereMaximumLogicalOperatorCountRule {
	return &StatementWhereMaximumLogicalOperatorCountRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
		maximum: maximum,
	}
}

// Name returns the rule name.
func (*StatementWhereMaximumLogicalOperatorCountRule) Name() string {
	return "StatementWhereMaximumLogicalOperatorCountRule"
}

// resetForStatement resets state for a new statement.
func (r *StatementWhereMaximumLogicalOperatorCountRule) resetForStatement() {
	r.reported = false
	r.depth = 0
	r.inPredicateExprIn = false
	r.maxOrCount = 0
	r.maxOrCountLine = 0
}

// OnEnter is called when entering a parse tree node.
func (r *StatementWhereMaximumLogicalOperatorCountRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case NodeTypeQuery:
		queryCtx, ok := ctx.(*mysql.QueryContext)
		if !ok {
			return nil
		}
		r.text = queryCtx.GetParser().GetTokenStream().GetTextFromRuleContext(queryCtx)
	case NodeTypePredicateExprIn:
		r.inPredicateExprIn = true
	case NodeTypeExprList:
		r.checkExprList(ctx.(*mysql.ExprListContext))
	case NodeTypeExprOr:
		r.checkExprOr(ctx.(*mysql.ExprOrContext))
	default:
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (r *StatementWhereMaximumLogicalOperatorCountRule) OnExit(_ antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case NodeTypePredicateExprIn:
		r.inPredicateExprIn = false
	case NodeTypeExprOr:
		r.depth--
	default:
	}
	return nil
}

func (r *StatementWhereMaximumLogicalOperatorCountRule) checkExprList(ctx *mysql.ExprListContext) {
	if r.reported {
		return
	}
	if !r.inPredicateExprIn {
		return
	}

	count := len(ctx.AllExpr())
	if count > r.maximum {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          code.StatementWhereMaximumLogicalOperatorCount.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf("Number of tokens (%d) in IN predicate operation exceeds limit (%d) in statement %q.", count, r.maximum, r.text),
			StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + ctx.GetStart().GetLine()),
		})
	}
}

func (r *StatementWhereMaximumLogicalOperatorCountRule) checkExprOr(ctx *mysql.ExprOrContext) {
	r.depth++
	count := r.depth + 1
	if count > r.maxOrCount {
		r.maxOrCount = count
		r.maxOrCountLine = r.baseLine + ctx.GetStart().GetLine()
	}
}

func (r *StatementWhereMaximumLogicalOperatorCountRule) checkOrConditions() {
	if r.maxOrCount > r.maximum {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          code.StatementWhereMaximumLogicalOperatorCount.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf("Number of tokens (%d) in the OR predicate operation exceeds limit (%d) in statement %q.", r.maxOrCount, r.maximum, r.text),
			StartPosition: common.ConvertANTLRLineToPosition(r.maxOrCountLine),
		})
	}
}
