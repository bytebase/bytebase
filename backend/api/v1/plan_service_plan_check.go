package v1

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
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
	project, err := s.GetProject(ctx, &store.FindProjectMessage{ResourceID: &plan.ProjectID})
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
			key := fmt.Sprintf("%s/%s", run.Type, run.Config.GetSheetSha256())
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
		// Ghost migrations are not supported for database group targets
		if config.EnableGhost {
			return nil, errors.Errorf("ghost migration is not supported for database group target")
		}
	default:
		return nil, errors.Errorf("unsupported change database config type %q for database group target", config.Type)
	}
	if len(config.Targets) != 1 {
		return nil, errors.Errorf("change database config with database group target must have exactly one target, but got %d targets", len(config.Targets))
	}
	target := config.Targets[0]

	databaseGroup, err := getDatabaseGroupByName(ctx, s, target, v1pb.DatabaseGroupView_DATABASE_GROUP_VIEW_FULL)
	if err != nil {
		// If database group was deleted, skip plan checks for it.
		if connect.CodeOf(err) == connect.CodeNotFound {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "failed to get database group %q", target)
	}
	if len(databaseGroup.MatchedDatabases) == 0 {
		return nil, errors.Errorf("no matched databases found in database group %q", target)
	}

	sheetSha256s := getSheetSha256sFromChangeDatabaseConfig(config)
	if len(sheetSha256s) == 0 {
		return nil, errors.Errorf("change database config must have sheet specified, but got none")
	}

	var planCheckRuns []*store.PlanCheckRunMessage
	for _, matchedDatabase := range databaseGroup.MatchedDatabases {
		for _, sheetSha256 := range sheetSha256s {
			database, err := getDatabaseMessage(ctx, s, matchedDatabase.Name)
			if err != nil {
				return nil, err
			}
			runs, err := getPlanCheckRunsFromChangeDatabaseConfigForDatabase(ctx, s, plan, config, sheetSha256, database)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get plan check runs from spec with change database config for database %q", database.DatabaseName)
			}
			planCheckRuns = append(planCheckRuns, runs...)
		}
	}

	return planCheckRuns, nil
}

func getPlanCheckRunsFromChangeDatabaseConfigDatabaseTarget(ctx context.Context, s *store.Store, plan *store.PlanMessage, config *storepb.PlanConfig_ChangeDatabaseConfig) ([]*store.PlanCheckRunMessage, error) {
	sheetSha256s := getSheetSha256sFromChangeDatabaseConfig(config)
	if len(sheetSha256s) == 0 {
		return nil, errors.Errorf("change database config must have sheet specified, but got none")
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
		for _, sheetSha256 := range sheetSha256s {
			v, err := getPlanCheckRunsFromChangeDatabaseConfigForDatabase(ctx, s, plan, config, sheetSha256, database)
			if err != nil {
				return nil, err
			}
			checks = append(checks, v...)
		}
	}
	return checks, nil
}

func getPlanCheckRunsFromChangeDatabaseConfigForDatabase(ctx context.Context, s *store.Store, plan *store.PlanMessage, config *storepb.PlanConfig_ChangeDatabaseConfig, sheetSha256 string, database *store.DatabaseMessage) ([]*store.PlanCheckRunMessage, error) {
	instance, err := s.GetInstance(ctx, &store.FindInstanceMessage{
		ResourceID: &database.InstanceID,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get instance %q", database.InstanceID)
	}
	if instance == nil {
		return nil, errors.Errorf("instance %q not found", database.InstanceID)
	}

	enableSDL := config.Type == storepb.PlanConfig_ChangeDatabaseConfig_SDL
	var planCheckRuns []*store.PlanCheckRunMessage
	planCheckRuns = append(planCheckRuns, &store.PlanCheckRunMessage{
		PlanUID: plan.UID,
		Status:  store.PlanCheckRunStatusRunning,
		Type:    store.PlanCheckDatabaseConnect,
		Config: &storepb.PlanCheckRunConfig{
			SheetSha256:  sheetSha256,
			InstanceId:   instance.ResourceID,
			DatabaseName: database.DatabaseName,
			EnableGhost:  config.EnableGhost,
			EnableSdl:    enableSDL,
		},
	})

	planCheckRuns = append(planCheckRuns, &store.PlanCheckRunMessage{
		PlanUID: plan.UID,
		Status:  store.PlanCheckRunStatusRunning,
		Type:    store.PlanCheckDatabaseStatementAdvise,
		Config: &storepb.PlanCheckRunConfig{
			SheetSha256:       sheetSha256,
			InstanceId:        instance.ResourceID,
			DatabaseName:      database.DatabaseName,
			EnablePriorBackup: config.EnablePriorBackup,
			EnableGhost:       config.EnableGhost,
			EnableSdl:         enableSDL,
		},
	})
	planCheckRuns = append(planCheckRuns, &store.PlanCheckRunMessage{
		PlanUID: plan.UID,
		Status:  store.PlanCheckRunStatusRunning,
		Type:    store.PlanCheckDatabaseStatementSummaryReport,
		Config: &storepb.PlanCheckRunConfig{
			SheetSha256:  sheetSha256,
			InstanceId:   instance.ResourceID,
			DatabaseName: database.DatabaseName,
			EnableGhost:  config.EnableGhost,
			EnableSdl:    enableSDL,
		},
	})
	if config.Type == storepb.PlanConfig_ChangeDatabaseConfig_MIGRATE && config.EnableGhost {
		planCheckRuns = append(planCheckRuns, &store.PlanCheckRunMessage{
			PlanUID: plan.UID,
			Status:  store.PlanCheckRunStatusRunning,
			Type:    store.PlanCheckDatabaseGhostSync,
			Config: &storepb.PlanCheckRunConfig{
				SheetSha256:  sheetSha256,
				InstanceId:   instance.ResourceID,
				DatabaseName: database.DatabaseName,
				EnableGhost:  config.EnableGhost,
				EnableSdl:    enableSDL,
				GhostFlags:   config.GhostFlags,
			},
		})
	}

	return planCheckRuns, nil
}

func getSheetSha256sFromChangeDatabaseConfig(config *storepb.PlanConfig_ChangeDatabaseConfig) []string {
	var sheetSha256s []string
	if config.SheetSha256 != "" {
		sheetSha256s = append(sheetSha256s, config.SheetSha256)
	}
	return sheetSha256s
}
