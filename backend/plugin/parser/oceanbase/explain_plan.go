package oceanbase

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
)

// EstimateAffectedRowsFromExplainJSON returns the EST.ROWS estimate from OceanBase
// `EXPLAIN FORMAT=JSON` output: the root operator's, or the largest child operator's when the
// root has no operator.
func EstimateAffectedRowsFromExplainJSON(plan string) (int64, error) {
	var root map[string]any
	if err := json.Unmarshal([]byte(plan), &root); err != nil {
		return 0, errors.Wrap(err, "failed to parse EXPLAIN FORMAT=JSON output")
	}
	if operator, ok := root["OPERATOR"].(string); ok && operator != "" {
		rows, ok := planNumber(root["EST.ROWS"])
		if !ok {
			return 0, errors.Errorf("operator %q has no EST.ROWS", operator)
		}
		return common.RoundRows(rows), nil
	}
	var maxRows float64
	found := false
	for key, value := range root {
		child, ok := value.(map[string]any)
		if !ok || !strings.HasPrefix(key, "CHILD_") {
			continue
		}
		operator, ok := child["OPERATOR"].(string)
		if !ok || operator == "" {
			continue
		}
		rows, ok := planNumber(child["EST.ROWS"])
		if !ok {
			return 0, errors.Errorf("operator %q has no EST.ROWS", operator)
		}
		if !found || rows > maxRows {
			maxRows = rows
			found = true
		}
	}
	if !found {
		return 0, errors.New("the plan has no operator")
	}
	return common.RoundRows(maxRows), nil
}

// planNumber reads an estimate the plan reports either as a JSON number or as a string.
func planNumber(value any) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case string:
		f, err := strconv.ParseFloat(v, 64)
		return f, err == nil
	default:
		return 0, false
	}
}
