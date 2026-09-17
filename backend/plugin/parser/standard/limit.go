package standard

import (
	"fmt"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterResultLimitFunc(storepb.Engine_CLICKHOUSE, statementWithResultLimit)
}

// statementWithResultLimit implements base.ResultLimitFunc for ClickHouse,
// which takes no engineVersion. ClickHouse is parsed only as text here (see
// query.go), so there is no AST to place the LIMIT precisely; every statement
// is wrapped in a CTE.
func statementWithResultLimit(statement string, limit int, _ string) string {
	return fmt.Sprintf("WITH result AS (%s) SELECT * FROM result LIMIT %d;", base.TrimStatement(statement), limit)
}
