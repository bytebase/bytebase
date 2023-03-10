package taskrun

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gosimple/slug"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/activity"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/state"
	enterpriseAPI "github.com/bytebase/bytebase/backend/enterprise/api"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/mysql"
	"github.com/bytebase/bytebase/backend/plugin/parser"
	"github.com/bytebase/bytebase/backend/plugin/parser/transform"
	vcsPlugin "github.com/bytebase/bytebase/backend/plugin/vcs"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
)

// Executor is the task executor.
type Executor interface {
	// RunOnce will be called periodically by the scheduler until terminated is true.
	//
	// NOTE
	//
	// 1. It's possible that err could be non-nil while terminated is false, which
	// usually indicates a transient error and will make scheduler retry later.
	// 2. If err is non-nil, then the detail field will be ignored since info is provided in the err.
	RunOnce(ctx context.Context, task *store.TaskMessage) (terminated bool, result *api.TaskRunResultPayload, err error)
}

// RunExecutorOnce wraps a TaskExecutor.RunOnce call with panic recovery.
func RunExecutorOnce(ctx context.Context, exec Executor, task *store.TaskMessage) (terminated bool, result *api.TaskRunResultPayload, err error) {
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

	return exec.RunOnce(ctx, task)
}

func preMigration(ctx context.Context, stores *store.Store, profile config.Profile, task *store.TaskMessage, migrationType db.MigrationType, statement, schemaVersion string, vcsPushEvent *vcsPlugin.PushEvent) (*db.MigrationInfo, error) {
	instance, err := stores.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: &task.InstanceID})
	if err != nil {
		return nil, err
	}
	environment, err := stores.GetEnvironmentV2(ctx, &store.FindEnvironmentMessage{ResourceID: &instance.EnvironmentID})
	if err != nil {
		return nil, err
	}
	database, err := stores.GetDatabaseV2(ctx, &store.FindDatabaseMessage{UID: task.DatabaseID})
	if err != nil {
		return nil, err
	}
	if database == nil {
		return nil, errors.Errorf("database not found")
	}

	mi := &db.MigrationInfo{
		InstanceID:     instance.UID,
		DatabaseID:     &database.UID,
		CreatorID:      api.SystemBotID,
		ReleaseVersion: profile.Version,
		Type:           migrationType,
		// TODO(d): support semantic versioning.
		Version:     schemaVersion,
		Description: task.Name,
		Environment: environment.ResourceID,
	}

	issue, err := stores.GetIssueV2(ctx, &store.FindIssueMessage{PipelineID: &task.PipelineID})
	if err != nil {
		log.Error("failed to find containing issue", zap.Error(err))
	}
	if issue != nil {
		// Concate issue title and task name as the migration description so that user can see
		// more context of the migration.
		mi.Description = fmt.Sprintf("%s - %s", issue.Title, task.Name)
		mi.IssueID = strconv.Itoa(issue.UID)
		mi.IssueIDInt = &issue.UID
	}

	if vcsPushEvent == nil {
		mi.Source = db.UI
		creator, err := stores.GetUserByID(ctx, task.CreatorID)
		if err != nil {
			// If somehow we unable to find the principal, we just emit the error since it's not
			// critical enough to fail the entire operation.
			log.Error("Failed to fetch creator for composing the migration info",
				zap.Int("task_id", task.ID),
				zap.Error(err),
			)
		} else {
			mi.Creator = creator.Name
			mi.CreatorID = creator.ID
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

	mi.Database = database.DatabaseName
	mi.Namespace = database.DatabaseName

	statement = strings.TrimSpace(statement)
	// Only baseline and SDL migration can have empty sql statement, which indicates empty database.
	if mi.Type != db.Baseline && mi.Type != db.MigrateSDL && statement == "" {
		return nil, errors.Errorf("empty statement")
	}
	// We will force migration for baseline, migrate and data type of migrations.
	// This usually happens when the previous attempt fails and the client retries the migration.
	// We also force migration for VCS migrations, which is usually a modified file to correct a former wrong migration commit.
	if mi.Type == db.Baseline || mi.Type == db.Migrate || mi.Type == db.MigrateSDL || mi.Type == db.Data {
		mi.Force = true
	}

	return mi, nil
}

func executeMigration(ctx context.Context, stores *store.Store, dbFactory *dbfactory.DBFactory, stateCfg *state.State, task *store.TaskMessage, statement string, mi *db.MigrationInfo) (migrationID string, schema string, err error) {
	statement = strings.TrimSpace(statement)
	instance, err := stores.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: &task.InstanceID})
	if err != nil {
		return "", "", err
	}
	database, err := stores.GetDatabaseV2(ctx, &store.FindDatabaseMessage{UID: task.DatabaseID})
	if err != nil {
		return "", "", err
	}

	driver, err := dbFactory.GetAdminDatabaseDriver(ctx, instance, database.DatabaseName)
	if err != nil {
		return "", "", errors.Wrapf(err, "failed to check migration setup for instance %q", instance.ResourceID)
	}
	defer driver.Close(ctx)

	statementRecord, _ := common.TruncateString(statement, common.MaxSheetSize)
	log.Debug("Start migration...",
		zap.String("instance", instance.ResourceID),
		zap.String("database", database.DatabaseName),
		zap.String("source", string(mi.Source)),
		zap.String("type", string(mi.Type)),
		zap.String("statement", statementRecord),
	)

	setup, err := driver.NeedsSetupMigration(ctx)
	if err != nil {
		return "", "", errors.Wrapf(err, "failed to check migration setup for instance %q", instance.ResourceID)
	}
	if setup {
		return "", "", common.Errorf(common.MigrationSchemaMissing, "missing migration schema for instance %q", instance.ResourceID)
	}

	if task.Type == api.TaskDatabaseDataUpdate && instance.Engine == db.MySQL {
		updatedTask, err := setThreadIDAndStartBinlogCoordinate(ctx, driver, task, stores)
		if err != nil {
			return "", "", errors.Wrap(err, "failed to update the task payload for MySQL rollback SQL")
		}
		task = updatedTask
	}

	// TODO(p0ny): migrate to instance change history
	if instance.Engine == db.Redis || instance.Engine == db.Oracle || instance.Engine == db.Spanner {
		migrationID, schema, err = utils.ExecuteMigration(ctx, stores, driver, mi, statement)
		if err != nil {
			return "", "", err
		}
	} else {
		migrationID, schema, err = driver.ExecuteMigration(ctx, mi, statement)
		if err != nil {
			return "", "", err
		}
	}

	if task.Type == api.TaskDatabaseDataUpdate && instance.Engine == db.MySQL {
		updatedTask, err := setMigrationIDAndEndBinlogCoordinate(ctx, driver, task, stores, migrationID)
		if err != nil {
			return "", "", errors.Wrap(err, "failed to update the task payload for MySQL rollback SQL")
		}

		payload := &api.TaskDatabaseDataUpdatePayload{}
		if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
			return "", "", errors.Wrap(err, "invalid database data update payload")
		}
		if payload.RollbackEnabled {
			// The runner will periodically scan the map to generate rollback SQL asynchronously.
			stateCfg.RollbackGenerate.Store(task.ID, updatedTask)
		}
	}

	return migrationID, schema, nil
}

