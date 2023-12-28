package mysql

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/pkg/errors"

	tidbast "github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/pingcap/tidb/pkg/parser/format"

	"github.com/bytebase/bytebase/backend/common/log"
	tidbparser "github.com/bytebase/bytebase/backend/plugin/parser/tidb"
)

func getStatementWithResultLimit(stmt string, limit int) string {
	stmt, err := getStatementWithResultLimitForTiDB(stmt, limit)
	if err != nil {
		slog.Error("fail to add limit clause", "statement", stmt, log.BBError(err))
		stmt = fmt.Sprintf("WITH result AS (%s) SELECT * FROM result LIMIT %d;", stmt, limit)
	}
	return stmt
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
