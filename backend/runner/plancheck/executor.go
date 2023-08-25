package plancheck

import (
	"context"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func runExecutorOnce(ctx context.Context, exec Executor, planCheckRun *store.PlanCheckRunMessage) (results []*storepb.PlanCheckRunResult_Result, err error) {
	defer func() {
		if r := recover(); r != nil {
			panicErr, ok := r.(error)
			if !ok {
				panicErr = errors.Errorf("%v", r)
			}
			log.Error("TaskExecutor PANIC RECOVER", zap.Error(panicErr), zap.Stack("panic-stack"))
			err = errors.Errorf("encounter internal error when executing check")
		}
	}()

	return exec.Run(ctx, planCheckRun)
}

// Executor is the plan check executor.
type Executor interface {
	// Run will be called periodically by the plan check scheduler
	Run(ctx context.Context, planCheckRun *store.PlanCheckRunMessage) (results []*storepb.PlanCheckRunResult_Result, err error)
}
