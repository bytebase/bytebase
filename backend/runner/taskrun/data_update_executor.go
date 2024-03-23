package taskrun

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/pkg/errors"

	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/activity"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/state"
	enterprise "github.com/bytebase/bytebase/backend/enterprise/api"
	"github.com/bytebase/bytebase/backend/runner/schemasync"

	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/store/model"
)

// NewDataUpdateExecutor creates a data update (DML) task executor.
func NewDataUpdateExecutor(store *store.Store, dbFactory *dbfactory.DBFactory, activityManager *activity.Manager, license enterprise.LicenseService, stateCfg *state.State, schemaSyncer *schemasync.Syncer, profile config.Profile) Executor {
	return &DataUpdateExecutor{
		store:           store,
		dbFactory:       dbFactory,
		activityManager: activityManager,
		license:         license,
		stateCfg:        stateCfg,
		schemaSyncer:    schemaSyncer,
		profile:         profile,
	}
}

// DataUpdateExecutor is the data update (DML) task executor.
type DataUpdateExecutor struct {
	store           *store.Store
	dbFactory       *dbfactory.DBFactory
	activityManager *activity.Manager
	license         enterprise.LicenseService
	stateCfg        *state.State
	schemaSyncer    *schemasync.Syncer
	profile         config.Profile
}

// RunOnce will run the data update (DML) task executor once.
func (exec *DataUpdateExecutor) RunOnce(ctx context.Context, driverCtx context.Context, task *store.TaskMessage, taskRunUID int) (terminated bool, result *api.TaskRunResultPayload, err error) {
	exec.stateCfg.TaskRunExecutionStatuses.Store(taskRunUID,
		state.TaskRunExecutionStatus{
			ExecutionStatus: v1pb.TaskRun_PRE_EXECUTING,
			UpdateTime:      time.Now(),
		})

	payload := &api.TaskDatabaseDataUpdatePayload{}
	if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
		return true, nil, errors.Wrap(err, "invalid database data update payload")
	}

	statement, err := exec.store.GetSheetStatementByID(ctx, payload.SheetID)
	if err != nil {
		return true, nil, err
	}
	if err := exec.backupData(ctx, driverCtx, statement, payload, task); err != nil {
		return true, nil, err
	}
	version := model.Version{Version: payload.SchemaVersion}
	return runMigration(ctx, driverCtx, exec.store, exec.dbFactory, exec.activityManager, exec.license, exec.stateCfg, exec.profile, task, taskRunUID, db.Data, statement, version, &payload.SheetID)
}

func (exec *DataUpdateExecutor) backupData(
	ctx context.Context,
	driverCtx context.Context,
	statement string,
	payload *api.TaskDatabaseDataUpdatePayload,
	task *store.TaskMessage,
) error {
	if payload.PreUpdateBackupDetail.Database == "" {
		return nil
	}

	instance, err := exec.store.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: &task.InstanceID})
	if err != nil {
		return err
	}
	database, err := exec.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{UID: task.DatabaseID})
	if err != nil {
		return err
	}
	issue, err := exec.store.GetIssueV2(ctx, &store.FindIssueMessage{PipelineID: &task.PipelineID})
	if err != nil {
		return errors.Wrapf(err, "failed to find issue for pipeline %v", task.PipelineID)
	}

	backupInstanceID, backupDatabaseName, err := common.GetInstanceDatabaseID(payload.PreUpdateBackupDetail.Database)
	if err != nil {
		return err
	}
	backupDatabase, err := exec.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{InstanceID: &backupInstanceID, DatabaseName: &backupDatabaseName})
	if err != nil {
		return err
	}
	if backupDatabase == nil {
		return errors.Errorf("backup database %q not found", payload.PreUpdateBackupDetail.Database)
	}

	driver, err := exec.dbFactory.GetAdminDatabaseDriver(driverCtx, instance, database, db.ConnectionContext{})
	if err != nil {
		return err
	}

	suffix := "_" + time.Now().Format("20060102150405")
	statements, err := base.TransformDMLToSelect(instance.Engine, statement, database.DatabaseName, backupDatabaseName, suffix)
	if err != nil {
		return errors.Wrap(err, "failed to transform DML to select")
	}

	for _, statement := range statements {
		if _, err := driver.Execute(driverCtx, statement.Statement, db.ExecuteOptions{}); err != nil {
			return err
		}
		if _, err := driver.Execute(driverCtx, fmt.Sprintf("ALTER TABLE `%s`.`%s` COMMENT = 'issue %d'", backupDatabaseName, statement.TableName, issue.UID), db.ExecuteOptions{}); err != nil {
			return err
		}

		createActivityPayload := api.ActivityPipelineTaskPriorBackupPayload{
			TaskID: task.ID,
			BackupSchemaMetadata: []api.SchemaMetadata{
				{
					Table: statement.TableName,
				},
			},
			IssueName: issue.Title,
			TaskName:  task.Name,
		}
		bytes, err := json.Marshal(createActivityPayload)
		if err != nil {
			return errors.Wrapf(err, "failed to marshal ActivityIssueCreate activity")
		}
		activityCreate := &store.ActivityMessage{
			CreatorUID:        api.SystemBotID,
			ResourceContainer: issue.Project.GetName(),
			ContainerUID:      task.PipelineID,
			Type:              api.ActivityPipelineTaskPriorBackup,
			Level:             api.ActivityInfo,
			Payload:           string(bytes),
		}
		if _, err := exec.activityManager.CreateActivity(ctx, activityCreate, &activity.Metadata{Issue: issue}); err != nil {
			slog.Error("failed to create activity",
				slog.Int("task", task.ID),
				log.BBError(err),
			)
		}
	}

	if err := exec.schemaSyncer.SyncDatabaseSchema(ctx, backupDatabase, true /* force */); err != nil {
		slog.Error("failed to sync backup database schema",
			slog.String("database", payload.PreUpdateBackupDetail.Database),
			log.BBError(err),
		)
	}
	return nil
}
