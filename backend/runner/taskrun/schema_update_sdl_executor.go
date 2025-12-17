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
	instance, err := exec.store.GetInstance(ctx, &store.FindInstanceMessage{ResourceID: &task.InstanceID})
	if err != nil {
		return true, nil, err
	}
	database, err := exec.store.GetDatabase(ctx, &store.FindDatabaseMessage{InstanceID: &task.InstanceID, DatabaseName: task.DatabaseName})
	if err != nil {
		return true, nil, err
	}

	sheetID := int(task.Payload.GetSheetId())
	sheet, err := exec.store.GetSheetFull(ctx, sheetID)
	if err != nil {
		return true, nil, err
	}
	sheetContent := sheet.Statement

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

	dbMetadata, err := s.GetDBSchema(ctx, &store.FindDBSchemaMessage{
		InstanceID:   database.InstanceID,
		DatabaseName: database.DatabaseName,
	})
	if err != nil {
		return "", errors.Wrapf(err, "failed to get database schema for database %q", database.DatabaseName)
	}
	if dbMetadata == nil {
		return "", errors.Errorf("database schema %q not found", database.DatabaseName)
	}

	// Try to get the previous successful SDL text and schema from task history
	previousUserSDLText, previousSchema, err := getPreviousSuccessfulSDLAndSchema(ctx, s, database.InstanceID, database.DatabaseName)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get previous SDL text and schema for database %q", database.DatabaseName)
	}

	// Use GetSDLDiff with previous SDL text and schema
	// - engine: the database engine
	// - currentSDLText: user's target SDL input
	// - previousUserSDLText: previous SDL text (empty triggers initialization scenario)
	// - currentSchema: current database schema (used as baseline in initialization)
	// - previousSchema: previous database schema from changelog
	schemaDiff, err := schema.GetSDLDiff(pengine, sheetContent, previousUserSDLText, dbMetadata, previousSchema)
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
// getPreviousSuccessfulSDLAndSchema gets both the SDL text and database schema from the most recent successful SDL changelog
func getPreviousSuccessfulSDLAndSchema(ctx context.Context, s *store.Store, instanceID string, databaseName string) (string, *model.DatabaseMetadata, error) {
	// Find the most recent successful SDL changelog for this database
	// We only want MIGRATE_SDL type changelogs that are completed (DONE status)
	doneStatus := store.ChangelogStatusDone
	limit := 1 // We only need the most recent one

	changelogs, err := s.ListChangelogs(ctx, &store.FindChangelogMessage{
		InstanceID:   &instanceID,
		DatabaseName: &databaseName,
		TypeList:     []string{storepb.ChangelogPayload_SDL.String()}, // Only SDL migrations
		Status:       &doneStatus,
		Limit:        &limit, // Get only the most recent one
		ShowFull:     false,  // We only need the PrevSyncHistoryUID and sheet reference
	})
	if err != nil {
		return "", nil, errors.Wrapf(err, "failed to list previous SDL changelogs for database %s", databaseName)
	}

	if len(changelogs) == 0 {
		// No previous SDL changelogs found - this is fine, we'll use initialization scenario
		return "", nil, nil
	}

	mostRecentChangelog := changelogs[0] // ListChangelogs should return them in descending order by creation time

	// Extract the sheet ID from the changelog payload
	var previousUserSDLText string
	if mostRecentChangelog.Payload != nil && mostRecentChangelog.Payload.Sheet != "" {
		sheetResource := mostRecentChangelog.Payload.Sheet

		// Extract sheet ID from resource string format: "projects/{project}/sheets/{sheet}"
		_, sheetID, err := common.GetProjectResourceIDSheetUID(sheetResource)
		if err != nil {
			return "", nil, errors.Wrapf(err, "failed to extract sheet ID from resource %s", sheetResource)
		}

		// Get the sheet content (original SDL text)
		sheet, err := s.GetSheetFull(ctx, sheetID)
		if err != nil {
			return "", nil, errors.Wrapf(err, "failed to get sheet statement for previous SDL changelog sheet ID %d", sheetID)
		}
		previousUserSDLText = sheet.Statement
	}

	// Get the previous schema from sync history
	// Use SyncHistoryUID (after applying the SDL) instead of PrevSyncHistoryUID (before applying)
	// This represents the database schema state after the previous SDL was successfully applied
	var previousSchema *model.DatabaseMetadata
	if mostRecentChangelog.SyncHistoryUID != nil {
		// Get the sync history record to obtain the schema metadata
		syncHistory, err := s.GetSyncHistoryByUID(ctx, *mostRecentChangelog.SyncHistoryUID)
		if err != nil {
			return "", nil, errors.Wrapf(err, "failed to get sync history by UID %d", *mostRecentChangelog.SyncHistoryUID)
		}

		if syncHistory != nil && syncHistory.Metadata != nil {
			// Get instance to determine engine and case sensitivity
			instance, err := s.GetInstance(ctx, &store.FindInstanceMessage{ResourceID: &instanceID})
			if err != nil {
				return "", nil, errors.Wrapf(err, "failed to get instance %s", instanceID)
			}
			if instance == nil {
				return "", nil, errors.Errorf("instance %s not found", instanceID)
			}

			// Create a DatabaseSchema wrapper using the metadata from sync history
			previousSchema = model.NewDatabaseMetadata(
				syncHistory.Metadata,
				[]byte(syncHistory.Schema), // Use the schema content from sync history
				&storepb.DatabaseConfig{},  // Empty config
				instance.Metadata.GetEngine(),
				store.IsObjectCaseSensitive(instance),
			)
		}
	}

	return previousUserSDLText, previousSchema, nil
}
