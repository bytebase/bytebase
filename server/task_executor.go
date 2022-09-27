package server

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/common/log"
	"github.com/bytebase/bytebase/plugin/db"
	vcsPlugin "github.com/bytebase/bytebase/plugin/vcs"
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
	// IsCompleted tells the scheduler if the task execution has completed.
	IsCompleted() bool
	// GetProgress returns the task progress.
	GetProgress() api.Progress
}

// RunTaskExecutorOnce wraps a TaskExecutor.RunOnce call with panic recovery.
func RunTaskExecutorOnce(ctx context.Context, exec TaskExecutor, server *Server, task *api.Task) (terminated bool, result *api.TaskRunResultPayload, err error) {
	defer func() {
		if r := recover(); r != nil {
			panicErr, ok := r.(error)
			if !ok {
				panicErr = errors.Errorf("%v", r)
			}
			log.Error("TaskExecutor PANIC RECOVER", zap.Error(panicErr), zap.Stack("panic-stack"))
			terminated = true
			result = nil
			err = errors.Errorf("encounter internal error when executing task")
		}
	}()

	return exec.RunOnce(ctx, server, task)
}

func preMigration(ctx context.Context, server *Server, task *api.Task, migrationType db.MigrationType, statement, schemaVersion string, vcsPushEvent *vcsPlugin.PushEvent) (*db.MigrationInfo, error) {
	if task.Database == nil {
		msg := "missing database when updating schema"
		if migrationType == db.Data {
			msg = "missing database when updating data"
		}
		return nil, errors.Errorf(msg)
	}
	databaseName := task.Database.Name

	mi := &db.MigrationInfo{
		ReleaseVersion: server.profile.Version,
		Type:           migrationType,
		// TODO(d): support semantic versioning.
		Version:     schemaVersion,
		Description: task.Name,
		Environment: task.Instance.Environment.Name,
	}
	if vcsPushEvent == nil {
		mi.Source = db.UI
		creator, err := server.store.GetPrincipalByID(ctx, task.CreatorID)
		if err != nil {
			// If somehow we unable to find the principal, we just emit the error since it's not
			// critical enough to fail the entire operation.
			log.Error("Failed to fetch creator for composing the migration info",
				zap.Int("task_id", task.ID),
				zap.Error(err),
			)
		} else {
			mi.Creator = creator.Name
		}
	} else {
		mi.Source = db.VCS
		mi.Creator = vcsPushEvent.FileCommit.AuthorName
		miPayload := &db.MigrationInfoPayload{
			VCSPushEvent: vcsPushEvent,
		}
		bytes, err := json.Marshal(miPayload)
		if err != nil {
			return nil, errors.Wrap(err, "failed to prepare for database migration, unable to marshal vcs push event payload")
		}
		mi.Payload = string(bytes)
	}

	mi.Database = databaseName
	mi.Namespace = databaseName

	issue, err := findIssueByTask(ctx, server, task)
	if err != nil {
		log.Error("failed to find containing issue", zap.Error(err))
	}
	if issue != nil {
		mi.IssueID = strconv.Itoa(issue.ID)
	}

	statement = strings.TrimSpace(statement)
	// Only baseline can have empty sql statement, which indicates empty database.
	if mi.Type != db.Baseline && statement == "" {
		return nil, errors.Errorf("empty statement")
	}
	// We will force migration for baseline and migrate type of migrations.
	// This usually happens when the previous attempt fails and the client retries the migration.
	if mi.Type == db.Baseline || mi.Type == db.Migrate {
		mi.Force = true
	}

	return mi, nil
}

func executeMigration(ctx context.Context, server *Server, task *api.Task, statement string, mi *db.MigrationInfo) (migrationID int64, schema string, err error) {
	statement = strings.TrimSpace(statement)
	databaseName := task.Database.Name

	driver, err := server.getAdminDatabaseDriver(ctx, task.Instance, databaseName)
	if err != nil {
		return 0, "", err
	}
	defer driver.Close(ctx)

	log.Debug("Start migration...",
		zap.String("instance", task.Instance.Name),
		zap.String("database", databaseName),
		zap.String("source", string(mi.Source)),
		zap.String("type", string(mi.Type)),
		zap.String("statement", statement),
	)

	setup, err := driver.NeedsSetupMigration(ctx)
	if err != nil {
		return 0, "", errors.Wrapf(err, "failed to check migration setup for instance %q", task.Instance.Name)
	}
	if setup {
		return 0, "", common.Errorf(common.MigrationSchemaMissing, "missing migration schema for instance %q", task.Instance.Name)
	}

	migrationID, schema, err = driver.ExecuteMigration(ctx, mi, statement)
	if err != nil {
		return 0, "", err
	}
	return migrationID, schema, nil
}

