package v1

import (
	"context"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/testcontainer"
	"github.com/bytebase/bytebase/backend/component/review"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	_ "github.com/bytebase/bytebase/backend/plugin/db/pg"
	"github.com/bytebase/bytebase/backend/store"
)

// TestClassifyRolloutError pins the two answers a rollout caller acts on. The
// races that produce them — an approval reset while the rollout was being
// built, a draft that was never submitted — are pinned against the database
// in component/review.
func TestClassifyRolloutError(t *testing.T) {
	t.Parallel()
	workflowErr := func(reason review.ErrorReason) error {
		return &review.Error{Code: review.ErrorFailedPrecondition, Reason: reason, Err: errors.New("refused")}
	}
	require.ErrorIs(t, classifyRolloutError(workflowErr(review.ReasonDraftIssue)), errDraftIssueNotSubmitted)
	require.True(t, IsStaleRolloutApprovalError(classifyRolloutError(workflowErr(review.ReasonApprovalRequired))))
	require.True(t, IsStaleRolloutApprovalError(classifyRolloutError(workflowErr(review.ReasonStaleInput))))
	require.True(t, IsStaleRolloutApprovalError(classifyRolloutError(errors.Wrap(errStaleRolloutApproval, "building tasks"))))

	other := errors.New("failed to get pipeline create")
	require.Equal(t, other, classifyRolloutError(other), "anything else passes through as it came")
}

// Two projects whose plan and task ids collide, each with two tasks on one
// stage, so a skip that ignored the project or the task id would show.
const skipFixtureSQL = `
	INSERT INTO workspace (resource_id) VALUES ('default');
	INSERT INTO project (resource_id, workspace, name) VALUES ('project-a', 'default', 'Project A');
	INSERT INTO project (resource_id, workspace, name) VALUES ('project-b', 'default', 'Project B');
	INSERT INTO instance (resource_id, workspace) VALUES ('instance-a', 'default');
	INSERT INTO plan (id, creator, project, name, description) VALUES
		(101, 'creator@example.com', 'project-a', 'Plan A', ''),
		(101, 'creator@example.com', 'project-b', 'Plan B', '');
	INSERT INTO task (id, project, plan_id, instance, environment, db_name, type) VALUES
		(101, 'project-a', 101, 'instance-a', 'test', 'db1', 'DATABASE_MIGRATE'),
		(102, 'project-a', 101, 'instance-a', 'test', 'db2', 'DATABASE_MIGRATE'),
		(101, 'project-b', 101, 'instance-a', 'test', 'db1', 'DATABASE_MIGRATE'),
		(102, 'project-b', 101, 'instance-a', 'test', 'db2', 'DATABASE_MIGRATE');
`

func TestBatchSkipTasksRecordsSkipperAndTime(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db, s, _ := testcontainer.NewMetadataDB(t)
	_, err := db.ExecContext(ctx, skipFixtureSQL)
	require.NoError(t, err)

	before := time.Now()
	require.NoError(t, s.BatchSkipTasks(ctx, "project-a", []int64{101}, "fixed in git", "alex@example.com"))

	tasksOf := func(projectID string) map[int64]*store.TaskMessage {
		planID := int64(101)
		tasks, err := s.ListTasks(ctx, &store.TaskFind{ProjectID: projectID, PlanID: &planID})
		require.NoError(t, err)
		require.Len(t, tasks, 2)
		byID := map[int64]*store.TaskMessage{}
		for _, task := range tasks {
			byID[task.ID] = task
		}
		return byID
	}
	projectA := &store.ProjectMessage{ResourceID: "project-a"}

	skipped, err := convertToTask(projectA, tasksOf("project-a")[101])
	require.NoError(t, err)
	require.Equal(t, "users/alex@example.com", skipped.Skipper)
	require.Equal(t, "fixed in git", skipped.SkippedReason)
	require.NotNil(t, skipped.SkipTime)
	require.WithinDuration(t, before, skipped.SkipTime.AsTime(), time.Minute)

	sibling, err := convertToTask(projectA, tasksOf("project-a")[102])
	require.NoError(t, err)
	require.Empty(t, sibling.Skipper)
	require.Nil(t, sibling.SkipTime)

	for id, task := range tasksOf("project-b") {
		require.False(t, task.Payload.GetSkipped(), "project-b task %d", id)
		require.Empty(t, task.Payload.GetSkipper(), "project-b task %d", id)
		require.Nil(t, task.Payload.GetSkipTime(), "project-b task %d", id)
	}
}

