package tests

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// TestClaimAvailableTaskRunsNoCrossProjectResurrection verifies that the
// `ClaimAvailableTaskRuns` store query cannot resurrect a DONE task_run in
// another project, even when both projects have the same numeric task_run id.
//
// Design: rather than calling ClaimAvailableTaskRuns directly (which would
// leave rows stuck in RUNNING because the test's replica_id doesn't match
// the scheduler's, so the real scheduler would skip them and waitRollout
// would hang), we let the live scheduler run the claim naturally via
// completeRolloutB. The claim SQL is the same either way — this test's
// regression value is in observing whether project A's DONE rows survive
// the scheduler's claim pass unchanged.
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

	// fixture.BaselineA was captured inside setupCollidingProjects BEFORE
	// project B's plan was created, so it reflects project A's state
	// uncontaminated by any scheduler side effect on project B.
	a.Greater(len(fixture.BaselineA.TaskRuns), 0, "project A should have task_runs")
	for _, tr := range fixture.BaselineA.TaskRuns {
		a.Equal(v1pb.TaskRun_DONE, tr.Status,
			"project A task_run should be DONE before any project B activity")
	}

	fixture.completeRolloutB(ctx, t, ctl)

	afterA := snapshotProject(ctx, t, ctl, fixture.ProjectA)
	assertProjectUnchanged(t, fixture.BaselineA, afterA, "project A after scheduler claim pass")
}

// TestClaimAvailablePlanCheckRunsNoCrossProjectTransition verifies that the
// `ClaimAvailablePlanCheckRuns` store query cannot transition a terminal
// plan_check_run in another project.
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

	// Project A's consolidated plan_check_run must be in a terminal state
	// before B does any work — otherwise a corruption during fixture setup
	// could be baked into the "before" oracle and escape detection.
	a.Greater(len(fixture.BaselineA.PlanCheckRuns), 0, "project A should have plan_check_runs")
	for _, pcr := range fixture.BaselineA.PlanCheckRuns {
		a.NotEqual(v1pb.PlanCheckRun_STATUS_UNSPECIFIED, pcr.Status,
			"project A plan_check_run status should be set at baseline")
		a.NotEqual(v1pb.PlanCheckRun_RUNNING, pcr.Status,
			"project A plan_check_run should not be RUNNING at baseline")
	}

	fixture.completeRolloutB(ctx, t, ctl)

	afterA := snapshotProject(ctx, t, ctl, fixture.ProjectA)
	assertProjectUnchanged(t, fixture.BaselineA, afterA, "project A after scheduler plan_check_run claim pass")
}
