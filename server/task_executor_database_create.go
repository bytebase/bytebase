package server

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bytebase/bytebase/api"
	"go.uber.org/zap"
)

func NewDatabaseCreateTaskExecutor(logger *zap.Logger) TaskExecutor {
	return &DatabaseCreateTaskExecutor{
		l: logger,
	}
}

type DatabaseCreateTaskExecutor struct {
	l *zap.Logger
}

func (exec *DatabaseCreateTaskExecutor) RunOnce(ctx context.Context, server *Server, task *api.Task) (terminated bool, detail string, err error) {
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
		return true, "", fmt.Errorf("invalid create database payload: %w", err)
	}

	if err := server.ComposeTaskRelationship(ctx, task); err != nil {
		return true, "", err
	}

	instance := task.Instance
	driver, err := GetDatabaseDriver(task.Instance, "", exec.l)
	if err != nil {
		return true, "", err
	}
	defer driver.Close(context.Background())

	exec.l.Debug("Start creating database...",
		zap.String("instance", instance.Name),
		zap.String("database", payload.DatabaseName),
		zap.String("sql", payload.Statement),
	)

	if err := driver.Execute(ctx, payload.Statement); err != nil {
		return true, "", err
	}

	return true, fmt.Sprintf("Created database %q", payload.DatabaseName), nil
}
