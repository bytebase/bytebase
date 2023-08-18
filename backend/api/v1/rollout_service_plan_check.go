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
	switch config := spec.Config.(type) {
	case *storepb.PlanConfig_Spec_CreateDatabaseConfig:
		// TODO(p0ny): implement
	case *storepb.PlanConfig_Spec_ChangeDatabaseConfig:
		return getPlanCheckRunsForSpecWithChangeDatabaseConfigDatabaseTarget(ctx, s, plan, spec, config)

	case *storepb.PlanConfig_Spec_RestoreDatabaseConfig:
		var planCheckRuns []*store.PlanCheckRunMessage
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
						SheetUid:           0,
						ChangeDatabaseType: storepb.PlanCheckRunConfig_CHANGE_DATABASE_TYPE_UNSPECIFIED,
						Target: &storepb.PlanCheckRunConfig_DatabaseTarget_{
							DatabaseTarget: &storepb.PlanCheckRunConfig_DatabaseTarget{
								InstanceUid:  int32(targetInstance.UID),
								DatabaseName: config.RestoreDatabaseConfig.CreateDatabaseConfig.Database,
							},
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
						SheetUid:           0,
						ChangeDatabaseType: storepb.PlanCheckRunConfig_CHANGE_DATABASE_TYPE_UNSPECIFIED,
						Target: &storepb.PlanCheckRunConfig_DatabaseTarget_{
							DatabaseTarget: &storepb.PlanCheckRunConfig_DatabaseTarget{
								InstanceUid:  int32(instance.UID),
								DatabaseName: database.DatabaseName,
							},
						},
					},
				})
			}
		}
		return planCheckRuns, nil
	default:
		return nil, errors.Errorf("unknown spec config type %T", config)
	}
	return nil, nil
}

func getPlanCheckRunsForSpecWithChangeDatabaseConfigDatabaseTarget(ctx context.Context, s *store.Store, plan *store.PlanMessage, spec *storepb.PlanConfig_Spec, config *storepb.PlanConfig_Spec_ChangeDatabaseConfig) ([]*store.PlanCheckRunMessage, error) {
	configTarget, err := func() (*storepb.PlanCheckRunConfig, error) {
		if instanceID, databaseName, err := common.GetInstanceDatabaseID(config.ChangeDatabaseConfig.Target); err == nil {
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

			return &storepb.PlanCheckRunConfig{
				Target: &storepb.PlanCheckRunConfig_DatabaseTarget_{
					DatabaseTarget: &storepb.PlanCheckRunConfig_DatabaseTarget{
						InstanceUid:  int32(instance.UID),
						DatabaseName: database.DatabaseName,
					},
				},
			}, nil
		}
		if projectID, databaseGroupID, err := common.GetProjectIDDatabaseGroupID(config.ChangeDatabaseConfig.Target); err == nil {
			project, err := s.GetProjectV2(ctx, &store.FindProjectMessage{
				ResourceID: &projectID,
			})
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get project %q", projectID)
			}
			if project == nil {
				return nil, errors.Errorf("project %q not found", projectID)
			}
			databaseGroup, err := s.GetDatabaseGroup(ctx, &store.FindDatabaseGroupMessage{
				ProjectUID: &project.UID,
				ResourceID: &databaseGroupID,
			})
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get database group %q", databaseGroupID)
			}
			if databaseGroup == nil {
				return nil, errors.Errorf("database group %q not found", databaseGroupID)
			}

			return &storepb.PlanCheckRunConfig{
				Target: &storepb.PlanCheckRunConfig_DatabaseGroupTarget_{
					DatabaseGroupTarget: &storepb.PlanCheckRunConfig_DatabaseGroupTarget{
						DatabaseGroupUid: databaseGroup.UID,
					},
				},
			}, nil
		}
		return nil, errors.Errorf("unknown target %q", config.ChangeDatabaseConfig.Target)
	}()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get plan check run config target")
	}

	_, sheetUIDStr, err := common.GetProjectResourceIDSheetID(config.ChangeDatabaseConfig.Sheet)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get sheet id from sheet name %q", config.ChangeDatabaseConfig.Sheet)
	}
	sheetUID, err := strconv.Atoi(sheetUIDStr)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to convert sheet id from %q", sheetUIDStr)
	}

	var planCheckRuns []*store.PlanCheckRunMessage

	planCheckRuns = append(planCheckRuns, &store.PlanCheckRunMessage{
		CreatorUID: api.SystemBotID,
		UpdaterUID: api.SystemBotID,
		PlanUID:    plan.UID,
		Status:     store.PlanCheckRunStatusRunning,
		Type:       store.PlanCheckDatabaseConnect,
		Config: &storepb.PlanCheckRunConfig{
			SheetUid:           0,
			ChangeDatabaseType: storepb.PlanCheckRunConfig_CHANGE_DATABASE_TYPE_UNSPECIFIED,
			Target:             configTarget.Target,
		},
	})
	planCheckRuns = append(planCheckRuns, &store.PlanCheckRunMessage{
		CreatorUID: api.SystemBotID,
		UpdaterUID: api.SystemBotID,
		PlanUID:    plan.UID,
		Status:     store.PlanCheckRunStatusRunning,
		Type:       store.PlanCheckDatabaseStatementType,
		Config: &storepb.PlanCheckRunConfig{
			SheetUid:           int32(sheetUID),
			ChangeDatabaseType: convertToChangeDatabaseType(config.ChangeDatabaseConfig.Type),
			Target:             configTarget.Target,
		},
	})
	planCheckRuns = append(planCheckRuns, &store.PlanCheckRunMessage{
		CreatorUID: api.SystemBotID,
		UpdaterUID: api.SystemBotID,
		PlanUID:    plan.UID,
		Status:     store.PlanCheckRunStatusRunning,
		Type:       store.PlanCheckDatabaseStatementAdvise,
		Config: &storepb.PlanCheckRunConfig{
			SheetUid:           int32(sheetUID),
			ChangeDatabaseType: convertToChangeDatabaseType(config.ChangeDatabaseConfig.Type),
			Target:             configTarget.Target,
		},
	})
	planCheckRuns = append(planCheckRuns, &store.PlanCheckRunMessage{
		CreatorUID: api.SystemBotID,
		UpdaterUID: api.SystemBotID,
		PlanUID:    plan.UID,
		Status:     store.PlanCheckRunStatusRunning,
		Type:       store.PlanCheckDatabaseStatementSummaryReport,
		Config: &storepb.PlanCheckRunConfig{
			SheetUid:           int32(sheetUID),
			ChangeDatabaseType: convertToChangeDatabaseType(config.ChangeDatabaseConfig.Type),
			Target:             configTarget.Target,
		},
	})
	if config.ChangeDatabaseConfig.Type == storepb.PlanConfig_ChangeDatabaseConfig_MIGRATE_GHOST {
		planCheckRuns = append(planCheckRuns, &store.PlanCheckRunMessage{
			CreatorUID: api.SystemBotID,
			UpdaterUID: api.SystemBotID,
			PlanUID:    plan.UID,
			Status:     store.PlanCheckRunStatusRunning,
			Type:       store.PlanCheckDatabaseGhostSync,
			Config: &storepb.PlanCheckRunConfig{
				SheetUid:           int32(sheetUID),
				ChangeDatabaseType: convertToChangeDatabaseType(config.ChangeDatabaseConfig.Type),
				Target:             configTarget.Target,
			},
		})
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

func isStatementReportSupported(dbType db.Type) bool {
	switch dbType {
	case db.Postgres, db.MySQL, db.OceanBase, db.Oracle:
		return true
	default:
		return false
	}
}
