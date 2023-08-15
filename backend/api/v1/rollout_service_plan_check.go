package v1

import (
	"context"
	"strconv"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func getPlanCheckRunsForPlan(ctx context.Context, s *store.Store, plan *store.PlanMessage) ([]*store.PlanCheckRunMessage, error) {
	var planCheckRuns []*store.PlanCheckRunMessage
	for _, step := range plan.Config.Steps {
		for _, spec := range step.Specs {
			runs, err := getPlanCheckRunsForSpec(ctx, s, plan, spec)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get plan check runs for plan")
			}
			planCheckRuns = append(planCheckRuns, runs...)
		}
	}
	return planCheckRuns, nil
}

func getPlanCheckRunsForSpec(ctx context.Context, s *store.Store, plan *store.PlanMessage, spec *storepb.PlanConfig_Spec) ([]*store.PlanCheckRunMessage, error) {
	var planCheckRuns []*store.PlanCheckRunMessage
	switch config := spec.Config.(type) {
	case *storepb.PlanConfig_Spec_CreateDatabaseConfig:
		// TODO(p0ny): implement
	case *storepb.PlanConfig_Spec_ChangeDatabaseConfig:
		target := config.ChangeDatabaseConfig.Target

		instanceID, databaseName, err := common.GetInstanceDatabaseID(target)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get instance database id from target %q", target)
		}

		_, sheetUIDStr, err := common.GetProjectResourceIDSheetID(config.ChangeDatabaseConfig.Sheet)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get sheet id from sheet name %q", config.ChangeDatabaseConfig.Sheet)
		}
		sheetUID, err := strconv.Atoi(sheetUIDStr)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert sheet id from %q", sheetUIDStr)
		}

		instance, err := s.GetInstanceV2(ctx, &store.FindInstanceMessage{
			ResourceID: &instanceID,
		})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get instance %q", instanceID)
		}
		if instance == nil {
			return nil, errors.Errorf("instance %q not found", instanceID)
		}
		database, err := s.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
			InstanceID:          &instanceID,
			DatabaseName:        &databaseName,
			IgnoreCaseSensitive: store.IgnoreDatabaseAndTableCaseSensitive(instance),
		})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get database %q", databaseName)
		}
		if database == nil {
			return nil, errors.Errorf("database %q not found", databaseName)
		}
		planCheckRuns = append(planCheckRuns, &store.PlanCheckRunMessage{
			CreatorUID: api.SystemBotID,
			UpdaterUID: api.SystemBotID,
			PlanUID:    plan.UID,
			Status:     store.PlanCheckRunStatusRunning,
			Type:       store.PlanCheckDatabaseConnect,
			Config: &storepb.PlanCheckRunConfig{
				SheetId:    0,
				InstanceId: int32(instance.UID),
				DatabaseId: int32(database.UID),
			},
		})
		if isStatementTypeCheckSupported(instance.Engine) {
			planCheckRuns = append(planCheckRuns, &store.PlanCheckRunMessage{
				CreatorUID: api.SystemBotID,
				UpdaterUID: api.SystemBotID,
				PlanUID:    plan.UID,
				Status:     store.PlanCheckRunStatusRunning,
				Type:       store.PlanCheckDatabaseStatementType,
				Config: &storepb.PlanCheckRunConfig{
					SheetId:            int32(sheetUID),
					InstanceId:         int32(instance.UID),
					DatabaseId:         int32(database.UID),
					ChangeDatabaseType: convertToChangeDatabaseType(config.ChangeDatabaseConfig.Type),
				},
			})
		}
		if isStatementAdviseSupported(instance.Engine) {
			planCheckRuns = append(planCheckRuns, &store.PlanCheckRunMessage{
				CreatorUID: api.SystemBotID,
				UpdaterUID: api.SystemBotID,
				PlanUID:    plan.UID,
				Status:     store.PlanCheckRunStatusRunning,
				Type:       store.PlanCheckDatabaseStatementAdvise,
				Config: &storepb.PlanCheckRunConfig{
					SheetId:            int32(sheetUID),
					InstanceId:         int32(instance.UID),
					DatabaseId:         int32(database.UID),
					ChangeDatabaseType: convertToChangeDatabaseType(config.ChangeDatabaseConfig.Type),
				},
			})
		}
		if config.ChangeDatabaseConfig.Type == storepb.PlanConfig_ChangeDatabaseConfig_MIGRATE_GHOST {
			planCheckRuns = append(planCheckRuns, &store.PlanCheckRunMessage{
				CreatorUID: api.SystemBotID,
				UpdaterUID: api.SystemBotID,
				PlanUID:    plan.UID,
				Status:     store.PlanCheckRunStatusRunning,
				Type:       store.PlanCheckDatabaseGhostSync,
				Config: &storepb.PlanCheckRunConfig{
					SheetId:            int32(sheetUID),
					InstanceId:         int32(instance.UID),
					DatabaseId:         int32(database.UID),
					ChangeDatabaseType: convertToChangeDatabaseType(config.ChangeDatabaseConfig.Type),
				},
			})
		}
	case *storepb.PlanConfig_Spec_RestoreDatabaseConfig:
		// mysql PITR check
		if _, ok := config.RestoreDatabaseConfig.Source.(*storepb.PlanConfig_RestoreDatabaseConfig_PointInTime); ok {
			if config.RestoreDatabaseConfig.CreateDatabaseConfig != nil {
				// Restore to a new database
				// check target instance
				targetInstanceID, err := common.GetInstanceID(config.RestoreDatabaseConfig.CreateDatabaseConfig.Target)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to get instance id from %q", config.RestoreDatabaseConfig.CreateDatabaseConfig.Target)
				}
				targetInstance, err := s.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &targetInstanceID})
				if err != nil {
					return nil, errors.Wrapf(err, "failed to find the instance with ID %q", targetInstanceID)
				}
				if targetInstance == nil {
					return nil, errors.Errorf("instance %q not found", targetInstanceID)
				}

				planCheckRuns = append(planCheckRuns, &store.PlanCheckRunMessage{
					CreatorUID: api.SystemBotID,
					UpdaterUID: api.SystemBotID,
					PlanUID:    plan.UID,
					Status:     store.PlanCheckRunStatusRunning,
					Type:       store.PlanCheckDatabasePITRMySQL,
					Config: &storepb.PlanCheckRunConfig{
						InstanceId: int32(targetInstance.UID),
						PitrConfig: &storepb.PlanCheckRunConfig_PitrConfig{
							TargetInstanceId:   int32(targetInstance.UID),
							TargetDatabaseName: config.RestoreDatabaseConfig.CreateDatabaseConfig.Database,
						},
					},
				})
			} else {
				// in-place restore
				// check instance
				target := config.RestoreDatabaseConfig.Target
				instanceID, databaseName, err := common.GetInstanceDatabaseID(target)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to get instance database id from target %q", target)
				}

				instance, err := s.GetInstanceV2(ctx, &store.FindInstanceMessage{
					ResourceID: &instanceID,
				})
				if err != nil {
					return nil, errors.Wrapf(err, "failed to get instance %q", instanceID)
				}
				if instance == nil {
					return nil, errors.Errorf("instance %q not found", instanceID)
				}
				database, err := s.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
					InstanceID:          &instanceID,
					DatabaseName:        &databaseName,
					IgnoreCaseSensitive: store.IgnoreDatabaseAndTableCaseSensitive(instance),
				})
				if err != nil {
					return nil, errors.Wrapf(err, "failed to get database %q", databaseName)
				}
				if database == nil {
					return nil, errors.Errorf("database %q not found", databaseName)
				}

				planCheckRuns = append(planCheckRuns, &store.PlanCheckRunMessage{
					CreatorUID: api.SystemBotID,
					UpdaterUID: api.SystemBotID,
					PlanUID:    plan.UID,
					Status:     store.PlanCheckRunStatusRunning,
					Type:       store.PlanCheckDatabasePITRMySQL,
					Config: &storepb.PlanCheckRunConfig{
						InstanceId: int32(instance.UID),
						DatabaseId: int32(database.UID),
					},
				})
			}
		}
	}
	return planCheckRuns, nil
}

