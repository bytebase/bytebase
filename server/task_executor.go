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
	"github.com/bytebase/bytebase/plugin/db/mysql"
	"github.com/bytebase/bytebase/plugin/parser"
	"github.com/bytebase/bytebase/plugin/parser/transform"
	vcsPlugin "github.com/bytebase/bytebase/plugin/vcs"
	"github.com/bytebase/bytebase/server/component/config"
	"github.com/bytebase/bytebase/server/component/dbfactory"
	"github.com/bytebase/bytebase/store"
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

func preMigration(ctx context.Context, store *store.Store, profile config.Profile, task *api.Task, migrationType db.MigrationType, statement, schemaVersion string, vcsPushEvent *vcsPlugin.PushEvent) (*db.MigrationInfo, error) {
	if task.Database == nil {
		msg := "missing database when updating schema"
		if migrationType == db.Data {
			msg = "missing database when updating data"
		}
		return nil, errors.Errorf(msg)
	}
	databaseName := task.Database.Name

	mi := &db.MigrationInfo{
		ReleaseVersion: profile.Version,
		Type:           migrationType,
		// TODO(d): support semantic versioning.
		Version:     schemaVersion,
		Description: task.Name,
		Environment: task.Instance.Environment.Name,
	}
	if vcsPushEvent == nil {
		mi.Source = db.UI
		creator, err := store.GetPrincipalByID(ctx, task.CreatorID)
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
		mi.Creator = vcsPushEvent.AuthorName
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

	issue, err := findIssueByTask(ctx, store, task)
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
	// We will force migration for baseline, migrate and data type of migrations.
	// This usually happens when the previous attempt fails and the client retries the migration.
	// We also force migration for VCS migrations, which is usually a modified file to correct a former wrong migration commit.
	if mi.Type == db.Baseline || mi.Type == db.Migrate || mi.Type == db.Data {
		mi.Force = true
	}

	return mi, nil
}

func executeMigration(ctx context.Context, store *store.Store, dbFactory *dbfactory.DBFactory, rollbackRunner *RollbackRunner, task *api.Task, statement string, mi *db.MigrationInfo) (migrationID int64, schema string, err error) {
	statement = strings.TrimSpace(statement)
	databaseName := task.Database.Name

	driver, err := dbFactory.GetAdminDatabaseDriver(ctx, task.Instance, databaseName)
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

	if task.Type == api.TaskDatabaseDataUpdate && task.Instance.Engine == db.MySQL {
		updatedTask, err := setThreadIDAndStartBinlogCoordinate(ctx, driver, task, store)
		if err != nil {
			return 0, "", errors.Wrap(err, "failed to update the task payload for MySQL rollback SQL")
		}
		task = updatedTask
	}

	migrationID, schema, err = driver.ExecuteMigration(ctx, mi, statement)
	if err != nil {
		return 0, "", err
	}

	if task.Type == api.TaskDatabaseDataUpdate && task.Instance.Engine == db.MySQL {
		updatedTask, err := setMigrationIDAndEndBinlogCoordinate(ctx, driver, task, store, migrationID)
		if err != nil {
			return 0, "", errors.Wrap(err, "failed to update the task payload for MySQL rollback SQL")
		}
		// The runner will periodically scan the map to generate rollback SQL asynchronously.
		rollbackRunner.generateMap.Store(updatedTask.ID, updatedTask)
	}

	return migrationID, schema, nil
}

func setThreadIDAndStartBinlogCoordinate(ctx context.Context, driver db.Driver, task *api.Task, store *store.Store) (*api.Task, error) {
	mysqlDriver, ok := driver.(*mysql.Driver)
	if !ok {
		return nil, errors.Errorf("failed to cast driver to mysql.Driver")
	}
	payload := &api.TaskDatabaseDataUpdatePayload{}
	if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
		return nil, errors.Wrap(err, "invalid database data update payload")
	}
	connID, err := mysqlDriver.GetMigrationConnID(ctx)
	if err != nil {
		return nil, err
	}
	payload.ThreadID = connID

	db, err := driver.GetDBConnection(ctx, "")
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the DB connection")
	}
	binlogInfo, err := mysql.GetBinlogInfo(ctx, db)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the binlog info before executing the migration transaction")
	}
	if (binlogInfo == api.BinlogInfo{}) {
		log.Warn("binlog is not enabled", zap.Int("task", task.ID))
		return task, nil
	}
	payload.BinlogFileStart = binlogInfo.FileName
	payload.BinlogPosStart = binlogInfo.Position

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal task payload")
	}
	payloadString := string(payloadBytes)
	patch := &api.TaskPatch{
		ID:        task.ID,
		UpdaterID: api.SystemBotID,
		Payload:   &payloadString,
	}
	updatedTask, err := store.PatchTask(ctx, patch)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to patch task %d with the MySQL thread ID", task.ID)
	}
	return updatedTask, nil
}

