package tests

import (
	"context"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// TestCollisionUpdateIssueNoCrossProjectEffect verifies that updating
// an issue in project A (a) actually changes A's title and (b) does not
// leak the title/status mutation into project B's same-id issue.
//
// Note on assertion scope: we deliberately avoid a full snapshot comparison
// of project B here. The approval runner actively updates project B's open
// issue in the background (approval template resolution, updated_at) —
// that's expected behavior, not a cross-project leak. Title and Status are
// the fields UpdateIssue(A) could actually cross-corrupt if the WHERE clause
// were broken, so those are what we assert.
func TestCollisionUpdateIssueNoCrossProjectEffect(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	fixture := setupCollidingProjects(ctx, t, ctl)

	issueBBefore, err := ctl.issueServiceClient.GetIssue(ctx,
		connect.NewRequest(&v1pb.GetIssueRequest{Name: fixture.IssueB.Name}))
	a.NoError(err)

	const updatedTitle = "Updated collision test A"
	_, err = ctl.issueServiceClient.UpdateIssue(ctx,
		connect.NewRequest(&v1pb.UpdateIssueRequest{
			Issue: &v1pb.Issue{
				Name:  fixture.IssueA.Name,
				Title: updatedTitle,
			},
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"title"}},
		}))
	a.NoError(err)

	// Positive check: confirm project A's issue actually received the update.
	issueAAfter, err := ctl.issueServiceClient.GetIssue(ctx,
		connect.NewRequest(&v1pb.GetIssueRequest{Name: fixture.IssueA.Name}))
	a.NoError(err)
	a.Equal(updatedTitle, issueAAfter.Msg.Title,
		"project A's issue title should have been updated")

	// Isolation check: project B's Title/Status — the fields UpdateIssue can
	// cross-corrupt — must be unchanged.
	issueBAfter, err := ctl.issueServiceClient.GetIssue(ctx,
		connect.NewRequest(&v1pb.GetIssueRequest{Name: fixture.IssueB.Name}))
	a.NoError(err)
	a.Equal(issueBBefore.Msg.Title, issueBAfter.Msg.Title,
		"project B's issue title leaked from project A update")
	a.Equal(issueBBefore.Msg.Status, issueBAfter.Msg.Status,
		"project B's issue status leaked from project A update")
}

// TestCollisionListPlansIsolation verifies that ListPlans scoped to project A
// does not leak project B's plans — closing the readback oracle for the
// `snapshotProject` helper which relies on this method.
func TestCollisionListPlansIsolation(t *testing.T) {
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

	snap := snapshotProject(ctx, t, s, projectAID)
	for _, p := range snap.Plans {
		a.Equal(projectAID, p.ProjectID,
			"ListPlans for project A returned a row from another project")
	}
}

// TestCollisionListTasksIsolation verifies that ListTasks scoped to project A
// does not leak project B's tasks.
func TestCollisionListTasksIsolation(t *testing.T) {
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

	fixture.completeRolloutB(ctx, t, ctl)

	snap := snapshotProject(ctx, t, s, projectAID)
	for _, tk := range snap.Tasks {
		a.Equal(projectAID, tk.ProjectID,
			"ListTasks for project A returned a row from another project")
	}
}

// TestCollisionListTaskRunsIsolation verifies that listing task_runs for
// project A returns only project A's rows, not project B's.
func TestCollisionListTaskRunsIsolation(t *testing.T) {
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

	fixture.completeRolloutB(ctx, t, ctl)

	snap := snapshotProject(ctx, t, s, projectAID)
	for _, tr := range snap.TaskRuns {
		a.Equal(projectAID, tr.ProjectID,
			"ListTaskRuns for project A returned a row from another project")
	}
}

// TestCollisionListPlanCheckRunsIsolation verifies that listing plan_check_runs
// for project A returns only project A's rows — even when project B has
// plan_check_runs with the same composite-PK (id) values.
//
// We must roll out project B first so it actually has plan_check_runs;
// otherwise the isolation loop is vacuously true.
func TestCollisionListPlanCheckRunsIsolation(t *testing.T) {
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

	fixture.completeRolloutB(ctx, t, ctl)

	snap := snapshotProject(ctx, t, s, projectAID)
	for _, pcr := range snap.PlanCheckRuns {
		a.Equal(projectAID, pcr.ProjectID,
			"ListPlanCheckRuns for project A returned a row from another project")
	}
}

// TestCollisionGetIssueIsolation verifies that GetIssue with project A's
// issue name returns project A's issue, not project B's.
func TestCollisionGetIssueIsolation(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	fixture := setupCollidingProjects(ctx, t, ctl)

	issueA, err := ctl.issueServiceClient.GetIssue(ctx,
		connect.NewRequest(&v1pb.GetIssueRequest{Name: fixture.IssueA.Name}))
	a.NoError(err)
	a.Equal(fixture.IssueA.Name, issueA.Msg.Name,
		"GetIssue returned wrong issue")
	a.Equal("Collision test A", issueA.Msg.Title,
		"GetIssue returned project B's issue title instead of project A's")
}

// TestCollisionListIssuesIsolation verifies that ListIssues scoped to
// project A does not return project B's issues.
func TestCollisionListIssuesIsolation(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	fixture := setupCollidingProjects(ctx, t, ctl)

	resp, err := ctl.issueServiceClient.ListIssues(ctx,
		connect.NewRequest(&v1pb.ListIssuesRequest{
			Parent:   fixture.ProjectA.Name,
			PageSize: 100,
		}))
	a.NoError(err)

	for _, issue := range resp.Msg.Issues {
		a.True(strings.HasPrefix(issue.Name, fixture.ProjectA.Name+"/"),
			"ListIssues for project A returned an issue from another project: %s", issue.Name)
	}
}
