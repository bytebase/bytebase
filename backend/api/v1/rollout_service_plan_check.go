package v1

import (
	"context"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func getPlanCheckRunsFromPlan(ctx context.Context, s *store.Store, plan *store.PlanMessage) ([]*store.PlanCheckRunMessage, error) {
	var planCheckRuns []*store.PlanCheckRunMessage
	for _, step := range plan.Config.Steps {
		for _, spec := range step.Specs {
			runs, err := getPlanCheckRunsFromSpec(ctx, s, plan, spec)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get plan check runs for plan")
			}
			planCheckRuns = append(planCheckRuns, runs...)
		}
	}
	return planCheckRuns, nil
}

func getPlanCheckRunsFromSpec(ctx context.Context, s *store.Store, plan *store.PlanMessage, spec *storepb.PlanConfig_Spec) ([]*store.PlanCheckRunMessage, error) {
	switch config := spec.Config.(type) {
	case *storepb.PlanConfig_Spec_CreateDatabaseConfig:
		// TODO(p0ny): implement
	case *storepb.PlanConfig_Spec_ChangeDatabaseConfig:
		if _, _, err := common.GetInstanceDatabaseID(config.ChangeDatabaseConfig.Target); err == nil {
			return getPlanCheckRunsFromChangeDatabaseConfigDatabaseTarget(ctx, s, plan, config.ChangeDatabaseConfig)
		}
		if _, _, err := common.GetProjectIDDatabaseGroupID(config.ChangeDatabaseConfig.Target); err == nil {
			return getPlanCheckRunsFromChangeDatabaseConfigDatabaseGroupTarget(ctx, s, plan, config.ChangeDatabaseConfig)
		}
		if _, _, err := common.GetProjectIDDeploymentConfigID(config.ChangeDatabaseConfig.Target); err == nil {
			return getPlanCheckRunsFromChangeDatabaseConfigDeploymentConfigTarget(ctx, s, plan, config.ChangeDatabaseConfig)
		}

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
						InstanceUid:        int32(targetInstance.UID),
						DatabaseName:       config.RestoreDatabaseConfig.CreateDatabaseConfig.Database,
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
						InstanceUid:        int32(instance.UID),
						DatabaseName:       database.DatabaseName,
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

func getPlanCheckRunsFromChangeDatabaseConfigDatabaseGroupTarget(ctx context.Context, s *store.Store, plan *store.PlanMessage, config *storepb.PlanConfig_ChangeDatabaseConfig) ([]*store.PlanCheckRunMessage, error) {
	switch config.Type {
	case storepb.PlanConfig_ChangeDatabaseConfig_MIGRATE:
	case storepb.PlanConfig_ChangeDatabaseConfig_DATA:
	default:
		return nil, errors.Errorf("unsupported change database config type %q for database group target", config.Type)
	}

	projectID, databaseGroupID, err := common.GetProjectIDDatabaseGroupID(config.Target)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get project id and database group id from target %q", config.Target)
	}

	_, sheetUID, err := common.GetProjectResourceIDSheetUID(config.Sheet)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get sheet id from sheet name %q", config.Sheet)
	}

	project, err := s.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &projectID,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get project %q", projectID)
	}
	if project == nil {
		return nil, errors.Errorf("project %q not found", projectID)
	}
	databaseGroup, err := s.GetDatabaseGroup(ctx, &store.FindDatabaseGroupMessage{ProjectUID: &project.UID, ResourceID: &databaseGroupID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database group %q", databaseGroupID)
	}
	if databaseGroup == nil {
		return nil, errors.Errorf("database group %q not found", databaseGroupID)
	}
	allDatabases, err := s.ListDatabases(ctx, &store.FindDatabaseMessage{ProjectID: &project.ResourceID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list databases for project %q", project.ResourceID)
	}

	matchedDatabases, _, err := utils.GetMatchedAndUnmatchedDatabasesInDatabaseGroup(ctx, databaseGroup, allDatabases)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get matched and unmatched databases in database group %q", databaseGroupID)
	}
	if len(matchedDatabases) == 0 {
		return nil, errors.Errorf("no matched databases found in database group %q", databaseGroupID)
	}

	var planCheckRuns []*store.PlanCheckRunMessage
	for _, database := range matchedDatabases {
		runs, err := getPlanCheckRunsFromChangeDatabaseConfigForDatabase(ctx, s, plan, config, sheetUID, database, &databaseGroup.UID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get plan check runs from spec with change database config for database %q", database.DatabaseName)
		}
		planCheckRuns = append(planCheckRuns, runs...)
	}

	return planCheckRuns, nil
}