func convertToChangeDatabaseType(t storepb.PlanConfig_ChangeDatabaseConfig_Type) storepb.PlanCheckRunConfig_ChangeDatabaseType {
	switch t {
	case
		storepb.PlanConfig_ChangeDatabaseConfig_MIGRATE,
		storepb.PlanConfig_ChangeDatabaseConfig_MIGRATE_GHOST:
		return storepb.PlanCheckRunConfig_DDL
	case storepb.PlanConfig_ChangeDatabaseConfig_MIGRATE_SDL:
		return storepb.PlanCheckRunConfig_SDL
	case storepb.PlanConfig_ChangeDatabaseConfig_DATA:
		return storepb.PlanCheckRunConfig_DML
	}
	return storepb.PlanCheckRunConfig_CHANGE_DATABASE_TYPE_UNSPECIFIED
}

func isStatementTypeCheckSupported(dbType db.Type) bool {
	switch dbType {
	case db.Postgres, db.TiDB, db.MySQL, db.MariaDB, db.OceanBase:
		return true
	default:
		return false
	}
}

func isStatementAdviseSupported(dbType db.Type) bool {
	switch dbType {
	case db.MySQL, db.TiDB, db.Postgres, db.Oracle, db.OceanBase, db.Snowflake, db.MSSQL:
		return true
	default:
		return false
	}
}
