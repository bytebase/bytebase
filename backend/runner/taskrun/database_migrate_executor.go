package taskrun

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/antlr4-go/antlr/v4"
	"github.com/bytebase/parser/postgresql"
	"github.com/github/gh-ost/go/logic"
	gomysql "github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/bus"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/ghost"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/oracle"
	parserbase "github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/parser/pg"
	"github.com/bytebase/bytebase/backend/runner/schemasync"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/store/model"
	"github.com/bytebase/bytebase/backend/utils"
)

// NewDatabaseMigrateExecutor creates a database migration task executor.
func NewDatabaseMigrateExecutor(store *store.Store, dbFactory *dbfactory.DBFactory, bus *bus.Bus, schemaSyncer *schemasync.Syncer, profile *config.Profile) Executor {
	return &DatabaseMigrateExecutor{
		store:        store,
		dbFactory:    dbFactory,
		bus:          bus,
		schemaSyncer: schemaSyncer,
		profile:      profile,
	}
}

// DatabaseMigrateExecutor is the database migration task executor.
type DatabaseMigrateExecutor struct {
	store        *store.Store
	dbFactory    *dbfactory.DBFactory
	bus          *bus.Bus
	schemaSyncer *schemasync.Syncer
	profile      *config.Profile
}

// RunOnce will run the database migration task executor once.
func (exec *DatabaseMigrateExecutor) RunOnce(ctx context.Context, driverCtx context.Context, task *store.TaskMessage, taskRunUID int) (bool, *storepb.TaskRunResult, error) {
	// Check if this is a release-based task
	if releaseName := task.Payload.GetRelease(); releaseName != "" {
		return exec.runReleaseTask(ctx, driverCtx, task, taskRunUID, releaseName)
	}

	if task.Payload.GetEnableGhost() {
		return exec.runGhostMigration(ctx, driverCtx, task, taskRunUID)
	}
	return exec.runMigrationWithPriorBackup(ctx, driverCtx, task, taskRunUID)
}

