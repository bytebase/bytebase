package tests

import (
	"context"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common"
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

// TestCollisionCreateRolloutIsolation verifies that the review module's
// transactional Plan/task writes stay scoped to the full project/ID keys.
func TestCollisionCreateRolloutIsolation(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	fixture := setupCollidingProjects(ctx, t, ctl)
	a.Greater(len(fixture.BaselineA.PlanCheckRuns), 0, "project A should have plan_check_runs")
	fixture.completeRolloutB(ctx, t, ctl)
	planCheckRunB, err := ctl.planServiceClient.GetPlanCheckRun(ctx,
		connect.NewRequest(&v1pb.GetPlanCheckRunRequest{
			Name: fixture.PlanB.Name + "/planCheckRun",
		}))
	a.NoError(err)
	a.True(strings.HasPrefix(planCheckRunB.Msg.Name, fixture.ProjectB.Name+"/"),
		"GetPlanCheckRun for plan B returned %s from another project", planCheckRunB.Msg.Name)
	aAfter := snapshotProject(ctx, t, ctl, fixture.ProjectA)
	assertProjectUnchanged(t, fixture.BaselineA, aAfter, "project A after project B rollout")
}

// TestCollisionBatchRunTasksNoCrossProjectEffect verifies that creating a
// pending task run in project B cannot mutate project A's colliding task or
// task_run rows.
func TestCollisionBatchRunTasksNoCrossProjectEffect(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	fixture := setupCollidingProjects(ctx, t, ctl)
	a.Greater(len(fixture.BaselineA.TaskRuns), 0, "project A should have task_runs")

	rolloutB, err := ctl.rolloutServiceClient.CreateRollout(ctx,
		connect.NewRequest(&v1pb.CreateRolloutRequest{Parent: fixture.PlanB.Name}))
	a.NoError(err)
	a.Greater(len(rolloutB.Msg.Stages), 0, "project B rollout should have stages")
	a.Greater(len(rolloutB.Msg.Stages[0].Tasks), 0, "project B rollout should have tasks")
	taskB := rolloutB.Msg.Stages[0].Tasks[0]

	_, err = ctl.rolloutServiceClient.BatchRunTasks(ctx,
		connect.NewRequest(&v1pb.BatchRunTasksRequest{
			Parent:  rolloutB.Msg.Stages[0].Name,
			Tasks:   []string{taskB.Name},
			RunTime: timestamppb.New(time.Now().Add(time.Hour)),
		}))
	a.NoError(err)

	assertTasksCollide(ctx, t, ctl, fixture)
	assertTaskRunsCollide(ctx, t, ctl, fixture)
	aAfter := snapshotProject(ctx, t, ctl, fixture.ProjectA)
	assertProjectUnchanged(t, fixture.BaselineA, aAfter, "project A after project B task run creation")
}

// TestCollisionBatchSkipTasksNoCrossProjectEffect verifies that skipping a
// task in project B cannot mark project A's same-id task as skipped.
func TestCollisionBatchSkipTasksNoCrossProjectEffect(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	fixture := setupCollidingProjects(ctx, t, ctl)
	a.Greater(len(fixture.BaselineA.TaskRuns), 0, "project A should have task_runs")

	rolloutA, err := ctl.rolloutServiceClient.GetRollout(ctx,
		connect.NewRequest(&v1pb.GetRolloutRequest{Name: fixture.PlanA.Name + "/rollout"}))
	a.NoError(err)
	rolloutB, err := ctl.rolloutServiceClient.CreateRollout(ctx,
		connect.NewRequest(&v1pb.CreateRolloutRequest{Parent: fixture.PlanB.Name}))
	a.NoError(err)
	a.Greater(len(rolloutA.Msg.Stages), 0, "project A rollout should have stages")
	a.Greater(len(rolloutA.Msg.Stages[0].Tasks), 0, "project A rollout should have tasks")
	a.Greater(len(rolloutB.Msg.Stages), 0, "project B rollout should have stages")
	a.Greater(len(rolloutB.Msg.Stages[0].Tasks), 0, "project B rollout should have tasks")
	taskA := rolloutA.Msg.Stages[0].Tasks[0]
	taskB := rolloutB.Msg.Stages[0].Tasks[0]
	_, _, _, taskAID, err := common.GetProjectIDPlanIDStageIDTaskID(taskA.Name)
	a.NoError(err)
	_, _, _, taskBID, err := common.GetProjectIDPlanIDStageIDTaskID(taskB.Name)
	a.NoError(err)
	a.Equal(taskAID, taskBID, "project A and B task IDs should collide")

	_, err = ctl.rolloutServiceClient.BatchSkipTasks(ctx,
		connect.NewRequest(&v1pb.BatchSkipTasksRequest{
			Parent: rolloutB.Msg.Stages[0].Name,
			Tasks:  []string{taskB.Name},
			Reason: "collision isolation",
		}))
	a.NoError(err)
	rolloutBAfter, err := ctl.rolloutServiceClient.GetRollout(ctx,
		connect.NewRequest(&v1pb.GetRolloutRequest{Name: rolloutB.Msg.Name}))
	a.NoError(err)
	a.Equal(v1pb.Task_SKIPPED, rolloutBAfter.Msg.Stages[0].Tasks[0].Status,
		"project B task should be skipped")

	aAfter := snapshotProject(ctx, t, ctl, fixture.ProjectA)
	assertProjectUnchanged(t, fixture.BaselineA, aAfter, "project A after project B task skip")
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
