package starrocks

import (
	"fmt"
	"log/slog"
	"strconv"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	mysql "github.com/bytebase/mysql-parser"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

func getStatementWithResultLimit(statement string, limit int) string {
	statement, err := getStatementWithResultLimitForMySQL(statement, limit)
	if err != nil {
		slog.Error("fail to add limit clause", "statement", statement, log.BBError(err))
		// MySQL 5.7 doesn't support WITH clause.
		statement = fmt.Sprintf("SELECT * FROM (%s) result LIMIT %d;", util.TrimStatement(statement), limit)
	}
	return statement
}

// singleStatement must be a selectStatement for mysql.
func getStatementWithResultLimitForMySQL(statement string, limitCount int) (string, error) {
	list, err := mysqlparser.ParseMySQL(statement)
	if err != nil {
		return "", err
	}

	listener := &mysqlRewriter{
		limitCount:     limitCount,
		outerMostQuery: true,
	}

	for _, stmt := range list {
		listener.rewriter = *antlr.NewTokenStreamRewriter(stmt.Tokens)
		antlr.ParseTreeWalkerDefault.Walk(listener, stmt.Tree)
		if listener.err != nil {
			return "", errors.Wrapf(listener.err, "statement: %s", statement)
		}
	}
	return listener.rewriter.GetTextDefault(), nil
}

type mysqlRewriter struct {
	*mysql.BaseMySQLParserListener

	rewriter       antlr.TokenStreamRewriter
	err            error
	outerMostQuery bool
	limitCount     int
}

func (r *mysqlRewriter) EnterQueryExpression(ctx *mysql.QueryExpressionContext) {
	if !r.outerMostQuery {
		return
	}
	r.outerMostQuery = false
	limitCluase := ctx.LimitClause()
	if limitCluase != nil {
		// limit clause already exists.
		userLimitText := limitCluase.LimitOptions().GetText()
		limit, err := strconv.Atoi(userLimitText)
		if err == nil {
			if r.limitCount < limit {
				limit = r.limitCount
			}
		}
		r.rewriter.ReplaceDefault(limitCluase.GetStart().GetTokenIndex(), limitCluase.GetStop().GetTokenIndex(), fmt.Sprintf("LIMIT %d", limit))
		return
	}

	if ctx.OrderClause() != nil {
		r.rewriter.InsertAfterDefault(ctx.OrderClause().GetStop().GetTokenIndex(), fmt.Sprintf(" LIMIT %d", r.limitCount))
	} else {
		switch {
		case ctx.QueryExpressionBody() != nil:
			r.rewriter.InsertAfterDefault(ctx.QueryExpressionBody().GetStop().GetTokenIndex(), fmt.Sprintf(" LIMIT %d", r.limitCount))
		case ctx.QueryExpressionParens() != nil:
			r.rewriter.InsertAfterDefault(ctx.QueryExpressionParens().GetStop().GetTokenIndex(), fmt.Sprintf(" LIMIT %d", r.limitCount))
		}
	}
}
