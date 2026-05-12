package tests

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/bytebase/bytebase/backend/common"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// collisionFixture holds references to entities created in two projects
// whose composite-PK ids naturally collide.
//
// When created by setupCollidingProjects, both projects share a single
// instance (Instance == InstanceA == InstanceB). When created by
// setupCollidingProjectsSeparateInstances, InstanceA and InstanceB are
// distinct — Instance is unset in that case.
type collisionFixture struct {
	ProjectA *v1pb.Project
	ProjectB *v1pb.Project
	// Instance is set only when both projects share an instance.
	Instance  *v1pb.Instance
	InstanceA *v1pb.Instance
	InstanceB *v1pb.Instance

	DatabaseA *v1pb.Database
	DatabaseB *v1pb.Database

	PlanA *v1pb.Plan
	PlanB *v1pb.Plan

	IssueA *v1pb.Issue
	IssueB *v1pb.Issue

	// BaselineA is project A's snapshot captured immediately after A's
	// rollout completes and BEFORE project B's plan/issue are created.
	// Tests should use this as the "before" oracle rather than calling
	// snapshotProject() at the start of the test, because the scheduler
	// may fire cross-project side effects during project B's creation.
	BaselineA *projectSnapshot
}

// setupCollidingProjects creates two projects with naturally colliding ids.
//
// Both projects get a plan → issue → rollout with a DML task targeting
// separate databases on the same SQLite instance. Because nextProjectID
// allocates per-project, the first plan/task/task_run/plan_check_run in
// each project receives the same numeric id.
//
// Project A's rollout is driven to DONE. Project B's rollout is left
// in a state controlled by the caller (default: tasks are NOT_STARTED).
func setupCollidingProjects(
	ctx context.Context,
	t *testing.T,
	ctl *controller,
) *collisionFixture {
	t.Helper()
	a := require.New(t)

	projectA, err := ctl.projectServiceClient.CreateProject(ctx, connect.NewRequest(&v1pb.CreateProjectRequest{
		ProjectId: generateRandomString("col-a"),
		Project:   &v1pb.Project{Title: "Collision Project A", AllowSelfApproval: true},
	}))
	a.NoError(err)

	projectB, err := ctl.projectServiceClient.CreateProject(ctx, connect.NewRequest(&v1pb.CreateProjectRequest{
		ProjectId: generateRandomString("col-b"),
		Project:   &v1pb.Project{Title: "Collision Project B", AllowSelfApproval: true},
	}))
	a.NoError(err)

	instanceRootDir := t.TempDir()
	instanceDir, err := ctl.provisionSQLiteInstance(instanceRootDir, "collision-instance")
	a.NoError(err)

	instanceResp, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("col-inst"),
		Instance: &v1pb.Instance{
			Title:       "collision-instance",
			Engine:      v1pb.Engine_SQLITE,
			Environment: new("environments/prod"),
			Activation:  true,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: instanceDir, Id: "admin"}},
		},
	}))
	a.NoError(err)
	instance := instanceResp.Msg

	dbNameA := "collision_db_a"
	err = ctl.createDatabase(ctx, projectA.Msg, instance, nil, dbNameA, "")
	a.NoError(err)

	dbNameB := "collision_db_b"
	err = ctl.createDatabase(ctx, projectB.Msg, instance, nil, dbNameB, "")
	a.NoError(err)

	dbA, err := ctl.databaseServiceClient.GetDatabase(ctx, connect.NewRequest(&v1pb.GetDatabaseRequest{
		Name: fmt.Sprintf("%s/databases/%s", instance.Name, dbNameA),
	}))
	a.NoError(err)

	dbB, err := ctl.databaseServiceClient.GetDatabase(ctx, connect.NewRequest(&v1pb.GetDatabaseRequest{
		Name: fmt.Sprintf("%s/databases/%s", instance.Name, dbNameB),
	}))
	a.NoError(err)

	sheetA, err := ctl.sheetServiceClient.CreateSheet(ctx, connect.NewRequest(&v1pb.CreateSheetRequest{
		Parent: projectA.Msg.Name,
		Sheet:  &v1pb.Sheet{Content: []byte("SELECT 1;")},
	}))
	a.NoError(err)

	planA, err := ctl.planServiceClient.CreatePlan(ctx, connect.NewRequest(&v1pb.CreatePlanRequest{
		Parent: projectA.Msg.Name,
		Plan: &v1pb.Plan{
			Specs: []*v1pb.Plan_Spec{{
				Id: uuid.NewString(),
				Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{
					ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
						Targets: []string{dbA.Msg.Name},
						Sheet:   sheetA.Msg.Name,
					},
				},
			}},
		},
	}))
	a.NoError(err)

	issueA, err := ctl.issueServiceClient.CreateIssue(ctx, connect.NewRequest(&v1pb.CreateIssueRequest{
		Parent: projectA.Msg.Name,
		Issue: &v1pb.Issue{
			Title: "Collision test A",
			Type:  v1pb.Issue_DATABASE_CHANGE,
			Plan:  planA.Msg.Name,
		},
	}))
	a.NoError(err)

	rolloutA, err := ctl.rolloutServiceClient.CreateRollout(ctx, connect.NewRequest(&v1pb.CreateRolloutRequest{
		Parent: planA.Msg.Name,
	}))
	a.NoError(err)

	err = ctl.waitRollout(ctx, issueA.Msg.Name, rolloutA.Msg.Name)
	a.NoError(err)

	// Capture project A's baseline BEFORE creating project B's plan. Anything
	// below (CreatePlan, CreateIssue) can tickle the scheduler; a cross-project
	// bug would fire here and corrupt project A. By snapshotting now, tests
	// get a clean oracle that reflects A's true post-rollout state.
	baselineA := snapshotProject(ctx, t, ctl, projectA.Msg)

	sheetB, err := ctl.sheetServiceClient.CreateSheet(ctx, connect.NewRequest(&v1pb.CreateSheetRequest{
		Parent: projectB.Msg.Name,
		Sheet:  &v1pb.Sheet{Content: []byte("SELECT 1;")},
	}))
	a.NoError(err)

	planB, err := ctl.planServiceClient.CreatePlan(ctx, connect.NewRequest(&v1pb.CreatePlanRequest{
		Parent: projectB.Msg.Name,
		Plan: &v1pb.Plan{
			Specs: []*v1pb.Plan_Spec{{
				Id: uuid.NewString(),
				Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{
					ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
						Targets: []string{dbB.Msg.Name},
						Sheet:   sheetB.Msg.Name,
					},
				},
			}},
		},
	}))
	a.NoError(err)

	issueB, err := ctl.issueServiceClient.CreateIssue(ctx, connect.NewRequest(&v1pb.CreateIssueRequest{
		Parent: projectB.Msg.Name,
		Issue: &v1pb.Issue{
			Title: "Collision test B",
			Type:  v1pb.Issue_DATABASE_CHANGE,
			Plan:  planB.Msg.Name,
		},
	}))
	a.NoError(err)

	f := &collisionFixture{
		ProjectA:  projectA.Msg,
		ProjectB:  projectB.Msg,
		Instance:  instance,
		InstanceA: instance,
		InstanceB: instance,
		DatabaseA: dbA.Msg,
		DatabaseB: dbB.Msg,
		PlanA:     planA.Msg,
		PlanB:     planB.Msg,
		IssueA:    issueA.Msg,
		IssueB:    issueB.Msg,
		BaselineA: baselineA,
	}
	assertFixtureIDsCollide(ctx, t, ctl, f)
	return f
}

