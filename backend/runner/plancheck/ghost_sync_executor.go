package plancheck

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"strings"
	"time"

	"github.com/github/gh-ost/go/logic"
	gomysql "github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/ghost"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
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

// Run runs the gh-ost sync check executor.
func (e *GhostSyncExecutor) Run(ctx context.Context, config *storepb.PlanCheckRunConfig) (results []*storepb.PlanCheckRunResult_Result, err error) {
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
					Status:  storepb.PlanCheckRunResult_Result_ERROR,
					Title:   "gh-ost dry run failed",
					Content: panicErr.Error(),
					Code:    common.Internal.Int32(),
					Report:  nil,
				},
			}
			err = nil
		}
	}()

	instance, err := e.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &config.InstanceId})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get instance %s", config.InstanceId)
	}
	if instance == nil {
		return nil, errors.Errorf("instance %s not found", config.InstanceId)
	}

	database, err := e.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{InstanceID: &instance.ResourceID, DatabaseName: &config.DatabaseName})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database %q", config.DatabaseName)
	}
	if database == nil {
		return nil, errors.Errorf("database not found %q", config.DatabaseName)
	}

	adminDataSource := utils.DataSourceFromInstanceWithType(instance, storepb.DataSourceType_ADMIN)
	if adminDataSource == nil {
		return nil, common.Errorf(common.Internal, "admin data source not found for instance %s", instance.ResourceID)
	}

	sheetUID := int(config.SheetUid)
	sheet, err := e.store.GetSheet(ctx, &store.FindSheetMessage{UID: &sheetUID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get sheet %d", sheetUID)
	}
	if sheet == nil {
		return nil, errors.Errorf("sheet %d not found", sheetUID)
	}
	statement, err := e.store.GetSheetStatementByID(ctx, sheetUID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get sheet statement %d", sheetUID)
	}

	materials := utils.GetSecretMapFromDatabaseMessage(database)
	// To avoid leaking the rendered statement, the error message should use the original statement and not the rendered statement.
	renderedStatement := utils.RenderStatement(statement, materials)
	// Trim trailing semicolons.
	renderedStatement = strings.TrimRight(renderedStatement, ";")

	tableName, err := ghost.GetTableNameFromStatement(renderedStatement)
	if err != nil {
		return nil, common.Wrapf(err, common.Internal, "failed to parse table name from statement, statement: %v", statement)
	}

	migrationContext, err := ghost.NewMigrationContext(ctx, rand.Intn(10000000), database, adminDataSource, tableName, fmt.Sprintf("_dryrun_%d", time.Now().Unix()), renderedStatement, true, config.GhostFlags, 20000000)
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
			ctx := context.Background()
			driver, err := e.dbFactory.GetAdminDatabaseDriver(ctx, instance, database, db.ConnectionContext{
				OperationalComponent: "remove-ghost",
			})
			if err != nil {
				return errors.Wrapf(err, "failed to get driver")
			}
			defer driver.Close(ctx)

			sql := fmt.Sprintf("DROP TABLE IF EXISTS `%s`.`%s`; DROP TABLE IF EXISTS `%s`.`%s`;",
				"bbdataarchive",
				migrationContext.GetGhostTableName(),
				"bbdataarchive",
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

	if err := migrator.Migrate(); err != nil {
		return []*storepb.PlanCheckRunResult_Result{
			{
				Status:  storepb.PlanCheckRunResult_Result_ERROR,
				Title:   "gh-ost dry run failed",
				Content: err.Error(),
				Code:    common.Internal.Int32(),
				Report:  nil,
			},
		}, nil
	}

	return []*storepb.PlanCheckRunResult_Result{
		{
			Status:  storepb.PlanCheckRunResult_Result_SUCCESS,
			Title:   "OK",
			Content: "gh-ost dry run succeeded",
			Code:    common.Ok.Int32(),
			Report:  nil,
		},
	}, nil
}
