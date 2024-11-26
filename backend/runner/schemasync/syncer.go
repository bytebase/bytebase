// Package schemasync is a runner that synchronize database schemas.
package schemasync

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/sourcegraph/conc/pool"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/state"
	enterprise "github.com/bytebase/bytebase/backend/enterprise/api"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/store/model"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

const (
	instanceSyncInterval        = 15 * time.Minute
	databaseSyncCheckerInterval = 10 * time.Second
	syncTimeout                 = 15 * time.Minute
	// defaultSyncInterval means never sync.
	defaultSyncInterval = 0 * time.Second
	MaximumOutstanding  = 100

	backupDatabaseName       = "bbdataarchive"
	oracleBackupDatabaseName = "BBDATAARCHIVE"
)

// NewSyncer creates a schema syncer.
func NewSyncer(stores *store.Store, dbFactory *dbfactory.DBFactory, stateCfg *state.State, profile *config.Profile, licenseService enterprise.LicenseService) *Syncer {
	return &Syncer{
		store:          stores,
		dbFactory:      dbFactory,
		stateCfg:       stateCfg,
		profile:        profile,
		licenseService: licenseService,
	}
}

// Syncer is the schema syncer.
type Syncer struct {
	sync.Mutex

	store           *store.Store
	dbFactory       *dbfactory.DBFactory
	stateCfg        *state.State
	profile         *config.Profile
	licenseService  enterprise.LicenseService
	databaseSyncMap sync.Map // map[int]*store.DatabaseMessage
}

// Run will run the schema syncer once.
func (s *Syncer) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	sp := pool.New()
	sp.Go(func() {
		slog.Debug(fmt.Sprintf("Schema syncer started and will run every %v", instanceSyncInterval))
		ticker := time.NewTicker(instanceSyncInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.trySyncAll(ctx)
			case <-ctx.Done(): // if cancel() execute
				return
			}
		}
	})

	sp.Go(func() {
		ticker := time.NewTicker(databaseSyncCheckerInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				instances, err := s.store.ListInstancesV2(ctx, &store.FindInstanceMessage{})
				if err != nil {
					if err != nil {
						slog.Error("Failed to list instance", log.BBError(err))
						return
					}
				}
				instanceMap := make(map[string]*store.InstanceMessage)
				for _, instance := range instances {
					instanceMap[instance.ResourceID] = instance
				}
				dbwp := pool.New().WithMaxGoroutines(MaximumOutstanding)
				s.databaseSyncMap.Range(func(key, value any) bool {
					database, ok := value.(*store.DatabaseMessage)
					if !ok {
						return true
					}

					instance, ok := instanceMap[database.InstanceID]
					if !ok {
						slog.Debug("Instance not found",
							slog.String("instance", database.InstanceID),
							log.BBError(err))
						return true
					}
					if s.stateCfg.InstanceOutstandingConnections.Increment(instance.UID, int(instance.Options.GetMaximumConnections())) {
						return true
					}

					s.databaseSyncMap.Delete(key)
					dbwp.Go(func() {
						defer func() {
							s.stateCfg.InstanceOutstandingConnections.Decrement(instance.UID)
						}()
						slog.Debug("Sync database schema", slog.String("instance", database.InstanceID), slog.String("database", database.DatabaseName))
						if err := s.SyncDatabaseSchema(ctx, database, false /* force */); err != nil {
							slog.Debug("Failed to sync database schema",
								slog.String("instance", database.InstanceID),
								slog.String("databaseName", database.DatabaseName),
								log.BBError(err))
						}
					})
					return true
				})
				dbwp.Wait()
			case <-ctx.Done(): // if cancel() execute
				return
			}
		}
	})
	sp.Wait()
}