func getPlanCheckRunsFromChangeDatabaseConfigDatabaseTarget(ctx context.Context, s *store.Store, plan *store.PlanMessage, config *storepb.PlanConfig_ChangeDatabaseConfig) ([]*store.PlanCheckRunMessage, error) {
	instanceID, databaseName, err := common.GetInstanceDatabaseID(config.Target)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get instance and database from target %q", config.Target)
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

	_, sheetUID, err := common.GetProjectResourceIDSheetUID(config.Sheet)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get sheet id from sheet name %q", config.Sheet)
	}

	return getPlanCheckRunsFromChangeDatabaseConfigForDatabase(ctx, s, plan, config, sheetUID, database, nil)
}

func getPlanCheckRunsFromChangeDatabaseConfigDeploymentConfigTarget(ctx context.Context, s *store.Store, plan *store.PlanMessage, config *storepb.PlanConfig_ChangeDatabaseConfig) ([]*store.PlanCheckRunMessage, error) {
	switch config.Type {
	case storepb.PlanConfig_ChangeDatabaseConfig_BASELINE:
	case storepb.PlanConfig_ChangeDatabaseConfig_MIGRATE:
	case storepb.PlanConfig_ChangeDatabaseConfig_MIGRATE_GHOST:
	case storepb.PlanConfig_ChangeDatabaseConfig_MIGRATE_SDL:
	case storepb.PlanConfig_ChangeDatabaseConfig_DATA:
	default:
		return nil, nil
	}

	projectID, _, err := common.GetProjectIDDeploymentConfigID(config.Target)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get project and deployment id from target %q", config.Target)
	}
	project, err := s.GetProjectV2(ctx, &store.FindProjectMessage{ResourceID: &projectID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get project %q", projectID)
	}
	if project == nil {
		return nil, errors.Errorf("project %q not found", projectID)
	}

	_, sheetUID, err := common.GetProjectResourceIDSheetUID(config.Sheet)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get sheet id from sheet name %q", config.Sheet)
	}

	deploymentConfig, err := s.GetDeploymentConfigV2(ctx, project.UID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get deployment config")
	}
	apiDeploymentConfig, err := deploymentConfig.ToAPIDeploymentConfig()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to convert deployment config to api deployment config")
	}
	deploySchedule, err := api.ValidateAndGetDeploymentSchedule(apiDeploymentConfig.Payload)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to validate and get deployment schedule")
	}
	allDatabases, err := s.ListDatabases(ctx, &store.FindDatabaseMessage{ProjectID: &project.ResourceID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list databases")
	}
	matrix, err := utils.GetDatabaseMatrixFromDeploymentSchedule(deploySchedule, allDatabases)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database matrix from deployment schedule")
	}

	var planCheckRuns []*store.PlanCheckRunMessage
	for _, databases := range matrix {
		for _, database := range databases {
			runs, err := getPlanCheckRunsFromChangeDatabaseConfigForDatabase(ctx, s, plan, config, sheetUID, database, nil)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get plan check runs from spec with change database config for database %q", database.DatabaseName)
			}
			planCheckRuns = append(planCheckRuns, runs...)
		}
	}

	return planCheckRuns, nil
}

