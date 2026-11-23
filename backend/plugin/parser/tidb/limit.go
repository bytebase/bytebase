package tidb

import (
	"github.com/bytebase/omni/tidb/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterResultLimitFunc(storepb.Engine_TIDB, statementWithResultLimit)
}

// statementWithResultLimit implements base.ResultLimitFunc for TiDB, which
// takes no engineVersion.
func statementWithResultLimit(statement string, limit int, _ string) string {
	return base.RewriteResultLimit(statement, limit, analysis.RewriteResultLimit, "WITH result AS (%s) SELECT * FROM result LIMIT %d;")
}
