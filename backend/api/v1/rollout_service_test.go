package v1

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

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

func TestConvertToTaskReturnsSkipperAndTime(t *testing.T) {
	skipTime := time.Date(2026, 10, 5, 7, 14, 0, 0, time.UTC)
	databaseName := "db1"
	task := &store.TaskMessage{
		ProjectID: "project-a", PlanID: 101, ID: 101, Environment: "test",
		InstanceID: "instance-a", DatabaseName: &databaseName, Type: storepb.Task_DATABASE_MIGRATE,
		Payload: &storepb.Task{
			Skipped: true, SkippedReason: "fixed in git",
			Skipper: "alex@example.com", SkipTime: timestamppb.New(skipTime),
		},
	}
	v1Task, err := convertToTask(&store.ProjectMessage{ResourceID: "project-a"}, task)
	require.NoError(t, err)
	require.Equal(t, "users/alex@example.com", v1Task.Skipper)
	require.True(t, v1Task.SkipTime.AsTime().Equal(skipTime))

	// A task skipped before the skipper was recorded has neither.
	task.Payload.Skipper, task.Payload.SkipTime = "", nil
	v1Task, err = convertToTask(&store.ProjectMessage{ResourceID: "project-a"}, task)
	require.NoError(t, err)
	require.Empty(t, v1Task.Skipper)
	require.Nil(t, v1Task.SkipTime)
}

type countingRevisionLister struct {
	finds []*store.FindRevisionMessage
}

func (c *countingRevisionLister) ListRevisions(_ context.Context, find *store.FindRevisionMessage) ([]*store.RevisionMessage, error) {
	c.finds = append(c.finds, find)
	return nil, nil
}

func TestReleaseStatesReadOneRevisionQueryPerInstance(t *testing.T) {
	release := &store.ReleaseMessage{Payload: &storepb.ReleasePayload{
		Type:  storepb.SchemaChangeType_VERSIONED,
		Files: []*storepb.ReleasePayload_File{{Version: "001"}, {Version: "002"}},
	}}
	databases := []*store.DatabaseMessage{
		{InstanceID: "instance-a", DatabaseName: "db1"},
		{InstanceID: "instance-a", DatabaseName: "db2"},
		{InstanceID: "instance-b", DatabaseName: "db1"},
		{InstanceID: "instance-a", DatabaseName: "db3"},
	}
	lister := &countingRevisionLister{}
	states, err := computeReleaseStates(context.Background(), lister, release, databases)
	require.NoError(t, err)

	namesOf := map[string][]string{}
	for _, find := range lister.finds {
		require.NotNil(t, find.DatabaseNames)
		namesOf[find.InstanceID] = *find.DatabaseNames
	}
	require.Len(t, lister.finds, 2)
	require.Equal(t, map[string][]string{
		"instance-a": {"db1", "db2", "db3"},
		"instance-b": {"db1"},
	}, namesOf)
	// With no revisions, every target applies every version: one range.
	require.Len(t, states, 4)
	for _, database := range databases {
		state := states[releaseStateKey(database)]
		require.Len(t, state.GetPendingRanges(), 1)
		require.Equal(t, int32(0), state.PendingRanges[0].Start)
		require.Equal(t, int32(2), state.PendingRanges[0].End)
	}
}

func TestReleaseStateStoresPendingFilesAsRanges(t *testing.T) {
	deletedAt := time.Date(2026, 10, 5, 9, 30, 0, 0, time.UTC)
	deleter := "alex@example.com"
	versions := []string{"001", "002", "003", "004", "005", "006"}
	revisions := []*store.RevisionMessage{
		{Version: "002"},
		{Version: "003"},
		{Version: "005", DeletedAt: &deletedAt, Deleter: &deleter},
	}
	state := releaseStateOf(versions, revisions, timestamppb.Now())

	type span struct{ start, end int32 }
	var spans []span
	for _, r := range state.GetPendingRanges() {
		spans = append(spans, span{r.Start, r.End})
	}
	require.Equal(t, []span{{0, 1}, {3, 6}}, spans)
	require.Equal(t, int32(2), state.AppliedCount)
	require.Len(t, state.DeletedRevisions, 1)
	require.Equal(t, "005", state.DeletedRevisions[0].Version)
	require.Equal(t, deleter, state.DeletedRevisions[0].Deleter)
	require.True(t, state.DeletedRevisions[0].DeleteTime.AsTime().Equal(deletedAt))

	// A large release on a new database is still one range.
	many := make([]string, 500)
	for i := range many {
		many[i] = fmt.Sprintf("%04d", i)
	}
	fresh := releaseStateOf(many, nil, timestamppb.Now())
	require.Len(t, fresh.PendingRanges, 1)
	require.Equal(t, int32(500), fresh.PendingRanges[0].End)
}

