package mariadb

import (
	"github.com/bytebase/omni/mariadb/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterResultLimitFunc(storepb.Engine_MARIADB, statementWithResultLimit)
}

// statementWithResultLimit implements base.ResultLimitFunc for MariaDB, which
// takes no engineVersion.
func statementWithResultLimit(statement string, limit int, _ string) string {
	return base.RewriteResultLimit(statement, limit, analysis.RewriteResultLimit, "SELECT * FROM (%s) result LIMIT %d;")
}
