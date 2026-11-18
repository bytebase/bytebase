package oceanbase

import (
	"github.com/bytebase/bytebase/backend/plugin/db"
	oceanbaseparser "github.com/bytebase/bytebase/backend/plugin/parser/oceanbase"
)

// estimatedRowsFromPlan reads the estimated row count from an OceanBase
// `EXPLAIN FORMAT=JSON` probe result, which splits the plan across rows.
func estimatedRowsFromPlan(res db.ProbeResult) (int64, error) {
	plan, err := res.Plan()
	if err != nil {
		return 0, err
	}
	return oceanbaseparser.EstimateAffectedRowsFromExplainJSON(plan)
}