// db1 has 001 and 002 live and 003 deleted; its sibling db2 on the same
// instance has 003 live; db3 is new apart from a declarative revision of 001.
// instance-b also has a db1, with 003 live, read in the same batch.
const releaseStateFixtureSQL = `
	INSERT INTO workspace (resource_id) VALUES ('default');
	INSERT INTO project (resource_id, workspace, name) VALUES ('project-a', 'default', 'Project A');
	INSERT INTO instance (resource_id, workspace, environment, metadata)
		VALUES ('instance-a', 'default', 'test', '{"engine":"POSTGRES"}'),
		       ('instance-b', 'default', 'test', '{"engine":"POSTGRES"}');
	INSERT INTO db (instance, name, project, environment) VALUES
		('instance-a', 'db1', 'project-a', 'test'),
		('instance-a', 'db2', 'project-a', 'test'),
		('instance-a', 'db3', 'project-a', 'test'),
		('instance-b', 'db1', 'project-a', 'test');
	INSERT INTO revision (instance, db_name, version, payload, deleter, deleted_at) VALUES
		('instance-a', 'db1', '001', '{"type":"VERSIONED"}', NULL, NULL),
		('instance-a', 'db1', '002', '{"type":"VERSIONED"}', NULL, NULL),
		('instance-a', 'db1', '003', '{"type":"VERSIONED"}', 'old@example.com', '2026-10-01T08:00:00Z'),
		('instance-a', 'db1', '003', '{"type":"VERSIONED"}', 'alex@example.com', '2026-10-05T09:30:00Z'),
		('instance-a', 'db2', '003', '{"type":"VERSIONED"}', NULL, NULL),
		('instance-a', 'db3', '001', '{"type":"DECLARATIVE"}', NULL, NULL),
		('instance-b', 'db1', '003', '{"type":"VERSIONED"}', NULL, NULL);
	INSERT INTO release (project, iteration, release_id, creator, payload) VALUES
		('project-a', 1, 'r1', 'ci@example.com',
		 '{"type":"VERSIONED","files":[{"path":"001_init.sql","version":"001"},{"path":"002_orders.sql","version":"002"},{"path":"003_index.sql","version":"003"}]}'),
		('project-a', 2, 'r2', 'ci@example.com',
		 '{"type":"DECLARATIVE","files":[{"path":"schema.sql","version":"001"}]}');
	INSERT INTO release (project, iteration, release_id, creator, payload, deleted) VALUES
		('project-a', 3, 'r3', 'ci@example.com',
		 '{"type":"VERSIONED","files":[{"path":"001_init.sql","version":"001"}]}', TRUE);
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
				"instances/instance-b/databases/db1",
			},
			Release: "projects/project-a/releases/r1",
		}, "project-a")
	require.NoError(t, err)
	require.Len(t, tasks, 4)

	stateOf := map[string]*storepb.Task_ReleaseState{}
	for _, task := range tasks {
		if task.InstanceID == "instance-a" {
			stateOf[*task.DatabaseName] = task.Payload.GetReleaseState()
		}
	}
	files := []string{"001", "002", "003"}
	pendingOf := func(state *storepb.Task_ReleaseState) []string {
		var versions []string
		for _, r := range state.GetPendingRanges() {
			versions = append(versions, files[r.Start:r.End]...)
		}
		return versions
	}

	// Two of three applied; the target's own deleted revision does not count,
	// and the sibling's live 003 does not either. The newest deletion is kept.
	db1 := stateOf["db1"]
	require.NotNil(t, db1)
	require.Equal(t, []string{"003"}, pendingOf(db1))
	require.Equal(t, int32(2), db1.AppliedCount)
	require.Len(t, db1.DeletedRevisions, 1)
	require.Equal(t, "003", db1.DeletedRevisions[0].Version)
	require.Equal(t, "alex@example.com", db1.DeletedRevisions[0].Deleter)
	require.Equal(t, time.Date(2026, 10, 5, 9, 30, 0, 0, time.UTC), db1.DeletedRevisions[0].DeleteTime.AsTime())
	require.WithinDuration(t, before, db1.ComputeTime.AsTime(), time.Minute)

	var db1Task *store.TaskMessage
	for _, task := range tasks {
		if task.InstanceID == "instance-a" && *task.DatabaseName == "db1" {
			db1Task = task
		}
	}
	db1Task.ProjectID, db1Task.PlanID, db1Task.ID = "project-a", 101, 101
	v1Task, err := convertToTask(&store.ProjectMessage{ResourceID: "project-a"}, db1Task)
	require.NoError(t, err)
	v1State := v1Task.GetDatabaseUpdate().GetReleaseState()
	require.NotNil(t, v1State)
	require.Equal(t, int32(2), v1State.AppliedCount)
	require.Equal(t, int32(1), v1State.PendingCount)
	require.Len(t, v1State.PendingRanges, 1)
	require.Equal(t, int32(2), v1State.PendingRanges[0].Start)
	require.Equal(t, int32(3), v1State.PendingRanges[0].End)
	require.Len(t, v1State.DeletedRevisions, 1)
	require.Equal(t, "003", v1State.DeletedRevisions[0].Version)
	require.Equal(t, "users/alex@example.com", v1State.DeletedRevisions[0].Deleter)
	require.True(t, v1State.DeletedRevisions[0].DeleteTime.AsTime().Equal(db1.DeletedRevisions[0].DeleteTime.AsTime()))
	require.True(t, v1State.ComputeTime.AsTime().Equal(db1.ComputeTime.AsTime()))

	// The sibling's state is its own.
	db2 := stateOf["db2"]
	require.Equal(t, []string{"001", "002"}, pendingOf(db2))
	require.Equal(t, int32(1), db2.AppliedCount)
	require.Empty(t, db2.DeletedRevisions)

	// A new database applies everything; a declarative revision is not a version.
	db3 := stateOf["db3"]
	require.Equal(t, []string{"001", "002", "003"}, pendingOf(db3))
	require.Equal(t, int32(0), db3.AppliedCount)

	// The other instance's db1 shares the name and the batch, not the state.
	var otherDB1 *storepb.Task_ReleaseState
	for _, task := range tasks {
		if task.InstanceID == "instance-b" && *task.DatabaseName == "db1" {
			otherDB1 = task.Payload.GetReleaseState()
		}
	}
	require.Equal(t, []string{"001", "002"}, pendingOf(otherDB1))
	require.Equal(t, int32(1), otherDB1.AppliedCount)

	// A declarative release computes its diff when it runs: no stored state.
	declarative, err := getTaskCreatesFromChangeDatabaseConfig(ctx, s,
		&storepb.PlanConfig_Spec{Id: "spec-2"},
		&storepb.PlanConfig_ChangeDatabaseConfig{
			Targets: []string{"instances/instance-a/databases/db3"},
			Release: "projects/project-a/releases/r2",
		}, "project-a")
	require.NoError(t, err)
	require.Len(t, declarative, 1)
	require.Nil(t, declarative[0].Payload.GetReleaseState())
	declarative[0].ProjectID, declarative[0].PlanID, declarative[0].ID = "project-a", 102, 103
	v1Declarative, err := convertToTask(&store.ProjectMessage{ResourceID: "project-a"}, declarative[0])
	require.NoError(t, err)
	require.Nil(t, v1Declarative.GetDatabaseUpdate().GetReleaseState())

	// A deleted release still gets its task, with no stored state, so the run
	// reports the missing release where users see it.
	missing, err := getTaskCreatesFromChangeDatabaseConfig(ctx, s,
		&storepb.PlanConfig_Spec{Id: "spec-3"},
		&storepb.PlanConfig_ChangeDatabaseConfig{
			Targets: []string{"instances/instance-a/databases/db3"},
			Release: "projects/project-a/releases/r3",
		}, "project-a")
	require.NoError(t, err)
	require.Len(t, missing, 1)
	require.Nil(t, missing[0].Payload.GetReleaseState())
}
