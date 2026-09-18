package tidb

import (
	"fmt"
	"log/slog"
	"strings"

	tidbast "github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/pingcap/tidb/pkg/parser/format"
	tidbdriver "github.com/pingcap/tidb/pkg/types/parser_driver"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterResultLimitFunc(storepb.Engine_TIDB, statementWithResultLimit)
}

// statementWithResultLimit implements base.ResultLimitFunc for TiDB, which
// takes no engineVersion.
func statementWithResultLimit(statement string, limit int, _ string) string {
	stmt, err := statementWithResultLimitInline(statement, limit)
	if err != nil {
		slog.Error("fail to add limit clause", slog.String("statement", statement), log.BBError(err))
		return fmt.Sprintf("WITH result AS (%s) SELECT * FROM result LIMIT %d;", base.TrimStatement(statement), limit)
	}
	return stmt
}

func statementWithResultLimitInline(statement string, limit int) (string, error) {
	stmtList, err := ParseTiDB(statement, "", "")
	if err != nil {
		return "", errors.Wrapf(err, "failed to parse tidb statement: %s", statement)
	}
	if len(stmtList) != 1 {
		return "", errors.Errorf("expect one single statement in the query, %s", statement)
	}
	restoreFlags := format.DefaultRestoreFlags | format.RestoreStringWithoutDefaultCharset
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
		ctx := format.NewRestoreCtx(restoreFlags, &buffer)
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
		ctx := format.NewRestoreCtx(restoreFlags, &buffer)
		if err := stmt.Restore(ctx); err != nil {
			return "", err
		}
		return buffer.String(), nil
	default:
	}
	return statement, nil
}
