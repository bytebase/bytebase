package snowflake

import (
	"github.com/bytebase/omni/snowflake/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterResultLimitFunc(storepb.Engine_SNOWFLAKE, statementWithResultLimit)
}

// statementWithResultLimit implements base.ResultLimitFunc for Snowflake, which
// takes no engineVersion.
func statementWithResultLimit(statement string, limit int, _ string) string {
	return base.RewriteResultLimit(statement, limit, analysis.RewriteResultLimit, "SELECT * FROM (%s) LIMIT %d")
}
