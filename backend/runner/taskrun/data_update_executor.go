package taskrun

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/state"
	enterprise "github.com/bytebase/bytebase/backend/enterprise/api"
	"github.com/bytebase/bytebase/backend/runner/schemasync"

	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/oracle"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/parser/sql/ast"
	pgrawparser "github.com/bytebase/bytebase/backend/plugin/parser/sql/engine/pg"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/store/model"
)

// NewDataUpdateExecutor creates a data update (DML) task executor.
func NewDataUpdateExecutor(store *store.Store, dbFactory *dbfactory.DBFactory, license enterprise.LicenseService, stateCfg *state.State, schemaSyncer *schemasync.Syncer, profile *config.Profile) Executor {
	return &DataUpdateExecutor{
		store:        store,
		dbFactory:    dbFactory,
		license:      license,
		stateCfg:     stateCfg,
		schemaSyncer: schemaSyncer,
		profile:      profile,
	}
}

// DataUpdateExecutor is the data update (DML) task executor.
type DataUpdateExecutor struct {
	store        *store.Store
	dbFactory    *dbfactory.DBFactory
	license      enterprise.LicenseService
	stateCfg     *state.State
	schemaSyncer *schemasync.Syncer
	profile      *config.Profile
}

// RunOnce will run the data update (DML) task executor once.
func (exec *DataUpdateExecutor) RunOnce(ctx context.Context, driverCtx context.Context, task *store.TaskMessage, taskRunUID int) (bool, *storepb.TaskRunResult, error) {
	sheetID := int(task.Payload.GetSheetId())
	statement, err := exec.store.GetSheetStatementByID(ctx, sheetID)
	if err != nil {
		return true, nil, err
	}

	instance, err := exec.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &task.InstanceID})
	if err != nil {
		return true, nil, errors.Wrap(err, "failed to get instance")
	}
	if instance == nil {
		return true, nil, errors.Errorf("instance not found for task %v", task.ID)
	}
	database, err := exec.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{InstanceID: &task.InstanceID, DatabaseName: task.DatabaseName})
	if err != nil {
		return true, nil, errors.Wrap(err, "failed to get database")
	}
	if database == nil {
		return true, nil, errors.Errorf("database not found for task %v", task.ID)
	}
	issueN, err := exec.store.GetIssueV2(ctx, &store.FindIssueMessage{PipelineID: &task.PipelineID})
	if err != nil {
		return true, nil, errors.Wrapf(err, "failed to find issue for pipeline %v", task.PipelineID)
	}

	exec.store.CreateTaskRunLogS(ctx, taskRunUID, time.Now(), exec.profile.DeployID, &storepb.TaskRunLog{
		Type:             storepb.TaskRunLog_PRIOR_BACKUP_START,
		PriorBackupStart: &storepb.TaskRunLog_PriorBackupStart{},
	})

	var priorBackupDetail *storepb.PriorBackupDetail
	// Check if we should skip backup or not.
	switch instance.Metadata.GetEngine() {
	case storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_MSSQL, storepb.Engine_ORACLE, storepb.Engine_POSTGRES:
		var backupErr error
		priorBackupDetail, backupErr = exec.backupData(ctx, driverCtx, statement, task.Payload, task, issueN, instance, database)
		if backupErr != nil {
			exec.store.CreateTaskRunLogS(ctx, taskRunUID, time.Now(), exec.profile.DeployID, &storepb.TaskRunLog{
				Type: storepb.TaskRunLog_PRIOR_BACKUP_END,
				PriorBackupEnd: &storepb.TaskRunLog_PriorBackupEnd{
					Error: backupErr.Error(),
				},
			})
			// Create issue comment for backup error.
			if issueN != nil {
				if _, err := exec.store.CreateIssueComment(ctx, &store.IssueCommentMessage{
					IssueUID: issueN.UID,
					Payload: &storepb.IssueCommentPayload{
						Event: &storepb.IssueCommentPayload_TaskPriorBackup_{
							TaskPriorBackup: &storepb.IssueCommentPayload_TaskPriorBackup{
								Task:  common.FormatTask(issueN.Project.ResourceID, task.PipelineID, task.StageID, task.ID),
								Error: backupErr.Error(),
							},
						},
					},
				}, api.SystemBotID); err != nil {
					slog.Warn("failed to create issue comment", "task", task.ID, log.BBError(err), "backup error", backupErr)
				}
			}
			// Check if we should skip backup error and continue to run migration.
			skip, err := exec.shouldSkipBackupError(ctx, task)
			if err != nil {
				return true, nil, errors.Errorf("failed to check skip backup error or not: %v", err)
			}
			if !skip {
				return true, nil, backupErr
			}
		} else {
			exec.store.CreateTaskRunLogS(ctx, taskRunUID, time.Now(), exec.profile.DeployID, &storepb.TaskRunLog{
				Type: storepb.TaskRunLog_PRIOR_BACKUP_END,
				PriorBackupEnd: &storepb.TaskRunLog_PriorBackupEnd{
					PriorBackupDetail: priorBackupDetail,
				},
			})
		}
	}

	terminated, result, err := runMigration(ctx, driverCtx, exec.store, exec.dbFactory, exec.stateCfg, exec.schemaSyncer, exec.profile, task, taskRunUID, db.Data, statement, task.Payload.GetSchemaVersion(), &sheetID)
	if result != nil {
		// Save prior backup detail to task run result.
		result.PriorBackupDetail = priorBackupDetail
	}
	return terminated, result, err
}