// setupCollidingProjectsSeparateInstances is a variant of setupCollidingProjects
// that places each project on its own SQLite instance. Useful for tests that
// need to distinguish correct per-instance cascade from a buggy cross-project
// DELETE USING predicate. Project A's rollout is driven to DONE; project B's
// rollout is left for the caller to trigger.
func setupCollidingProjectsSeparateInstances(
	ctx context.Context,
	t *testing.T,
	ctl *controller,
) *collisionFixture {
	t.Helper()
	a := require.New(t)

	projectA, err := ctl.projectServiceClient.CreateProject(ctx, connect.NewRequest(&v1pb.CreateProjectRequest{
		ProjectId: generateRandomString("col-a"),
		Project:   &v1pb.Project{Title: "Collision Project A", AllowSelfApproval: true},
	}))
	a.NoError(err)

	projectB, err := ctl.projectServiceClient.CreateProject(ctx, connect.NewRequest(&v1pb.CreateProjectRequest{
		ProjectId: generateRandomString("col-b"),
		Project:   &v1pb.Project{Title: "Collision Project B", AllowSelfApproval: true},
	}))
	a.NoError(err)

	instA := createSQLiteInstance(ctx, t, ctl, "col-inst-a")
	instB := createSQLiteInstance(ctx, t, ctl, "col-inst-b")

	const dbNameA = "collision_db_a"
	a.NoError(ctl.createDatabase(ctx, projectA.Msg, instA, nil, dbNameA, ""))
	dbA, err := ctl.databaseServiceClient.GetDatabase(ctx, connect.NewRequest(&v1pb.GetDatabaseRequest{
		Name: fmt.Sprintf("%s/databases/%s", instA.Name, dbNameA),
	}))
	a.NoError(err)

	const dbNameB = "collision_db_b"
	a.NoError(ctl.createDatabase(ctx, projectB.Msg, instB, nil, dbNameB, ""))
	dbB, err := ctl.databaseServiceClient.GetDatabase(ctx, connect.NewRequest(&v1pb.GetDatabaseRequest{
		Name: fmt.Sprintf("%s/databases/%s", instB.Name, dbNameB),
	}))
	a.NoError(err)

	planA, issueA, rolloutA := createPlanIssueRollout(ctx, t, ctl, projectA.Msg, dbA.Msg, "Collision test A")
	a.NoError(ctl.waitRollout(ctx, issueA.Name, rolloutA.Name))

	// Snapshot A BEFORE creating project B's plan — see note in setupCollidingProjects.
	baselineA := snapshotProject(ctx, t, ctl, projectA.Msg)

	planB, issueB := createPlanAndIssue(ctx, t, ctl, projectB.Msg, dbB.Msg, "Collision test B")

	f := &collisionFixture{
		ProjectA:  projectA.Msg,
		ProjectB:  projectB.Msg,
		InstanceA: instA,
		InstanceB: instB,
		DatabaseA: dbA.Msg,
		DatabaseB: dbB.Msg,
		PlanA:     planA,
		PlanB:     planB,
		IssueA:    issueA,
		IssueB:    issueB,
		BaselineA: baselineA,
	}
	assertFixtureIDsCollide(ctx, t, ctl, f)
	return f
}

