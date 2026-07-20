package store_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

func TestCreatePlanCheckRunDoesNotDeadlockWithProjectDeletion(t *testing.T) {
	err := runWithConcurrentProjectDeletion(t, `
		INSERT INTO plan (id, creator, project, name, description) VALUES (101, 'creator@example.com', 'project-a', 'Plan A', '');
		INSERT INTO plan_check_run (id, project, plan_id, status) VALUES (101, 'project-a', 101, 'FAILED');
	`, "plan_check_run", 9902, func(ctx context.Context, s *store.Store) error {
		_, err := s.CreatePlanCheckRun(ctx, &store.PlanCheckRunMessage{
			ProjectID: "project-a",
			PlanUID:   101,
			Result:    &storepb.PlanCheckRunResult{},
		})
		return err
	})
	require.NoError(t, err)
}

func TestCreateFirstPlanCheckRunStopsAfterProjectDeletion(t *testing.T) {
	var created bool
	err := runWithConcurrentProjectDeletion(t, `
		INSERT INTO plan (id, creator, project, name, description) VALUES (101, 'creator@example.com', 'project-a', 'Plan A', '');
	`, "plan", 9904, func(ctx context.Context, s *store.Store) error {
		var err error
		created, err = s.CreatePlanCheckRun(ctx, &store.PlanCheckRunMessage{
			ProjectID: "project-a",
			PlanUID:   101,
			Result:    &storepb.PlanCheckRunResult{},
		})
		return err
	})
	require.NoError(t, err)
	require.False(t, created)
}

func TestCreatePlanCheckRunRejectsDeletedProjectDuringProjectDeletion(t *testing.T) {
	operationErr, deleteErr := runWithCreationBeforeProjectDeletion(t, `
		INSERT INTO plan (id, creator, project, name, description) VALUES (101, 'creator@example.com', 'project-a', 'Plan A', '');
	`, "plan", func(ctx context.Context, s *store.Store) error {
		_, err := s.CreatePlanCheckRun(ctx, &store.PlanCheckRunMessage{
			ProjectID: "project-a",
			PlanUID:   101,
			Result:    &storepb.PlanCheckRunResult{},
		})
		return err
	})
	require.NoError(t, deleteErr)
	require.Equal(t, common.NotFound, common.ErrorCode(operationErr))
}
