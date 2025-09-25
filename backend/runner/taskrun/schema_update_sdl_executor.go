package taskrun

import (
	"context"

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

	// Try to get the previous successful SDL text from task history
	previousUserSDLText, err := getPreviousSuccessfulSDLText(ctx, s, database.InstanceID, database.DatabaseName)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get previous SDL text for database %q", database.DatabaseName)
	}

	// Use GetSDLDiff with previous SDL text or initialization scenario support
	// - engine: the database engine
	// - currentSDLText: user's target SDL input
	// - previousUserSDLText: previous SDL text (empty triggers initialization scenario)
	// - currentSchema: current database schema (used as baseline in initialization)
	// - previousSchema: not needed for this implementation
	schemaDiff, err := schema.GetSDLDiff(pengine, sheetContent, previousUserSDLText, dbSchema, nil)
	if err != nil {
		return "", errors.Wrap(err, "failed to compute SDL schema diff")
	}

	// Filter out bbdataarchive schema changes for Postgres
	if instance.Metadata.GetEngine() == storepb.Engine_POSTGRES {
		schemaDiff = schema.FilterPostgresArchiveSchema(schemaDiff)
	}

	migrationSQL, err := schema.GenerateMigration(pengine, schemaDiff)
	if err != nil {
		return "", errors.Wrapf(err, "failed to generate migration SQL")
	}

	return migrationSQL, nil
}

// getPreviousSuccessfulSDLText retrieves the SDL text from the most recent
// successfully completed SDL changelog for the given database.
// Returns empty string if no previous successful SDL changelog is found.
func getPreviousSuccessfulSDLText(ctx context.Context, s *store.Store, instanceID string, databaseName string) (string, error) {
	// Find the most recent successful SDL changelog for this database
	// We only want MIGRATE_SDL type changelogs that are completed (DONE status)
	doneStatus := store.ChangelogStatusDone
	limit := 1 // We only need the most recent one

	changelogs, err := s.ListChangelogs(ctx, &store.FindChangelogMessage{
		InstanceID:   &instanceID,
		DatabaseName: &databaseName,
		TypeList:     []string{storepb.ChangelogPayload_MIGRATE_SDL.String()}, // Only SDL migrations
		Status:       &doneStatus,
		Limit:        &limit, // Get only the most recent one
		ShowFull:     false,  // We only need the sheet reference from payload
	})
	if err != nil {
		return "", errors.Wrapf(err, "failed to list previous SDL changelogs for database %s", databaseName)
	}

	if len(changelogs) == 0 {
		// No previous SDL changelogs found - this is fine, we'll use initialization scenario
		return "", nil
	}

	mostRecentChangelog := changelogs[0] // ListChangelogs should return them in descending order by creation time

	// Extract the sheet ID from the changelog payload
	if mostRecentChangelog.Payload == nil {
		return "", nil
	}

	sheetResource := mostRecentChangelog.Payload.Sheet
	if sheetResource == "" {
		return "", nil
	}

	// Extract sheet ID from resource string format: "projects/{project}/sheets/{sheet}"
	_, sheetID, err := common.GetProjectResourceIDSheetUID(sheetResource)
	if err != nil {
		return "", errors.Wrapf(err, "failed to extract sheet ID from resource %s", sheetResource)
	}

	// Get the sheet content (original SDL text)
	previousSDLText, err := s.GetSheetStatementByID(ctx, sheetID)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get sheet statement for previous SDL changelog sheet ID %d", sheetID)
	}

	return previousSDLText, nil
}