func (s *Syncer) trySyncAll(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			err, ok := r.(error)
			if !ok {
				err = errors.Errorf("%v", r)
			}
			slog.Error("Instance syncer PANIC RECOVER", log.BBError(err), log.BBStack("panic-stack"))
		}
	}()

	wp := pool.New().WithMaxGoroutines(MaximumOutstanding)
	instances, err := s.store.ListInstancesV2(ctx, &store.FindInstanceMessage{})
	if err != nil {
		slog.Error("Failed to retrieve instances", log.BBError(err))
		return
	}
	now := time.Now()
	for _, instance := range instances {
		instance := instance
		interval := getOrDefaultSyncInterval(instance)
		if interval == defaultSyncInterval {
			continue
		}
		lastSyncTime := getOrDefaultLastSyncTime(instance.Metadata.LastSyncTime)
		// lastSyncTime + syncInterval > now
		// Next round not started yet.
		nextSyncTime := lastSyncTime.Add(interval)
		if now.Before(nextSyncTime) {
			continue
		}

		wp.Go(func() {
			slog.Debug("Sync instance schema", slog.String("instance", instance.ResourceID))
			if _, _, err := s.SyncInstance(ctx, instance); err != nil {
				slog.Debug("Failed to sync instance",
					slog.String("instance", instance.ResourceID),
					slog.String("error", err.Error()))
			}
		})
	}
	wp.Wait()

	instancesMap := map[string]*store.InstanceMessage{}
	for _, instance := range instances {
		instancesMap[instance.ResourceID] = instance
	}

	databases, err := s.store.ListDatabases(ctx, &store.FindDatabaseMessage{})
	if err != nil {
		slog.Error("Failed to retrieve databases", log.BBError(err))
		return
	}
	for _, database := range databases {
		database := database
		if database.SyncState != api.OK {
			continue
		}
		instance, ok := instancesMap[database.InstanceID]
		if !ok {
			continue
		}
		// The database inherits the sync interval from the instance.
		interval := getOrDefaultSyncInterval(instance)
		if interval == defaultSyncInterval {
			continue
		}
		lastSyncTime := getOrDefaultLastSyncTime(database.Metadata.LastSyncTime)
		// lastSyncTime + syncInterval > now
		// Next round not started yet.
		nextSyncTime := lastSyncTime.Add(interval)
		if now.Before(nextSyncTime) {
			continue
		}

		s.databaseSyncMap.Store(database.UID, database)
	}
}

func (s *Syncer) SyncAllDatabases(ctx context.Context, instance *store.InstanceMessage) {
	find := &store.FindDatabaseMessage{}
	if instance != nil {
		find.InstanceID = &instance.ResourceID
	}
	databases, err := s.store.ListDatabases(ctx, find)
	if err != nil {
		slog.Debug("Failed to find databases to sync",
			slog.String("error", err.Error()))
		return
	}

	for _, database := range databases {
		// Skip deleted databases.
		if database.SyncState != api.OK {
			continue
		}
		s.databaseSyncMap.Store(database.UID, database)
	}
}

func (s *Syncer) SyncDatabasesAsync(databases []*store.DatabaseMessage) {
	for _, database := range databases {
		// Skip deleted databases.
		if database.SyncState != api.OK {
			continue
		}
		s.databaseSyncMap.Store(database.UID, database)
	}
}

// SyncInstance syncs the schema for all databases in an instance.
func (s *Syncer) SyncInstance(ctx context.Context, instance *store.InstanceMessage) (*store.InstanceMessage, []*store.DatabaseMessage, error) {
	if s.profile.Readonly {
		return nil, nil, nil
	}

	driver, err := s.dbFactory.GetAdminDatabaseDriver(ctx, instance, nil /* database */, db.ConnectionContext{})
	if err != nil {
		s.upsertInstanceConnectionAnomaly(ctx, instance, err)
		return nil, nil, err
	}
	defer driver.Close(ctx)
	s.upsertInstanceConnectionAnomaly(ctx, instance, nil)

	deadlineCtx, cancelFunc := context.WithDeadline(ctx, time.Now().Add(syncTimeout))
	defer cancelFunc()
	instanceMeta, err := driver.SyncInstance(deadlineCtx)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to sync instance: %s", instance.ResourceID)
	}

	if instanceMeta.Metadata == nil {
		instanceMeta.Metadata = &storepb.InstanceMetadata{}
	}
	instanceMeta.Metadata.LastSyncTime = timestamppb.Now()
	updateInstance := &store.UpdateInstanceMessage{
		ResourceID: instance.ResourceID,
		Metadata:   instanceMeta.Metadata,
		UpdaterID:  api.SystemBotID,
	}
	if instanceMeta.Version != instance.EngineVersion {
		updateInstance.EngineVersion = &instanceMeta.Version
	}
	updatedInstance, err := s.store.UpdateInstanceV2(ctx, updateInstance, -1)
	if err != nil {
		return nil, nil, err
	}

	databases, err := s.store.ListDatabases(ctx, &store.FindDatabaseMessage{InstanceID: &instance.ResourceID})
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to sync database for instance: %s. Failed to find database list", instance.ResourceID)
	}
	var newDatabases []*store.DatabaseMessage
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
			newDatabase, err := s.store.CreateDatabaseDefault(ctx, &store.DatabaseMessage{
				InstanceID:   instance.ResourceID,
				DatabaseName: databaseMetadata.Name,
				DataShare:    databaseMetadata.Datashare,
				ServiceName:  databaseMetadata.ServiceName,
				ProjectID:    api.DefaultProjectID,
			})
			if err != nil {
				return nil, nil, errors.Wrapf(err, "failed to create instance %q database %q in sync runner", instance.ResourceID, databaseMetadata.Name)
			}
			newDatabases = append(newDatabases, newDatabase)
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
				return nil, nil, errors.Errorf("failed to update database %q for instance %q", database.DatabaseName, instance.ResourceID)
			}
		}
	}

	return updatedInstance, newDatabases, nil
}

