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
	vcsPlugin "github.com/bytebase/bytebase/plugin/vcs"
	"go.uber.org/zap"
)

// TaskExecutor is the task executor.
type TaskExecutor interface {
	// RunOnce will be called periodically by the scheduler until terminated is true.
	//
	// NOTE
	//
	// 1. It's possible that err could be non-nil while terminated is false, which
	// usually indicates a transient error and will make scheduler retry later.
	// 2. If err is non-nil, then the detail field will be ignored since info is provided in the err.
	RunOnce(ctx context.Context, server *Server, task *api.Task) (terminated bool, result *api.TaskRunResultPayload, err error)
}

func preMigration(ctx context.Context, l *zap.Logger, server *Server, task *api.Task, migrationType db.MigrationType, statement, schemaVersion string, vcsPushEvent *vcsPlugin.PushEvent) (*db.MigrationInfo, error) {
	if task.Database == nil {
		msg := "missing database when updating schema"
		if migrationType == db.Data {
			msg = "missing database when updating data"
		}
		return nil, fmt.Errorf(msg)
	}
	databaseName := task.Database.Name

	var repoRawOutter *api.RepositoryRaw
	mi := &db.MigrationInfo{
		ReleaseVersion: server.version,
		Type:           migrationType,
	}
	if vcsPushEvent == nil {
		mi.Source = db.UI
		creator, err := server.store.GetPrincipalByID(ctx, task.CreatorID)
		if err != nil {
			// If somehow we unable to find the principal, we just emit the error since it's not
			// critical enough to fail the entire operation.
			l.Error("Failed to fetch creator for composing the migration info",
				zap.Int("task_id", task.ID),
				zap.Error(err),
			)
		} else {
			mi.Creator = creator.Name
		}
		// TODO(d): support semantic versioning.
		mi.Version = schemaVersion
		mi.Description = task.Name
	} else {
		repoFind := &api.RepositoryFind{
			ProjectID: &task.Database.ProjectID,
		}
		repoRawInner, err := server.RepositoryService.FindRepository(ctx, repoFind)
		if err != nil {
			return nil, fmt.Errorf("failed to find linked repository for database %q", databaseName)
		}
		if repoRawInner == nil {
			return nil, fmt.Errorf("repository not found with project ID %v", task.Database.ProjectID)
		}
		repoRawOutter = repoRawInner

		mi, err = db.ParseMigrationInfo(
			vcsPushEvent.FileCommit.Added,
			filepath.Join(vcsPushEvent.BaseDirectory, repoRawOutter.FilePathTemplate),
		)
		// This should not happen normally as we already check this when creating the issue. Just in case.
		if err != nil {
			return nil, fmt.Errorf("failed to start migration, error: %w", err)
		}
		mi.Creator = vcsPushEvent.FileCommit.AuthorName

		miPayload := &db.MigrationInfoPayload{
			VCSPushEvent: vcsPushEvent,
		}
		bytes, err := json.Marshal(miPayload)
		if err != nil {
			return nil, fmt.Errorf("failed to start migration, unable to marshal vcs push event payload %w", err)
		}
		mi.Payload = string(bytes)
	}

	mi.Database = databaseName
	mi.Namespace = databaseName

	issueFind := &api.IssueFind{
		PipelineID: &task.PipelineID,
	}
	issueRaw, err := server.IssueService.FindIssue(ctx, issueFind)
	if err != nil {
		// If somehow we cannot find the issue, emit the error since it's not fatal.
		l.Error("Failed to fetch containing issue for composing the migration info",
			zap.Int("task_id", task.ID),
			zap.Error(err),
		)
	}
	var issue *api.Issue
	if issueRaw == nil {
		err := fmt.Errorf("failed to fetch containing issue for composing the migration info, issue not found with pipeline ID %v", task.PipelineID)
		l.Error(err.Error(),
			zap.Int("task_id", task.ID),
			zap.Error(err),
		)
	} else {
		issue, err = server.composeIssueRelationship(ctx, issueRaw)
		if err != nil {
			return nil, err
		}
		mi.IssueID = strconv.Itoa(issue.ID)
	}

	statement = strings.TrimSpace(statement)
	// Only baseline can have empty sql statement, which indicates empty database.
	if mi.Type != db.Baseline && statement == "" {
		return nil, fmt.Errorf("empty statement")
	}

	return mi, nil
}

