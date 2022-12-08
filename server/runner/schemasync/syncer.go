// Package schemasync is a runner that synchronize database schemas.
package schemasync

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/golang/protobuf/jsonpb"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/common/log"
	"github.com/bytebase/bytebase/plugin/db"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	"github.com/bytebase/bytebase/server/component/config"
	"github.com/bytebase/bytebase/server/component/dbfactory"
	"github.com/bytebase/bytebase/server/component/state"
	"github.com/bytebase/bytebase/server/utils"
	"github.com/bytebase/bytebase/store"
)

const (
	schemaSyncInterval = 30 * time.Minute
)

// NewSyncer creates a schema syncer.
func NewSyncer(store *store.Store, dbFactory *dbfactory.DBFactory, stateCfg *state.State, profile config.Profile) *Syncer {
	return &Syncer{
		store:     store,
		dbFactory: dbFactory,
		stateCfg:  stateCfg,
		profile:   profile,
	}
}

// Syncer is the schema syncer.
type Syncer struct {
	store     *store.Store
	dbFactory *dbfactory.DBFactory
	stateCfg  *state.State
	profile   config.Profile
}

// Run will run the schema syncer once.
func (s *Syncer) Run(ctx context.Context, wg *sync.WaitGroup) {
	ticker := time.NewTicker(schemaSyncInterval)
	defer ticker.Stop()
	defer wg.Done()
	log.Debug(fmt.Sprintf("Schema syncer started and will run every %v", schemaSyncInterval))
	for {
		select {
		case <-ticker.C:
			s.syncAllInstances(ctx)
			// Sync all databases for all instances.
			s.syncAllDatabases(ctx, nil /* instanceID */)
		case instance := <-s.stateCfg.InstanceDatabaseSyncChan:
			// Sync all databases for instance.
			s.syncAllDatabases(ctx, &instance.ID)
		case <-ctx.Done(): // if cancel() execute
			return
		}
	}
}

func (s *Syncer) syncAllInstances(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			err, ok := r.(error)
			if !ok {
				err = errors.Errorf("%v", r)
			}
			log.Error("Instance syncer PANIC RECOVER", zap.Error(err), zap.Stack("panic-stack"))
		}
	}()

	rowStatus := api.Normal
	instanceFind := &api.InstanceFind{
		RowStatus: &rowStatus,
	}
	instanceList, err := s.store.FindInstance(ctx, instanceFind)
	if err != nil {
		log.Error("Failed to retrieve instances", zap.Error(err))
		return
	}

	var instanceWG sync.WaitGroup
	for _, instance := range instanceList {
		instanceWG.Add(1)
		go func(instance *api.Instance) {
			defer instanceWG.Done()
			log.Debug("Sync instance schema", zap.String("instance", instance.Name))
			if _, err := s.SyncInstance(ctx, instance); err != nil {
				log.Debug("Failed to sync instance",
					zap.Int("id", instance.ID),
					zap.String("name", instance.Name),
					zap.String("error", err.Error()))
				return
			}
		}(instance)
	}
	instanceWG.Wait()
}

