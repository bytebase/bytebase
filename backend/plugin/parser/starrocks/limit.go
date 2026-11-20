package starrocks

import (
	"github.com/bytebase/omni/starrocks/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterResultLimitFunc(storepb.Engine_STARROCKS, statementWithResultLimit)
}

func statementWithResultLimit(statement string, limit int, _ string) string {
	return analysis.RewriteResultLimit(statement, limit)
}
