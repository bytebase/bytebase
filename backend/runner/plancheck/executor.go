package plancheck

import (
	"context"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// Executor is the plan check executor.
type Executor interface {
	// Run will be called periodically by the plan check scheduler
	Run(ctx context.Context, config *storepb.PlanCheckRunConfig) (results []*storepb.PlanCheckRunResult_Result, err error)
}