func (exec *DataUpdateExecutor) shouldSkipBackupError(ctx context.Context, task *store.TaskMessage) (bool, error) {
	pipeline, pipelineErr := exec.store.GetPipelineV2ByID(ctx, task.PipelineID)
	if pipelineErr != nil {
		return false, errors.Wrapf(pipelineErr, "failed to get pipeline %v", task.PipelineID)
	}
	project, projectErr := exec.store.GetProjectV2(ctx, &store.FindProjectMessage{ResourceID: &pipeline.ProjectID})
	if projectErr != nil {
		return false, errors.Wrapf(projectErr, "failed to get project %v", pipeline.ProjectID)
	}
	if project == nil {
		return false, errors.Errorf("project not found for pipeline %v", task.PipelineID)
	}
	if project.Setting == nil {
		return false, nil
	}
	return project.Setting.SkipBackupErrors, nil
}

func (exec *DataUpdateExecutor) backupData(
	ctx context.Context,
	driverCtx context.Context,
	originStatement string,
	payload *storepb.TaskPayload,
	task *store.TaskMessage,
	issueN *store.IssueMessage,
	instance *store.InstanceMessage,
	database *store.DatabaseMessage,
) (*storepb.PriorBackupDetail, error) {
	if payload.GetPreUpdateBackupDetail().GetDatabase() == "" {
		return nil, nil
	}

	sourceDatabaseName := common.FormatDatabase(database.InstanceID, database.DatabaseName)
	// Format: instances/{instance}/databases/{database}
	targetDatabaseName := payload.PreUpdateBackupDetail.Database
	var backupDatabase *store.DatabaseMessage
	var backupDriver db.Driver

	backupInstanceID, backupDatabaseName, err := common.GetInstanceDatabaseID(targetDatabaseName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse backup database")
	}

	if instance.Metadata.GetEngine() != storepb.Engine_POSTGRES {
		backupDatabase, err = exec.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{InstanceID: &backupInstanceID, DatabaseName: &backupDatabaseName})
		if err != nil {
			return nil, errors.Wrap(err, "failed to get backup database")
		}
		if backupDatabase == nil {
			return nil, errors.Errorf("backup database %q not found", targetDatabaseName)
		}
		backupDriver, err = exec.dbFactory.GetAdminDatabaseDriver(driverCtx, instance, backupDatabase, db.ConnectionContext{})
		if err != nil {
			return nil, errors.Wrap(err, "failed to get backup database driver")
		}
		defer backupDriver.Close(driverCtx)
	}

	useDatabaseOwner, err := getUseDatabaseOwner(ctx, exec.store, instance, database)
	if err != nil {
		return nil, errors.Wrap(err, "failed to check use database owner")
	}
	driver, err := exec.dbFactory.GetAdminDatabaseDriver(driverCtx, instance, database, db.ConnectionContext{
		UseDatabaseOwner: useDatabaseOwner,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get database driver")
	}
	defer driver.Close(driverCtx)

	tc := base.TransformContext{
		InstanceID:              instance.ResourceID,
		GetDatabaseMetadataFunc: BuildGetDatabaseMetadataFunc(exec.store),
		ListDatabaseNamesFunc:   BuildListDatabaseNamesFunc(exec.store),
		IsCaseSensitive:         store.IsObjectCaseSensitive(instance),
	}
	if instance.Metadata.GetEngine() == storepb.Engine_ORACLE {
		oracleDriver, ok := driver.(*oracle.Driver)
		if ok {
			if version, err := oracleDriver.GetVersion(); err == nil {
				tc.Version = version
			}
		}
	}

	if len(originStatement) > common.MaxSheetCheckSize {
		return nil, errors.Errorf("statement size %d exceeds the limit %d", len(originStatement), common.MaxSheetCheckSize)
	}

	prefix := "_" + time.Now().Format("20060102150405")
	statements, err := base.TransformDMLToSelect(ctx, instance.Metadata.GetEngine(), tc, originStatement, database.DatabaseName, backupDatabaseName, prefix)
	if err != nil {
		return nil, errors.Wrap(err, "failed to transform DML to select")
	}

	preAppendStatements, err := getPreAppendStatements(instance.Metadata.GetEngine(), originStatement)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get pre append statements")
	}

	priorBackupDetail := &storepb.PriorBackupDetail{}
	bbSource := fmt.Sprintf("task %d", task.ID)
	if issueN != nil {
		bbSource = fmt.Sprintf("issue %d", issueN.UID)
	}
	for _, statement := range statements {
		backupStatement := statement.Statement
		if preAppendStatements != "" {
			backupStatement = preAppendStatements + backupStatement
		}
		if _, err := driver.Execute(driverCtx, backupStatement, db.ExecuteOptions{}); err != nil {
			return nil, errors.Wrapf(err, "failed to execute backup statement %q", backupStatement)
		}
		switch instance.Metadata.GetEngine() {
		case storepb.Engine_TIDB:
			if _, err := driver.Execute(driverCtx, fmt.Sprintf("ALTER TABLE `%s`.`%s` COMMENT = '%s, source table (%s, %s)'", backupDatabaseName, statement.TargetTableName, bbSource, database.DatabaseName, statement.SourceTableName), db.ExecuteOptions{}); err != nil {
				return nil, errors.Wrap(err, "failed to set table comment")
			}
		case storepb.Engine_MYSQL:
			if _, err := driver.Execute(driverCtx, fmt.Sprintf("ALTER TABLE `%s`.`%s` COMMENT = '%s, source table (%s, %s)'", backupDatabaseName, statement.TargetTableName, bbSource, database.DatabaseName, statement.SourceTableName), db.ExecuteOptions{}); err != nil {
				return nil, errors.Wrap(err, "failed to set table comment")
			}
		case storepb.Engine_MSSQL:
			schemaName := statement.SourceSchema
			if schemaName == "" {
				schemaName = "dbo"
			}
			if _, err := backupDriver.Execute(driverCtx, fmt.Sprintf("EXEC sp_addextendedproperty 'MS_Description', '%s, source table (%s, %s, %s)', 'SCHEMA', 'dbo', 'TABLE', '%s'", bbSource, database.DatabaseName, schemaName, statement.SourceTableName, statement.TargetTableName), db.ExecuteOptions{}); err != nil {
				return nil, errors.Wrap(err, "failed to set table comment")
			}
		case storepb.Engine_POSTGRES:
			schemaName := statement.SourceSchema
			if schemaName == "" {
				schemaName = "public"
			}
			if _, err := driver.Execute(driverCtx, fmt.Sprintf(`COMMENT ON TABLE "%s"."%s" IS '%s, source table (%s, %s)'`, backupDatabaseName, statement.TargetTableName, bbSource, schemaName, statement.SourceTableName), db.ExecuteOptions{}); err != nil {
				return nil, errors.Wrap(err, "failed to set table comment")
			}
		case storepb.Engine_ORACLE:
			if _, err := driver.Execute(driverCtx, fmt.Sprintf(`COMMENT ON TABLE "%s"."%s" IS '%s, source table (%s, %s)'`, backupDatabaseName, statement.TargetTableName, bbSource, database.DatabaseName, statement.SourceTableName), db.ExecuteOptions{}); err != nil {
				return nil, errors.Wrap(err, "failed to set table comment")
			}
		}

		item := &storepb.PriorBackupDetail_Item{
			SourceTable: &storepb.PriorBackupDetail_Item_Table{
				Database: sourceDatabaseName,
				Schema:   statement.SourceSchema,
				Table:    statement.SourceTableName,
			},
			TargetTable: &storepb.PriorBackupDetail_Item_Table{
				Database: targetDatabaseName,
				Schema:   "",
				Table:    statement.TargetTableName,
			},
			StartPosition: statement.StartPosition,
			EndPosition:   statement.EndPosition,
		}
		if instance.Metadata.GetEngine() == storepb.Engine_POSTGRES {
			item.TargetTable = &storepb.PriorBackupDetail_Item_Table{
				Database: sourceDatabaseName,
				// postgres uses schema as the backup database name currently.
				Schema: backupDatabaseName,
				Table:  statement.TargetTableName,
			}
		}
		priorBackupDetail.Items = append(priorBackupDetail.Items, item)

		if issueN != nil {
			if _, err := exec.store.CreateIssueComment(ctx, &store.IssueCommentMessage{
				IssueUID: issueN.UID,
				Payload: &storepb.IssueCommentPayload{
					Event: &storepb.IssueCommentPayload_TaskPriorBackup_{
						TaskPriorBackup: &storepb.IssueCommentPayload_TaskPriorBackup{
							Task:     common.FormatTask(issueN.Project.ResourceID, task.PipelineID, task.StageID, task.ID),
							Database: backupDatabaseName,
							Tables: []*storepb.IssueCommentPayload_TaskPriorBackup_Table{
								{
									Schema: "",
									Table:  statement.TargetTableName,
								},
							},
						},
					},
				},
			}, api.SystemBotID); err != nil {
				slog.Warn("failed to create issue comment", "task", task.ID, log.BBError(err))
			}
		}
	}

	if instance.Metadata.GetEngine() != storepb.Engine_POSTGRES {
		if err := exec.schemaSyncer.SyncDatabaseSchema(ctx, backupDatabase); err != nil {
			slog.Error("failed to sync backup database schema",
				slog.String("database", payload.PreUpdateBackupDetail.Database),
				log.BBError(err),
			)
		}
	} else {
		if err := exec.schemaSyncer.SyncDatabaseSchema(ctx, database); err != nil {
			slog.Error("failed to sync backup database schema",
				slog.String("database", fmt.Sprintf("/instances/%s/databases/%s", instance.ResourceID, database.DatabaseName)),
				log.BBError(err),
			)
		}
	}

	return priorBackupDetail, nil
}

