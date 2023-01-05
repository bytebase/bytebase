// Package schemasync is a runner that synchronize database schemas.
package schemasync

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"google.golang.org/protobuf/testing/protocmp"

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
				// We don't force dump database schema because it's rarely changed till the metadata is changed.
				if err := s.SyncDatabaseSchema(ctx, database, false /* force */); err != nil {
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

	instanceMeta, err := driver.SyncInstance(ctx)
	if err != nil {
		return nil, err
	}

	// Underlying version may change due to upgrade, however it's a rare event, so we only update if it actually differs
	// to avoid changing the updated_ts.
	if instanceMeta.Version != instance.EngineVersion {
		if _, err := s.store.PatchInstance(ctx, &store.InstancePatch{
			ID:            instance.ID,
			UpdaterID:     api.SystemBotID,
			EngineVersion: &instanceMeta.Version,
		}); err != nil {
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
		if _, err := s.store.UpsertInstanceUser(ctx, userUpsert); err != nil {
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
			if err := s.store.DeleteInstanceUser(ctx, userDelete); err != nil {
				return nil, errors.Wrapf(err, "failed to sync user for instance: %s. Failed to delete user: %s", instance.Name, user.Name)
			}
		}
	}

	databases, err := s.store.ListDatabases(ctx, &store.FindDatabaseMessage{InstanceID: &instance.ResourceID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to sync database for instance: %s. Failed to find database list", instance.Name)
	}
	for _, databaseMetadata := range instanceMeta.DatabaseList {
		exist := false
		for _, database := range databases {
			if database.DatabaseName == databaseMetadata.Name {
				exist = true
				break
			}
		}
		if !exist {
			databaseCreate := &api.DatabaseCreate{
				CreatorID:     api.SystemBotID,
				ProjectID:     api.DefaultProjectID,
				InstanceID:    instance.ID,
				EnvironmentID: instance.EnvironmentID,
				Name:          databaseMetadata.Name,
				CharacterSet:  databaseMetadata.CharacterSet,
				Collation:     databaseMetadata.Collation,
			}
			if _, err := s.store.CreateDatabase(ctx, databaseCreate); err != nil {
				if common.ErrorCode(err) == common.Conflict {
					return nil, errors.Errorf("failed to sync database for instance: %s. Database name already exists: %s", instance.Name, databaseCreate.Name)
				}
				return nil, errors.Wrapf(err, "failed to sync database for instance: %s. Failed to import new database: %s", instance.Name, databaseCreate.Name)
			}
		}
	}

	for _, database := range databases {
		exist := false
		for _, databaseMetadata := range instanceMeta.DatabaseList {
			if database.DatabaseName == databaseMetadata.Name {
				exist = true
				break
			}
		}
		if !exist {
			syncStatus := api.NotFound
			ts := time.Now().Unix()
			databasePatch := &api.DatabasePatch{
				ID:                   database.UID,
				UpdaterID:            api.SystemBotID,
				SyncStatus:           &syncStatus,
				LastSuccessfulSyncTs: &ts,
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
func (s *Syncer) SyncDatabaseSchema(ctx context.Context, database *api.Database, force bool) error {
	driver, err := s.dbFactory.GetAdminDatabaseDriver(ctx, database.Instance, database.Name)
	if err != nil {
		return err
	}
	defer driver.Close(ctx)
	// Sync database schema
	databaseMetadata, err := driver.SyncDBSchema(ctx, database.Name)
	if err != nil {
		return err
	}

	var patchSchemaVersion *string
	if force {
		// When there are too many databases, this might have performance issue and will
		// cause frontend timeout since we set a 30s limit (INSTANCE_OPERATION_TIMEOUT).
		schemaVersion, err := utils.GetLatestSchemaVersion(ctx, driver, databaseMetadata.Name)
		if err != nil {
			return err
		}
		patchSchemaVersion = &schemaVersion
	}

	syncStatus := api.OK
	ts := time.Now().Unix()
	databasePatch := &api.DatabasePatch{
		ID:                   database.ID,
		UpdaterID:            api.SystemBotID,
		SyncStatus:           &syncStatus,
		LastSuccessfulSyncTs: &ts,
		SchemaVersion:        patchSchemaVersion,
		// TODO(d): update CharacterSet and Collation.
	}
	database, err = s.store.PatchDatabase(ctx, databasePatch)
	if err != nil {
		if common.ErrorCode(err) == common.NotFound {
			return errors.Errorf("failed to sync database for instance: %s. Database not found: %v", database.Instance.Name, database.Name)
		}
		return errors.Wrapf(err, "failed to sync database for instance: %s. Failed to update database: %s", database.Instance.Name, database.Name)
	}

	return syncDBSchema(ctx, s.store, database, databaseMetadata, driver, force)
}

func syncDBSchema(ctx context.Context, stores *store.Store, database *api.Database, databaseMetadata *storepb.DatabaseMetadata, driver db.Driver, force bool) error {
	dbSchema, err := stores.GetDBSchema(ctx, database.ID)
	if err != nil {
		return err
	}
	var oldDatabaseMetadata *storepb.DatabaseMetadata
	if dbSchema != nil {
		oldDatabaseMetadata = dbSchema.Metadata
	}

	if !cmp.Equal(oldDatabaseMetadata, databaseMetadata, protocmp.Transform()) {
		var rawDump []byte
		if dbSchema != nil {
			rawDump = dbSchema.Schema
		}
		// Avoid updating dump everytime by dumping the schema only when the database metadata is changed.
		// if oldDatabaseMetadata is nil and databaseMetadata is not, they are not equal resulting a sync.
		if force || !equalDatabaseMetadata(oldDatabaseMetadata, databaseMetadata) {
			var schemaBuf bytes.Buffer
			if _, err := driver.Dump(ctx, database.Name, &schemaBuf, true /* schemaOnly */); err != nil {
				return err
			}
			rawDump = schemaBuf.Bytes()
		}

		if err := stores.UpsertDBSchema(ctx, database.ID, &store.DBSchema{
			Metadata: databaseMetadata,
			Schema:   rawDump,
		}, api.SystemBotID); err != nil {
			return err
		}
	}
	return nil
}

func equalDatabaseMetadata(x, y *storepb.DatabaseMetadata) bool {
	return cmp.Equal(x, y, protocmp.Transform(),
		protocmp.IgnoreFields(&storepb.TableMetadata{}, "row_count", "data_size", "index_size", "data_free"),
	)
}
