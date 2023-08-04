// Package schemasync is a runner that synchronize database schemas.
package schemasync

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/state"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
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
			s.syncAllDatabases(ctx, instance)
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

	instances, err := s.store.ListInstancesV2(ctx, &store.FindInstanceMessage{})
	if err != nil {
		log.Error("Failed to retrieve instances", zap.Error(err))
		return
	}

	var instanceWG sync.WaitGroup
	for _, instance := range instances {
		instanceWG.Add(1)
		go func(instance *store.InstanceMessage) {
			defer instanceWG.Done()
			log.Debug("Sync instance schema", zap.String("instance", instance.ResourceID))
			if _, err := s.SyncInstance(ctx, instance); err != nil {
				log.Debug("Failed to sync instance",
					zap.String("instance", instance.ResourceID),
					zap.String("error", err.Error()))
				return
			}
		}(instance)
	}
	instanceWG.Wait()
}

func (s *Syncer) syncAllDatabases(ctx context.Context, instance *store.InstanceMessage) {
	defer func() {
		if r := recover(); r != nil {
			err, ok := r.(error)
			if !ok {
				err = errors.Errorf("%v", r)
			}
			log.Error("Database syncer PANIC RECOVER", zap.Error(err), zap.Stack("panic-stack"))
		}
	}()

	find := &store.FindDatabaseMessage{}
	if instance != nil {
		find.InstanceID = &instance.ResourceID
	}
	databases, err := s.store.ListDatabases(ctx, find)
	if err != nil {
		log.Debug("Failed to find databases to sync",
			zap.String("error", err.Error()))
		return
	}

	instanceMap := make(map[string][]*store.DatabaseMessage)
	for _, database := range databases {
		// Skip deleted databases.
		if database.SyncState != api.OK {
			continue
		}
		instanceMap[database.InstanceID] = append(instanceMap[database.InstanceID], database)
	}

	var instanceWG sync.WaitGroup
	for _, databaseList := range instanceMap {
		instanceWG.Add(1)
		go func(databaseList []*store.DatabaseMessage) {
			defer instanceWG.Done()

			if len(databaseList) == 0 {
				return
			}
			instanceID := databaseList[0].InstanceID
			for _, database := range databaseList {
				log.Debug("Sync database schema",
					zap.String("instance", instanceID),
					zap.String("database", database.DatabaseName),
					zap.Int64("lastSuccessfulSyncTs", database.SuccessfulSyncTimeTs),
				)
				// If we fail to sync a particular database due to permission issue, we will continue to sync the rest of the databases.
				// We don't force dump database schema because it's rarely changed till the metadata is changed.
				if err := s.SyncDatabaseSchema(ctx, database, false /* force */); err != nil {
					log.Debug("Failed to sync database schema",
						zap.String("instance", instanceID),
						zap.String("databaseName", database.DatabaseName),
						zap.Error(err))
				}
			}
		}(databaseList)
	}
	instanceWG.Wait()
}

