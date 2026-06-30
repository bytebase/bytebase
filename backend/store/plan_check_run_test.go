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

func TestClaimAvailablePlanCheckRunsDefaultsMissingApprovalInputVersion(t *testing.T) {
	ctx := context.Background()
	s := setupPlanCheckRunVersionStore(ctx, t)

	plan, err := s.CreatePlan(ctx, &store.PlanMessage{
		ProjectID:   "project-a",
		Name:        "plan-a",
		Description: "",
		Config:      &storepb.PlanConfig{},
	}, "creator@example.com")
	require.NoError(t, err)

	created, err := s.CreatePlanCheckRun(ctx, &store.PlanCheckRunMessage{
		ProjectID: "project-a",
		PlanUID:   plan.UID,
		Result:    &storepb.PlanCheckRunResult{},
	})
	require.NoError(t, err)
	require.True(t, created)

	claimed, err := s.ClaimAvailablePlanCheckRuns(ctx)
	require.NoError(t, err)
	require.Len(t, claimed, 1)
	require.EqualValues(t, 0, claimed[0].ApprovalInputVersion)
}

func TestUpdatePlanCheckRunIfApprovalInputVersionSkipsStaleWorkerOnRefreshedRow(t *testing.T) {
	ctx := context.Background()
	s := setupPlanCheckRunVersionStore(ctx, t)

	plan, err := s.CreatePlan(ctx, &store.PlanMessage{
		ProjectID:   "project-a",
		Name:        "plan-a",
		Description: "",
		Config:      &storepb.PlanConfig{ApprovalInputVersion: 1},
	}, "creator@example.com")
	require.NoError(t, err)

	created, err := s.CreatePlanCheckRun(ctx, &store.PlanCheckRunMessage{
		ProjectID: "project-a",
		PlanUID:   plan.UID,
		Result:    &storepb.PlanCheckRunResult{ApprovalInputVersion: 1},
	})
	require.NoError(t, err)
	require.True(t, created)

	claimed, err := s.ClaimAvailablePlanCheckRuns(ctx)
	require.NoError(t, err)
	require.Len(t, claimed, 1)
	require.EqualValues(t, 1, claimed[0].ApprovalInputVersion)

	_, err = s.UpdatePlan(ctx, &store.UpdatePlanMessage{
		UID:       plan.UID,
		ProjectID: plan.ProjectID,
		Config:    &storepb.PlanConfig{ApprovalInputVersion: 2},
	})
	require.NoError(t, err)

	created, err = s.CreatePlanCheckRun(ctx, &store.PlanCheckRunMessage{
		ProjectID: "project-a",
		PlanUID:   plan.UID,
		Result:    &storepb.PlanCheckRunResult{ApprovalInputVersion: 2},
	})
	require.NoError(t, err)
	require.True(t, created)

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

func TestUpdatePlanCheckRunIfApprovalInputVersionSkipsStaleWorkerAfterPlanVersionBump(t *testing.T) {
	ctx := context.Background()
	s := setupPlanCheckRunVersionStore(ctx, t)

	plan, err := s.CreatePlan(ctx, &store.PlanMessage{
		ProjectID:   "project-a",
		Name:        "plan-a",
		Description: "",
		Config:      &storepb.PlanConfig{ApprovalInputVersion: 1},
	}, "creator@example.com")
	require.NoError(t, err)

	created, err := s.CreatePlanCheckRun(ctx, &store.PlanCheckRunMessage{
		ProjectID: "project-a",
		PlanUID:   plan.UID,
		Result:    &storepb.PlanCheckRunResult{ApprovalInputVersion: 1},
	})
	require.NoError(t, err)
	require.True(t, created)

	claimed, err := s.ClaimAvailablePlanCheckRuns(ctx)
	require.NoError(t, err)
	require.Len(t, claimed, 1)
	require.EqualValues(t, 1, claimed[0].ApprovalInputVersion)

	_, err = s.UpdatePlan(ctx, &store.UpdatePlanMessage{
		UID:       plan.UID,
		ProjectID: plan.ProjectID,
		Config:    &storepb.PlanConfig{ApprovalInputVersion: 2},
	})
	require.NoError(t, err)

	updated, err := s.UpdatePlanCheckRunIfApprovalInputVersion(ctx, "project-a", store.PlanCheckRunStatusDone, &storepb.PlanCheckRunResult{
		ApprovalInputVersion: 1,
		Results: []*storepb.PlanCheckRunResult_Result{{
			Status: storepb.Advice_SUCCESS,
			Title:  "stale result",
		}},
	}, claimed[0].UID, 1)
	require.NoError(t, err)
	require.False(t, updated)

	run, err := s.GetPlanCheckRun(ctx, "project-a", plan.UID)
	require.NoError(t, err)
	require.NotNil(t, run)
	require.Equal(t, store.PlanCheckRunStatusRunning, run.Status)
	require.EqualValues(t, 1, run.Result.GetApprovalInputVersion())
	require.Empty(t, run.Result.GetResults())
}

func TestCreatePlanCheckRunDoesNotResetActiveSameVersionRun(t *testing.T) {
	ctx := context.Background()
	s := setupPlanCheckRunVersionStore(ctx, t)

	plan, err := s.CreatePlan(ctx, &store.PlanMessage{
		ProjectID:   "project-a",
		Name:        "plan-a",
		Description: "",
		Config:      &storepb.PlanConfig{ApprovalInputVersion: 2},
	}, "creator@example.com")
	require.NoError(t, err)

	created, err := s.CreatePlanCheckRun(ctx, &store.PlanCheckRunMessage{
		ProjectID: "project-a",
		PlanUID:   plan.UID,
		Result:    &storepb.PlanCheckRunResult{ApprovalInputVersion: 2},
	})
	require.NoError(t, err)
	require.True(t, created)

	claimed, err := s.ClaimAvailablePlanCheckRuns(ctx)
	require.NoError(t, err)
	require.Len(t, claimed, 1)
	require.EqualValues(t, 2, claimed[0].ApprovalInputVersion)

	created, err = s.CreatePlanCheckRun(ctx, &store.PlanCheckRunMessage{
		ProjectID: "project-a",
		PlanUID:   plan.UID,
		Result:    &storepb.PlanCheckRunResult{ApprovalInputVersion: 2},
	})
	require.NoError(t, err)
	require.False(t, created)

	run, err := s.GetPlanCheckRun(ctx, "project-a", plan.UID)
	require.NoError(t, err)
	require.NotNil(t, run)
	require.Equal(t, store.PlanCheckRunStatusRunning, run.Status)
	require.EqualValues(t, 2, run.Result.GetApprovalInputVersion())
}

func TestCreatePlanCheckRunDoesNotResetActiveSameVersionAvailableRun(t *testing.T) {
	ctx := context.Background()
	s := setupPlanCheckRunVersionStore(ctx, t)

	plan, err := s.CreatePlan(ctx, &store.PlanMessage{
		ProjectID:   "project-a",
		Name:        "plan-a",
		Description: "",
		Config:      &storepb.PlanConfig{ApprovalInputVersion: 2},
	}, "creator@example.com")
	require.NoError(t, err)

	created, err := s.CreatePlanCheckRun(ctx, &store.PlanCheckRunMessage{
		ProjectID: "project-a",
		PlanUID:   plan.UID,
		Result:    &storepb.PlanCheckRunResult{ApprovalInputVersion: 2},
	})
	require.NoError(t, err)
	require.True(t, created)

	created, err = s.CreatePlanCheckRun(ctx, &store.PlanCheckRunMessage{
		ProjectID: "project-a",
		PlanUID:   plan.UID,
		Result:    &storepb.PlanCheckRunResult{ApprovalInputVersion: 2},
	})
	require.NoError(t, err)
	require.False(t, created)

	run, err := s.GetPlanCheckRun(ctx, "project-a", plan.UID)
	require.NoError(t, err)
	require.NotNil(t, run)
	require.Equal(t, store.PlanCheckRunStatusAvailable, run.Status)
	require.EqualValues(t, 2, run.Result.GetApprovalInputVersion())
}

func TestCreatePlanCheckRunAllowsTerminalSameVersionRerun(t *testing.T) {
	ctx := context.Background()
	s := setupPlanCheckRunVersionStore(ctx, t)

	plan, err := s.CreatePlan(ctx, &store.PlanMessage{
		ProjectID:   "project-a",
		Name:        "plan-a",
		Description: "",
		Config:      &storepb.PlanConfig{ApprovalInputVersion: 2},
	}, "creator@example.com")
	require.NoError(t, err)

	created, err := s.CreatePlanCheckRun(ctx, &store.PlanCheckRunMessage{
		ProjectID: "project-a",
		PlanUID:   plan.UID,
		Result:    &storepb.PlanCheckRunResult{ApprovalInputVersion: 2},
	})
	require.NoError(t, err)
	require.True(t, created)

	claimed, err := s.ClaimAvailablePlanCheckRuns(ctx)
	require.NoError(t, err)
	require.Len(t, claimed, 1)

	updated, err := s.UpdatePlanCheckRunIfApprovalInputVersion(ctx, "project-a", store.PlanCheckRunStatusDone, &storepb.PlanCheckRunResult{
		ApprovalInputVersion: 2,
		Results: []*storepb.PlanCheckRunResult_Result{{
			Status: storepb.Advice_SUCCESS,
			Title:  "current result",
		}},
	}, claimed[0].UID, 2)
	require.NoError(t, err)
	require.True(t, updated)

	created, err = s.CreatePlanCheckRun(ctx, &store.PlanCheckRunMessage{
		ProjectID: "project-a",
		PlanUID:   plan.UID,
		Result:    &storepb.PlanCheckRunResult{ApprovalInputVersion: 2},
	})
	require.NoError(t, err)
	require.True(t, created)

	run, err := s.GetPlanCheckRun(ctx, "project-a", plan.UID)
	require.NoError(t, err)
	require.NotNil(t, run)
	require.Equal(t, store.PlanCheckRunStatusAvailable, run.Status)
	require.EqualValues(t, 2, run.Result.GetApprovalInputVersion())
}

func TestCreatePlanCheckRunSkipsStaleIncomingApprovalInputVersion(t *testing.T) {
	ctx := context.Background()
	s := setupPlanCheckRunVersionStore(ctx, t)

	plan, err := s.CreatePlan(ctx, &store.PlanMessage{
		ProjectID:   "project-a",
		Name:        "plan-a",
		Description: "",
		Config:      &storepb.PlanConfig{ApprovalInputVersion: 2},
	}, "creator@example.com")
	require.NoError(t, err)

	created, err := s.CreatePlanCheckRun(ctx, &store.PlanCheckRunMessage{
		ProjectID: "project-a",
		PlanUID:   plan.UID,
		Result:    &storepb.PlanCheckRunResult{ApprovalInputVersion: 2},
	})
	require.NoError(t, err)
	require.True(t, created)

	created, err = s.CreatePlanCheckRun(ctx, &store.PlanCheckRunMessage{
		ProjectID: "project-a",
		PlanUID:   plan.UID,
		Result:    &storepb.PlanCheckRunResult{ApprovalInputVersion: 1},
	})
	require.NoError(t, err)
	require.False(t, created)

	run, err := s.GetPlanCheckRun(ctx, "project-a", plan.UID)
	require.NoError(t, err)
	require.NotNil(t, run)
	require.Equal(t, store.PlanCheckRunStatusAvailable, run.Status)
	require.EqualValues(t, 2, run.Result.GetApprovalInputVersion())
}

func TestRefreshPlanCheckRunIfStaleApprovalInputVersionDoesNotResetRunningCheck(t *testing.T) {
	ctx := context.Background()
	s := setupPlanCheckRunVersionStore(ctx, t)

	plan, err := s.CreatePlan(ctx, &store.PlanMessage{
		ProjectID:   "project-a",
		Name:        "plan-a",
		Description: "",
		Config:      &storepb.PlanConfig{ApprovalInputVersion: 2},
	}, "creator@example.com")
	require.NoError(t, err)

	created, err := s.CreatePlanCheckRun(ctx, &store.PlanCheckRunMessage{
		ProjectID: "project-a",
		PlanUID:   plan.UID,
		Result:    &storepb.PlanCheckRunResult{ApprovalInputVersion: 2},
	})
	require.NoError(t, err)
	require.True(t, created)

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

func TestRefreshPlanCheckRunIfStaleApprovalInputVersionRefreshesTerminalStaleCheck(t *testing.T) {
	ctx := context.Background()
	s := setupPlanCheckRunVersionStore(ctx, t)

	plan, err := s.CreatePlan(ctx, &store.PlanMessage{
		ProjectID:   "project-a",
		Name:        "plan-a",
		Description: "",
		Config:      &storepb.PlanConfig{ApprovalInputVersion: 1},
	}, "creator@example.com")
	require.NoError(t, err)

	created, err := s.CreatePlanCheckRun(ctx, &store.PlanCheckRunMessage{
		ProjectID: "project-a",
		PlanUID:   plan.UID,
		Result:    &storepb.PlanCheckRunResult{ApprovalInputVersion: 1},
	})
	require.NoError(t, err)
	require.True(t, created)

	claimed, err := s.ClaimAvailablePlanCheckRuns(ctx)
	require.NoError(t, err)
	require.Len(t, claimed, 1)

	updated, err := s.UpdatePlanCheckRunIfApprovalInputVersion(ctx, "project-a", store.PlanCheckRunStatusDone, &storepb.PlanCheckRunResult{
		ApprovalInputVersion: 1,
		Results: []*storepb.PlanCheckRunResult_Result{{
			Status: storepb.Advice_SUCCESS,
			Title:  "current result",
		}},
	}, claimed[0].UID, 1)
	require.NoError(t, err)
	require.True(t, updated)

	_, err = s.UpdatePlan(ctx, &store.UpdatePlanMessage{
		UID:       plan.UID,
		ProjectID: plan.ProjectID,
		Config:    &storepb.PlanConfig{ApprovalInputVersion: 2},
	})
	require.NoError(t, err)

	refreshed, err := s.RefreshPlanCheckRunIfStaleApprovalInputVersion(ctx, "project-a", plan.UID, 2)
	require.NoError(t, err)
	require.True(t, refreshed)

	run, err := s.GetPlanCheckRun(ctx, "project-a", plan.UID)
	require.NoError(t, err)
	require.NotNil(t, run)
	require.Equal(t, store.PlanCheckRunStatusAvailable, run.Status)
	require.EqualValues(t, 2, run.Result.GetApprovalInputVersion())
	require.Empty(t, run.Result.GetResults())
}

func TestRefreshPlanCheckRunIfStaleApprovalInputVersionSkipsStaleRequestedVersion(t *testing.T) {
	ctx := context.Background()
	s := setupPlanCheckRunVersionStore(ctx, t)

	plan, err := s.CreatePlan(ctx, &store.PlanMessage{
		ProjectID:   "project-a",
		Name:        "plan-a",
		Description: "",
		Config:      &storepb.PlanConfig{ApprovalInputVersion: 1},
	}, "creator@example.com")
	require.NoError(t, err)

	created, err := s.CreatePlanCheckRun(ctx, &store.PlanCheckRunMessage{
		ProjectID: "project-a",
		PlanUID:   plan.UID,
		Result:    &storepb.PlanCheckRunResult{ApprovalInputVersion: 1},
	})
	require.NoError(t, err)
	require.True(t, created)

	claimed, err := s.ClaimAvailablePlanCheckRuns(ctx)
	require.NoError(t, err)
	require.Len(t, claimed, 1)

	updated, err := s.UpdatePlanCheckRunIfApprovalInputVersion(ctx, "project-a", store.PlanCheckRunStatusDone, &storepb.PlanCheckRunResult{
		ApprovalInputVersion: 1,
		Results: []*storepb.PlanCheckRunResult_Result{{
			Status: storepb.Advice_SUCCESS,
			Title:  "stale result",
		}},
	}, claimed[0].UID, 1)
	require.NoError(t, err)
	require.True(t, updated)

	_, err = s.UpdatePlan(ctx, &store.UpdatePlanMessage{
		UID:       plan.UID,
		ProjectID: plan.ProjectID,
		Config:    &storepb.PlanConfig{ApprovalInputVersion: 2},
	})
	require.NoError(t, err)

	refreshed, err := s.RefreshPlanCheckRunIfStaleApprovalInputVersion(ctx, "project-a", plan.UID, 1)
	require.NoError(t, err)
	require.False(t, refreshed)

	run, err := s.GetPlanCheckRun(ctx, "project-a", plan.UID)
	require.NoError(t, err)
	require.NotNil(t, run)
	require.Equal(t, store.PlanCheckRunStatusDone, run.Status)
	require.EqualValues(t, 1, run.Result.GetApprovalInputVersion())
}

func TestCancelPlanCheckRunIfApprovalInputVersionSkipsRefreshedRow(t *testing.T) {
	ctx := context.Background()
	s := setupPlanCheckRunVersionStore(ctx, t)

	plan, err := s.CreatePlan(ctx, &store.PlanMessage{
		ProjectID:   "project-a",
		Name:        "plan-a",
		Description: "",
		Config:      &storepb.PlanConfig{ApprovalInputVersion: 1},
	}, "creator@example.com")
	require.NoError(t, err)

	created, err := s.CreatePlanCheckRun(ctx, &store.PlanCheckRunMessage{
		ProjectID: "project-a",
		PlanUID:   plan.UID,
		Result:    &storepb.PlanCheckRunResult{ApprovalInputVersion: 1},
	})
	require.NoError(t, err)
	require.True(t, created)

	claimed, err := s.ClaimAvailablePlanCheckRuns(ctx)
	require.NoError(t, err)
	require.Len(t, claimed, 1)

	_, err = s.UpdatePlan(ctx, &store.UpdatePlanMessage{
		UID:       plan.UID,
		ProjectID: plan.ProjectID,
		Config:    &storepb.PlanConfig{ApprovalInputVersion: 2},
	})
	require.NoError(t, err)

	created, err = s.CreatePlanCheckRun(ctx, &store.PlanCheckRunMessage{
		ProjectID: "project-a",
		PlanUID:   plan.UID,
		Result:    &storepb.PlanCheckRunResult{ApprovalInputVersion: 2},
	})
	require.NoError(t, err)
	require.True(t, created)

	canceled, err := s.CancelPlanCheckRunIfApprovalInputVersion(ctx, "project-a", claimed[0].UID, 1)
	require.NoError(t, err)
	require.False(t, canceled)

	run, err := s.GetPlanCheckRun(ctx, "project-a", plan.UID)
	require.NoError(t, err)
	require.NotNil(t, run)
	require.Equal(t, store.PlanCheckRunStatusAvailable, run.Status)
	require.EqualValues(t, 2, run.Result.GetApprovalInputVersion())

	canceled, err = s.CancelPlanCheckRunIfApprovalInputVersion(ctx, "project-a", claimed[0].UID, 2)
	require.NoError(t, err)
	require.True(t, canceled)

	run, err = s.GetPlanCheckRun(ctx, "project-a", plan.UID)
	require.NoError(t, err)
	require.Equal(t, store.PlanCheckRunStatusCanceled, run.Status)
	require.EqualValues(t, 2, run.Result.GetApprovalInputVersion())
}

func TestCancelPlanCheckRunIfApprovalInputVersionAllowsObservedStaleRow(t *testing.T) {
	ctx := context.Background()
	s := setupPlanCheckRunVersionStore(ctx, t)

	plan, err := s.CreatePlan(ctx, &store.PlanMessage{
		ProjectID:   "project-a",
		Name:        "plan-a",
		Description: "",
		Config:      &storepb.PlanConfig{ApprovalInputVersion: 1},
	}, "creator@example.com")
	require.NoError(t, err)

	created, err := s.CreatePlanCheckRun(ctx, &store.PlanCheckRunMessage{
		ProjectID: "project-a",
		PlanUID:   plan.UID,
		Result:    &storepb.PlanCheckRunResult{ApprovalInputVersion: 1},
	})
	require.NoError(t, err)
	require.True(t, created)

	claimed, err := s.ClaimAvailablePlanCheckRuns(ctx)
	require.NoError(t, err)
	require.Len(t, claimed, 1)

	_, err = s.UpdatePlan(ctx, &store.UpdatePlanMessage{
		UID:       plan.UID,
		ProjectID: plan.ProjectID,
		Config:    &storepb.PlanConfig{ApprovalInputVersion: 2},
	})
	require.NoError(t, err)

	canceled, err := s.CancelPlanCheckRunIfApprovalInputVersion(ctx, "project-a", claimed[0].UID, 1)
	require.NoError(t, err)
	require.True(t, canceled)

	run, err := s.GetPlanCheckRun(ctx, "project-a", plan.UID)
	require.NoError(t, err)
	require.NotNil(t, run)
	require.Equal(t, store.PlanCheckRunStatusCanceled, run.Status)
	require.EqualValues(t, 1, run.Result.GetApprovalInputVersion())
}

func TestFailStalePlanCheckRunsPreservesApprovalInputVersion(t *testing.T) {
	ctx := context.Background()
	s := setupPlanCheckRunVersionStore(ctx, t)

	plan, err := s.CreatePlan(ctx, &store.PlanMessage{
		ProjectID:   "project-a",
		Name:        "plan-a",
		Description: "",
		Config:      &storepb.PlanConfig{ApprovalInputVersion: 7},
	}, "creator@example.com")
	require.NoError(t, err)

	created, err := s.CreatePlanCheckRun(ctx, &store.PlanCheckRunMessage{
		ProjectID: "project-a",
		PlanUID:   plan.UID,
		Result:    &storepb.PlanCheckRunResult{ApprovalInputVersion: 7},
	})
	require.NoError(t, err)
	require.True(t, created)
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