// assertFixtureIDsCollide verifies that the "colliding IDs" invariant the
// whole harness depends on is actually true. If a future change adds an
// extra project-scoped allocation on either path, the tests would silently
// stop exercising the composite-PK collision case; this assertion fails
// fast in that case.
//
// Coverage matrix (by table):
//   - plan       — asserted here (both projects have plans at fixture time)
//   - issue      — asserted here (both projects have issues at fixture time)
//   - task       — asserted in completeRolloutB (B's rollout required)
//   - task_run   — asserted in completeRolloutB (B's rollout required)
//   - plan_check_run — NOT ASSERTED. The v1 API exposes PCRs via a
//     UID-less singleton name, so the UID is not observable from public
//     gRPC. nextProjectID is per-table, so task_run collision does NOT
//     imply PCR collision. The PCR claim test should be treated as
//     belt-and-suspenders for the task_run claim test, not an
//     independent regression lock.
func assertFixtureIDsCollide(ctx context.Context, t *testing.T, ctl *controller, f *collisionFixture) {
	t.Helper()
	a := require.New(t)

	aPlans := listPlanUIDs(ctx, t, ctl, f.ProjectA.Name)
	bPlans := listPlanUIDs(ctx, t, ctl, f.ProjectB.Name)
	a.Greater(len(aPlans), 0, "project A should have at least one plan")
	a.Greater(len(bPlans), 0, "project B should have at least one plan")
	assertAtLeastOneUIDCollides(t, aPlans, bPlans, "plan")

	aIssues := listIssueUIDs(ctx, t, ctl, f.ProjectA.Name)
	bIssues := listIssueUIDs(ctx, t, ctl, f.ProjectB.Name)
	a.Greater(len(aIssues), 0, "project A should have at least one issue")
	a.Greater(len(bIssues), 0, "project B should have at least one issue")
	assertAtLeastOneUIDCollides(t, aIssues, bIssues, "issue")
}

