package bigquery

import (
	"fmt"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterResultLimitFunc(storepb.Engine_BIGQUERY, statementWithResultLimit)
}

// statementWithResultLimit implements base.ResultLimitFunc for BigQuery, which
// takes no engineVersion. The caller only applies this to a statement it has
// already confirmed is a SELECT (see bigquery.go), so this wraps unconditionally.
func statementWithResultLimit(statement string, limit int, _ string) string {
	limitPart := ""
	if limit > 0 {
		limitPart = fmt.Sprintf(" LIMIT %d", limit)
	}
	return fmt.Sprintf("WITH result AS (%s) SELECT * FROM result%s;", base.TrimStatement(statement), limitPart)
}
