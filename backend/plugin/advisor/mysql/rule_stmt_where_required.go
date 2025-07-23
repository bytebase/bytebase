package mysql

import (
	"context"
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	mysql "github.com/bytebase/mysql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

var (
	_ advisor.Advisor = (*WhereRequirementAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.MySQLWhereRequirement, &WhereRequirementAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, advisor.MySQLWhereRequirement, &WhereRequirementAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, advisor.MySQLWhereRequirement, &WhereRequirementAdvisor{})
}

// WhereRequirementAdvisor is the advisor checking for the WHERE clause requirement.
type WhereRequirementAdvisor struct {
}

// Check checks for the WHERE clause requirement.
func (*WhereRequirementAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	root, ok := checkCtx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to StmtNode")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	// Create the rule
	rule := NewWhereRequirementRule(level, string(checkCtx.Rule.Type))

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	for _, stmtNode := range root {
		rule.SetBaseLine(stmtNode.BaseLine)
		checker.SetBaseLine(stmtNode.BaseLine)
		antlr.ParseTreeWalkerDefault.Walk(checker, stmtNode.Tree)
	}

	return checker.GetAdviceList(), nil
}

// WhereRequirementRule checks for the WHERE clause requirement.
type WhereRequirementRule struct {
	BaseRule
	text string
}

// NewWhereRequirementRule creates a new WhereRequirementRule.
func NewWhereRequirementRule(level storepb.Advice_Status, title string) *WhereRequirementRule {
	return &WhereRequirementRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
	}
}

// Name returns the rule name.
func (*WhereRequirementRule) Name() string {
	return "WhereRequirementRule"
}

// OnEnter is called when entering a parse tree node.
func (r *WhereRequirementRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
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
	case NodeTypeQuerySpecification:
		r.checkQuerySpecification(ctx.(*mysql.QuerySpecificationContext))
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*WhereRequirementRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *WhereRequirementRule) checkDeleteStatement(ctx *mysql.DeleteStatementContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	if ctx.WhereClause() == nil || ctx.WhereClause().WHERE_SYMBOL() == nil {
		r.handleWhereClause(ctx.GetStart().GetLine())
	}
}

func (r *WhereRequirementRule) checkUpdateStatement(ctx *mysql.UpdateStatementContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	if ctx.WhereClause() == nil || ctx.WhereClause().WHERE_SYMBOL() == nil {
		r.handleWhereClause(ctx.GetStart().GetLine())
	}
}

func (r *WhereRequirementRule) checkQuerySpecification(ctx *mysql.QuerySpecificationContext) {
	// Allow SELECT queries without a FROM clause to proceed, e.g. SELECT 1.
	if ctx.FromClause() == nil {
		return
	}
	if ctx.WhereClause() == nil || ctx.WhereClause().WHERE_SYMBOL() == nil {
		r.handleWhereClause(ctx.GetStart().GetLine())
	}
}

func (r *WhereRequirementRule) handleWhereClause(lineNumber int) {
	r.AddAdvice(&storepb.Advice{
		Status:        r.level,
		Code:          advisor.StatementNoWhere.Int32(),
		Title:         r.title,
		Content:       fmt.Sprintf("\"%s\" requires WHERE clause", r.text),
		StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + lineNumber),
	})
}
