package pg

import (
	"github.com/bytebase/omni/pg/analysis"

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
	return base.RewriteResultLimit(statement, limit, analysis.RewriteResultLimit, "WITH result AS (\n%s\n) SELECT * FROM result LIMIT %d;")
}
