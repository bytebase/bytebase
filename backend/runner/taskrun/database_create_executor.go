package taskrun

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/state"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/runner/schemasync"
	"github.com/bytebase/bytebase/backend/store"
)

// NewDatabaseCreateExecutor creates a database create task executor.
func NewDatabaseCreateExecutor(store *store.Store, dbFactory *dbfactory.DBFactory, schemaSyncer *schemasync.Syncer, stateCfg *state.State, profile *config.Profile) Executor {
	return &DatabaseCreateExecutor{
		store:        store,
		dbFactory:    dbFactory,
		schemaSyncer: schemaSyncer,
		stateCfg:     stateCfg,
		profile:      profile,
	}
}

// DatabaseCreateExecutor is the database create task executor.
type DatabaseCreateExecutor struct {
	store        *store.Store
	dbFactory    *dbfactory.DBFactory
	schemaSyncer *schemasync.Syncer
	stateCfg     *state.State
	profile      *config.Profile
}

// RunOnce will run the database create task executor once.
func (exec *DatabaseCreateExecutor) RunOnce(ctx context.Context, driverCtx context.Context, task *store.TaskMessage, _ int) (terminated bool, result *storepb.TaskRunResult, err error) {
	sheet, err := exec.store.GetSheetFull(ctx, task.Payload.GetSheetSha256())
	if err != nil {
		return true, nil, errors.Wrapf(err, "failed to get sheet: %s", task.Payload.GetSheetSha256())
	}
	if sheet == nil {
		return true, nil, errors.Errorf("sheet not found: %s", task.Payload.GetSheetSha256())
	}
	statement := sheet.Statement

	statement = strings.TrimSpace(statement)
	if statement == "" {
		return true, nil, errors.Errorf("empty create database statement")
	}

	instance, err := exec.store.GetInstance(ctx, &store.FindInstanceMessage{ResourceID: &task.InstanceID})
	if err != nil {
		return true, nil, err
	}

	if !common.EngineSupportCreateDatabase(instance.Metadata.GetEngine()) {
		return true, nil, errors.Errorf("creating database is not supported for engine %v", instance.Metadata.GetEngine().String())
	}

	plan, err := exec.store.GetPlan(ctx, &store.FindPlanMessage{UID: &task.PlanID})
	if err != nil {
		return true, nil, errors.Wrapf(err, "failed to get plan %v", task.PlanID)
	}
	if plan == nil {
		return true, nil, errors.Errorf("plan %v not found", task.PlanID)
	}

	// Create database.
	slog.Debug("Start creating database...",
		slog.String("instance", instance.Metadata.GetTitle()),
		slog.String("database", task.Payload.GetDatabaseName()),
		slog.String("statement", statement),
	)

	envID := task.Payload.GetEnvironmentId()
	var environmentID *string
	if envID != "" {
		environmentID = &envID
	}
	database, err := exec.store.UpsertDatabase(ctx, &store.DatabaseMessage{
		ProjectID:     plan.ProjectID,
		InstanceID:    instance.ResourceID,
		DatabaseName:  task.Payload.GetDatabaseName(),
		EnvironmentID: environmentID,
		Metadata:      &storepb.DatabaseMetadata{},
	})
	if err != nil {
		return true, nil, err
	}

	var defaultDBDriver db.Driver
	switch instance.Metadata.GetEngine() {
	case storepb.Engine_MONGODB:
		// For MongoDB, it allows us to connect to the non-existing database. So we pass the database name to driver to let us connect to the specific database.
		// And run the create collection statement later.
		// NOTE: we have to hack the database message.
		defaultDBDriver, err = exec.dbFactory.GetAdminDatabaseDriver(ctx, instance, database, db.ConnectionContext{})
		if err != nil {
			return true, nil, err
		}
	default:
		defaultDBDriver, err = exec.dbFactory.GetAdminDatabaseDriver(ctx, instance, nil /* database */, db.ConnectionContext{})
		if err != nil {
			return true, nil, err
		}
	}
	defer defaultDBDriver.Close(ctx)
	if _, err := defaultDBDriver.Execute(driverCtx, statement, db.ExecuteOptions{CreateDatabase: true}); err != nil {
		return true, nil, err
	}

	if err := exec.schemaSyncer.SyncDatabaseSchema(ctx, database); err != nil {
		slog.Error("failed to sync database schema",
			slog.String("instanceName", instance.ResourceID),
			slog.String("databaseName", database.DatabaseName),
			log.BBError(err),
		)
	}

	return true, &storepb.TaskRunResult{
		Detail: fmt.Sprintf("Created database %q", task.Payload.GetDatabaseName()),
	}, nil
}