func setMigrationIDAndEndBinlogCoordinate(ctx context.Context, driver db.Driver, task *api.Task, store *store.Store, migrationID int64) (*api.Task, error) {
	payload := &api.TaskDatabaseDataUpdatePayload{}
	if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
		return nil, errors.Wrap(err, "invalid database data update payload")
	}

	payload.MigrationID = int(migrationID)
	db, err := driver.GetDBConnection(ctx, "")
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the DB connection")
	}
	binlogInfo, err := mysql.GetBinlogInfo(ctx, db)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the binlog info before executing the migration transaction")
	}
	if (binlogInfo == api.BinlogInfo{}) {
		log.Warn("binlog is not enabled", zap.Int("task", task.ID))
		return task, nil
	}
	payload.BinlogFileEnd = binlogInfo.FileName
	payload.BinlogPosEnd = binlogInfo.Position

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal task payload")
	}
	payloadString := string(payloadBytes)
	patch := &api.TaskPatch{
		ID:        task.ID,
		UpdaterID: api.SystemBotID,
		Payload:   &payloadString,
	}
	updatedTask, err := store.PatchTask(ctx, patch)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to patch task %d with the MySQL thread ID", task.ID)
	}
	return updatedTask, nil
}

func postMigration(ctx context.Context, store *store.Store, activityManager *ActivityManager, profile config.Profile, task *api.Task, vcsPushEvent *vcsPlugin.PushEvent, mi *db.MigrationInfo, migrationID int64, schema string) (bool, *api.TaskRunResultPayload, error) {
	databaseName := task.Database.Name
	issue, err := findIssueByTask(ctx, store, task)
	if err != nil {
		// If somehow we cannot find the issue, emit the error since it's not fatal.
		log.Error("failed to find containing issue", zap.Error(err))
	}
	project, err := store.GetProjectByID(ctx, task.Database.ProjectID)
	if err != nil {
		return true, nil, err
	}
	repo, err := store.GetRepository(ctx, &api.RepositoryFind{
		ProjectID: &task.Database.ProjectID,
	})
	if err != nil {
		return true, nil, errors.Errorf("failed to find linked repository for database %q", task.Database.Name)
	}

	// On the presence of schema path template and non-wildcard branch filter, We write back the latest schema after migration for VCS-based projects for
	// 1) baseline migration for SDL,
	// 2) all DDL/Ghost migrations.
	writeBack := false
	if repo != nil && repo.SchemaPathTemplate != "" && !strings.Contains(repo.BranchFilter, "*") {
		if repo.Project.SchemaChangeType == api.ProjectSchemaChangeTypeSDL {
			if task.Type == api.TaskDatabaseSchemaBaseline {
				writeBack = true
				// Transform the schema to standard style for SDL mode.
				if task.Database.Instance.Engine == db.MySQL {
					standardSchema, err := transform.SchemaTransform(parser.MySQL, schema)
					if err != nil {
						return true, nil, errors.Errorf("failed to transform to standard schema for database %q", task.Database.Name)
					}
					schema = standardSchema
				}
			}
		} else {
			writeBack = (vcsPushEvent != nil) && (task.Type == api.TaskDatabaseSchemaUpdate || task.Type == api.TaskDatabaseSchemaUpdateGhostCutover)
		}
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

		vcs, err := store.GetVCSByID(ctx, repo.VCSID)
		if err != nil {
			return true, nil, errors.Errorf("failed to sync schema file %s after applying migration %s to %q", latestSchemaFile, mi.Version, databaseName)
		}
		if vcs == nil {
			return true, nil, errors.Errorf("VCS ID not found: %d", repo.VCSID)
		}
		repo.VCS = vcs

		bytebaseURL := ""
		if issue != nil {
			bytebaseURL = fmt.Sprintf("%s/issue/%s?stage=%d", profile.ExternalURL, api.IssueSlug(issue), task.StageID)
		}

		commitID, err := writeBackLatestSchema(ctx, store, repo, vcsPushEvent, mi, repo.BranchFilter, latestSchemaFile, schema, bytebaseURL)
		if err != nil {
			return true, nil, err
		}

		// Create file commit activity
		{
			payload, err := json.Marshal(api.ActivityPipelineTaskFileCommitPayload{
				TaskID:             task.ID,
				VCSInstanceURL:     repo.VCS.InstanceURL,
				RepositoryFullPath: repo.FullPath,
				Branch:             repo.BranchFilter,
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

			_, err = activityManager.CreateActivity(ctx, activityCreate, &ActivityMeta{})
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

	// Remove schema drift anomalies.
	if err := store.ArchiveAnomaly(ctx, &api.AnomalyArchive{
		DatabaseID: task.DatabaseID,
		Type:       api.AnomalyDatabaseSchemaDrift,
	}); err != nil && common.ErrorCode(err) != common.NotFound {
		log.Error("Failed to archive anomaly",
			zap.String("instance", task.Instance.Name),
			zap.String("database", task.Database.Name),
			zap.String("type", string(api.AnomalyDatabaseSchemaDrift)),
			zap.Error(err))
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

func runMigration(ctx context.Context, store *store.Store, dbFactory *dbfactory.DBFactory, rollbackRunner *RollbackRunner, activityManager *ActivityManager, profile config.Profile, task *api.Task, migrationType db.MigrationType, statement, schemaVersion string, vcsPushEvent *vcsPlugin.PushEvent) (terminated bool, result *api.TaskRunResultPayload, err error) {
	mi, err := preMigration(ctx, store, profile, task, migrationType, statement, schemaVersion, vcsPushEvent)
	if err != nil {
		return true, nil, err
	}
	migrationID, schema, err := executeMigration(ctx, store, dbFactory, rollbackRunner, task, statement, mi)
	if err != nil {
		return true, nil, err
	}
	return postMigration(ctx, store, activityManager, profile, task, vcsPushEvent, mi, migrationID, schema)
}

func findIssueByTask(ctx context.Context, store *store.Store, task *api.Task) (*api.Issue, error) {
	issue, err := store.GetIssueByPipelineID(ctx, task.PipelineID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to fetch containing issue for composing the migration info, task_id: %v", task.ID)
	}
	if issue == nil {
		return nil, errors.Wrapf(err, "failed to fetch containing issue for composing the migration info, issue not found, pipeline ID: %v, task_id: %v", task.PipelineID, task.ID)
	}
	return issue, nil
}

// Writes back the latest schema to the repository after migration
// Returns the commit id on success.
func writeBackLatestSchema(ctx context.Context, store *store.Store, repository *api.Repository, pushEvent *vcsPlugin.PushEvent, mi *db.MigrationInfo, branch string, latestSchemaFile string, schema string, bytebaseURL string) (string, error) {
	schemaFileMeta, err := vcsPlugin.Get(repository.VCS.Type, vcsPlugin.ProviderConfig{}).ReadFileMeta(
		ctx,
		common.OauthContext{
			ClientID:     repository.VCS.ApplicationID,
			ClientSecret: repository.VCS.Secret,
			AccessToken:  repository.AccessToken,
			RefreshToken: repository.RefreshToken,
			Refresher:    refreshToken(ctx, store, repository.WebURL),
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

	var commitMessage string
	if mi.Type == db.Baseline {
		commitMessage = fmt.Sprintf("[Bytebase] establish baseline for %q", mi.Database)
	} else {
		commitTitle := fmt.Sprintf("[Bytebase] %s latest schema for %q after migration %s", verb, mi.Database, mi.Version)
		commitBody := "THIS COMMIT IS AUTO-GENERATED BY BYTEBASE"
		if bytebaseURL != "" {
			commitBody += "\n\n" + bytebaseURL
		}
		commitBody += "\n\n--------Original migration change--------\n\n"
		if len(pushEvent.CommitList) == 0 {
			// For legacy data in task payload stored in the database.
			// TODO(dragonly): Remove the field FileCommit.
			commitBody += fmt.Sprintf("%s\n\n%s",
				pushEvent.FileCommit.URL,
				pushEvent.FileCommit.Message,
			)
		} else {
			commitBody += fmt.Sprintf("%s\n\n%s",
				pushEvent.CommitList[0].URL,
				pushEvent.CommitList[0].Message,
			)
		}
		commitMessage = fmt.Sprintf("%s\n\n%s", commitTitle, commitBody)
	}

	// Retrieve the latest AccessToken and RefreshToken as the previous VCS call may have
	// updated the stored token pair. VCS will fetch and store the new token pair if the
	// existing token pair has expired.
	repo2, err := store.GetRepository(ctx, &api.RepositoryFind{ID: &repository.ID})
	if err != nil {
		return "", errors.Wrap(err, "failed to fetch repository for schema write-back")
	}
	if repo2 == nil {
		return "", errors.Errorf("repository not found for schema write-back: %v", repository.ID)
	}

	schemaFileCommit := vcsPlugin.FileCommitCreate{
		Branch:        branch,
		CommitMessage: commitMessage,
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
				Refresher:    refreshToken(ctx, store, repo2.WebURL),
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
				Refresher:    refreshToken(ctx, store, repo2.WebURL),
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
	repo2, err = store.GetRepository(ctx, &api.RepositoryFind{ID: &repository.ID})
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
			Refresher:    refreshToken(ctx, store, repo2.WebURL),
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
