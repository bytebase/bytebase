package starrocks

import (
	"github.com/bytebase/omni/starrocks/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterExplainStatementFunc(storepb.Engine_STARROCKS, explainStatement)
}

func explainStatement(statement string, _ base.ExplainFormat) (string, error) {
	planned, err := analysis.ExplainStatement(statement)
	if err != nil {
		return "", convertParseError(statement, err, nil)
	}
	return planned, nil
}
