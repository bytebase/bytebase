package oceanbase

import (
	"github.com/bytebase/omni/oceanbase/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterResultLimitFunc(storepb.Engine_OCEANBASE, statementWithResultLimit)
}

// statementWithResultLimit implements base.ResultLimitFunc for OceanBase. It takes
// no engineVersion.
func statementWithResultLimit(statement string, limit int, _ string) string {
	return base.RewriteResultLimit(statement, limit, analysis.RewriteResultLimit, "SELECT * FROM (%s) result LIMIT %d;")
}