func executeMigration(ctx context.Context, l *zap.Logger, task *api.Task, statement string, mi *db.MigrationInfo) (migrationID int64, schema string, err error) {

	statement = strings.TrimSpace(statement)
	databaseName := task.Database.Name

	driver, err := getAdminDatabaseDriver(ctx, task.Instance, databaseName, l)
	if err != nil {
		return 0, "", err
	}
	defer driver.Close(ctx)

	l.Debug("Start sql migration...",
		zap.String("instance", task.Instance.Name),
		zap.String("database", databaseName),
		zap.String("source", mi.Source.String()),
		zap.String("type", mi.Type.String()),
		zap.String("statement", statement),
	)

	setup, err := driver.NeedsSetupMigration(ctx)
	if err != nil {
		return 0, "", fmt.Errorf("failed to check migration setup for instance %q: %w", task.Instance.Name, err)
	}
	if setup {
		return 0, "", common.Errorf(common.MigrationSchemaMissing, fmt.Errorf("missing migration schema for instance %q", task.Instance.Name))
	}

	migrationID, schema, err = driver.ExecuteMigration(ctx, mi, statement)
	if err != nil {
		return 0, "", err
	}
	return migrationID, schema, nil
}

func postMigration(ctx context.Context, l *zap.Logger, server *Server, task *api.Task, vcsPushEvent *vcsPlugin.PushEvent, mi *db.MigrationInfo, migrationID int64, schema string) (bool, *api.TaskRunResultPayload, error) {
	databaseName := task.Database.Name
	issueFind := &api.IssueFind{
		PipelineID: &task.PipelineID,
	}
	issueRaw, err := server.IssueService.FindIssue(ctx, issueFind)
	if err != nil {
		// If somehow we cannot find the issue, emit the error since it's not fatal.
		l.Error("failed to fetch containing issue for composing the migration info",
			zap.Int("task_id", task.ID),
			zap.Error(err),
		)
	}
	var issue *api.Issue
	if issueRaw == nil {
		err := fmt.Errorf("failed to fetch containing issue for composing the migration info, issue not found with pipeline ID %v", task.PipelineID)
		l.Error(err.Error(),
			zap.Int("task_id", task.ID),
			zap.Error(err),
		)
	} else {
		issue, err = server.composeIssueRelationship(ctx, issueRaw)
		if err != nil {
			return true, nil, err
		}
	}
	var repoRawOutter *api.RepositoryRaw
	if vcsPushEvent != nil {
		repoFind := &api.RepositoryFind{
			ProjectID: &task.Database.ProjectID,
		}
		repoRawInner, err := server.RepositoryService.FindRepository(ctx, repoFind)
		if err != nil {
			return true, nil, fmt.Errorf("failed to find linked repository for database %q", databaseName)
		}
		if repoRawInner == nil {
			return true, nil, fmt.Errorf("repository not found with project ID %v", task.Database.ProjectID)
		}
		repoRawOutter = repoRawInner
	}
	// If VCS based and schema path template is specified, then we will write back the latest schema file after migration.
	writeBack := (vcsPushEvent != nil) && (repoRawOutter.SchemaPathTemplate != "")
	// For tenant mode project, we will only write back latest schema file on the last task.
	project, err := server.store.GetProjectByID(ctx, task.Database.ProjectID)
	if err != nil {
		return true, nil, err
	}
	if writeBack && issue != nil {
		if project.TenantMode == api.TenantModeTenant {
			var lastTask *api.Task
			for i := len(issue.Pipeline.StageList) - 1; i >= 0; i-- {
				stage := issue.Pipeline.StageList[i]
				if len(stage.TaskList) > 0 {
					lastTask = stage.TaskList[len(stage.TaskList)-1]
					break
				}
			}
			// Not the last task yet.
			if lastTask != nil && task.ID != lastTask.ID {
				writeBack = false
			}
		}
	}

	if writeBack {
		dbName, err := api.GetBaseDatabaseName(mi.Database, project.DBNameTemplate, task.Database.Labels)
		if err != nil {
			return true, nil, fmt.Errorf("failed to get BaseDatabaseName for instance %q, database %q: %w", task.Instance.Name, task.Database.Name, err)
		}
		latestSchemaFile := filepath.Join(repoRawOutter.BaseDirectory, repoRawOutter.SchemaPathTemplate)
		latestSchemaFile = strings.ReplaceAll(latestSchemaFile, "{{ENV_NAME}}", mi.Environment)
		latestSchemaFile = strings.ReplaceAll(latestSchemaFile, "{{DB_NAME}}", dbName)

		// TODO(dragonly): revisit the usage of a not-fully-composed Repository here
		repo := repoRawOutter.ToRepository()
		vcs, err := server.store.GetVCSByID(ctx, repoRawOutter.VCSID)
		if err != nil {
			return true, nil, fmt.Errorf("failed to sync schema file %s after applying migration %s to %q", latestSchemaFile, mi.Version, databaseName)
		}
		if vcs == nil {
			return true, nil, fmt.Errorf("VCS ID not found: %d", repoRawOutter.VCSID)
		}
		repo.VCS = vcs

		// Writes back the latest schema file to the same branch as the push event.
		branch, err := vcsPlugin.Branch(vcsPushEvent.Ref)
		if err != nil {
			return true, nil, err
		}

		bytebaseURL := ""
		if issue != nil {
			bytebaseURL = fmt.Sprintf("%s:%d/issue/%s?stage=%d", server.frontendHost, server.frontendPort, api.IssueSlug(issue), task.StageID)
		}

		commitID, err := writeBackLatestSchema(ctx, server, repo, vcsPushEvent, mi, branch, latestSchemaFile, schema, bytebaseURL)
		if err != nil {
			return true, nil, err
		}

		// Create file commit activity
		{
			payload, err := json.Marshal(api.ActivityPipelineTaskFileCommitPayload{
				TaskID:             task.ID,
				VCSInstanceURL:     repo.VCS.InstanceURL,
				RepositoryFullPath: vcsPushEvent.RepositoryFullPath,
				Branch:             branch,
				FilePath:           latestSchemaFile,
				CommitID:           commitID,
			})
			if err != nil {
				l.Error("Failed to marshal file commit activity after writing back the latest schema",
					zap.Int("task_id", task.ID),
					zap.String("repository", repoRawOutter.WebURL),
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
					dbName,
				),
				Payload: string(payload),
			}

			_, err = server.ActivityManager.CreateActivity(ctx, activityCreate, &ActivityMeta{})
			if err != nil {
				l.Error("Failed to create file commit activity after writing back the latest schema",
					zap.Int("task_id", task.ID),
					zap.String("repository", repoRawOutter.WebURL),
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

func runMigration(ctx context.Context, l *zap.Logger, server *Server, task *api.Task, migrationType db.MigrationType, statement, schemaVersion string, vcsPushEvent *vcsPlugin.PushEvent) (terminated bool, result *api.TaskRunResultPayload, err error) {
	mi, err := preMigration(ctx, l, server, task, migrationType, statement, schemaVersion, vcsPushEvent)
	if err != nil {
		return true, nil, err
	}
	migrationID, schema, err := executeMigration(ctx, l, task, statement, mi)
	if err != nil {
		return true, nil, err
	}
	return postMigration(ctx, l, server, task, vcsPushEvent, mi, migrationID, schema)
}

// Writes back the latest schema to the repository after migration
// Returns the commit id on success.
func writeBackLatestSchema(ctx context.Context, server *Server, repository *api.Repository, pushEvent *vcsPlugin.PushEvent, mi *db.MigrationInfo, branch string, latestSchemaFile string, schema string, bytebaseURL string) (string, error) {
	schemaFileMeta, err := vcsPlugin.Get(vcsPlugin.GitLabSelfHost, vcsPlugin.ProviderConfig{Logger: server.l}).ReadFileMeta(
		ctx,
		common.OauthContext{
			ClientID:     repository.VCS.ApplicationID,
			ClientSecret: repository.VCS.Secret,
			AccessToken:  repository.AccessToken,
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
	commitBody := "THIS COMMIT IS AUTO-GENERATED BY BYTEBASE"
	if bytebaseURL != "" {
		commitBody += "\n\n" + bytebaseURL
	}
	commitBody += "\n\n--------Original migration change--------\n\n"
	commitBody += fmt.Sprintf("%s\n\n%s",
		pushEvent.FileCommit.URL,
		pushEvent.FileCommit.Message,
	)

	schemaFileCommit := vcsPlugin.FileCommitCreate{
		Branch:        branch,
		CommitMessage: fmt.Sprintf("%s\n\n%s", commitTitle, commitBody),
		Content:       schema,
	}
	if createSchemaFile {
		err := vcsPlugin.Get(vcsPlugin.GitLabSelfHost, vcsPlugin.ProviderConfig{Logger: server.l}).CreateFile(
			ctx,
			common.OauthContext{
				ClientID:     repository.VCS.ApplicationID,
				ClientSecret: repository.VCS.Secret,
				AccessToken:  repository.AccessToken,
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
		err := vcsPlugin.Get(vcsPlugin.GitLabSelfHost, vcsPlugin.ProviderConfig{Logger: server.l}).OverwriteFile(
			ctx,
			common.OauthContext{
				ClientID:     repository.VCS.ApplicationID,
				ClientSecret: repository.VCS.Secret,
				AccessToken:  repository.AccessToken,
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
	schemaFileMeta, err = vcsPlugin.Get(vcsPlugin.GitLabSelfHost, vcsPlugin.ProviderConfig{Logger: server.l}).ReadFileMeta(
		ctx,
		common.OauthContext{
			ClientID:     repository.VCS.ApplicationID,
			ClientSecret: repository.VCS.Secret,
			AccessToken:  repository.AccessToken,
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
