package tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/type/expr"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/bytebase/bytebase/backend/common"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// TestCollision_PlanWebhookDeliveryWrite verifies that claiming a
// PIPELINE_COMPLETED webhook delivery for a plan in project A writes a
// plan_webhook_delivery row scoped by (project, plan_id) without touching
// project B's rows — even though the two projects' plan ids collide under
// the composite primary key.
func TestCollision_PlanWebhookDeliveryWrite(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	fixture := setupCollidingProjects(ctx, t, ctl)

	// Give project B a completed rollout too, so B owns a
	// plan_webhook_delivery row whose plan id collides with the fresh
	// project-A plan we are about to roll out.
	planB2, issueB2, rolloutB2 := createPlanIssueRollout(ctx, t, ctl, fixture.ProjectB, fixture.DatabaseB, "webhook delivery test B")
	a.NoError(ctl.waitRollout(ctx, issueB2.Name, rolloutB2.Name))

	projectBID, err := common.GetProjectID(fixture.ProjectB.Name)
	a.NoError(err)
	_, planB2UID, err := common.GetProjectIDPlanID(planB2.Name)
	a.NoError(err)
	// The scheduler claims the delivery row asynchronously after the last
	// task is marked DONE, so wait for the row to land before snapshotting.
	waitForPlanWebhookDelivery(ctx, t, ctl, projectBID, planB2UID)
	// The fixture's create-database plan also claims PIPELINE_COMPLETED
	// asynchronously during setup; wait for the whole row set to settle so a
	// late legitimate claim cannot masquerade as a cross-project leak.
	waitForPlanWebhookDeliveriesStable(ctx, t, ctl, projectBID)

	beforeB := snapshotProject(ctx, t, ctl, fixture.ProjectB)
	a.NotEmpty(beforeB.PlanWebhookDeliveries, "project B should own PIPELINE_COMPLETED delivery rows")
	foundB2Delivery := false
	for _, d := range beforeB.PlanWebhookDeliveries {
		if d.PlanID == planB2UID {
			foundB2Delivery = true
			break
		}
	}
	a.True(foundB2Delivery, "project B should own the delivery row for its plan %d", planB2UID)

	// Roll out a fresh plan in project A whose plan id collides with B2's.
	planA2, issueA2, rolloutA2 := createPlanIssueRollout(ctx, t, ctl, fixture.ProjectA, fixture.DatabaseA, "webhook delivery test A")
	_, planA2UID, err := common.GetProjectIDPlanID(planA2.Name)
	a.NoError(err)
	a.Equal(planA2UID, planB2UID, "plan ids must collide across projects for the composite-PK case")

	a.NoError(ctl.waitRollout(ctx, issueA2.Name, rolloutA2.Name))

	// Positive check: project A actually claimed its delivery row. Without
	// this, a regression that turned the claim into a no-op (e.g. resolving
	// project B's plan by colliding plan id alone) would still pass the
	// project-B-unchanged check below.
	projectAID, err := common.GetProjectID(fixture.ProjectA.Name)
	a.NoError(err)
	waitForPlanWebhookDelivery(ctx, t, ctl, projectAID, planA2UID)
	// Project B's delivery row set was stabilized before the baseline; the
	// webhook table is async-claimed so it is compared here explicitly
	// rather than inside the generic assertProjectUnchanged.
	waitForPlanWebhookDeliveriesStable(ctx, t, ctl, projectBID)

	afterB := snapshotProject(ctx, t, ctl, fixture.ProjectB)
	assertProjectUnchanged(t, beforeB, afterB, "project B after plan A webhook delivery claim")
	assertNoChange(t, beforeB.PlanWebhookDeliveries, afterB.PlanWebhookDeliveries,
		func(d *planWebhookDelivery) string { return fmt.Sprintf("%d", d.PlanID) },
		func(b, af *planWebhookDelivery) {
			a.Equal(b.EventType, af.EventType, "project B plan_webhook_delivery plan %d event_type changed", b.PlanID)
			a.True(b.DeliveredAt.Equal(af.DeliveredAt), "project B plan_webhook_delivery plan %d delivered_at changed", b.PlanID)
		},
		"project B after plan A webhook delivery claim", "plan_webhook_delivery")
}

