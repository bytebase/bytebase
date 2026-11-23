package tsql

import (
	omnimssql "github.com/bytebase/omni/mssql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterResultLimitFunc(storepb.Engine_MSSQL, statementWithResultLimit)
}

// statementWithResultLimit implements base.ResultLimitFunc for SQL Server,
// which takes no engineVersion.
func statementWithResultLimit(statement string, limit int, _ string) string {
	return base.RewriteResultLimit(statement, limit, omnimssql.StatementWithResultLimit, "WITH result AS (%s) SELECT TOP %d * FROM result;")
}
