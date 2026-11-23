package mysql

import (
	"github.com/bytebase/omni/mysql/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterResultLimitFunc(storepb.Engine_MYSQL, statementWithResultLimit)
}

// statementWithResultLimit implements base.ResultLimitFunc for MySQL, which
// takes no engineVersion. MySQL 5.7 does not support WITH, so the fallback
// wraps in a derived table.
func statementWithResultLimit(statement string, limit int, _ string) string {
	return base.RewriteResultLimit(statement, limit, analysis.RewriteResultLimit, "SELECT * FROM (%s) result LIMIT %d;")
}
