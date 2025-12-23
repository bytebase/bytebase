package v1

import (
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
)

// getPlanCheckRunFromPlan returns a single consolidated plan check run for a plan.
func getPlanCheckRunFromPlan(project *store.ProjectMessage, plan *store.PlanMessage, databaseGroup *v1pb.DatabaseGroup) (*store.PlanCheckRunMessage, error) {
	var targets []*storepb.PlanCheckRunConfig_CheckTarget

	for _, spec := range plan.Config.Specs {
		switch config := spec.Config.(type) {
		case *storepb.PlanConfig_Spec_CreateDatabaseConfig:
			// No checks for create database.
		case *storepb.PlanConfig_Spec_ExportDataConfig:
			// No checks for export data.
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

			// Apply sampling upfront
			if samplingSize := project.Setting.GetCiSamplingSize(); samplingSize > 0 {
				if len(databases) > int(samplingSize) {
					databases = databases[:samplingSize]
				}
			}

			enableSDL := config.ChangeDatabaseConfig.Type == storepb.PlanConfig_ChangeDatabaseConfig_SDL
			enableGhost := config.ChangeDatabaseConfig.Type == storepb.PlanConfig_ChangeDatabaseConfig_MIGRATE && config.ChangeDatabaseConfig.EnableGhost

			for _, target := range databases {
				instanceID, databaseName, err := common.GetInstanceDatabaseID(target)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to parse %q", target)
				}

				checkTypes := []storepb.PlanCheckType{
					storepb.PlanCheckType_PLAN_CHECK_TYPE_STATEMENT_ADVISE,
					storepb.PlanCheckType_PLAN_CHECK_TYPE_STATEMENT_SUMMARY_REPORT,
				}
				if enableGhost {
					checkTypes = append(checkTypes, storepb.PlanCheckType_PLAN_CHECK_TYPE_GHOST_SYNC)
				}

				targets = append(targets, &storepb.PlanCheckRunConfig_CheckTarget{
					InstanceId:        instanceID,
					DatabaseName:      databaseName,
					SheetSha256:       config.ChangeDatabaseConfig.SheetSha256,
					EnablePriorBackup: config.ChangeDatabaseConfig.EnablePriorBackup,
					EnableGhost:       config.ChangeDatabaseConfig.EnableGhost,
					EnableSdl:         enableSDL,
					GhostFlags:        config.ChangeDatabaseConfig.GhostFlags,
					CheckTypes:        checkTypes,
				})
			}
		default:
			return nil, errors.Errorf("unknown spec config type %T", config)
		}
	}

	if len(targets) == 0 {
		return nil, nil
	}

	return &store.PlanCheckRunMessage{
		PlanUID: plan.UID,
		Status:  store.PlanCheckRunStatusRunning,
		Config:  &storepb.PlanCheckRunConfig{Targets: targets},
	}, nil
}
