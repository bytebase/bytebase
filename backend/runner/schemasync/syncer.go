// Package schemasync is a runner that synchronize database schemas.
package schemasync

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/state"
	enterprise "github.com/bytebase/bytebase/backend/enterprise/api"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/store/model"
	"github.com/bytebase/bytebase/backend/utils"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

const (
	schemaSyncInterval = 1 * time.Minute
	// defaultSyncInterval means never sync.
	defaultSyncInterval = 0 * time.Second
)

// NewSyncer creates a schema syncer.
func NewSyncer(store *store.Store, dbFactory *dbfactory.DBFactory, stateCfg *state.State, profile config.Profile, licenseService enterprise.LicenseService) *Syncer {
	return &Syncer{
		store:          store,
		dbFactory:      dbFactory,
		stateCfg:       stateCfg,
		profile:        profile,
		licenseService: licenseService,
	}
}

// Syncer is the schema syncer.
type Syncer struct {
	store          *store.Store
	dbFactory      *dbfactory.DBFactory
	stateCfg       *state.State
	profile        config.Profile
	licenseService enterprise.LicenseService
}

// Run will run the schema syncer once.
func (s *Syncer) Run(ctx context.Context, wg *sync.WaitGroup) {
	ticker := time.NewTicker(schemaSyncInterval)
	defer ticker.Stop()
	defer wg.Done()
	slog.Debug(fmt.Sprintf("Schema syncer started and will run every %v", schemaSyncInterval))
	for {
		select {
		case <-ticker.C:
			s.trySyncAll(ctx)
		case instance := <-s.stateCfg.InstanceDatabaseSyncChan:
			// Sync all databases for instance.
			s.syncAllDatabases(ctx, instance)
		case <-ctx.Done(): // if cancel() execute
			return
		}
	}
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
	instances, err := s.store.ListInstancesV2(ctx, &store.FindInstanceMessage{})
	if err != nil {
		slog.Error("Failed to retrieve instances", log.BBError(err))
		return
	}

	environments, err := s.store.ListEnvironmentV2(ctx, &store.FindEnvironmentMessage{})
	if err != nil {
		slog.Error("Failed to retrieve environments", log.BBError(err))
		return
	}

	environmentsMap := map[string]*store.EnvironmentMessage{}
	for _, environment := range environments {
		environmentsMap[environment.ResourceID] = environment
	}

	backupPlanPolicyMap := make(map[string]*api.BackupPlanPolicy)
	for _, environment := range environments {
		policy, err := s.store.GetBackupPlanPolicyByEnvID(ctx, environment.UID)
		if err != nil {
			slog.Error("Failed to retrieve backup policy",
				slog.String("environment", environment.Title),
				log.BBError(err))
			return
		}
		backupPlanPolicyMap[environment.ResourceID] = policy
	}

	now := time.Now()
	for _, instance := range instances {
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

		slog.Debug("Sync instance schema", slog.String("instance", instance.ResourceID))
		if err := s.SyncInstance(ctx, instance); err != nil {
			slog.Debug("Failed to sync instance",
				slog.String("instance", instance.ResourceID),
				slog.String("error", err.Error()))
		}
	}

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
		if err := s.SyncDatabaseSchema(ctx, database, false /* force */); err != nil {
			slog.Debug("Failed to sync database schema",
				slog.String("instance", instance.ResourceID),
				slog.String("databaseName", database.DatabaseName),
				log.BBError(err))
		}

		environment, ok := environmentsMap[database.EffectiveEnvironmentID]
		if !ok {
			continue
		}
		backupPlanPolicy, ok := backupPlanPolicyMap[database.EffectiveEnvironmentID]
		if !ok {
			continue
		}

		s.checkBackupAnomaly(ctx, environment, instance, database, backupPlanPolicy)
	}
}

