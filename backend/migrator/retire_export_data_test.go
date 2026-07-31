package migrator

import (
	"context"
	"io/fs"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
)

// TestRetireExportDataMigration seeds legacy DATABASE_EXPORT rows and verifies
// the 3.22 purge deletes the whole export family — including the draft
// DATABASE_CHANGE issue attached to an export plan, which a type-only delete
// would leave behind to break the plan delete on the issue→plan FK — while
// unrelated plans, issues, tasks, and comments survive.
//
// Project p2 seeds a change workflow whose plan/issue/task/task_run ids all
// collide with p1's export family: every purge predicate must carry the
// project column of the composite PKs, so p2's rows survive untouched.
func TestRetireExportDataMigration(t *testing.T) {
	ctx := context.Background()
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(ctx) })

	db := container.GetDB()
	require.NoError(t, MigrateSchema(ctx, db))

	// Fresh installs no longer have export_archive; recreate the pre-3.22 table
	// so the DROP TABLE in the migration runs against a realistic schema.
	_, err := db.ExecContext(ctx, `
		CREATE TABLE export_archive (
			resource_id text PRIMARY KEY DEFAULT gen_random_uuid()::text,
			workspace text NOT NULL REFERENCES workspace(resource_id),
			created_at timestamptz NOT NULL DEFAULT now(),
			bytes bytea,
			payload jsonb NOT NULL DEFAULT '{}'
		);

		INSERT INTO workspace (resource_id) VALUES ('default');
		INSERT INTO principal (name, email, password_hash) VALUES ('creator', 'creator@example.com', 'unused');
		INSERT INTO project (resource_id, workspace, name) VALUES ('p1', 'default', 'Project 1');
		INSERT INTO instance (resource_id, workspace) VALUES ('i1', 'default');

		-- Family A: classic completed export workflow.
		INSERT INTO plan (id, creator, project, name, description, config) VALUES
			(100, 'creator@example.com', 'p1', 'export plan', '',
			 '{"specs":[{"id":"s1","exportDataConfig":{"targets":["instances/i1/databases/db1"]}}]}');
		INSERT INTO issue (id, creator, project, plan_id, name, status, type) VALUES
			(200, 'creator@example.com', 'p1', 100, 'export issue', 'DONE', 'DATABASE_EXPORT');
		INSERT INTO task (id, project, plan_id, instance, db_name, type) VALUES
			(300, 'p1', 100, 'i1', 'db1', 'DATABASE_EXPORT');
		INSERT INTO task_run (id, creator, project, task_id, attempt, status, result) VALUES
			(400, 'creator@example.com', 'p1', 300, 0, 'DONE', '{"exportArchiveId":"arch-1"}');
		INSERT INTO task_run_log (project, task_run_id) VALUES ('p1', 400);
		INSERT INTO plan_check_run (id, project, plan_id, status) VALUES (500, 'p1', 100, 'DONE');
		INSERT INTO issue_comment (creator, project, issue_id) VALUES ('creator@example.com', 'p1', 200);
		INSERT INTO plan_webhook_delivery (project, plan_id, event_type) VALUES ('p1', 100, 'PIPELINE_COMPLETED');
		INSERT INTO export_archive (resource_id, workspace, bytes) VALUES ('arch-1', 'default', ''::bytea);

		-- Family B: export plan carrying a draft DATABASE_CHANGE issue (the FK trap).
		INSERT INTO plan (id, creator, project, name, description, config) VALUES
			(101, 'creator@example.com', 'p1', 'export plan with draft', '',
			 '{"specs":[{"id":"s1","exportDataConfig":{}}]}');
		INSERT INTO issue (id, creator, project, plan_id, name, status, type, payload) VALUES
			(201, 'creator@example.com', 'p1', 101, 'draft on export plan', 'OPEN', 'DATABASE_CHANGE', '{"draft":true}');
		INSERT INTO issue_comment (creator, project, issue_id) VALUES ('creator@example.com', 'p1', 201);

		-- Family D: legacy mixed plan (change + export specs). The whole family
		-- is purged via the plan-membership arm, including its DATABASE_MIGRATE
		-- task and DATABASE_CHANGE issue.
		INSERT INTO plan (id, creator, project, name, description, config) VALUES
			(103, 'creator@example.com', 'p1', 'mixed plan', '',
			 '{"specs":[{"id":"s1","changeDatabaseConfig":{}},{"id":"s2","exportDataConfig":{}}]}');
		INSERT INTO issue (id, creator, project, plan_id, name, status, type) VALUES
			(205, 'creator@example.com', 'p1', 103, 'mixed issue', 'DONE', 'DATABASE_CHANGE');
		INSERT INTO task (id, project, plan_id, instance, db_name, type) VALUES
			(303, 'p1', 103, 'i1', 'db1', 'DATABASE_MIGRATE');
		INSERT INTO task_run (id, creator, project, task_id, attempt, status) VALUES
			(403, 'creator@example.com', 'p1', 303, 0, 'DONE');
		INSERT INTO task_run_log (project, task_run_id) VALUES ('p1', 403);

		-- An export-typed issue with no plan at all.
		INSERT INTO issue (id, creator, project, plan_id, name, status, type) VALUES
			(204, 'creator@example.com', 'p1', NULL, 'orphan export issue', 'CANCELED', 'DATABASE_EXPORT');

		-- Family C: unrelated change workflow that must survive.
		INSERT INTO plan (id, creator, project, name, description, config) VALUES
			(102, 'creator@example.com', 'p1', 'change plan', '',
			 '{"specs":[{"id":"s1","changeDatabaseConfig":{"targets":["instances/i1/databases/db1"]}}]}');
		INSERT INTO issue (id, creator, project, plan_id, name, status, type) VALUES
			(202, 'creator@example.com', 'p1', 102, 'change issue', 'DONE', 'DATABASE_CHANGE');
		INSERT INTO task (id, project, plan_id, instance, db_name, type) VALUES
			(302, 'p1', 102, 'i1', 'db1', 'DATABASE_MIGRATE');
		INSERT INTO task_run (id, creator, project, task_id, attempt, status) VALUES
			(402, 'creator@example.com', 'p1', 302, 0, 'DONE');
		INSERT INTO task_run_log (project, task_run_id) VALUES ('p1', 402);
		INSERT INTO plan_check_run (id, project, plan_id, status) VALUES (502, 'p1', 102, 'DONE');
		INSERT INTO issue_comment (creator, project, issue_id) VALUES ('creator@example.com', 'p1', 202);

		-- A plan-less role grant issue that must survive.
		INSERT INTO issue (id, creator, project, plan_id, name, status, type) VALUES
			(203, 'creator@example.com', 'p1', NULL, 'role grant', 'OPEN', 'ROLE_GRANT');

		-- Project p2: a change workflow whose ids collide with p1's export
		-- family. Survives in full — proves composite-PK project scoping.
		INSERT INTO project (resource_id, workspace, name) VALUES ('p2', 'default', 'Project 2');
		INSERT INTO plan (id, creator, project, name, description, config) VALUES
			(100, 'creator@example.com', 'p2', 'p2 change plan', '',
			 '{"specs":[{"id":"s1","changeDatabaseConfig":{"targets":["instances/i1/databases/db2"]}}]}');
		INSERT INTO issue (id, creator, project, plan_id, name, status, type) VALUES
			(200, 'creator@example.com', 'p2', 100, 'p2 change issue', 'DONE', 'DATABASE_CHANGE');
		INSERT INTO task (id, project, plan_id, instance, db_name, type) VALUES
			(300, 'p2', 100, 'i1', 'db2', 'DATABASE_MIGRATE');
		INSERT INTO task_run (id, creator, project, task_id, attempt, status) VALUES
			(400, 'creator@example.com', 'p2', 300, 0, 'DONE');
		INSERT INTO task_run_log (project, task_run_id) VALUES ('p2', 400);
		INSERT INTO plan_check_run (id, project, plan_id, status) VALUES (500, 'p2', 100, 'DONE');
		INSERT INTO issue_comment (creator, project, issue_id) VALUES ('creator@example.com', 'p2', 200);
		INSERT INTO plan_webhook_delivery (project, plan_id, event_type) VALUES ('p2', 100, 'PIPELINE_COMPLETED');
	`)
	require.NoError(t, err)

	// Positive preconditions: the seeds must exist before the purge, so the
	// survival assertions below cannot pass vacuously.
	var planCount, issueCount int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM plan`).Scan(&planCount))
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM issue`).Scan(&issueCount))
	require.Equal(t, 5, planCount)
	require.Equal(t, 7, issueCount)

	buf, err := fs.ReadFile(migrationFS, "migration/3.22/0000##retire_export_data.sql")
	require.NoError(t, err)

	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer conn.Close()
	require.NoError(t, executeMigration(ctx, conn, string(buf), "test-retire-export-data"))

	queryIDs := func(query string) []int64 {
		rows, err := db.QueryContext(ctx, query)
		require.NoError(t, err)
		defer rows.Close()
		var ids []int64
		for rows.Next() {
			var id int64
			require.NoError(t, rows.Scan(&id))
			ids = append(ids, id)
		}
		require.NoError(t, rows.Err())
		return ids
	}

	require.Equal(t, []int64{102}, queryIDs(`SELECT id FROM plan WHERE project = 'p1' ORDER BY id`))
	require.Equal(t, []int64{202, 203}, queryIDs(`SELECT id FROM issue WHERE project = 'p1' ORDER BY id`))
	require.Equal(t, []int64{302}, queryIDs(`SELECT id FROM task WHERE project = 'p1' ORDER BY id`))
	require.Equal(t, []int64{402}, queryIDs(`SELECT id FROM task_run WHERE project = 'p1' ORDER BY id`))
	require.Equal(t, []int64{402}, queryIDs(`SELECT task_run_id FROM task_run_log WHERE project = 'p1' ORDER BY task_run_id`))
	require.Equal(t, []int64{502}, queryIDs(`SELECT id FROM plan_check_run WHERE project = 'p1' ORDER BY id`))
	require.Equal(t, []int64{202}, queryIDs(`SELECT issue_id FROM issue_comment WHERE project = 'p1' ORDER BY issue_id`))
	require.Empty(t, queryIDs(`SELECT plan_id FROM plan_webhook_delivery WHERE project = 'p1'`))

	// p2's colliding-id change workflow survives in full.
	require.Equal(t, []int64{100}, queryIDs(`SELECT id FROM plan WHERE project = 'p2' ORDER BY id`))
	require.Equal(t, []int64{200}, queryIDs(`SELECT id FROM issue WHERE project = 'p2' ORDER BY id`))
	require.Equal(t, []int64{300}, queryIDs(`SELECT id FROM task WHERE project = 'p2' ORDER BY id`))
	require.Equal(t, []int64{400}, queryIDs(`SELECT id FROM task_run WHERE project = 'p2' ORDER BY id`))
	require.Equal(t, []int64{400}, queryIDs(`SELECT task_run_id FROM task_run_log WHERE project = 'p2' ORDER BY task_run_id`))
	require.Equal(t, []int64{500}, queryIDs(`SELECT id FROM plan_check_run WHERE project = 'p2' ORDER BY id`))
	require.Equal(t, []int64{200}, queryIDs(`SELECT issue_id FROM issue_comment WHERE project = 'p2' ORDER BY issue_id`))
	require.Equal(t, []int64{100}, queryIDs(`SELECT plan_id FROM plan_webhook_delivery WHERE project = 'p2'`))

	var exportArchiveExists bool
	require.NoError(t, db.QueryRowContext(ctx,
		`SELECT to_regclass('export_archive') IS NOT NULL`).Scan(&exportArchiveExists))
	require.False(t, exportArchiveExists)
}
