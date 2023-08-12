package taskrun

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/gosimple/slug"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"

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
	parser "github.com/bytebase/bytebase/backend/plugin/parser/sql"

	"github.com/bytebase/bytebase/backend/plugin/parser/sql/transform"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"

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
	// driverCtx is used by the database driver so that we can cancel the query
	// while have the ability to cleanup migration history etc.
	RunOnce(ctx context.Context, driverCtx context.Context, task *store.TaskMessage) (terminated bool, result *api.TaskRunResultPayload, err error)
}

// RunExecutorOnce wraps a TaskExecutor.RunOnce call with panic recovery.
func RunExecutorOnce(ctx context.Context, driverCtx context.Context, exec Executor, task *store.TaskMessage) (terminated bool, result *api.TaskRunResultPayload, err error) {
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

	return exec.RunOnce(ctx, driverCtx, task)
}

func getMigrationInfo(ctx context.Context, stores *store.Store, profile config.Profile, task *store.TaskMessage, migrationType db.MigrationType, statement, schemaVersion string, vcsPushEvent *vcsPlugin.PushEvent) (*db.MigrationInfo, error) {
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
		InstanceID:     &instance.UID,
		DatabaseID:     &database.UID,
		CreatorID:      task.CreatorID,
		ReleaseVersion: profile.Version,
		Type:           migrationType,
		// TODO(d): support semantic versioning.
		Version:     schemaVersion,
		Description: task.Name,
		Environment: environment.ResourceID,
		Database:    database.DatabaseName,
		Namespace:   database.DatabaseName,
		Payload:     &storepb.InstanceChangeHistoryPayload{},
	}

	taskCheckType := api.TaskCheckDatabaseStatementTypeReport
	typeReportTaskCheckRunFind := &store.TaskCheckRunFind{
		TaskID:     &task.ID,
		StageID:    &task.StageID,
		PipelineID: &task.PipelineID,
		Type:       &taskCheckType,
		StatusList: &[]api.TaskCheckRunStatus{api.TaskCheckRunDone},
	}
	taskCheckRun, err := stores.ListTaskCheckRuns(ctx, typeReportTaskCheckRunFind)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list task check runs")
	}
	sort.Slice(taskCheckRun, func(i, j int) bool {
		return taskCheckRun[i].ID > taskCheckRun[j].ID
	})
	if len(taskCheckRun) > 0 {
		checkResult := &api.TaskCheckRunResultPayload{}
		if err := json.Unmarshal([]byte(taskCheckRun[0].Result), checkResult); err != nil {
			return nil, err
		}
		mi.Payload.ChangedResources, err = mergeChangedResources(checkResult.ResultList)
		if err != nil {
			return nil, err
		}
	}

	issue, err := stores.GetIssueV2(ctx, &store.FindIssueMessage{PipelineID: &task.PipelineID})
	if err != nil {
		log.Error("failed to find containing issue", zap.Error(err))
	}
	if issue != nil {
		// Concat issue title and task name as the migration description so that user can see
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
		mi.Payload.PushEvent = utils.ConvertVcsPushEvent(vcsPushEvent)
	}

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