func (s *Syncer) syncAllDatabases(ctx context.Context, instance *store.InstanceMessage) {
	defer func() {
		if r := recover(); r != nil {
			err, ok := r.(error)
			if !ok {
				err = errors.Errorf("%v", r)
			}
			slog.Error("Database syncer PANIC RECOVER", log.BBError(err), log.BBStack("panic-stack"))
		}
	}()

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

	instanceMap := make(map[string][]*store.DatabaseMessage)
	for _, database := range databases {
		// Skip deleted databases.
		if database.SyncState != api.OK {
			continue
		}
		instanceMap[database.InstanceID] = append(instanceMap[database.InstanceID], database)
	}

	for _, databaseList := range instanceMap {
		for _, database := range databaseList {
			instanceID := database.InstanceID
			slog.Debug("Sync database schema",
				slog.String("instance", instanceID),
				slog.String("database", database.DatabaseName),
				slog.Int64("lastSuccessfulSyncTs", database.SuccessfulSyncTimeTs),
			)
			// If we fail to sync a particular database due to permission issue, we will continue to sync the rest of the databases.
			// We don't force dump database schema because it's rarely changed till the metadata is changed.
			if err := s.SyncDatabaseSchema(ctx, database, false /* force */); err != nil {
				slog.Debug("Failed to sync database schema",
					slog.String("instance", instanceID),
					slog.String("databaseName", database.DatabaseName),
					log.BBError(err))
			}
		}
	}
}

// SyncInstance syncs the schema for all databases in an instance.
func (s *Syncer) SyncInstance(ctx context.Context, instance *store.InstanceMessage) error {
	if s.profile.Readonly {
		return nil
	}

	driver, err := s.dbFactory.GetAdminDatabaseDriver(ctx, instance, nil /* database */)
	if err != nil {
		s.upsertInstanceConnectionAnomaly(ctx, instance, err)
		return err
	}
	defer driver.Close(ctx)
	s.upsertInstanceConnectionAnomaly(ctx, instance, nil)

	instanceMeta, err := driver.SyncInstance(ctx)
	if err != nil {
		return err
	}

	updateInstance := &store.UpdateInstanceMessage{
		UpdaterID:     api.SystemBotID,
		EnvironmentID: instance.EnvironmentID,
		ResourceID:    instance.ResourceID,
		Metadata: &storepb.InstanceMetadata{
			LastSyncTime: timestamppb.Now(),
		},
	}
	if instanceMeta.Version != instance.EngineVersion {
		updateInstance.EngineVersion = &instanceMeta.Version
	}
	if !equalInstanceMetadata(instanceMeta.Metadata, instance.Metadata) {
		updateInstance.Metadata.MysqlLowerCaseTableNames = instanceMeta.Metadata.GetMysqlLowerCaseTableNames()
	}
	if _, err := s.store.UpdateInstanceV2(ctx, updateInstance, -1); err != nil {
		return err
	}

	var instanceUsers []*store.InstanceUserMessage
	for _, instanceUser := range instanceMeta.InstanceRoles {
		instanceUsers = append(instanceUsers, &store.InstanceUserMessage{
			Name:  instanceUser.Name,
			Grant: instanceUser.Grant,
		})
	}
	if err := s.store.UpsertInstanceUsers(ctx, instance.UID, instanceUsers); err != nil {
		return err
	}

	databases, err := s.store.ListDatabases(ctx, &store.FindDatabaseMessage{InstanceID: &instance.ResourceID})
	if err != nil {
		return errors.Wrapf(err, "failed to sync database for instance: %s. Failed to find database list", instance.ResourceID)
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
				ProjectID:    api.DefaultProjectID,
			}); err != nil {
				return errors.Wrapf(err, "failed to create instance %q database %q in sync runner", instance.ResourceID, databaseMetadata.Name)
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
				return errors.Errorf("failed to update database %q for instance %q", database.DatabaseName, instance.ResourceID)
			}
		}
	}

	return nil
}