// completeRolloutB drives project B's rollout to completion and proves that
// task and task_run ids collide across the two projects. This is the ONLY
// supported way for a collision test to roll out B — every call site gets
// the collision invariants for free, so no test can silently become vacuous
// by forgetting an assertion.
//
// Known coverage gap — plan_check_run:
// The v1 API exposes plan_check_runs via a UID-less singleton name
// ({plan.name}/planCheckRun), so PCR UIDs cannot be observed from public
// gRPC. The store's nextProjectID allocator is keyed per-table per-project,
// so task_run collision is NOT evidence of PCR collision — the two
// sequences can diverge independently. This means
// TestClaimAvailablePlanCheckRunsNoCrossProjectTransition exercises the
// PCR claim SQL path (via the scheduler) but cannot prove its assertions
// fire against a genuinely colliding row. The regression lock for the
// BYT-9259 SQL pattern is TestClaimAvailableTaskRunsNoCrossProjectResurrection,
// which does prove collision. The PCR test is kept as belt-and-suspenders
// coverage; do not read it as an independent guarantee.
//
// Why this method lives here rather than inside setupCollidingProjects:
// the fixture returns before B's rollout fires, so task and task_run don't
// exist yet at fixture-construction time. Tests that don't need B's
// rollout (pure read-isolation tests on plan/issue) don't pay the cost.
func (f *collisionFixture) completeRolloutB(ctx context.Context, t *testing.T, ctl *controller) {
	t.Helper()
	a := require.New(t)

	rolloutB, err := ctl.rolloutServiceClient.CreateRollout(ctx,
		connect.NewRequest(&v1pb.CreateRolloutRequest{
			Parent: f.PlanB.Name,
		}))
	a.NoError(err)
	a.NoError(ctl.waitRollout(ctx, f.IssueB.Name, rolloutB.Msg.Name))

	assertTasksCollide(ctx, t, ctl, f)
	assertTaskRunsCollide(ctx, t, ctl, f)
}

// assertTaskRunsCollide verifies that after both projects have rolled out,
// their task_run ids collide. Call this from tests that run project B's
// rollout and depend on task_run collisions.
//
// Note: nextProjectID allocates per table, so task_run collision does not
// imply plan_check_run collision. Each table needs its own assertion.
func assertTaskRunsCollide(ctx context.Context, t *testing.T, ctl *controller, f *collisionFixture) {
	t.Helper()
	a := require.New(t)

	aTaskRuns, _ := listTaskRunAndTaskUIDs(ctx, t, ctl, f.PlanA.Name)
	bTaskRuns, _ := listTaskRunAndTaskUIDs(ctx, t, ctl, f.PlanB.Name)
	a.Greater(len(aTaskRuns), 0, "project A should have at least one task_run")
	a.Greater(len(bTaskRuns), 0, "project B should have at least one task_run — did you forget to roll out B?")
	assertAtLeastOneUIDCollides(t, aTaskRuns, bTaskRuns, "task_run")
}

