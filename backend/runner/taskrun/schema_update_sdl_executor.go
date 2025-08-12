package taskrun

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/state"
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/runner/schemasync"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/store/model"
)

// NewSchemaDeclareExecutor creates a schema declare (SDL) task executor.
func NewSchemaDeclareExecutor(store *store.Store, dbFactory *dbfactory.DBFactory, license *enterprise.LicenseService, stateCfg *state.State, schemaSyncer *schemasync.Syncer, profile *config.Profile) Executor {
	return &SchemaDeclareExecutor{
		store:        store,
		dbFactory:    dbFactory,
		license:      license,
		stateCfg:     stateCfg,
		schemaSyncer: schemaSyncer,
		profile:      profile,
	}
}

// SchemaDeclareExecutor is the schema declare (SDL) task executor.
type SchemaDeclareExecutor struct {
	store        *store.Store
	dbFactory    *dbfactory.DBFactory
	license      *enterprise.LicenseService
	stateCfg     *state.State
	schemaSyncer *schemasync.Syncer
	profile      *config.Profile
}

// RunOnce will run the schema declare (SDL) task executor once.
func (exec *SchemaDeclareExecutor) RunOnce(ctx context.Context, driverCtx context.Context, task *store.TaskMessage, taskRunUID int) (bool, *storepb.TaskRunResult, error) {
	instance, err := exec.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &task.InstanceID})
	if err != nil {
		return true, nil, err
	}
	database, err := exec.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{InstanceID: &task.InstanceID, DatabaseName: task.DatabaseName})
	if err != nil {
		return true, nil, err
	}

	sheetID := int(task.Payload.GetSheetId())
	sheetContent, err := exec.store.GetSheetStatementByID(ctx, sheetID)
	if err != nil {
		return true, nil, err
	}

	execFunc := func(ctx context.Context, execStatement string, driver db.Driver, opts db.ExecuteOptions) error {
		// TODO(p0ny): log diff and migration SQL
		migrationSQL, err := diff(ctx, exec.store, instance, database, execStatement)
		if err != nil {
			return errors.Wrapf(err, "failed to diff database schema")
		}
		if _, err := driver.Execute(ctx, migrationSQL, opts); err != nil {
			return err
		}
		return nil
	}

	return runMigrationWithFunc(ctx, driverCtx, exec.store, exec.dbFactory, exec.stateCfg, exec.schemaSyncer, exec.profile, task, taskRunUID, sheetContent, task.Payload.GetSchemaVersion(), &sheetID, execFunc)
}

func diff(ctx context.Context, s *store.Store, instance *store.InstanceMessage, database *store.DatabaseMessage, sheetContent string) (string, error) {
	pengine, err := common.ConvertToParserEngine(instance.Metadata.GetEngine())
	if err != nil {
		return "", errors.Wrapf(err, "failed to convert %q to parser engine", instance.Metadata.GetEngine())
	}

	dbSchema, err := s.GetDBSchema(ctx, database.InstanceID, database.DatabaseName)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get database schema for database %q", database.DatabaseName)
	}
	if dbSchema == nil {
		return "", errors.Errorf("database schema %q not found", database.DatabaseName)
	}

	targetSchemaMetadata, err := schema.GetDatabaseMetadata(pengine, sheetContent)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get database metadata for database %q", database.DatabaseName)
	}

	targetSchema := model.NewDatabaseSchema(
		targetSchemaMetadata,
		[]byte(sheetContent),
		&storepb.DatabaseConfig{},
		pengine,
		store.IsObjectCaseSensitive(instance),
	)

	schemaDiff, err := schema.GetDatabaseSchemaDiff(pengine, dbSchema, targetSchema)
	if err != nil {
		return "", connect.NewError(connect.CodeInternal, errors.Errorf("failed to compute schema diff, error: %v", err))
	}

	// Filter out bbdataarchive schema changes for Postgres
	if instance.Metadata.GetEngine() == storepb.Engine_POSTGRES {
		schemaDiff = schema.FilterPostgresArchiveSchema(schemaDiff)
	}

	migrationSQL, err := schema.GenerateMigration(pengine, schemaDiff)
	if err != nil {
		return "", connect.NewError(connect.CodeInternal, errors.Errorf("failed to generate migration SQL, error: %v", err))
	}

	return migrationSQL, nil
}
