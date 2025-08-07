package v1

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
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
			if task.LatestTaskRunStatus == storepb.TaskRun_DONE {
				skippedSpecIDs[task.Payload.GetSpecId()] = struct{}{}
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

	var planCheckRuns []*store.PlanCheckRunMessage
	for _, spec := range plan.Config.Specs {
		if _, ok := skippedSpecIDs[spec.Id]; ok {
			continue
		}
		runs, err := getPlanCheckRunsFromSpec(ctx, s, plan, spec)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get plan check runs for plan")
		}
		planCheckRuns = append(planCheckRuns, runs...)
	}
	if project.Setting.GetCiSamplingSize() > 0 {
		var updatedRuns []*store.PlanCheckRunMessage
		countMap := make(map[string]int32)
		for _, run := range planCheckRuns {
			key := fmt.Sprintf("%s/%d", run.Type, run.Config.GetSheetUid())
			if countMap[key] >= project.Setting.GetCiSamplingSize() {
				continue
			}
			updatedRuns = append(updatedRuns, run)
			countMap[key]++
		}
		planCheckRuns = updatedRuns
	}
	return planCheckRuns, nil
}

func getPlanCheckRunsFromSpec(ctx context.Context, s *store.Store, plan *store.PlanMessage, spec *storepb.PlanConfig_Spec) ([]*store.PlanCheckRunMessage, error) {
	switch config := spec.Config.(type) {
	case *storepb.PlanConfig_Spec_CreateDatabaseConfig:
		// No checks for create database.
	case *storepb.PlanConfig_Spec_ExportDataConfig:
		// No checks export data.
	case *storepb.PlanConfig_Spec_ChangeDatabaseConfig:
		// Filtered using scheduledDatabase for ChangeDatabase specs.
		if len(config.ChangeDatabaseConfig.Targets) == 1 {
			if _, _, err := common.GetProjectIDDatabaseGroupID(config.ChangeDatabaseConfig.Targets[0]); err == nil {
				return getPlanCheckRunsFromChangeDatabaseConfigDatabaseGroupTarget(ctx, s, plan, config.ChangeDatabaseConfig)
			}
		}
		return getPlanCheckRunsFromChangeDatabaseConfigDatabaseTarget(ctx, s, plan, config.ChangeDatabaseConfig)
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
	if len(config.Targets) != 1 {
		return nil, errors.Errorf("change database config with database group target must have exactly one target, but got %d targets", len(config.Targets))
	}
	target := config.Targets[0]

	databaseGroup, err := getDatabaseGroupByName(ctx, s, target, v1pb.DatabaseGroupView_DATABASE_GROUP_VIEW_FULL)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database group %q", target)
	}
	if len(databaseGroup.MatchedDatabases) == 0 {
		return nil, errors.Errorf("no matched databases found in database group %q", target)
	}

	sheetUIDs, err := getSheetUIDsFromChangeDatabaseConfig(ctx, s, config)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get sheets from change database config")
	}
	if len(sheetUIDs) == 0 {
		return nil, errors.Errorf("change database config must have either sheet or release specified, but got neither")
	}

	var planCheckRuns []*store.PlanCheckRunMessage
	for _, matchedDatabase := range databaseGroup.MatchedDatabases {
		for _, sheetUID := range sheetUIDs {
			database, err := getDatabaseMessage(ctx, s, matchedDatabase.Name)
			if err != nil {
				return nil, err
			}
			runs, err := getPlanCheckRunsFromChangeDatabaseConfigForDatabase(ctx, s, plan, config, sheetUID, database)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get plan check runs from spec with change database config for database %q", database.DatabaseName)
			}
			planCheckRuns = append(planCheckRuns, runs...)
		}
	}

	return planCheckRuns, nil
}

func getPlanCheckRunsFromChangeDatabaseConfigDatabaseTarget(ctx context.Context, s *store.Store, plan *store.PlanMessage, config *storepb.PlanConfig_ChangeDatabaseConfig) ([]*store.PlanCheckRunMessage, error) {
	sheetUIDs, err := getSheetUIDsFromChangeDatabaseConfig(ctx, s, config)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get sheets from change database config")
	}
	if len(sheetUIDs) == 0 {
		return nil, errors.Errorf("change database config must have either sheet or release specified, but got neither")
	}

	var checks []*store.PlanCheckRunMessage
	for _, target := range config.Targets {
		database, err := getDatabaseMessage(ctx, s, target)
		if err != nil {
			return nil, err
		}
		if database.Deleted {
			return nil, errors.Errorf("database %q was deleted", target)
		}
		for _, sheetUID := range sheetUIDs {
			v, err := getPlanCheckRunsFromChangeDatabaseConfigForDatabase(ctx, s, plan, config, sheetUID, database)
			if err != nil {
				return nil, err
			}
			checks = append(checks, v...)
		}
	}
	return checks, nil
}

func getPlanCheckRunsFromChangeDatabaseConfigForDatabase(ctx context.Context, s *store.Store, plan *store.PlanMessage, config *storepb.PlanConfig_ChangeDatabaseConfig, sheetUID int, database *store.DatabaseMessage) ([]*store.PlanCheckRunMessage, error) {
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

	planCheckRuns = append(planCheckRuns, &store.PlanCheckRunMessage{
		PlanUID: plan.UID,
		Status:  store.PlanCheckRunStatusRunning,
		Type:    store.PlanCheckDatabaseStatementAdvise,
		Config: &storepb.PlanCheckRunConfig{
			SheetUid:           int32(sheetUID),
			ChangeDatabaseType: convertToChangeDatabaseType(config.Type),
			InstanceId:         instance.ResourceID,
			DatabaseName:       database.DatabaseName,
			EnablePriorBackup:  config.EnablePriorBackup,
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
	default:
	}
	return storepb.PlanCheckRunConfig_CHANGE_DATABASE_TYPE_UNSPECIFIED
}

func getSheetUIDsFromChangeDatabaseConfig(ctx context.Context, s *store.Store, config *storepb.PlanConfig_ChangeDatabaseConfig) ([]int, error) {
	var sheetUIDs []int
	if config.Sheet != "" {
		_, sheetUID, err := common.GetProjectResourceIDSheetUID(config.Sheet)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get sheet id from sheet name %q", config.Sheet)
		}
		sheetUIDs = append(sheetUIDs, sheetUID)
	} else if config.Release != "" {
		_, releaseUID, err := common.GetProjectReleaseUID(config.Release)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get release id from release name %q", config.Release)
		}
		release, err := s.GetRelease(ctx, releaseUID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get release %q", config.Release)
		}
		if release == nil {
			return nil, errors.Errorf("release %q not found", config.Release)
		}
		for _, file := range release.Payload.Files {
			_, sheetUID, err := common.GetProjectResourceIDSheetUID(file.Sheet)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get sheet id from sheet name %q", file.Sheet)
			}
			sheetUIDs = append(sheetUIDs, sheetUID)
		}
	} else {
		return nil, errors.Errorf("change database config must have either sheet or release specified, but got neither")
	}
	return sheetUIDs, nil
}
