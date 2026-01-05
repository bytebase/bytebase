package plancheck

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"strings"
	"time"

	"github.com/bytebase/bytebase/backend/utils"

	"github.com/github/gh-ost/go/logic"
	gomysql "github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/ghost"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/store"
)

// NewGhostSyncExecutor creates a gh-ost sync check executor.
func NewGhostSyncExecutor(store *store.Store, dbFactory *dbfactory.DBFactory) Executor {
	return &GhostSyncExecutor{
		store:     store,
		dbFactory: dbFactory,
	}
}

// GhostSyncExecutor is the gh-ost sync check executor.
type GhostSyncExecutor struct {
	store     *store.Store
	dbFactory *dbfactory.DBFactory
}

// RunForTarget runs the gh-ost sync check for a single target.
func (e *GhostSyncExecutor) RunForTarget(ctx context.Context, target *CheckTarget) (results []*storepb.PlanCheckRunResult_Result, err error) {
	// gh-ost dry run could panic.
	// It may be bytebase who panicked, but that's rare. So
	// capture the error and send it into the result list.
	defer func() {
		if r := recover(); r != nil {
			panicErr, ok := r.(error)
			if !ok {
				panicErr = errors.Errorf("%v", r)
			}

			results = []*storepb.PlanCheckRunResult_Result{
				{
					Status:  storepb.Advice_ERROR,
					Title:   "gh-ost dry run failed",
					Content: panicErr.Error(),
					Code:    common.Internal.Int32(),
					Report:  nil,
				},
			}
			err = nil
		}
	}()

	instanceID, databaseName, err := common.GetInstanceDatabaseID(target.Target)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse target %s", target.Target)
	}

	instance, err := e.store.GetInstance(ctx, &store.FindInstanceMessage{ResourceID: &instanceID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get instance %s", instanceID)
	}
	if instance == nil {
		return nil, errors.Errorf("instance %s not found", instanceID)
	}

	database, err := e.store.GetDatabase(ctx, &store.FindDatabaseMessage{InstanceID: &instance.ResourceID, DatabaseName: &databaseName})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database %q", databaseName)
	}
	if database == nil {
		return nil, errors.Errorf("database not found %q", databaseName)
	}

	adminDataSource := utils.DataSourceFromInstanceWithType(instance, storepb.DataSourceType_ADMIN)
	if adminDataSource == nil {
		return nil, common.Errorf(common.Internal, "admin data source not found for instance %s", instance.ResourceID)
	}

	sheet, err := e.store.GetSheetFull(ctx, target.SheetSha256)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get sheet %s", target.SheetSha256)
	}
	if sheet == nil {
		return nil, errors.Errorf("sheet %s not found", target.SheetSha256)
	}
	statement := sheet.Statement

	// Database secrets feature has been removed
	// To avoid leaking the rendered statement, the error message should use the original statement and not the rendered statement.
	// Database secrets feature removed - using original statement directly
	// Trim trailing semicolons.
	statement = strings.TrimRight(statement, ";")

	tableName, err := ghost.GetTableNameFromStatement(statement)
	if err != nil {
		return nil, common.Wrapf(err, common.Internal, "failed to parse table name from statement, statement: %v", statement)
	}

	// Validate binlog access before attempting migration
	// This prevents retry storms and provides early feedback in plan checks
	driver, err := e.dbFactory.GetAdminDatabaseDriver(ctx, instance, database, db.ConnectionContext{})
	if err != nil {
		return []*storepb.PlanCheckRunResult_Result{
			{
				Status:  storepb.Advice_ERROR,
				Title:   "Failed to connect to database",
				Content: fmt.Sprintf("Cannot establish connection: %v", err),
				Code:    common.Internal.Int32(),
			},
		}, nil
	}
	defer driver.Close(ctx)

	validationResult := ghost.ValidateBinlogAccess(ctx, driver, adminDataSource)
	if !validationResult.Valid {
		title, content := validationResult.GetUserFriendlyError()
		return []*storepb.PlanCheckRunResult_Result{
			{
				Status:  storepb.Advice_ERROR,
				Title:   title,
				Content: content,
				Code:    common.Internal.Int32(),
			},
		}, nil
	}

	migrationContext, err := ghost.NewMigrationContext(ctx, rand.Intn(10000000), database, adminDataSource, tableName, fmt.Sprintf("_dryrun_%d", time.Now().Unix()), statement, true, target.GhostFlags, 20000000)
	if err != nil {
		return nil, common.Wrapf(err, common.Internal, "failed to create migration context")
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
			// Note: We're reusing the ctx from parent scope and creating a new driver
			// because the original driver was already closed
			cleanupCtx := context.Background()
			cleanupDriver, err := e.dbFactory.GetAdminDatabaseDriver(cleanupCtx, instance, database, db.ConnectionContext{})
			if err != nil {
				return errors.Wrapf(err, "failed to get driver for cleanup")
			}
			defer cleanupDriver.Close(cleanupCtx)

			// Use the backup database name of MySQL as the ghost database name.
			ghostDBName := common.BackupDatabaseNameOfEngine(storepb.Engine_MYSQL)
			sql := fmt.Sprintf("DROP TABLE IF EXISTS `%s`.`%s`; DROP TABLE IF EXISTS `%s`.`%s`;",
				ghostDBName,
				migrationContext.GetGhostTableName(),
				ghostDBName,
				migrationContext.GetChangelogTableName(),
			)

			if _, err := cleanupDriver.GetDB().ExecContext(cleanupCtx, sql); err != nil {
				return errors.Wrapf(err, "failed to drop gh-ost temp tables")
			}
			return nil
		}(); err != nil {
			slog.Warn("failed to cleanup gh-ost temp tables", log.BBError(err))
		}
	}()

	if err := migrator.Migrate(); err != nil {
		return []*storepb.PlanCheckRunResult_Result{
			{
				Status:  storepb.Advice_ERROR,
				Title:   "gh-ost dry run failed",
				Content: err.Error(),
				Code:    common.Internal.Int32(),
				Report:  nil,
			},
		}, nil
	}

	return []*storepb.PlanCheckRunResult_Result{
		{
			Status:  storepb.Advice_SUCCESS,
			Title:   "OK",
			Content: "gh-ost dry run succeeded",
			Code:    common.Ok.Int32(),
			Report:  nil,
		},
	}, nil
}
