package oceanbase

import (
	"encoding/json"

	"github.com/pkg/errors"
)

// ExplainJSON represents the JSON structure returned by EXPLAIN format=json
// We only care about the EST.ROWS field for row count estimation
type ExplainJSON struct {
	EstRows int64 `json:"EST.ROWS"`
}

// getEstimatedRowsFromJSON extracts the estimated row count from OceanBase EXPLAIN format=json result
func getEstimatedRowsFromJSON(res []any) (int64, error) {
	// For EXPLAIN format=json, OceanBase returns JSON data
	// The res struct is []any{columnName, columnTable, rowDataList}
	if len(res) < 3 {
		return 0, errors.Errorf("expected at least 3 elements but got %d", len(res))
	}

	rowList, ok := res[2].([]any)
	if !ok {
		return 0, errors.Errorf("expected []any for row data but got %T", res[2])
	}
	if len(rowList) == 0 {
		return 0, errors.Errorf("no data returned from EXPLAIN")
	}

	// OceanBase might return JSON data split across multiple elements
	// We need to concatenate them to form a valid JSON string
	var jsonStr string
	for _, rowAny := range rowList {
		row, ok := rowAny.([]any)
		if !ok {
			return 0, errors.Errorf("expected []any for row but got %T", rowAny)
		}
		for _, cellAny := range row {
			if cell, ok := cellAny.(string); ok {
				jsonStr += cell
			}
		}
	}

	if jsonStr == "" {
		return 0, errors.Errorf("no JSON data found in EXPLAIN result")
	}

	var explainData ExplainJSON
	if err := json.Unmarshal([]byte(jsonStr), &explainData); err != nil {
		return 0, errors.Errorf("failed to parse JSON: %v", err)
	}

	// Return the estimated rows from the JSON response
	return explainData.EstRows, nil
}
