package tests

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
)

// TestClaimAvailableTaskRunsNoCrossProjectResurrection verifies that the
// `ClaimAvailableTaskRuns` store query cannot resurrect a DONE task_run in
// another project, even when both projects have the same numeric task_run id.
//
// Design: rather than calling ClaimAvailableTaskRuns directly (which would
// leave rows stuck in RUNNING because the test's replica_id doesn't match
// the scheduler's, so the real scheduler would skip them and waitRollout
// would hang), we let the live scheduler run the claim naturally via
// CreateRollout + waitRollout. The claim SQL is the same either way — this
// test's regression value is in observing whether project A's DONE rows
// survive the scheduler's claim pass unchanged.
//
// Regression lock for BYT-9259 (customer data loss from silent re-execution).
func TestClaimAvailableTaskRunsNoCrossProjectResurrection(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	fixture := setupCollidingProjects(ctx, t, ctl)
	s := ctl.server.StoreForTest()

	projectAID := mustGetProjectID(t, fixture.ProjectA.Name)

	// fixture.BaselineA was captured inside setupCollidingProjects BEFORE
	// project B's plan was created, so it reflects project A's state
	// uncontaminated by any scheduler side effect on project B.
	a.Greater(len(fixture.BaselineA.TaskRuns), 0, "project A should have task_runs")
	for _, tr := range fixture.BaselineA.TaskRuns {
		a.Equal(storepb.TaskRun_DONE, tr.Status,
			"project A task_run should be DONE before any project B activity")
	}

	// Drive project B's rollout to completion. Under the hood, the live
	// task-run scheduler calls ClaimAvailableTaskRuns — exactly the SQL we
	// want to exercise. A regression of BYT-9259 would mutate project A's
	// DONE rows here because the buggy WHERE predicate ignored project.
	rolloutB, err := ctl.rolloutServiceClient.CreateRollout(ctx,
		connect.NewRequest(&v1pb.CreateRolloutRequest{
			Parent: fixture.PlanB.Name,
		}))
	a.NoError(err)
	err = ctl.waitRollout(ctx, fixture.IssueB.Name, rolloutB.Msg.Name)
	a.NoError(err, "project B's rollout should complete — claim path broken if this fails")

	// Prove the composite-PK collision actually exists before declaring the
	// test meaningful. If the invariant is broken (e.g., future changes to
	// nextProjectID allocation), the regression guard would silently weaken.
	assertTaskRunsCollide(ctx, t, ctl, fixture)

	// The regression invariant: project A's task_runs are completely unchanged.
	afterA := snapshotProject(ctx, t, s, projectAID)
	assertProjectUnchanged(t, fixture.BaselineA, afterA, "project A after scheduler claim pass")
}

// TestClaimAvailablePlanCheckRunsNoCrossProjectTransition verifies that the
// `ClaimAvailablePlanCheckRuns` store query cannot transition a terminal
// plan_check_run in another project. Uses the same natural-scheduler design
// as the task-run test above.
func TestClaimAvailablePlanCheckRunsNoCrossProjectTransition(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	fixture := setupCollidingProjects(ctx, t, ctl)
	s := ctl.server.StoreForTest()

	projectAID := mustGetProjectID(t, fixture.ProjectA.Name)

	// Use the pre-B-plan baseline to ensure any corruption during project B's
	// creation is detected (not baked into the oracle).
	a.Greater(len(fixture.BaselineA.PlanCheckRuns), 0, "project A should have plan_check_runs")
	for _, pcr := range fixture.BaselineA.PlanCheckRuns {
		a.NotEqual(store.PlanCheckRunStatusAvailable, pcr.Status,
			"project A plan_check_run should not be AVAILABLE at baseline")
		a.NotEqual(store.PlanCheckRunStatusRunning, pcr.Status,
			"project A plan_check_run should not be RUNNING at baseline")
	}

	// Drive project B's rollout to completion. The plan-check scheduler calls
	// ClaimAvailablePlanCheckRuns behind the scenes; a cross-project SQL
	// regression would touch project A's terminal rows here.
	rolloutB, err := ctl.rolloutServiceClient.CreateRollout(ctx,
		connect.NewRequest(&v1pb.CreateRolloutRequest{
			Parent: fixture.PlanB.Name,
		}))
	a.NoError(err)
	err = ctl.waitRollout(ctx, fixture.IssueB.Name, rolloutB.Msg.Name)
	a.NoError(err, "project B's rollout should complete — claim path broken if this fails")

	// Verify task_run collision invariant as a proxy for plan_check_run
	// collision (both use per-project nextProjectID allocation).
	assertTaskRunsCollide(ctx, t, ctl, fixture)

	afterA := snapshotProject(ctx, t, s, projectAID)
	assertProjectUnchanged(t, fixture.BaselineA, afterA, "project A after scheduler plan_check_run claim pass")
}
