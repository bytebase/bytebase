package standard

import (
	chanalysis "github.com/bytebase/omni/clickhouse/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterResultLimitFunc(storepb.Engine_CLICKHOUSE, statementWithResultLimit)
}

// statementWithResultLimit implements base.ResultLimitFunc for ClickHouse,
// which takes no engineVersion.
func statementWithResultLimit(statement string, limit int, _ string) string {
	return chanalysis.WrapResultLimit(statement, limit)
}