func setThreadIDAndStartBinlogCoordinate(ctx context.Context, driver db.Driver, task *store.TaskMessage, store *store.Store) (*store.TaskMessage, error) {
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
	updatedTask, err := store.UpdateTaskV2(ctx, patch)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to patch task %d with the MySQL thread ID", task.ID)
	}
	return updatedTask, nil
}

func setMigrationIDAndEndBinlogCoordinate(ctx context.Context, driver db.Driver, task *store.TaskMessage, store *store.Store, migrationID string) (*store.TaskMessage, error) {
	payload := &api.TaskDatabaseDataUpdatePayload{}
	if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
		return nil, errors.Wrap(err, "invalid database data update payload")
	}

	payload.MigrationID = migrationID
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
	updatedTask, err := store.UpdateTaskV2(ctx, patch)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to patch task %d with the MySQL thread ID", task.ID)
	}
	return updatedTask, nil
}

func postMigration(ctx context.Context, stores *store.Store, activityManager *activity.Manager, license enterpriseAPI.LicenseService, task *store.TaskMessage, vcsPushEvent *vcsPlugin.PushEvent, mi *db.MigrationInfo, migrationID string, schema string) (bool, *api.TaskRunResultPayload, error) {
	instance, err := stores.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: &task.InstanceID})
	if err != nil {
		return true, nil, err
	}
	environment, err := stores.GetEnvironmentV2(ctx, &store.FindEnvironmentMessage{ResourceID: &instance.EnvironmentID})
	if err != nil {
		return true, nil, err
	}
	database, err := stores.GetDatabaseV2(ctx, &store.FindDatabaseMessage{UID: task.DatabaseID})
	if err != nil {
		return true, nil, err
	}
	project, err := stores.GetProjectV2(ctx, &store.FindProjectMessage{ResourceID: &database.ProjectID})
	if err != nil {
		return true, nil, err
	}

	issue, err := stores.GetIssueV2(ctx, &store.FindIssueMessage{PipelineID: &task.PipelineID})
	if err != nil {
		log.Error("failed to find containing issue", zap.Error(err))
	}
	if err != nil {
		// If somehow we cannot find the issue, emit the error since it's not fatal.
		log.Error("failed to find containing issue", zap.Error(err))
	}
	repo, err := stores.GetRepository(ctx, &api.RepositoryFind{
		ProjectID: &project.UID,
	})
	if err != nil {
		return true, nil, errors.Errorf("failed to find linked repository for database %q", database.DatabaseName)
	}

	if mi.Type == db.Migrate || mi.Type == db.MigrateSDL {
		if _, err := stores.UpdateDatabase(ctx, &store.UpdateDatabaseMessage{
			EnvironmentID: environment.ResourceID,
			InstanceID:    instance.ResourceID,
			DatabaseName:  database.DatabaseName,
			SchemaVersion: &mi.Version,
		}, api.SystemBotID); err != nil {
			return true, nil, errors.Errorf("failed to update database %q for instance %q", database.DatabaseName, instance.ResourceID)
		}
	}

	writebackBranch, err := isWriteBack(ctx, stores, license, project, repo, task, vcsPushEvent)
	if err != nil {
		return true, nil, err
	}

	log.Debug("Post migration...",
		zap.String("instance", instance.ResourceID),
		zap.String("database", database.DatabaseName),
		zap.String("writeback_branch", writebackBranch),
	)

	if writebackBranch != "" {
		// Transform the schema to standard style for SDL mode.
		if instance.Engine == db.MySQL {
			standardSchema, err := transform.SchemaTransform(parser.MySQL, schema)
			if err != nil {
				return true, nil, errors.Errorf("failed to transform to standard schema for database %q", database.DatabaseName)
			}
			schema = standardSchema
		}

		latestSchemaFile := filepath.Join(repo.BaseDirectory, repo.SchemaPathTemplate)
		latestSchemaFile = strings.ReplaceAll(latestSchemaFile, "{{ENV_NAME}}", mi.Environment)
		latestSchemaFile = strings.ReplaceAll(latestSchemaFile, "{{DB_NAME}}", mi.Database)

		vcs, err := stores.GetVCSByID(ctx, repo.VCSID)
		if err != nil {
			return true, nil, errors.Errorf("failed to sync schema file %s after applying migration %s to %q", latestSchemaFile, mi.Version, database.DatabaseName)
		}
		if vcs == nil {
			return true, nil, errors.Errorf("VCS ID not found: %d", repo.VCSID)
		}
		repo.VCS = vcs

		bytebaseURL := ""
		if issue != nil {
			setting, err := stores.GetWorkspaceGeneralSetting(ctx)
			if err != nil {
				return true, nil, errors.Wrapf(err, "failed to get workspace setting")
			}

			bytebaseURL = fmt.Sprintf("%s/issue/%s-%d?stage=%d", setting.ExternalUrl, slug.Make(issue.Title), issue.UID, task.StageID)
		}

		commitID, err := writeBackLatestSchema(ctx, stores, repo, vcsPushEvent, mi, writebackBranch, latestSchemaFile, schema, bytebaseURL)
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
					mi.Database,
				),
				Payload: string(payload),
			}

			if _, err := activityManager.CreateActivity(ctx, activityCreate, &activity.Metadata{}); err != nil {
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
	if err := stores.ArchiveAnomalyV2(ctx, &store.ArchiveAnomalyMessage{
		DatabaseUID: task.DatabaseID,
		Type:        api.AnomalyDatabaseSchemaDrift,
	}); err != nil && common.ErrorCode(err) != common.NotFound {
		log.Error("Failed to archive anomaly",
			zap.String("instance", instance.ResourceID),
			zap.String("database", database.DatabaseName),
			zap.String("type", string(api.AnomalyDatabaseSchemaDrift)),
			zap.Error(err))
	}

	detail := fmt.Sprintf("Applied migration version %s to database %q.", mi.Version, database.DatabaseName)
	if mi.Type == db.Baseline {
		detail = fmt.Sprintf("Established baseline version %s for database %q.", mi.Version, database.DatabaseName)
	}

	return true, &api.TaskRunResultPayload{
		Detail:      detail,
		MigrationID: migrationID,
		Version:     mi.Version,
	}, nil
}

