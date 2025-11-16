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
	_ advisor.Advisor = (*WhereRequirementForSelectAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, advisor.SchemaRuleStatementRequireWhereForSelect, &WhereRequirementForSelectAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, advisor.SchemaRuleStatementRequireWhereForSelect, &WhereRequirementForSelectAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, advisor.SchemaRuleStatementRequireWhereForSelect, &WhereRequirementForSelectAdvisor{})
}

// WhereRequirementForSelectAdvisor is the advisor checking for the WHERE clause requirement for SELECT statements.
type WhereRequirementForSelectAdvisor struct {
}

// Check checks for the WHERE clause requirement.
func (*WhereRequirementForSelectAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	root, ok := checkCtx.AST.([]*mysqlparser.ParseResult)
	if !ok {
		return nil, errors.Errorf("failed to convert to StmtNode")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	// Create the rule
	rule := NewWhereRequirementForSelectRule(level, string(checkCtx.Rule.Type))

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	for _, stmtNode := range root {
		rule.SetBaseLine(stmtNode.BaseLine)
		checker.SetBaseLine(stmtNode.BaseLine)
		antlr.ParseTreeWalkerDefault.Walk(checker, stmtNode.Tree)
	}

	return checker.GetAdviceList(), nil
}

// WhereRequirementForSelectRule checks for WHERE clause requirement in SELECT.
type WhereRequirementForSelectRule struct {
	BaseRule
	text string
}

// NewWhereRequirementForSelectRule creates a new WhereRequirementForSelectRule.
func NewWhereRequirementForSelectRule(level storepb.Advice_Status, title string) *WhereRequirementForSelectRule {
	return &WhereRequirementForSelectRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
	}
}

// Name returns the rule name.
func (*WhereRequirementForSelectRule) Name() string {
	return "WhereRequirementForSelectRule"
}

// OnEnter is called when entering a parse tree node.
func (r *WhereRequirementForSelectRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case NodeTypeQuery:
		queryCtx, ok := ctx.(*mysql.QueryContext)
		if !ok {
			return nil
		}
		r.text = queryCtx.GetParser().GetTokenStream().GetTextFromRuleContext(queryCtx)
	case NodeTypeQuerySpecification:
		r.checkQuerySpecification(ctx.(*mysql.QuerySpecificationContext))
	default:
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (*WhereRequirementForSelectRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *WhereRequirementForSelectRule) checkQuerySpecification(ctx *mysql.QuerySpecificationContext) {
	// Allow SELECT queries without a FROM clause to proceed, e.g. SELECT 1.
	if ctx.FromClause() == nil {
		return
	}
	if ctx.WhereClause() == nil || ctx.WhereClause().WHERE_SYMBOL() == nil {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          code.StatementNoWhere.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf("\"%s\" requires WHERE clause", r.text),
			StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + ctx.GetStart().GetLine()),
		})
	}
}
