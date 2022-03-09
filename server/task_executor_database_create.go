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

// NewDatabaseCreateTaskExecutor creates a database create task executor.
func NewDatabaseCreateTaskExecutor(logger *zap.Logger) TaskExecutor {
	return &DatabaseCreateTaskExecutor{
		l: logger,
	}
}

// DatabaseCreateTaskExecutor is the database create task executor.
type DatabaseCreateTaskExecutor struct {
	l *zap.Logger
}

// RunOnce will run the database create task executor once.
func (exec *DatabaseCreateTaskExecutor) RunOnce(ctx context.Context, server *Server, task *api.Task) (terminated bool, result *api.TaskRunResultPayload, err error) {
	defer func() {
		if r := recover(); r != nil {
			panicErr, ok := r.(error)
			if !ok {
				panicErr = fmt.Errorf("%v", r)
			}
			exec.l.Error("DatabaseCreateTaskExecutor PANIC RECOVER", zap.Error(panicErr), zap.Stack("stack"))
			terminated = true
			err = fmt.Errorf("encounter internal error when creating database")
		}
	}()

	payload := &api.TaskDatabaseCreatePayload{}
	if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
		return true, nil, fmt.Errorf("invalid create database payload: %w", err)
	}

	statement := strings.TrimSpace(payload.Statement)
	if statement == "" {
		return true, nil, fmt.Errorf("empty create database statement")
	}

	if err := server.composeTaskRelationship(ctx, task); err != nil {
		return true, nil, err
	}

	instance := task.Instance
	driver, err := getAdminDatabaseDriver(ctx, task.Instance, "", exec.l)
	if err != nil {
		return true, nil, err
	}
	defer driver.Close(ctx)

	exec.l.Debug("Start creating database...",
		zap.String("instance", instance.Name),
		zap.String("database", payload.DatabaseName),
		zap.String("statement", statement),
	)

	// Create a baseline migration history upon creating the database.
	mi := &db.MigrationInfo{
		ReleaseVersion: server.version,
		Version:        payload.SchemaVersion,
		Namespace:      payload.DatabaseName,
		Database:       payload.DatabaseName,
		Environment:    instance.Environment.Name,
		Source:         db.UI,
		Type:           db.Baseline,
		Description:    "Create database",
		CreateDatabase: true,
	}
	creator, err := server.composePrincipalByID(ctx, task.CreatorID)
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

	issueFind := &api.IssueFind{
		PipelineID: &task.PipelineID,
	}
	issue, err := server.IssueService.FindIssue(ctx, issueFind)
	if err != nil {
		// If somehow we unable to find the issue, we just emit the error since it's not
		// critical enough to fail the entire operation.
		exec.l.Error("Failed to fetch containing issue for composing the migration info",
			zap.Int("task_id", task.ID),
			zap.Error(err),
		)
	}
	if issue == nil {
		err := fmt.Errorf("Failed to fetch containing issue for composing the migration info, issue not found with pipeline ID %v", task.PipelineID)
		exec.l.Error(err.Error(),
			zap.Int("task_id", task.ID),
			zap.Error(err),
		)
	} else {
		mi.IssueID = strconv.Itoa(issue.ID)
	}

	migrationID, _, err := driver.ExecuteMigration(ctx, mi, statement)
	if err != nil {
		return true, nil, err
	}

	// If the database creation statement executed successfully,
	// then we will create a database entry immediately
	// instead of waiting for the next schema sync cycle to sync over this newly created database.
	// This is for 2 reasons:
	// 1. Assign the proper project to the newly created database. Otherwise, the periodic schema
	// sync will place the synced db into the default project.
	// 2. Allow user to see the created database right away.
	databaseCreate := &api.DatabaseCreate{
		CreatorID:     api.SystemBotID,
		ProjectID:     payload.ProjectID,
		InstanceID:    task.InstanceID,
		EnvironmentID: instance.EnvironmentID,
		Name:          payload.DatabaseName,
		CharacterSet:  payload.CharacterSet,
		Collation:     payload.Collation,
		Labels:        &payload.Labels,
		SchemaVersion: payload.SchemaVersion,
	}
	database, err := server.DatabaseService.CreateDatabase(ctx, databaseCreate)
	if err != nil {
		return true, nil, err
	}

	// After the task related database entry created successfully,
	// we need to update task's database_id with the newly created database immediately.
	// Here is the main reason:
	// The task database_id represents its related database entry both for creating and patching,
	// so we should sync its value right here when the related database entry created.
	taskDatabaseIDPatch := &api.TaskPatch{
		ID:         task.ID,
		UpdaterID:  api.SystemBotID,
		DatabaseID: &database.ID,
	}
	_, err = server.TaskService.PatchTask(ctx, taskDatabaseIDPatch)
	if err != nil {
		return true, nil, err
	}

	if payload.Labels != "" {
		// Compose database relationship for setting database labels.
		err = server.composeDatabaseRelationship(ctx, database)
		if err != nil {
			return true, nil, err
		}

		project, err := server.composeProjectByID(ctx, payload.ProjectID)
		if err != nil {
			return true, nil, fmt.Errorf("failed to find project: %v", payload.ProjectID)
		}
		if project == nil {
			return true, nil, fmt.Errorf("project ID not found %v", payload.ProjectID)
		}

		// Set database labels, except bb.environment is immutable and must match instance environment.
		err = server.setDatabaseLabels(ctx, payload.Labels, database, project, database.CreatorID, false)
		if err != nil {
			return true, nil, fmt.Errorf("failed to record database labels after creating database %v", database.ID)
		}
	}

	return true, &api.TaskRunResultPayload{
		Detail:      fmt.Sprintf("Created database %q", payload.DatabaseName),
		MigrationID: migrationID,
		Version:     mi.Version,
	}, nil
}
