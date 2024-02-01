package starrocks

import (
	"fmt"
	"log/slog"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	mysql "github.com/bytebase/mysql-parser"

	"github.com/bytebase/bytebase/backend/common/log"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

func getStatementWithResultLimit(stmt string, limit int) string {
	stmt, err := getStatementWithResultLimitForMySQL(stmt, limit)
	if err != nil {
		slog.Error("fail to add limit clause", "statement", stmt, log.BBError(err))
		// MySQL 5.7 doesn't support WITH clause.
		stmt = fmt.Sprintf("SELECT * FROM (%s) result LIMIT %d;", stmt, limit)
	}
	return stmt
}

// singleStatement must be a selectStatement for mysql.
func getStatementWithResultLimitForMySQL(singleStatement string, limitCount int) (string, error) {
	list, err := mysqlparser.ParseMySQL(singleStatement)
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
			return "", errors.Wrapf(listener.err, "statement: %s", singleStatement)
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
	if ctx.LimitClause() != nil {
		// limit clause already exists.
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
