package taskrun

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gosimple/slug"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/activity"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/state"
	enterprise "github.com/bytebase/bytebase/backend/enterprise/api"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/mysql"
	"github.com/bytebase/bytebase/backend/plugin/parser/sql/transform"
	vcsplugin "github.com/bytebase/bytebase/backend/plugin/vcs"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/store/model"
	"github.com/bytebase/bytebase/backend/utils"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
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
	RunOnce(ctx context.Context, driverCtx context.Context, task *store.TaskMessage, taskRunUID int) (terminated bool, result *api.TaskRunResultPayload, err error)
}

// RunExecutorOnce wraps a TaskExecutor.RunOnce call with panic recovery.
func RunExecutorOnce(ctx context.Context, driverCtx context.Context, exec Executor, task *store.TaskMessage, taskRunUID int) (terminated bool, result *api.TaskRunResultPayload, err error) {
	defer func() {
		if r := recover(); r != nil {
			panicErr, ok := r.(error)
			if !ok {
				panicErr = errors.Errorf("%v", r)
			}
			slog.Error("TaskExecutor PANIC RECOVER", log.BBError(panicErr), log.BBStack("panic-stack"))
			terminated = true
			result = nil
			err = errors.Errorf("encounter internal error when executing task")
		}
	}()

	return exec.RunOnce(ctx, driverCtx, task, taskRunUID)
}

func getMigrationInfo(ctx context.Context, stores *store.Store, profile config.Profile, task *store.TaskMessage, migrationType db.MigrationType, statement string, schemaVersion model.Version) (*db.MigrationInfo, error) {
	instance, err := stores.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: &task.InstanceID})
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
	environment, err := stores.GetEnvironmentV2(ctx, &store.FindEnvironmentMessage{ResourceID: &database.EffectiveEnvironmentID})
	if err != nil {
		return nil, err
	}

	mi := &db.MigrationInfo{
		InstanceID:     &instance.UID,
		DatabaseID:     &database.UID,
		CreatorID:      task.CreatorID,
		ReleaseVersion: profile.Version,
		Type:           migrationType,
		Version:        schemaVersion,
		Description:    task.Name,
		Environment:    environment.ResourceID,
		Database:       database.DatabaseName,
		Namespace:      database.DatabaseName,
		Payload:        &storepb.InstanceChangeHistoryPayload{},
	}

	plans, err := stores.ListPlans(ctx, &store.FindPlanMessage{PipelineID: &task.PipelineID})
	if err != nil {
		return nil, err
	}
	if len(plans) == 1 {
		planTypes := []store.PlanCheckRunType{store.PlanCheckDatabaseStatementSummaryReport}
		status := []store.PlanCheckRunStatus{store.PlanCheckRunStatusDone}
		runs, err := stores.ListPlanCheckRuns(ctx, &store.FindPlanCheckRunMessage{
			PlanUID: &plans[0].UID,
			Type:    &planTypes,
			Status:  &status,
		})
		if err != nil {
			return nil, errors.Wrap(err, "failed to list plan check runs")
		}
		sort.Slice(runs, func(i, j int) bool {
			return runs[i].UID > runs[j].UID
		})
		foundChangedResources := false
		for _, run := range runs {
			if foundChangedResources {
				break
			}
			if run.Config.InstanceUid != int32(task.InstanceID) {
				continue
			}
			if run.Config.DatabaseName != database.DatabaseName {
				continue
			}
			if run.Result == nil {
				continue
			}
			for _, result := range run.Result.Results {
				if result.Status != storepb.PlanCheckRunResult_Result_SUCCESS {
					continue
				}
				if report := result.GetSqlSummaryReport(); report != nil {
					mi.Payload.ChangedResources = report.ChangedResources
					foundChangedResources = true
					break
				}
			}
		}
	}

	issue, err := stores.GetIssueV2(ctx, &store.FindIssueMessage{PipelineID: &task.PipelineID})
	if err != nil {
		slog.Error("failed to find containing issue", log.BBError(err))
	}
	if issue != nil {
		// Concat issue title and task name as the migration description so that user can see
		// more context of the migration.
		mi.Description = fmt.Sprintf("%s - %s", issue.Title, task.Name)
		mi.IssueUID = &issue.UID
	}

	mi.Source = db.UI
	creator, err := stores.GetUserByID(ctx, task.CreatorID)
	if err != nil {
		// If somehow we unable to find the principal, we just emit the error since it's not
		// critical enough to fail the entire operation.
		slog.Error("Failed to fetch creator for composing the migration info",
			slog.Int("task_id", task.ID),
			log.BBError(err),
		)
	} else {
		mi.Creator = creator.Name
		mi.CreatorID = creator.ID
	}

	statement = strings.TrimSpace(statement)
	// Only baseline and SDL migration can have empty sql statement, which indicates empty database.
	if mi.Type != db.Baseline && mi.Type != db.MigrateSDL && statement == "" {
		return nil, errors.Errorf("empty statement")
	}
	return mi, nil
}

