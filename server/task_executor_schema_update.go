package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/external/gitlab"
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

	var repository *api.Repository
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
		repositoryFind := &api.RepositoryFind{
			ProjectId: &task.Database.ProjectId,
		}
		repository, err = server.RepositoryService.FindRepository(context.Background(), repositoryFind)
		if err != nil {
			return true, "", fmt.Errorf("failed to find linked repository for database %q", databaseName)
		}

		mi, err = db.ParseMigrationInfo(
			payload.VCSPushEvent.FileCommit.Added,
			filepath.Join(payload.VCSPushEvent.BaseDirectory, repository.FilePathTemplate),
		)
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

	schema, err := driver.ExecuteMigration(ctx, mi, sql)
	if err != nil {
		return true, "", err
	}

	// If VCS based, then we will write back the latest schema file after migration.
	if payload.VCSPushEvent != nil {
		bytebaseURL := ""
		if issue != nil {
			bytebaseURL = fmt.Sprintf("%s:%d/issue/%s?stage=%d", server.frontendHost, server.frontendPort, api.IssueSlug(issue), task.StageId)
		}

		// Writes back the latest schema file to the same branch as the push event.
		// Ref format refs/heads/<<branch>>
		refComponents := strings.Split(payload.VCSPushEvent.Ref, "/")
		branch := refComponents[len(refComponents)-1]
		latestSchemaFile := fmt.Sprintf("%s__#LATEST.sql", databaseName)
		filePath := fmt.Sprintf("projects/%s/repository/files/%s", repository.ExternalId, url.QueryEscape(latestSchemaFile))
		getFilePath := filePath + "?ref=" + url.QueryEscape(branch)

		repository.VCS, err = server.ComposeVCSById(context.Background(), repository.VCSId)
		if err != nil {
			return true, "", fmt.Errorf("failed to sync schema file %s after applying migration %s to %q", filePath, mi.Version, databaseName)
		}

		getResp, err := gitlab.GET(
			repository.VCS.InstanceURL,
			getFilePath,
			repository.AccessToken,
		)
		if err != nil {
			return true, "", fmt.Errorf("failed to fetch latest schema file from %s, err: %w", repository.VCS.InstanceURL, err)
		}
		defer getResp.Body.Close()

		createSchemaFile := false
		verb := "Update"
		if getResp.StatusCode >= 300 && getResp.StatusCode != 404 {
			return true, "", fmt.Errorf("failed to fetch latest schema file from %s, status code: %d, status: %s",
				repository.VCS.InstanceURL,
				getResp.StatusCode,
				getResp.Status,
			)
		} else if getResp.StatusCode == 404 {
			createSchemaFile = true
			verb = "Create"
		}

		commitTitle := fmt.Sprintf("[Bytebase] %s latest schema for %q after migration %s", verb, databaseName, mi.Version)
		commitBody := "THIS COMMIT IS AUTO-GENERATED BY BTYEBASE"
		if bytebaseURL != "" {
			commitBody += "\n\n" + bytebaseURL
		}
		commitBody += "\n\n--------Original migration change--------\n\n"
		commitBody += fmt.Sprintf("%s\n\n%s",
			payload.VCSPushEvent.FileCommit.URL,
			payload.VCSPushEvent.FileCommit.Message,
		)

		schemaFileCommit := gitlab.FileCommit{
			Branch:        branch,
			CommitMessage: fmt.Sprintf("%s\n\n%s", commitTitle, commitBody),
			Content:       schema,
		}
		if createSchemaFile {
			body, err := json.Marshal(schemaFileCommit)
			if err != nil {
				return true, "", fmt.Errorf("failed to marshal file request %s after applying migration %s to %q", filePath, mi.Version, databaseName)
			}

			resp, err := gitlab.POST(repository.VCS.InstanceURL, filePath, repository.AccessToken, bytes.NewBuffer(body))
			if err != nil {
				return true, "", fmt.Errorf("failed to create file %s after applying migration %s to %q, err: %w", filePath, mi.Version, databaseName, err)
			}
			defer resp.Body.Close()

			if resp.StatusCode >= 300 {
				return true, "", fmt.Errorf("failed to create file %s after applying migration %s to %q, status code: %d, status: %s",
					filePath,
					mi.Version,
					databaseName,
					resp.StatusCode,
					resp.Status,
				)
			}
		} else {
			file := &gitlab.File{}
			if err := json.NewDecoder(getResp.Body).Decode(file); err != nil {
				return true, "", fmt.Errorf("failed to unmarshal file response %s after applying migration %s to %q", filePath, mi.Version, databaseName)
			}

			schemaFileCommit.LastCommitId = file.LastCommitId
			body, err := json.Marshal(schemaFileCommit)
			if err != nil {
				return true, "", fmt.Errorf("failed to marshal file request %s after applying migration %s to %q", filePath, mi.Version, databaseName)
			}

			resp, err := gitlab.PUT(repository.VCS.InstanceURL, filePath, repository.AccessToken, bytes.NewBuffer(body))
			if err != nil {
				return true, "", fmt.Errorf("failed to create file %s after applying migration %s to %q, error: %w", filePath, mi.Version, databaseName, err)
			}
			defer resp.Body.Close()

			if resp.StatusCode >= 300 {
				return true, "", fmt.Errorf("failed to create file %s after applying migration %s to %q, status code: %d, status: %s",
					filePath,
					mi.Version,
					databaseName,
					resp.StatusCode,
					resp.Status,
				)
			}
		}
	}

	detail = fmt.Sprintf("Applied migration version %s to database %q", mi.Version, databaseName)
	if mi.Type == db.Baseline {
		detail = fmt.Sprintf("Established baseline version %s for database %q", mi.Version, databaseName)
	}

	return true, detail, nil
}
