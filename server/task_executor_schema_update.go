package server

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/plugin/vcs"
	"go.uber.org/zap"
)

// NewSchemaUpdateTaskExecutor creates a schema update task executor.
func NewSchemaUpdateTaskExecutor(logger *zap.Logger) TaskExecutor {
	return &SchemaUpdateTaskExecutor{
		l: logger,
	}
}

// SchemaUpdateTaskExecutor is the schema update task executor.
type SchemaUpdateTaskExecutor struct {
	l *zap.Logger
}

// RunOnce will run the schema update task executor once.
func (exec *SchemaUpdateTaskExecutor) RunOnce(ctx context.Context, server *Server, task *api.Task) (terminated bool, result *api.TaskRunResultPayload, err error) {
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
		return true, nil, fmt.Errorf("missing database when updating schema")
	}
	databaseName := task.Database.Name

	payload := &api.TaskDatabaseSchemaUpdatePayload{}
	if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
		return true, nil, fmt.Errorf("invalid database schema update payload: %w", err)
	}

	var repository *api.Repository
	mi := &db.MigrationInfo{
		ReleaseVersion: server.version,
		Type:           payload.MigrationType,
	}
	if payload.VCSPushEvent == nil {
		mi.Engine = db.UI
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
		mi.Version = defaultMigrationVersionFromTaskID(task.ID)
		mi.Database = databaseName
		mi.Namespace = databaseName
		mi.Description = task.Name
	} else {
		repositoryFind := &api.RepositoryFind{
			ProjectID: &task.Database.ProjectID,
		}
		repository, err = server.RepositoryService.FindRepository(ctx, repositoryFind)
		if err != nil {
			return true, nil, fmt.Errorf("failed to find linked repository for database %q", databaseName)
		}
		if repository == nil {
			return true, nil, fmt.Errorf("repository not found with project ID %v", task.Database.ProjectID)
		}

		mi, err = db.ParseMigrationInfo(
			payload.VCSPushEvent.FileCommit.Added,
			filepath.Join(payload.VCSPushEvent.BaseDirectory, repository.FilePathTemplate),
		)
		// This should not happen normally as we already check this when creating the issue. Just in case.
		if err != nil {
			return true, nil, fmt.Errorf("failed to start schema migration, error: %w", err)
		}
		mi.Creator = payload.VCSPushEvent.FileCommit.AuthorName

		miPayload := &db.MigrationInfoPayload{
			VCSPushEvent: payload.VCSPushEvent,
		}
		bytes, err := json.Marshal(miPayload)
		if err != nil {
			return true, nil, fmt.Errorf("failed to start schema migration, unable to marshal vcs push event payload %w", err)
		}
		mi.Payload = string(bytes)
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

	statement := strings.TrimSpace(payload.Statement)
	// Only baseline can have empty sql statement, which indicates empty database.
	if mi.Type != db.Baseline && statement == "" {
		return true, nil, fmt.Errorf("empty statement")
	}

	if err := server.composeTaskRelationship(ctx, task); err != nil {
		return true, nil, err
	}

	driver, err := getDatabaseDriver(ctx, task.Instance, databaseName, exec.l)
	if err != nil {
		return true, nil, err
	}
	defer driver.Close(ctx)

	exec.l.Debug("Start sql migration...",
		zap.String("instance", task.Instance.Name),
		zap.String("database", databaseName),
		zap.String("engine", mi.Engine.String()),
		zap.String("type", mi.Type.String()),
		zap.String("statement", statement),
	)

	setup, err := driver.NeedsSetupMigration(ctx)
	if err != nil {
		return true, nil, fmt.Errorf("failed to check migration setup for instance %q: %w", task.Instance.Name, err)
	}
	if setup {
		return true, nil, common.Errorf(common.MigrationSchemaMissing, fmt.Errorf("missing migration schema for instance %q", task.Instance.Name))
	}

	migrationID, schema, err := driver.ExecuteMigration(ctx, mi, statement)
	if err != nil {
		return true, nil, err
	}

	// If VCS based and schema path template is specified, then we will write back the latest schema file after migration.
	if payload.VCSPushEvent != nil && repository.SchemaPathTemplate != "" {
		latestSchemaFile := filepath.Join(repository.BaseDirectory, repository.SchemaPathTemplate)
		latestSchemaFile = strings.ReplaceAll(latestSchemaFile, "{{ENV_NAME}}", mi.Environment)
		latestSchemaFile = strings.ReplaceAll(latestSchemaFile, "{{DB_NAME}}", mi.Database)

		repository.VCS, err = server.composeVCSByID(ctx, repository.VCSID)
		if err != nil {
			return true, nil, fmt.Errorf("failed to sync schema file %s after applying migration %s to %q", latestSchemaFile, mi.Version, databaseName)
		}
		if repository.VCS == nil {
			return true, nil, fmt.Errorf("VCS ID not found: %d", repository.VCSID)
		}

		// Writes back the latest schema file to the same branch as the push event.
		// Ref format refs/heads/<<branch>>
		refComponents := strings.Split(payload.VCSPushEvent.Ref, "/")
		branch := refComponents[len(refComponents)-1]

		bytebaseURL := ""
		if issue != nil {
			bytebaseURL = fmt.Sprintf("%s:%d/issue/%s?stage=%d", server.frontendHost, server.frontendPort, api.IssueSlug(issue), task.StageID)
		}

		commitID, err := writeBackLatestSchema(ctx, server, repository, payload.VCSPushEvent, mi, branch, latestSchemaFile, schema, bytebaseURL)
		if err != nil {
			return true, nil, err
		}

		// Create file commit activity
		{
			payload, err := json.Marshal(api.ActivityPipelineTaskFileCommitPayload{
				TaskID:             task.ID,
				VCSInstanceURL:     repository.VCS.InstanceURL,
				RepositoryFullPath: payload.VCSPushEvent.RepositoryFullPath,
				Branch:             branch,
				FilePath:           latestSchemaFile,
				CommitID:           commitID,
			})
			if err != nil {
				exec.l.Error("Failed to marshal file commit activity after writing back the latest schema",
					zap.Int("task_id", task.ID),
					zap.String("repository", repository.WebURL),
					zap.String("file_path", latestSchemaFile),
					zap.Error(err),
				)
			}

			containerID := task.PipelineID
			if issue != nil {
				containerID = issue.ID
			}
			activityCreate := &api.ActivityCreate{
				CreatorID:   task.CreatorID,
				ContainerID: containerID,
				Type:        api.ActivityPipelineTaskFileCommit,
				Level:       api.ActivityInfo,
				Comment: fmt.Sprintf("Committed the latest schema after applying migration version %s to %q.",
					mi.Version,
					mi.Database,
				),
				Payload: string(payload),
			}

			_, err = server.ActivityManager.CreateActivity(ctx, activityCreate, &ActivityMeta{})
			if err != nil {
				exec.l.Error("Failed to create file commit activity after writing back the latest schema",
					zap.Int("task_id", task.ID),
					zap.String("repository", repository.WebURL),
					zap.String("file_path", latestSchemaFile),
					zap.Error(err),
				)
			}
		}
	}

	detail := fmt.Sprintf("Applied migration version %s to database %q.", mi.Version, databaseName)
	if mi.Type == db.Baseline {
		detail = fmt.Sprintf("Established baseline version %s for database %q.", mi.Version, databaseName)
	}

	return true, &api.TaskRunResultPayload{
		Detail:      detail,
		MigrationID: migrationID,
		Version:     mi.Version,
	}, nil
}

