package v1

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func getPlanCheckRunsFromPlan(ctx context.Context, s *store.Store, plan *store.PlanMessage) ([]*store.PlanCheckRunMessage, error) {
	var skippedSpecIDs map[string]struct{}
	if plan.PipelineUID != nil {
		tasks, err := s.ListTasks(ctx, &store.TaskFind{PipelineID: plan.PipelineUID})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get tasks for pipeline %d", *plan.PipelineUID)
		}
		skippedSpecIDs = make(map[string]struct{})
		for _, task := range tasks {
			var taskSpecID struct {
				SpecID string `json:"specId"`
			}
			if err := json.Unmarshal([]byte(task.Payload), &taskSpecID); err != nil {
				return nil, errors.Wrapf(err, "failed to unmarshal task payload")
			}
			if task.LatestTaskRunStatus == api.TaskRunDone {
				skippedSpecIDs[taskSpecID.SpecID] = struct{}{}
			}
		}
	}
	return getPlanCheckRunsFromPlanSpecs(ctx, s, plan, skippedSpecIDs)
}

func getPlanCheckRunsFromPlanSpecs(ctx context.Context, s *store.Store, plan *store.PlanMessage, skippedSpecIDs map[string]struct{}) ([]*store.PlanCheckRunMessage, error) {
	project, err := s.GetProjectV2(ctx, &store.FindProjectMessage{ResourceID: &plan.ProjectID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get project %v", plan.ProjectID)
	}
	if project == nil {
		return nil, errors.Errorf("project %v not found", plan.ProjectID)
	}

	deploymentConfig, err := s.GetDeploymentConfigV2(ctx, project.ResourceID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get deployment config")
	}
	if err := utils.ValidateDeploymentSchedule(deploymentConfig.Config.GetSchedule()); err != nil {
		return nil, errors.Wrapf(err, "failed to validate and get deployment schedule")
	}

	allDatabases, err := s.ListDatabases(ctx, &store.FindDatabaseMessage{ProjectID: &project.ResourceID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list databases for project %q", project.ResourceID)
	}
	matrix, err := utils.GetDatabaseMatrixFromDeploymentSchedule(deploymentConfig.Config.GetSchedule(), allDatabases)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database matrix from deployment schedule")
	}

	scheduledDatabase := map[string]bool{}
	for _, dbs := range matrix {
		for _, db := range dbs {
			scheduledDatabase[db.String()] = true
		}
	}

	var planCheckRuns []*store.PlanCheckRunMessage
	for _, step := range plan.Config.Steps {
		for _, spec := range step.Specs {
			if _, ok := skippedSpecIDs[spec.Id]; ok {
				continue
			}
			runs, err := getPlanCheckRunsFromSpec(ctx, s, plan, spec, scheduledDatabase)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get plan check runs for plan")
			}
			planCheckRuns = append(planCheckRuns, runs...)
		}
	}
	return planCheckRuns, nil
}

func getPlanCheckRunsFromSpec(ctx context.Context, s *store.Store, plan *store.PlanMessage, spec *storepb.PlanConfig_Spec, scheduledDatabase map[string]bool) ([]*store.PlanCheckRunMessage, error) {
	switch config := spec.Config.(type) {
	case *storepb.PlanConfig_Spec_CreateDatabaseConfig:
		// TODO(p0ny): implement
	case *storepb.PlanConfig_Spec_ChangeDatabaseConfig:
		// Filtered using scheduledDatabase for ChangeDatabase specs.
		if _, _, err := common.GetInstanceDatabaseID(config.ChangeDatabaseConfig.Target); err == nil {
			return getPlanCheckRunsFromChangeDatabaseConfigDatabaseTarget(ctx, s, plan, config.ChangeDatabaseConfig, scheduledDatabase)
		}
		if _, _, err := common.GetProjectIDDatabaseGroupID(config.ChangeDatabaseConfig.Target); err == nil {
			return getPlanCheckRunsFromChangeDatabaseConfigDatabaseGroupTarget(ctx, s, plan, config.ChangeDatabaseConfig, scheduledDatabase)
		}
	case *storepb.PlanConfig_Spec_ExportDataConfig:
		if _, _, err := common.GetInstanceDatabaseID(config.ExportDataConfig.Target); err == nil {
			return getPlanCheckRunsFromExportDataConfigDatabaseTarget(ctx, s, plan, config.ExportDataConfig)
		}
	default:
		return nil, errors.Errorf("unknown spec config type %T", config)
	}
	return nil, nil
}

func getPlanCheckRunsFromChangeDatabaseConfigDatabaseGroupTarget(ctx context.Context, s *store.Store, plan *store.PlanMessage, config *storepb.PlanConfig_ChangeDatabaseConfig, scheduledDatabase map[string]bool) ([]*store.PlanCheckRunMessage, error) {
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
	databaseGroup, err := s.GetDatabaseGroup(ctx, &store.FindDatabaseGroupMessage{ProjectID: &project.ResourceID, ResourceID: &databaseGroupID})
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
		runs, err := getPlanCheckRunsFromChangeDatabaseConfigForDatabase(ctx, s, plan, config, sheetUID, database, scheduledDatabase)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get plan check runs from spec with change database config for database %q", database.DatabaseName)
		}
		planCheckRuns = append(planCheckRuns, runs...)
	}

	return planCheckRuns, nil
}