// SyncInstance syncs the schema for all databases in an instance.
func (s *Syncer) SyncInstance(ctx context.Context, instance *store.InstanceMessage) ([]string, error) {
	driver, err := s.dbFactory.GetAdminDatabaseDriver(ctx, instance, nil /* database */)
	if err != nil {
		return nil, err
	}
	defer driver.Close(ctx)

	instanceMeta, err := driver.SyncInstance(ctx)
	if err != nil {
		return nil, err
	}

	updateInstance := (*store.UpdateInstanceMessage)(nil)
	if instanceMeta.Version != instance.EngineVersion {
		updateInstance = &store.UpdateInstanceMessage{
			UpdaterID:     api.SystemBotID,
			EnvironmentID: instance.EnvironmentID,
			ResourceID:    instance.ResourceID,
			EngineVersion: &instanceMeta.Version,
		}
	}
	if !cmp.Equal(instanceMeta.Metadata, instance.Metadata, protocmp.Transform()) {
		if updateInstance == nil {
			updateInstance = &store.UpdateInstanceMessage{
				UpdaterID:     api.SystemBotID,
				EnvironmentID: instance.EnvironmentID,
				ResourceID:    instance.ResourceID,
				Metadata:      instanceMeta.Metadata,
			}
		} else {
			updateInstance.Metadata = instanceMeta.Metadata
		}
	}
	if updateInstance != nil {
		if _, err := s.store.UpdateInstanceV2(ctx, updateInstance, -1); err != nil {
			return nil, err
		}
	}

	var instanceUsers []*store.InstanceUserMessage
	for _, instanceUser := range instanceMeta.InstanceRoles {
		instanceUsers = append(instanceUsers, &store.InstanceUserMessage{
			Name:  instanceUser.Name,
			Grant: instanceUser.Grant,
		})
	}
	if err := s.store.UpsertInstanceUsers(ctx, instance.UID, instanceUsers); err != nil {
		return nil, err
	}

	databases, err := s.store.ListDatabases(ctx, &store.FindDatabaseMessage{InstanceID: &instance.ResourceID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to sync database for instance: %s. Failed to find database list", instance.ResourceID)
	}
	for _, databaseMetadata := range instanceMeta.Databases {
		exist := false
		for _, database := range databases {
			if database.DatabaseName == databaseMetadata.Name {
				exist = true
				break
			}
		}
		if !exist {
			// Create the database in the default project.
			if err := s.store.CreateDatabaseDefault(ctx, &store.DatabaseMessage{
				InstanceID:   instance.ResourceID,
				DatabaseName: databaseMetadata.Name,
				DataShare:    databaseMetadata.Datashare,
				ServiceName:  databaseMetadata.ServiceName,
			}); err != nil {
				return nil, errors.Wrapf(err, "failed to create instance %q database %q in sync runner", instance.ResourceID, databaseMetadata.Name)
			}
		}
	}

	for _, database := range databases {
		exist := false
		for _, databaseMetadata := range instanceMeta.Databases {
			if database.DatabaseName == databaseMetadata.Name {
				exist = true
				break
			}
		}
		if !exist {
			syncStatus := api.NotFound
			if _, err := s.store.UpdateDatabase(ctx, &store.UpdateDatabaseMessage{
				InstanceID:   instance.ResourceID,
				DatabaseName: database.DatabaseName,
				SyncState:    &syncStatus,
			}, api.SystemBotID); err != nil {
				return nil, errors.Errorf("failed to update database %q for instance %q", database.DatabaseName, instance.ResourceID)
			}
		}
	}

	var databaseList []string
	for _, database := range instanceMeta.Databases {
		databaseList = append(databaseList, database.Name)
	}
	return databaseList, nil
}

// SyncDatabaseSchema will sync the schema for a database.
func (s *Syncer) SyncDatabaseSchema(ctx context.Context, database *store.DatabaseMessage, force bool) error {
	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &database.InstanceID})
	if err != nil {
		return err
	}
	if instance == nil {
		return errors.Errorf("instance %q not found", database.InstanceID)
	}
	driver, err := s.dbFactory.GetAdminDatabaseDriver(ctx, instance, database)
	if err != nil {
		return err
	}
	defer driver.Close(ctx)
	// Sync database schema
	databaseMetadata, err := driver.SyncDBSchema(ctx)
	if err != nil {
		return err
	}
	setClassificationAndUserCommentFromComment(databaseMetadata)

	var patchSchemaVersion *string
	if force {
		// When there are too many databases, this might have performance issue and will
		// cause frontend timeout since we set a 30s limit (INSTANCE_OPERATION_TIMEOUT).
		schemaVersion, err := utils.GetLatestSchemaVersion(ctx, s.store, instance.UID, database.UID, databaseMetadata.Name)
		if err != nil {
			return err
		}
		patchSchemaVersion = &schemaVersion
	}

	syncStatus := api.OK
	ts := time.Now().Unix()
	if _, err := s.store.UpdateDatabase(ctx, &store.UpdateDatabaseMessage{
		InstanceID:           database.InstanceID,
		DatabaseName:         database.DatabaseName,
		SyncState:            &syncStatus,
		SuccessfulSyncTimeTs: &ts,
		DataShare:            &database.DataShare,
		SchemaVersion:        patchSchemaVersion,
		ServiceName:          &database.ServiceName,
	}, api.SystemBotID); err != nil {
		return errors.Wrapf(err, "failed to update database %q for instance %q", database.DatabaseName, database.InstanceID)
	}

	return syncDBSchema(ctx, s.store, database, databaseMetadata, driver, force)
}

func syncDBSchema(ctx context.Context, stores *store.Store, database *store.DatabaseMessage, databaseMetadata *storepb.DatabaseSchemaMetadata, driver db.Driver, force bool) error {
	dbSchema, err := stores.GetDBSchema(ctx, database.UID)
	if err != nil {
		return err
	}
	var oldDatabaseMetadata *storepb.DatabaseSchemaMetadata
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
			if _, err := driver.Dump(ctx, &schemaBuf, true /* schemaOnly */); err != nil {
				return err
			}
			rawDump = schemaBuf.Bytes()
		}

		if err := stores.UpsertDBSchema(ctx, database.UID, &store.DBSchema{
			Metadata: databaseMetadata,
			Schema:   rawDump,
		}, api.SystemBotID); err != nil {
			return err
		}
	}
	return nil
}

func equalDatabaseMetadata(x, y *storepb.DatabaseSchemaMetadata) bool {
	return cmp.Equal(x, y, protocmp.Transform(),
		protocmp.IgnoreFields(&storepb.TableMetadata{}, "row_count", "data_size", "index_size", "data_free"),
	)
}

func setClassificationAndUserCommentFromComment(dbSchema *storepb.DatabaseSchemaMetadata) {
	for _, schema := range dbSchema.Schemas {
		for _, table := range schema.Tables {
			table.Classification, table.UserComment = common.GetClassificationAndUserComment(table.Comment)
			for _, col := range table.Columns {
				col.Classification, col.UserComment = common.GetClassificationAndUserComment(col.Comment)
			}
		}
	}
}
