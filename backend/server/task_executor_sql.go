package server

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bytebase/bytebase/api"
	"go.uber.org/zap"
)

func NewSqlTaskExecutor(logger *zap.Logger) TaskExecutor {
	return &SqlTaskExecutor{
		l: logger,
	}
}

type SqlTaskExecutor struct {
	l *zap.Logger
}

func (exec *SqlTaskExecutor) Run(ctx context.Context, server *Server, taskRun api.TaskRun) (terminated bool, err error) {
	payload := &api.TaskDatabaseSchemaUpdatePayload{}
	if err := json.Unmarshal(taskRun.Payload, payload); err != nil {
		return true, fmt.Errorf("sql executor: invalid payload: %w", err)
	}

	if payload.Sql == "" {
		return true, fmt.Errorf("sql executor: missing sql statement")
	}

	exec.l.Info(fmt.Sprintf("sql executor: run %v", payload.Sql))

	// tx, err := exec.db.BeginTx(ctx, nil)
	// if err != nil {
	// 	// Transient error
	// 	return false, err
	// }
	// defer tx.Rollback()

	// res, err := tx.ExecContext(ctx, payload.Sql)
	// if err != nil {
	// 	return true, err
	// }

	// rows, err := res.RowsAffected()
	// if err != nil {
	// 	// Transient error
	// 	return false, err
	// }

	// exec.l.Printf("sql executor: rows affected %v", rows)

	return true, nil
}
