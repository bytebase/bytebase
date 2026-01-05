package plancheck

import (
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
)

// DeriveCheckTargets derives check targets from a plan and optional database group.
// This replaces the stored config by computing targets at runtime.
func DeriveCheckTargets(project *store.ProjectMessage, plan *store.PlanMessage, databaseGroup *v1pb.DatabaseGroup) ([]*CheckTarget, error) {
	var targets []*CheckTarget

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

			enableGhost := config.ChangeDatabaseConfig.EnableGhost

			for _, target := range databases {
				types := []storepb.PlanCheckType{
					storepb.PlanCheckType_PLAN_CHECK_TYPE_STATEMENT_ADVISE,
					storepb.PlanCheckType_PLAN_CHECK_TYPE_STATEMENT_SUMMARY_REPORT,
				}
				if enableGhost {
					types = append(types, storepb.PlanCheckType_PLAN_CHECK_TYPE_GHOST_SYNC)
				}

				targets = append(targets, &CheckTarget{
					Target:            target,
					SheetSha256:       config.ChangeDatabaseConfig.SheetSha256,
					EnablePriorBackup: config.ChangeDatabaseConfig.EnablePriorBackup,
					EnableGhost:       config.ChangeDatabaseConfig.EnableGhost,
					GhostFlags:        config.ChangeDatabaseConfig.GhostFlags,
					Types:             types,
				})
			}
		default:
			return nil, errors.Errorf("unknown spec config type %T", config)
		}
	}

	return targets, nil
}
