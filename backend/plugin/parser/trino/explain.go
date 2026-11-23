package trino

import (
	"github.com/bytebase/omni/trino/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterExplainStatementFunc(storepb.Engine_TRINO, explainStatement)
}

// explainStatement implements base.ExplainStatement. Trino spells its structured
// plan (FORMAT JSON); any other format the caller names gets the engine's own
// text plan (the driver capability table refuses the rest).
func explainStatement(statement string, format base.ExplainFormat) (string, error) {
	explain, err := analysis.ExplainStatement(statement, format == base.ExplainFormatJSON)
	return base.ExplainStatementResult(explain, err, func(e error) error {
		return convertParseError(statement, e, nil)
	})
}
