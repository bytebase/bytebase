package mysql

import (
	"context"
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	"github.com/bytebase/parser/mysql"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

var (
	_ advisor.Advisor = (*StatementDisallowLimitAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_STATEMENT_DISALLOW_LIMIT, &StatementDisallowLimitAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_STATEMENT_DISALLOW_LIMIT, &StatementDisallowLimitAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_STATEMENT_DISALLOW_LIMIT, &StatementDisallowLimitAdvisor{})
}

// StatementDisallowLimitAdvisor is the advisor checking for no LIMIT clause in INSERT/UPDATE statement.
type StatementDisallowLimitAdvisor struct {
}

// Check checks for no LIMIT clause in INSERT/UPDATE statement.
func (*StatementDisallowLimitAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	// Create the rule
	rule := NewStatementDisallowLimitRule(level, checkCtx.Rule.Type.String())

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

// StatementDisallowLimitRule checks for no LIMIT clause in INSERT/UPDATE statement.
type StatementDisallowLimitRule struct {
	BaseRule
	isInsertStmt bool
	text         string
	line         int
}

// NewStatementDisallowLimitRule creates a new StatementDisallowLimitRule.
func NewStatementDisallowLimitRule(level storepb.Advice_Status, title string) *StatementDisallowLimitRule {
	return &StatementDisallowLimitRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
	}
}

// Name returns the rule name.
func (*StatementDisallowLimitRule) Name() string {
	return "StatementDisallowLimitRule"
}

// OnEnter is called when entering a parse tree node.
func (r *StatementDisallowLimitRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case NodeTypeQuery:
		if queryCtx, ok := ctx.(*mysql.QueryContext); ok {
			r.text = queryCtx.GetParser().GetTokenStream().GetTextFromRuleContext(queryCtx)
		}
	case NodeTypeDeleteStatement:
		r.checkDeleteStatement(ctx.(*mysql.DeleteStatementContext))
	case NodeTypeUpdateStatement:
		r.checkUpdateStatement(ctx.(*mysql.UpdateStatementContext))
	case NodeTypeInsertStatement:
		r.isInsertStmt = true
	case NodeTypeQueryExpression:
		r.checkQueryExpression(ctx.(*mysql.QueryExpressionContext))
	default:
		// Ignore other node types
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (r *StatementDisallowLimitRule) OnExit(_ antlr.ParserRuleContext, nodeType string) error {
	if nodeType == NodeTypeInsertStatement {
		r.isInsertStmt = false
	}
	return nil
}

func (r *StatementDisallowLimitRule) checkDeleteStatement(ctx *mysql.DeleteStatementContext) {
	if ctx.SimpleLimitClause() != nil && ctx.SimpleLimitClause().LIMIT_SYMBOL() != nil {
		r.handleLimitClause(code.DeleteUseLimit, ctx.GetStart().GetLine())
	}
}

func (r *StatementDisallowLimitRule) checkUpdateStatement(ctx *mysql.UpdateStatementContext) {
	if ctx.SimpleLimitClause() != nil && ctx.SimpleLimitClause().LIMIT_SYMBOL() != nil {
		r.handleLimitClause(code.UpdateUseLimit, ctx.GetStart().GetLine())
	}
}

func (r *StatementDisallowLimitRule) checkQueryExpression(ctx *mysql.QueryExpressionContext) {
	if !r.isInsertStmt {
		return
	}
	if ctx.LimitClause() != nil && ctx.LimitClause().LIMIT_SYMBOL() != nil {
		r.handleLimitClause(code.InsertUseLimit, ctx.GetStart().GetLine())
	}
}

func (r *StatementDisallowLimitRule) handleLimitClause(code code.Code, lineNumber int) {
	r.AddAdvice(&storepb.Advice{
		Status:        r.level,
		Code:          code.Int32(),
		Title:         r.title,
		Content:       fmt.Sprintf("LIMIT clause is forbidden in INSERT, UPDATE and DELETE statement, but \"%s\" uses", r.text),
		StartPosition: common.ConvertANTLRLineToPosition(r.line + lineNumber),
	})
}
