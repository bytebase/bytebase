package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	"github.com/bytebase/bytebase/backend/store"
)

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

	skipped := tasksOf("project-a")[101].Payload
	require.True(t, skipped.GetSkipped())
	require.Equal(t, "fixed in git", skipped.GetSkippedReason())
	require.Equal(t, "alex@example.com", skipped.GetSkipper())
	require.NotNil(t, skipped.GetSkipTime())
	require.WithinDuration(t, before, skipped.GetSkipTime().AsTime(), time.Minute)

	sibling := tasksOf("project-a")[102].Payload
	require.False(t, sibling.GetSkipped())
	require.Empty(t, sibling.GetSkipper())
	require.Nil(t, sibling.GetSkipTime())

	for id, task := range tasksOf("project-b") {
		require.False(t, task.Payload.GetSkipped(), "project-b task %d", id)
		require.Empty(t, task.Payload.GetSkipper(), "project-b task %d", id)
		require.Nil(t, task.Payload.GetSkipTime(), "project-b task %d", id)
	}
}
