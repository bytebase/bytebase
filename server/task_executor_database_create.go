package server

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/db"
	"go.uber.org/zap"
)

func NewDatabaseCreateTaskExecutor(logger *zap.Logger) TaskExecutor {
	return &SchemaUpdateTaskExecutor{
		l: logger,
	}
}

type DatabaseCreateTaskExecutor struct {
	l *zap.Logger
}

func (exec *DatabaseCreateTaskExecutor) RunOnce(ctx context.Context, server *Server, task *api.Task) (terminated bool, err error) {
	defer func() {
		if r := recover(); r != nil {
			panicErr, ok := r.(error)
			if !ok {
				panicErr = fmt.Errorf("%v", r)
			}
			exec.l.Error("DatabaseCreateTaskExecutor PANIC RECOVER", zap.Error(panicErr))
			terminated = true
			err = fmt.Errorf("encounter internal error when creating database")
		}
	}()

	payload := &api.TaskDatabaseCreatePayload{}
	if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
		return true, fmt.Errorf("invalid create database payload: %w", err)
	}

	if err := server.ComposeTaskRelationship(ctx, task, []string{SECRET_KEY}); err != nil {
		return true, err
	}

	instance := task.Instance
	driver, err := db.Open(instance.Engine, db.DriverConfig{Logger: exec.l}, db.ConnectionConfig{
		Username: instance.Username,
		Password: instance.Password,
		Host:     instance.Host,
		Port:     instance.Port,
	})
	if err != nil {
		return true, fmt.Errorf("failed to connect instance: %v with user: %v. %w", instance.Name, instance.Username, err)
	}

	exec.l.Debug("Start creating database...",
		zap.String("instance", instance.Name),
		zap.String("database", payload.DatabaseName),
		zap.String("sql", payload.Statement),
	)

	if err := driver.Execute(ctx, payload.Statement); err != nil {
		return true, err
	}

	return true, nil
}
