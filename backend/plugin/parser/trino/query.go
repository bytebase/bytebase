package trino

import (
	"github.com/bytebase/omni/trino/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterQueryValidator(storepb.Engine_TRINO, validateQuery)
}

func validateQuery(statement string) (bool, bool, error) {
	readOnly, returnsData, err := analysis.ValidateQuery(statement)
	if err != nil {
		return false, false, convertParseError(statement, err, nil)
	}
	return readOnly, returnsData, nil
}
