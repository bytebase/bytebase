package standard

import (
	"log/slog"

	chanalysis "github.com/bytebase/omni/clickhouse/analysis"

	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterQueryValidator(storepb.Engine_CLICKHOUSE, ValidateSQLForEditor)
	base.RegisterQueryValidator(storepb.Engine_SPANNER, ValidateSQLForEditor)
	base.RegisterQueryValidator(storepb.Engine_HIVE, ValidateSQLForEditor)
	base.RegisterQueryValidator(storepb.Engine_BIGQUERY, ValidateSQLForEditor)
	base.RegisterQueryValidator(storepb.Engine_CASSANDRA, ValidateSQLForEditor)
}

// ValidateSQLForEditor validates the SQL statement for SQL editor.
func ValidateSQLForEditor(statement string) (bool, bool, error) {
	readOnly, returnsData, err := chanalysis.ValidateQuery(statement)
	if err != nil {
		slog.Debug("Failed to remove quoted text and comment", slog.String("statement", statement), log.BBError(err))
		return false, false, err
	}
	return readOnly, returnsData, nil
}

// CheckStatementWithoutQuotedTextAndComment classifies a single statement whose
// quoted text and comments have already been removed.
func CheckStatementWithoutQuotedTextAndComment(statement string) bool {
	return chanalysis.CheckStatementWithoutQuotedTextAndComment(statement)
}