// SyncDatabaseSchema will sync the schema for a database.
func (s *Syncer) SyncDatabaseSchemaToHistory(ctx context.Context, database *store.DatabaseMessage, force bool) (int64, error) {
	if s.profile.Readonly {
		return 0, nil
	}

	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &database.InstanceID})
	if err != nil {
		return 0, errors.Wrapf(err, "failed to get instance %q", database.InstanceID)
	}
	if instance == nil {
		return 0, errors.Errorf("instance %q not found", database.InstanceID)
	}
	driver, err := s.dbFactory.GetAdminDatabaseDriver(ctx, instance, database, db.ConnectionContext{})
	if err != nil {
		s.upsertDatabaseConnectionAnomaly(ctx, instance, database, err)
		return 0, err
	}
	defer driver.Close(ctx)
	s.upsertDatabaseConnectionAnomaly(ctx, instance, database, nil)
	// Sync database schema
	deadlineCtx, cancelFunc := context.WithDeadline(ctx, time.Now().Add(syncTimeout))
	defer cancelFunc()
	databaseMetadata, err := driver.SyncDBSchema(deadlineCtx)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to sync database schema for database %q", database.DatabaseName)
	}

	dbSchema, err := s.store.GetDBSchema(ctx, database.UID)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to get database schema for database %q", database.DatabaseName)
	}

	dbModelConfig := model.NewDatabaseConfig(nil)
	if dbSchema != nil {
		dbModelConfig = dbSchema.GetInternalConfig()
	}

	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &database.ProjectID,
	})
	if err != nil {
		return 0, errors.Wrapf(err, `failed to get project by id "%s"`, database.ProjectID)
	}
	classificationConfig, err := s.store.GetDataClassificationConfigByID(ctx, project.DataClassificationConfigID)
	if err != nil {
		return 0, errors.Wrapf(err, `failed to get classification config by id "%s"`, project.DataClassificationConfigID)
	}

	if instance.Engine != storepb.Engine_MYSQL && instance.Engine != storepb.Engine_POSTGRES {
		// Force to disable classification from comment if the engine is not MYSQL or PG.
		classificationConfig.ClassificationFromConfig = true
	}
	if classificationConfig.ClassificationFromConfig {
		// Only set the user comment.
		setUserCommentFromComment(databaseMetadata)
	} else {
		// Get classification from the comment.
		setClassificationAndUserCommentFromComment(databaseMetadata, dbModelConfig, classificationConfig)
	}

	syncStatus := api.OK
	ts := time.Now().Unix()
	if _, err := s.store.UpdateDatabase(ctx, &store.UpdateDatabaseMessage{
		InstanceID:           database.InstanceID,
		DatabaseName:         database.DatabaseName,
		SyncState:            &syncStatus,
		SuccessfulSyncTimeTs: &ts,
		MetadataUpsert: &storepb.DatabaseMetadata{
			LastSyncTime:    timestamppb.New(time.Unix(ts, 0)),
			BackupAvailable: s.hasBackupSchema(ctx, instance, databaseMetadata),
		},
	}, api.SystemBotID); err != nil {
		return 0, errors.Wrapf(err, "failed to update database %q for instance %q", database.DatabaseName, database.InstanceID)
	}

	var oldDatabaseMetadata *storepb.DatabaseSchemaMetadata
	var rawDump []byte
	if dbSchema != nil {
		oldDatabaseMetadata = dbSchema.GetMetadata()
		rawDump = dbSchema.GetSchema()
	}

	// Avoid updating dump everytime by dumping the schema only when the database metadata is changed.
	// if oldDatabaseMetadata is nil and databaseMetadata is not, they are not equal resulting a sync.
	if force || !common.EqualDatabaseSchemaMetadataFast(oldDatabaseMetadata, databaseMetadata) {
		var schemaBuf bytes.Buffer
		if err := driver.Dump(ctx, &schemaBuf, databaseMetadata); err != nil {
			return 0, errors.Wrapf(err, "failed to dump database schema for database %q", database.DatabaseName)
		}
		rawDump = schemaBuf.Bytes()
	}

	if err := s.store.UpsertDBSchema(ctx,
		database.UID,
		model.NewDBSchema(databaseMetadata, rawDump, dbModelConfig.BuildDatabaseConfig()),
		api.SystemBotID,
	); err != nil {
		if strings.Contains(err.Error(), "escape sequence") {
			if metadataBytes, err := protojson.Marshal(databaseMetadata); err == nil {
				slog.Error("unsupported Unicode escape sequence", slog.String("metadata", string(metadataBytes)), slog.String("raw_dump", string(rawDump)))
			}
		}
		return 0, errors.Wrapf(err, "failed to upsert database schema for database %q", database.DatabaseName)
	}

	id, err := s.store.CreateSyncHistory(ctx, database.UID, databaseMetadata, string(rawDump), api.SystemBotID)
	if err != nil {
		if strings.Contains(err.Error(), "escape sequence") {
			if metadataBytes, err := protojson.Marshal(databaseMetadata); err == nil {
				slog.Error("unsupported Unicode escape sequence", slog.String("metadata", string(metadataBytes)), slog.String("raw_dump", string(rawDump)))
			}
		}
		return 0, errors.Wrapf(err, "failed to insert sync history for database %q", database.DatabaseName)
	}

	return id, nil
}

