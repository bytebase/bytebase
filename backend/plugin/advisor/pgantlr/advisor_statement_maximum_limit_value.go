package pgantlr

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
)

var (
	_ advisor.Advisor = (*StatementMaximumLimitValueAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.SchemaRuleStatementMaximumLimitValue, &StatementMaximumLimitValueAdvisor{})
}

// StatementMaximumLimitValueAdvisor is the advisor checking for maximum LIMIT value.
type StatementMaximumLimitValueAdvisor struct {
}

// Check checks for maximum LIMIT value.
func (*StatementMaximumLimitValueAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, err := getANTLRTree(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	payload, err := advisor.UnmarshalNumberTypeRulePayload(checkCtx.Rule.Payload)
	if err != nil {
		return nil, err
	}

	checker := &statementMaximumLimitValueChecker{
		BasePostgreSQLParserListener: &parser.BasePostgreSQLParserListener{},
		level:                        level,
		title:                        string(checkCtx.Rule.Type),
		limitMaxValue:                payload.Number,
	}

	antlr.ParseTreeWalkerDefault.Walk(checker, tree.Tree)

	return checker.adviceList, nil
}

type statementMaximumLimitValueChecker struct {
	*parser.BasePostgreSQLParserListener

	adviceList    []*storepb.Advice
	level         storepb.Advice_Status
	title         string
	limitMaxValue int
}

// EnterSelectstmt handles SELECT statements with LIMIT clauses.
func (c *statementMaximumLimitValueChecker) EnterSelectstmt(ctx *parser.SelectstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Check for LIMIT clause in the SELECT statement
	limitValue := c.extractLimitValue(ctx)
	if limitValue > 0 && limitValue > c.limitMaxValue {
		c.adviceList = append(c.adviceList, &storepb.Advice{
			Status:  c.level,
			Code:    advisor.StatementExceedMaximumLimitValue.Int32(),
			Title:   c.title,
			Content: fmt.Sprintf("The limit value %d exceeds the maximum allowed value %d", limitValue, c.limitMaxValue),
			StartPosition: &storepb.Position{
				Line:   int32(ctx.GetStart().GetLine()),
				Column: 0,
			},
		})
	}
}

// extractLimitValue extracts the LIMIT value from a SELECT statement.
// Returns 0 if no LIMIT clause is found or if LIMIT is not a simple integer.
func (c *statementMaximumLimitValueChecker) extractLimitValue(ctx *parser.SelectstmtContext) int {
	if ctx == nil {
		return 0
	}

	// Try select_no_parens
	if ctx.Select_no_parens() != nil {
		return c.extractLimitFromSelectNoParens(ctx.Select_no_parens())
	}

	// Try select_with_parens
	if ctx.Select_with_parens() != nil {
		return c.extractLimitFromSelectWithParens(ctx.Select_with_parens())
	}

	return 0
}

// extractLimitFromSelectNoParens extracts LIMIT value from select_no_parens.
func (c *statementMaximumLimitValueChecker) extractLimitFromSelectNoParens(ctx parser.ISelect_no_parensContext) int {
	if ctx == nil {
		return 0
	}

	var selectLimit parser.ISelect_limitContext
	if ctx.Select_limit() != nil {
		selectLimit = ctx.Select_limit()
	}
	if ctx.Opt_select_limit() != nil {
		selectLimit = ctx.Opt_select_limit().Select_limit()
	}

	// Check for select_limit directly in select_no_parens
	if selectLimit != nil {
		return c.extractLimitFromSelectLimit(selectLimit)
	}

	return 0
}

// extractLimitFromSelectWithParens extracts LIMIT value from select_with_parens.
func (c *statementMaximumLimitValueChecker) extractLimitFromSelectWithParens(ctx parser.ISelect_with_parensContext) int {
	if ctx == nil {
		return 0
	}

	// Recursively check inner select statements
	if ctx.Select_no_parens() != nil {
		return c.extractLimitFromSelectNoParens(ctx.Select_no_parens())
	}

	if ctx.Select_with_parens() != nil {
		return c.extractLimitFromSelectWithParens(ctx.Select_with_parens())
	}

	return 0
}

// extractLimitFromSelectLimit extracts LIMIT value from select_limit clause.
func (c *statementMaximumLimitValueChecker) extractLimitFromSelectLimit(ctx parser.ISelect_limitContext) int {
	if ctx == nil {
		return 0
	}

	// PostgreSQL supports several LIMIT formats:
	// 1. LIMIT count
	// 2. LIMIT count OFFSET start
	// 3. OFFSET start (without LIMIT)
	// 4. LIMIT ALL

	// Check for limit_clause
	if ctx.Limit_clause() != nil {
		limitClause := ctx.Limit_clause()
		// Get the select_limit_value
		if limitClause.Select_limit_value() != nil {
			return c.extractLimitValueFromLimitValue(limitClause.Select_limit_value())
		}
	}

	// Check for offset_clause (which may have FETCH FIRST/NEXT)
	if ctx.Offset_clause() != nil {
		offsetClause := ctx.Offset_clause()
		// Check for FETCH FIRST/NEXT syntax
		if offsetClause.Select_fetch_first_value() != nil {
			return c.extractLimitValueFromFetchFirst(offsetClause.Select_fetch_first_value())
		}
	}

	return 0
}

// extractLimitValueFromLimitValue extracts integer value from select_limit_value.
func (c *statementMaximumLimitValueChecker) extractLimitValueFromLimitValue(ctx parser.ISelect_limit_valueContext) int {
	if ctx == nil {
		return 0
	}

	// Check for ALL keyword (means no limit)
	if ctx.ALL() != nil {
		return 0
	}

	// Check for a_expr which contains the actual value
	if ctx.A_expr() != nil {
		return c.extractIntFromAExpr(ctx.A_expr())
	}

	return 0
}

// extractLimitValueFromFetchFirst extracts integer value from FETCH FIRST/NEXT clause.
func (c *statementMaximumLimitValueChecker) extractLimitValueFromFetchFirst(ctx parser.ISelect_fetch_first_valueContext) int {
	if ctx == nil {
		return 0
	}

	// Try to extract the numeric value from the text
	text := ctx.GetText()
	return c.parseIntFromText(text)
}

// extractIntFromAExpr attempts to extract an integer constant from an a_expr.
func (c *statementMaximumLimitValueChecker) extractIntFromAExpr(ctx parser.IA_exprContext) int {
	if ctx == nil {
		return 0
	}

	// Try to parse the text directly - this handles simple numeric literals
	text := ctx.GetText()
	return c.parseIntFromText(text)
}

// parseIntFromText attempts to parse an integer from text.
// This handles simple numeric literals and returns 0 if parsing fails.
func (*statementMaximumLimitValueChecker) parseIntFromText(text string) int {
	// Clean up the text (remove whitespace)
	text = strings.TrimSpace(text)

	// Try to parse as integer
	val, err := strconv.Atoi(text)
	if err != nil {
		return 0
	}

	return val
}
