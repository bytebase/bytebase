package redshift

import (
	"github.com/bytebase/omni/redshift/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	// Override the existing PG-based validator with Redshift-specific one
	base.RegisterQueryValidator(storepb.Engine_REDSHIFT, ValidateSQLForEditor)
}

// ValidateSQLForEditor validates the SQL statement for SQL editor.
func ValidateSQLForEditor(statement string) (bool, bool, error) {
	readOnly, returnsData, err := analysis.ValidateQuery(statement)
	if err != nil {
		return false, false, convertError(err, base.Statement{
			Text:  statement,
			Start: &storepb.Position{Line: 1, Column: 1},
		})
	}
	return readOnly, returnsData, nil
}
