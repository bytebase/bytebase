package snowflake

import (
	"fmt"
	"log/slog"

	"github.com/bytebase/omni/snowflake/analysis"

	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterResultLimitFunc(storepb.Engine_SNOWFLAKE, statementWithResultLimit)
}

// statementWithResultLimit implements base.ResultLimitFunc for Snowflake, which
// takes no engineVersion. A statement omni cannot rewrite is wrapped instead.
func statementWithResultLimit(statement string, limit int, _ string) string {
	stmt, err := analysis.RewriteResultLimit(statement, limit)
	if err != nil {
		slog.Error("fail to add limit clause", slog.String("statement", statement), log.BBError(err))
		return fmt.Sprintf("SELECT * FROM (%s) LIMIT %d", base.TrimStatement(statement), limit)
	}
	return stmt
}
