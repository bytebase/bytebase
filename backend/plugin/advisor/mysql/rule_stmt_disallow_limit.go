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
	_ advisor.Advisor = (*StatementDisallowLimitAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.SchemaRuleStatementDisallowLimit, &StatementDisallowLimitAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, advisor.SchemaRuleStatementDisallowLimit, &StatementDisallowLimitAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, advisor.SchemaRuleStatementDisallowLimit, &StatementDisallowLimitAdvisor{})
}

// StatementDisallowLimitAdvisor is the advisor checking for no LIMIT clause in INSERT/UPDATE statement.
type StatementDisallowLimitAdvisor struct {
}

// Check checks for no LIMIT clause in INSERT/UPDATE statement.
func (*StatementDisallowLimitAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmtList, ok := checkCtx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to mysql parser result")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	// Create the rule
	rule := NewStatementDisallowLimitRule(level, string(checkCtx.Rule.Type))

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	for _, stmt := range stmtList {
		rule.SetBaseLine(stmt.BaseLine)
		checker.SetBaseLine(stmt.BaseLine)
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
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
