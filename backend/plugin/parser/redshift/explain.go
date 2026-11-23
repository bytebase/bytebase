package redshift

import (
	"github.com/bytebase/omni/redshift/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterExplainStatementFunc(storepb.Engine_REDSHIFT, explainStatement)
}

// explainStatement implements base.ExplainStatement. Redshift returns one plan
// format here whatever the caller asked for: its EXPLAIN has no machine-readable
// form (the driver capability table offers this engine the default plan only).
func explainStatement(statement string, _ base.ExplainFormat) (string, error) {
	explained, err := analysis.ExplainStatement(statement)
	return base.ExplainStatementResult(explained, err, nil)
}
