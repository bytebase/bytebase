package oceanbase

import "github.com/bytebase/omni/oceanbase/analysis"

// EstimateAffectedRowsFromExplainJSON returns the EST.ROWS estimate from OceanBase
// `EXPLAIN FORMAT=JSON` output.
func EstimateAffectedRowsFromExplainJSON(plan string) (int64, error) {
	return analysis.EstimateAffectedRowsFromExplainJSON(plan)
}