func executeMigration(ctx context.Context, driverCtx context.Context, stores *store.Store, dbFactory *dbfactory.DBFactory, stateCfg *state.State, profile config.Profile, task *store.TaskMessage, taskRunUID int, statement string, sheetID *int, mi *db.MigrationInfo) (string, string, error) {
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
	slog.Debug("Start migration...",
		slog.String("instance", instance.ResourceID),
		slog.String("database", database.DatabaseName),
		slog.String("source", string(mi.Source)),
		slog.String("type", string(mi.Type)),
		slog.String("statement", statementRecord),
	)

	var migrationID string
	opts := db.ExecuteOptions{}
	if task.Type == api.TaskDatabaseDataUpdate && (instance.Engine == storepb.Engine_MYSQL || instance.Engine == storepb.Engine_MARIADB) {
		opts.BeginFunc = func(ctx context.Context, conn *sql.Conn) error {
			updatedTask, err := setThreadIDAndStartBinlogCoordinate(ctx, conn, task, stores)
			if err != nil {
				return errors.Wrap(err, "failed to update the task payload for MySQL rollback SQL")
			}
			task = updatedTask
			return nil
		}
	}
	if task.Type == api.TaskDatabaseDataUpdate && instance.Engine == storepb.Engine_ORACLE {
		// getSetOracleTransactionIdFunc will update the task payload to set the Oracle transaction id, we need to re-retrieve the task to store to the RollbackGenerate.
		opts.EndTransactionFunc = getSetOracleTransactionIDFunc(ctx, task, stores)
	}

	if profile.Mode == common.ReleaseModeDev && stateCfg != nil {
		switch task.Type {
		case api.TaskDatabaseSchemaUpdate, api.TaskDatabaseDataUpdate:
			switch instance.Engine {
			case storepb.Engine_MYSQL:
				opts.IndividualSubmission = true
				opts.UpdateExecutionStatus = func(detail *v1pb.TaskRun_ExecutionDetail) {
					stateCfg.TaskRunExecutionStatuses.Store(taskRunUID,
						state.TaskRunExecutionStatus{
							ExecutionStatus: v1pb.TaskRun_EXECUTING,
							ExecutionDetail: detail,
							UpdateTime:      time.Now(),
						})
				}
			default:
				// do nothing
			}
		}
	}

	migrationID, schema, err := utils.ExecuteMigrationDefault(ctx, driverCtx, stores, stateCfg, taskRunUID, driver, mi, statement, sheetID, opts)
	if err != nil {
		return "", "", err
	}

	// If the migration is a data migration, enable the rollback SQL generation and the type of the driver is Oracle, we need to get the rollback SQL before the transaction is committed.
	if task.Type == api.TaskDatabaseDataUpdate && instance.Engine == storepb.Engine_ORACLE {
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

	if task.Type == api.TaskDatabaseDataUpdate && (instance.Engine == storepb.Engine_MYSQL || instance.Engine == storepb.Engine_MARIADB) {
		conn, err := driver.GetDB().Conn(ctx)
		if err != nil {
			return "", "", errors.Wrap(err, "failed to create connection")
		}
		defer conn.Close()
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
			slog.Error("failed to unmarshal task payload", slog.Int("TaskId", task.ID), log.BBError(err))
			return nil
		}
		// Get oracle current transaction id;
		transactionID, err := tx.QueryContext(ctx, "SELECT RAWTOHEX(tx.xid) FROM v$transaction tx JOIN v$session s ON tx.ses_addr = s.saddr")
		if err != nil {
			slog.Error("failed to transaction id in task", slog.Int("TaskId", task.ID), log.BBError(err))
			return nil
		}
		defer transactionID.Close()
		var txID string
		for transactionID.Next() {
			err := transactionID.Scan(&txID)
			if err != nil {
				slog.Error("failed to the Oracle transaction id in task", slog.Int("TaskId", task.ID), log.BBError(err))
				return nil
			}
		}
		if err := transactionID.Err(); err != nil {
			return err
		}
		payload.TransactionID = txID
		updatedPayload, err := json.Marshal(payload)
		if err != nil {
			slog.Error("failed to unmarshal task payload", slog.Int("TaskId", task.ID), log.BBError(err), slog.Any("payload", updatedPayload))
			return nil
		}
		updatedPayloadString := string(updatedPayload)
		patch := &api.TaskPatch{
			ID:        task.ID,
			UpdaterID: api.SystemBotID,
			Payload:   &updatedPayloadString,
		}
		if _, err = store.UpdateTaskV2(ctx, patch); err != nil {
			slog.Error("failed to update task with new payload", slog.Any("TaskPatch", patch), log.BBError(err))
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
		slog.Warn("binlog is not enabled", slog.Int("task", task.ID))
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
		slog.Warn("binlog is not enabled", slog.Int("task", task.ID))
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

func postMigration(ctx context.Context, stores *store.Store, activityManager *activity.Manager, license enterprise.LicenseService, task *store.TaskMessage, mi *db.MigrationInfo, migrationID string, schema string, sheetID *int) (bool, *api.TaskRunResultPayload, error) {
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
		slog.Error("failed to find containing issue", log.BBError(err))
	}
	if err != nil {
		// If somehow we cannot find the issue, emit the error since it's not fatal.
		slog.Error("failed to find containing issue", log.BBError(err))
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

	writebackBranch, err := isWriteBack(ctx, stores, license, project, repo, task)
	if err != nil {
		return true, nil, err
	}

	slog.Debug("Post migration...",
		slog.String("instance", instance.ResourceID),
		slog.String("database", database.DatabaseName),
		slog.String("writeback_branch", writebackBranch),
	)

	if writebackBranch != "" {
		// Transform the schema to standard style for SDL mode.
		if instance.Engine == storepb.Engine_MYSQL || instance.Engine == storepb.Engine_MARIADB || instance.Engine == storepb.Engine_OCEANBASE {
			standardSchema, err := transform.SchemaTransform(storepb.Engine_MYSQL, schema)
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
			return true, nil, errors.Errorf("failed to sync schema file %s after applying migration %s to %q", latestSchemaFile, mi.Version.Version, database.DatabaseName)
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

		commitID, err := writeBackLatestSchema(ctx, stores, repo, vcs, nil, mi, writebackBranch, latestSchemaFile, schema, bytebaseURL)
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
				slog.Error("Failed to marshal file commit activity after writing back the latest schema",
					slog.Int("task_id", task.ID),
					slog.String("repository", repo.WebURL),
					slog.String("file_path", latestSchemaFile),
					log.BBError(err),
				)
			}

			activityCreate := &store.ActivityMessage{
				CreatorUID:   task.CreatorID,
				ContainerUID: task.PipelineID,
				Type:         api.ActivityPipelineTaskFileCommit,
				Level:        api.ActivityInfo,
				Comment: fmt.Sprintf("Committed the latest schema after applying migration version %s to %q.",
					mi.Version.Version,
					mi.Database,
				),
				Payload: string(payload),
			}

			if _, err := activityManager.CreateActivity(ctx, activityCreate, &activity.Metadata{}); err != nil {
				slog.Error("Failed to create file commit activity after writing back the latest schema",
					slog.Int("task_id", task.ID),
					slog.String("repository", repo.WebURL),
					slog.String("file_path", latestSchemaFile),
					log.BBError(err),
				)
			}
		}
	}

	// Set schema config.
	if sheetID != nil && task.DatabaseID != nil {
		sheet, err := stores.GetSheet(ctx, &store.FindSheetMessage{
			UID: sheetID,
		}, api.SystemBotID)
		if err != nil {
			slog.Error("Failed to get sheet from store", slog.Int("sheetID", *sheetID), log.BBError(err))
		} else if sheet.Payload != nil && (sheet.Payload.DatabaseConfig != nil || sheet.Payload.BaselineDatabaseConfig != nil) {
			databaseSchema, err := stores.GetDBSchema(ctx, *task.DatabaseID)
			if err != nil {
				slog.Error("Failed to get database config from store", slog.Int("sheetID", *sheetID), slog.Int("databaseUID", *task.DatabaseID), log.BBError(err))
			} else {
				updatedDatabaseConfig := mergeDatabaseConfig(sheet.Payload.DatabaseConfig, sheet.Payload.BaselineDatabaseConfig, databaseSchema.GetConfig())
				err = stores.UpdateDBSchema(ctx, *task.DatabaseID, &store.UpdateDBSchemaMessage{
					Config: updatedDatabaseConfig,
				}, api.SystemBotID)
				if err != nil {
					slog.Error("Failed to update database config", slog.Int("sheetID", *sheetID), slog.Int("databaseUID", *task.DatabaseID), log.BBError(err))
				}
			}
		}
	}

	// Remove schema drift anomalies.
	if err := stores.ArchiveAnomalyV2(ctx, &store.ArchiveAnomalyMessage{
		DatabaseUID: task.DatabaseID,
		Type:        api.AnomalyDatabaseSchemaDrift,
	}); err != nil && common.ErrorCode(err) != common.NotFound {
		slog.Error("Failed to archive anomaly",
			slog.String("instance", instance.ResourceID),
			slog.String("database", database.DatabaseName),
			slog.String("type", string(api.AnomalyDatabaseSchemaDrift)),
			log.BBError(err))
	}

	detail := fmt.Sprintf("Applied migration version %s to database %q.", mi.Version.Version, database.DatabaseName)
	if mi.Type == db.Baseline {
		detail = fmt.Sprintf("Established baseline version %s for database %q.", mi.Version.Version, database.DatabaseName)
	}

	storedVersion, err := mi.Version.Marshal()
	if err != nil {
		slog.Error("failed to convert database schema version",
			slog.String("version", mi.Version.Version),
			log.BBError(err),
		)
	}
	return true, &api.TaskRunResultPayload{
		Detail:        detail,
		MigrationID:   migrationID,
		ChangeHistory: fmt.Sprintf("instances/%s/databases/%s/changeHistories/%s", instance.ResourceID, database.DatabaseName, migrationID),
		Version:       storedVersion,
	}, nil
}

func isWriteBack(ctx context.Context, stores *store.Store, license enterprise.LicenseService, project *store.ProjectMessage, repo *store.RepositoryMessage, task *store.TaskMessage) (string, error) {
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
	if instance.Engine == storepb.Engine_RISINGWAVE {
		return "", nil
	}

	if err := license.IsFeatureEnabledForInstance(api.FeatureVCSSchemaWriteBack, instance); err != nil {
		slog.Debug(err.Error(), slog.String("instance", instance.ResourceID))
		return "", nil
	}

	branch := ""
	// Prefer write back to the commit branch than the repo branch.
	if !strings.Contains(repo.BranchFilter, "*") {
		branch = repo.BranchFilter
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

func runMigration(ctx context.Context, driverCtx context.Context, store *store.Store, dbFactory *dbfactory.DBFactory, activityManager *activity.Manager, license enterprise.LicenseService, stateCfg *state.State, profile config.Profile, task *store.TaskMessage, taskRunUID int, migrationType db.MigrationType, statement string, schemaVersion model.Version, sheetID *int) (terminated bool, result *api.TaskRunResultPayload, err error) {
	mi, err := getMigrationInfo(ctx, store, profile, task, migrationType, statement, schemaVersion)
	if err != nil {
		return true, nil, err
	}

	migrationID, schema, err := executeMigration(ctx, driverCtx, store, dbFactory, stateCfg, profile, task, taskRunUID, statement, sheetID, mi)
	if err != nil {
		return true, nil, err
	}
	return postMigration(ctx, store, activityManager, license, task, mi, migrationID, schema, sheetID)
}

// Writes back the latest schema to the repository after migration.
// Returns the commit id on success.
func writeBackLatestSchema(
	ctx context.Context,
	storage *store.Store,
	repository *store.RepositoryMessage,
	vcs *store.ExternalVersionControlMessage,
	pushEvent *vcsplugin.PushEvent,
	mi *db.MigrationInfo,
	writebackBranch, latestSchemaFile string, schema string, bytebaseURL string,
) (string, error) {
	oauthContext := &common.OauthContext{
		ClientID:     vcs.ApplicationID,
		ClientSecret: vcs.Secret,
		AccessToken:  repository.AccessToken,
		RefreshToken: repository.RefreshToken,
		Refresher:    utils.RefreshToken(ctx, storage, repository.WebURL),
	}
	schemaFileMeta, err := vcsplugin.Get(vcs.Type, vcsplugin.ProviderConfig{}).ReadFileMeta(
		ctx,
		oauthContext,
		vcs.InstanceURL,
		repository.ExternalID,
		latestSchemaFile,
		vcsplugin.RefInfo{
			RefType: vcsplugin.RefTypeBranch,
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
		commitTitle := fmt.Sprintf("[Bytebase] %s latest schema for %q after migration %s", verb, mi.Database, mi.Version.Version)
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

	schemaFileCommit := vcsplugin.FileCommitCreate{
		Branch:        writebackBranch,
		CommitMessage: commitMessage,
		Content:       schema,
		AuthorName:    vcsplugin.BytebaseAuthorName,
		AuthorEmail:   vcsplugin.BytebaseAuthorEmail,
	}
	if createSchemaFile {
		slog.Debug("Create latest schema file",
			slog.String("schema_file", latestSchemaFile),
		)

		err := vcsplugin.Get(vcs2.Type, vcsplugin.ProviderConfig{}).CreateFile(
			ctx,
			oauthContext,
			vcs2.InstanceURL,
			repo2.ExternalID,
			latestSchemaFile,
			schemaFileCommit,
		)
		if err != nil {
			return "", errors.Wrapf(err, "failed to create file after applying migration %s to %q", mi.Version.Version, mi.Database)
		}
	} else {
		slog.Debug("Update latest schema file",
			slog.String("schema_file", latestSchemaFile),
		)

		schemaFileCommit.LastCommitID = schemaFileMeta.LastCommitID
		schemaFileCommit.SHA = schemaFileMeta.SHA
		err := vcsplugin.Get(vcs2.Type, vcsplugin.ProviderConfig{}).OverwriteFile(
			ctx,
			oauthContext,
			vcs2.InstanceURL,
			repo2.ExternalID,
			latestSchemaFile,
			schemaFileCommit,
		)
		if err != nil {
			return "", errors.Wrapf(err, "failed to create file after applying migration %s to %q", mi.Version.Version, mi.Database)
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
	schemaFileMeta, err = vcsplugin.Get(vcs2.Type, vcsplugin.ProviderConfig{}).ReadFileMeta(
		ctx,
		oauthContext,
		vcs2.InstanceURL,
		repo2.ExternalID,
		latestSchemaFile,
		vcsplugin.RefInfo{
			RefType: vcsplugin.RefTypeBranch,
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
