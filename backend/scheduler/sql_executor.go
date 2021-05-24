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

func (exec *SqlExecutor) Run(ctx context.Context, taskRun TaskRun) (terminated bool, err error) {
	payload := &api.TaskDatabaseSchemaUpdatePayload{}
	if err := json.Unmarshal(taskRun.Payload, payload); err != nil {
		return true, fmt.Errorf("sql executor: invalid payload: %w", err)
	}

	if payload.Sql == "" {
		return true, fmt.Errorf("sql executor: missing sql statement")
	}

	exec.l.Printf("sql executor: run %v", payload.Sql)

	tx, err := exec.db.BeginTx(ctx, nil)
	if err != nil {
		// Transient error
		return false, err
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx, payload.Sql)
	if err != nil {
		return true, err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		// Transient error
		return false, err
	}

	exec.l.Printf("sql executor: rows affected %v", rows)

	return true, nil
}
