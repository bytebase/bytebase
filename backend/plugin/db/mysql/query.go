package mysql

import (
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	tidbast "github.com/pingcap/tidb/parser/ast"
	"github.com/pingcap/tidb/parser/format"

	mysql "github.com/bytebase/mysql-parser"

	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	tidbparser "github.com/bytebase/bytebase/backend/plugin/parser/tidb"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func (driver *Driver) getStatementWithResultLimit(stmt string, limit int) (string, error) {
	switch driver.dbType {
	case storepb.Engine_TIDB:
		return getStatementWithResultLimitForTiDB(stmt, limit)
	default:
		return getStatementWithResultLimitForMySQL(stmt, limit)
	}
}

func getStatementWithResultLimitForTiDB(singleStatement string, limitCount int) (string, error) {
	stmtList, err := tidbparser.ParseTiDB(singleStatement, "", "")
	if err != nil {
		return "", errors.Wrapf(err, "failed to parse tidb statement: %s", singleStatement)
	}
	for _, stmt := range stmtList {
		switch stmt := stmt.(type) {
		case *tidbast.SelectStmt:
			limit := &tidbast.Limit{
				Count: tidbast.NewValueExpr(int64(limitCount), "", ""),
			}
			if stmt.Limit != nil {
				limit = stmt.Limit
				if stmt.Limit.Count != nil {
					// If the statement already has limit clause, we will return the original statement.
					return singleStatement, nil
				}
				stmt.Limit.Count = tidbast.NewValueExpr(int64(limitCount), "", "")
			}
			stmt.Limit = limit
			var buffer strings.Builder
			ctx := format.NewRestoreCtx(format.DefaultRestoreFlags, &buffer)
			if err := stmt.Restore(ctx); err != nil {
				return "", err
			}
			return buffer.String(), nil
		case *tidbast.SetOprStmt:
			limit := &tidbast.Limit{
				Count: tidbast.NewValueExpr(int64(limitCount), "", ""),
			}
			if stmt.Limit != nil {
				limit = stmt.Limit
				if stmt.Limit.Count != nil {
					// If the statement already has limit clause, we will return the original statement.
					return singleStatement, nil
				}
				stmt.Limit.Count = tidbast.NewValueExpr(int64(limitCount), "", "")
			}
			stmt.Limit = limit
			var buffer strings.Builder
			ctx := format.NewRestoreCtx(format.DefaultRestoreFlags, &buffer)
			if err := stmt.Restore(ctx); err != nil {
				return "", err
			}
			return buffer.String(), nil

		default:
			continue
		}
	}
	return "", nil
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