// SyncDatabaseSchema will sync the schema for a database.
func (s *Syncer) SyncDatabaseSchema(ctx context.Context, database *store.DatabaseMessage, force bool) (retErr error) {
	if s.profile.Readonly {
		return nil
	}

	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &database.InstanceID})
	if err != nil {
		return errors.Wrapf(err, "failed to get instance %q", database.InstanceID)
	}
	if instance == nil {
		return errors.Errorf("instance %q not found", database.InstanceID)
	}
	driver, err := s.dbFactory.GetAdminDatabaseDriver(ctx, instance, database, db.ConnectionContext{})
	if err != nil {
		s.upsertDatabaseConnectionAnomaly(ctx, instance, database, err)
		return err
	}
	defer driver.Close(ctx)
	s.upsertDatabaseConnectionAnomaly(ctx, instance, database, nil)
	// Sync database schema
	deadlineCtx, cancelFunc := context.WithDeadline(ctx, time.Now().Add(syncTimeout))
	defer cancelFunc()
	databaseMetadata, err := driver.SyncDBSchema(deadlineCtx)
	if err != nil {
		return errors.Wrapf(err, "failed to sync database schema for database %q", database.DatabaseName)
	}

	dbSchema, err := s.store.GetDBSchema(ctx, database.UID)
	if err != nil {
		return errors.Wrapf(err, "failed to get database schema for database %q", database.DatabaseName)
	}

	dbModelConfig := model.NewDatabaseConfig(nil)
	if dbSchema != nil {
		dbModelConfig = dbSchema.GetInternalConfig()
	}

	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &database.ProjectID,
	})
	if err != nil {
		return errors.Wrapf(err, `failed to get project by id "%s"`, database.ProjectID)
	}
	classificationConfig, err := s.store.GetDataClassificationConfigByID(ctx, project.DataClassificationConfigID)
	if err != nil {
		return errors.Wrapf(err, `failed to get classification config by id "%s"`, project.DataClassificationConfigID)
	}

	if instance.Engine != storepb.Engine_MYSQL && instance.Engine != storepb.Engine_POSTGRES {
		// Force to disable classification from comment if the engine is not MYSQL or PG.
		classificationConfig.ClassificationFromConfig = true
	}
	if classificationConfig.ClassificationFromConfig {
		// Only set the user comment.
		setUserCommentFromComment(databaseMetadata)
	} else {
		// Get classification from the comment.
		setClassificationAndUserCommentFromComment(databaseMetadata, dbModelConfig, classificationConfig)
	}

	syncStatus := api.OK
	ts := time.Now().Unix()
	if _, err := s.store.UpdateDatabase(ctx, &store.UpdateDatabaseMessage{
		InstanceID:           database.InstanceID,
		DatabaseName:         database.DatabaseName,
		SyncState:            &syncStatus,
		SuccessfulSyncTimeTs: &ts,
		MetadataUpsert: &storepb.DatabaseMetadata{
			LastSyncTime:    timestamppb.New(time.Unix(ts, 0)),
			BackupAvailable: s.hasBackupSchema(ctx, instance, databaseMetadata),
		},
	}, api.SystemBotID); err != nil {
		return errors.Wrapf(err, "failed to update database %q for instance %q", database.DatabaseName, database.InstanceID)
	}

	var oldDatabaseMetadata *storepb.DatabaseSchemaMetadata
	var rawDump []byte
	if dbSchema != nil {
		oldDatabaseMetadata = dbSchema.GetMetadata()
		rawDump = dbSchema.GetSchema()
	}

	// Avoid updating dump everytime by dumping the schema only when the database metadata is changed.
	// if oldDatabaseMetadata is nil and databaseMetadata is not, they are not equal resulting a sync.
	if force || !common.EqualDatabaseSchemaMetadataFast(oldDatabaseMetadata, databaseMetadata) {
		var schemaBuf bytes.Buffer
		if err := driver.Dump(ctx, &schemaBuf, databaseMetadata); err != nil {
			return errors.Wrapf(err, "failed to dump database schema for database %q", database.DatabaseName)
		}
		rawDump = schemaBuf.Bytes()
	}

	if err := s.store.UpsertDBSchema(ctx,
		database.UID,
		model.NewDBSchema(databaseMetadata, rawDump, dbModelConfig.BuildDatabaseConfig()),
		api.SystemBotID,
	); err != nil {
		if strings.Contains(err.Error(), "escape sequence") {
			if metadataBytes, err := protojson.Marshal(databaseMetadata); err == nil {
				slog.Error("unsupported Unicode escape sequence", slog.String("metadata", string(metadataBytes)), slog.String("raw_dump", string(rawDump)))
			}
		}
		return errors.Wrapf(err, "failed to upsert database schema for database %q", database.DatabaseName)
	}

	// Check schema drift
	if s.licenseService.IsFeatureEnabledForInstance(api.FeatureSchemaDrift, instance) == nil {
		// Redis and MongoDB are schemaless.
		if disableSchemaDriftAnomalyCheck(instance.Engine) {
			return nil
		}
		limit := 1
		list, err := s.store.ListInstanceChangeHistory(ctx, &store.FindInstanceChangeHistoryMessage{
			InstanceID: &instance.UID,
			DatabaseID: &database.UID,
			TypeList:   []db.MigrationType{db.Migrate, db.Baseline},
			ShowFull:   true,
			Limit:      &limit,
		})
		if err != nil {
			slog.Error("Failed to check anomaly",
				slog.String("instance", instance.ResourceID),
				slog.String("database", database.DatabaseName),
				slog.String("type", string(api.AnomalyDatabaseSchemaDrift)),
				log.BBError(err))
			return nil
		}
		latestSchema := string(rawDump)
		if len(list) > 0 {
			if list[0].Schema != latestSchema {
				anomalyPayload := &storepb.AnomalyDatabaseSchemaDriftPayload{
					Version: list[0].Version.Version,
					Expect:  list[0].Schema,
					Actual:  latestSchema,
				}
				payload, err := protojson.Marshal(anomalyPayload)
				if err != nil {
					slog.Error("Failed to marshal anomaly payload",
						slog.String("instance", instance.ResourceID),
						slog.String("database", database.DatabaseName),
						slog.String("type", string(api.AnomalyDatabaseSchemaDrift)),
						log.BBError(err))
				} else {
					if _, err = s.store.UpsertActiveAnomalyV2(ctx, api.SystemBotID, &store.AnomalyMessage{
						InstanceID:  instance.ResourceID,
						DatabaseUID: &database.UID,
						Type:        api.AnomalyDatabaseSchemaDrift,
						Payload:     string(payload),
					}); err != nil {
						slog.Error("Failed to create anomaly",
							slog.String("instance", instance.ResourceID),
							slog.String("database", database.DatabaseName),
							slog.String("type", string(api.AnomalyDatabaseSchemaDrift)),
							log.BBError(err))
					}
				}
			} else {
				err := s.store.ArchiveAnomalyV2(ctx, &store.ArchiveAnomalyMessage{
					DatabaseUID: &database.UID,
					Type:        api.AnomalyDatabaseSchemaDrift,
				})
				if err != nil && common.ErrorCode(err) != common.NotFound {
					slog.Error("Failed to close anomaly",
						slog.String("instance", instance.ResourceID),
						slog.String("database", database.DatabaseName),
						slog.String("type", string(api.AnomalyDatabaseSchemaDrift)),
						log.BBError(err))
				}
			}
		}
	}
	return nil
}