// SyncDatabaseSchema will sync the schema for a database.
func (s *Syncer) SyncDatabaseSchema(ctx context.Context, database *store.DatabaseMessage, force bool) (retErr error) {
	if s.profile.Readonly {
		return nil
	}

	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &database.InstanceID})
	if err != nil {
		return err
	}
	if instance == nil {
		return errors.Errorf("instance %q not found", database.InstanceID)
	}
	driver, err := s.dbFactory.GetAdminDatabaseDriver(ctx, instance, database)
	if err != nil {
		s.upsertDatabaseConnectionAnomaly(ctx, instance, database, err)
		return err
	}
	defer driver.Close(ctx)
	s.upsertDatabaseConnectionAnomaly(ctx, instance, database, nil)
	// Sync database schema
	databaseMetadata, err := driver.SyncDBSchema(ctx)
	if err != nil {
		return err
	}
	setClassificationAndUserCommentFromComment(databaseMetadata)

	var patchSchemaVersion *model.Version
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
		SchemaVersion:        patchSchemaVersion,
		MetadataUpsert: &storepb.DatabaseMetadata{
			LastSyncTime: timestamppb.New(time.Unix(ts, 0)),
		},
	}, api.SystemBotID); err != nil {
		return errors.Wrapf(err, "failed to update database %q for instance %q", database.DatabaseName, database.InstanceID)
	}

	dbSchema, err := s.store.GetDBSchema(ctx, database.UID)
	if err != nil {
		return err
	}
	var oldDatabaseMetadata *storepb.DatabaseSchemaMetadata
	var rawDump []byte
	if dbSchema != nil {
		oldDatabaseMetadata = dbSchema.Metadata
		rawDump = dbSchema.Schema
	}

	if !cmp.Equal(oldDatabaseMetadata, databaseMetadata, protocmp.Transform()) {
		// Avoid updating dump everytime by dumping the schema only when the database metadata is changed.
		// if oldDatabaseMetadata is nil and databaseMetadata is not, they are not equal resulting a sync.
		if force || !equalDatabaseMetadata(oldDatabaseMetadata, databaseMetadata) {
			var schemaBuf bytes.Buffer
			if _, err := driver.Dump(ctx, &schemaBuf, true /* schemaOnly */); err != nil {
				return err
			}
			rawDump = schemaBuf.Bytes()
		}

		if err := s.store.UpsertDBSchema(ctx, database.UID, &store.DBSchema{
			Metadata: databaseMetadata,
			Schema:   rawDump,
		}, api.SystemBotID); err != nil {
			return err
		}
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
				anomalyPayload := api.AnomalyDatabaseSchemaDriftPayload{
					Version: list[0].Version.Version,
					Expect:  list[0].Schema,
					Actual:  latestSchema,
				}
				payload, err := json.Marshal(anomalyPayload)
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

func (s *Syncer) upsertInstanceConnectionAnomaly(ctx context.Context, instance *store.InstanceMessage, connErr error) {
	if connErr != nil {
		anomalyPayload := api.AnomalyInstanceConnectionPayload{
			Detail: connErr.Error(),
		}
		payload, err := json.Marshal(anomalyPayload)
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
		anomalyPayload := api.AnomalyDatabaseConnectionPayload{
			Detail: connErr.Error(),
		}
		payload, err := json.Marshal(anomalyPayload)
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

func (s *Syncer) checkBackupAnomaly(ctx context.Context, environment *store.EnvironmentMessage, instance *store.InstanceMessage, database *store.DatabaseMessage, policy *api.BackupPlanPolicy) {
	if disableBackupAnomalyCheck(instance.Engine) {
		// skip checking backup anomalies for MongoDB, Spanner, Redis, Oracle, etc. because they don't support Backup.
		return
	}

	schedule := api.BackupPlanPolicyScheduleUnset
	backupSetting, err := s.store.GetBackupSettingV2(ctx, database.UID)
	if err != nil {
		slog.Error("Failed to retrieve backup setting",
			slog.String("instance", instance.ResourceID),
			slog.String("database", database.DatabaseName),
			log.BBError(err))
		return
	}

	if backupSetting != nil && backupSetting.Enabled && backupSetting.HourOfDay != -1 {
		if backupSetting.DayOfWeek == -1 {
			schedule = api.BackupPlanPolicyScheduleDaily
		} else {
			schedule = api.BackupPlanPolicyScheduleWeekly
		}
	}

	// Check backup policy violation
	{
		var backupPolicyAnomalyPayload *api.AnomalyDatabaseBackupPolicyViolationPayload
		if policy.Schedule != api.BackupPlanPolicyScheduleUnset {
			if policy.Schedule == api.BackupPlanPolicyScheduleDaily &&
				schedule != api.BackupPlanPolicyScheduleDaily {
				backupPolicyAnomalyPayload = &api.AnomalyDatabaseBackupPolicyViolationPayload{
					EnvironmentID:          environment.UID,
					ExpectedBackupSchedule: policy.Schedule,
					ActualBackupSchedule:   schedule,
				}
			} else if policy.Schedule == api.BackupPlanPolicyScheduleWeekly &&
				schedule == api.BackupPlanPolicyScheduleUnset {
				backupPolicyAnomalyPayload = &api.AnomalyDatabaseBackupPolicyViolationPayload{
					EnvironmentID:          environment.UID,
					ExpectedBackupSchedule: policy.Schedule,
					ActualBackupSchedule:   schedule,
				}
			}
		}

		if backupPolicyAnomalyPayload != nil {
			payload, err := json.Marshal(*backupPolicyAnomalyPayload)
			if err != nil {
				slog.Error("Failed to marshal anomaly payload",
					slog.String("instance", instance.ResourceID),
					slog.String("database", database.DatabaseName),
					slog.String("type", string(api.AnomalyDatabaseBackupPolicyViolation)),
					log.BBError(err))
			} else {
				if _, err = s.store.UpsertActiveAnomalyV2(ctx, api.SystemBotID, &store.AnomalyMessage{
					InstanceID:  instance.ResourceID,
					DatabaseUID: &database.UID,
					Type:        api.AnomalyDatabaseBackupPolicyViolation,
					Payload:     string(payload),
				}); err != nil {
					slog.Error("Failed to create anomaly",
						slog.String("instance", instance.ResourceID),
						slog.String("database", database.DatabaseName),
						slog.String("type", string(api.AnomalyDatabaseBackupPolicyViolation)),
						log.BBError(err))
				}
			}
		} else {
			err := s.store.ArchiveAnomalyV2(ctx, &store.ArchiveAnomalyMessage{
				DatabaseUID: &database.UID,
				Type:        api.AnomalyDatabaseBackupPolicyViolation,
			})
			if err != nil && common.ErrorCode(err) != common.NotFound {
				slog.Error("Failed to close anomaly",
					slog.String("instance", instance.ResourceID),
					slog.String("database", database.DatabaseName),
					slog.String("type", string(api.AnomalyDatabaseBackupPolicyViolation)),
					log.BBError(err))
			}
		}
	}

	// Check backup missing
	{
		var backupMissingAnomalyPayload *api.AnomalyDatabaseBackupMissingPayload
		// The anomaly fires if backup is enabled, however no successful backup has been taken during the period.
		if backupSetting != nil && backupSetting.Enabled {
			expectedSchedule := api.BackupPlanPolicyScheduleWeekly
			backupMaxAge := time.Duration(7*24) * time.Hour
			if backupSetting.DayOfWeek == -1 {
				expectedSchedule = api.BackupPlanPolicyScheduleDaily
				backupMaxAge = time.Duration(24) * time.Hour
			}

			// Ignore if backup setting has been changed after the max age.
			if backupSetting.UpdatedTs < time.Now().Add(-backupMaxAge).Unix() {
				status := api.BackupStatusDone
				backupFind := &store.FindBackupMessage{
					DatabaseUID: &database.UID,
					Status:      &status,
				}
				backupList, err := s.store.ListBackupV2(ctx, backupFind)
				if err != nil {
					slog.Error("Failed to retrieve backup list",
						slog.String("instance", instance.ResourceID),
						slog.String("database", database.DatabaseName),
						log.BBError(err))
				}

				hasValidBackup := false
				if len(backupList) > 0 {
					if backupList[0].UpdatedTs >= time.Now().Add(-backupMaxAge).Unix() {
						hasValidBackup = true
					}
				}

				if !hasValidBackup {
					backupMissingAnomalyPayload = &api.AnomalyDatabaseBackupMissingPayload{
						ExpectedBackupSchedule: expectedSchedule,
					}
					if len(backupList) > 0 {
						backupMissingAnomalyPayload.LastBackupTs = backupList[0].UpdatedTs
					}
				}
			}
		}

		if backupMissingAnomalyPayload != nil {
			payload, err := json.Marshal(*backupMissingAnomalyPayload)
			if err != nil {
				slog.Error("Failed to marshal anomaly payload",
					slog.String("instance", instance.ResourceID),
					slog.String("database", database.DatabaseName),
					slog.String("type", string(api.AnomalyDatabaseBackupMissing)),
					log.BBError(err))
			} else {
				if _, err = s.store.UpsertActiveAnomalyV2(ctx, api.SystemBotID, &store.AnomalyMessage{
					InstanceID:  instance.ResourceID,
					DatabaseUID: &database.UID,
					Type:        api.AnomalyDatabaseBackupMissing,
					Payload:     string(payload),
				}); err != nil {
					slog.Error("Failed to create anomaly",
						slog.String("instance", instance.ResourceID),
						slog.String("database", database.DatabaseName),
						slog.String("type", string(api.AnomalyDatabaseBackupMissing)),
						log.BBError(err))
				}
			}
		} else {
			err := s.store.ArchiveAnomalyV2(ctx, &store.ArchiveAnomalyMessage{
				DatabaseUID: &database.UID,
				Type:        api.AnomalyDatabaseBackupMissing,
			})
			if err != nil && common.ErrorCode(err) != common.NotFound {
				slog.Error("Failed to close anomaly",
					slog.String("instance", instance.ResourceID),
					slog.String("database", database.DatabaseName),
					slog.String("type", string(api.AnomalyDatabaseBackupMissing)),
					log.BBError(err))
			}
		}
	}
}

func equalInstanceMetadata(x, y *storepb.InstanceMetadata) bool {
	return cmp.Equal(x, y, protocmp.Transform(), protocmp.IgnoreFields(&storepb.InstanceMetadata{}, "last_sync_time"))
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
		storepb.Engine_MONGODB:          {},
		storepb.Engine_REDIS:            {},
		storepb.Engine_ORACLE:           {},
		storepb.Engine_OCEANBASE_ORACLE: {},
		storepb.Engine_MSSQL:            {},
		storepb.Engine_REDSHIFT:         {},
	}
	_, ok := m[dbTp]
	return ok
}

func disableBackupAnomalyCheck(dbTp storepb.Engine) bool {
	m := map[storepb.Engine]struct{}{
		storepb.Engine_MONGODB:          {},
		storepb.Engine_SPANNER:          {},
		storepb.Engine_REDIS:            {},
		storepb.Engine_ORACLE:           {},
		storepb.Engine_OCEANBASE_ORACLE: {},
		storepb.Engine_MSSQL:            {},
		storepb.Engine_MARIADB:          {},
		storepb.Engine_REDSHIFT:         {},
	}
	_, ok := m[dbTp]
	return ok
}