// assertTasksCollide verifies that task UIDs collide across projects.
// Task UIDs are embedded in task_run resource names, so we can derive
// them without a separate ListTasks API (which gRPC does not expose).
func assertTasksCollide(ctx context.Context, t *testing.T, ctl *controller, f *collisionFixture) {
	t.Helper()
	a := require.New(t)

	_, aTasks := listTaskRunAndTaskUIDs(ctx, t, ctl, f.PlanA.Name)
	_, bTasks := listTaskRunAndTaskUIDs(ctx, t, ctl, f.PlanB.Name)
	a.Greater(len(aTasks), 0, "project A should have at least one task")
	a.Greater(len(bTasks), 0, "project B should have at least one task — did you forget to roll out B?")
	assertAtLeastOneUIDCollides(t, aTasks, bTasks, "task")
}

func assertAtLeastOneUIDCollides(t *testing.T, aUIDs, bUIDs []int64, label string) {
	t.Helper()
	aSet := make(map[int64]bool, len(aUIDs))
	for _, u := range aUIDs {
		aSet[u] = true
	}
	for _, u := range bUIDs {
		if aSet[u] {
			return
		}
	}
	require.Failf(t, "fixture invariant broken",
		"projects A and B should have at least one %s UID in common. Got A=%v, B=%v", label, aUIDs, bUIDs)
}

// listPlanUIDs returns every plan UID in the project, via the public gRPC
// ListPlans API. UIDs are parsed from the resource names.
func listPlanUIDs(ctx context.Context, t *testing.T, ctl *controller, projectName string) []int64 {
	t.Helper()
	a := require.New(t)
	resp, err := ctl.planServiceClient.ListPlans(ctx, connect.NewRequest(&v1pb.ListPlansRequest{
		Parent:   projectName,
		PageSize: 1000,
	}))
	a.NoError(err, "ListPlans(%s)", projectName)
	out := make([]int64, 0, len(resp.Msg.Plans))
	for _, p := range resp.Msg.Plans {
		_, uid, err := common.GetProjectIDPlanID(p.Name)
		a.NoError(err, "parse plan name %q", p.Name)
		out = append(out, uid)
	}
	return out
}

// listIssueUIDs returns every issue UID in the project.
func listIssueUIDs(ctx context.Context, t *testing.T, ctl *controller, projectName string) []int64 {
	t.Helper()
	a := require.New(t)
	resp, err := ctl.issueServiceClient.ListIssues(ctx, connect.NewRequest(&v1pb.ListIssuesRequest{
		Parent:   projectName,
		PageSize: 1000,
	}))
	a.NoError(err, "ListIssues(%s)", projectName)
	out := make([]int64, 0, len(resp.Msg.Issues))
	for _, i := range resp.Msg.Issues {
		_, uid, err := common.GetProjectIDIssueUID(i.Name)
		a.NoError(err, "parse issue name %q", i.Name)
		out = append(out, uid)
	}
	return out
}

// listTaskRunAndTaskUIDs returns every task_run UID and the set of task UIDs
// underneath a plan's rollout. Both come from parsing task_run names
// (projects/X/plans/Y/rollout/stages/Z/tasks/{taskUID}/taskRuns/{taskRunUID}),
// so one ListTaskRuns call covers both composite-PK tables without needing
// a separate ListTasks gRPC (which doesn't exist).
func listTaskRunAndTaskUIDs(ctx context.Context, t *testing.T, ctl *controller, planName string) (taskRunUIDs, taskUIDs []int64) {
	t.Helper()
	a := require.New(t)
	resp, err := ctl.rolloutServiceClient.ListTaskRuns(ctx, connect.NewRequest(&v1pb.ListTaskRunsRequest{
		Parent: planName + "/rollout/stages/-/tasks/-",
	}))
	a.NoError(err, "ListTaskRuns(%s)", planName)
	seenTasks := make(map[int64]bool)
	for _, tr := range resp.Msg.TaskRuns {
		_, _, _, taskUID, trUID, err := common.GetProjectIDPlanIDStageIDTaskIDTaskRunID(tr.Name)
		a.NoError(err, "parse task_run name %q", tr.Name)
		taskRunUIDs = append(taskRunUIDs, trUID)
		if !seenTasks[taskUID] {
			seenTasks[taskUID] = true
			taskUIDs = append(taskUIDs, taskUID)
		}
	}
	return taskRunUIDs, taskUIDs
}

