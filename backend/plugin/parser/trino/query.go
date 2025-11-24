package trino

import (
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterQueryValidator(storepb.Engine_TRINO, validateQuery)
}

// validateQuery validates if the given SQL statement is valid for SQL editor.
// We only allow read-only queries in the SQL editor.
// Returns (canRunInReadOnly, returnsData, error):
// - canRunInReadOnly: whether all queries can run in read-only mode
// - returnsData: whether all queries return data
// - error: parsing error if the statement is invalid
func validateQuery(statement string) (bool, bool, error) {
	parseResults, err := ParseTrino(statement)
	if err != nil {
		return false, false, err
	}

	allReadOnly := true
	allReturnData := true

	// Validate each statement
	for _, result := range parseResults {
		queryType, isAnalyze := getQueryType(result.Tree)

		// If it's an EXPLAIN ANALYZE, the query will be executed
		if isAnalyze {
			// Only allow EXPLAIN ANALYZE for SELECT statements
			if queryType != base.Select {
				return false, false, nil
			}
			// EXPLAIN ANALYZE SELECT is read-only and returns data
			continue
		}

		// Determine if the statement is read-only
		readOnly := queryType == base.Select ||
			queryType == base.Explain ||
			queryType == base.SelectInfoSchema

		// Determine if the statement returns data
		returnsData := queryType == base.Select ||
			queryType == base.Explain ||
			queryType == base.SelectInfoSchema

		if !readOnly {
			allReadOnly = false
		}
		if !returnsData {
			allReturnData = false
		}

		// If any statement fails validation, return immediately
		if !allReadOnly {
			break
		}
	}

	return allReadOnly, allReturnData, nil
}
