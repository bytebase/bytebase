package pg

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

var (
	_ advisor.Advisor = (*StatementMaximumLimitValueAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_MAXIMUM_LIMIT_VALUE, &StatementMaximumLimitValueAdvisor{})
}

// StatementMaximumLimitValueAdvisor is the advisor checking for maximum LIMIT value.
type StatementMaximumLimitValueAdvisor struct {
}

// Check checks for maximum LIMIT value.
func (*StatementMaximumLimitValueAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	numberPayload := checkCtx.Rule.GetNumberPayload()
	if numberPayload == nil {
		return nil, errors.New("number_payload is required for this rule")
	}

	rule := &statementMaximumLimitValueRule{
		BaseRule: BaseRule{
			level: level,
			title: checkCtx.Rule.Type.String(),
		},
		limitMaxValue: int(numberPayload.Number),
	}

	checker := NewGenericChecker([]Rule{rule})

	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		antlrAST, ok := base.GetANTLRAST(stmt.AST)
		if !ok {
			continue
		}
		rule.SetBaseLine(stmt.BaseLine())
		checker.SetBaseLine(stmt.BaseLine())
		antlr.ParseTreeWalkerDefault.Walk(checker, antlrAST.Tree)
	}

	return checker.GetAdviceList(), nil
}

type statementMaximumLimitValueRule struct {
	BaseRule
	limitMaxValue int
}

// Name returns the rule name.
func (*statementMaximumLimitValueRule) Name() string {
	return "statement_maximum_limit_value"
}

// OnEnter handles node entry events.
func (r *statementMaximumLimitValueRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	if nodeType != "Selectstmt" {
		return nil
	}

	selectstmtCtx, ok := ctx.(*parser.SelectstmtContext)
	if !ok {
		return nil
	}

	if !isTopLevel(selectstmtCtx.GetParent()) {
		return nil
	}

	// Check for LIMIT clause in the SELECT statement
	limitValue := r.extractLimitValue(selectstmtCtx)
	if limitValue > 0 && limitValue > r.limitMaxValue {
		r.AddAdvice(&storepb.Advice{
			Status:  r.level,
			Code:    code.StatementExceedMaximumLimitValue.Int32(),
			Title:   r.title,
			Content: fmt.Sprintf("The limit value %d exceeds the maximum allowed value %d", limitValue, r.limitMaxValue),
			StartPosition: &storepb.Position{
				Line:   int32(selectstmtCtx.GetStart().GetLine()),
				Column: 0,
			},
		})
	}

	return nil
}

// OnExit handles node exit events.
func (*statementMaximumLimitValueRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

// extractLimitValue extracts the LIMIT value from a SELECT statement.
// Returns 0 if no LIMIT clause is found or if LIMIT is not a simple integer.
func (r *statementMaximumLimitValueRule) extractLimitValue(ctx *parser.SelectstmtContext) int {
	if ctx == nil {
		return 0
	}

	// Try select_no_parens
	if ctx.Select_no_parens() != nil {
		return r.extractLimitFromSelectNoParens(ctx.Select_no_parens())
	}

	// Try select_with_parens
	if ctx.Select_with_parens() != nil {
		return r.extractLimitFromSelectWithParens(ctx.Select_with_parens())
	}

	return 0
}

// extractLimitFromSelectNoParens extracts LIMIT value from select_no_parens.
func (r *statementMaximumLimitValueRule) extractLimitFromSelectNoParens(ctx parser.ISelect_no_parensContext) int {
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
		return r.extractLimitFromSelectLimit(selectLimit)
	}

	return 0
}

// extractLimitFromSelectWithParens extracts LIMIT value from select_with_parens.
func (r *statementMaximumLimitValueRule) extractLimitFromSelectWithParens(ctx parser.ISelect_with_parensContext) int {
	if ctx == nil {
		return 0
	}

	// Recursively check inner select statements
	if ctx.Select_no_parens() != nil {
		return r.extractLimitFromSelectNoParens(ctx.Select_no_parens())
	}

	if ctx.Select_with_parens() != nil {
		return r.extractLimitFromSelectWithParens(ctx.Select_with_parens())
	}

	return 0
}

// extractLimitFromSelectLimit extracts LIMIT value from select_limit clause.
func (r *statementMaximumLimitValueRule) extractLimitFromSelectLimit(ctx parser.ISelect_limitContext) int {
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
			return r.extractLimitValueFromLimitValue(limitClause.Select_limit_value())
		}
	}

	// Check for offset_clause (which may have FETCH FIRST/NEXT)
	if ctx.Offset_clause() != nil {
		offsetClause := ctx.Offset_clause()
		// Check for FETCH FIRST/NEXT syntax
		if offsetClause.Select_fetch_first_value() != nil {
			return r.extractLimitValueFromFetchFirst(offsetClause.Select_fetch_first_value())
		}
	}

	return 0
}

// extractLimitValueFromLimitValue extracts integer value from select_limit_value.
func (r *statementMaximumLimitValueRule) extractLimitValueFromLimitValue(ctx parser.ISelect_limit_valueContext) int {
	if ctx == nil {
		return 0
	}

	// Check for ALL keyword (means no limit)
	if ctx.ALL() != nil {
		return 0
	}

	// Check for a_expr which contains the actual value
	if ctx.A_expr() != nil {
		return r.extractIntFromAExpr(ctx.A_expr())
	}

	return 0
}

// extractLimitValueFromFetchFirst extracts integer value from FETCH FIRST/NEXT clause.
func (r *statementMaximumLimitValueRule) extractLimitValueFromFetchFirst(ctx parser.ISelect_fetch_first_valueContext) int {
	if ctx == nil {
		return 0
	}

	// Try to extract the numeric value from the text
	text := ctx.GetText()
	return r.parseIntFromText(text)
}

// extractIntFromAExpr attempts to extract an integer constant from an a_expr.
func (r *statementMaximumLimitValueRule) extractIntFromAExpr(ctx parser.IA_exprContext) int {
	if ctx == nil {
		return 0
	}

	// Try to parse the text directly - this handles simple numeric literals
	text := ctx.GetText()
	return r.parseIntFromText(text)
}

// parseIntFromText attempts to parse an integer from text.
// This handles simple numeric literals and returns 0 if parsing fails.
func (*statementMaximumLimitValueRule) parseIntFromText(text string) int {
	// Clean up the text (remove whitespace)
	text = strings.TrimSpace(text)

	// Try to parse as integer
	val, err := strconv.Atoi(text)
	if err != nil {
		return 0
	}

	return val
}
