package tests

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
)

// TestCollisionDeleteProjectCascade verifies that permanently deleting
// project A does not remove or modify any rows belonging to project B,
// even when both projects have colliding composite-PK ids.
//
// Scope: covers plan, issue, task, task_run, plan_check_run isolation via
// the shared snapshot helper. Tables NOT covered here (currently): task_run_log
// and plan_webhook_delivery. Add targeted tests for those if a future change
// touches their DELETE paths.
func TestCollisionDeleteProjectCascade(t *testing.T) {
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

	projectBID := mustGetProjectID(t, fixture.ProjectB.Name)

	rolloutB, err := ctl.rolloutServiceClient.CreateRollout(ctx,
		connect.NewRequest(&v1pb.CreateRolloutRequest{
			Parent: fixture.PlanB.Name,
		}))
	a.NoError(err)
	err = ctl.waitRollout(ctx, fixture.IssueB.Name, rolloutB.Msg.Name)
	a.NoError(err)

	beforeB := snapshotProject(ctx, t, s, projectBID)
	a.Greater(len(beforeB.TaskRuns), 0, "project B should have task_runs")
	a.Greater(len(beforeB.Plans), 0, "project B should have plans")
	a.Greater(len(beforeB.Tasks), 0, "project B should have tasks")
	a.Greater(len(beforeB.Issues), 0, "project B should have issues")

	// Soft-delete project A (required by DeleteProject per AIP-164).
	_, err = ctl.projectServiceClient.DeleteProject(ctx,
		connect.NewRequest(&v1pb.DeleteProjectRequest{
			Name: fixture.ProjectA.Name,
		}))
	a.NoError(err)

	// Permanently delete project A (triggers cascade DELETEs).
	projectAID := mustGetProjectID(t, fixture.ProjectA.Name)
	workspace, err := s.GetWorkspaceID(ctx)
	a.NoError(err)
	err = s.DeleteProject(ctx, workspace, projectAID)
	a.NoError(err)

	// Positive check: project A's rows must actually be gone. Without this,
	// a regression that turned DeleteProject into a no-op would still pass
	// the project-B-unchanged check below.
	afterA := snapshotProject(ctx, t, s, projectAID)
	a.Equal(0, len(afterA.Plans), "project A plans should be gone after delete")
	a.Equal(0, len(afterA.Issues), "project A issues should be gone after delete")
	a.Equal(0, len(afterA.Tasks), "project A tasks should be gone after delete")
	a.Equal(0, len(afterA.TaskRuns), "project A task_runs should be gone after delete")
	a.Equal(0, len(afterA.PlanCheckRuns), "project A plan_check_runs should be gone after delete")

	afterB := snapshotProject(ctx, t, s, projectBID)
	assertProjectUnchanged(t, beforeB, afterB, "project B after project A deleted")
}