// waitForPlanWebhookDelivery polls the metadata DB until the
// plan_webhook_delivery row for (projectID, planUID) appears. The claim
// runs in the scheduler after the rollout's last task is marked DONE, so
// the row may land slightly after waitRollout returns.
func waitForPlanWebhookDelivery(ctx context.Context, t *testing.T, ctl *controller, projectID string, planUID int64) {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for {
		for _, d := range listPlanWebhookDeliveries(ctx, t, ctl, projectID) {
			if d.PlanID == planUID {
				return
			}
		}
		if time.Now().After(deadline) {
			require.Failf(t, "timed out waiting for plan webhook delivery",
				"no plan_webhook_delivery row for project %s plan %d", projectID, planUID)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// waitForPlanWebhookDeliveriesStable polls until the project's
// plan_webhook_delivery row set stops changing for a full second. Delivery
// claims run in the scheduler after a rollout completes, so a snapshot must
// not race a legitimate late claim.
func waitForPlanWebhookDeliveriesStable(ctx context.Context, t *testing.T, ctl *controller, projectID string) {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	lastCount := -1
	stableSince := time.Now()
	for {
		count := len(listPlanWebhookDeliveries(ctx, t, ctl, projectID))
		if count != lastCount {
			lastCount = count
			stableSince = time.Now()
		}
		if time.Since(stableSince) >= time.Second {
			return
		}
		if time.Now().After(deadline) {
			require.Failf(t, "timed out waiting for plan webhook deliveries to stabilize",
				"project %s delivery rows kept changing; last count %d", projectID, lastCount)
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// TestCollision_TaskRunLogWrite verifies that executing a rollout in
// project A writes task_run_log rows scoped by (project, task_run_id)
// without touching project B's rows, even though task run ids collide
// across the two projects.
func TestCollision_TaskRunLogWrite(t *testing.T) {
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

	// Give project B a second completed rollout (plan B2, task run UID 2)
	// so B owns task_run_log rows whose ids collide with the fresh
	// project-A rollout we are about to run.
	planB2, issueB2, rolloutB2 := createPlanIssueRollout(ctx, t, ctl, fixture.ProjectB, fixture.DatabaseB, "task run log test B")
	a.NoError(ctl.waitRollout(ctx, issueB2.Name, rolloutB2.Name))

	beforeB := snapshotProject(ctx, t, ctl, fixture.ProjectB)
	a.Greater(len(beforeB.TaskRunLogs), 0, "project B should own task_run_log rows after its rollout")

	// Roll out a fresh plan in project A whose task run ids collide with
	// B2's.
	planA2, issueA2, rolloutA2 := createPlanIssueRollout(ctx, t, ctl, fixture.ProjectA, fixture.DatabaseA, "task run log test A")
	a.NoError(ctl.waitRollout(ctx, issueA2.Name, rolloutA2.Name))

	// The task runs of the second pair must genuinely collide, otherwise a
	// cross-project write could not be detected (B has no task run with
	// A2's UID, so the FK on (project, task_run_id) would reject it).
	a2TaskRunUIDs, _ := listTaskRunAndTaskUIDs(ctx, t, ctl, planA2.Name)
	b2TaskRunUIDs, _ := listTaskRunAndTaskUIDs(ctx, t, ctl, planB2.Name)
	a.Greater(len(a2TaskRunUIDs), 0, "project A's fresh plan should have task runs")
	assertAtLeastOneUIDCollides(t, a2TaskRunUIDs, b2TaskRunUIDs, "task_run (second pair)")

	// Positive check: project A's fresh task run actually wrote log rows.
	a2Logs := listTaskRunLogsForPlan(ctx, t, ctl, planA2.Name)
	a.NotEmpty(a2Logs, "project A plan A2 should have task runs")
	totalEntries := 0
	for _, l := range a2Logs {
		totalEntries += len(l.Entries)
	}
	a.Greater(totalEntries, 0, "project A's plan A2 task run should have written task_run_log entries")

	afterB := snapshotProject(ctx, t, ctl, fixture.ProjectB)
	assertProjectUnchanged(t, beforeB, afterB, "project B after task run log write in project A")
}

// listTaskRunLogsForPlan returns the task run log messages for every task
// run under the plan, via the public GetTaskRunLog API.
func listTaskRunLogsForPlan(ctx context.Context, t *testing.T, ctl *controller, planName string) []*v1pb.TaskRunLog {
	t.Helper()
	a := require.New(t)
	resp, err := ctl.rolloutServiceClient.ListTaskRuns(ctx, connect.NewRequest(&v1pb.ListTaskRunsRequest{
		Parent: planName + "/rollout/stages/-/tasks/-",
	}))
	a.NoError(err, "ListTaskRuns(%s)", planName)
	var logs []*v1pb.TaskRunLog
	for _, tr := range resp.Msg.TaskRuns {
		logResp, err := ctl.rolloutServiceClient.GetTaskRunLog(ctx, connect.NewRequest(&v1pb.GetTaskRunLogRequest{
			Parent: tr.Name,
		}))
		a.NoError(err, "GetTaskRunLog(%s)", tr.Name)
		logs = append(logs, logResp.Msg)
	}
	return logs
}

// TestCollision_DbGroupWrite verifies that creating and updating a
// database group in project A writes db_group rows scoped by
// (project, resource_id) without touching project B's rows, even though
// both projects use the same group resource id.
func TestCollision_DbGroupWrite(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	fixture := setupCollidingProjects(ctx, t, ctl)

	// Project B owns the baseline group with resource id "collide-group".
	_, err = ctl.databaseGroupServiceClient.CreateDatabaseGroup(ctx, connect.NewRequest(&v1pb.CreateDatabaseGroupRequest{
		Parent:          fixture.ProjectB.Name,
		DatabaseGroupId: "collide-group",
		DatabaseGroup: &v1pb.DatabaseGroup{
			Title:        "B group",
			DatabaseExpr: &expr.Expr{Expression: "true"},
		},
	}))
	a.NoError(err)

	beforeB := snapshotProject(ctx, t, ctl, fixture.ProjectB)
	a.Len(beforeB.DatabaseGroups, 1, "project B should own one database group")

	// Same resource id in project A — (project, resource_id) collides.
	groupA, err := ctl.databaseGroupServiceClient.CreateDatabaseGroup(ctx, connect.NewRequest(&v1pb.CreateDatabaseGroupRequest{
		Parent:          fixture.ProjectA.Name,
		DatabaseGroupId: "collide-group",
		DatabaseGroup: &v1pb.DatabaseGroup{
			Title:        "A group",
			DatabaseExpr: &expr.Expr{Expression: "true"},
		},
	}))
	a.NoError(err)
	_, groupAID, err := common.GetProjectIDDatabaseGroupID(groupA.Msg.Name)
	a.NoError(err)
	a.Equal("collide-group", groupAID, "database group resource id must collide across projects")

	// Exercise the update writer (Store.UpdateDatabaseGroup) as well.
	_, err = ctl.databaseGroupServiceClient.UpdateDatabaseGroup(ctx, connect.NewRequest(&v1pb.UpdateDatabaseGroupRequest{
		DatabaseGroup: &v1pb.DatabaseGroup{
			Name:         groupA.Msg.Name,
			Title:        "A group updated",
			DatabaseExpr: &expr.Expr{Expression: `resource.database_name.startsWith("collision_")`},
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"title", "database_expr"}},
	}))
	a.NoError(err)

	// Positive check: the update landed on project A's group, not B's.
	gotA, err := ctl.databaseGroupServiceClient.GetDatabaseGroup(ctx, connect.NewRequest(&v1pb.GetDatabaseGroupRequest{
		Name: groupA.Msg.Name,
	}))
	a.NoError(err)
	a.Equal("A group updated", gotA.Msg.Title)

	afterB := snapshotProject(ctx, t, ctl, fixture.ProjectB)
	assertProjectUnchanged(t, beforeB, afterB, "project B after database group write in project A")
}

// TestCollision_ReleaseWrite verifies that creating a release in project A
// writes a release row scoped by (project, train, iteration) without
// touching project B's rows, even though both projects produce the same
// release id (same train and iteration).
func TestCollision_ReleaseWrite(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	fixture := setupCollidingProjects(ctx, t, ctl)

	const releaseIDTemplate = "collision_{date}-RC{iteration}"
	createRelease := func(project *v1pb.Project, category string) *v1pb.Release {
		t.Helper()
		resp, err := ctl.releaseServiceClient.CreateRelease(ctx, connect.NewRequest(&v1pb.CreateReleaseRequest{
			Parent:            project.Name,
			ReleaseIdTemplate: releaseIDTemplate,
			Release: &v1pb.Release{
				Type:     v1pb.Release_VERSIONED,
				Category: category,
				Files: []*v1pb.Release_File{{
					Path:      "V0001__create.sql",
					Version:   "1",
					Statement: []byte("CREATE TABLE t1 (id int);"),
				}},
			},
		}))
		a.NoError(err, "CreateRelease(%s)", project.Name)
		return resp.Msg
	}

	// Project B owns the baseline release with the same train as project A's.
	releaseB := createRelease(fixture.ProjectB, "b")

	beforeB := snapshotProject(ctx, t, ctl, fixture.ProjectB)
	a.Len(beforeB.Releases, 1, "project B should own one release")

	// Same template and first iteration (0) in both projects — the
	// (project, train, iteration) primary key collides across projects.
	releaseA := createRelease(fixture.ProjectA, "a")
	_, releaseAID, err := common.GetProjectReleaseID(releaseA.Name)
	a.NoError(err)
	_, releaseBID, err := common.GetProjectReleaseID(releaseB.Name)
	a.NoError(err)
	a.Equal(releaseBID, releaseAID, "release ids must collide across projects (same train and iteration)")

	// Positive check: project A really owns its release.
	gotA := snapshotProject(ctx, t, ctl, fixture.ProjectA)
	a.Len(gotA.Releases, 1, "project A should own its release")

	afterB := snapshotProject(ctx, t, ctl, fixture.ProjectB)
	assertProjectUnchanged(t, beforeB, afterB, "project B after release write in project A")
}
