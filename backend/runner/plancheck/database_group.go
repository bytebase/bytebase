package plancheck

import (
	"context"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
)

// GetDatabaseGroupForPlan checks if the plan targets a database group and returns it with matched databases.
// Returns nil if the plan does not target a database group.
func GetDatabaseGroupForPlan(ctx context.Context, stores *store.Store, plan *store.PlanMessage, allDatabases []*store.DatabaseMessage) (*v1pb.DatabaseGroup, error) {
	for _, spec := range plan.Config.GetSpecs() {
		cfg := spec.GetChangeDatabaseConfig()
		if cfg == nil {
			continue
		}
		if len(cfg.Targets) != 1 {
			continue
		}

		target := cfg.Targets[0]
		_, databaseGroupID, err := common.GetProjectIDDatabaseGroupID(target)
		if err != nil {
			continue
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
	return nil, nil
}