func BuildGetDatabaseMetadataFunc(storeInstance *store.Store) base.GetDatabaseMetadataFunc {
	return func(ctx context.Context, instanceID, databaseName string) (string, *model.DatabaseMetadata, error) {
		database, err := storeInstance.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
			InstanceID:   &instanceID,
			DatabaseName: &databaseName,
		})
		if err != nil {
			return "", nil, err
		}
		if database == nil {
			return "", nil, nil
		}
		databaseMetadata, err := storeInstance.GetDBSchema(ctx, database.InstanceID, database.DatabaseName)
		if err != nil {
			return "", nil, err
		}
		if databaseMetadata == nil {
			return "", nil, nil
		}
		return databaseName, databaseMetadata.GetDatabaseMetadata(), nil
	}
}

func BuildListDatabaseNamesFunc(storeInstance *store.Store) base.ListDatabaseNamesFunc {
	return func(ctx context.Context, instanceID string) ([]string, error) {
		databases, err := storeInstance.ListDatabases(ctx, &store.FindDatabaseMessage{
			InstanceID: &instanceID,
		})
		if err != nil {
			return nil, err
		}
		names := make([]string, 0, len(databases))
		for _, database := range databases {
			names = append(names, database.DatabaseName)
		}
		return names, nil
	}
}

func getPreAppendStatements(engine storepb.Engine, statement string) (string, error) {
	if engine != storepb.Engine_POSTGRES {
		return "", nil
	}

	nodes, err := pgrawparser.Parse(pgrawparser.ParseContext{}, statement)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse statement")
	}

	for _, node := range nodes {
		if n, ok := node.(*ast.VariableSetStmt); ok {
			if n.Name == "role" {
				return n.Text(), nil
			}
		}
	}

	return "", nil
}