// createSQLiteInstance provisions and registers a SQLite instance for test use.
func createSQLiteInstance(ctx context.Context, t *testing.T, ctl *controller, titlePrefix string) *v1pb.Instance {
	t.Helper()
	a := require.New(t)
	instanceDir, err := ctl.provisionSQLiteInstance(t.TempDir(), titlePrefix)
	a.NoError(err)
	resp, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString(titlePrefix),
		Instance: &v1pb.Instance{
			Title:       titlePrefix,
			Engine:      v1pb.Engine_SQLITE,
			Environment: new("environments/prod"),
			Activation:  true,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: instanceDir, Id: "admin"}},
		},
	}))
	a.NoError(err)
	return resp.Msg
}

// createPlanAndIssue creates a plan + issue for a DML change targeting the given
// database, without triggering a rollout. The caller must create the rollout.
func createPlanAndIssue(ctx context.Context, t *testing.T, ctl *controller, project *v1pb.Project, db *v1pb.Database, title string) (*v1pb.Plan, *v1pb.Issue) {
	t.Helper()
	a := require.New(t)

	sheet, err := ctl.sheetServiceClient.CreateSheet(ctx, connect.NewRequest(&v1pb.CreateSheetRequest{
		Parent: project.Name,
		Sheet:  &v1pb.Sheet{Content: []byte("SELECT 1;")},
	}))
	a.NoError(err)

	plan, err := ctl.planServiceClient.CreatePlan(ctx, connect.NewRequest(&v1pb.CreatePlanRequest{
		Parent: project.Name,
		Plan: &v1pb.Plan{
			Specs: []*v1pb.Plan_Spec{{
				Id: uuid.NewString(),
				Config: &v1pb.Plan_Spec_ChangeDatabaseConfig{
					ChangeDatabaseConfig: &v1pb.Plan_ChangeDatabaseConfig{
						Targets: []string{db.Name},
						Sheet:   sheet.Msg.Name,
					},
				},
			}},
		},
	}))
	a.NoError(err)

	issue, err := ctl.issueServiceClient.CreateIssue(ctx, connect.NewRequest(&v1pb.CreateIssueRequest{
		Parent: project.Name,
		Issue: &v1pb.Issue{
			Title: title,
			Type:  v1pb.Issue_DATABASE_CHANGE,
			Plan:  plan.Msg.Name,
		},
	}))
	a.NoError(err)

	return plan.Msg, issue.Msg
}

// createPlanIssueRollout wraps createPlanAndIssue and also creates the rollout.
func createPlanIssueRollout(ctx context.Context, t *testing.T, ctl *controller, project *v1pb.Project, db *v1pb.Database, title string) (*v1pb.Plan, *v1pb.Issue, *v1pb.Rollout) {
	t.Helper()
	a := require.New(t)
	plan, issue := createPlanAndIssue(ctx, t, ctl, project, db, title)
	rollout, err := ctl.rolloutServiceClient.CreateRollout(ctx, connect.NewRequest(&v1pb.CreateRolloutRequest{
		Parent: plan.Name,
	}))
	a.NoError(err)
	return plan, issue, rollout.Msg
}

// projectSnapshot captures the state of a project's composite-PK rows as
// observed through the public gRPC API. Each slice's rows are keyed by the
// UID parsed from their resource name.
type projectSnapshot struct {
	Plans         []*v1pb.Plan
	Issues        []*v1pb.Issue
	TaskRuns      []*v1pb.TaskRun
	PlanCheckRuns []*v1pb.PlanCheckRun
	IssueComments []*v1pb.IssueComment
}