// TestCollisionDeleteInstanceNoCrossProjectCorruption verifies that deleting
// a shared instance cleans up both projects' instance-scoped rows (task,
// task_run) via the USING-join DELETE paths in the store, without
// cross-project corruption of project-scoped rows (plan, issue).
// Note: task_run_log cascade is exercised but not explicitly asserted here.
//
// Scope note: because setupCollidingProjects places both projects on a
// shared instance, this test cannot distinguish "bug: delete also wiped B's
// tasks because it was cross-matching on id alone" from "correct: delete
// wiped B's tasks because they were on the deleted instance". It therefore
// focuses on two things the composite-PK bug class can break:
//  1. Symmetric cleanup of instance-scoped rows for both projects.
//  2. Survival of project-scoped rows (plans, issues) for both projects.
//
// A separate test with per-project instances would be needed to distinguish
// the first case — deferring that pending a fixture variant.
func TestCollisionDeleteInstanceNoCrossProjectCorruption(t *testing.T) {
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
	projectBID := mustGetProjectID(t, fixture.ProjectB.Name)

	rolloutB, err := ctl.rolloutServiceClient.CreateRollout(ctx,
		connect.NewRequest(&v1pb.CreateRolloutRequest{
			Parent: fixture.PlanB.Name,
		}))
	a.NoError(err)
	err = ctl.waitRollout(ctx, fixture.IssueB.Name, rolloutB.Msg.Name)
	a.NoError(err)

	beforeA := snapshotProject(ctx, t, s, projectAID)
	beforeB := snapshotProject(ctx, t, s, projectBID)
	a.Greater(len(beforeA.TaskRuns), 0, "project A should have task_runs before instance deletion")
	a.Greater(len(beforeB.TaskRuns), 0, "project B should have task_runs before instance deletion")
	a.Greater(len(beforeA.Tasks), 0, "project A should have tasks before instance deletion")
	a.Greater(len(beforeB.Tasks), 0, "project B should have tasks before instance deletion")

	// Soft-delete then permanently delete the shared instance.
	_, err = ctl.instanceServiceClient.DeleteInstance(ctx,
		connect.NewRequest(&v1pb.DeleteInstanceRequest{
			Name:  fixture.Instance.Name,
			Force: true, // instance has attached databases; API rejects soft-delete otherwise
		}))
	a.NoError(err)

	instanceID := mustGetInstanceID(t, fixture.Instance.Name)
	workspace, err := s.GetWorkspaceID(ctx)
	a.NoError(err)
	err = s.DeleteInstance(ctx, workspace, instanceID)
	a.NoError(err)

	afterA := snapshotProject(ctx, t, s, projectAID)
	afterB := snapshotProject(ctx, t, s, projectBID)

	// Instance-scoped rows (task, task_run) are removed for BOTH projects.
	// Asymmetric counts here would indicate a cross-project bug in the
	// DELETE USING predicates.
	a.Equal(0, len(afterA.TaskRuns),
		"project A task_runs should be cleaned up")
	a.Equal(0, len(afterB.TaskRuns),
		"project B task_runs should be cleaned up")
	a.Equal(0, len(afterA.Tasks),
		"project A tasks should be cleaned up")
	a.Equal(0, len(afterB.Tasks),
		"project B tasks should be cleaned up")

	// Project-scoped rows (plan, issue) are NOT instance-scoped and must
	// survive unchanged. We compare by UID AND content (Title, Status,
	// UpdatedAt) — just matching UIDs would miss a swap bug because both
	// projects' plans/issues start with the same numeric UIDs.
	assertPlansUnchanged(t, beforeA.Plans, afterA.Plans, "project A plans")
	assertPlansUnchanged(t, beforeB.Plans, afterB.Plans, "project B plans")
	assertIssuesUnchanged(t, beforeA.Issues, afterA.Issues, "project A issues")
	assertIssuesUnchanged(t, beforeB.Issues, afterB.Issues, "project B issues")
}

