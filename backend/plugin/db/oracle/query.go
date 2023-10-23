package oracle

import (
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	plsql "github.com/bytebase/plsql-parser"

	plsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/plsql"
)

// singleStatement must be a selectStatement for oracle.
func getStatementWithResultLimitFor11g(singleStatement string, limitCount int) string {
	return fmt.Sprintf("SELECT * FROM (%s) WHERE ROWNUM <= %d", singleStatement, limitCount)
}

// singleStatement must be a selectStatement for oracle.
func getStatementWithResultLimitFor12c(singleStatement string, limitCount int) (string, error) {
	tree, stream, err := plsqlparser.ParsePLSQL(singleStatement)
	if err != nil {
		return "", err
	}

	listener := &plsqlRewriter{
		limitCount:        limitCount,
		selectFetch:       false,
		outerMostSubQuery: true,
	}

	listener.rewriter = *antlr.NewTokenStreamRewriter(stream)
	antlr.ParseTreeWalkerDefault.Walk(listener, tree)
	if listener.err != nil {
		return "", errors.Wrapf(listener.err, "statement: %s", singleStatement)
	}

	res := listener.rewriter.GetTextDefault()
	// https://stackoverflow.com/questions/27987882/how-can-i-solve-ora-00911-invalid-character-error
	res = strings.TrimRight(res, " \t\n\r\f;")

	return res, nil
}

type plsqlRewriter struct {
	*plsql.BasePlSqlParserListener

	rewriter antlr.TokenStreamRewriter
	err      error
	// fetch in select_statement
	selectFetch bool
	// fetch in subquery
	outerMostSubQuery bool
	limitCount        int
}

func (r *plsqlRewriter) EnterSelect_statement(ctx *plsql.Select_statementContext) {
	if ctx.AllFetch_clause() != nil && len(ctx.AllFetch_clause()) > 0 {
		r.selectFetch = true
		return
	}
}

func (r *plsqlRewriter) EnterSubquery(ctx *plsql.SubqueryContext) {
	if !r.outerMostSubQuery || r.selectFetch {
		return
	}
	r.outerMostSubQuery = false
	// union | intersect | minus
	if ctx.AllSubquery_operation_part() != nil && len(ctx.AllSubquery_operation_part()) > 0 {
		lastPart := ctx.Subquery_operation_part(len(ctx.AllSubquery_operation_part()) - 1)
		// respect original fetch.
		if lastPart.Subquery_basic_elements().Query_block().Fetch_clause() != nil {
			return
		}
		if subqueryOp, ok := lastPart.(*plsql.Subquery_operation_partContext); ok {
			r.rewriter.InsertAfterDefault(subqueryOp.GetStop().GetTokenIndex(), fmt.Sprintf(" FETCH NEXT %d ROWS ONLY", r.limitCount))
			return
		}
	}

	// otherwise (subquery and normally)
	basicElements := ctx.Subquery_basic_elements()
	// respect original fetch;
	if basicElements.Query_block().Fetch_clause() != nil {
		return
	}
	r.rewriter.InsertAfterDefault(basicElements.GetStop().GetTokenIndex(), fmt.Sprintf(" FETCH NEXT %d ROWS ONLY", r.limitCount))
}