// snapshotProject captures every plan/issue/task_run/plan_check_run row
// visible to the public gRPC API for the given project. Every read goes
// through the service layer (auth + audit), not the raw store — so a
// read-path regression that leaks cross-project data would also surface in
// these calls.
//
// PlanCheckRuns are fetched via GetPlanCheckRun(planName+"/planCheckRun"),
// which is the single-consolidated-PCR endpoint the UI itself uses. One
// PCR per plan.
func snapshotProject(
	ctx context.Context,
	t *testing.T,
	ctl *controller,
	project *v1pb.Project,
) *projectSnapshot {
	t.Helper()
	a := require.New(t)

	plansResp, err := ctl.planServiceClient.ListPlans(ctx, connect.NewRequest(&v1pb.ListPlansRequest{
		Parent:   project.Name,
		PageSize: 1000,
	}))
	a.NoError(err, "ListPlans(%s)", project.Name)
	plans := plansResp.Msg.Plans
	for _, p := range plans {
		a.True(strings.HasPrefix(p.Name, project.Name+"/"),
			"ListPlans(%s) returned %q — read-path leak", project.Name, p.Name)
	}

	issuesResp, err := ctl.issueServiceClient.ListIssues(ctx, connect.NewRequest(&v1pb.ListIssuesRequest{
		Parent:   project.Name,
		PageSize: 1000,
	}))
	a.NoError(err, "ListIssues(%s)", project.Name)
	issues := issuesResp.Msg.Issues
	for _, i := range issues {
		a.True(strings.HasPrefix(i.Name, project.Name+"/"),
			"ListIssues(%s) returned %q — read-path leak", project.Name, i.Name)
	}

	var taskRuns []*v1pb.TaskRun
	var planCheckRuns []*v1pb.PlanCheckRun
	for _, p := range plans {
		trResp, err := ctl.rolloutServiceClient.ListTaskRuns(ctx, connect.NewRequest(&v1pb.ListTaskRunsRequest{
			Parent: p.Name + "/rollout/stages/-/tasks/-",
		}))
		a.NoError(err, "ListTaskRuns for plan %s", p.Name)
		for _, tr := range trResp.Msg.TaskRuns {
			a.True(strings.HasPrefix(tr.Name, project.Name+"/"),
				"ListTaskRuns for plan %s returned %q — read-path leak", p.Name, tr.Name)
		}
		taskRuns = append(taskRuns, trResp.Msg.TaskRuns...)

		pcrResp, err := ctl.planServiceClient.GetPlanCheckRun(ctx, connect.NewRequest(&v1pb.GetPlanCheckRunRequest{
			Name: p.Name + "/planCheckRun",
		}))
		// A plan may not have a check run yet; only NotFound is acceptable here.
		// Other codes (Internal, Permission, Unavailable) likely indicate a
		// real regression in the read path and should fail the snapshot.
		switch {
		case err == nil:
			a.True(strings.HasPrefix(pcrResp.Msg.Name, project.Name+"/"),
				"GetPlanCheckRun for plan %s returned %q — read-path leak", p.Name, pcrResp.Msg.Name)
			planCheckRuns = append(planCheckRuns, pcrResp.Msg)
		case connect.CodeOf(err) == connect.CodeNotFound:
			// expected: plan has no PCR yet
		default:
			a.NoError(err, "GetPlanCheckRun for plan %s", p.Name)
		}
	}

	var issueComments []*v1pb.IssueComment
	for _, i := range issues {
		icResp, err := ctl.issueServiceClient.ListIssueComments(ctx, connect.NewRequest(&v1pb.ListIssueCommentsRequest{
			Parent:   i.Name,
			PageSize: 1000,
		}))
		a.NoError(err, "ListIssueComments for issue %s", i.Name)
		for _, c := range icResp.Msg.IssueComments {
			a.True(strings.HasPrefix(c.Name, project.Name+"/"),
				"ListIssueComments for issue %s returned %q — read-path leak", i.Name, c.Name)
		}
		issueComments = append(issueComments, icResp.Msg.IssueComments...)
	}

	return &projectSnapshot{
		Plans:         plans,
		Issues:        issues,
		TaskRuns:      taskRuns,
		PlanCheckRuns: planCheckRuns,
		IssueComments: issueComments,
	}
}

