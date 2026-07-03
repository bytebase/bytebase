package plancheck

import (
	"context"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
)

// GetDatabaseGroupForPlan checks if the plan targets a database group and returns it with matched databases.
// Returns nil if the plan does not target a database group.
// allDatabases must be nil or the full unfiltered database list for the plan's project;
// a filtered list would silently make the group expansion incomplete.
func GetDatabaseGroupForPlan(ctx context.Context, stores *store.Store, plan *store.PlanMessage, allDatabases []*store.DatabaseMessage) (*v1pb.DatabaseGroup, error) {
	target, databaseGroupID, ok := findDatabaseGroupTarget(plan.Config.GetSpecs())
	if !ok {
		return nil, nil
	}

	dbGroup, err := stores.GetDatabaseGroup(ctx, &store.FindDatabaseGroupMessage{
		ResourceID: &databaseGroupID,
		ProjectIDs: []string{plan.ProjectID},
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database group %q", target)
	}
	if dbGroup == nil {
		return nil, errors.Errorf("database group %q not found", target)
	}

	if allDatabases == nil {
		allDatabases, err = stores.ListDatabases(ctx, &store.FindDatabaseMessage{ProjectID: &plan.ProjectID})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to list databases for project %q", plan.ProjectID)
		}
	}

	matchedDatabases, err := utils.GetMatchedDatabasesInDatabaseGroup(ctx, dbGroup, allDatabases)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get matched databases for group %q", databaseGroupID)
	}

	result := &v1pb.DatabaseGroup{Name: target}
	for _, db := range matchedDatabases {
		result.MatchedDatabases = append(result.MatchedDatabases, &v1pb.DatabaseGroup_Database{
			Name: common.FormatDatabase(db.InstanceID, db.DatabaseName),
		})
	}
	return result, nil
}

// HasDatabaseGroupTarget reports whether any change-database spec targets a database group.
// It only parses target resource names and does not touch the store.
func HasDatabaseGroupTarget(specs []*storepb.PlanConfig_Spec) bool {
	_, _, ok := findDatabaseGroupTarget(specs)
	return ok
}

func findDatabaseGroupTarget(specs []*storepb.PlanConfig_Spec) (string, string, bool) {
	for _, spec := range specs {
		cfg := spec.GetChangeDatabaseConfig()
		if cfg == nil || len(cfg.Targets) != 1 {
			continue
		}

		target := cfg.Targets[0]
		_, databaseGroupID, err := common.GetProjectIDDatabaseGroupID(target)
		if err != nil {
			continue
		}
		return target, databaseGroupID, true
	}
	return "", "", false
}
