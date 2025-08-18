package taskrun

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/antlr4-go/antlr/v4"
	"github.com/bytebase/parser/postgresql"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/state"
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/oracle"
	parserbase "github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/parser/pg"
	"github.com/bytebase/bytebase/backend/runner/schemasync"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/store/model"
)

// NewDataUpdateExecutor creates a data update (DML) task executor.
func NewDataUpdateExecutor(store *store.Store, dbFactory *dbfactory.DBFactory, license *enterprise.LicenseService, stateCfg *state.State, schemaSyncer *schemasync.Syncer, profile *config.Profile) Executor {
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
	license      *enterprise.LicenseService
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
	if common.EngineSupportPriorBackup(instance.Metadata.GetEngine()) {
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
								Task:  common.FormatTask(issueN.Project.ResourceID, task.PipelineID, task.Environment, task.ID),
								Error: backupErr.Error(),
							},
						},
					},
				}, common.SystemBotID); err != nil {
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

	terminated, result, err := runMigration(ctx, driverCtx, exec.store, exec.dbFactory, exec.stateCfg, exec.schemaSyncer, exec.profile, task, taskRunUID, statement, task.Payload.GetSchemaVersion(), &sheetID)
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
	payload *storepb.Task,
	task *store.TaskMessage,
	issueN *store.IssueMessage,
	instance *store.InstanceMessage,
	database *store.DatabaseMessage,
) (*storepb.PriorBackupDetail, error) {
	if !payload.GetEnablePriorBackup() {
		return nil, nil
	}

	sourceDatabaseName := common.FormatDatabase(database.InstanceID, database.DatabaseName)
	// Format: instances/{instance}/databases/{database}
	backupDBName := common.BackupDatabaseNameOfEngine(instance.Metadata.GetEngine())
	targetDatabaseName := common.FormatDatabase(database.InstanceID, backupDBName)
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

	tc := parserbase.TransformContext{
		InstanceID:              instance.ResourceID,
		GetDatabaseMetadataFunc: BuildGetDatabaseMetadataFunc(exec.store),
		ListDatabaseNamesFunc:   BuildListDatabaseNamesFunc(exec.store),
		IsCaseSensitive:         store.IsObjectCaseSensitive(instance),
		DatabaseName:            database.DatabaseName,
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
		return nil, errors.Errorf("statement size %d exceeds the limit %d, please disable data backup", len(originStatement), common.MaxSheetCheckSize)
	}

	prefix := "_" + time.Now().Format("20060102150405")
	statements, err := parserbase.TransformDMLToSelect(ctx, instance.Metadata.GetEngine(), tc, originStatement, database.DatabaseName, backupDatabaseName, prefix)
	if err != nil {
		return nil, errors.Wrap(err, "failed to transform DML to select")
	}

	prependStatements, err := getPrependStatements(instance.Metadata.GetEngine(), originStatement)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get prepend statements")
	}

	priorBackupDetail := &storepb.PriorBackupDetail{}
	bbSource := fmt.Sprintf("task %d", task.ID)
	if issueN != nil {
		bbSource = fmt.Sprintf("issue %d", issueN.UID)
	}
	for _, statement := range statements {
		backupStatement := statement.Statement
		if prependStatements != "" {
			backupStatement = prependStatements + backupStatement
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
		default:
			// No action needed for other database engines
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
							Task:     common.FormatTask(issueN.Project.ResourceID, task.PipelineID, task.Environment, task.ID),
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
			}, common.SystemBotID); err != nil {
				slog.Warn("failed to create issue comment", "task", task.ID, log.BBError(err))
			}
		}
	}

	if instance.Metadata.GetEngine() != storepb.Engine_POSTGRES {
		if err := exec.schemaSyncer.SyncDatabaseSchema(ctx, backupDatabase); err != nil {
			slog.Error("failed to sync backup database schema",
				slog.String("database", targetDatabaseName),
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

func BuildGetDatabaseMetadataFunc(storeInstance *store.Store) parserbase.GetDatabaseMetadataFunc {
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

func BuildListDatabaseNamesFunc(storeInstance *store.Store) parserbase.ListDatabaseNamesFunc {
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

func getPrependStatements(engine storepb.Engine, statement string) (string, error) {
	if engine != storepb.Engine_POSTGRES {
		return "", nil
	}

	parseResult, err := pg.ParsePostgreSQL(statement)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse statement")
	}

	visitor := &prependStatementsVisitor{
		statement: statement,
	}
	antlr.ParseTreeWalkerDefault.Walk(visitor, parseResult.Tree)

	return visitor.result, nil
}

// prependStatementsVisitor extracts SET role and search_path statements
type prependStatementsVisitor struct {
	*postgresql.BasePostgreSQLParserListener
	statement string
	result    string
}

func (v *prependStatementsVisitor) EnterVariablesetstmt(ctx *postgresql.VariablesetstmtContext) {
	// If we already found a result, don't process more statements
	if v.result != "" {
		return
	}

	setRest := ctx.Set_rest()
	if setRest == nil {
		return
	}
	setRestMore := setRest.Set_rest_more()
	if setRestMore == nil {
		return
	}
	genericSet := setRestMore.Generic_set()
	if genericSet == nil {
		return
	}
	varName := genericSet.Var_name()
	if varName == nil {
		return
	}
	if len(varName.AllColid()) != 1 {
		return
	}

	name := pg.NormalizePostgreSQLColid(varName.Colid(0))
	if name == "role" || name == "search_path" {
		// Extract the text for this SET statement
		v.result = v.extractStatementText(ctx)
	}
}

// extractStatementText extracts the original text for a SET statement context
// This matches pg_query_go behavior: trim leading/trailing whitespace, preserve internal whitespace
func (v *prependStatementsVisitor) extractStatementText(ctx *postgresql.VariablesetstmtContext) string {
	// Extract text from the original statement
	start := ctx.GetStart().GetStart()
	stop := ctx.GetStop().GetStop()

	// Handle potential edge cases with token positions
	if start < 0 || stop < 0 || start >= len(v.statement) {
		return ""
	}

	// Find the semicolon that ends this statement by looking ahead from the stop token
	endPos := stop + 1
	stmtLen := len(v.statement)
	for endPos < stmtLen {
		char := v.statement[endPos]
		if char == ';' {
			// Include the semicolon and any whitespace before it
			stop = endPos
			break
		}
		if char != ' ' && char != '\t' && char != '\n' && char != '\r' {
			// Hit non-whitespace, non-semicolon character, stop looking
			break
		}
		endPos++
	}

	// Ensure stop doesn't exceed statement length
	if stop >= stmtLen {
		stop = stmtLen - 1
	}

	// Extract the raw text
	text := v.statement[start : stop+1]

	// Match pg_query_go behavior: trim leading and trailing whitespace but preserve internal whitespace
	text = strings.TrimSpace(text)

	// Add semicolon if not present (to match pg_query_go behavior)
	if !strings.HasSuffix(text, ";") {
		text += ";"
	}

	return text
}
