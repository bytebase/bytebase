package tests

import (
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
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
	projectAID, err := common.GetProjectID(projectA.Msg.Name)
	a.NoError(err)
	baselineA := snapshotProject(ctx, t, ctl.server.StoreForTest(), projectAID)

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
	projectAID, err := common.GetProjectID(projectA.Msg.Name)
	a.NoError(err)
	baselineA := snapshotProject(ctx, t, ctl.server.StoreForTest(), projectAID)

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
// Covers plan and issue (both created in fixture). task/task_run/plan_check_run
// are only created in project A at fixture time (B's rollout is deferred to
// individual tests), so those collisions are verified by assertTaskRunsCollide
// which tests may call after running project B's rollout.
func assertFixtureIDsCollide(ctx context.Context, t *testing.T, ctl *controller, f *collisionFixture) {
	t.Helper()
	a := require.New(t)
	s := ctl.server.StoreForTest()

	projectAID, err := common.GetProjectID(f.ProjectA.Name)
	a.NoError(err)
	projectBID, err := common.GetProjectID(f.ProjectB.Name)
	a.NoError(err)

	aPlans, err := s.ListPlans(ctx, &store.FindPlanMessage{ProjectID: projectAID})
	a.NoError(err)
	bPlans, err := s.ListPlans(ctx, &store.FindPlanMessage{ProjectID: projectBID})
	a.NoError(err)
	a.Greater(len(aPlans), 0, "project A should have at least one plan")
	a.Greater(len(bPlans), 0, "project B should have at least one plan")
	assertAtLeastOneUIDCollides(t, planUIDs(aPlans), planUIDs(bPlans), "plan")

	aIssues, err := s.ListIssues(ctx, &store.FindIssueMessage{ProjectIDs: []string{projectAID}})
	a.NoError(err)
	bIssues, err := s.ListIssues(ctx, &store.FindIssueMessage{ProjectIDs: []string{projectBID}})
	a.NoError(err)
	a.Greater(len(aIssues), 0, "project A should have at least one issue")
	a.Greater(len(bIssues), 0, "project B should have at least one issue")
	assertAtLeastOneUIDCollides(t, issueUIDs(aIssues), issueUIDs(bIssues), "issue")
}

// completeRolloutB drives project B's rollout to completion and proves the
// composite-PK ids collide for every table whose rows are created by the
// rollout (task, task_run, plan_check_run). This is the ONLY supported way
// for a collision test to roll out B — every call site gets the full
// collision invariant for free, so no test can silently become vacuous by
// forgetting one of the three assertCollide helpers.
//
// Why it lives here rather than inside setupCollidingProjects: the fixture
// returns before B's rollout fires, so task/task_run/plan_check_run don't
// exist yet at fixture-construction time. Tests that don't need B's rollout
// (pure read-isolation tests on plan/issue) don't pay the cost.
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
	assertPlanCheckRunsCollide(ctx, t, ctl, f)
}

// assertTaskRunsCollide verifies that after both projects have rolled out,
// their task_run ids collide. Call this from tests that run project B's
// rollout and depend on task_run collisions.
//
// Note: nextProjectID allocates per table, so task_run collision does not
// imply plan_check_run collision. Use assertPlanCheckRunsCollide for that.
func assertTaskRunsCollide(ctx context.Context, t *testing.T, ctl *controller, f *collisionFixture) {
	t.Helper()
	a := require.New(t)
	s := ctl.server.StoreForTest()

	projectAID, err := common.GetProjectID(f.ProjectA.Name)
	a.NoError(err)
	projectBID, err := common.GetProjectID(f.ProjectB.Name)
	a.NoError(err)

	aTaskRuns, err := s.ListTaskRuns(ctx, &store.FindTaskRunMessage{ProjectID: projectAID})
	a.NoError(err)
	bTaskRuns, err := s.ListTaskRuns(ctx, &store.FindTaskRunMessage{ProjectID: projectBID})
	a.NoError(err)
	a.Greater(len(aTaskRuns), 0, "project A should have at least one task_run")
	a.Greater(len(bTaskRuns), 0, "project B should have at least one task_run — did you forget to roll out B?")
	assertAtLeastOneUIDCollides(t, taskRunIDs(aTaskRuns), taskRunIDs(bTaskRuns), "task_run")
}

