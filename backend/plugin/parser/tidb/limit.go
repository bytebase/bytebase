package tidb

import (
	"fmt"
	"log/slog"

	"github.com/bytebase/omni/tidb/analysis"

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
	stmt, err := analysis.RewriteResultLimit(statement, limit)
	if err != nil {
		slog.Error("fail to add limit clause", slog.String("statement", statement), log.BBError(err))
		return fmt.Sprintf("WITH result AS (%s) SELECT * FROM result LIMIT %d;", base.TrimStatement(statement), limit)
	}
	return stmt
}