// assertProjectUnchanged fails if any row in the `before` snapshot went
// missing or got mutated in `after`. Keyed by resource name (unique per
// project by construction) so comparison is order-insensitive.
//
// Issue.UpdateTime is deliberately NOT compared — the approval runner
// actively updates open issues in the background, which is expected
// behavior, not a cross-project leak. Title and Status are the fields a
// cross-project corruption would mutate.
func assertProjectUnchanged(
	t *testing.T,
	before, after *projectSnapshot,
	label string,
) {
	t.Helper()
	a := require.New(t)

	assertNoChange(t, before.Plans, after.Plans,
		func(p *v1pb.Plan) string { return p.Name },
		func(b, af *v1pb.Plan) {
			a.True(proto.Equal(b.UpdateTime, af.UpdateTime), "%s: plan %s update_time changed", label, b.Name)
		},
		label, "plan")

	assertNoChange(t, before.Issues, after.Issues,
		func(i *v1pb.Issue) string { return i.Name },
		func(b, af *v1pb.Issue) {
			a.Equal(b.Title, af.Title, "%s: issue %s title changed", label, b.Name)
			a.Equal(b.Status, af.Status, "%s: issue %s status changed", label, b.Name)
		},
		label, "issue")

	assertNoChange(t, before.TaskRuns, after.TaskRuns,
		func(tr *v1pb.TaskRun) string { return tr.Name },
		func(b, af *v1pb.TaskRun) {
			a.Equal(b.Status, af.Status, "%s: task_run %s status changed from %v to %v", label, b.Name, b.Status, af.Status)
			a.True(proto.Equal(b.UpdateTime, af.UpdateTime), "%s: task_run %s update_time changed", label, b.Name)
		},
		label, "task_run")

	assertNoChange(t, before.PlanCheckRuns, after.PlanCheckRuns,
		func(p *v1pb.PlanCheckRun) string { return p.Name },
		func(b, af *v1pb.PlanCheckRun) {
			a.Equal(b.Status, af.Status, "%s: plan_check_run %s status changed from %v to %v", label, b.Name, b.Status, af.Status)
		},
		label, "plan_check_run")

	// issue_comment is keyed by composite (project, id); a cross-project
	// emission would surface here as either a new row in `after` or a
	// disappeared row.
	assertNoChange(t, before.IssueComments, after.IssueComments,
		func(c *v1pb.IssueComment) string { return c.Name },
		func(b, af *v1pb.IssueComment) {
			a.Equal(b.Comment, af.Comment, "%s: issue_comment %s comment changed", label, b.Name)
		},
		label, "issue_comment")
}

// assertNoChange compares two row slices keyed by name, fails if any
// row disappeared, appeared, or duplicated. Per-row equality beyond
// identity is delegated to the caller's `compare` func.
func assertNoChange[T any](
	t *testing.T,
	before, after []T,
	key func(T) string,
	compare func(b, af T),
	label, kind string,
) {
	t.Helper()
	a := require.New(t)
	byKey := make(map[string]T, len(before))
	for _, r := range before {
		_, dup := byKey[key(r)]
		a.False(dup, "%s: duplicate %s %q in before snapshot", label, kind, key(r))
		byKey[key(r)] = r
	}
	a.Equal(len(before), len(after), "%s: %s count changed", label, kind)
	seen := make(map[string]bool, len(after))
	for _, r := range after {
		a.False(seen[key(r)], "%s: duplicate %s %q in after snapshot", label, kind, key(r))
		seen[key(r)] = true
		b, ok := byKey[key(r)]
		a.True(ok, "%s: %s %q appeared after", label, kind, key(r))
		if ok {
			compare(b, r)
		}
	}
}
