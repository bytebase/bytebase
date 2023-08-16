package plancheck

import (
	"context"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/plugin/db/mysql"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// NewPITRMySQLExecutor creates a plan check PITR MySQL executor.
func NewPITRMySQLExecutor(store *store.Store, dbFactory *dbfactory.DBFactory) Executor {
	return &PITRMySQLExecutor{
		store:     store,
		dbFactory: dbFactory,
	}
}

// PITRMySQLExecutor is to check if the MySQL database is ready for PITR.
type PITRMySQLExecutor struct {
	store     *store.Store
	dbFactory *dbfactory.DBFactory
}

// Run runs the PITR MySQL executor.
func (e *PITRMySQLExecutor) Run(ctx context.Context, planCheckRun *store.PlanCheckRunMessage) ([]*storepb.PlanCheckRunResult_Result, error) {
	instance, databaseName, err := func() (*store.InstanceMessage, string, error) {
		if planCheckRun.Config.GetPitrConfig().GetTargetInstanceId() != 0 {
			instanceUID := int(planCheckRun.Config.GetPitrConfig().GetTargetInstanceId())
			databaseName := planCheckRun.Config.GetPitrConfig().GetTargetDatabaseName()

			instance, err := e.store.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: &instanceUID})
			if err != nil {
				return nil, "", errors.Wrapf(err, "failed to get instance %v", instanceUID)
			}
			if instance == nil {
				return nil, "", errors.Errorf("instance not found %v", instanceUID)
			}
			return instance, databaseName, nil
		}

		databaseID := int(planCheckRun.Config.DatabaseId)
		database, err := e.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{UID: &databaseID})
		if err != nil {
			return nil, "", errors.Wrapf(err, "failed to get database ID %v", databaseID)
		}
		if database == nil {
			return nil, "", errors.Errorf("database not found %v", databaseID)
		}
		instance, err := e.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &database.InstanceID})
		if err != nil {
			return nil, "", errors.Wrapf(err, "failed to get instance %v", database.InstanceID)
		}
		if instance == nil {
			return nil, "", errors.Errorf("instance not found %v", database.InstanceID)
		}

		return instance, database.DatabaseName, nil
	}()
	if err != nil {
		return nil, err
	}

	driver, err := e.dbFactory.GetAdminDatabaseDriver(ctx, instance, nil /* database */)
	if err != nil {
		return nil, err
	}
	defer driver.Close(ctx)
	mysqlDriver, ok := driver.(*mysql.Driver)
	if !ok {
		return nil, errors.Errorf("Failed to cast driver to mysql.Driver")
	}

	if err := mysqlDriver.CheckServerVersionForPITR(ctx); err != nil {
		return convertErrorToResult(err), nil
	}

	if err := mysqlDriver.CheckEngineInnoDB(ctx, databaseName); err != nil {
		return convertErrorToResult(err), nil
	}

	if err := mysqlDriver.CheckBinlogRowFormat(ctx); err != nil {
		return convertErrorToResult(err), nil
	}

	return []*storepb.PlanCheckRunResult_Result{
		{
			Status:  storepb.PlanCheckRunResult_Result_SUCCESS,
			Code:    common.Ok.Int64(),
			Title:   "OK",
			Content: "Ready to do PITR",
		},
	}, nil
}

func convertErrorToResult(err error) []*storepb.PlanCheckRunResult_Result {
	return []*storepb.PlanCheckRunResult_Result{
		{
			Status:  storepb.PlanCheckRunResult_Result_ERROR,
			Code:    common.Internal.Int64(),
			Title:   "Error",
			Content: err.Error(),
		},
	}
}
