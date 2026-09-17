package tsql

import (
	"fmt"
	"log/slog"

	omnimssql "github.com/bytebase/omni/mssql"

	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterResultLimitFunc(storepb.Engine_MSSQL, statementWithResultLimit)
}

// statementWithResultLimit implements base.ResultLimitFunc for SQL Server,
// which takes no engineVersion. The rewrite itself lives in omni, which already
// owns the T-SQL grammar this needs.
func statementWithResultLimit(statement string, limit int, _ string) string {
	stmt, err := omnimssql.StatementWithResultLimit(statement, limit)
	if err != nil {
		slog.Error("fail to add limit clause", slog.String("statement", statement), log.BBError(err))
		return fmt.Sprintf("WITH result AS (%s) SELECT TOP %d * FROM result;", base.TrimStatement(statement), limit)
	}
	return stmt
}