func (s *Syncer) hasBackupSchema(ctx context.Context, instance *store.InstanceMessage, dbSchema *storepb.DatabaseSchemaMetadata) bool {
	switch instance.Engine {
	case storepb.Engine_POSTGRES:
		if dbSchema == nil {
			return false
		}
		for _, schema := range dbSchema.Schemas {
			if schema.GetName() == backupDatabaseName {
				return true
			}
		}
	case storepb.Engine_MYSQL, storepb.Engine_MSSQL, storepb.Engine_TIDB:
		dbName := backupDatabaseName
		backupDB, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
			InstanceID:   &instance.ResourceID,
			DatabaseName: &dbName,
		})
		if err != nil {
			slog.Debug("Failed to get backup database", "err", err)
			return false
		}
		return backupDB != nil
	case storepb.Engine_ORACLE:
		dbName := oracleBackupDatabaseName
		backupDB, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
			InstanceID:   &instance.ResourceID,
			DatabaseName: &dbName,
		})
		if err != nil {
			slog.Debug("Failed to get backup database", "err", err)
			return false
		}
		return backupDB != nil
	}
	return false
}

func (s *Syncer) upsertInstanceConnectionAnomaly(ctx context.Context, instance *store.InstanceMessage, connErr error) {
	if connErr != nil {
		anomalyPayload := &storepb.AnomalyConnectionPayload{
			Detail: connErr.Error(),
		}
		payload, err := protojson.Marshal(anomalyPayload)
		if err != nil {
			slog.Error("Failed to marshal anomaly payload",
				slog.String("instance", instance.ResourceID),
				slog.String("type", string(api.AnomalyInstanceConnection)),
				log.BBError(err))
			return
		}
		if _, err = s.store.UpsertActiveAnomalyV2(ctx, api.SystemBotID, &store.AnomalyMessage{
			InstanceID: instance.ResourceID,
			Type:       api.AnomalyInstanceConnection,
			Payload:    string(payload),
		}); err != nil {
			slog.Error("Failed to create anomaly",
				slog.String("instance", instance.ResourceID),
				slog.String("type", string(api.AnomalyInstanceConnection)),
				log.BBError(err))
		}
		return
	}

	err := s.store.ArchiveAnomalyV2(ctx, &store.ArchiveAnomalyMessage{
		InstanceID: &instance.ResourceID,
		Type:       api.AnomalyInstanceConnection,
	})
	if err != nil && common.ErrorCode(err) != common.NotFound {
		slog.Error("Failed to close anomaly",
			slog.String("instance", instance.ResourceID),
			slog.String("type", string(api.AnomalyInstanceConnection)),
			log.BBError(err))
	}
}