// db1 has 001 and 002 live and 003 deleted; its sibling db2 on the same
// instance has 003 live; db3 is new apart from a declarative revision of 001.
const releaseStateFixtureSQL = `
	INSERT INTO workspace (resource_id) VALUES ('default');
	INSERT INTO project (resource_id, workspace, name) VALUES ('project-a', 'default', 'Project A');
	INSERT INTO instance (resource_id, workspace, environment, metadata)
		VALUES ('instance-a', 'default', 'test', '{"engine":"POSTGRES"}');
	INSERT INTO db (instance, name, project, environment) VALUES
		('instance-a', 'db1', 'project-a', 'test'),
		('instance-a', 'db2', 'project-a', 'test'),
		('instance-a', 'db3', 'project-a', 'test');
	INSERT INTO revision (instance, db_name, version, payload, deleter, deleted_at) VALUES
		('instance-a', 'db1', '001', '{"type":"VERSIONED"}', NULL, NULL),
		('instance-a', 'db1', '002', '{"type":"VERSIONED"}', NULL, NULL),
		('instance-a', 'db1', '003', '{"type":"VERSIONED"}', 'old@example.com', '2026-10-01T08:00:00Z'),
		('instance-a', 'db1', '003', '{"type":"VERSIONED"}', 'alex@example.com', '2026-10-05T09:30:00Z'),
		('instance-a', 'db2', '003', '{"type":"VERSIONED"}', NULL, NULL),
		('instance-a', 'db3', '001', '{"type":"DECLARATIVE"}', NULL, NULL);
	INSERT INTO release (project, release_id, creator, payload) VALUES
		('project-a', 'r1', 'ci@example.com',
		 '{"type":"VERSIONED","files":[{"path":"001_init.sql","version":"001"},{"path":"002_orders.sql","version":"002"},{"path":"003_index.sql","version":"003"}]}');
`

func TestRolloutCreationStoresReleaseState(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db, s, _ := testcontainer.NewMetadataDB(t)
	_, err := db.ExecContext(ctx, releaseStateFixtureSQL)
	require.NoError(t, err)
	ctx = context.WithValue(ctx, common.WorkspaceIDContextKey, "default")

	before := time.Now()
	tasks, err := getTaskCreatesFromChangeDatabaseConfig(ctx, s,
		&storepb.PlanConfig_Spec{Id: "spec-1"},
		&storepb.PlanConfig_ChangeDatabaseConfig{
			Targets: []string{
				"instances/instance-a/databases/db1",
				"instances/instance-a/databases/db2",
				"instances/instance-a/databases/db3",
			},
			Release: "projects/project-a/releases/r1",
		}, "project-a")
	require.NoError(t, err)
	require.Len(t, tasks, 3)

	stateOf := map[string]*storepb.Task_ReleaseState{}
	for _, task := range tasks {
		stateOf[*task.DatabaseName] = task.Payload.GetReleaseState()
	}
	pendingOf := func(state *storepb.Task_ReleaseState) []string {
		var versions []string
		for _, v := range state.GetPendingVersions() {
			versions = append(versions, v.Version)
		}
		return versions
	}

	// Two of three applied; the target's own deleted revision does not count,
	// and the sibling's live 003 does not either. The newest deletion is kept.
	db1 := stateOf["db1"]
	require.NotNil(t, db1)
	require.Equal(t, []string{"003"}, pendingOf(db1))
	require.Equal(t, int32(2), db1.AppliedCount)
	require.Equal(t, "alex@example.com", db1.PendingVersions[0].RevisionDeleter)
	require.Equal(t, time.Date(2026, 10, 5, 9, 30, 0, 0, time.UTC), db1.PendingVersions[0].RevisionDeleteTime.AsTime())
	require.WithinDuration(t, before, db1.ComputeTime.AsTime(), time.Minute)

	var db1Task *store.TaskMessage
	for _, task := range tasks {
		if *task.DatabaseName == "db1" {
			db1Task = task
		}
	}
	db1Task.ProjectID, db1Task.PlanID, db1Task.ID = "project-a", 101, 101
	v1Task, err := convertToTask(&store.ProjectMessage{ResourceID: "project-a"}, db1Task)
	require.NoError(t, err)
	v1State := v1Task.GetDatabaseUpdate().GetReleaseState()
	require.NotNil(t, v1State)
	require.Equal(t, int32(2), v1State.AppliedCount)
	require.Len(t, v1State.PendingVersions, 1)
	require.Equal(t, "003", v1State.PendingVersions[0].Version)
	require.Equal(t, "users/alex@example.com", v1State.PendingVersions[0].RevisionDeleter)
	require.True(t, v1State.PendingVersions[0].RevisionDeleteTime.AsTime().Equal(db1.PendingVersions[0].RevisionDeleteTime.AsTime()))
	require.True(t, v1State.ComputeTime.AsTime().Equal(db1.ComputeTime.AsTime()))

	// The sibling's state is its own.
	db2 := stateOf["db2"]
	require.Equal(t, []string{"001", "002"}, pendingOf(db2))
	require.Equal(t, int32(1), db2.AppliedCount)
	require.Empty(t, db2.PendingVersions[0].RevisionDeleter)
	require.Nil(t, db2.PendingVersions[0].RevisionDeleteTime)

	// A new database applies everything; a declarative revision is not a version.
	db3 := stateOf["db3"]
	require.Equal(t, []string{"001", "002", "003"}, pendingOf(db3))
	require.Equal(t, int32(0), db3.AppliedCount)
}
