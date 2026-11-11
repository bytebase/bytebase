package tidb

import (
	"fmt"
	"log/slog"

	omniast "github.com/bytebase/omni/tidb/ast"
	omnideparse "github.com/bytebase/omni/tidb/deparse"
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
	list, err := ParseTiDB(statement)
	if err != nil {
		return "", errors.Wrapf(err, "failed to parse tidb statement: %s", statement)
	}
	if list == nil || len(list.Items) != 1 {
		return "", errors.Errorf("expect one single statement in the query, %s", statement)
	}
	// omni folds UNION/INTERSECT/EXCEPT into SelectStmt, so a single case covers
	// both a plain SELECT and a set operation; the trailing LIMIT lives on the
	// outermost SelectStmt either way.
	stmt, ok := list.Items[0].(*omniast.SelectStmt)
	if !ok {
		return statement, nil
	}
	if stmt.Limit != nil && stmt.Limit.Count != nil {
		if v, ok := stmt.Limit.Count.(*omniast.IntLit); ok {
			userLimit := int(v.Value)
			if limit < userLimit {
				userLimit = limit
			}
			stmt.Limit.Count = &omniast.IntLit{Value: int64(userLimit)}
		}
	} else {
		stmt.Limit = &omniast.Limit{
			Count: &omniast.IntLit{Value: int64(limit)},
		}
	}
	// RestoreStringWithoutCharset mirrors pingcap's RestoreStringWithoutDefaultCharset:
	// string literals keep no _UTF8MB4 introducer. The remaining omni defaults
	// (uppercase keywords, backtick names, single-quoted strings) match the
	// previous pingcap DefaultRestoreFlags output.
	return omnideparse.RestoreSelectWith(stmt, omnideparse.RestoreStringWithoutCharset), nil
}
