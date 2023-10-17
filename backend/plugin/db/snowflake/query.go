package snowflake

import (
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	snowsql "github.com/bytebase/snowsql-parser"

	snowparser "github.com/bytebase/bytebase/backend/plugin/parser/snowflake"
)

// singleStatement must be a selectStatement for snowflake.
func getStatementWithResultLimit(singleStatement string, limitCount int) (string, error) {
	result, err := snowparser.ParseSnowSQL(singleStatement)
	if err != nil {
		return "", err
	}

	listener := &snowsqlRewriter{
		limitCount: limitCount,
		depth:      0,
	}

	listener.rewriter = *antlr.NewTokenStreamRewriter(result.Tokens)
	antlr.ParseTreeWalkerDefault.Walk(listener, result.Tree)
	if listener.err != nil {
		return "", errors.Wrapf(listener.err, "statement: %s", singleStatement)
	}

	return listener.rewriter.GetTextDefault(), nil
}

type snowsqlRewriter struct {
	*snowsql.BaseSnowflakeParserListener

	rewriter   antlr.TokenStreamRewriter
	err        error
	depth      int
	limitCount int
}

func (r *snowsqlRewriter) EnterQuery_statement(*snowsql.Query_statementContext) {
	r.depth++
}

func (r *snowsqlRewriter) ExitQuery_statement(ctx *snowsql.Query_statementContext) {
	r.depth--
	// outermost query_statement.
	if r.depth > 0 {
		return
	}

	// if has set_operators, use last select_statement of set_operator.
	var selectCtx *snowsql.Select_statementContext
	if ctx.AllSet_operators() != nil {
		setOperator := ctx.Set_operators(len(ctx.AllSet_operators()) - 1)
		if setOperator != nil && setOperator.Select_statement() != nil {
			var ok bool
			if selectCtx, ok = setOperator.Select_statement().(*snowsql.Select_statementContext); !ok {
				return
			}
		}
	}
	// otherwise, ues outermost direct select_statement.
	if selectCtx == nil {
		var ok bool
		if selectCtx, ok = ctx.Select_statement().(*snowsql.Select_statementContext); !ok {
			return
		}
	}

	// do not change original limit clause.
	if selectCtx.Limit_clause() != nil {
		return
	}

	// limit and top are not allowed together.
	if selectCtx.Select_top_clause() != nil {
		return
	}

	// append after select_optional_clauses
	r.rewriter.InsertAfterDefault(selectCtx.Select_optional_clauses().GetStop().GetTokenIndex(), fmt.Sprintf(" LIMIT %d", r.limitCount))
}
