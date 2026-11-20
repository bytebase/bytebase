package doris

import (
	"github.com/bytebase/omni/doris/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterResultLimitFunc(storepb.Engine_DORIS, statementWithResultLimit)
}

func statementWithResultLimit(statement string, limit int, _ string) string {
	return analysis.RewriteResultLimit(statement, limit)
}
