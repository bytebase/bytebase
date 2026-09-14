package oceanbase

import (
	"strings"

	"github.com/pkg/errors"

	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

// getEstimatedRowsFromJSON extracts the estimated row count from an OceanBase
// `EXPLAIN FORMAT=JSON` result shaped as []any{columnNames, columnTypes, rows}.
func getEstimatedRowsFromJSON(res []any) (int64, error) {
	if len(res) < 3 {
		return 0, errors.Errorf("expected at least 3 elements but got %d", len(res))
	}
	rowList, ok := res[2].([]any)
	if !ok {
		return 0, errors.Errorf("expected []any for row data but got %T", res[2])
	}

	// OceanBase splits the JSON plan across rows.
	var plan strings.Builder
	for _, rowAny := range rowList {
		row, ok := rowAny.([]any)
		if !ok {
			return 0, errors.Errorf("expected []any for row but got %T", rowAny)
		}
		for _, cellAny := range row {
			if cell, ok := cellAny.(string); ok {
				plan.WriteString(cell)
			}
		}
	}
	return mysqlparser.EstimateAffectedRowsFromOceanBaseExplainJSON(plan.String())
}
