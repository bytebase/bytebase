package snowflake

import (
	"fmt"
	"log/slog"
	"strconv"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	snowsql "github.com/bytebase/parser/snowflake"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	snowparser "github.com/bytebase/bytebase/backend/plugin/parser/snowflake"
)

func getStatementWithResultLimit(statement string, limit int) string {
	stmt, err := getStatementWithResultLimitInline(statement, limit)
	if err != nil {
		slog.Error("fail to add limit clause", slog.String("statement", statement), log.BBError(err))
		return fmt.Sprintf("SELECT * FROM (%s) LIMIT %d", util.TrimStatement(statement), limit)
	}
	return stmt
}

func getStatementWithResultLimitInline(singleStatement string, limitCount int) (string, error) {
	results, err := snowparser.ParseSnowSQL(singleStatement)
	if err != nil {
		return "", err
	}

	if len(results) != 1 {
		return "", errors.Errorf("expected exactly 1 statement, got %d", len(results))
	}

	result := results[0]

	listener := &snowsqlRewriter{
		limitCount:     limitCount,
		outerMostQuery: true,
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

	rewriter       antlr.TokenStreamRewriter
	err            error
	outerMostQuery bool
	limitCount     int
}

func (r *snowsqlRewriter) EnterQuery_statement(ctx *snowsql.Query_statementContext) {
	if !r.outerMostQuery {
		return
	}
	r.outerMostQuery = false

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

	limitClause := selectCtx.Limit_clause()
	if limitClause != nil {
		if limitClause.LIMIT() != nil {
			if len(limitClause.AllNum()) > 0 {
				firstNumber := limitClause.Num(0)
				userLimitText := firstNumber.GetText()
				limit, _ := strconv.Atoi(userLimitText)
				if limit == 0 || r.limitCount < limit {
					limit = r.limitCount
				}
				r.rewriter.ReplaceDefault(firstNumber.GetStart().GetTokenIndex(), firstNumber.GetStop().GetTokenIndex(), fmt.Sprintf("%d", limit))
			}
		} else {
			var num snowsql.INumContext
			if limitClause.OFFSET() != nil {
				if len(limitClause.AllNum()) > 1 {
					num = limitClause.Num(1)
				}
			} else {
				if len(limitClause.AllNum()) > 0 {
					num = limitClause.Num(0)
				}
			}
			if num != nil {
				userLimitText := num.GetText()
				limit, _ := strconv.Atoi(userLimitText)
				if limit == 0 || r.limitCount < limit {
					limit = r.limitCount
				}
				r.rewriter.ReplaceDefault(num.GetStart().GetTokenIndex(), num.GetStop().GetTokenIndex(), fmt.Sprintf("%d", limit))
			}
		}
		return
	}

	selectTopClause := selectCtx.Select_top_clause()
	if selectTopClause != nil && selectTopClause.Select_list_top() != nil && selectTopClause.Select_list_top().Top_clause() != nil {
		topClause := selectTopClause.Select_list_top().Top_clause()
		number := topClause.Num()
		userLimitText := number.GetText()
		limit, _ := strconv.Atoi(userLimitText)
		if limit == 0 || r.limitCount < limit {
			limit = r.limitCount
		}
		r.rewriter.ReplaceDefault(number.GetStart().GetTokenIndex(), number.GetStop().GetTokenIndex(), fmt.Sprintf("%d", limit))
		return
	}

	// Append after select_optional_clauses.
	r.rewriter.InsertAfterDefault(selectCtx.Select_optional_clauses().GetStop().GetTokenIndex(), fmt.Sprintf(" LIMIT %d", r.limitCount))
}