func (s *Syncer) upsertDatabaseConnectionAnomaly(ctx context.Context, instance *store.InstanceMessage, database *store.DatabaseMessage, connErr error) {
	if connErr != nil {
		anomalyPayload := &storepb.AnomalyConnectionPayload{
			Detail: connErr.Error(),
		}
		payload, err := protojson.Marshal(anomalyPayload)
		if err != nil {
			slog.Error("Failed to marshal anomaly payload",
				slog.String("instance", instance.ResourceID),
				slog.String("database", database.DatabaseName),
				slog.String("type", string(api.AnomalyDatabaseConnection)),
				log.BBError(err))
		} else {
			if _, err = s.store.UpsertActiveAnomalyV2(ctx, api.SystemBotID, &store.AnomalyMessage{
				InstanceID:  instance.ResourceID,
				DatabaseUID: &database.UID,
				Type:        api.AnomalyDatabaseConnection,
				Payload:     string(payload),
			}); err != nil {
				slog.Error("Failed to create anomaly",
					slog.String("instance", instance.ResourceID),
					slog.String("database", database.DatabaseName),
					slog.String("type", string(api.AnomalyDatabaseConnection)),
					log.BBError(err))
			}
		}
		return
	}

	err := s.store.ArchiveAnomalyV2(ctx, &store.ArchiveAnomalyMessage{
		DatabaseUID: &database.UID,
		Type:        api.AnomalyDatabaseConnection,
	})
	if err != nil && common.ErrorCode(err) != common.NotFound {
		slog.Error("Failed to close anomaly",
			slog.String("instance", instance.ResourceID),
			slog.String("database", database.DatabaseName),
			slog.String("type", string(api.AnomalyDatabaseConnection)),
			log.BBError(err))
	}
}

