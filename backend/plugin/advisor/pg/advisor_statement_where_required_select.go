package pg

import (
	"context"
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*StatementWhereRequiredSelectAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.SchemaRuleStatementRequireWhereForSelect, &StatementWhereRequiredSelectAdvisor{})
}

// StatementWhereRequiredSelectAdvisor is the advisor checking for WHERE clause requirement in SELECT statements.
type StatementWhereRequiredSelectAdvisor struct {
}

// Check checks for WHERE clause requirement in SELECT statements.
func (*StatementWhereRequiredSelectAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, err := getANTLRTree(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &statementWhereRequiredSelectRule{
		BaseRule: BaseRule{
			level: level,
			title: string(checkCtx.Rule.Type),
		},
		statementsText: checkCtx.Statements,
	}

	checker := NewGenericChecker([]Rule{rule})
	antlr.ParseTreeWalkerDefault.Walk(checker, tree.Tree)

	return checker.GetAdviceList(), nil
}

type statementWhereRequiredSelectRule struct {
	BaseRule
	statementsText string
}

// Name returns the rule name.
func (*statementWhereRequiredSelectRule) Name() string {
	return "statement.where-required-select"
}

// OnEnter is called when the parser enters a rule context.
func (r *statementWhereRequiredSelectRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case "Selectstmt":
		r.handleSelectstmt(ctx.(*parser.SelectstmtContext))
	case "Select_with_parens":
		r.handleSelectWithParens(ctx.(*parser.Select_with_parensContext))
	default:
		// Do nothing for other node types
	}
	return nil
}

// OnExit is called when the parser exits a rule context.
func (*statementWhereRequiredSelectRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *statementWhereRequiredSelectRule) handleSelectstmt(ctx *parser.SelectstmtContext) {
	r.checkSelect(ctx, ctx.GetStart(), ctx.GetStop(), func() (bool, bool) {
		return r.checkSelectClauses(ctx)
	})
}

func (r *statementWhereRequiredSelectRule) handleSelectWithParens(ctx *parser.Select_with_parensContext) {
	// Skip if this is the top-level statement (already handled by handleSelectstmt)
	if isTopLevel(ctx.GetParent()) {
		return
	}

	r.checkSelect(ctx, ctx.GetStart(), ctx.GetStop(), func() (bool, bool) {
		return r.checkSelectWithParensForWhere(ctx)
	})
}

// checkSelect is a common function to check for WHERE clause requirement
func (r *statementWhereRequiredSelectRule) checkSelect(
	ctx antlr.ParserRuleContext,
	_ antlr.Token,
	_ antlr.Token,
	checkFunc func() (hasWhere bool, hasFrom bool),
) {
	// Check if this SELECT has a WHERE clause and FROM clause
	hasWhere, hasFrom := checkFunc()

	// Allow SELECT queries without a FROM clause to proceed, e.g. SELECT 1
	if !hasFrom {
		return
	}

	// If there's a WHERE clause, all good
	if hasWhere {
		return
	}

	// Always use the full top-level statement text for the error message
	// This matches the legacy behavior where violations in subqueries
	// are reported with the full statement text
	stmtLine := r.findTopLevelLine(ctx)
	stmtText := extractStatementText(r.statementsText, stmtLine, stmtLine)

	r.AddAdvice(&storepb.Advice{
		Status:  r.level,
		Code:    code.StatementNoWhere.Int32(),
		Title:   r.title,
		Content: fmt.Sprintf("\"%s\" requires WHERE clause", stmtText),
		StartPosition: &storepb.Position{
			Line:   int32(stmtLine),
			Column: 0,
		},
	})
}

// findTopLevelLine finds the line number of the top-level statement
func (*statementWhereRequiredSelectRule) findTopLevelLine(ctx antlr.ParserRuleContext) int {
	for ctx != nil {
		if isTopLevel(ctx.GetParent()) {
			return ctx.GetStart().GetLine()
		}
		parent := ctx.GetParent()
		if ruleCtx, ok := parent.(antlr.ParserRuleContext); ok {
			ctx = ruleCtx
		} else {
			break
		}
	}
	return ctx.GetStart().GetLine()
}

// checkSelectWithParensForWhere checks a select_with_parens for WHERE and FROM
func (r *statementWhereRequiredSelectRule) checkSelectWithParensForWhere(ctx parser.ISelect_with_parensContext) (hasWhere bool, hasFrom bool) {
	if ctx == nil {
		return false, false
	}

	// select_with_parens can contain either select_no_parens or another select_with_parens
	if ctx.Select_no_parens() != nil {
		selectNoParens := ctx.Select_no_parens()
		if selectNoParens.Select_clause() != nil {
			return r.checkSelectClause(selectNoParens.Select_clause())
		}
	}

	if ctx.Select_with_parens() != nil {
		return r.checkSelectWithParensForWhere(ctx.Select_with_parens())
	}

	return false, false
}

// checkSelectClauses checks if a SELECT statement has WHERE and FROM clauses
func (r *statementWhereRequiredSelectRule) checkSelectClauses(ctx *parser.SelectstmtContext) (hasWhere bool, hasFrom bool) {
	// Try Select_no_parens first
	if ctx.Select_no_parens() != nil {
		selectNoParens := ctx.Select_no_parens()
		if selectNoParens.Select_clause() != nil {
			return r.checkSelectClause(selectNoParens.Select_clause())
		}
	}

	// Try Select_with_parens
	if ctx.Select_with_parens() != nil {
		// For a selectstmt that directly contains select_with_parens,
		// delegate to the recursive handler
		return r.checkSelectWithParensForWhere(ctx.Select_with_parens())
	}

	return false, false
}

// checkSelectClause checks a select_clause for WHERE and FROM
func (*statementWhereRequiredSelectRule) checkSelectClause(selectClause parser.ISelect_clauseContext) (hasWhere bool, hasFrom bool) {
	if selectClause == nil {
		return false, false
	}

	// Get all simple_select_intersect
	allIntersects := selectClause.AllSimple_select_intersect()
	for _, intersect := range allIntersects {
		if intersect == nil {
			continue
		}
		// Get all simple_select_pramary (note: typo in parser, it's "pramary" not "primary")
		allPrimary := intersect.AllSimple_select_pramary()
		for _, primary := range allPrimary {
			if primary == nil {
				continue
			}
			// Check for WHERE clause
			if primary.Where_clause() != nil {
				hasWhere = true
			}
			// Check for FROM clause
			if primary.From_clause() != nil {
				hasFrom = true
			}
		}
	}

	return hasWhere, hasFrom
}
