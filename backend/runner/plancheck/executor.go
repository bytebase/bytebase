package plancheck

import (
	"context"
	"log/slog"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func runExecutorOnce(ctx context.Context, exec Executor, config *storepb.PlanCheckRunConfig) (results []*storepb.PlanCheckRunResult_Result, err error) {
	defer func() {
		if r := recover(); r != nil {
			panicErr, ok := r.(error)
			if !ok {
				panicErr = errors.Errorf("%v", r)
			}
			slog.Error("planCheckExecutor PANIC RECOVER", log.BBError(panicErr), log.BBStack("panic-stack"))
			err = errors.Errorf("planCheckExecutor PANIC RECOVER, err: %v", panicErr)
		}
	}()

	return exec.Run(ctx, config)
}

// Executor is the plan check executor.
type Executor interface {
	// Run will be called periodically by the plan check scheduler
	Run(ctx context.Context, config *storepb.PlanCheckRunConfig) (results []*storepb.PlanCheckRunResult_Result, err error)
}
