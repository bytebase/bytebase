package pg

import (
	"fmt"
	"log/slog"

	"github.com/bytebase/omni/pg/analysis"

	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterResultLimitFunc(storepb.Engine_POSTGRES, statementWithResultLimit)
	// CockroachDB parses through the PostgreSQL grammar here as it does for the
	// read-only gate and EXPLAIN (see query.go, explain.go).
	base.RegisterResultLimitFunc(storepb.Engine_COCKROACHDB, statementWithResultLimit)
}

// statementWithResultLimit implements base.ResultLimitFunc for PostgreSQL and
// CockroachDB, which share this grammar. Neither takes an engineVersion. A
// statement omni cannot rewrite is wrapped instead.
func statementWithResultLimit(statement string, limit int, _ string) string {
	stmt, err := analysis.RewriteResultLimit(statement, limit)
	if err != nil {
		slog.Error("fail to add limit clause", slog.String("statement", statement), log.BBError(err))
		// Fallback to CTE approach for problematic queries
		return fmt.Sprintf("WITH result AS (\n%s\n) SELECT * FROM result LIMIT %d;", base.TrimStatement(statement), limit)
	}
	return stmt
}
