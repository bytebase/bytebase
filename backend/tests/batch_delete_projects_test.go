package tests

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// TestCollisionBatchDeleteProjectsCascade is the batch counterpart of
// TestCollisionDeleteProjectCascade: BatchDeleteProjects purges through
// Store.DeleteProjects, whose predicates carry the project scope for every
// composite-PK table, so purging project A must leave project B's colliding
// ids untouched.
func TestCollisionBatchDeleteProjectsCascade(t *testing.T) {
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

	_, err = ctl.projectServiceClient.BatchDeleteProjects(ctx,
		connect.NewRequest(&v1pb.BatchDeleteProjectsRequest{Names: []string{fixture.ProjectA.Name}}))
	a.NoError(err)
	_, err = ctl.projectServiceClient.BatchDeleteProjects(ctx,
		connect.NewRequest(&v1pb.BatchDeleteProjectsRequest{
			Names: []string{fixture.ProjectA.Name},
			Purge: true,
		}))
	a.NoError(err)

	_, err = ctl.projectServiceClient.GetProject(ctx,
		connect.NewRequest(&v1pb.GetProjectRequest{Name: fixture.ProjectA.Name}))
	a.Error(err, "project A should be gone after batch purge; GetProject should fail")

	afterB := snapshotProject(ctx, t, ctl, fixture.ProjectB)
	assertProjectUnchanged(t, beforeB, afterB, "project B after project A batch deleted")
}

// TestBatchDeleteProjectsPurgeIsAtomic pins that a batch purge naming one
// project that is not archived purges nothing: before the purge became a
// single transaction, the projects ahead of the failure in the list were
// already irreversibly gone by the time the error was returned.
func TestBatchDeleteProjectsPurgeIsAtomic(t *testing.T) {
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
	a.Greater(len(beforeA.Plans), 0, "project A should have plans")

	// Archive A only. B is still active, so the batch fails its
	// archived-before-purge guard on B.
	_, err = ctl.projectServiceClient.DeleteProject(ctx,
		connect.NewRequest(&v1pb.DeleteProjectRequest{Name: fixture.ProjectA.Name}))
	a.NoError(err)

	_, err = ctl.projectServiceClient.BatchDeleteProjects(ctx,
		connect.NewRequest(&v1pb.BatchDeleteProjectsRequest{
			Names: []string{fixture.ProjectA.Name, fixture.ProjectB.Name},
			Purge: true,
		}))
	a.Error(err, "purging an active project must fail the whole batch")

	// Project A is archived but every row it owns is still there: the failed
	// batch purged nothing.
	projectA, err := ctl.projectServiceClient.GetProject(ctx,
		connect.NewRequest(&v1pb.GetProjectRequest{Name: fixture.ProjectA.Name}))
	a.NoError(err, "project A must survive a failed batch purge")
	a.Equal(v1pb.State_DELETED, projectA.Msg.State)

	_, err = ctl.projectServiceClient.UndeleteProject(ctx,
		connect.NewRequest(&v1pb.UndeleteProjectRequest{Name: fixture.ProjectA.Name}))
	a.NoError(err)
	assertProjectUnchanged(t, beforeA, snapshotProject(ctx, t, ctl, fixture.ProjectA),
		"project A after the failed batch purge")
	assertProjectUnchanged(t, beforeB, snapshotProject(ctx, t, ctl, fixture.ProjectB),
		"project B after the failed batch purge")
}
