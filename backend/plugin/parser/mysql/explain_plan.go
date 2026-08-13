package mysql

import (
	"encoding/json"
	"strconv"
)

// GetEstimatedAffectedRowsFromExplainJSON estimates the number of rows affected by an
// UPDATE/DELETE statement from `EXPLAIN FORMAT=JSON` output.
//
// MySQL and MariaDB flag the plan node of each DML target table with "update"/"delete"
// (boolean true on MySQL, integer 1 on MariaDB). Unlike the first "rows" value of the
// tabular EXPLAIN output — which may be the scan estimate of a driving table unrelated
// to the target — the target node carries the optimizer's cardinality estimate for the
// rows actually reaching the modify step. Estimates of multiple target nodes
// (multi-table DELETE/UPDATE) are summed.
//
// The boolean result reports whether any flagged target node with a row estimate was
// found; callers should fall back to the tabular-EXPLAIN heuristic when it is false.
func GetEstimatedAffectedRowsFromExplainJSON(plan string) (int64, bool) {
	var root any
	if err := json.Unmarshal([]byte(plan), &root); err != nil {
		return 0, false
	}
	var total float64
	found := false
	var walk func(node any)
	walk = func(node any) {
		switch n := node.(type) {
		case map[string]any:
			if isDMLTargetNode(n) {
				if estimate, ok := targetNodeRowEstimate(n); ok {
					total += estimate
					found = true
				}
			}
			for _, v := range n {
				walk(v)
			}
		case []any:
			for _, v := range n {
				walk(v)
			}
		default:
		}
	}
	walk(root)
	if !found {
		return 0, false
	}
	return int64(total), true
}

func isDMLTargetNode(node map[string]any) bool {
	for _, key := range []string{"update", "delete"} {
		switch v := node[key].(type) {
		case bool:
			if v {
				return true
			}
		case float64:
			if v != 0 {
				return true
			}
		default:
		}
	}
	return false
}

// targetNodeRowEstimate prefers "rows_produced_per_join", the cumulative cardinality
// after all join and filter conditions up to this node. When absent (single-table
// plans, MariaDB), it scales the per-scan row estimate by the node's "filtered"
// percentage.
func targetNodeRowEstimate(node map[string]any) (float64, bool) {
	if v, ok := toFloat(node["rows_produced_per_join"]); ok {
		return v, true
	}
	base, ok := toFloat(node["rows_examined_per_scan"])
	if !ok {
		// MariaDB names the per-scan estimate "rows".
		base, ok = toFloat(node["rows"])
	}
	if !ok {
		return 0, false
	}
	if filtered, ok := toFloat(node["filtered"]); ok {
		base = base * filtered / 100
	}
	return base, true
}

func toFloat(v any) (float64, bool) {
	switch value := v.(type) {
	case float64:
		return value, true
	case string:
		// MySQL renders "filtered" as a string such as "10.00".
		f, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return 0, false
		}
		return f, true
	default:
		return 0, false
	}
}