func isWriteBack(ctx context.Context, stores *store.Store, license enterpriseAPI.LicenseService, project *store.ProjectMessage, repo *api.Repository, task *store.TaskMessage, vcsPushEvent *vcsPlugin.PushEvent) (string, error) {
	if !license.IsFeatureEnabled(api.FeatureVCSSchemaWriteBack) {
		return "", nil
	}
	if task.Type != api.TaskDatabaseSchemaBaseline && task.Type != api.TaskDatabaseSchemaUpdate && task.Type != api.TaskDatabaseSchemaUpdateGhostCutover {
		return "", nil
	}
	if repo == nil || repo.SchemaPathTemplate == "" {
		return "", nil
	}

	branch := ""
	// Prefer write back to the commit branch than the repo branch.
	if !strings.Contains(repo.BranchFilter, "*") {
		branch = repo.BranchFilter
	}
	if vcsPushEvent != nil && vcsPushEvent.Ref != "" {
		b, err := vcsPlugin.Branch(vcsPushEvent.Ref)
		if err != nil {
			return "", err
		}
		branch = b
	}
	if branch == "" {
		return "", nil
	}

	// For tenant mode project, we will only write back once and we happen to write back on lastTask done.
	if project.TenantMode == api.TenantModeTenant {
		stages, err := stores.ListStageV2(ctx, task.PipelineID)
		if err != nil {
			return "", err
		}
		if len(stages) == 0 {
			return "", nil
		}
		if stages[len(stages)-1].ID != task.StageID {
			return "", nil
		}
		tasks, err := stores.ListTasks(ctx, &api.TaskFind{PipelineID: &task.PipelineID, StageID: &task.StageID})
		if err != nil {
			return "", err
		}
		if len(tasks) == 0 {
			return "", nil
		}
		if tasks[len(tasks)-1].ID != task.ID {
			return "", nil
		}
	}

	return branch, nil
}