// assertPlanCheckRunsCollide verifies that after both projects have plan
// check runs, their ids collide. Because nextProjectID allocates per-table,
// task_run and plan_check_run sequences can diverge independently — each
// table that a test depends on needs its own collision assertion.
func assertPlanCheckRunsCollide(ctx context.Context, t *testing.T, ctl *controller, f *collisionFixture) {
	t.Helper()
	a := require.New(t)
	s := ctl.server.StoreForTest()

	projectAID, err := common.GetProjectID(f.ProjectA.Name)
	a.NoError(err)
	projectBID, err := common.GetProjectID(f.ProjectB.Name)
	a.NoError(err)

	aPCRs, err := s.ListPlanCheckRuns(ctx, &store.FindPlanCheckRunMessage{ProjectID: projectAID})
	a.NoError(err)
	bPCRs, err := s.ListPlanCheckRuns(ctx, &store.FindPlanCheckRunMessage{ProjectID: projectBID})
	a.NoError(err)
	a.Greater(len(aPCRs), 0, "project A should have at least one plan_check_run")
	a.Greater(len(bPCRs), 0, "project B should have at least one plan_check_run — did you forget to roll out B?")
	assertAtLeastOneUIDCollides(t, planCheckRunUIDs(aPCRs), planCheckRunUIDs(bPCRs), "plan_check_run")
}

