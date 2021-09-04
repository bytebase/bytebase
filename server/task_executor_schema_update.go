package server

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/plugin/db"
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

func (exec *SchemaUpdateTaskExecutor) RunOnce(ctx context.Context, server *Server, task *api.Task) (terminated bool, detail string, err error) {
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

	if task.Database == nil {
		return true, "", fmt.Errorf("missing database when updating schema")
	}
	databaseName := task.Database.Name

	payload := &api.TaskDatabaseSchemaUpdatePayload{}
	if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
		return true, "", fmt.Errorf("invalid database schema update payload: %w", err)
	}

	mi := &db.MigrationInfo{
		Type: db.Migrate,
	}
	if payload.VCSPushEvent == nil {
		mi.Engine = db.UI
		creator, err := server.ComposePrincipalById(context.Background(), task.CreatorId)
		if err != nil {
			// If somehow we unable to find the principal, we just emit the error since it's not
			// critical enough to fail the entire operation.
			exec.l.Error("Failed to fetch creator for composing the migration info",
				zap.Int("task_id", task.ID),
				zap.Error(err),
			)
		} else {
			mi.Creator = creator.Name
		}
		mi.Version = defaultMigrationVersionFromTaskId(task.ID)
		mi.Database = databaseName
		mi.Namespace = databaseName
		mi.Description = task.Name
	} else {
		mi, err = db.ParseMigrationInfo(payload.VCSPushEvent.FileCommit.Added, payload.VCSPushEvent.BaseDirectory)
		// This should not happen normally as we already check this when creating the issue. Just in case.
		if err != nil {
			return true, "", fmt.Errorf("failed to start schema migration, error: %w", err)
		}
		mi.Creator = payload.VCSPushEvent.FileCommit.AuthorName

		miPayload := &db.MigrationInfoPayload{
			VCSPushEvent: payload.VCSPushEvent,
		}
		bytes, err := json.Marshal(miPayload)
		if err != nil {
			return true, "", fmt.Errorf("failed to start schema migration, unable to marshal vcs push event payload %w", err)
		}
		mi.Payload = string(bytes)
	}

	issueFind := &api.IssueFind{
		PipelineId: &task.PipelineId,
	}
	issue, err := server.IssueService.FindIssue(ctx, issueFind)
	if err != nil {
		// If somehow we unable to find the issue, we just emit the error since it's not
		// critical enough to fail the entire operation.
		exec.l.Error("Failed to fetch containing issue for composing the migration info",
			zap.Int("task_id", task.ID),
			zap.Error(err),
		)
	} else {
		mi.IssueId = strconv.Itoa(issue.ID)
	}

	sql := strings.TrimSpace(payload.Statement)
	// Only baseline can have empty sql statement, which indicates empty database.
	if mi.Type != db.Baseline && sql == "" {
		return true, "", fmt.Errorf("empty sql statement")
	}

	if err := server.ComposeTaskRelationship(ctx, task); err != nil {
		return true, "", err
	}

	driver, err := GetDatabaseDriver(task.Instance, databaseName, exec.l)
	if err != nil {
		return true, "", err
	}
	defer driver.Close(context.Background())

	exec.l.Debug("Start sql migration...",
		zap.String("instance", task.Instance.Name),
		zap.String("database", databaseName),
		zap.String("engine", mi.Engine.String()),
		zap.String("type", mi.Type.String()),
		zap.String("sql", sql),
	)

	setup, err := driver.NeedsSetupMigration(ctx)
	if err != nil {
		return true, "", fmt.Errorf("failed to check migration setup for instance %q: %w", task.Instance.Name, err)
	}
	if setup {
		return true, "", fmt.Errorf("missing migration schema for instance %q", task.Instance.Name)
	}

	if err := driver.ExecuteMigration(ctx, mi, sql); err != nil {
		return true, "", err
	}

	detail = fmt.Sprintf("Applied migration version %s to database %q", mi.Version, databaseName)
	if mi.Type == db.Baseline {
		detail = fmt.Sprintf("Established baseline version %s for database %q", mi.Version, databaseName)
	}

	return true, detail, nil
}
