package plancheck

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
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
	if target := planCheckRun.Config.GetDatabaseTarget(); target != nil {
		return e.runForDatabaseTarget(ctx, target)
	}
	if target := planCheckRun.Config.GetDatabaseGroupTarget(); target != nil {
		return e.runForDatabaseGroupTarget(ctx, target)
	}
	return nil, errors.New("plan check run target is required")
}

func (e *DatabaseConnectExecutor) runForDatabaseTarget(ctx context.Context, target *storepb.PlanCheckRunConfig_DatabaseTarget) ([]*storepb.PlanCheckRunResult_Result, error) {
	instanceUID := int(target.InstanceUid)
	instance, err := e.store.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: &instanceUID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get instance UID %v", instanceUID)
	}
	if instance == nil {
		return nil, errors.Errorf("instance not found UID %v", instanceUID)
	}

	database, err := e.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{InstanceID: &instance.ResourceID, DatabaseName: &target.DatabaseName})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database %q", target.DatabaseName)
	}
	if database == nil {
		return nil, errors.Errorf("database not found %q", target.DatabaseName)
	}

	return []*storepb.PlanCheckRunResult_Result{e.checkDatabaseConnection(ctx, instance, database)}, nil
}

func (e *DatabaseConnectExecutor) runForDatabaseGroupTarget(ctx context.Context, target *storepb.PlanCheckRunConfig_DatabaseGroupTarget) ([]*storepb.PlanCheckRunResult_Result, error) {
	databaseGroup, err := e.store.GetDatabaseGroup(ctx, &store.FindDatabaseGroupMessage{
		UID: &target.DatabaseGroupUid,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database group %d", target.DatabaseGroupUid)
	}
	if databaseGroup == nil {
		return nil, errors.Errorf("database group not found %d", target.DatabaseGroupUid)
	}
	project, err := e.store.GetProjectV2(ctx, &store.FindProjectMessage{
		UID: &databaseGroup.ProjectUID,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get project %d", databaseGroup.ProjectUID)
	}
	if project == nil {
		return nil, errors.Errorf("project not found %d", databaseGroup.ProjectUID)
	}

	allDatabases, err := e.store.ListDatabases(ctx, &store.FindDatabaseMessage{ProjectID: &project.ResourceID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list databases for project %q", project.ResourceID)
	}

	matchedDatabases, _, err := utils.GetMatchedAndUnmatchedDatabasesInDatabaseGroup(ctx, databaseGroup, allDatabases)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get matched and unmatched databases in database group %q", databaseGroup.ResourceID)
	}
	if len(matchedDatabases) == 0 {
		return nil, errors.Errorf("no matched databases found in database group %q", databaseGroup.ResourceID)
	}

	instances := map[string]*store.InstanceMessage{}

	for _, db := range matchedDatabases {
		if _, ok := instances[db.InstanceID]; ok {
			continue
		}
		instance, err := e.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &db.InstanceID})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get instance %q", db.InstanceID)
		}
		if instance == nil {
			return nil, errors.Errorf("instance not found %q", db.InstanceID)
		}
		instances[db.InstanceID] = instance
	}

	var results []*storepb.PlanCheckRunResult_Result
	for _, db := range matchedDatabases {
		results = append(results, e.checkDatabaseConnection(ctx, instances[db.InstanceID], db))
	}

	return results, nil
}

func (e *DatabaseConnectExecutor) checkDatabaseConnection(ctx context.Context, instance *store.InstanceMessage, database *store.DatabaseMessage) *storepb.PlanCheckRunResult_Result {
	err := func() error {
		driver, err := e.dbFactory.GetAdminDatabaseDriver(ctx, instance, database)
		if err != nil {
			return err
		}
		defer driver.Close(ctx)
		return driver.Ping(ctx)
	}()
	if err != nil {
		return &storepb.PlanCheckRunResult_Result{
			Status:  storepb.PlanCheckRunResult_Result_ERROR,
			Code:    common.DbConnectionFailure.Int64(),
			Title:   fmt.Sprintf("Failed to connect %q", database.DatabaseName),
			Content: err.Error(),
		}
	}
	return &storepb.PlanCheckRunResult_Result{
		Status:  storepb.PlanCheckRunResult_Result_SUCCESS,
		Code:    common.Ok.Int64(),
		Title:   "OK",
		Content: fmt.Sprintf("Successfully connected %q", database.DatabaseName),
	}
}