func getPlanCheckRunsFromChangeDatabaseConfigDatabaseTarget(ctx context.Context, s *store.Store, plan *store.PlanMessage, config *storepb.PlanConfig_ChangeDatabaseConfig, scheduledDatabase map[string]bool) ([]*store.PlanCheckRunMessage, error) {
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
		InstanceID:      &instanceID,
		DatabaseName:    &databaseName,
		IsCaseSensitive: store.IsObjectCaseSensitive(instance),
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

	return getPlanCheckRunsFromChangeDatabaseConfigForDatabase(ctx, s, plan, config, sheetUID, database, scheduledDatabase)
}

func getPlanCheckRunsFromChangeDatabaseConfigForDatabase(ctx context.Context, s *store.Store, plan *store.PlanMessage, config *storepb.PlanConfig_ChangeDatabaseConfig, sheetUID int, database *store.DatabaseMessage, scheduledDatabase map[string]bool) ([]*store.PlanCheckRunMessage, error) {
	if !scheduledDatabase[database.String()] {
		return nil, nil
	}

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
		PlanUID: plan.UID,
		Status:  store.PlanCheckRunStatusRunning,
		Type:    store.PlanCheckDatabaseConnect,
		Config: &storepb.PlanCheckRunConfig{
			SheetUid:           int32(sheetUID),
			ChangeDatabaseType: convertToChangeDatabaseType(config.Type),
			InstanceId:         instance.ResourceID,
			DatabaseName:       database.DatabaseName,
		},
	})

	if config.Type == storepb.PlanConfig_ChangeDatabaseConfig_BASELINE {
		return planCheckRuns, nil
	}

	preUpdateBackupDetail := (*storepb.PreUpdateBackupDetail)(nil)
	if config.PreUpdateBackupDetail != nil {
		preUpdateBackupDetail = &storepb.PreUpdateBackupDetail{
			Database: config.PreUpdateBackupDetail.Database,
		}
	}
	planCheckRuns = append(planCheckRuns, &store.PlanCheckRunMessage{
		PlanUID: plan.UID,
		Status:  store.PlanCheckRunStatusRunning,
		Type:    store.PlanCheckDatabaseStatementAdvise,
		Config: &storepb.PlanCheckRunConfig{
			SheetUid:              int32(sheetUID),
			ChangeDatabaseType:    convertToChangeDatabaseType(config.Type),
			InstanceId:            instance.ResourceID,
			DatabaseName:          database.DatabaseName,
			PreUpdateBackupDetail: preUpdateBackupDetail,
		},
	})
	planCheckRuns = append(planCheckRuns, &store.PlanCheckRunMessage{
		PlanUID: plan.UID,
		Status:  store.PlanCheckRunStatusRunning,
		Type:    store.PlanCheckDatabaseStatementSummaryReport,
		Config: &storepb.PlanCheckRunConfig{
			SheetUid:           int32(sheetUID),
			ChangeDatabaseType: convertToChangeDatabaseType(config.Type),
			InstanceId:         instance.ResourceID,
			DatabaseName:       database.DatabaseName,
		},
	})
	if config.Type == storepb.PlanConfig_ChangeDatabaseConfig_MIGRATE_GHOST {
		planCheckRuns = append(planCheckRuns, &store.PlanCheckRunMessage{
			PlanUID: plan.UID,
			Status:  store.PlanCheckRunStatusRunning,
			Type:    store.PlanCheckDatabaseGhostSync,
			Config: &storepb.PlanCheckRunConfig{
				SheetUid:           int32(sheetUID),
				ChangeDatabaseType: convertToChangeDatabaseType(config.Type),
				InstanceId:         instance.ResourceID,
				DatabaseName:       database.DatabaseName,
				GhostFlags:         config.GhostFlags,
			},
		})
	}

	return planCheckRuns, nil
}

func getPlanCheckRunsFromExportDataConfigDatabaseTarget(ctx context.Context, s *store.Store, plan *store.PlanMessage, config *storepb.PlanConfig_ExportDataConfig) ([]*store.PlanCheckRunMessage, error) {
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
		InstanceID:      &instanceID,
		DatabaseName:    &databaseName,
		IsCaseSensitive: store.IsObjectCaseSensitive(instance),
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

	return getPlanCheckRunsFromExportDataConfigForDatabase(ctx, s, plan, config, sheetUID, database)
}

func getPlanCheckRunsFromExportDataConfigForDatabase(ctx context.Context, s *store.Store, plan *store.PlanMessage, _ *storepb.PlanConfig_ExportDataConfig, sheetUID int, database *store.DatabaseMessage) ([]*store.PlanCheckRunMessage, error) {
	instance, err := s.GetInstanceV2(ctx, &store.FindInstanceMessage{
		ResourceID: &database.InstanceID,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get instance %q", database.InstanceID)
	}
	if instance == nil {
		return nil, errors.Errorf("instance %q not found", database.InstanceID)
	}

	planCheckRunTypes := []store.PlanCheckRunType{
		store.PlanCheckDatabaseConnect,
		store.PlanCheckDatabaseStatementAdvise,
	}
	planCheckRuns := []*store.PlanCheckRunMessage{}
	for _, planCheckRunType := range planCheckRunTypes {
		planCheckRuns = append(planCheckRuns, &store.PlanCheckRunMessage{
			PlanUID: plan.UID,
			Status:  store.PlanCheckRunStatusRunning,
			Type:    planCheckRunType,
			Config: &storepb.PlanCheckRunConfig{
				SheetUid:           int32(sheetUID),
				ChangeDatabaseType: storepb.PlanCheckRunConfig_DML,
				InstanceId:         instance.ResourceID,
				DatabaseName:       database.DatabaseName,
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
