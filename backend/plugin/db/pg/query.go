package pg

import (
	"fmt"

	"github.com/antlr4-go/antlr/v4"

	pg "github.com/bytebase/postgresql-parser"

	pgrawparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
)

func getStatementWithResultLimit(singleStatement string, limitCount int) (string, error) {
	result, err := pgrawparser.ParsePostgreSQL(singleStatement)
	if err != nil {
		return "", err
	}

	listener := &pgRewriter{
		limitCount: limitCount,
	}

	listener.rewriter = *antlr.NewTokenStreamRewriter(result.Tokens)
	antlr.ParseTreeWalkerDefault.Walk(listener, result.Tree)

	res := listener.rewriter.GetTextDefault()

	return res, nil
}

type pgRewriter struct {
	*pg.BasePostgreSQLParserListener

	rewriter   antlr.TokenStreamRewriter
	limitCount int
}

// EnterSelectstmt is called when production selectstmt is entered.
func (r *pgRewriter) EnterSelectstmt(ctx *pg.SelectstmtContext) {
	if ctx.GetParent() != nil {
		if _, ok := ctx.GetParent().(*pg.PreparablestmtContext); ok {
			// select statement in cte, do not process it.
			return
		}
	}

	if ctx.Select_no_parens() != nil {
		r.handleSelectNoParens(ctx.Select_no_parens())
	}
}

func (r *pgRewriter) handleSelectNoParens(ctx pg.ISelect_no_parensContext) {
	// respect original limit.
	if r.checkSelectLimit(ctx.Select_limit()) {
		return
	}
	if ctx.Opt_select_limit() != nil {
		if r.checkSelectLimit(ctx.Opt_select_limit().Select_limit()) {
			return
		}
	}

	if ctx.For_locking_clause() != nil {
		r.rewriter.InsertAfterDefault(ctx.For_locking_clause().GetStop().GetTokenIndex(), fmt.Sprintf(" LIMIT %d", r.limitCount))
		return
	}
	if ctx.Opt_for_locking_clause() != nil {
		r.rewriter.InsertBeforeDefault(ctx.Opt_for_locking_clause().GetStop().GetTokenIndex(), fmt.Sprintf(" LIMTI %d ", r.limitCount))
		return
	}

	if ctx.Opt_sort_clause() != nil {
		r.rewriter.InsertAfterDefault(ctx.Opt_sort_clause().GetStop().GetTokenIndex(), fmt.Sprintf(" LIMIT %d", r.limitCount))
		return
	}
	r.rewriter.InsertAfterDefault(ctx.Select_clause().GetStop().GetTokenIndex(), fmt.Sprintf(" LIMIT %d", r.limitCount))
}

// checkSelectLimit check whether this select_limit is empty.
func (*pgRewriter) checkSelectLimit(ctx pg.ISelect_limitContext) bool {
	if ctx == nil {
		return false
	}

	if ctx.Limit_clause() != nil {
		return true
	}
	return false
}
