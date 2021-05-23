package scheduler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"github.com/bytebase/bytebase/api"
)

func NewSqlExecutor(logger *log.Logger, db *sql.DB) Executor {
	return &SqlExecutor{
		l:  logger,
		db: db,
	}
}

type SqlExecutor struct {
	l  *log.Logger
	db *sql.DB
}

func (exec *SqlExecutor) Run(ctx context.Context, taskRun TaskRun) error {
	payload := &api.TaskDatabaseSchemaUpdatePayload{}
	if err := json.Unmarshal(taskRun.Payload, payload); err != nil {
		return fmt.Errorf("sql executor: invalid payload: %w", err)
	}

	if payload.Sql == "" {
		return fmt.Errorf("sql executor: missing sql statement")
	}

	exec.l.Printf("sql executor: run %v", payload.Sql)

	tx, err := exec.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	row, err := tx.QueryContext(ctx, payload.Sql)
	if err != nil {
		return err
	}
	defer row.Close()

	row.Next()
	var result int
	if err := row.Scan(
		&result,
	); err != nil {
		return err
	}

	exec.l.Printf("sql executor: result %v", result)

	return nil
}