// TestCollisionDeleteInstanceCrossProjectIsolation is the variant that can
// distinguish correct cascade from cross-project over-delete. Each project
// gets its OWN instance, but plan/task/task_run ids still collide across
// projects via per-project nextProjectID allocation. Deleting instance A
// must only affect project A's rows — project B's tasks/task_runs must
// survive unchanged.
//
// This is the test that catches a buggy `DELETE ... USING` predicate that
// cross-matches rows on `id` alone across projects. The shared-instance
// sibling test above covers the complementary scenario where both projects'
// rows should be cleaned up symmetrically.
func TestCollisionDeleteInstanceCrossProjectIsolation(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	fixture := setupCollidingProjectsSeparateInstances(ctx, t, ctl)
	s := ctl.server.StoreForTest()

	projectAID := mustGetProjectID(t, fixture.ProjectA.Name)
	projectBID := mustGetProjectID(t, fixture.ProjectB.Name)

	// Complete project B's rollout so it has task/task_run rows.
	rolloutB, err := ctl.rolloutServiceClient.CreateRollout(ctx,
		connect.NewRequest(&v1pb.CreateRolloutRequest{
			Parent: fixture.PlanB.Name,
		}))
	a.NoError(err)
	err = ctl.waitRollout(ctx, fixture.IssueB.Name, rolloutB.Msg.Name)
	a.NoError(err)

	beforeA := snapshotProject(ctx, t, s, projectAID)
	beforeB := snapshotProject(ctx, t, s, projectBID)
	a.Greater(len(beforeA.Tasks), 0, "project A should have tasks before deletion")
	a.Greater(len(beforeA.TaskRuns), 0, "project A should have task_runs before deletion")
	a.Greater(len(beforeB.TaskRuns), 0, "project B should have task_runs")
	a.Greater(len(beforeB.Tasks), 0, "project B should have tasks")

	// Prove task AND task_run IDs actually collide. nextProjectID allocates
	// per-table, so non-collision would make this test vacuous against
	// cross-project DELETE USING regressions.
	assertTasksCollide(ctx, t, ctl, fixture)
	assertTaskRunsCollide(ctx, t, ctl, fixture)

	// Delete instance A — project B's rows reference instance B and must
	// survive. A buggy USING join that matches on id alone would also
	// delete B's colliding rows.
	_, err = ctl.instanceServiceClient.DeleteInstance(ctx,
		connect.NewRequest(&v1pb.DeleteInstanceRequest{
			Name:  fixture.InstanceA.Name,
			Force: true, // instance has attached databases; API rejects soft-delete otherwise
		}))
	a.NoError(err)

	instanceAID := mustGetInstanceID(t, fixture.InstanceA.Name)
	workspace, err := s.GetWorkspaceID(ctx)
	a.NoError(err)
	err = s.DeleteInstance(ctx, workspace, instanceAID)
	a.NoError(err)

	// Positive check: project A's instance-scoped rows (task, task_run)
	// must actually be removed. Without this, a no-op DeleteInstance would
	// leave orphaned state and still pass the cross-project isolation check.
	afterA := snapshotProject(ctx, t, s, projectAID)
	a.Equal(0, len(afterA.Tasks),
		"project A tasks should be cleaned up after its instance is deleted")
	a.Equal(0, len(afterA.TaskRuns),
		"project A task_runs should be cleaned up after its instance is deleted")

	// Isolation check: project B's rows reference instance B and must
	// survive entirely unchanged.
	afterB := snapshotProject(ctx, t, s, projectBID)
	assertProjectUnchanged(t, beforeB, afterB, "project B after instance A deleted")
}

// assertPlansUnchanged verifies that plans survived unchanged — matching by
// UID and comparing the mutable fields (Description, UpdatedAt). Matching
// only by UID would miss a cross-project swap because colliding UIDs are
// valid within each project.
func assertPlansUnchanged(t *testing.T, before, after []*store.PlanMessage, label string) {
	t.Helper()
	a := require.New(t)
	a.Equal(len(before), len(after), "%s: count changed", label)
	afterByUID := make(map[int64]*store.PlanMessage, len(after))
	for _, p := range after {
		afterByUID[p.UID] = p
	}
	for _, b := range before {
		af, ok := afterByUID[b.UID]
		a.True(ok, "%s: plan UID %d missing after", label, b.UID)
		if ok {
			a.Equal(b.Description, af.Description,
				"%s: plan %d description changed", label, b.UID)
			a.Equal(b.UpdatedAt, af.UpdatedAt,
				"%s: plan %d updated_at changed", label, b.UID)
		}
	}
}

// assertIssuesUnchanged verifies that issues survived unchanged — matching by
// UID and comparing mutable content fields.
func assertIssuesUnchanged(t *testing.T, before, after []*store.IssueMessage, label string) {
	t.Helper()
	a := require.New(t)
	a.Equal(len(before), len(after), "%s: count changed", label)
	afterByUID := make(map[int64]*store.IssueMessage, len(after))
	for _, i := range after {
		afterByUID[i.UID] = i
	}
	for _, b := range before {
		af, ok := afterByUID[b.UID]
		a.True(ok, "%s: issue UID %d missing after", label, b.UID)
		if ok {
			a.Equal(b.Title, af.Title,
				"%s: issue %d title changed", label, b.UID)
			a.Equal(b.Status, af.Status,
				"%s: issue %d status changed", label, b.UID)
			a.Equal(b.UpdatedAt, af.UpdatedAt,
				"%s: issue %d updated_at changed", label, b.UID)
		}
	}
}