func (s *Syncer) syncAllDatabases(ctx context.Context, instanceID *int) {
	defer func() {
		if r := recover(); r != nil {
			err, ok := r.(error)
			if !ok {
				err = errors.Errorf("%v", r)
			}
			log.Error("Database syncer PANIC RECOVER", zap.Error(err), zap.Stack("panic-stack"))
		}
	}()

	okSyncStatus := api.OK
	databaseList, err := s.store.FindDatabase(ctx, &api.DatabaseFind{
		InstanceID: instanceID,
		SyncStatus: &okSyncStatus,
	})
	if err != nil {
		log.Debug("Failed to find databases to sync",
			zap.String("error", err.Error()))
		return
	}

	instanceMap := make(map[int][]*api.Database)
	for _, database := range databaseList {
		instanceMap[database.InstanceID] = append(instanceMap[database.InstanceID], database)
	}

	var instanceWG sync.WaitGroup
	for _, databaseList := range instanceMap {
		instanceWG.Add(1)
		go func(databaseList []*api.Database) {
			defer instanceWG.Done()

			if len(databaseList) == 0 {
				return
			}
			instance := databaseList[0].Instance
			for _, database := range databaseList {
				log.Debug("Sync database schema",
					zap.String("instance", instance.Name),
					zap.String("database", database.Name),
					zap.Int64("lastSuccessfulSyncTs", database.LastSuccessfulSyncTs),
				)
				// If we fail to sync a particular database due to permission issue, we will continue to sync the rest of the databases.
				if err := s.SyncDatabaseSchema(ctx, instance, database.Name); err != nil {
					log.Debug("Failed to sync database schema",
						zap.Int("instanceID", instance.ID),
						zap.String("instanceName", instance.Name),
						zap.String("databaseName", database.Name),
						zap.Error(err))
				}
			}
		}(databaseList)
	}
	instanceWG.Wait()
}

// SyncInstance syncs the schema for all databases in an instance.
func (s *Syncer) SyncInstance(ctx context.Context, instance *api.Instance) ([]string, error) {
	driver, err := s.dbFactory.GetAdminDatabaseDriver(ctx, instance, "")
	if err != nil {
		return nil, err
	}
	defer driver.Close(ctx)

	return s.syncInstanceSchema(ctx, instance, driver)
}

