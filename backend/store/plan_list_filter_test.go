package store_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	"github.com/bytebase/bytebase/backend/store"
)

// TestListPlansSourceAndTaskStatusFilters runs the has_release, release, and
// task_status filters against PostgreSQL. Both projects hold a plan 101 so a
// filter that drops the project from its task join would match across them.
func TestListPlansSourceAndTaskStatusFilters(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db, s, _ := testcontainer.NewMetadataDB(t)
	_, err := db.ExecContext(ctx, `
		INSERT INTO workspace (resource_id) VALUES ('default');
		INSERT INTO project (resource_id, workspace, name) VALUES
			('project-a', 'default', 'Project A'),
			('project-b', 'default', 'Project B');
		INSERT INTO instance (resource_id, workspace) VALUES ('instance-a', 'default');
		INSERT INTO plan (id, creator, project, name, description, config, deleted) VALUES
			(101, 'creator@example.com', 'project-a', 'gitops failed', '', '{"hasRollout": true, "specs": [{"id": "s1", "changeDatabaseConfig": {"release": "projects/project-a/releases/r1"}}]}', FALSE),
			(102, 'creator@example.com', 'project-a', 'gitops done', '', '{"hasRollout": true, "specs": [{"id": "s1", "changeDatabaseConfig": {"release": "projects/project-a/releases/r2"}}]}', FALSE),
			(103, 'creator@example.com', 'project-a', 'gitops closed', '', '{"specs": [{"id": "s1", "changeDatabaseConfig": {"release": "projects/project-a/releases/r3"}}]}', TRUE),
			(104, 'creator@example.com', 'project-a', 'clickops waiting', '', '{"hasRollout": true, "specs": [{"id": "s1", "changeDatabaseConfig": {"sheet": "projects/project-a/sheets/1"}}]}', FALSE),
			(105, 'creator@example.com', 'project-a', 'clickops no rollout', '', '{"specs": [{"id": "s1", "changeDatabaseConfig": {"sheet": "projects/project-a/sheets/2", "release": ""}}]}', FALSE),
			(101, 'creator@example.com', 'project-b', 'project b running', '', '{"hasRollout": true, "specs": [{"id": "s1", "changeDatabaseConfig": {"release": "projects/project-b/releases/r1"}}]}', FALSE);
		INSERT INTO task (id, project, plan_id, instance, environment, type, payload) VALUES
			(201, 'project-a', 101, 'instance-a', 'test', 'DATABASE_MIGRATE', '{}'),
			(202, 'project-a', 101, 'instance-a', 'prod', 'DATABASE_MIGRATE', '{}'),
			(203, 'project-a', 102, 'instance-a', 'test', 'DATABASE_MIGRATE', '{}'),
			(204, 'project-a', 102, 'instance-a', 'prod', 'DATABASE_MIGRATE', '{"skipped": true}'),
			(205, 'project-a', 104, 'instance-a', 'test', 'DATABASE_MIGRATE', '{}'),
			(201, 'project-b', 101, 'instance-a', 'test', 'DATABASE_MIGRATE', '{}');
		INSERT INTO task_run (id, project, task_id, attempt, status) VALUES
			(301, 'project-a', 201, 0, 'DONE'),
			(302, 'project-a', 202, 0, 'RUNNING'),
			(303, 'project-a', 202, 1, 'FAILED'),
			(304, 'project-a', 203, 0, 'DONE'),
			(305, 'project-a', 205, 0, 'AVAILABLE'),
			(301, 'project-b', 201, 0, 'RUNNING');
	`)
	require.NoError(t, err)

	list := func(projectID, filter string) []string {
		t.Helper()
		filterQ, err := store.GetListPlanFilter(filter)
		require.NoError(t, err)
		plans, err := s.ListPlans(ctx, &store.FindPlanMessage{Workspace: "default", ProjectID: projectID, FilterQ: filterQ})
		require.NoError(t, err)
		var got []string
		for _, plan := range plans {
			got = append(got, plan.Name)
		}
		return got
	}

	require.ElementsMatch(t, []string{"gitops failed", "gitops done", "gitops closed"}, list("project-a", `has_release == true`))
	require.ElementsMatch(t, []string{"clickops waiting", "clickops no rollout"}, list("project-a", `has_release == false`))

	require.Equal(t, []string{"gitops done"}, list("project-a", `release == "projects/project-a/releases/r2"`))
	require.Empty(t, list("project-a", `release == "projects/project-b/releases/r1"`))

	// The latest run decides a task's status, so the earlier RUNNING run of task 202 does not match.
	require.Equal(t, []string{"gitops failed"}, list("project-a", `task_status in ["FAILED"]`))
	require.Empty(t, list("project-a", `task_status in ["RUNNING"]`))
	require.Equal(t, []string{"clickops waiting"}, list("project-a", `task_status in ["PENDING"]`))
	require.Equal(t, []string{"gitops done"}, list("project-a", `task_status in ["SKIPPED"]`))
	require.ElementsMatch(t, []string{"gitops failed", "gitops done"}, list("project-a", `has_release == true && task_status in ["DONE"]`))

	// Project B's plan 101 is running; project A's plan 101 is not.
	require.Equal(t, []string{"project b running"}, list("project-b", `task_status in ["RUNNING"]`))
	require.Empty(t, list("project-b", `task_status in ["FAILED"]`))

	// The stage summaries read the same task statuses the filter matches.
	counts, err := s.ListTaskStatusCountByPlanIDs(ctx, []string{"project-a"}, []int64{101, 102, 104})
	require.NoError(t, err)
	var got []string
	for _, c := range counts {
		got = append(got, fmt.Sprintf("%d/%s/%s/%d", c.PlanID, c.Environment, c.Status, c.Count))
	}
	require.ElementsMatch(t, []string{
		"101/test/DONE/1", "101/prod/FAILED/1",
		"102/test/DONE/1", "102/prod/SKIPPED/1",
		"104/test/PENDING/1",
	}, got)
}
