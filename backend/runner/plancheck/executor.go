package plancheck

import (
	"context"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// Executor is the plan check executor.
type Executor interface {
	// RunForTarget will be called periodically by the plan check scheduler for each target
	RunForTarget(ctx context.Context, target *CheckTarget) (results []*storepb.PlanCheckRunResult_Result, err error)
}