// syncInstanceSchema syncs the instance and all database metadata first without diving into the deep structure of each database.
func (s *Syncer) syncInstanceSchema(ctx context.Context, instance *api.Instance, driver db.Driver) ([]string, error) {
	// Sync instance metadata.
	instanceMeta, err := driver.SyncInstance(ctx)
	if err != nil {
		return nil, err
	}

	// Underlying version may change due to upgrade, however it's a rare event, so we only update if it actually differs
	// to avoid changing the updated_ts.
	if instanceMeta.Version != instance.EngineVersion {
		_, err := s.store.PatchInstance(ctx, &api.InstancePatch{
			ID:            instance.ID,
			UpdaterID:     api.SystemBotID,
			EngineVersion: &instanceMeta.Version,
		})
		if err != nil {
			return nil, err
		}
		instance.EngineVersion = instanceMeta.Version
	}

	instanceUserList, err := s.store.FindInstanceUserByInstanceID(ctx, instance.ID)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch user list for instance: %v", instance.ID)).SetInternal(err)
	}

	// Upsert user found in the instance
	for _, user := range instanceMeta.UserList {
		userUpsert := &api.InstanceUserUpsert{
			CreatorID:  api.SystemBotID,
			InstanceID: instance.ID,
			Name:       user.Name,
			Grant:      user.Grant,
		}
		_, err := s.store.UpsertInstanceUser(ctx, userUpsert)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to sync user for instance: %s. Failed to upsert user", instance.Name)
		}
	}

	// Delete user no longer found in the instance
	for _, user := range instanceUserList {
		found := false
		for _, dbUser := range instanceMeta.UserList {
			if user.Name == dbUser.Name {
				found = true
				break
			}
		}

		if !found {
			userDelete := &api.InstanceUserDelete{
				ID: user.ID,
			}
			err := s.store.DeleteInstanceUser(ctx, userDelete)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to sync user for instance: %s. Failed to delete user: %s", instance.Name, user.Name)
			}
		}
	}

	// Compare the stored db info with the just synced db schema.
	// Case 1: If item appears in both stored db info and the synced db metadata, then it's a no-op. We rely on syncDatabaseSchema() later to sync its details.
	// Case 2: If item only appears in the synced schema and not in the stored db, then we CREATE the database record in the stored db.
	// Case 3: Conversely, if item only appears in the stored db, but not in the synced schema, then we MARK the record as NOT_FOUND.
	//   	   We don't delete the entry because:
	//   	   1. This entry has already been associated with other entities, we can't simply delete it.
	//   	   2. The deletion in the schema might be a mistake, so it's better to surface as NOT_FOUND to let user review it.
	databaseFind := &api.DatabaseFind{
		InstanceID: &instance.ID,
	}
	dbList, err := s.store.FindDatabase(ctx, databaseFind)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to sync database for instance: %s. Failed to find database list", instance.Name)
	}
	for _, databaseMetadata := range instanceMeta.DatabaseList {
		databaseName := databaseMetadata.Name

		var matchedDb *api.Database
		for _, db := range dbList {
			if db.Name == databaseName {
				matchedDb = db
				break
			}
		}
		if matchedDb != nil {
			// Case 1, appear in both the Bytebase metadata and the synced database metadata.
			// We rely on syncDatabaseSchema() to sync the database details.
			continue
		}
		// Case 2, only appear in the synced db schema.
		databaseCreate := &api.DatabaseCreate{
			CreatorID:            api.SystemBotID,
			ProjectID:            api.DefaultProjectID,
			InstanceID:           instance.ID,
			EnvironmentID:        instance.EnvironmentID,
			Name:                 databaseName,
			CharacterSet:         databaseMetadata.CharacterSet,
			Collation:            databaseMetadata.Collation,
			LastSuccessfulSyncTs: 0,
		}
		if _, err := s.store.CreateDatabase(ctx, databaseCreate); err != nil {
			if common.ErrorCode(err) == common.Conflict {
				return nil, errors.Errorf("failed to sync database for instance: %s. Database name already exists: %s", instance.Name, databaseCreate.Name)
			}
			return nil, errors.Wrapf(err, "failed to sync database for instance: %s. Failed to import new database: %s", instance.Name, databaseCreate.Name)
		}
	}

	// Case 3, only appear in the Bytebase metadata
	for _, db := range dbList {
		found := false
		for _, databaseMetadata := range instanceMeta.DatabaseList {
			if db.Name == databaseMetadata.Name {
				found = true
				break
			}
		}
		if !found {
			syncStatus := api.NotFound
			ts := time.Now().Unix()
			databasePatch := &api.DatabasePatch{
				ID:                   db.ID,
				UpdaterID:            api.SystemBotID,
				SyncStatus:           &syncStatus,
				LastSuccessfulSyncTs: &ts,
				// SchemaVersion will not be over-written.
			}
			database, err := s.store.PatchDatabase(ctx, databasePatch)
			if err != nil {
				if common.ErrorCode(err) == common.NotFound {
					return nil, errors.Errorf("failed to sync database for instance: %s. Database not found: %s", instance.Name, database.Name)
				}
				return nil, errors.Wrapf(err, "failed to sync database for instance: %s. Failed to update database: %s", instance.Name, database.Name)
			}
		}
	}

	var databaseList []string
	for _, database := range instanceMeta.DatabaseList {
		databaseList = append(databaseList, database.Name)
	}

	return databaseList, nil
}

