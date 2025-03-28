package taskrun

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/state"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/runner/schemasync"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
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

var cannotCreateDatabase = map[storepb.Engine]bool{
	storepb.Engine_REDIS:            true,
	storepb.Engine_ORACLE:           true,
	storepb.Engine_DM:               true,
	storepb.Engine_OCEANBASE_ORACLE: true,
}

// RunOnce will run the database create task executor once.
func (exec *DatabaseCreateExecutor) RunOnce(ctx context.Context, driverCtx context.Context, task *store.TaskMessage, _ int) (terminated bool, result *storepb.TaskRunResult, err error) {
	sheetID := int(task.Payload.GetSheetId())
	statement, err := exec.store.GetSheetStatementByID(ctx, sheetID)
	if err != nil {
		return true, nil, errors.Wrapf(err, "failed to get sheet statement of sheet: %d", sheetID)
	}
	sheet, err := exec.store.GetSheet(ctx, &store.FindSheetMessage{UID: &sheetID})
	if err != nil {
		return true, nil, errors.Wrapf(err, "failed to get sheet: %d", sheetID)
	}
	if sheet == nil {
		return true, nil, errors.Errorf("sheet not found: %d", sheetID)
	}

	statement = strings.TrimSpace(statement)
	if statement == "" {
		return true, nil, errors.Errorf("empty create database statement")
	}

	instance, err := exec.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &task.InstanceID})
	if err != nil {
		return true, nil, err
	}

	if cannotCreateDatabase[instance.Metadata.GetEngine()] {
		return true, nil, errors.Errorf("Creating database is not supported")
	}

	pipeline, err := exec.store.GetPipelineV2ByID(ctx, task.PipelineID)
	if err != nil {
		return true, nil, errors.Wrapf(err, "failed to get pipeline")
	}
	if pipeline == nil {
		return true, nil, errors.Errorf("pipeline %v not found", task.PipelineID)
	}
	project, err := exec.store.GetProjectV2(ctx, &store.FindProjectMessage{ResourceID: &pipeline.ProjectID})
	if err != nil {
		return true, nil, errors.Errorf("failed to find project %s", pipeline.ProjectID)
	}
	if project == nil {
		return true, nil, errors.Errorf("project %s not found", pipeline.ProjectID)
	}

	// Create database.
	slog.Debug("Start creating database...",
		slog.String("instance", instance.Metadata.GetTitle()),
		slog.String("database", task.Payload.GetDatabaseName()),
		slog.String("statement", statement),
	)

	database, err := exec.store.UpsertDatabase(ctx, &store.DatabaseMessage{
		ProjectID:     pipeline.ProjectID,
		InstanceID:    instance.ResourceID,
		DatabaseName:  task.Payload.GetDatabaseName(),
		EnvironmentID: task.Payload.GetEnvironmentId(),
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
		defaultDBDriver, err = exec.dbFactory.GetAdminDatabaseDriver(ctx, instance, database, db.ConnectionContext{
			OperationalComponent: "create-database",
		})
		if err != nil {
			return true, nil, err
		}
	default:
		defaultDBDriver, err = exec.dbFactory.GetAdminDatabaseDriver(ctx, instance, nil /* database */, db.ConnectionContext{
			OperationalComponent: "create-database",
		})
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
