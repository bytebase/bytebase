package redshift

import (
	"github.com/bytebase/omni/redshift/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterResultLimitFunc(storepb.Engine_REDSHIFT, statementWithResultLimit)
}

// statementWithResultLimit implements base.ResultLimitFunc for Redshift, which
// takes no engineVersion.
func statementWithResultLimit(statement string, limit int, _ string) string {
	return analysis.StatementWithResultLimit(statement, limit)
}
