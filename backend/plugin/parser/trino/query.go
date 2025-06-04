package trino

import (
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	base.RegisterQueryValidator(storepb.Engine_TRINO, validateQuery)
}

// validateQuery validates if the given SQL statement is valid for SQL editor.
// We only allow read-only queries in the SQL editor.
func validateQuery(statement string) (bool, bool, error) {
	result, err := ParseTrino(statement)
	if err != nil {
		return false, false, err
	}

	queryType, isAnalyze := getQueryType(result.Tree, false)

	// If it's an EXPLAIN ANALYZE, the query will be executed
	if isAnalyze {
		// Only allow EXPLAIN ANALYZE for SELECT statements
		if queryType == base.Select {
			return true, true, nil
		}
		return false, false, nil
	}

	// Determine if the statement is read-only
	readOnly := queryType == base.Select ||
		queryType == base.Explain ||
		queryType == base.SelectInfoSchema

	// Determine if the statement returns data
	returnsData := queryType == base.Select ||
		queryType == base.Explain ||
		queryType == base.SelectInfoSchema

	return readOnly, returnsData, nil
}
