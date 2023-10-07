package mysql

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"

	tidbast "github.com/pingcap/tidb/parser/ast"
	"github.com/pingcap/tidb/parser/format"

	tidbparser "github.com/bytebase/bytebase/backend/plugin/parser/tidb"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func (driver *Driver) getStatementWithResultLimit(stmt string, limit int) (string, error) {
	switch driver.dbType {
	case storepb.Engine_MYSQL, storepb.Engine_MARIADB:
		// MySQL 5.7 doesn't support WITH clause.
		return fmt.Sprintf("SELECT * FROM (%s) result LIMIT %d;", stmt, limit), nil
	case storepb.Engine_TIDB:
		return getStatementWithResultLimitForTiDB(stmt, limit)
	default:
		return "", errors.Errorf("unsupported database type %s", driver.dbType)
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
		default:
			continue
		}
	}
	return "", nil
}