func postMigration(ctx context.Context, server *Server, task *api.Task, vcsPushEvent *vcsPlugin.PushEvent, mi *db.MigrationInfo, migrationID int64, schema string) (bool, *api.TaskRunResultPayload, error) {
	databaseName := task.Database.Name
	issue, err := findIssueByTask(ctx, server, task)
	if err != nil {
		// If somehow we cannot find the issue, emit the error since it's not fatal.
		log.Error("failed to find containing issue", zap.Error(err))
	}
	var repo *api.Repository
	if vcsPushEvent != nil {
		repo, err = findRepositoryByTask(ctx, server, task)
		if err != nil {
			return true, nil, err
		}
	}

	// We write back the latest schema after migration for VCS-based projects if the
	// schema path template is specified and the schema migration type is not SDL.
	writeBack := (vcsPushEvent != nil) && (repo.Project.SchemaChangeType != api.ProjectSchemaChangeTypeSDL) && (repo.SchemaPathTemplate != "")
	project, err := server.store.GetProjectByID(ctx, task.Database.ProjectID)
	if err != nil {
		return true, nil, err
	}
	if writeBack && issue != nil {
		if project.TenantMode == api.TenantModeTenant {
			// For tenant mode project, we will only write back once and we happen to write back on lastTask done.
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

	log.Debug("Post migration...",
		zap.String("instance", task.Instance.Name),
		zap.String("database", databaseName),
		zap.Bool("writeBack", writeBack),
	)

	if writeBack {
		dbName, err := api.GetBaseDatabaseName(mi.Database, project.DBNameTemplate, task.Database.Labels)
		if err != nil {
			return true, nil, errors.Wrapf(err, "failed to get BaseDatabaseName for instance %q, database %q", task.Instance.Name, task.Database.Name)
		}
		latestSchemaFile := filepath.Join(repo.BaseDirectory, repo.SchemaPathTemplate)
		latestSchemaFile = strings.ReplaceAll(latestSchemaFile, "{{ENV_NAME}}", mi.Environment)
		latestSchemaFile = strings.ReplaceAll(latestSchemaFile, "{{DB_NAME}}", dbName)

		vcs, err := server.store.GetVCSByID(ctx, repo.VCSID)
		if err != nil {
			return true, nil, errors.Errorf("failed to sync schema file %s after applying migration %s to %q", latestSchemaFile, mi.Version, databaseName)
		}
		if vcs == nil {
			return true, nil, errors.Errorf("VCS ID not found: %d", repo.VCSID)
		}
		repo.VCS = vcs

		// Writes back the latest schema file to the same branch as the push event.
		branch, err := vcsPlugin.Branch(vcsPushEvent.Ref)
		if err != nil {
			return true, nil, err
		}

		bytebaseURL := ""
		if issue != nil {
			bytebaseURL = fmt.Sprintf("%s/issue/%s?stage=%d", server.profile.ExternalURL, api.IssueSlug(issue), task.StageID)
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
				log.Error("Failed to marshal file commit activity after writing back the latest schema",
					zap.Int("task_id", task.ID),
					zap.String("repository", repo.WebURL),
					zap.String("file_path", latestSchemaFile),
					zap.Error(err),
				)
			}

			activityCreate := &api.ActivityCreate{
				CreatorID:   task.CreatorID,
				ContainerID: task.PipelineID,
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
				log.Error("Failed to create file commit activity after writing back the latest schema",
					zap.Int("task_id", task.ID),
					zap.String("repository", repo.WebURL),
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

func runMigration(ctx context.Context, server *Server, task *api.Task, migrationType db.MigrationType, statement, schemaVersion string, vcsPushEvent *vcsPlugin.PushEvent) (terminated bool, result *api.TaskRunResultPayload, err error) {
	mi, err := preMigration(ctx, server, task, migrationType, statement, schemaVersion, vcsPushEvent)
	if err != nil {
		return true, nil, err
	}
	migrationID, schema, err := executeMigration(ctx, server, task, statement, mi)
	if err != nil {
		return true, nil, err
	}
	return postMigration(ctx, server, task, vcsPushEvent, mi, migrationID, schema)
}

func findIssueByTask(ctx context.Context, server *Server, task *api.Task) (*api.Issue, error) {
	issue, err := server.store.GetIssueByPipelineID(ctx, task.PipelineID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to fetch containing issue for composing the migration info, task_id: %v", task.ID)
	}
	if issue == nil {
		return nil, errors.Wrapf(err, "failed to fetch containing issue for composing the migration info, issue not found, pipeline ID: %v, task_id: %v", task.PipelineID, task.ID)
	}
	return issue, nil
}

func findRepositoryByTask(ctx context.Context, server *Server, task *api.Task) (*api.Repository, error) {
	repoFind := &api.RepositoryFind{
		ProjectID: &task.Database.ProjectID,
	}
	repo, err := server.store.GetRepository(ctx, repoFind)
	if err != nil {
		return nil, errors.Errorf("failed to find linked repository for database %q", task.Database.Name)
	}
	if repo == nil {
		return nil, errors.Errorf("repository not found with project ID %v", task.Database.ProjectID)
	}
	return repo, nil
}

// Writes back the latest schema to the repository after migration
// Returns the commit id on success.
func writeBackLatestSchema(ctx context.Context, server *Server, repository *api.Repository, pushEvent *vcsPlugin.PushEvent, mi *db.MigrationInfo, branch string, latestSchemaFile string, schema string, bytebaseURL string) (string, error) {
	schemaFileMeta, err := vcsPlugin.Get(repository.VCS.Type, vcsPlugin.ProviderConfig{}).ReadFileMeta(
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
			return "", errors.Wrap(err, "failed to fetch latest schema")
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

	// Retrieve the latest AccessToken and RefreshToken as the previous VCS call may have
	// updated the stored token pair. VCS will fetch and store the new token pair if the
	// existing token pair has expired.
	repo2, err := server.store.GetRepository(ctx, &api.RepositoryFind{ID: &repository.ID})
	if err != nil {
		return "", errors.Wrap(err, "failed to fetch repository for schema write-back")
	}
	if repo2 == nil {
		return "", errors.Errorf("repository not found for schema write-back: %v", repository.ID)
	}

	schemaFileCommit := vcsPlugin.FileCommitCreate{
		Branch:        branch,
		CommitMessage: fmt.Sprintf("%s\n\n%s", commitTitle, commitBody),
		Content:       schema,
	}
	if createSchemaFile {
		log.Debug("Create latest schema file",
			zap.String("schema_file", latestSchemaFile),
		)

		err := vcsPlugin.Get(repo2.VCS.Type, vcsPlugin.ProviderConfig{}).CreateFile(
			ctx,
			common.OauthContext{
				ClientID:     repo2.VCS.ApplicationID,
				ClientSecret: repo2.VCS.Secret,
				AccessToken:  repo2.AccessToken,
				RefreshToken: repo2.RefreshToken,
				Refresher:    server.refreshToken(ctx, repo2.ID),
			},
			repo2.VCS.InstanceURL,
			repo2.ExternalID,
			latestSchemaFile,
			schemaFileCommit,
		)

		if err != nil {
			return "", errors.Wrapf(err, "failed to create file after applying migration %s to %q", mi.Version, mi.Database)
		}
	} else {
		log.Debug("Update latest schema file",
			zap.String("schema_file", latestSchemaFile),
		)

		schemaFileCommit.LastCommitID = schemaFileMeta.LastCommitID
		err := vcsPlugin.Get(repo2.VCS.Type, vcsPlugin.ProviderConfig{}).OverwriteFile(
			ctx,
			common.OauthContext{
				ClientID:     repo2.VCS.ApplicationID,
				ClientSecret: repo2.VCS.Secret,
				AccessToken:  repo2.AccessToken,
				RefreshToken: repo2.RefreshToken,
				Refresher:    server.refreshToken(ctx, repo2.ID),
			},
			repo2.VCS.InstanceURL,
			repo2.ExternalID,
			latestSchemaFile,
			schemaFileCommit,
		)
		if err != nil {
			return "", errors.Wrapf(err, "failed to create file after applying migration %s to %q", mi.Version, mi.Database)
		}
	}

	// Retrieve the latest AccessToken and RefreshToken as the previous VCS call may have
	// updated the stored token pair. VCS will fetch and store the new token pair if the
	// existing token pair has expired.
	repo2, err = server.store.GetRepository(ctx, &api.RepositoryFind{ID: &repository.ID})
	if err != nil {
		return "", errors.Wrap(err, "failed to fetch repository after schema write-back")
	}
	if repo2 == nil {
		return "", errors.Errorf("repository not found after schema write-back: %v", repository.ID)
	}
	// VCS such as GitLab API doesn't return the commit on write, so we have to call ReadFileMeta again
	schemaFileMeta, err = vcsPlugin.Get(repo2.VCS.Type, vcsPlugin.ProviderConfig{}).ReadFileMeta(
		ctx,
		common.OauthContext{
			ClientID:     repo2.VCS.ApplicationID,
			ClientSecret: repo2.VCS.Secret,
			AccessToken:  repo2.AccessToken,
			RefreshToken: repo2.RefreshToken,
			Refresher:    server.refreshToken(ctx, repo2.ID),
		},
		repo2.VCS.InstanceURL,
		repo2.ExternalID,
		latestSchemaFile,
		branch,
	)

	if err != nil {
		return "", errors.Wrapf(err, "failed to fetch latest schema file %s after update", latestSchemaFile)
	}
	return schemaFileMeta.LastCommitID, nil
}
