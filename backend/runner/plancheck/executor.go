package plancheck

import (
	"context"
	"log/slog"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/db"
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

func isStatementTypeCheckSupported(dbType db.Type) bool {
	switch dbType {
	case db.Postgres, db.TiDB, db.MySQL, db.MariaDB, db.OceanBase:
		return true
	default:
		return false
	}
}

func isStatementAdviseSupported(dbType db.Type) bool {
	switch dbType {
	case db.MySQL, db.TiDB, db.Postgres, db.Oracle, db.OceanBase, db.Snowflake, db.MSSQL:
		return true
	default:
		return false
	}
}

func isStatementReportSupported(dbType db.Type) bool {
	switch dbType {
	case db.Postgres, db.MySQL, db.OceanBase, db.Oracle:
		return true
	default:
		return false
	}
}
