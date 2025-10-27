package pgantlr

import (
	"context"
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"

	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
)

var (
	_ advisor.Advisor = (*InsertDisallowOrderByRandAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.SchemaRuleStatementInsertDisallowOrderByRand, &InsertDisallowOrderByRandAdvisor{})
}

// InsertDisallowOrderByRandAdvisor is the advisor checking for to disallow order by rand in INSERT statements.
type InsertDisallowOrderByRandAdvisor struct {
}

// Check checks for to disallow order by rand in INSERT statements.
func (*InsertDisallowOrderByRandAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, err := getANTLRTree(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	checker := &insertDisallowOrderByRandChecker{
		BasePostgreSQLParserListener: &parser.BasePostgreSQLParserListener{},
		level:                        level,
		title:                        string(checkCtx.Rule.Type),
		statementsText:               checkCtx.Statements,
	}

	antlr.ParseTreeWalkerDefault.Walk(checker, tree.Tree)

	return checker.adviceList, nil
}

type insertDisallowOrderByRandChecker struct {
	*parser.BasePostgreSQLParserListener

	adviceList     []*storepb.Advice
	level          storepb.Advice_Status
	title          string
	statementsText string
}

func (c *insertDisallowOrderByRandChecker) EnterInsertstmt(ctx *parser.InsertstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Check if this is INSERT...SELECT
	if ctx.Insert_rest() == nil || ctx.Insert_rest().Selectstmt() == nil {
		return
	}

	// Check for ORDER BY random() in the SELECT statement
	if c.hasOrderByRandom(ctx.Insert_rest().Selectstmt()) {
		// Extract the statement text from the original statements
		stmtText := extractStatementText(c.statementsText, ctx.GetStart().GetLine(), ctx.GetStop().GetLine())

		c.adviceList = append(c.adviceList, &storepb.Advice{
			Status:  c.level,
			Code:    advisor.InsertUseOrderByRand.Int32(),
			Title:   c.title,
			Content: fmt.Sprintf("The INSERT statement uses ORDER BY random() or random_between(), related statement \"%s\"", stmtText),
			StartPosition: &storepb.Position{
				Line:   int32(ctx.GetStart().GetLine()),
				Column: 0,
			},
		})
	}
}

// hasOrderByRandom checks if a SELECT statement has ORDER BY random() or random_between()
func (*insertDisallowOrderByRandChecker) hasOrderByRandom(selectCtx parser.ISelectstmtContext) bool {
	if selectCtx == nil {
		return false
	}

	// Check both Select_no_parens and Select_with_parens
	if selectCtx.Select_no_parens() != nil {
		if hasOrderByRandomInSelect(selectCtx.Select_no_parens()) {
			return true
		}
	}

	if selectCtx.Select_with_parens() != nil {
		// Select_with_parens might contain another Selectstmt
		// We need to recursively check
		return hasOrderByRandomInParens(selectCtx.Select_with_parens())
	}

	return false
}

// hasOrderByRandomInSelect checks Select_no_parens for ORDER BY random()
func hasOrderByRandomInSelect(selectCtx parser.ISelect_no_parensContext) bool {
	if selectCtx == nil {
		return false
	}

	// Check Opt_sort_clause (ORDER BY clause)
	if selectCtx.Opt_sort_clause() != nil && selectCtx.Opt_sort_clause().Sort_clause() != nil {
		sortClause := selectCtx.Opt_sort_clause().Sort_clause()
		// Get all sort by items via Sortby_list
		if sortClause.Sortby_list() != nil {
			allSortBy := sortClause.Sortby_list().AllSortby()
			for _, sortBy := range allSortBy {
				if sortBy.A_expr() != nil {
					text := strings.ToLower(sortBy.A_expr().GetText())
					if strings.Contains(text, "random()") || strings.Contains(text, "random_between()") {
						return true
					}
				}
			}
		}
	}

	return false
}

// hasOrderByRandomInParens checks Select_with_parens recursively
func hasOrderByRandomInParens(selectCtx parser.ISelect_with_parensContext) bool {
	if selectCtx == nil {
		return false
	}

	// Select_with_parens can contain Select_no_parens or another Select_with_parens
	if selectCtx.Select_no_parens() != nil {
		return hasOrderByRandomInSelect(selectCtx.Select_no_parens())
	}

	if selectCtx.Select_with_parens() != nil {
		return hasOrderByRandomInParens(selectCtx.Select_with_parens())
	}

	return false
}
