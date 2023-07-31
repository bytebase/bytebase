package plancheck

import (
	"context"

	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// Executor is the plan check executor.
type Executor interface {
	// Run will be called periodically by the plan check scheduler
	Run(ctx context.Context, planCheckRun *store.PlanCheckRunMessage) (results []*storepb.PlanCheckRunResult_Result, err error)
}
