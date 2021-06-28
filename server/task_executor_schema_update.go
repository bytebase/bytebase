package server

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/db"
	"go.uber.org/zap"
)

func NewSchemaUpdateTaskExecutor(logger *zap.Logger) TaskExecutor {
	return &SchemaUpdateTaskExecutor{
		l: logger,
	}
}

type SchemaUpdateTaskExecutor struct {
	l *zap.Logger
}

func (exec *SchemaUpdateTaskExecutor) RunOnce(ctx context.Context, server *Server, task *api.Task) (terminated bool, err error) {
	defer func() {
		if r := recover(); r != nil {
			panicErr, ok := r.(error)
			if !ok {
				panicErr = fmt.Errorf("%v", r)
			}
			exec.l.Error("SchemaUpdateTaskExecutor PANIC RECOVER", zap.Error(panicErr))
			terminated = true
			err = fmt.Errorf("encounter internal error when executing sql")
		}
	}()

	payload := &api.TaskDatabaseSchemaUpdatePayload{}
	if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
		return true, fmt.Errorf("invalid database schema update payload: %w", err)
	}

	mi := &db.MigrationInfo{
		Type: db.Sql,
	}
	if payload.VCSPushEvent != nil {
		mi, err = db.ParseMigrationInfo(payload.VCSPushEvent.FileCommit.Added, payload.VCSPushEvent.BaseDirectory)
		// This should not happen normally as we already check this when creating the issue. Just in case.
		if err != nil {
			return true, fmt.Errorf("failed to start schema migration, error: %w", err)
		}
		mi.Creator = payload.VCSPushEvent.FileCommit.AuthorName
	}

	sql := strings.TrimSpace(payload.Statement)
	// Only baseline can have empty sql statement, which indicates empty database.
	if mi.Type != db.Baseline && sql == "" {
		return true, fmt.Errorf("empty sql statement")
	}

	if err := server.ComposeTaskRelationship(ctx, task, []string{}); err != nil {
		return true, err
	}

	instance := task.Instance
	databaseName := ""
	if task.Database != nil {
		databaseName = task.Database.Name
	}
	driver, err := db.Open(instance.Engine, db.DriverConfig{Logger: exec.l}, db.ConnectionConfig{
		Username: instance.Username,
		Password: instance.Password,
		Host:     instance.Host,
		Port:     instance.Port,
		Database: databaseName,
	})
	if err != nil {
		return true, fmt.Errorf("failed to connect instance: %v with user: %v. %w", instance.Name, instance.Username, err)
	}

	if payload.VCSPushEvent == nil {
		exec.l.Debug("Start executing sql...",
			zap.String("instance", instance.Name),
			zap.String("database", databaseName),
			zap.String("sql", sql),
		)

		if err := driver.Execute(ctx, sql); err != nil {
			return true, err
		}
	} else {
		exec.l.Debug("Start sql migration...",
			zap.String("instance", instance.Name),
			zap.String("database", databaseName),
			zap.String("type", mi.Type.String()),
			zap.String("sql", sql),
		)

		setup, err := driver.NeedsSetupMigration(ctx)
		if err != nil {
			return true, fmt.Errorf("failed to check migration setup for instance: %v, %w", instance.Name, err)
		}
		if setup {
			return true, fmt.Errorf("missing migration schema for instance: %v", instance.Name)
		}

		if err := driver.ExecuteMigration(ctx, mi, sql); err != nil {
			return true, err
		}
	}

	return true, nil
}
