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

func (exec *SqlTaskExecutor) RunOnce(ctx context.Context, server *Server, task *api.Task) (terminated bool, err error) {
	payload := &api.TaskDatabaseSchemaUpdatePayload{}
	if err := json.Unmarshal(task.Payload, payload); err != nil {
		return true, fmt.Errorf("invalid schema update payload: %w", err)
	}

	if payload.Statement == "" {
		return true, fmt.Errorf("missing sql statement")
	}

	exec.l.Info(fmt.Sprintf("sql executor: run %v", payload.Statement))

	if err := server.ComposeTaskRelationship(ctx, task, []string{}); err != nil {
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

	_, err = db.Execute(ctx, payload.Statement)
	if err != nil {
		return true, err
	}

	return true, nil
}
