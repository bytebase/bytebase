package store_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	"github.com/bytebase/bytebase/backend/migrator"
	"github.com/bytebase/bytebase/backend/store"
)

// TestListTaskRunsPagination pins the offset pagination behind ListTaskRuns:
// pages are read in (project, id) order, so a project's task runs arrive
// exactly once across pages even when another project's ids collide, and the
// cross-project runner list keeps the same total order.
func TestListTaskRunsPagination(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(context.Background()) })
	db := container.GetDB()
	require.NoError(t, migrator.MigrateSchema(ctx, db))

	_, err := db.ExecContext(ctx, `
		INSERT INTO workspace (resource_id) VALUES ('default');
		INSERT INTO project (resource_id, workspace, name) VALUES
			('project-a', 'default', 'Project A'),
			('project-b', 'default', 'Project B');
		INSERT INTO instance (resource_id, workspace) VALUES ('instance-a', 'default');
		INSERT INTO plan (id, creator, project, name, description) VALUES
			(101, 'creator@example.com', 'project-a', 'Plan A', ''),
			(101, 'creator@example.com', 'project-b', 'Plan B', '');
		INSERT INTO task (id, project, plan_id, instance, environment, type) VALUES
			(101, 'project-a', 101, 'instance-a', 'prod', 'DATABASE_SCHEMA_UPDATE'),
			(101, 'project-b', 101, 'instance-a', 'prod', 'DATABASE_SCHEMA_UPDATE');
		INSERT INTO task_run (id, project, task_id, attempt, status) VALUES
			(101, 'project-a', 101, 0, 'DONE'),
			(102, 'project-a', 101, 1, 'DONE'),
			(103, 'project-a', 101, 2, 'DONE'),
			(101, 'project-b', 101, 0, 'DONE'),
			(102, 'project-b', 101, 1, 'DONE');
	`)
	require.NoError(t, err)

	pgURL := fmt.Sprintf(
		"host=%s port=%s user=postgres password=root-password database=postgres",
		container.GetHost(), container.GetPort(),
	)
	s, err := store.New(ctx, pgURL, false)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, s.Close()) })

	planUID := int64(101)
	ids := func(taskRuns []*store.TaskRunMessage) []string {
		var got []string
		for _, taskRun := range taskRuns {
			got = append(got, fmt.Sprintf("%s/%d", taskRun.ProjectID, taskRun.ID))
		}
		return got
	}

	// Two pages of two cover project A's three runs exactly once.
	limit, offset := 2, 0
	page, err := s.ListTaskRuns(ctx, &store.FindTaskRunMessage{ProjectID: "project-a", PlanUID: &planUID, Limit: &limit, Offset: &offset})
	require.NoError(t, err)
	require.Equal(t, []string{"project-a/101", "project-a/102"}, ids(page))

	offset = 2
	page, err = s.ListTaskRuns(ctx, &store.FindTaskRunMessage{ProjectID: "project-a", PlanUID: &planUID, Limit: &limit, Offset: &offset})
	require.NoError(t, err)
	require.Equal(t, []string{"project-a/103"}, ids(page))

	// The unpaginated cross-project list is ordered by (project, id): id alone
	// is not unique across projects.
	all, err := s.ListTaskRuns(ctx, &store.FindTaskRunMessage{})
	require.NoError(t, err)
	require.Equal(t, []string{"project-a/101", "project-a/102", "project-a/103", "project-b/101", "project-b/102"}, ids(all))
}