// assertTasksCollide verifies that after both projects have rolled out,
// their task ids collide. Needed separately from task_runs because of
// per-table allocation.
func assertTasksCollide(ctx context.Context, t *testing.T, ctl *controller, f *collisionFixture) {
	t.Helper()
	a := require.New(t)
	s := ctl.server.StoreForTest()

	projectAID, err := common.GetProjectID(f.ProjectA.Name)
	a.NoError(err)
	projectBID, err := common.GetProjectID(f.ProjectB.Name)
	a.NoError(err)

	aTasks, err := s.ListTasks(ctx, &store.TaskFind{ProjectID: projectAID})
	a.NoError(err)
	bTasks, err := s.ListTasks(ctx, &store.TaskFind{ProjectID: projectBID})
	a.NoError(err)
	a.Greater(len(aTasks), 0, "project A should have at least one task")
	a.Greater(len(bTasks), 0, "project B should have at least one task — did you forget to roll out B?")
	assertAtLeastOneUIDCollides(t, taskIDs(aTasks), taskIDs(bTasks), "task")
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

func planUIDs(plans []*store.PlanMessage) []int64 {
	out := make([]int64, 0, len(plans))
	for _, p := range plans {
		out = append(out, p.UID)
	}
	return out
}

func issueUIDs(issues []*store.IssueMessage) []int64 {
	out := make([]int64, 0, len(issues))
	for _, i := range issues {
		out = append(out, i.UID)
	}
	return out
}

func taskRunIDs(trs []*store.TaskRunMessage) []int64 {
	out := make([]int64, 0, len(trs))
	for _, tr := range trs {
		out = append(out, tr.ID)
	}
	return out
}

func planCheckRunUIDs(pcrs []*store.PlanCheckRunMessage) []int64 {
	out := make([]int64, 0, len(pcrs))
	for _, pcr := range pcrs {
		out = append(out, pcr.UID)
	}
	return out
}

func taskIDs(tasks []*store.TaskMessage) []int64 {
	out := make([]int64, 0, len(tasks))
	for _, tk := range tasks {
		out = append(out, tk.ID)
	}
	return out
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

// projectSnapshot captures the state of composite-PK rows for a project.
type projectSnapshot struct {
	Plans         []*store.PlanMessage
	Issues        []*store.IssueMessage
	Tasks         []*store.TaskMessage
	TaskRuns      []*store.TaskRunMessage
	PlanCheckRuns []*store.PlanCheckRunMessage
}

// snapshotProject queries the store for composite-PK rows belonging to a project.
func snapshotProject(
	ctx context.Context,
	t *testing.T,
	s *store.Store,
	projectID string,
) *projectSnapshot {
	t.Helper()
	a := require.New(t)

	plans, err := s.ListPlans(ctx, &store.FindPlanMessage{
		ProjectID: projectID,
	})
	a.NoError(err, "ListPlans for project %s", projectID)
	for _, p := range plans {
		a.Equal(projectID, p.ProjectID,
			"ListPlans(%s) returned a row belonging to project %s — read-path leak", projectID, p.ProjectID)
	}

	issues, err := s.ListIssues(ctx, &store.FindIssueMessage{
		ProjectIDs: []string{projectID},
	})
	a.NoError(err, "ListIssues for project %s", projectID)
	for _, i := range issues {
		a.Equal(projectID, i.ProjectID,
			"ListIssues(%s) returned a row belonging to project %s — read-path leak", projectID, i.ProjectID)
	}

	tasks, err := s.ListTasks(ctx, &store.TaskFind{
		ProjectID: projectID,
	})
	a.NoError(err, "ListTasks for project %s", projectID)
	for _, tk := range tasks {
		a.Equal(projectID, tk.ProjectID,
			"ListTasks(%s) returned a row belonging to project %s — read-path leak", projectID, tk.ProjectID)
	}

	taskRuns, err := s.ListTaskRuns(ctx, &store.FindTaskRunMessage{
		ProjectID: projectID,
	})
	a.NoError(err, "ListTaskRuns for project %s", projectID)
	for _, tr := range taskRuns {
		a.Equal(projectID, tr.ProjectID,
			"ListTaskRuns(%s) returned a row belonging to project %s — read-path leak", projectID, tr.ProjectID)
	}

	planCheckRuns, err := s.ListPlanCheckRuns(ctx, &store.FindPlanCheckRunMessage{
		ProjectID: projectID,
	})
	a.NoError(err, "ListPlanCheckRuns for project %s", projectID)
	for _, pcr := range planCheckRuns {
		a.Equal(projectID, pcr.ProjectID,
			"ListPlanCheckRuns(%s) returned a row belonging to project %s — read-path leak", projectID, pcr.ProjectID)
	}

	return &projectSnapshot{
		Plans:         plans,
		Issues:        issues,
		Tasks:         tasks,
		TaskRuns:      taskRuns,
		PlanCheckRuns: planCheckRuns,
	}
}

// assertProjectUnchanged compares two snapshots and fails if any row was
// added, removed, or modified. Rows are keyed by primary-key identifier so
// the comparison is insensitive to result ordering. Duplicate keys in a
// single snapshot (which would indicate a JOIN regression) are caught
// explicitly rather than silently collapsed by the map.
func assertProjectUnchanged(
	t *testing.T,
	before, after *projectSnapshot,
	label string,
) {
	t.Helper()
	a := require.New(t)

	beforePlans := make(map[int64]*store.PlanMessage, len(before.Plans))
	for _, p := range before.Plans {
		_, dup := beforePlans[p.UID]
		a.False(dup, "%s: duplicate plan UID %d in before snapshot", label, p.UID)
		beforePlans[p.UID] = p
	}
	a.Equal(len(before.Plans), len(after.Plans),
		"%s: plan count changed", label)
	seenPlans := make(map[int64]bool, len(after.Plans))
	for _, p := range after.Plans {
		a.False(seenPlans[p.UID], "%s: duplicate plan UID %d in after snapshot", label, p.UID)
		seenPlans[p.UID] = true
		b, ok := beforePlans[p.UID]
		a.True(ok, "%s: plan %d appeared after", label, p.UID)
		if ok {
			a.Equal(b.UpdatedAt, p.UpdatedAt,
				"%s: plan %d updated_at changed", label, p.UID)
		}
	}

	// Note: Issue.UpdatedAt is NOT compared — the approval runner actively
	// updates open issues' updated_at timestamps in the background, which
	// is expected behavior, not a cross-project leak. Title and Status are
	// the fields a cross-project corruption would mutate.
	beforeIssues := make(map[int64]*store.IssueMessage, len(before.Issues))
	for _, i := range before.Issues {
		_, dup := beforeIssues[i.UID]
		a.False(dup, "%s: duplicate issue UID %d in before snapshot", label, i.UID)
		beforeIssues[i.UID] = i
	}
	a.Equal(len(before.Issues), len(after.Issues),
		"%s: issue count changed", label)
	seenIssues := make(map[int64]bool, len(after.Issues))
	for _, i := range after.Issues {
		a.False(seenIssues[i.UID], "%s: duplicate issue UID %d in after snapshot", label, i.UID)
		seenIssues[i.UID] = true
		b, ok := beforeIssues[i.UID]
		a.True(ok, "%s: issue %d appeared after", label, i.UID)
		if ok {
			a.Equal(b.Title, i.Title, "%s: issue %d title changed", label, i.UID)
			a.Equal(b.Status, i.Status, "%s: issue %d status changed", label, i.UID)
		}
	}

	beforeTasks := make(map[int64]*store.TaskMessage, len(before.Tasks))
	for _, tk := range before.Tasks {
		_, dup := beforeTasks[tk.ID]
		a.False(dup, "%s: duplicate task ID %d in before snapshot", label, tk.ID)
		beforeTasks[tk.ID] = tk
	}
	a.Equal(len(before.Tasks), len(after.Tasks),
		"%s: task count changed", label)
	seenTasks := make(map[int64]bool, len(after.Tasks))
	for _, tk := range after.Tasks {
		a.False(seenTasks[tk.ID], "%s: duplicate task ID %d in after snapshot", label, tk.ID)
		seenTasks[tk.ID] = true
		b, ok := beforeTasks[tk.ID]
		a.True(ok, "%s: task %d appeared after", label, tk.ID)
		if ok {
			a.Equal(b.UpdatedAt, tk.UpdatedAt,
				"%s: task %d updated_at changed", label, tk.ID)
		}
	}

	beforeTaskRuns := make(map[int64]*store.TaskRunMessage, len(before.TaskRuns))
	for _, tr := range before.TaskRuns {
		_, dup := beforeTaskRuns[tr.ID]
		a.False(dup, "%s: duplicate task_run ID %d in before snapshot", label, tr.ID)
		beforeTaskRuns[tr.ID] = tr
	}
	a.Equal(len(before.TaskRuns), len(after.TaskRuns),
		"%s: task_run count changed", label)
	seenTaskRuns := make(map[int64]bool, len(after.TaskRuns))
	for _, tr := range after.TaskRuns {
		a.False(seenTaskRuns[tr.ID], "%s: duplicate task_run ID %d in after snapshot", label, tr.ID)
		seenTaskRuns[tr.ID] = true
		b, ok := beforeTaskRuns[tr.ID]
		a.True(ok, "%s: task_run %d appeared after", label, tr.ID)
		if ok {
			a.Equal(b.Status, tr.Status,
				"%s: task_run %d status changed from %v to %v", label, tr.ID, b.Status, tr.Status)
			a.Equal(b.UpdatedAt, tr.UpdatedAt,
				"%s: task_run %d updated_at changed", label, tr.ID)
		}
	}

	beforePlanCheckRuns := make(map[int64]*store.PlanCheckRunMessage, len(before.PlanCheckRuns))
	for _, pcr := range before.PlanCheckRuns {
		_, dup := beforePlanCheckRuns[pcr.UID]
		a.False(dup, "%s: duplicate plan_check_run UID %d in before snapshot", label, pcr.UID)
		beforePlanCheckRuns[pcr.UID] = pcr
	}
	a.Equal(len(before.PlanCheckRuns), len(after.PlanCheckRuns),
		"%s: plan_check_run count changed", label)
	seenPlanCheckRuns := make(map[int64]bool, len(after.PlanCheckRuns))
	for _, pcr := range after.PlanCheckRuns {
		a.False(seenPlanCheckRuns[pcr.UID], "%s: duplicate plan_check_run UID %d in after snapshot", label, pcr.UID)
		seenPlanCheckRuns[pcr.UID] = true
		b, ok := beforePlanCheckRuns[pcr.UID]
		a.True(ok, "%s: plan_check_run %d appeared after", label, pcr.UID)
		if ok {
			a.Equal(b.Status, pcr.Status,
				"%s: plan_check_run %d status changed from %v to %v", label, pcr.UID, b.Status, pcr.Status)
			a.Equal(b.UpdatedAt, pcr.UpdatedAt,
				"%s: plan_check_run %d updated_at changed", label, pcr.UID)
		}
	}
}

// mustGetProjectID extracts the resource ID from a project name, failing the test on error.
func mustGetProjectID(t *testing.T, name string) string {
	t.Helper()
	id, err := common.GetProjectID(name)
	require.NoError(t, err, "failed to extract project ID from %q", name)
	return id
}

// mustGetInstanceID extracts the resource ID from an instance name, failing the test on error.
func mustGetInstanceID(t *testing.T, name string) string {
	t.Helper()
	id, err := common.GetInstanceID(name)
	require.NoError(t, err, "failed to extract instance ID from %q", name)
	return id
}
