package oceanbase

import (
	"fmt"
	"log/slog"

	"github.com/bytebase/omni/oceanbase/analysis"

	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterResultLimitFunc(storepb.Engine_OCEANBASE, statementWithResultLimit)
}

// statementWithResultLimit implements base.ResultLimitFunc for OceanBase. It takes
// no engineVersion.
func statementWithResultLimit(statement string, limit int, _ string) string {
	stmt, err := analysis.RewriteResultLimit(statement, limit)
	if err != nil {
		slog.Error("fail to add limit clause", slog.String("statement", statement), log.BBError(err))
		// MySQL 5.7 doesn't support WITH clause.
		return fmt.Sprintf("SELECT * FROM (%s) result LIMIT %d;", base.TrimStatement(statement), limit)
	}
	return stmt
}