func (exec *DatabaseMigrateExecutor) runMigrationWithPriorBackup(ctx context.Context, driverCtx context.Context, task *store.TaskMessage, taskRunUID int) (bool, *storepb.TaskRunResult, error) {
	sheet, err := exec.store.GetSheetFull(ctx, task.Payload.GetSheetSha256())
	if err != nil {
		return true, nil, err
	}
	if sheet == nil {
		return true, nil, errors.Errorf("sheet not found: %s", task.Payload.GetSheetSha256())
	}

	// Handle prior backup if enabled.
	// TransformDMLToSelect will automatically filter out DDL statements,
	// so this works correctly for mixed DDL+DML statements.
	var priorBackupDetail *storepb.PriorBackupDetail
	if task.Payload.GetEnablePriorBackup() {
		instance, err := exec.store.GetInstance(ctx, &store.FindInstanceMessage{ResourceID: &task.InstanceID})
		if err != nil {
			return true, nil, errors.Wrap(err, "failed to get instance")
		}
		if instance == nil {
			return true, nil, errors.Errorf("instance not found for task %v", task.ID)
		}
		database, err := exec.store.GetDatabase(ctx, &store.FindDatabaseMessage{InstanceID: &task.InstanceID, DatabaseName: task.DatabaseName})
		if err != nil {
			return true, nil, errors.Wrap(err, "failed to get database")
		}
		if database == nil {
			return true, nil, errors.Errorf("database not found for task %v", task.ID)
		}

		exec.store.CreateTaskRunLogS(ctx, taskRunUID, time.Now(), exec.profile.DeployID, &storepb.TaskRunLog{
			Type:             storepb.TaskRunLog_PRIOR_BACKUP_START,
			PriorBackupStart: &storepb.TaskRunLog_PriorBackupStart{},
		})

		// Check if we should skip backup or not.
		if common.EngineSupportPriorBackup(database.Engine) {
			var backupErr error
			priorBackupDetail, backupErr = exec.backupData(ctx, driverCtx, sheet.Statement, task.Payload, task, instance, database)
			if backupErr != nil {
				exec.store.CreateTaskRunLogS(ctx, taskRunUID, time.Now(), exec.profile.DeployID, &storepb.TaskRunLog{
					Type: storepb.TaskRunLog_PRIOR_BACKUP_END,
					PriorBackupEnd: &storepb.TaskRunLog_PriorBackupEnd{
						Error: backupErr.Error(),
					},
				})

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
	}

	terminated, result, err := runMigration(ctx, driverCtx, exec.store, exec.dbFactory, exec.schemaSyncer, exec.profile, task, taskRunUID, sheet, "")
	if result != nil {
		// Save prior backup detail to task run result.
		result.PriorBackupDetail = priorBackupDetail
	}
	return terminated, result, err
}

func (exec *DatabaseMigrateExecutor) runGhostMigration(ctx context.Context, driverCtx context.Context, task *store.TaskMessage, taskRunUID int) (bool, *storepb.TaskRunResult, error) {
	sheet, err := exec.store.GetSheetFull(ctx, task.Payload.GetSheetSha256())
	if err != nil {
		return true, nil, err
	}
	if sheet == nil {
		return true, nil, errors.Errorf("sheet not found: %s", task.Payload.GetSheetSha256())
	}
	flags := task.Payload.GetFlags()

	instance, err := exec.store.GetInstance(ctx, &store.FindInstanceMessage{ResourceID: &task.InstanceID})
	if err != nil {
		return true, nil, err
	}
	if instance == nil {
		return true, nil, errors.Errorf("instance %s not found", task.InstanceID)
	}
	database, err := exec.store.GetDatabase(ctx, &store.FindDatabaseMessage{InstanceID: &task.InstanceID, DatabaseName: task.DatabaseName})
	if err != nil {
		return true, nil, err
	}
	if database == nil {
		return true, nil, errors.Errorf("database not found")
	}

	execFunc := func(execCtx context.Context, execStatement string, driver db.Driver, _ db.ExecuteOptions) error {
		// set buffer size to 1 to unblock the sender because there is no listener if the task is canceled.
		migrationError := make(chan error, 1)

		statement := strings.TrimSpace(execStatement)
		// Trim trailing semicolons.
		statement = strings.TrimRight(statement, ";")

		tableName, err := ghost.GetTableNameFromStatement(statement)
		if err != nil {
			return err
		}

		adminDataSource := utils.DataSourceFromInstanceWithType(instance, storepb.DataSourceType_ADMIN)
		if adminDataSource == nil {
			return common.Errorf(common.Internal, "admin data source not found for instance %s", instance.ResourceID)
		}

		migrationContext, err := ghost.NewMigrationContext(ctx, task.ID, database, adminDataSource, tableName, fmt.Sprintf("_%d", time.Now().Unix()), execStatement, false, flags, 10000000)
		if err != nil {
			return errors.Wrap(err, "failed to init migrationContext for gh-ost")
		}
		defer func() {
			// Use migrationContext.Uuid as the tls_config_key by convention.
			// We need to deregister it when gh-ost exits.
			// https://github.com/bytebase/gh-ost2/pull/4
			gomysql.DeregisterTLSConfig(migrationContext.Uuid)
		}()

		migrator := logic.NewMigrator(migrationContext, "bb")

		defer func() {
			if err := func() error {
				ctx := context.Background()

				// Use the backup database name of MySQL as the ghost database name.
				ghostDBName := common.BackupDatabaseNameOfEngine(storepb.Engine_MYSQL)
				sql := fmt.Sprintf("DROP TABLE IF EXISTS `%s`.`%s`; DROP TABLE IF EXISTS `%s`.`%s`;",
					ghostDBName,
					migrationContext.GetGhostTableName(),
					ghostDBName,
					migrationContext.GetChangelogTableName(),
				)

				if _, err := driver.GetDB().ExecContext(ctx, sql); err != nil {
					return errors.Wrapf(err, "failed to drop gh-ost temp tables")
				}
				return nil
			}(); err != nil {
				slog.Warn("failed to cleanup gh-ost temp tables", log.BBError(err))
			}
		}()

		go func() {
			if err := migrator.Migrate(); err != nil {
				slog.Error("failed to run gh-ost migration", log.BBError(err))
				migrationError <- err
				return
			}
			migrationError <- nil
		}()

		select {
		case err := <-migrationError:
			if err != nil {
				return err
			}
			return nil
		case <-execCtx.Done():
			migrationContext.PanicAbort <- errors.New("task canceled")
			return errors.New("task canceled")
		}
	}

	return runMigrationWithFunc(ctx, driverCtx, exec.store, exec.dbFactory, exec.schemaSyncer, exec.profile, task, taskRunUID, sheet, "", execFunc)
}

func (exec *DatabaseMigrateExecutor) runReleaseTask(ctx context.Context, driverCtx context.Context, task *store.TaskMessage, taskRunUID int, releaseName string) (bool, *storepb.TaskRunResult, error) {
	// Parse release name to get project ID and release UID
	_, releaseUID, err := common.GetProjectReleaseUID(releaseName)
	if err != nil {
		return true, nil, errors.Wrapf(err, "failed to parse release name %q", releaseName)
	}

	// Fetch the release
	release, err := exec.store.GetReleaseByUID(ctx, releaseUID)
	if err != nil {
		return true, nil, errors.Wrapf(err, "failed to get release %d", releaseUID)
	}
	if release == nil {
		return true, nil, errors.Errorf("release %d not found", releaseUID)
	}

	// Get existing revisions for this database
	revisions, err := exec.store.ListRevisions(ctx, &store.FindRevisionMessage{
		InstanceID:   &task.InstanceID,
		DatabaseName: task.DatabaseName,
	})
	if err != nil {
		return true, nil, errors.Wrapf(err, "failed to list revisions for database %q", *task.DatabaseName)
	}

	// Build map of applied versions
	appliedVersions := make(map[string]bool)
	var maxDeclarativeVersion *model.Version
	var maxDeclarativeVersionString string
	for _, revision := range revisions {
		switch revision.Payload.Type {
		case storepb.SchemaChangeType_VERSIONED:
			appliedVersions[revision.Version] = true
		case storepb.SchemaChangeType_DECLARATIVE:
			v, err := model.NewVersion(revision.Version)
			if err != nil {
				return true, nil, errors.Wrapf(err, "failed to parse revision version %q", revision.Version)
			}
			if maxDeclarativeVersion == nil || maxDeclarativeVersion.LessThan(v) {
				maxDeclarativeVersion = v
				maxDeclarativeVersionString = revision.Version
			}
		default:
			// Ignore other schema change types
		}
	}

	// Execute unapplied files in order
	for _, file := range release.Payload.Files {
		switch file.Type {
		case storepb.SchemaChangeType_VERSIONED:
			// Skip if already applied
			if appliedVersions[file.Version] {
				slog.Info("skipping already applied version",
					slog.String("version", file.Version),
					slog.String("database", *task.DatabaseName))
				continue
			}

			// Fetch and execute this file
			sheet, err := exec.store.GetSheetFull(ctx, file.SheetSha256)
			if err != nil {
				return true, nil, errors.Wrapf(err, "failed to get sheet %s for version %s", file.SheetSha256, file.Version)
			}
			if sheet == nil {
				return true, nil, errors.Errorf("sheet not found: %s", file.SheetSha256)
			}

			slog.Info("executing release file",
				slog.String("version", file.Version),
				slog.String("database", *task.DatabaseName),
				slog.String("file", file.Path))

			// Execute migration for this file
			_, _, err = runMigration(ctx, driverCtx, exec.store, exec.dbFactory, exec.schemaSyncer, exec.profile, task, taskRunUID, sheet, file.Version)
			if err != nil {
				return true, nil, errors.Wrapf(err, "failed to execute release file %s (version %s)", file.Path, file.Version)
			}

		case storepb.SchemaChangeType_DECLARATIVE:
			// Skip if a higher or equal version has been applied
			v, err := model.NewVersion(file.Version)
			if err != nil {
				return true, nil, errors.Wrapf(err, "failed to parse file version %q", file.Version)
			}
			if maxDeclarativeVersion != nil && v.LessThanOrEqual(maxDeclarativeVersion) {
				slog.Info("skipping declarative file with version <= max applied",
					slog.String("version", file.Version),
					slog.String("max_applied", maxDeclarativeVersionString),
					slog.String("database", *task.DatabaseName))
				continue
			}

			// Fetch and execute this file
			sheet, err := exec.store.GetSheetFull(ctx, file.SheetSha256)
			if err != nil {
				return true, nil, errors.Wrapf(err, "failed to get sheet %s for version %s", file.SheetSha256, file.Version)
			}
			if sheet == nil {
				return true, nil, errors.Errorf("sheet not found: %s", file.SheetSha256)
			}

			slog.Info("executing declarative release file",
				slog.String("version", file.Version),
				slog.String("database", *task.DatabaseName),
				slog.String("file", file.Path))

			// Get instance and database for SDL diff
			instance, err := exec.store.GetInstance(ctx, &store.FindInstanceMessage{ResourceID: &task.InstanceID})
			if err != nil {
				return true, nil, errors.Wrapf(err, "failed to get instance %s", task.InstanceID)
			}
			database, err := exec.store.GetDatabase(ctx, &store.FindDatabaseMessage{InstanceID: &task.InstanceID, DatabaseName: task.DatabaseName})
			if err != nil {
				return true, nil, errors.Wrapf(err, "failed to get database %s", *task.DatabaseName)
			}

			// Create execFunc that uses SDL diff logic (same pattern as SchemaDeclareExecutor)
			execFunc := func(ctx context.Context, execStatement string, driver db.Driver, opts db.ExecuteOptions) error {
				opts.LogComputeDiffStart()
				migrationSQL, err := diff(ctx, exec.store, instance, database, execStatement)
				if err != nil {
					opts.LogComputeDiffEnd(err.Error())
					return errors.Wrapf(err, "failed to diff database schema")
				}
				opts.LogComputeDiffEnd("")

				// Log statement string.
				opts.LogCommandStatement = true
				if _, err := driver.Execute(ctx, migrationSQL, opts); err != nil {
					return err
				}
				return nil
			}

			// Temporarily change task type to DATABASE_SDL so revision is created with correct type
			originalTaskType := task.Type
			task.Type = storepb.Task_DATABASE_SDL

			// Execute SDL migration for this file using the diff logic
			_, _, err = runMigrationWithFunc(ctx, driverCtx, exec.store, exec.dbFactory, exec.schemaSyncer, exec.profile, task, taskRunUID, sheet, file.Version, execFunc)

			// Restore original task type
			task.Type = originalTaskType

			if err != nil {
				return true, nil, errors.Wrapf(err, "failed to execute declarative release file %s (version %s)", file.Path, file.Version)
			}

		default:
			return true, nil, errors.Errorf("unsupported release file type %q", file.Type)
		}
	}

	// All files executed successfully
	return true, &storepb.TaskRunResult{
		Detail: "All release files executed successfully",
	}, nil
}

func (exec *DatabaseMigrateExecutor) shouldSkipBackupError(ctx context.Context, task *store.TaskMessage) (bool, error) {
	plan, err := exec.store.GetPlan(ctx, &store.FindPlanMessage{UID: &task.PlanID})
	if err != nil {
		return false, errors.Wrapf(err, "failed to get plan %v", task.PlanID)
	}
	if plan == nil {
		return false, errors.Errorf("plan %v not found", task.PlanID)
	}

	project, projectErr := exec.store.GetProject(ctx, &store.FindProjectMessage{ResourceID: &plan.ProjectID})
	if projectErr != nil {
		return false, errors.Wrapf(projectErr, "failed to get project %v", plan.ProjectID)
	}
	if project == nil {
		return false, errors.Errorf("project not found for plan %v", task.PlanID)
	}
	if project.Setting == nil {
		return false, nil
	}
	return project.Setting.SkipBackupErrors, nil
}

func (exec *DatabaseMigrateExecutor) backupData(
	ctx context.Context,
	driverCtx context.Context,
	originStatement string,
	payload *storepb.Task,
	task *store.TaskMessage,
	instance *store.InstanceMessage,
	database *store.DatabaseMessage,
) (*storepb.PriorBackupDetail, error) {
	if !payload.GetEnablePriorBackup() {
		return nil, nil
	}

	sourceDatabaseName := common.FormatDatabase(database.InstanceID, database.DatabaseName)
	// Format: instances/{instance}/databases/{database}
	backupDBName := common.BackupDatabaseNameOfEngine(database.Engine)
	targetDatabaseName := common.FormatDatabase(database.InstanceID, backupDBName)
	var backupDatabase *store.DatabaseMessage
	var backupDriver db.Driver

	backupInstanceID, backupDatabaseName, err := common.GetInstanceDatabaseID(targetDatabaseName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse backup database")
	}

	if database.Engine != storepb.Engine_POSTGRES {
		backupDatabase, err = exec.store.GetDatabase(ctx, &store.FindDatabaseMessage{InstanceID: &backupInstanceID, DatabaseName: &backupDatabaseName})
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

	project, err := exec.store.GetProject(ctx, &store.FindProjectMessage{ResourceID: &database.ProjectID})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get project")
	}
	driver, err := exec.dbFactory.GetAdminDatabaseDriver(driverCtx, instance, database, db.ConnectionContext{
		TenantMode: project.Setting.GetPostgresDatabaseTenantMode(),
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get database driver")
	}
	defer driver.Close(driverCtx)

	tc := parserbase.TransformContext{
		InstanceID:              instance.ResourceID,
		GetDatabaseMetadataFunc: buildGetDatabaseMetadataFunc(exec.store),
		ListDatabaseNamesFunc:   buildListDatabaseNamesFunc(exec.store),
		IsCaseSensitive:         store.IsObjectCaseSensitive(instance),
		DatabaseName:            database.DatabaseName,
	}
	if database.Engine == storepb.Engine_ORACLE {
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
	statements, err := parserbase.TransformDMLToSelect(ctx, database.Engine, tc, originStatement, database.DatabaseName, backupDatabaseName, prefix)
	if err != nil {
		return nil, errors.Wrap(err, "failed to transform DML to select")
	}

	prependStatements, err := getPrependStatements(database.Engine, originStatement)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get prepend statements")
	}

	priorBackupDetail := &storepb.PriorBackupDetail{}
	bbSource := fmt.Sprintf("task %d", task.ID)
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
		if database.Engine == storepb.Engine_POSTGRES {
			item.TargetTable = &storepb.PriorBackupDetail_Item_Table{
				Database: sourceDatabaseName,
				// postgres uses schema as the backup database name currently.
				Schema: backupDatabaseName,
				Table:  statement.TargetTableName,
			}
		}
		priorBackupDetail.Items = append(priorBackupDetail.Items, item)
	}

	if database.Engine != storepb.Engine_POSTGRES {
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

func buildGetDatabaseMetadataFunc(storeInstance *store.Store) parserbase.GetDatabaseMetadataFunc {
	return func(ctx context.Context, instanceID, databaseName string) (string, *model.DatabaseMetadata, error) {
		database, err := storeInstance.GetDatabase(ctx, &store.FindDatabaseMessage{
			InstanceID:   &instanceID,
			DatabaseName: &databaseName,
		})
		if err != nil {
			return "", nil, err
		}
		if database == nil {
			return "", nil, nil
		}
		databaseMetadata, err := storeInstance.GetDBSchema(ctx, &store.FindDBSchemaMessage{
			InstanceID:   instanceID,
			DatabaseName: databaseName,
		})
		if err != nil {
			return "", nil, err
		}
		if databaseMetadata == nil {
			return "", nil, nil
		}
		return databaseName, databaseMetadata, nil
	}
}

func buildListDatabaseNamesFunc(storeInstance *store.Store) parserbase.ListDatabaseNamesFunc {
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

	parseResults, err := pg.ParsePostgreSQL(statement)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse statement")
	}

	visitor := &prependStatementsVisitor{
		statement: statement,
	}

	// Walk through all statements to find the first SET role/search_path statement
	// The visitor will stop after finding the first one due to its internal check
	for _, result := range parseResults {
		antlr.ParseTreeWalkerDefault.Walk(visitor, result.Tree)
		// If we found a result, stop walking remaining statements
		if visitor.result != "" {
			break
		}
	}

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
