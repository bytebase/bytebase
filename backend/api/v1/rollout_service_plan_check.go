package v1

import (
	"context"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
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
	var planCheckRuns []*store.PlanCheckRunMessage
	switch config := spec.Config.(type) {
	case *storepb.PlanConfig_Spec_CreateDatabaseConfig:
		// TODO(p0ny): implement
	case *storepb.PlanConfig_Spec_ChangeDatabaseConfig:
		target := config.ChangeDatabaseConfig.Target
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
			Type:       store.PlanCheckDatabaseConnect,
			Config: &storepb.PlanCheckRunConfig{
				SheetId:    0,
				DatabaseId: int32(database.UID),
			},
		})
	case *storepb.PlanConfig_Spec_RestoreDatabaseConfig:
		// TODO(p0ny): implement
	}
	return planCheckRuns, nil
}
