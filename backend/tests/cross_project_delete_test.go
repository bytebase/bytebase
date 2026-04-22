package tests

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// TestCollisionDeleteProjectCascade verifies that permanently deleting
// project A does not remove or modify any rows belonging to project B,
// even when both projects have colliding composite-PK ids.
//
// Scope: covers plan, issue, task_run, plan_check_run isolation via the
// gRPC snapshot. Tables NOT covered here (currently): task, task_run_log,
// plan_webhook_delivery. Add targeted tests if a future change touches
// their DELETE paths.
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

	fixture.completeRolloutB(ctx, t, ctl)

	beforeB := snapshotProject(ctx, t, ctl, fixture.ProjectB)
	a.Greater(len(beforeB.TaskRuns), 0, "project B should have task_runs")
	a.Greater(len(beforeB.Plans), 0, "project B should have plans")
	a.Greater(len(beforeB.Issues), 0, "project B should have issues")

	// DeleteProject with Purge=true handles both soft-delete and hard-delete
	// (cascade) in one gRPC call.
	_, err = ctl.projectServiceClient.DeleteProject(ctx,
		connect.NewRequest(&v1pb.DeleteProjectRequest{
			Name:  fixture.ProjectA.Name,
			Purge: true,
		}))
	a.NoError(err)

	// Positive check: project A itself must actually be gone. Without this,
	// a regression that turned DeleteProject into a no-op would still pass
	// the project-B-unchanged check below.
	_, err = ctl.projectServiceClient.GetProject(ctx,
		connect.NewRequest(&v1pb.GetProjectRequest{Name: fixture.ProjectA.Name}))
	a.Error(err, "project A should be gone after purge; GetProject should fail")

	afterB := snapshotProject(ctx, t, ctl, fixture.ProjectB)
	assertProjectUnchanged(t, beforeB, afterB, "project B after project A deleted")
}

// TestCollisionDeleteInstanceNoCrossProjectCorruption verifies that deleting
// a shared instance cleans up both projects' instance-scoped rows (task_run)
// via the USING-join DELETE paths in the store, without cross-project
// corruption of project-scoped rows (plan, issue).
//
// Scope note: because setupCollidingProjects places both projects on a
// shared instance, this test cannot distinguish "bug: delete also wiped B's
// rows because it was cross-matching on id alone" from "correct: delete
// wiped B's rows because they were on the deleted instance". It therefore
// focuses on two things the composite-PK bug class can break:
//  1. Symmetric cleanup of instance-scoped rows for both projects.
//  2. Survival of project-scoped rows (plans, issues) for both projects.
//
// TestCollisionDeleteInstanceCrossProjectIsolation (below) covers the
// asymmetric scenario with separate per-project instances.
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

	fixture.completeRolloutB(ctx, t, ctl)

	beforeA := snapshotProject(ctx, t, ctl, fixture.ProjectA)
	beforeB := snapshotProject(ctx, t, ctl, fixture.ProjectB)
	a.Greater(len(beforeA.TaskRuns), 0, "project A should have task_runs before instance deletion")
	a.Greater(len(beforeB.TaskRuns), 0, "project B should have task_runs before instance deletion")

	purgeInstance(ctx, t, ctl, fixture.Instance.Name)

	afterA := snapshotProject(ctx, t, ctl, fixture.ProjectA)
	afterB := snapshotProject(ctx, t, ctl, fixture.ProjectB)

	// Instance-scoped rows (task_run) are removed for BOTH projects.
	// Asymmetric counts here would indicate a cross-project bug in the
	// DELETE USING predicates.
	a.Equal(0, len(afterA.TaskRuns), "project A task_runs should be cleaned up")
	a.Equal(0, len(afterB.TaskRuns), "project B task_runs should be cleaned up")

	// Project-scoped rows (plan, issue) are NOT instance-scoped and must
	// survive. Cross-project corruption would shift counts or swap rows.
	assertProjectUnchanged(t, plansIssuesOnly(beforeA), plansIssuesOnly(afterA), "project A plans/issues")
	assertProjectUnchanged(t, plansIssuesOnly(beforeB), plansIssuesOnly(afterB), "project B plans/issues")
}

// TestCollisionDeleteInstanceCrossProjectIsolation is the variant that can
// distinguish correct cascade from cross-project over-delete. Each project
// gets its OWN instance, but composite-PK ids still collide across projects
// via per-project nextProjectID allocation. Deleting instance A must only
// affect project A's rows — project B's task_runs must survive unchanged.
//
// This is the test that catches a buggy `DELETE ... USING` predicate that
// cross-matches rows on `id` alone across projects.
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

	fixture.completeRolloutB(ctx, t, ctl)

	beforeA := snapshotProject(ctx, t, ctl, fixture.ProjectA)
	beforeB := snapshotProject(ctx, t, ctl, fixture.ProjectB)
	a.Greater(len(beforeA.TaskRuns), 0, "project A should have task_runs before deletion")
	a.Greater(len(beforeB.TaskRuns), 0, "project B should have task_runs before deletion")

	purgeInstance(ctx, t, ctl, fixture.InstanceA.Name)

	// Positive check: project A's instance-scoped rows must actually be
	// removed. Without this, a no-op DeleteInstance would leave orphaned
	// state and still pass the cross-project isolation check.
	afterA := snapshotProject(ctx, t, ctl, fixture.ProjectA)
	a.Equal(0, len(afterA.TaskRuns),
		"project A task_runs should be cleaned up after its instance is deleted")

	// Isolation check: project B's rows reference instance B and must
	// survive entirely unchanged.
	afterB := snapshotProject(ctx, t, ctl, fixture.ProjectB)
	assertProjectUnchanged(t, beforeB, afterB, "project B after instance A deleted")
}

// purgeInstance soft-deletes (with Force because the instance has attached
// databases) then hard-deletes the instance via the gRPC API. The DeleteInstance
// API requires the instance already be soft-deleted before Purge works.
func purgeInstance(ctx context.Context, t *testing.T, ctl *controller, name string) {
	t.Helper()
	a := require.New(t)
	_, err := ctl.instanceServiceClient.DeleteInstance(ctx,
		connect.NewRequest(&v1pb.DeleteInstanceRequest{
			Name:  name,
			Force: true,
		}))
	a.NoError(err, "soft-delete instance %s", name)
	_, err = ctl.instanceServiceClient.DeleteInstance(ctx,
		connect.NewRequest(&v1pb.DeleteInstanceRequest{
			Name:  name,
			Purge: true,
		}))
	a.NoError(err, "purge instance %s", name)
}

// plansIssuesOnly returns a snapshot containing only the plan and issue
// slices — used by tests that expect task_run rows to be deleted but
// plan/issue rows to be preserved.
func plansIssuesOnly(s *projectSnapshot) *projectSnapshot {
	return &projectSnapshot{
		Plans:  s.Plans,
		Issues: s.Issues,
	}
}