func executeMigration(ctx context.Context, driverCtx context.Context, stores *store.Store, dbFactory *dbfactory.DBFactory, stateCfg *state.State, task *store.TaskMessage, statement string, sheetID *int, mi *db.MigrationInfo) (string, string, error) {
	instance, err := stores.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: &task.InstanceID})
	if err != nil {
		return "", "", err
	}
	database, err := stores.GetDatabaseV2(ctx, &store.FindDatabaseMessage{UID: task.DatabaseID})
	if err != nil {
		return "", "", err
	}

	driver, err := dbFactory.GetAdminDatabaseDriver(ctx, instance, database)
	if err != nil {
		return "", "", errors.Wrapf(err, "failed to get driver connection for instance %q", instance.ResourceID)
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

	var migrationID string
	opts := db.ExecuteOptions{}
	if task.Type == api.TaskDatabaseDataUpdate && (instance.Engine == db.MySQL || instance.Engine == db.MariaDB) {
		opts.BeginFunc = func(ctx context.Context, conn *sql.Conn) error {
			updatedTask, err := setThreadIDAndStartBinlogCoordinate(ctx, conn, task, stores)
			if err != nil {
				return errors.Wrap(err, "failed to update the task payload for MySQL rollback SQL")
			}
			task = updatedTask
			return nil
		}
	}
	if task.Type == api.TaskDatabaseDataUpdate && instance.Engine == db.Oracle {
		// getSetOracleTransactionIdFunc will update the task payload to set the Oracle transaction id, we need to re-retrieve the task to store to the RollbackGenerate.
		opts.EndTransactionFunc = getSetOracleTransactionIDFunc(ctx, task, stores)
	}

	migrationID, schema, err := utils.ExecuteMigrationDefault(ctx, driverCtx, stores, driver, mi, statement, sheetID, opts)
	if err != nil {
		return "", "", err
	}

	// If the migration is a data migration, enable the rollback SQL generation and the type of the driver is Oracle, we need to get the rollback SQL before the transaction is committed.
	if task.Type == api.TaskDatabaseDataUpdate && instance.Engine == db.Oracle {
		updatedTask, err := stores.GetTaskV2ByID(ctx, task.ID)
		if err != nil {
			return "", "", errors.Wrapf(err, "cannot get task by id %d", task.ID)
		}
		payload := &api.TaskDatabaseDataUpdatePayload{}
		if err := json.Unmarshal([]byte(updatedTask.Payload), payload); err != nil {
			return "", "", errors.Wrap(err, "invalid database data update payload")
		}
		if payload.RollbackEnabled {
			// The runner will periodically scan the map to generate rollback SQL asynchronously.
			stateCfg.RollbackGenerate.Store(task.ID, updatedTask)
		}
	}

	if task.Type == api.TaskDatabaseDataUpdate && (instance.Engine == db.MySQL || instance.Engine == db.MariaDB) {
		conn, err := driver.GetDB().Conn(ctx)
		if err != nil {
			return "", "", errors.Wrap(err, "failed to create connection")
		}
		updatedTask, err := setMigrationIDAndEndBinlogCoordinate(ctx, conn, task, stores, migrationID)
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

func getSetOracleTransactionIDFunc(ctx context.Context, task *store.TaskMessage, store *store.Store) func(tx *sql.Tx) error {
	return func(tx *sql.Tx) error {
		payload := &api.TaskDatabaseDataUpdatePayload{}
		if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
			log.Error("failed to unmarshal task payload", zap.Int("TaskId", task.ID), zap.Error(err))
			return nil
		}
		// Get oracle current transaction id;
		transactionID, err := tx.QueryContext(ctx, "SELECT RAWTOHEX(tx.xid) FROM v$transaction tx JOIN v$session s ON tx.ses_addr = s.saddr")
		if err != nil {
			log.Error("failed to transaction id in task", zap.Int("TaskId", task.ID), zap.Error(err))
			return nil
		}
		defer transactionID.Close()
		var txID string
		for transactionID.Next() {
			err := transactionID.Scan(&txID)
			if err != nil {
				log.Error("failed to the Oracle transaction id in task", zap.Int("TaskId", task.ID), zap.Error(err))
				return nil
			}
		}
		if err := transactionID.Err(); err != nil {
			return err
		}
		payload.TransactionID = txID
		updatedPayload, err := json.Marshal(payload)
		if err != nil {
			log.Error("failed to unmarshal task payload", zap.Int("TaskId", task.ID), zap.Error(err), zap.Any("payload", updatedPayload))
			return nil
		}
		updatedPayloadString := string(updatedPayload)
		patch := &api.TaskPatch{
			ID:        task.ID,
			UpdaterID: api.SystemBotID,
			Payload:   &updatedPayloadString,
		}
		if _, err = store.UpdateTaskV2(ctx, patch); err != nil {
			log.Error("failed to update task with new payload", zap.Any("TaskPatch", patch), zap.Error(err))
			return nil
		}
		return nil
	}
}

func setThreadIDAndStartBinlogCoordinate(ctx context.Context, conn *sql.Conn, task *store.TaskMessage, store *store.Store) (*store.TaskMessage, error) {
	payload := &api.TaskDatabaseDataUpdatePayload{}
	if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
		return nil, errors.Wrap(err, "invalid database data update payload")
	}

	var connID string
	if err := conn.QueryRowContext(ctx, "SELECT CONNECTION_ID();").Scan(&connID); err != nil {
		return nil, errors.Wrap(err, "failed to get the connection ID")
	}
	payload.ThreadID = connID

	binlogInfo, err := mysql.GetBinlogInfo(ctx, conn)
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

func setMigrationIDAndEndBinlogCoordinate(ctx context.Context, conn *sql.Conn, task *store.TaskMessage, store *store.Store, migrationID string) (*store.TaskMessage, error) {
	payload := &api.TaskDatabaseDataUpdatePayload{}
	if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
		return nil, errors.Wrap(err, "invalid database data update payload")
	}

	payload.MigrationID = migrationID
	binlogInfo, err := mysql.GetBinlogInfo(ctx, conn)
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
	repo, err := stores.GetRepositoryV2(ctx, &store.FindRepositoryMessage{
		ProjectResourceID: &project.ResourceID,
	})
	if err != nil {
		return true, nil, errors.Errorf("failed to find linked repository for database %q", database.DatabaseName)
	}

	if mi.Type == db.Migrate || mi.Type == db.MigrateSDL {
		if _, err := stores.UpdateDatabase(ctx, &store.UpdateDatabaseMessage{
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
		if instance.Engine == db.MySQL || instance.Engine == db.MariaDB || instance.Engine == db.OceanBase {
			standardSchema, err := transform.SchemaTransform(parser.MySQL, schema)
			if err != nil {
				return true, nil, errors.Wrapf(err, "failed to transform to standard schema for database %q", database.DatabaseName)
			}
			schema = standardSchema
		}

		latestSchemaFile := filepath.Join(repo.BaseDirectory, repo.SchemaPathTemplate)
		latestSchemaFile = strings.ReplaceAll(latestSchemaFile, "{{ENV_ID}}", mi.Environment)
		latestSchemaFile = strings.ReplaceAll(latestSchemaFile, "{{DB_NAME}}", mi.Database)

		vcs, err := stores.GetExternalVersionControlV2(ctx, repo.VCSUID)
		if err != nil {
			return true, nil, errors.Errorf("failed to sync schema file %s after applying migration %s to %q", latestSchemaFile, mi.Version, database.DatabaseName)
		}
		if vcs == nil {
			return true, nil, errors.Errorf("VCS ID not found: %d", repo.VCSUID)
		}

		bytebaseURL := ""
		if issue != nil {
			setting, err := stores.GetWorkspaceGeneralSetting(ctx)
			if err != nil {
				return true, nil, errors.Wrapf(err, "failed to get workspace setting")
			}

			bytebaseURL = fmt.Sprintf("%s/issue/%s-%d?stage=%d", setting.ExternalUrl, slug.Make(issue.Title), issue.UID, task.StageID)
		}

		commitID, err := writeBackLatestSchema(ctx, stores, repo, vcs, vcsPushEvent, mi, writebackBranch, latestSchemaFile, schema, bytebaseURL)
		if err != nil {
			return true, nil, err
		}

		// Create file commit activity
		{
			payload, err := json.Marshal(api.ActivityPipelineTaskFileCommitPayload{
				TaskID:             task.ID,
				VCSInstanceURL:     vcs.InstanceURL,
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

			activityCreate := &store.ActivityMessage{
				CreatorUID:   task.CreatorID,
				ContainerUID: task.PipelineID,
				Type:         api.ActivityPipelineTaskFileCommit,
				Level:        api.ActivityInfo,
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
		Detail:        detail,
		MigrationID:   migrationID,
		ChangeHistory: fmt.Sprintf("instances/%s/databases/%s/migrations/%s", instance.ResourceID, database.DatabaseName, migrationID),
		Version:       mi.Version,
	}, nil
}

func isWriteBack(ctx context.Context, stores *store.Store, license enterpriseAPI.LicenseService, project *store.ProjectMessage, repo *store.RepositoryMessage, task *store.TaskMessage, vcsPushEvent *vcsPlugin.PushEvent) (string, error) {
	if task.Type != api.TaskDatabaseSchemaBaseline && task.Type != api.TaskDatabaseSchemaUpdate && task.Type != api.TaskDatabaseSchemaUpdateGhostCutover {
		return "", nil
	}
	if repo == nil || repo.SchemaPathTemplate == "" {
		return "", nil
	}

	instance, err := stores.GetInstanceV2(ctx, &store.FindInstanceMessage{
		UID:         &task.InstanceID,
		ShowDeleted: false,
	})
	if err != nil {
		return "", err
	}
	if instance == nil {
		return "", errors.Errorf("cannot found instance %d", task.InstanceID)
	}
	if instance.Engine == db.RisingWave {
		return "", nil
	}

	if err := license.IsFeatureEnabledForInstance(api.FeatureVCSSchemaWriteBack, instance); err != nil {
		log.Debug(err.Error(), zap.String("instance", instance.ResourceID))
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

func runMigration(ctx context.Context, driverCtx context.Context, store *store.Store, dbFactory *dbfactory.DBFactory, activityManager *activity.Manager, license enterpriseAPI.LicenseService, stateCfg *state.State, profile config.Profile, task *store.TaskMessage, migrationType db.MigrationType, statement, schemaVersion string, sheetID *int, vcsPushEvent *vcsPlugin.PushEvent) (terminated bool, result *api.TaskRunResultPayload, err error) {
	mi, err := getMigrationInfo(ctx, store, profile, task, migrationType, statement, schemaVersion, vcsPushEvent)
	if err != nil {
		return true, nil, err
	}

	migrationID, schema, err := executeMigration(ctx, driverCtx, store, dbFactory, stateCfg, task, statement, sheetID, mi)
	if err != nil {
		return true, nil, err
	}
	return postMigration(ctx, store, activityManager, license, task, vcsPushEvent, mi, migrationID, schema)
}

// Writes back the latest schema to the repository after migration.
// Returns the commit id on success.
func writeBackLatestSchema(
	ctx context.Context,
	storage *store.Store,
	repository *store.RepositoryMessage,
	vcs *store.ExternalVersionControlMessage,
	pushEvent *vcsPlugin.PushEvent,
	mi *db.MigrationInfo,
	writebackBranch, latestSchemaFile string, schema string, bytebaseURL string,
) (string, error) {
	schemaFileMeta, err := vcsPlugin.Get(vcs.Type, vcsPlugin.ProviderConfig{}).ReadFileMeta(
		ctx,
		common.OauthContext{
			ClientID:     vcs.ApplicationID,
			ClientSecret: vcs.Secret,
			AccessToken:  repository.AccessToken,
			RefreshToken: repository.RefreshToken,
			Refresher:    utils.RefreshToken(ctx, storage, repository.WebURL),
		},
		vcs.InstanceURL,
		repository.ExternalID,
		latestSchemaFile,
		vcsPlugin.RefInfo{
			RefType: vcsPlugin.RefTypeBranch,
			RefName: writebackBranch,
		},
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
	repo2, vcs2, err := getRepositoryAndVCS(ctx, storage, repository.UID, repository.VCSUID)
	if err != nil {
		return "", err
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

		err := vcsPlugin.Get(vcs2.Type, vcsPlugin.ProviderConfig{}).CreateFile(
			ctx,
			common.OauthContext{
				ClientID:     vcs2.ApplicationID,
				ClientSecret: vcs2.Secret,
				AccessToken:  repo2.AccessToken,
				RefreshToken: repo2.RefreshToken,
				Refresher:    utils.RefreshToken(ctx, storage, repo2.WebURL),
			},
			vcs2.InstanceURL,
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
		err := vcsPlugin.Get(vcs2.Type, vcsPlugin.ProviderConfig{}).OverwriteFile(
			ctx,
			common.OauthContext{
				ClientID:     vcs2.ApplicationID,
				ClientSecret: vcs2.Secret,
				AccessToken:  repo2.AccessToken,
				RefreshToken: repo2.RefreshToken,
				Refresher:    utils.RefreshToken(ctx, storage, repo2.WebURL),
			},
			vcs2.InstanceURL,
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
	repo2, vcs2, err = getRepositoryAndVCS(ctx, storage, repository.UID, repository.VCSUID)
	if err != nil {
		return "", err
	}

	// VCS such as GitLab API doesn't return the commit on write, so we have to call ReadFileMeta again
	schemaFileMeta, err = vcsPlugin.Get(vcs2.Type, vcsPlugin.ProviderConfig{}).ReadFileMeta(
		ctx,
		common.OauthContext{
			ClientID:     vcs2.ApplicationID,
			ClientSecret: vcs2.Secret,
			AccessToken:  repo2.AccessToken,
			RefreshToken: repo2.RefreshToken,
			Refresher:    utils.RefreshToken(ctx, storage, repo2.WebURL),
		},
		vcs2.InstanceURL,
		repo2.ExternalID,
		latestSchemaFile,
		vcsPlugin.RefInfo{
			RefType: vcsPlugin.RefTypeBranch,
			RefName: writebackBranch,
		},
	)

	if err != nil {
		return "", errors.Wrapf(err, "failed to fetch latest schema file %s after update", latestSchemaFile)
	}
	return schemaFileMeta.LastCommitID, nil
}

func getRepositoryAndVCS(ctx context.Context, storage *store.Store, repoUID, vcsUID int) (*store.RepositoryMessage, *store.ExternalVersionControlMessage, error) {
	repo, err := storage.GetRepositoryV2(ctx, &store.FindRepositoryMessage{
		UID: &repoUID,
	})
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to fetch repository after schema write-back")
	}
	if repo == nil {
		return nil, nil, errors.Errorf("repository not found after schema write-back: %v", repoUID)
	}
	vcs, err := storage.GetExternalVersionControlV2(ctx, vcsUID)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to fetch vcs for schema write-back")
	}
	if vcs == nil {
		return nil, nil, errors.Errorf("vcs not found for schema write-back: %v", vcsUID)
	}
	return repo, vcs, nil
}

type resourceDatabase struct {
	name    string
	schemas schemaMap
}

type databaseMap map[string]*resourceDatabase

type resourceSchema struct {
	name   string
	tables tableMap
}

type schemaMap map[string]*resourceSchema

type resourceTable struct {
	name string
}

type tableMap map[string]*resourceTable

func mergeChangedResources(list []api.TaskCheckResult) (*storepb.ChangedResources, error) {
	databaseMap := make(databaseMap)
	for _, item := range list {
		if item.ChangedResources == "" {
			continue
		}
		meta := storepb.ChangedResources{}
		if err := protojson.Unmarshal([]byte(item.ChangedResources), &meta); err != nil {
			return nil, err
		}
		for _, database := range meta.Databases {
			dbMeta, ok := databaseMap[database.Name]
			if !ok {
				dbMeta = &resourceDatabase{
					name:    database.Name,
					schemas: make(schemaMap),
				}
				databaseMap[database.Name] = dbMeta
			}
			for _, schema := range database.Schemas {
				schemaMeta, ok := dbMeta.schemas[schema.Name]
				if !ok {
					schemaMeta = &resourceSchema{
						name:   schema.Name,
						tables: make(tableMap),
					}
					dbMeta.schemas[schema.Name] = schemaMeta
				}
				for _, table := range schema.Tables {
					schemaMeta.tables[table.Name] = &resourceTable{
						name: table.Name,
					}
				}
			}
		}
	}

	result := &storepb.ChangedResources{}
	for _, dbMeta := range databaseMap {
		db := &storepb.ChangedResourceDatabase{
			Name: dbMeta.name,
		}
		for _, schemaMeta := range dbMeta.schemas {
			schema := &storepb.ChangedResourceSchema{
				Name: schemaMeta.name,
			}
			for _, tableMeta := range schemaMeta.tables {
				table := &storepb.ChangedResourceTable{
					Name: tableMeta.name,
				}
				schema.Tables = append(schema.Tables, table)
			}
			sort.Slice(schema.Tables, func(i, j int) bool {
				return schema.Tables[i].Name < schema.Tables[j].Name
			})
			db.Schemas = append(db.Schemas, schema)
		}
		sort.Slice(db.Schemas, func(i, j int) bool {
			return db.Schemas[i].Name < db.Schemas[j].Name
		})
		result.Databases = append(result.Databases, db)
	}
	sort.Slice(result.Databases, func(i, j int) bool {
		return result.Databases[i].Name < result.Databases[j].Name
	})
	return result, nil
}
