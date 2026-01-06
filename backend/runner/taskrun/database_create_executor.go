package taskrun

import (
	"context"
	"log/slog"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/runner/schemasync"
	"github.com/bytebase/bytebase/backend/store"
)

// NewDatabaseCreateExecutor creates a database create task executor.
func NewDatabaseCreateExecutor(store *store.Store, dbFactory *dbfactory.DBFactory, schemaSyncer *schemasync.Syncer) Executor {
	return &DatabaseCreateExecutor{
		store:        store,
		dbFactory:    dbFactory,
		schemaSyncer: schemaSyncer,
	}
}

// DatabaseCreateExecutor is the database create task executor.
type DatabaseCreateExecutor struct {
	store        *store.Store
	dbFactory    *dbfactory.DBFactory
	schemaSyncer *schemasync.Syncer
}

// RunOnce will run the database create task executor once.
func (exec *DatabaseCreateExecutor) RunOnce(ctx context.Context, driverCtx context.Context, task *store.TaskMessage, _ int) (*storepb.TaskRunResult, error) {
	sheet, err := exec.store.GetSheetFull(ctx, task.Payload.GetSheetSha256())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get sheet: %s", task.Payload.GetSheetSha256())
	}
	if sheet == nil {
		return nil, errors.Errorf("sheet not found: %s", task.Payload.GetSheetSha256())
	}
	statement := sheet.Statement

	statement = strings.TrimSpace(statement)
	if statement == "" {
		return nil, errors.Errorf("empty create database statement")
	}

	instance, err := exec.store.GetInstance(ctx, &store.FindInstanceMessage{ResourceID: &task.InstanceID})
	if err != nil {
		return nil, err
	}

	if !common.EngineSupportCreateDatabase(instance.Metadata.GetEngine()) {
		return nil, errors.Errorf("creating database is not supported for engine %v", instance.Metadata.GetEngine().String())
	}

	plan, err := exec.store.GetPlan(ctx, &store.FindPlanMessage{UID: &task.PlanID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get plan %v", task.PlanID)
	}
	if plan == nil {
		return nil, errors.Errorf("plan %v not found", task.PlanID)
	}

	// For create database plans, there is always exactly one spec
	if len(plan.Config.Specs) == 0 {
		return nil, errors.Errorf("plan has no specs")
	}
	createConfig := plan.Config.Specs[0].GetCreateDatabaseConfig()
	if createConfig == nil {
		return nil, errors.Errorf("spec does not contain create database config")
	}

	// Create database.
	slog.Debug("Start creating database...",
		slog.String("instance", instance.Metadata.GetTitle()),
		slog.String("database", createConfig.Database),
		slog.String("statement", statement),
	)

	var environmentID *string
	if createConfig.Environment != "" {
		envID, err := common.GetEnvironmentID(createConfig.Environment)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse environment %s", createConfig.Environment)
		}
		environmentID = &envID
	}
	database, err := exec.store.UpsertDatabase(ctx, &store.DatabaseMessage{
		ProjectID:     plan.ProjectID,
		InstanceID:    instance.ResourceID,
		DatabaseName:  createConfig.Database,
		EnvironmentID: environmentID,
		Metadata:      &storepb.DatabaseMetadata{},
	})
	if err != nil {
		return nil, err
	}

	var defaultDBDriver db.Driver
	switch instance.Metadata.GetEngine() {
	case storepb.Engine_MONGODB:
		// For MongoDB, it allows us to connect to the non-existing database. So we pass the database name to driver to let us connect to the specific database.
		// And run the create collection statement later.
		// NOTE: we have to hack the database message.
		defaultDBDriver, err = exec.dbFactory.GetAdminDatabaseDriver(ctx, instance, database, db.ConnectionContext{})
		if err != nil {
			return nil, err
		}
	default:
		defaultDBDriver, err = exec.dbFactory.GetAdminDatabaseDriver(ctx, instance, nil /* database */, db.ConnectionContext{})
		if err != nil {
			return nil, err
		}
	}
	defer defaultDBDriver.Close(ctx)
	if _, err := defaultDBDriver.Execute(driverCtx, statement, db.ExecuteOptions{CreateDatabase: true}); err != nil {
		return nil, err
	}

	if err := exec.schemaSyncer.SyncDatabaseSchema(ctx, database); err != nil {
		slog.Error("failed to sync database schema",
			slog.String("instanceName", instance.ResourceID),
			slog.String("databaseName", database.DatabaseName),
			log.BBError(err),
		)
	}

	return &storepb.TaskRunResult{}, nil
}
