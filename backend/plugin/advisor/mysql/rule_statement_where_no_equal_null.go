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
	_ advisor.Advisor = (*StatementWhereNoEqualNullAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.SchemaRuleStatementWhereNoEqualNull, &StatementWhereNoEqualNullAdvisor{})
}

type StatementWhereNoEqualNullAdvisor struct {
}

func (*StatementWhereNoEqualNullAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmtList, ok := checkCtx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parse result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	// Create the rule
	rule := NewStatementWhereNoEqualNullRule(level, string(checkCtx.Rule.Type))

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	for _, stmt := range stmtList {
		rule.SetBaseLine(stmt.BaseLine)
		checker.SetBaseLine(stmt.BaseLine)
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
	}

	return checker.GetAdviceList(), nil
}

// StatementWhereNoEqualNullRule checks for equal NULL in WHERE clause.
type StatementWhereNoEqualNullRule struct {
	BaseRule
	text     string
	isSelect bool
}

// NewStatementWhereNoEqualNullRule creates a new StatementWhereNoEqualNullRule.
func NewStatementWhereNoEqualNullRule(level storepb.Advice_Status, title string) *StatementWhereNoEqualNullRule {
	return &StatementWhereNoEqualNullRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
	}
}

// Name returns the rule name.
func (*StatementWhereNoEqualNullRule) Name() string {
	return "StatementWhereNoEqualNullRule"
}

// OnEnter is called when entering a parse tree node.
func (r *StatementWhereNoEqualNullRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case NodeTypeQuery:
		queryCtx, ok := ctx.(*mysql.QueryContext)
		if !ok {
			return nil
		}
		r.text = queryCtx.GetParser().GetTokenStream().GetTextFromRuleContext(queryCtx)
	case NodeTypeSelectStatement:
		r.isSelect = true
	case NodeTypePrimaryExprCompare:
		r.checkPrimaryExprCompare(ctx.(*mysql.PrimaryExprCompareContext))
	default:
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (r *StatementWhereNoEqualNullRule) OnExit(_ antlr.ParserRuleContext, nodeType string) error {
	if nodeType == NodeTypeSelectStatement {
		r.isSelect = false
	}
	return nil
}

func (r *StatementWhereNoEqualNullRule) checkPrimaryExprCompare(ctx *mysql.PrimaryExprCompareContext) {
	if !r.isSelect {
		return
	}

	compOp := ctx.CompOp()
	// We only check for equal and not equal.
	if compOp == nil || (compOp.EQUAL_OPERATOR() == nil && compOp.NOT_EQUAL_OPERATOR() == nil) {
		return
	}
	if ctx.Predicate() != nil && ctx.Predicate().GetText() == "NULL" {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          code.StatementWhereNoEqualNull.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf("WHERE clause contains equal null: %s", r.text),
			StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + ctx.GetStart().GetLine()),
		})
	}
}
