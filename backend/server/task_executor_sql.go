package server

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/db"
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

	task, err := server.ComposeTaskById(ctx, taskRun.TaskId, []string{SECRET_KEY})
	if err != nil {
		return true, err
	}

	instance := task.Database.Instance
	db, err := db.Open(instance.Engine, db.DriverConfig{Logger: exec.l}, db.ConnectionConfig{
		Username: instance.Username,
		Password: instance.Password,
		Host:     instance.Host,
		Port:     instance.Port,
		Database: task.Database.Name,
	})
	if err != nil {
		return true, fmt.Errorf("failed to connect instance: %v with user: %v. %w", instance.Name, instance.Username, err)
	}

	_, err = db.Execute(ctx, payload.Sql)
	if err != nil {
		return true, err
	}

	return true, nil
}
