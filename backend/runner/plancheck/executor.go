package plancheck

import (
	"context"
	"log/slog"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func runExecutorOnce(ctx context.Context, exec Executor, config *storepb.PlanCheckRunConfig) (results []*storepb.PlanCheckRunResult_Result, err error) {
	defer func() {
		if r := recover(); r != nil {
			panicErr, ok := r.(error)
			if !ok {
				panicErr = errors.Errorf("%v", r)
			}
			slog.Error("TaskExecutor PANIC RECOVER", log.BBError(panicErr), log.BBStack("panic-stack"))
			err = errors.Errorf("encounter internal error when executing check")
		}
	}()

	return exec.Run(ctx, config)
}

// Executor is the plan check executor.
type Executor interface {
	// Run will be called periodically by the plan check scheduler
	Run(ctx context.Context, config *storepb.PlanCheckRunConfig) (results []*storepb.PlanCheckRunResult_Result, err error)
}

func isStatementTypeCheckSupported(dbType storepb.Engine) bool {
	switch dbType {
	case storepb.Engine_POSTGRES, storepb.Engine_TIDB, storepb.Engine_MYSQL, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
		return true
	default:
		return false
	}
}

func isStatementAdviseSupported(dbType storepb.Engine) bool {
	switch dbType {
	case storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_POSTGRES, storepb.Engine_ORACLE, storepb.Engine_OCEANBASE_ORACLE, storepb.Engine_OCEANBASE, storepb.Engine_SNOWFLAKE, storepb.Engine_MSSQL:
		return true
	default:
		return false
	}
}

func isStatementReportSupported(dbType storepb.Engine) bool {
	switch dbType {
	case storepb.Engine_POSTGRES, storepb.Engine_MYSQL, storepb.Engine_OCEANBASE, storepb.Engine_ORACLE, storepb.Engine_OCEANBASE_ORACLE:
		return true
	default:
		return false
	}
}