func setClassificationAndUserCommentFromComment(dbSchema *storepb.DatabaseSchemaMetadata, databaseConfig *model.DatabaseConfig, classificationConfig *storepb.DataClassificationSetting_DataClassificationConfig) {
	for _, schema := range dbSchema.Schemas {
		schemaConfig := databaseConfig.CreateOrGetSchemaConfig(schema.Name)

		for _, table := range schema.Tables {
			tableConfig := schemaConfig.CreateOrGetTableConfig(table.Name)
			classification, userComment := common.GetClassificationAndUserComment(table.Comment, classificationConfig)

			table.UserComment = userComment
			tableConfig.ClassificationID = classification

			for _, col := range table.Columns {
				columnConfig := tableConfig.CreateOrGetColumnConfig(col.Name)
				colClassification, colUserComment := common.GetClassificationAndUserComment(col.Comment, classificationConfig)

				col.UserComment = colUserComment
				columnConfig.ClassificationId = colClassification

				if isEmptyColumnConfig(columnConfig) {
					tableConfig.RemoveColumnConfig(col.Name)
				}
			}

			if tableConfig.IsEmpty() {
				schemaConfig.RemoveTableConfig(table.Name)
			}
		}
		if schemaConfig.IsEmpty() {
			databaseConfig.RemoveSchemaConfig(schema.Name)
		}
	}
}

func isEmptyColumnConfig(config *storepb.ColumnConfig) bool {
	return len(config.Labels) == 0 && config.ClassificationId == "" && config.SemanticTypeId == ""
}

func setUserCommentFromComment(dbSchema *storepb.DatabaseSchemaMetadata) {
	for _, schema := range dbSchema.Schemas {
		for _, table := range schema.Tables {
			table.UserComment = table.Comment
			for _, col := range table.Columns {
				col.UserComment = col.Comment
			}
		}
	}
}

func getOrDefaultSyncInterval(instance *store.InstanceMessage) time.Duration {
	if !instance.Activation {
		return defaultSyncInterval
	}
	if !instance.Options.SyncInterval.IsValid() {
		return defaultSyncInterval
	}
	if instance.Options.SyncInterval.GetSeconds() == 0 && instance.Options.SyncInterval.GetNanos() == 0 {
		return defaultSyncInterval
	}
	return instance.Options.SyncInterval.AsDuration()
}

func getOrDefaultLastSyncTime(t *timestamppb.Timestamp) time.Time {
	if t.IsValid() {
		return t.AsTime()
	}
	return time.Unix(0, 0)
}

func disableSchemaDriftAnomalyCheck(dbTp storepb.Engine) bool {
	m := map[storepb.Engine]struct{}{
		storepb.Engine_MONGODB:    {},
		storepb.Engine_REDIS:      {},
		storepb.Engine_REDSHIFT:   {},
		storepb.Engine_RISINGWAVE: {},
	}
	_, ok := m[dbTp]
	return ok
}