func getPlanCheckRunsFromChangeDatabaseConfigForDatabase(ctx context.Context, s *store.Store, plan *store.PlanMessage, config *storepb.PlanConfig_ChangeDatabaseConfig, sheetUID int, database *store.DatabaseMessage, databaseGroupUID *int64) ([]*store.PlanCheckRunMessage, error) {
	instance, err := s.GetInstanceV2(ctx, &store.FindInstanceMessage{
		ResourceID: &database.InstanceID,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get instance %q", database.InstanceID)
	}
	if instance == nil {
		return nil, errors.Errorf("instance %q not found", database.InstanceID)
	}

	var planCheckRuns []*store.PlanCheckRunMessage
	planCheckRuns = append(planCheckRuns, &store.PlanCheckRunMessage{
		CreatorUID: api.SystemBotID,
		UpdaterUID: api.SystemBotID,
		PlanUID:    plan.UID,
		Status:     store.PlanCheckRunStatusRunning,
		Type:       store.PlanCheckDatabaseConnect,
		Config: &storepb.PlanCheckRunConfig{
			SheetUid:           int32(sheetUID),
			ChangeDatabaseType: convertToChangeDatabaseType(config.Type),
			InstanceUid:        int32(instance.UID),
			DatabaseName:       database.DatabaseName,
			DatabaseGroupUid:   nil,
		},
	})

	if config.Type == storepb.PlanConfig_ChangeDatabaseConfig_BASELINE || config.Type == storepb.PlanConfig_ChangeDatabaseConfig_BRANCH {
		return planCheckRuns, nil
	}

	planCheckRuns = append(planCheckRuns, &store.PlanCheckRunMessage{
		CreatorUID: api.SystemBotID,
		UpdaterUID: api.SystemBotID,
		PlanUID:    plan.UID,
		Status:     store.PlanCheckRunStatusRunning,
		Type:       store.PlanCheckDatabaseStatementType,
		Config: &storepb.PlanCheckRunConfig{
			SheetUid:           int32(sheetUID),
			ChangeDatabaseType: convertToChangeDatabaseType(config.Type),
			InstanceUid:        int32(instance.UID),
			DatabaseName:       database.DatabaseName,
			DatabaseGroupUid:   databaseGroupUID,
		},
	})
	preUpdateBackupDetail := (*storepb.PlanCheckRunConfig_PreUpdateBackupDetail)(nil)
	if config.PreUpdateBackupDetail != nil {
		preUpdateBackupDetail = &storepb.PlanCheckRunConfig_PreUpdateBackupDetail{
			Database: config.PreUpdateBackupDetail.Database,
		}
	}
	planCheckRuns = append(planCheckRuns, &store.PlanCheckRunMessage{
		CreatorUID: api.SystemBotID,
		UpdaterUID: api.SystemBotID,
		PlanUID:    plan.UID,
		Status:     store.PlanCheckRunStatusRunning,
		Type:       store.PlanCheckDatabaseStatementAdvise,
		Config: &storepb.PlanCheckRunConfig{
			SheetUid:              int32(sheetUID),
			ChangeDatabaseType:    convertToChangeDatabaseType(config.Type),
			InstanceUid:           int32(instance.UID),
			DatabaseName:          database.DatabaseName,
			DatabaseGroupUid:      databaseGroupUID,
			PreUpdateBackupDetail: preUpdateBackupDetail,
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
			ChangeDatabaseType: convertToChangeDatabaseType(config.Type),
			InstanceUid:        int32(instance.UID),
			DatabaseName:       database.DatabaseName,
			DatabaseGroupUid:   databaseGroupUID,
		},
	})
	if databaseGroupUID == nil && config.Type == storepb.PlanConfig_ChangeDatabaseConfig_MIGRATE_GHOST {
		planCheckRuns = append(planCheckRuns, &store.PlanCheckRunMessage{
			CreatorUID: api.SystemBotID,
			UpdaterUID: api.SystemBotID,
			PlanUID:    plan.UID,
			Status:     store.PlanCheckRunStatusRunning,
			Type:       store.PlanCheckDatabaseGhostSync,
			Config: &storepb.PlanCheckRunConfig{
				SheetUid:           int32(sheetUID),
				ChangeDatabaseType: convertToChangeDatabaseType(config.Type),
				InstanceUid:        int32(instance.UID),
				DatabaseName:       database.DatabaseName,
				DatabaseGroupUid:   databaseGroupUID,
				GhostFlags:         config.GhostFlags,
			},
		})
	}

	return planCheckRuns, nil
}

func convertToChangeDatabaseType(t storepb.PlanConfig_ChangeDatabaseConfig_Type) storepb.PlanCheckRunConfig_ChangeDatabaseType {
	switch t {
	case storepb.PlanConfig_ChangeDatabaseConfig_MIGRATE:
		return storepb.PlanCheckRunConfig_DDL
	case storepb.PlanConfig_ChangeDatabaseConfig_MIGRATE_GHOST:
		return storepb.PlanCheckRunConfig_DDL_GHOST
	case storepb.PlanConfig_ChangeDatabaseConfig_MIGRATE_SDL:
		return storepb.PlanCheckRunConfig_SDL
	case storepb.PlanConfig_ChangeDatabaseConfig_DATA:
		return storepb.PlanCheckRunConfig_DML
	}
	return storepb.PlanCheckRunConfig_CHANGE_DATABASE_TYPE_UNSPECIFIED
}
