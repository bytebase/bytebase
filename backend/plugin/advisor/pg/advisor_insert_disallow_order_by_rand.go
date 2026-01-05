package pg

import (
	"context"
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"

	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

var (
	_ advisor.Advisor = (*InsertDisallowOrderByRandAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_INSERT_DISALLOW_ORDER_BY_RAND, &InsertDisallowOrderByRandAdvisor{})
}

// InsertDisallowOrderByRandAdvisor is the advisor checking for to disallow order by rand in INSERT statements.
type InsertDisallowOrderByRandAdvisor struct {
}

// Check checks for to disallow order by rand in INSERT statements.
func (*InsertDisallowOrderByRandAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
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
		rule := &insertDisallowOrderByRandRule{
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

type insertDisallowOrderByRandRule struct {
	BaseRule

	tokens *antlr.CommonTokenStream
}

func (*insertDisallowOrderByRandRule) Name() string {
	return "insert_disallow_order_by_rand"
}

func (r *insertDisallowOrderByRandRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case "Insertstmt":
		r.handleInsertstmt(ctx)
	default:
		// Do nothing for other node types
	}
	return nil
}

func (*insertDisallowOrderByRandRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *insertDisallowOrderByRandRule) handleInsertstmt(ctx antlr.ParserRuleContext) {
	insertstmtCtx, ok := ctx.(*parser.InsertstmtContext)
	if !ok {
		return
	}

	if !isTopLevel(insertstmtCtx.GetParent()) {
		return
	}

	// Check if this is INSERT...SELECT
	if insertstmtCtx.Insert_rest() == nil || insertstmtCtx.Insert_rest().Selectstmt() == nil {
		return
	}

	// Check for ORDER BY random() in the SELECT statement
	if r.hasOrderByRandom(insertstmtCtx.Insert_rest().Selectstmt()) {
		r.AddAdvice(&storepb.Advice{
			Status:  r.level,
			Code:    code.InsertUseOrderByRand.Int32(),
			Title:   r.title,
			Content: fmt.Sprintf("The INSERT statement uses ORDER BY random() or random_between(), related statement \"%s\"", getTextFromTokens(r.tokens, insertstmtCtx)),
			StartPosition: &storepb.Position{
				Line:   int32(insertstmtCtx.GetStart().GetLine()),
				Column: 0,
			},
		})
	}
}

// hasOrderByRandom checks if a SELECT statement has ORDER BY random() or random_between()
func (*insertDisallowOrderByRandRule) hasOrderByRandom(selectCtx parser.ISelectstmtContext) bool {
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
