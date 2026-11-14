package plsql

import (
	"log/slog"

	"github.com/bytebase/omni/oracle/analysis"

	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterResultLimitFunc(storepb.Engine_ORACLE, statementWithResultLimit)
}

// statementWithResultLimit implements base.ResultLimitFunc for Oracle.
// engineVersion picks the clause: ROWNUM before version 12, FETCH NEXT from
// 12c on. A FETCH NEXT rewrite that fails falls back to ROWNUM.
func statementWithResultLimit(statement string, limit int, engineVersion string) string {
	stmt, err := analysis.RewriteResultLimit(statement, limit, engineVersion)
	if err != nil {
		slog.Error("failed to add FETCH NEXT clause, falling back to ROWNUM",
			slog.String("statement", statement), log.BBError(err))
		return analysis.RowNumResultLimit(statement, limit)
	}
	return stmt
}