func runMigration(ctx context.Context, store *store.Store, dbFactory *dbfactory.DBFactory, activityManager *activity.Manager, license enterpriseAPI.LicenseService, stateCfg *state.State, profile config.Profile, task *store.TaskMessage, migrationType db.MigrationType, statement, schemaVersion string, vcsPushEvent *vcsPlugin.PushEvent) (terminated bool, result *api.TaskRunResultPayload, err error) {
	mi, err := preMigration(ctx, store, profile, task, migrationType, statement, schemaVersion, vcsPushEvent)
	if err != nil {
		return true, nil, err
	}
	migrationID, schema, err := executeMigration(ctx, store, dbFactory, stateCfg, task, statement, mi)
	if err != nil {
		return true, nil, err
	}
	return postMigration(ctx, store, activityManager, license, task, vcsPushEvent, mi, migrationID, schema)
}

// Writes back the latest schema to the repository after migration.
// Returns the commit id on success.
func writeBackLatestSchema(ctx context.Context, store *store.Store, repository *api.Repository, pushEvent *vcsPlugin.PushEvent, mi *db.MigrationInfo, writebackBranch, latestSchemaFile string, schema string, bytebaseURL string) (string, error) {
	schemaFileMeta, err := vcsPlugin.Get(repository.VCS.Type, vcsPlugin.ProviderConfig{}).ReadFileMeta(
		ctx,
		common.OauthContext{
			ClientID:     repository.VCS.ApplicationID,
			ClientSecret: repository.VCS.Secret,
			AccessToken:  repository.AccessToken,
			RefreshToken: repository.RefreshToken,
			Refresher:    utils.RefreshToken(ctx, store, repository.WebURL),
		},
		repository.VCS.InstanceURL,
		repository.ExternalID,
		latestSchemaFile,
		writebackBranch,
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
		if pushEvent != nil {
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
		Branch:        writebackBranch,
		CommitMessage: commitMessage,
		Content:       schema,
		AuthorName:    vcsPlugin.BytebaseAuthorName,
		AuthorEmail:   vcsPlugin.BytebaseAuthorEmail,
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
				Refresher:    utils.RefreshToken(ctx, store, repo2.WebURL),
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
		schemaFileCommit.SHA = schemaFileMeta.SHA
		err := vcsPlugin.Get(repo2.VCS.Type, vcsPlugin.ProviderConfig{}).OverwriteFile(
			ctx,
			common.OauthContext{
				ClientID:     repo2.VCS.ApplicationID,
				ClientSecret: repo2.VCS.Secret,
				AccessToken:  repo2.AccessToken,
				RefreshToken: repo2.RefreshToken,
				Refresher:    utils.RefreshToken(ctx, store, repo2.WebURL),
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
			Refresher:    utils.RefreshToken(ctx, store, repo2.WebURL),
		},
		repo2.VCS.InstanceURL,
		repo2.ExternalID,
		latestSchemaFile,
		writebackBranch,
	)

	if err != nil {
		return "", errors.Wrapf(err, "failed to fetch latest schema file %s after update", latestSchemaFile)
	}
	return schemaFileMeta.LastCommitID, nil
}
