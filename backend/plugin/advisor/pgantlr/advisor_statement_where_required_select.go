package pgantlr

import (
	"context"
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
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

	checker := &statementWhereRequiredSelectChecker{
		BasePostgreSQLParserListener: &parser.BasePostgreSQLParserListener{},
		level:                        level,
		title:                        string(checkCtx.Rule.Type),
		statementsText:               checkCtx.Statements,
	}

	antlr.ParseTreeWalkerDefault.Walk(checker, tree.Tree)

	return checker.adviceList, nil
}

type statementWhereRequiredSelectChecker struct {
	*parser.BasePostgreSQLParserListener

	adviceList     []*storepb.Advice
	level          storepb.Advice_Status
	title          string
	statementsText string
}

// EnterSelectstmt handles SELECT statements (including subqueries)
func (c *statementWhereRequiredSelectChecker) EnterSelectstmt(ctx *parser.SelectstmtContext) {
	c.checkSelect(ctx, ctx.GetStart(), ctx.GetStop(), func() (bool, bool) {
		return c.checkSelectClauses(ctx)
	})
}

// EnterSelect_with_parens handles SELECT statements in parentheses (subqueries)
func (c *statementWhereRequiredSelectChecker) EnterSelect_with_parens(ctx *parser.Select_with_parensContext) {
	// Skip if this is the top-level statement (already handled by EnterSelectstmt)
	if isTopLevel(ctx.GetParent()) {
		return
	}

	c.checkSelect(ctx, ctx.GetStart(), ctx.GetStop(), func() (bool, bool) {
		return c.checkSelectWithParensForWhere(ctx)
	})
}

// checkSelect is a common function to check for WHERE clause requirement
func (c *statementWhereRequiredSelectChecker) checkSelect(
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
	stmtLine := c.findTopLevelLine(ctx)
	stmtText := extractStatementText(c.statementsText, stmtLine, stmtLine)

	c.adviceList = append(c.adviceList, &storepb.Advice{
		Status:  c.level,
		Code:    advisor.StatementNoWhere.Int32(),
		Title:   c.title,
		Content: fmt.Sprintf("\"%s\" requires WHERE clause", stmtText),
		StartPosition: &storepb.Position{
			Line:   int32(stmtLine),
			Column: 0,
		},
	})
}

// findTopLevelLine finds the line number of the top-level statement
func (*statementWhereRequiredSelectChecker) findTopLevelLine(ctx antlr.ParserRuleContext) int {
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
func (c *statementWhereRequiredSelectChecker) checkSelectWithParensForWhere(ctx parser.ISelect_with_parensContext) (hasWhere bool, hasFrom bool) {
	if ctx == nil {
		return false, false
	}

	// select_with_parens can contain either select_no_parens or another select_with_parens
	if ctx.Select_no_parens() != nil {
		selectNoParens := ctx.Select_no_parens()
		if selectNoParens.Select_clause() != nil {
			return c.checkSelectClause(selectNoParens.Select_clause())
		}
	}

	if ctx.Select_with_parens() != nil {
		return c.checkSelectWithParensForWhere(ctx.Select_with_parens())
	}

	return false, false
}

// checkSelectClauses checks if a SELECT statement has WHERE and FROM clauses
func (c *statementWhereRequiredSelectChecker) checkSelectClauses(ctx *parser.SelectstmtContext) (hasWhere bool, hasFrom bool) {
	// Try Select_no_parens first
	if ctx.Select_no_parens() != nil {
		selectNoParens := ctx.Select_no_parens()
		if selectNoParens.Select_clause() != nil {
			return c.checkSelectClause(selectNoParens.Select_clause())
		}
	}

	// Try Select_with_parens
	if ctx.Select_with_parens() != nil {
		// For a selectstmt that directly contains select_with_parens,
		// delegate to the recursive handler
		return c.checkSelectWithParensForWhere(ctx.Select_with_parens())
	}

	return false, false
}

// checkSelectClause checks a select_clause for WHERE and FROM
func (*statementWhereRequiredSelectChecker) checkSelectClause(selectClause parser.ISelect_clauseContext) (hasWhere bool, hasFrom bool) {
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
