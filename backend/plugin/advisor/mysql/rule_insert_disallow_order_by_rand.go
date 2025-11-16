package mysql

import (
	"context"
	"fmt"
	"strings"

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
	_ advisor.Advisor = (*InsertDisallowOrderByRandAdvisor)(nil)
)

const RandFn = "rand()"

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.SchemaRuleStatementInsertDisallowOrderByRand, &InsertDisallowOrderByRandAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, advisor.SchemaRuleStatementInsertDisallowOrderByRand, &InsertDisallowOrderByRandAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, advisor.SchemaRuleStatementInsertDisallowOrderByRand, &InsertDisallowOrderByRandAdvisor{})
}

// InsertDisallowOrderByRandAdvisor is the advisor checking for to disallow order by rand in INSERT statements.
type InsertDisallowOrderByRandAdvisor struct {
}

// Check checks for to disallow order by rand in INSERT statements.
func (*InsertDisallowOrderByRandAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmtList, ok := checkCtx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parse result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	// Create the rule
	rule := NewInsertDisallowOrderByRandRule(level, string(checkCtx.Rule.Type))

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	for _, stmt := range stmtList {
		rule.SetBaseLine(stmt.BaseLine)
		checker.SetBaseLine(stmt.BaseLine)
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
	}

	return checker.GetAdviceList(), nil
}

// InsertDisallowOrderByRandRule checks for to disallow order by rand in INSERT statements.
type InsertDisallowOrderByRandRule struct {
	BaseRule
	isInsertStmt bool
	text         string
}

// NewInsertDisallowOrderByRandRule creates a new InsertDisallowOrderByRandRule.
func NewInsertDisallowOrderByRandRule(level storepb.Advice_Status, title string) *InsertDisallowOrderByRandRule {
	return &InsertDisallowOrderByRandRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
	}
}

// Name returns the rule name.
func (*InsertDisallowOrderByRandRule) Name() string {
	return "InsertDisallowOrderByRandRule"
}

// OnEnter is called when entering a parse tree node.
func (r *InsertDisallowOrderByRandRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case NodeTypeQuery:
		queryCtx, ok := ctx.(*mysql.QueryContext)
		if !ok {
			return nil
		}
		r.text = queryCtx.GetParser().GetTokenStream().GetTextFromRuleContext(queryCtx)
	case NodeTypeInsertStatement:
		insertCtx, ok := ctx.(*mysql.InsertStatementContext)
		if !ok {
			return nil
		}
		if mysqlparser.IsTopMySQLRule(&insertCtx.BaseParserRuleContext) {
			if insertCtx.InsertQueryExpression() != nil {
				r.isInsertStmt = true
			}
		}
	case NodeTypeQueryExpression:
		r.checkQueryExpression(ctx.(*mysql.QueryExpressionContext))
	default:
		// Ignore other node types
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (r *InsertDisallowOrderByRandRule) OnExit(_ antlr.ParserRuleContext, nodeType string) error {
	if nodeType == NodeTypeInsertStatement {
		r.isInsertStmt = false
	}
	return nil
}

func (r *InsertDisallowOrderByRandRule) checkQueryExpression(ctx *mysql.QueryExpressionContext) {
	if !r.isInsertStmt {
		return
	}

	if ctx.OrderClause() == nil || ctx.OrderClause().OrderList() == nil {
		return
	}

	for _, expr := range ctx.OrderClause().OrderList().AllOrderExpression() {
		text := expr.GetText()
		if strings.EqualFold(text, RandFn) {
			r.AddAdvice(&storepb.Advice{
				Status:        r.level,
				Code:          code.InsertUseOrderByRand.Int32(),
				Title:         r.title,
				Content:       fmt.Sprintf("\"%s\" uses ORDER BY RAND in the INSERT statement", r.text),
				StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + ctx.GetStart().GetLine()),
			})
		}
	}
}
