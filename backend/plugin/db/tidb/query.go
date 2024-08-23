package tidb

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/pkg/errors"

	tidbast "github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/pingcap/tidb/pkg/parser/format"
	tidbdriver "github.com/pingcap/tidb/pkg/types/parser_driver"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	tidbparser "github.com/bytebase/bytebase/backend/plugin/parser/tidb"
)

func getStatementWithResultLimit(statement string, limit int) string {
	statement, err := getStatementWithResultLimitInline(statement, limit)
	if err != nil {
		slog.Error("fail to add limit clause", "statement", statement, log.BBError(err))
		statement = fmt.Sprintf("WITH result AS (%s) SELECT * FROM result LIMIT %d;", util.TrimStatement(statement), limit)
	}
	return statement
}

func getStatementWithResultLimitInline(statement string, limit int) (string, error) {
	stmtList, err := tidbparser.ParseTiDB(statement, "", "")
	if err != nil {
		return "", errors.Wrapf(err, "failed to parse tidb statement: %s", statement)
	}
	if len(stmtList) != 1 {
		return "", errors.Errorf("expect one single statement in the query, %s", statement)
	}
	stmt := stmtList[0]
	switch stmt := stmt.(type) {
	case *tidbast.SelectStmt:
		if stmt.Limit != nil && stmt.Limit.Count != nil {
			if v, ok := stmt.Limit.Count.(*tidbdriver.ValueExpr); ok {
				userLimit := int(v.GetInt64())
				if limit < userLimit {
					userLimit = limit
				}
				stmt.Limit.Count = tidbast.NewValueExpr(int64(userLimit), "", "")
			}
		} else {
			stmt.Limit = &tidbast.Limit{
				Count: tidbast.NewValueExpr(int64(limit), "", ""),
			}
		}
		var buffer strings.Builder
		ctx := format.NewRestoreCtx(format.DefaultRestoreFlags, &buffer)
		if err := stmt.Restore(ctx); err != nil {
			return "", err
		}
		return buffer.String(), nil
	case *tidbast.SetOprStmt:
		if stmt.Limit != nil && stmt.Limit.Count != nil {
			if v, ok := stmt.Limit.Count.(*tidbdriver.ValueExpr); ok {
				userLimit := int(v.GetInt64())
				if limit < userLimit {
					userLimit = limit
				}
				stmt.Limit.Count = tidbast.NewValueExpr(int64(userLimit), "", "")
			}
		} else {
			stmt.Limit = &tidbast.Limit{
				Count: tidbast.NewValueExpr(int64(limit), "", ""),
			}
		}
		var buffer strings.Builder
		ctx := format.NewRestoreCtx(format.DefaultRestoreFlags, &buffer)
		if err := stmt.Restore(ctx); err != nil {
			return "", err
		}
		return buffer.String(), nil
	default:
	}
	return statement, nil
}
