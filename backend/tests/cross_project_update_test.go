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
// does not leak project B's plans.
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

	resp, err := ctl.planServiceClient.ListPlans(ctx,
		connect.NewRequest(&v1pb.ListPlansRequest{Parent: fixture.ProjectA.Name, PageSize: 100}))
	a.NoError(err)

	// Positive precondition — without this, an over-filtering regression
	// that returned zero rows would silently pass the prefix check below.
	a.Greater(len(resp.Msg.Plans), 0, "project A should have plans (fixture creates them)")

	for _, p := range resp.Msg.Plans {
		a.True(strings.HasPrefix(p.Name, fixture.ProjectA.Name+"/"),
			"ListPlans for project A returned a plan from another project: %s", p.Name)
	}
}

// TestCollisionListTaskRunsIsolation verifies that ListTaskRuns scoped to a
// project A rollout does not leak project B's task_runs, using the wildcard
// parent form that the production API supports.
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
	fixture.completeRolloutB(ctx, t, ctl)

	resp, err := ctl.rolloutServiceClient.ListTaskRuns(ctx,
		connect.NewRequest(&v1pb.ListTaskRunsRequest{
			Parent: fixture.PlanA.Name + "/rollout/stages/-/tasks/-",
		}))
	a.NoError(err)

	// Positive precondition: project A's rollout completed during fixture
	// setup, so it must have task_runs. Without this, an over-filtering
	// regression returning zero rows would silently pass the prefix check.
	a.Greater(len(resp.Msg.TaskRuns), 0, "project A should have task_runs after fixture rollout")

	for _, tr := range resp.Msg.TaskRuns {
		a.True(strings.HasPrefix(tr.Name, fixture.ProjectA.Name+"/"),
			"ListTaskRuns for project A returned %s from another project", tr.Name)
	}
}

// TestCollisionGetPlanCheckRunIsolation verifies that GetPlanCheckRun for
// project A's plan returns only project A's check run, even when project B
// has a plan_check_run with the same numeric composite-PK id.
func TestCollisionGetPlanCheckRunIsolation(t *testing.T) {
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

	resp, err := ctl.planServiceClient.GetPlanCheckRun(ctx,
		connect.NewRequest(&v1pb.GetPlanCheckRunRequest{
			Name: fixture.PlanA.Name + "/planCheckRun",
		}))
	a.NoError(err)
	a.True(strings.HasPrefix(resp.Msg.Name, fixture.ProjectA.Name+"/"),
		"GetPlanCheckRun for plan A returned %s from another project", resp.Msg.Name)
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

	// Positive precondition — guard against an over-filter regression
	// that would make the prefix check vacuously true.
	a.Greater(len(resp.Msg.Issues), 0, "project A should have issues (fixture creates them)")

	for _, issue := range resp.Msg.Issues {
		a.True(strings.HasPrefix(issue.Name, fixture.ProjectA.Name+"/"),
			"ListIssues for project A returned an issue from another project: %s", issue.Name)
	}
}
