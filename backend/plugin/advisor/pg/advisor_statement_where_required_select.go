package pg

import (
	"context"
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

var (
	_ advisor.Advisor = (*StatementWhereRequiredSelectAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_SELECT, &StatementWhereRequiredSelectAdvisor{})
}

// StatementWhereRequiredSelectAdvisor is the advisor checking for WHERE clause requirement in SELECT statements.
type StatementWhereRequiredSelectAdvisor struct {
}

// Check checks for WHERE clause requirement in SELECT statements.
func (*StatementWhereRequiredSelectAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	var adviceList []*storepb.Advice
	for _, stmtInfo := range checkCtx.ParsedStatements {
		if stmtInfo.AST == nil {
			continue
		}
		antlrAST, ok := base.GetANTLRAST(stmtInfo.AST)
		if !ok {
			continue
		}
		rule := &statementWhereRequiredSelectRule{
			BaseRule: BaseRule{
				level: level,
				title: checkCtx.Rule.Type.String(),
			},
			tokens: antlrAST.Tokens,
		}
		rule.SetBaseLine(stmtInfo.BaseLine())

		checker := NewGenericChecker([]Rule{rule})
		antlr.ParseTreeWalkerDefault.Walk(checker, antlrAST.Tree)
		adviceList = append(adviceList, checker.GetAdviceList()...)
	}

	return adviceList, nil
}

type statementWhereRequiredSelectRule struct {
	BaseRule
	tokens *antlr.CommonTokenStream
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
	r.checkSelect(ctx, func() (bool, bool) {
		return r.checkSelectClauses(ctx)
	})
}

func (r *statementWhereRequiredSelectRule) handleSelectWithParens(ctx *parser.Select_with_parensContext) {
	// Skip if this is the top-level statement (already handled by handleSelectstmt)
	if isTopLevel(ctx.GetParent()) {
		return
	}

	r.checkSelect(ctx, func() (bool, bool) {
		return r.checkSelectWithParensForWhere(ctx)
	})
}

// checkSelect is a common function to check for WHERE clause requirement
func (r *statementWhereRequiredSelectRule) checkSelect(
	ctx antlr.ParserRuleContext,
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

	// Get clean statement text from token stream
	stmtText := getTextFromTokens(r.tokens, ctx)
	if stmtText == "" {
		stmtText = "<unknown statement>"
	}

	r.AddAdvice(&storepb.Advice{
		Status:  r.level,
		Code:    code.StatementNoWhere.Int32(),
		Title:   r.title,
		Content: fmt.Sprintf("\"%s\" requires WHERE clause", stmtText),
		StartPosition: &storepb.Position{
			Line:   int32(ctx.GetStart().GetLine()),
			Column: 0,
		},
	})
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
