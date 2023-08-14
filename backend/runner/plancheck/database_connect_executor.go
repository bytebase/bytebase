package plancheck

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var _ Executor = (*DatabaseConnectExecutor)(nil)

// NewDatabaseConnectExecutor creates a task check database connect executor.
func NewDatabaseConnectExecutor(store *store.Store, dbFactory *dbfactory.DBFactory) Executor {
	return &DatabaseConnectExecutor{
		store:     store,
		dbFactory: dbFactory,
	}
}

// DatabaseConnectExecutor checks if the database connection is valid.
type DatabaseConnectExecutor struct {
	store     *store.Store
	dbFactory *dbfactory.DBFactory
}

// Run runs the executor.
func (e *DatabaseConnectExecutor) Run(ctx context.Context, planCheckRun *store.PlanCheckRunMessage) ([]*storepb.PlanCheckRunResult_Result, error) {
	databaseID := int(planCheckRun.Config.DatabaseId)
	database, err := e.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{UID: &databaseID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database ID %v", databaseID)
	}
	if database == nil {
		return nil, errors.Errorf("database not found %v", databaseID)
	}
	instance, err := e.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &database.InstanceID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get instance %v", database.InstanceID)
	}
	if instance == nil {
		return nil, errors.Errorf("instance not found %v", database.InstanceID)
	}

	driver, err := e.dbFactory.GetAdminDatabaseDriver(ctx, instance, database)
	if err == nil {
		err = driver.Ping(ctx)
	}
	if err != nil {
		return []*storepb.PlanCheckRunResult_Result{
			{
				Status:  storepb.PlanCheckRunResult_Result_ERROR,
				Code:    common.DbConnectionFailure.Int64(),
				Title:   fmt.Sprintf("Failed to connect %q", database.DatabaseName),
				Content: err.Error(),
			},
		}, nil
	}
	defer driver.Close(ctx)

	return []*storepb.PlanCheckRunResult_Result{
		{
			Status:  storepb.PlanCheckRunResult_Result_SUCCESS,
			Code:    common.Ok.Int64(),
			Title:   "OK",
			Content: fmt.Sprintf("Successfully connected %q", database.DatabaseName),
		},
	}, nil
}
