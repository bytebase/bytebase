package v1

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
)

func getPlanCheckRunsFromPlan(project *store.ProjectMessage, plan *store.PlanMessage, databaseGroup *v1pb.DatabaseGroup) ([]*store.PlanCheckRunMessage, error) {
	var planCheckRuns []*store.PlanCheckRunMessage
	for _, spec := range plan.Config.Specs {
		switch config := spec.Config.(type) {
		case *storepb.PlanConfig_Spec_CreateDatabaseConfig:
			// No checks for create database.
		case *storepb.PlanConfig_Spec_ExportDataConfig:
			// No checks export data.
		case *storepb.PlanConfig_Spec_ChangeDatabaseConfig:
			// Skip plan checks for releases.
			if config.ChangeDatabaseConfig.Release != "" {
				continue
			}

			var databases []string
			if len(config.ChangeDatabaseConfig.Targets) == 1 && databaseGroup != nil && config.ChangeDatabaseConfig.Targets[0] == databaseGroup.Name {
				for _, m := range databaseGroup.MatchedDatabases {
					databases = append(databases, m.Name)
				}
			} else {
				databases = config.ChangeDatabaseConfig.Targets
			}

			for _, target := range databases {
				instanceID, databaseName, err := common.GetInstanceDatabaseID(target)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to parse %q", target)
				}
				enableSDL := config.ChangeDatabaseConfig.Type == storepb.PlanConfig_ChangeDatabaseConfig_SDL
				planCheckRuns = append(planCheckRuns, &store.PlanCheckRunMessage{
					PlanUID: plan.UID,
					Status:  store.PlanCheckRunStatusRunning,
					Type:    store.PlanCheckDatabaseStatementAdvise,
					Config: &storepb.PlanCheckRunConfig{
						SheetSha256:       config.ChangeDatabaseConfig.SheetSha256,
						InstanceId:        instanceID,
						DatabaseName:      databaseName,
						EnablePriorBackup: config.ChangeDatabaseConfig.EnablePriorBackup,
						EnableGhost:       config.ChangeDatabaseConfig.EnableGhost,
						EnableSdl:         enableSDL,
					},
				})
				planCheckRuns = append(planCheckRuns, &store.PlanCheckRunMessage{
					PlanUID: plan.UID,
					Status:  store.PlanCheckRunStatusRunning,
					Type:    store.PlanCheckDatabaseStatementSummaryReport,
					Config: &storepb.PlanCheckRunConfig{
						SheetSha256:  config.ChangeDatabaseConfig.SheetSha256,
						InstanceId:   instanceID,
						DatabaseName: databaseName,
						EnableGhost:  config.ChangeDatabaseConfig.EnableGhost,
						EnableSdl:    enableSDL,
					},
				})
				if config.ChangeDatabaseConfig.Type == storepb.PlanConfig_ChangeDatabaseConfig_MIGRATE && config.ChangeDatabaseConfig.EnableGhost {
					planCheckRuns = append(planCheckRuns, &store.PlanCheckRunMessage{
						PlanUID: plan.UID,
						Status:  store.PlanCheckRunStatusRunning,
						Type:    store.PlanCheckDatabaseGhostSync,
						Config: &storepb.PlanCheckRunConfig{
							SheetSha256:  config.ChangeDatabaseConfig.SheetSha256,
							InstanceId:   instanceID,
							DatabaseName: databaseName,
							EnableGhost:  config.ChangeDatabaseConfig.EnableGhost,
							EnableSdl:    enableSDL,
							GhostFlags:   config.ChangeDatabaseConfig.GhostFlags,
						},
					})
				}
			}
		default:
			return nil, errors.Errorf("unknown spec config type %T", config)
		}
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
