package store_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/migrator"
	"github.com/bytebase/bytebase/backend/store"

	_ "github.com/bytebase/bytebase/backend/plugin/db/pg"
)

func TestUpdatePlanCheckRunIfApprovalInputVersionSkipsStaleWorkerOnRefreshedRow(t *testing.T) {
	ctx := context.Background()
	s := setupPlanCheckRunVersionStore(ctx, t)

	plan, err := s.CreatePlan(ctx, &store.PlanMessage{
		ProjectID:   "project-a",
		Name:        "plan-a",
		Description: "",
		Config:      &storepb.PlanConfig{},
	}, "creator@example.com")
	require.NoError(t, err)

	require.NoError(t, s.CreatePlanCheckRun(ctx, &store.PlanCheckRunMessage{
		ProjectID: "project-a",
		PlanUID:   plan.UID,
		Result:    &storepb.PlanCheckRunResult{ApprovalInputVersion: 1},
	}))

	claimed, err := s.ClaimAvailablePlanCheckRuns(ctx)
	require.NoError(t, err)
	require.Len(t, claimed, 1)
	require.EqualValues(t, 1, claimed[0].ApprovalInputVersion)

	require.NoError(t, s.CreatePlanCheckRun(ctx, &store.PlanCheckRunMessage{
		ProjectID: "project-a",
		PlanUID:   plan.UID,
		Result:    &storepb.PlanCheckRunResult{ApprovalInputVersion: 2},
	}))

	refreshedRun, err := s.GetPlanCheckRun(ctx, "project-a", plan.UID)
	require.NoError(t, err)
	require.NotNil(t, refreshedRun)
	require.EqualValues(t, claimed[0].UID, refreshedRun.UID)

	updated, err := s.UpdatePlanCheckRunIfApprovalInputVersion(ctx, "project-a", store.PlanCheckRunStatusDone, &storepb.PlanCheckRunResult{
		ApprovalInputVersion: 1,
		Error:                "stale result",
	}, claimed[0].UID, 1)
	require.NoError(t, err)
	require.False(t, updated)

	run, err := s.GetPlanCheckRun(ctx, "project-a", plan.UID)
	require.NoError(t, err)
	require.NotNil(t, run)
	require.Equal(t, store.PlanCheckRunStatusAvailable, run.Status)
	require.EqualValues(t, 2, run.Result.GetApprovalInputVersion())
	require.Empty(t, run.Result.GetError())
}

func TestRefreshPlanCheckRunIfStaleApprovalInputVersionDoesNotResetRunningCheck(t *testing.T) {
	ctx := context.Background()
	s := setupPlanCheckRunVersionStore(ctx, t)

	plan, err := s.CreatePlan(ctx, &store.PlanMessage{
		ProjectID:   "project-a",
		Name:        "plan-a",
		Description: "",
		Config:      &storepb.PlanConfig{},
	}, "creator@example.com")
	require.NoError(t, err)

	require.NoError(t, s.CreatePlanCheckRun(ctx, &store.PlanCheckRunMessage{
		ProjectID: "project-a",
		PlanUID:   plan.UID,
		Result:    &storepb.PlanCheckRunResult{ApprovalInputVersion: 2},
	}))

	claimed, err := s.ClaimAvailablePlanCheckRuns(ctx)
	require.NoError(t, err)
	require.Len(t, claimed, 1)
	require.EqualValues(t, 2, claimed[0].ApprovalInputVersion)

	refreshed, err := s.RefreshPlanCheckRunIfStaleApprovalInputVersion(ctx, "project-a", plan.UID, 2)
	require.NoError(t, err)
	require.False(t, refreshed)

	run, err := s.GetPlanCheckRun(ctx, "project-a", plan.UID)
	require.NoError(t, err)
	require.NotNil(t, run)
	require.Equal(t, store.PlanCheckRunStatusRunning, run.Status)
	require.EqualValues(t, 2, run.Result.GetApprovalInputVersion())
}

func TestFailStalePlanCheckRunsPreservesApprovalInputVersion(t *testing.T) {
	ctx := context.Background()
	s := setupPlanCheckRunVersionStore(ctx, t)

	plan, err := s.CreatePlan(ctx, &store.PlanMessage{
		ProjectID:   "project-a",
		Name:        "plan-a",
		Description: "",
		Config:      &storepb.PlanConfig{},
	}, "creator@example.com")
	require.NoError(t, err)

	require.NoError(t, s.CreatePlanCheckRun(ctx, &store.PlanCheckRunMessage{
		ProjectID: "project-a",
		PlanUID:   plan.UID,
		Result:    &storepb.PlanCheckRunResult{ApprovalInputVersion: 7},
	}))
	claimed, err := s.ClaimAvailablePlanCheckRuns(ctx)
	require.NoError(t, err)
	require.Len(t, claimed, 1)

	rowsAffected, err := s.FailStalePlanCheckRuns(ctx, 0)
	require.NoError(t, err)
	require.EqualValues(t, 1, rowsAffected)

	run, err := s.GetPlanCheckRun(ctx, "project-a", plan.UID)
	require.NoError(t, err)
	require.Equal(t, store.PlanCheckRunStatusFailed, run.Status)
	require.EqualValues(t, 7, run.Result.GetApprovalInputVersion())
	require.NotEmpty(t, run.Result.GetResults())
}

func setupPlanCheckRunVersionStore(ctx context.Context, t *testing.T) *store.Store {
	t.Helper()

	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(ctx) })

	db := container.GetDB()
	require.NoError(t, migrator.MigrateSchema(ctx, db))

	_, err := db.ExecContext(ctx, `
		INSERT INTO workspace (resource_id) VALUES ('default');
		INSERT INTO principal (name, email, password_hash) VALUES ('creator', 'creator@example.com', 'unused');
		INSERT INTO project (resource_id, workspace, name) VALUES ('project-a', 'default', 'Project A');
	`)
	require.NoError(t, err)

	pgURL := fmt.Sprintf(
		"host=%s port=%s user=postgres password=root-password database=postgres",
		container.GetHost(), container.GetPort(),
	)
	s, err := store.New(ctx, pgURL, false)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, s.Close()) })
	return s
}