// Writes back the latest schema to the repository after migration
// Returns the commit id on success.
func writeBackLatestSchema(ctx context.Context, server *Server, repository *api.Repository, pushEvent *vcs.VCSPushEvent, mi *db.MigrationInfo, branch string, latestSchemaFile string, schema string, bytebaseURL string) (string, error) {
	schemaFileMeta, err := vcs.Get(vcs.GitLabSelfHost, vcs.ProviderConfig{Logger: server.l}).ReadFileMeta(
		ctx,
		common.OauthContext{
			ClientID:     repository.VCS.ApplicationID,
			ClientSecret: repository.VCS.Secret,
			RefreshToken: repository.RefreshToken,
			Refresher:    server.refreshToken(ctx, repository.ID),
		},
		repository.VCS.InstanceURL,
		repository.ExternalID,
		latestSchemaFile,
		branch,
	)

	createSchemaFile := false
	verb := "Update"
	if err != nil {
		if common.ErrorCode(err) == common.NotFound {
			createSchemaFile = true
			verb = "Create"
		} else {
			return "", fmt.Errorf("failed to fetch latest schema: %w", err)
		}
	}

	commitTitle := fmt.Sprintf("[Bytebase] %s latest schema for %q after migration %s", verb, mi.Database, mi.Version)
	commitBody := "THIS COMMIT IS AUTO-GENERATED BY BTYEBASE"
	if bytebaseURL != "" {
		commitBody += "\n\n" + bytebaseURL
	}
	commitBody += "\n\n--------Original migration change--------\n\n"
	commitBody += fmt.Sprintf("%s\n\n%s",
		pushEvent.FileCommit.URL,
		pushEvent.FileCommit.Message,
	)

	schemaFileCommit := vcs.FileCommitCreate{
		Branch:        branch,
		CommitMessage: fmt.Sprintf("%s\n\n%s", commitTitle, commitBody),
		Content:       schema,
	}
	if createSchemaFile {
		err := vcs.Get(vcs.GitLabSelfHost, vcs.ProviderConfig{Logger: server.l}).CreateFile(
			ctx,
			common.OauthContext{
				ClientID:     repository.VCS.ApplicationID,
				ClientSecret: repository.VCS.Secret,
				RefreshToken: repository.RefreshToken,
				Refresher:    server.refreshToken(ctx, repository.ID),
			},
			repository.VCS.InstanceURL,
			repository.ExternalID,
			latestSchemaFile,
			schemaFileCommit,
		)

		if err != nil {
			return "", fmt.Errorf("failed to create file after applying migration %s to %q: %w", mi.Version, mi.Database, err)
		}
	} else {
		schemaFileCommit.LastCommitID = schemaFileMeta.LastCommitID
		err := vcs.Get(vcs.GitLabSelfHost, vcs.ProviderConfig{Logger: server.l}).OverwriteFile(
			ctx,
			common.OauthContext{
				ClientID:     repository.VCS.ApplicationID,
				ClientSecret: repository.VCS.Secret,
				RefreshToken: repository.RefreshToken,
				Refresher:    server.refreshToken(ctx, repository.ID),
			},
			repository.VCS.InstanceURL,
			repository.ExternalID,
			latestSchemaFile,
			schemaFileCommit,
		)
		if err != nil {
			return "", fmt.Errorf("failed to create file after applying migration %s to %q: %w", mi.Version, mi.Database, err)
		}
	}

	// VCS such as GitLab API doesn't return the commit on write, so we have to call ReadFileMeta again
	schemaFileMeta, err = vcs.Get(vcs.GitLabSelfHost, vcs.ProviderConfig{Logger: server.l}).ReadFileMeta(
		ctx,
		common.OauthContext{
			ClientID:     repository.VCS.ApplicationID,
			ClientSecret: repository.VCS.Secret,
			RefreshToken: repository.RefreshToken,
			Refresher:    server.refreshToken(ctx, repository.ID),
		},
		repository.VCS.InstanceURL,
		repository.ExternalID,
		latestSchemaFile,
		branch,
	)

	if err != nil {
		return "", fmt.Errorf("failed to fetch latest schema file %s after update: %w", latestSchemaFile, err)
	}
	return schemaFileMeta.LastCommitID, nil
}
