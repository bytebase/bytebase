package trino

import (
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterQueryValidator(storepb.Engine_TRINO, validateQuery)
}

// validateQuery reports whether the given statement is valid for the SQL editor,
// which only permits read-only queries.
//
// It returns (canRunInReadOnly, returnsData, error):
//   - canRunInReadOnly: every statement can run in read-only mode;
//   - returnsData: every statement returns data;
//   - error: a syntax error if the statement is invalid.
//
// EXPLAIN ANALYZE is special-cased exactly as the legacy plugin did: because it
// executes the inner query, it is only accepted when getQueryType reports it as
// read-only (base.Select); it then counts as read-only and data-returning.
func validateQuery(statement string) (bool, bool, error) {
	parsed, err := parseTrinoSQL(statement)
	if err != nil {
		return false, false, err
	}

	allReadOnly := true
	allReturnData := true

	for _, p := range parsed {
		queryType, isAnalyze := getQueryType(p.Node())

		if isAnalyze {
			// EXPLAIN ANALYZE runs the query. Only allow it when the type is a
			// read-only SELECT; in that case it is read-only and returns data.
			if queryType != base.Select {
				return false, false, nil
			}
			continue
		}

		readOnly := queryType == base.Select ||
			queryType == base.Explain ||
			queryType == base.SelectInfoSchema
		returnsData := readOnly

		if !readOnly {
			allReadOnly = false
		}
		if !returnsData {
			allReturnData = false
		}
		if !allReadOnly {
			break
		}
	}

	return allReadOnly, allReturnData, nil
}