// SyncDatabaseSchema will sync the schema for a database.
func (s *Syncer) SyncDatabaseSchema(ctx context.Context, instance *api.Instance, databaseName string) error {
	driver, err := s.dbFactory.GetAdminDatabaseDriver(ctx, instance, "")
	if err != nil {
		return err
	}
	defer driver.Close(ctx)

	databaseFind := &api.DatabaseFind{
		InstanceID: &instance.ID,
		Name:       &databaseName,
	}
	matchedDb, err := s.store.GetDatabase(ctx, databaseFind)
	if err != nil {
		return errors.Wrapf(err, "failed to sync database for instance: %s. Failed to find database list", instance.Name)
	}

	// Sync database schema
	schema, err := driver.SyncDBSchema(ctx, databaseName)
	if err != nil {
		return err
	}

	// When there are too many databases, this might have performance issue and will
	// cause frontend timeout since we set a 30s limit (INSTANCE_OPERATION_TIMEOUT).
	schemaVersion, err := utils.GetLatestSchemaVersion(ctx, driver, schema.Name)
	if err != nil {
		return err
	}

	var database *api.Database
	if matchedDb != nil {
		syncStatus := api.OK
		ts := time.Now().Unix()
		databasePatch := &api.DatabasePatch{
			ID:                   matchedDb.ID,
			UpdaterID:            api.SystemBotID,
			SyncStatus:           &syncStatus,
			LastSuccessfulSyncTs: &ts,
			SchemaVersion:        &schemaVersion,
		}
		dbPatched, err := s.store.PatchDatabase(ctx, databasePatch)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return errors.Errorf("failed to sync database for instance: %s. Database not found: %v", instance.Name, matchedDb.Name)
			}
			return errors.Wrapf(err, "failed to sync database for instance: %s. Failed to update database: %s", instance.Name, matchedDb.Name)
		}
		database = dbPatched
	} else {
		databaseCreate := &api.DatabaseCreate{
			CreatorID:            api.SystemBotID,
			ProjectID:            api.DefaultProjectID,
			InstanceID:           instance.ID,
			EnvironmentID:        instance.EnvironmentID,
			Name:                 schema.Name,
			CharacterSet:         schema.CharacterSet,
			Collation:            schema.Collation,
			SchemaVersion:        schemaVersion,
			LastSuccessfulSyncTs: time.Now().Unix(),
		}
		createdDatabase, err := s.store.CreateDatabase(ctx, databaseCreate)
		if err != nil {
			if common.ErrorCode(err) == common.Conflict {
				return errors.Errorf("failed to sync database for instance: %s. Database name already exists: %s", instance.Name, databaseCreate.Name)
			}
			return errors.Wrapf(err, "failed to sync database for instance: %s. Failed to import new database: %s", instance.Name, databaseCreate.Name)
		}
		database = createdDatabase
	}
	// Sync database schema
	if s.profile.Mode == common.ReleaseModeDev {
		if err := syncDBSchema(ctx, s.store, database, schema); err != nil {
			return err
		}
	}
	if err := syncTableSchema(ctx, s.store, database, schema); err != nil {
		return err
	}
	if err := syncViewSchema(ctx, s.store, database, schema); err != nil {
		return err
	}
	return syncDBExtensionSchema(ctx, s.store, database, schema)
}

func syncTableSchema(ctx context.Context, store *store.Store, database *api.Database, schema *db.Schema) error {
	return store.SetTableList(ctx, schema, database.ID)
}

func syncViewSchema(ctx context.Context, store *store.Store, database *api.Database, schema *db.Schema) error {
	return store.SetViewList(ctx, schema, database.ID)
}

func syncDBExtensionSchema(ctx context.Context, store *store.Store, database *api.Database, schema *db.Schema) error {
	return store.SetDBExtensionList(ctx, schema, database.ID)
}

func syncDBSchema(ctx context.Context, store *store.Store, database *api.Database, schema *db.Schema) error {
	dbSchema, err := store.GetDBSchema(ctx, database.ID)
	if err != nil {
		return err
	}
	// TODO(d): finish the implementation.
	metadata := &storepb.SchemaMetadata{}
	for _, t := range schema.TableList {
		metadata.TableMetadata = append(metadata.TableMetadata, &storepb.TableMetadata{
			Name: t.Name,
		})
	}
	m := jsonpb.Marshaler{}
	metadataString, err := m.MarshalToString(metadata)
	if err != nil {
		return err
	}
	if dbSchema == nil || dbSchema.Metadata != metadataString {
		if _, err := store.UpsertDBSchema(ctx, api.DBSchemaUpsert{
			UpdatorID:  api.SystemBotID,
			DatabaseID: database.ID,
			Metadata:   metadataString,
			RawDump:    "",
		}); err != nil {
			return err
		}
	}
	return nil
}
