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
	_ advisor.Advisor = (*WhereRequirementForUpdateDeleteAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.SchemaRuleStatementRequireWhereForUpdateDelete, &WhereRequirementForUpdateDeleteAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, advisor.SchemaRuleStatementRequireWhereForUpdateDelete, &WhereRequirementForUpdateDeleteAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, advisor.SchemaRuleStatementRequireWhereForUpdateDelete, &WhereRequirementForUpdateDeleteAdvisor{})
}

// WhereRequirementForUpdateDeleteAdvisor is the advisor checking for the WHERE clause requirement for SELECT statements.
type WhereRequirementForUpdateDeleteAdvisor struct {
}

// Check checks for the WHERE clause requirement.
func (*WhereRequirementForUpdateDeleteAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmtList, ok := checkCtx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to StmtNode")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	// Create the rule
	rule := NewWhereRequirementForUpdateDeleteRule(level, string(checkCtx.Rule.Type))

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	for _, stmt := range stmtList {
		rule.SetBaseLine(stmt.BaseLine)
		checker.SetBaseLine(stmt.BaseLine)
		antlr.ParseTreeWalkerDefault.Walk(checker, stmt.Tree)
	}

	return checker.GetAdviceList(), nil
}

// WhereRequirementForUpdateDeleteRule checks for the WHERE clause requirement.
type WhereRequirementForUpdateDeleteRule struct {
	BaseRule
	text string
}

// NewWhereRequirementForUpdateDeleteRule creates a new WhereRequirementForUpdateDeleteRule.
func NewWhereRequirementForUpdateDeleteRule(level storepb.Advice_Status, title string) *WhereRequirementForUpdateDeleteRule {
	return &WhereRequirementForUpdateDeleteRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
	}
}

// Name returns the rule name.
func (*WhereRequirementForUpdateDeleteRule) Name() string {
	return "WhereRequirementForUpdateDeleteRule"
}

// OnEnter is called when entering a parse tree node.
func (r *WhereRequirementForUpdateDeleteRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case NodeTypeQuery:
		queryCtx, ok := ctx.(*mysql.QueryContext)
		if !ok {
			return nil
		}
		r.text = queryCtx.GetParser().GetTokenStream().GetTextFromRuleContext(queryCtx)
	case NodeTypeDeleteStatement:
		r.checkDeleteStatement(ctx.(*mysql.DeleteStatementContext))
	case NodeTypeUpdateStatement:
		r.checkUpdateStatement(ctx.(*mysql.UpdateStatementContext))
	default:
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*WhereRequirementForUpdateDeleteRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *WhereRequirementForUpdateDeleteRule) checkDeleteStatement(ctx *mysql.DeleteStatementContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	if ctx.WhereClause() == nil || ctx.WhereClause().WHERE_SYMBOL() == nil {
		r.handleWhereClause(ctx.GetStart().GetLine())
	}
}

func (r *WhereRequirementForUpdateDeleteRule) checkUpdateStatement(ctx *mysql.UpdateStatementContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	if ctx.WhereClause() == nil || ctx.WhereClause().WHERE_SYMBOL() == nil {
		r.handleWhereClause(ctx.GetStart().GetLine())
	}
}

func (r *WhereRequirementForUpdateDeleteRule) handleWhereClause(lineNumber int) {
	r.AddAdvice(&storepb.Advice{
		Status:        r.level,
		Code:          code.StatementNoWhere.Int32(),
		Title:         r.title,
		Content:       fmt.Sprintf("\"%s\" requires WHERE clause", r.text),
		StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + lineNumber),
	})
}
