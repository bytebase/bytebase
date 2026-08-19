package migrator

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/blang/semver/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	_ "github.com/bytebase/bytebase/backend/plugin/db/pg"
	"github.com/bytebase/bytebase/backend/store"
)

func TestLatestVersion(t *testing.T) {
	files, err := getSortedVersionedFiles()
	require.NoError(t, err)
	require.Equal(t, semver.MustParse("3.22.10"), *files[len(files)-1].version)
	require.Equal(t, "migration/3.22/0010##saved_query_updated_at_index.sql", files[len(files)-1].path)
}

func TestVersionUnique(t *testing.T) {
	files, err := getSortedVersionedFiles()
	require.NoError(t, err)
	versions := make(map[string]struct{})
	for _, file := range files {
		if file.version == nil {
			continue
		}
		if _, ok := versions[file.version.String()]; ok {
			require.Fail(t, "duplicate version %s", file.version.String())
		}
		versions[file.version.String()] = struct{}{}
	}
}

// TestMigration3_16_2_TaskRunLogDuplicateTimestamps verifies that the 3.16.2
// id-column cleanup succeeds on legacy task_run_log data holding duplicate
// (task_run_id, created_at) pairs, which were legal under the old id primary
// key. Regression test for BYT-10035.
func TestMigration3_16_2_TaskRunLogDuplicateTimestamps(t *testing.T) {
	ctx := context.Background()
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(ctx) })

	db := container.GetDB()

	// Minimal pre-3.16.2 shapes for every table the migration touches.
	setup := `
		CREATE TABLE project (id BIGSERIAL PRIMARY KEY, resource_id TEXT NOT NULL);
		CREATE TABLE instance (id BIGSERIAL PRIMARY KEY, resource_id TEXT NOT NULL);
		CREATE TABLE db (id BIGSERIAL PRIMARY KEY, instance TEXT NOT NULL, name TEXT NOT NULL);
		CREATE TABLE setting (id BIGSERIAL PRIMARY KEY, name TEXT NOT NULL);
		CREATE TABLE policy (id BIGSERIAL PRIMARY KEY, resource_type TEXT NOT NULL, resource TEXT NOT NULL, type TEXT NOT NULL);
		CREATE TABLE idp (id BIGSERIAL PRIMARY KEY, resource_id TEXT NOT NULL);
		CREATE TABLE role (id BIGSERIAL PRIMARY KEY, resource_id TEXT NOT NULL);
		CREATE TABLE db_schema (id BIGSERIAL PRIMARY KEY, instance TEXT NOT NULL, db_name TEXT NOT NULL);
		CREATE TABLE db_group (id BIGSERIAL PRIMARY KEY, project TEXT NOT NULL, resource_id TEXT NOT NULL);
		CREATE TABLE release (id BIGSERIAL PRIMARY KEY, project TEXT NOT NULL, train TEXT NOT NULL, iteration INT NOT NULL);
		CREATE TABLE task_run_log (
			id BIGSERIAL PRIMARY KEY,
			task_run_id INTEGER NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			payload JSONB NOT NULL DEFAULT '{}'
		);
		CREATE INDEX idx_task_run_log_task_run_id ON task_run_log(task_run_id);

		INSERT INTO task_run_log (task_run_id, created_at, payload) VALUES
			(100, '2026-01-01 00:00:00.000001+00', '{"type":"COMMAND_EXECUTE"}'),
			(100, '2026-01-01 00:00:00.000001+00', '{"type":"COMMAND_RESPONSE"}'),
			(100, '2026-01-01 00:00:00.000002+00', '{"type":"COMMAND_EXECUTE"}');
	`
	_, err := db.ExecContext(ctx, setup)
	require.NoError(t, err)

	statement, err := migrationFS.ReadFile("migration/3.16/0002##drop_unused_id_columns.sql")
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, string(statement))
	require.NoError(t, err)

	var rowCount int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM task_run_log`).Scan(&rowCount))
	require.Equal(t, 3, rowCount)

	var hasPkey bool
	require.NoError(t, db.QueryRowContext(ctx, `
		SELECT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'task_run_log_pkey' AND conrelid = 'task_run_log'::regclass)
	`).Scan(&hasPkey))
	require.False(t, hasPkey)

	var hasIDColumn bool
	require.NoError(t, db.QueryRowContext(ctx, `
		SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'task_run_log' AND column_name = 'id')
	`).Scan(&hasIDColumn))
	require.False(t, hasIDColumn)

	var hasIndex bool
	require.NoError(t, db.QueryRowContext(ctx, `
		SELECT EXISTS (SELECT 1 FROM pg_indexes WHERE schemaname = 'public' AND indexname = 'idx_task_run_log_task_run_id_created_at')
	`).Scan(&hasIndex))
	require.True(t, hasIndex)

	var hasOldIndex bool
	require.NoError(t, db.QueryRowContext(ctx, `
		SELECT EXISTS (SELECT 1 FROM pg_indexes WHERE schemaname = 'public' AND indexname = 'idx_task_run_log_task_run_id')
	`).Scan(&hasOldIndex))
	require.False(t, hasOldIndex)
}

// TestMigration3_17_1_TaskRunLogDuplicateTimestamps verifies that the 3.17.1
// project-scoping migration accepts duplicate (task_run_id, created_at) pairs:
// it must backfill project and index the table without imposing uniqueness.
// Regression test for BYT-10035.
func TestMigration3_17_1_TaskRunLogDuplicateTimestamps(t *testing.T) {
	ctx := context.Background()
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(ctx) })

	db := container.GetDB()

	// Minimal post-3.16.2 shapes for the plan-chain tables the migration touches;
	// task_run_log has no id column and no primary key at this point.
	setup := `
		CREATE TABLE project (resource_id TEXT PRIMARY KEY);
		CREATE TABLE plan (id BIGINT PRIMARY KEY, project TEXT NOT NULL);
		CREATE TABLE issue (id BIGINT PRIMARY KEY, project TEXT NOT NULL, plan_id BIGINT);
		CREATE TABLE task (id INT PRIMARY KEY, plan_id BIGINT NOT NULL, environment TEXT NOT NULL);
		CREATE TABLE task_run (id INT PRIMARY KEY, task_id INT NOT NULL, attempt INT NOT NULL);
		CREATE TABLE plan_check_run (id INT PRIMARY KEY, plan_id BIGINT NOT NULL);
		CREATE TABLE plan_webhook_delivery (id INT PRIMARY KEY, plan_id BIGINT NOT NULL);
		CREATE TABLE task_run_log (
			task_run_id INTEGER NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			payload JSONB NOT NULL DEFAULT '{}'
		);
		CREATE INDEX idx_task_run_log_task_run_id_created_at ON task_run_log(task_run_id, created_at);

		INSERT INTO project VALUES ('proj-a');
		INSERT INTO plan VALUES (1, 'proj-a');
		INSERT INTO task VALUES (10, 1, 'environments/prod');
		INSERT INTO task_run VALUES (100, 10, 1);
		INSERT INTO task_run_log (task_run_id, created_at, payload) VALUES
			(100, '2026-01-01 00:00:00.000001+00', '{"type":"COMMAND_EXECUTE"}'),
			(100, '2026-01-01 00:00:00.000001+00', '{"type":"COMMAND_RESPONSE"}'),
			(100, '2026-01-01 00:00:00.000002+00', '{"type":"COMMAND_EXECUTE"}');
	`
	_, err := db.ExecContext(ctx, setup)
	require.NoError(t, err)

	statement, err := migrationFS.ReadFile("migration/3.17/0001##add_project_to_plan_chain.sql")
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, string(statement))
	require.NoError(t, err)

	var backfilledCount int
	require.NoError(t, db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM task_run_log WHERE project = 'proj-a'
	`).Scan(&backfilledCount))
	require.Equal(t, 3, backfilledCount)

	var hasPkey bool
	require.NoError(t, db.QueryRowContext(ctx, `
		SELECT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'task_run_log_pkey' AND conrelid = 'task_run_log'::regclass)
	`).Scan(&hasPkey))
	require.False(t, hasPkey)

	var hasIndex bool
	require.NoError(t, db.QueryRowContext(ctx, `
		SELECT EXISTS (SELECT 1 FROM pg_indexes WHERE schemaname = 'public' AND indexname = 'idx_task_run_log_project_task_run_id_created_at')
	`).Scan(&hasIndex))
	require.True(t, hasIndex)

	var oldIndexCount int
	require.NoError(t, db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM pg_indexes WHERE schemaname = 'public' AND indexname IN ('idx_task_run_log_task_run_id', 'idx_task_run_log_task_run_id_created_at')
	`).Scan(&oldIndexCount))
	require.Zero(t, oldIndexCount)
}

// TestMigration3_22_4_DropTaskRunLogPkey verifies that the pkey drop converges
// every historical shape — the 2-column PK left by upgrading through the
// original 3.16.2, the 3-column PK of fresh installs, and the no-PK state left
// by a release-branch backport of the 3.16.2 fix — is idempotent, and that
// same-microsecond entries insert afterwards instead of colliding.
func TestMigration3_22_4_DropTaskRunLogPkey(t *testing.T) {
	ctx := context.Background()
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(ctx) })

	db := container.GetDB()

	statement, err := migrationFS.ReadFile("migration/3.22/0004##drop_task_run_log_pkey.sql")
	require.NoError(t, err)

	const tableDDL = `
		CREATE TABLE task_run_log (
			project TEXT NOT NULL,
			task_run_id INTEGER NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			payload JSONB NOT NULL DEFAULT '{}'`

	scenarios := []struct {
		name  string
		setup string
	}{
		{
			"2-col pkey from the original 3.16.2",
			tableDDL + `, PRIMARY KEY (task_run_id, created_at));
			CREATE INDEX idx_task_run_log_task_run_id ON task_run_log(task_run_id);`,
		},
		{
			"3-col pkey from a fresh install",
			tableDDL + `, PRIMARY KEY (project, task_run_id, created_at));`,
		},
		{
			"no pkey from a release-branch backport of the 3.16.2 fix",
			tableDDL + `);
			CREATE INDEX idx_task_run_log_task_run_id_created_at ON task_run_log(task_run_id, created_at);`,
		},
	}
	for _, scenario := range scenarios {
		_, err := db.ExecContext(ctx, `DROP TABLE IF EXISTS task_run_log;`+scenario.setup)
		require.NoError(t, err, scenario.name)

		_, err = db.ExecContext(ctx, string(statement))
		require.NoError(t, err, scenario.name)
		// Re-running must be a no-op.
		_, err = db.ExecContext(ctx, string(statement))
		require.NoError(t, err, scenario.name)

		var hasPkey bool
		require.NoError(t, db.QueryRowContext(ctx, `
			SELECT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'task_run_log_pkey' AND conrelid = 'task_run_log'::regclass)
		`).Scan(&hasPkey))
		require.False(t, hasPkey, scenario.name)

		var hasIndex bool
		require.NoError(t, db.QueryRowContext(ctx, `
			SELECT EXISTS (SELECT 1 FROM pg_indexes WHERE schemaname = 'public' AND indexname = 'idx_task_run_log_project_task_run_id_created_at')
		`).Scan(&hasIndex))
		require.True(t, hasIndex, scenario.name)

		var oldIndexCount int
		require.NoError(t, db.QueryRowContext(ctx, `
			SELECT COUNT(*) FROM pg_indexes WHERE schemaname = 'public' AND indexname IN ('idx_task_run_log_task_run_id', 'idx_task_run_log_task_run_id_created_at')
		`).Scan(&oldIndexCount))
		require.Zero(t, oldIndexCount, scenario.name)

		_, err = db.ExecContext(ctx, `
			INSERT INTO task_run_log (project, task_run_id, created_at) VALUES
				('proj-a', 100, '2026-01-01 00:00:00.000001+00'),
				('proj-a', 100, '2026-01-01 00:00:00.000001+00')`)
		require.NoError(t, err, "same-microsecond entries must both insert")
	}
}

func TestMigration3_21_2_MigrateInstanceSyncDatabases(t *testing.T) {
	ctx := context.Background()
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(ctx) })

	db := container.GetDB()

	setup := `
		CREATE TABLE instance (
			resource_id TEXT PRIMARY KEY,
			metadata JSONB NOT NULL DEFAULT '{}'
		);

		INSERT INTO instance (resource_id, metadata) VALUES
			(
				'instance-with-selected-databases',
				'{"engine":"POSTGRES","syncDatabases":["db1","db2"]}'
			),
			(
				'instance-with-empty-databases',
				'{"engine":"POSTGRES","syncDatabases":[]}'
			),
			(
				'instance-with-new-shape',
				'{"engine":"POSTGRES","syncDatabases":{"databases":["db3"]}}'
			),
			(
				'instance-without-sync-databases',
				'{"engine":"POSTGRES"}'
			);
	`
	_, err := db.ExecContext(ctx, setup)
	require.NoError(t, err)

	statement, err := migrationFS.ReadFile("migration/3.21/0002##migrate_instance_sync_databases.sql")
	require.NoError(t, err)

	_, err = db.ExecContext(ctx, string(statement))
	require.NoError(t, err)

	getMetadata := func(resourceID string) string {
		t.Helper()
		var metadata string
		err := db.QueryRowContext(ctx, `SELECT metadata::text FROM instance WHERE resource_id = $1`, resourceID).Scan(&metadata)
		require.NoError(t, err)
		return metadata
	}

	require.JSONEq(t, `{"engine":"POSTGRES","syncDatabases":{"databases":["db1","db2"]}}`, getMetadata("instance-with-selected-databases"))
	require.JSONEq(t, `{"engine":"POSTGRES","syncDatabases":{"databases":[]}}`, getMetadata("instance-with-empty-databases"))
	require.JSONEq(t, `{"engine":"POSTGRES","syncDatabases":{"databases":["db3"]}}`, getMetadata("instance-with-new-shape"))
	require.JSONEq(t, `{"engine":"POSTGRES"}`, getMetadata("instance-without-sync-databases"))
}

func TestMigration3_22_2_AddInstanceProject(t *testing.T) {
	ctx := context.Background()
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(ctx) })

	db := container.GetDB()
	_, err := db.ExecContext(ctx, `
		CREATE TABLE project (
			resource_id TEXT PRIMARY KEY
		);
		CREATE TABLE instance (
			resource_id TEXT PRIMARY KEY,
			metadata JSONB NOT NULL DEFAULT '{}'
		);
		INSERT INTO instance (resource_id) VALUES ('legacy-workspace-instance');
	`)
	require.NoError(t, err)

	statement, err := migrationFS.ReadFile("migration/3.22/0002##add_instance_project.sql")
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, string(statement))
	require.NoError(t, err)

	var projectID *string
	require.NoError(t, db.QueryRowContext(ctx, `
		SELECT project FROM instance WHERE resource_id = 'legacy-workspace-instance'
	`).Scan(&projectID))
	require.Nil(t, projectID)

	var indexDefinition string
	require.NoError(t, db.QueryRowContext(ctx, `
		SELECT indexdef
		FROM pg_indexes
		WHERE schemaname = 'public' AND indexname = 'idx_instance_project'
	`).Scan(&indexDefinition))
	require.Contains(t, indexDefinition, "WHERE (project IS NOT NULL)")
}

func TestMigration3_21_3_MigrateGCPDataSourceFields(t *testing.T) {
	ctx := context.Background()
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(ctx) })

	db := container.GetDB()

	setup := `
		CREATE TABLE instance (
			resource_id TEXT PRIMARY KEY,
			metadata JSONB NOT NULL DEFAULT '{}'
		);

		INSERT INTO instance (resource_id, metadata) VALUES
			(
				'spanner-legacy',
				'{"engine":"SPANNER","dataSources":[{"id":"admin","host":"projects/my-proj/instances/my-inst"}]}'
			),
			(
				'spanner-multi-data-source',
				'{"engine":"SPANNER","dataSources":[{"id":"admin","host":"projects/my-proj/instances/my-inst"},{"id":"ro-1","type":"READ_ONLY","host":"projects/my-proj/instances/my-inst"},{"id":"ro-2","type":"READ_ONLY","host":"projects/my-proj/instances/my-inst"}]}'
			),
			(
				'spanner-new-shape',
				'{"engine":"SPANNER","dataSources":[{"id":"admin","projectId":"my-proj","instanceId":"my-inst","host":"spanner-nonprod.p.googleapis.com"}]}'
			),
			(
				'spanner-no-data-sources',
				'{"engine":"SPANNER","dataSources":[]}'
			),
			(
				'bigquery-legacy',
				'{"engine":"BIGQUERY","dataSources":[{"id":"admin","host":"my-proj"}]}'
			),
			(
				'bigquery-new-shape',
				'{"engine":"BIGQUERY","dataSources":[{"id":"admin","projectId":"my-proj","host":"bigquery-nonprod.p.googleapis.com"}]}'
			),
			(
				'postgres-untouched',
				'{"engine":"POSTGRES","dataSources":[{"id":"admin","host":"localhost","port":"5432"}]}'
			);
	`
	_, err := db.ExecContext(ctx, setup)
	require.NoError(t, err)

	statement, err := migrationFS.ReadFile("migration/3.21/0003##migrate_gcp_data_source_fields.sql")
	require.NoError(t, err)

	_, err = db.ExecContext(ctx, string(statement))
	require.NoError(t, err)

	getMetadata := func(resourceID string) string {
		t.Helper()
		var metadata string
		err := db.QueryRowContext(ctx, `SELECT metadata::text FROM instance WHERE resource_id = $1`, resourceID).Scan(&metadata)
		require.NoError(t, err)
		return metadata
	}

	require.JSONEq(t, `{"engine":"SPANNER","dataSources":[{"id":"admin","projectId":"my-proj","instanceId":"my-inst"}]}`, getMetadata("spanner-legacy"))
	// JSONEq treats arrays as ordered: this also asserts the data source order is preserved.
	require.JSONEq(t, `{"engine":"SPANNER","dataSources":[{"id":"admin","projectId":"my-proj","instanceId":"my-inst"},{"id":"ro-1","type":"READ_ONLY","projectId":"my-proj","instanceId":"my-inst"},{"id":"ro-2","type":"READ_ONLY","projectId":"my-proj","instanceId":"my-inst"}]}`, getMetadata("spanner-multi-data-source"))
	require.JSONEq(t, `{"engine":"SPANNER","dataSources":[{"id":"admin","projectId":"my-proj","instanceId":"my-inst","host":"spanner-nonprod.p.googleapis.com"}]}`, getMetadata("spanner-new-shape"))
	require.JSONEq(t, `{"engine":"SPANNER","dataSources":[]}`, getMetadata("spanner-no-data-sources"))
	require.JSONEq(t, `{"engine":"BIGQUERY","dataSources":[{"id":"admin","projectId":"my-proj"}]}`, getMetadata("bigquery-legacy"))
	require.JSONEq(t, `{"engine":"BIGQUERY","dataSources":[{"id":"admin","projectId":"my-proj","host":"bigquery-nonprod.p.googleapis.com"}]}`, getMetadata("bigquery-new-shape"))
	require.JSONEq(t, `{"engine":"POSTGRES","dataSources":[{"id":"admin","host":"localhost","port":"5432"}]}`, getMetadata("postgres-untouched"))
}

func TestMigration3_21_1_BackfillUIPlanDraftIssues(t *testing.T) {
	ctx := context.Background()
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(ctx) })

	tx, err := container.GetDB().BeginTx(ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback()
	var migrationTime time.Time
	require.NoError(t, tx.QueryRowContext(ctx, `SELECT CURRENT_TIMESTAMP`).Scan(&migrationTime))

	_, err = tx.ExecContext(ctx, `
		CREATE TABLE project (
			resource_id TEXT PRIMARY KEY
		);
		CREATE TABLE plan (
			id BIGINT NOT NULL,
			deleted BOOLEAN NOT NULL DEFAULT FALSE,
			creator TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			project TEXT NOT NULL,
			name TEXT NOT NULL,
			description TEXT NOT NULL,
			config JSONB NOT NULL,
			PRIMARY KEY (project, id)
		);
		CREATE TABLE issue (
			id BIGINT NOT NULL,
			creator TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			project TEXT NOT NULL,
			plan_id BIGINT,
			name TEXT NOT NULL,
			status TEXT NOT NULL,
			type TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			payload JSONB NOT NULL DEFAULT '{}',
			ts_vector TSVECTOR,
			PRIMARY KEY (project, id),
			UNIQUE (project, plan_id)
		);

		INSERT INTO project (resource_id) VALUES ('project-a'), ('project-b');
		INSERT INTO plan (id, creator, created_at, project, name, description, config) VALUES
			(1, 'change@example.com', CURRENT_TIMESTAMP - INTERVAL '29 days', 'project-a', 'recent change', 'change description',
				'{"specs":[{"changeDatabaseConfig":{}},{"changeDatabaseConfig":{}}]}'),
			(2, 'create@example.com', CURRENT_TIMESTAMP - INTERVAL '30 days', 'project-a', 'boundary create', 'create description',
				'{"specs":[{"createDatabaseConfig":{}}]}'),
			(3, 'gitops@example.com', CURRENT_TIMESTAMP - INTERVAL '1 day', 'project-a', 'release GitOps', '',
				'{"specs":[{"changeDatabaseConfig":{"release":"projects/project-a/releases/release-a"}}]}'),
			(4, 'export@example.com', CURRENT_TIMESTAMP - INTERVAL '1 day', 'project-a', 'export', '',
				'{"specs":[{"exportDataConfig":{}}]}'),
			(5, 'mixed@example.com', CURRENT_TIMESTAMP - INTERVAL '1 day', 'project-a', 'mixed', '',
				'{"specs":[{"createDatabaseConfig":{}},{"changeDatabaseConfig":{}}]}'),
			(6, 'deleted@example.com', CURRENT_TIMESTAMP - INTERVAL '1 day', 'project-a', 'deleted', '',
				'{"specs":[{"changeDatabaseConfig":{}}]}'),
			(7, 'old@example.com', CURRENT_TIMESTAMP - INTERVAL '30 days 1 microsecond', 'project-a', 'old', '',
				'{"specs":[{"changeDatabaseConfig":{}}]}'),
			(8, 'linked@example.com', CURRENT_TIMESTAMP - INTERVAL '1 day', 'project-a', 'linked', '',
				'{"specs":[{"changeDatabaseConfig":{}}]}'),
			(9, 'other@example.com', CURRENT_TIMESTAMP - INTERVAL '1 day', 'project-b', 'other project change', 'other description',
				'{"specs":[{"changeDatabaseConfig":{}}]}'),
			(10, 'rollout@example.com', CURRENT_TIMESTAMP - INTERVAL '1 day', 'project-a', 'rolled out', '',
				'{"specs":[{"changeDatabaseConfig":{}}],"hasRollout":true}');
		UPDATE plan SET deleted = TRUE WHERE project = 'project-a' AND id = 6;
		INSERT INTO issue (id, creator, project, plan_id, name, status, type, payload)
		VALUES (150, 'linked@example.com', 'project-a', 8, 'existing issue', 'OPEN', 'DATABASE_CHANGE', '{}');
	`)
	require.NoError(t, err)
	require.NoError(t, tx.Commit())

	conn, err := container.GetDB().Conn(ctx)
	require.NoError(t, err)
	defer conn.Close()
	var migrationPID int
	require.NoError(t, conn.QueryRowContext(ctx, `SELECT pg_backend_pid()`).Scan(&migrationPID))

	createPlanTx, err := container.GetDB().BeginTx(ctx, nil)
	require.NoError(t, err)
	defer createPlanTx.Rollback()
	var lockedProjectID string
	require.NoError(t, createPlanTx.QueryRowContext(ctx, `
		SELECT resource_id FROM project WHERE resource_id = 'project-a' FOR UPDATE`).Scan(&lockedProjectID))
	_, err = createPlanTx.ExecContext(ctx, `
		INSERT INTO plan (id, creator, created_at, project, name, description, config)
		VALUES (11, 'late@example.com', CURRENT_TIMESTAMP, 'project-a', 'late commit', 'late description',
			'{"specs":[{"changeDatabaseConfig":{}}]}')`)
	require.NoError(t, err)

	discoveryDone := make(chan error, 1)
	go func() {
		discoveryDone <- migrate3_21_1At(ctx, conn, migrationTime)
	}()
	require.Eventually(t, func() bool {
		var waiting bool
		err := container.GetDB().QueryRowContext(ctx, `
			SELECT wait_event_type = 'Lock'
			FROM pg_stat_activity
			WHERE pid = $1`, migrationPID).Scan(&waiting)
		return err == nil && waiting
	}, 5*time.Second, 10*time.Millisecond)
	var hasPlanTableLock bool
	require.NoError(t, container.GetDB().QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM pg_locks
			WHERE pid = $1
			  AND locktype = 'relation'
			  AND relation = 'plan'::regclass
			  AND mode = 'ShareLock'
		)`, migrationPID).Scan(&hasPlanTableLock))
	assert.False(t, hasPlanTableLock)
	require.NoError(t, createPlanTx.Commit())
	require.NoError(t, <-discoveryDone)

	rows, err := container.GetDB().QueryContext(ctx, `
		SELECT
			plan.name,
			issue.creator,
			issue.name,
			issue.description,
			issue.status,
			issue.type,
			COALESCE((issue.payload->>'draft')::boolean, FALSE),
			COALESCE(jsonb_array_length(issue.payload->'labels'), 0),
			issue.created_at = $1,
			issue.ts_vector::TEXT
		FROM issue
		JOIN plan ON plan.project = issue.project AND plan.id = issue.plan_id
		WHERE COALESCE((issue.payload->>'draft')::boolean, FALSE)
		ORDER BY plan.project, plan.id
	`, migrationTime)
	require.NoError(t, err)
	defer rows.Close()

	type draft struct {
		creator            string
		name               string
		description        string
		status             string
		issueType          string
		draft              bool
		labelCount         int
		createdAtMigration bool
		searchVector       string
	}
	got := make(map[string]draft)
	for rows.Next() {
		var planName string
		var value draft
		require.NoError(t, rows.Scan(
			&planName,
			&value.creator,
			&value.name,
			&value.description,
			&value.status,
			&value.issueType,
			&value.draft,
			&value.labelCount,
			&value.createdAtMigration,
			&value.searchVector,
		))
		got[planName] = value
	}
	require.NoError(t, rows.Err())

	require.Equal(t, map[string]draft{
		"recent change": {
			creator: "change@example.com", name: "recent change", description: "change description",
			status: "OPEN", issueType: "DATABASE_CHANGE", draft: true, createdAtMigration: true,
			searchVector: "'change':2,3 'description':4 'recent':1",
		},
		"boundary create": {
			creator: "create@example.com", name: "boundary create", description: "create description",
			status: "OPEN", issueType: "DATABASE_CHANGE", draft: true, createdAtMigration: true,
			searchVector: "'boundary':1 'create':2,3 'description':4",
		},
		"other project change": {
			creator: "other@example.com", name: "other project change", description: "other description",
			status: "OPEN", issueType: "DATABASE_CHANGE", draft: true, createdAtMigration: true,
			searchVector: "'change':3 'description':5 'other':1,4 'project':2",
		},
		"late commit": {
			creator: "late@example.com", name: "late commit", description: "late description",
			status: "OPEN", issueType: "DATABASE_CHANGE", draft: true, createdAtMigration: true,
			searchVector: "'commit':2 'description':4 'late':1,3",
		},
	}, got)

	_, err = container.GetDB().ExecContext(ctx, `
		INSERT INTO plan (id, creator, created_at, project, name, description, config)
		SELECT id, 'page@example.com', CURRENT_TIMESTAMP, 'project-a', 'paged plan', '',
			'{"specs":[{"changeDatabaseConfig":{}}]}'
		FROM generate_series(1000, 1100) AS id`)
	require.NoError(t, err)
	require.NoError(t, migrate3_21_1At(ctx, conn, migrationTime))
	var pagedIssueCount int
	require.NoError(t, container.GetDB().QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM issue
		WHERE project = 'project-a' AND plan_id BETWEEN 1000 AND 1100`).Scan(&pagedIssueCount))
	require.Equal(t, 101, pagedIssueCount)

	_, err = container.GetDB().ExecContext(ctx, `
		INSERT INTO plan (id, creator, created_at, project, name, description, config)
		VALUES (12, 'race@example.com', CURRENT_TIMESTAMP, 'project-a', 'racing rollout', '',
			'{"specs":[{"changeDatabaseConfig":{}}]}')`)
	require.NoError(t, err)

	rolloutTx, err := container.GetDB().BeginTx(ctx, nil)
	require.NoError(t, err)
	defer rolloutTx.Rollback()
	require.NoError(t, store.AcquireAdvisoryXactLockWithStringKey(
		ctx,
		rolloutTx,
		store.AdvisoryLockKeyPlanIssueRollout,
		"project-a/12",
	))

	raceConn, err := container.GetDB().Conn(ctx)
	require.NoError(t, err)
	defer raceConn.Close()
	migrationDone := make(chan error, 1)
	go func() {
		migrationDone <- migrate3_21_1At(ctx, raceConn, migrationTime)
	}()
	require.Eventually(t, func() bool {
		var waiting bool
		err := container.GetDB().QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM pg_locks
				WHERE locktype = 'advisory'
				  AND classid = $1
				  AND NOT granted
			)`, store.AdvisoryLockKeyPlanIssueRollout).Scan(&waiting)
		return err == nil && waiting
	}, 5*time.Second, 10*time.Millisecond)
	_, err = rolloutTx.ExecContext(ctx, `
		UPDATE plan
		SET config = jsonb_set(config, '{hasRollout}', 'true'::jsonb, true)
		WHERE project = 'project-a' AND id = 12`)
	require.NoError(t, err)
	require.NoError(t, rolloutTx.Commit())
	require.NoError(t, <-migrationDone)

	var racingIssueCount int
	require.NoError(t, container.GetDB().QueryRowContext(ctx, `
		SELECT COUNT(*) FROM issue WHERE project = 'project-a' AND plan_id = 12`).Scan(&racingIssueCount))
	require.Zero(t, racingIssueCount)
}

func TestMigration3_17_15_DedupeReadOnlyDataSources(t *testing.T) {
	ctx := context.Background()
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(ctx) })

	db := container.GetDB()

	setup := `
		CREATE TABLE instance (
			resource_id TEXT PRIMARY KEY,
			metadata JSONB NOT NULL DEFAULT '{}'
		);

		INSERT INTO instance (resource_id, metadata) VALUES
			(
				'instance-with-duplicate-read-only',
				'{"dataSources":[{"id":"admin","type":"ADMIN","username":"admin"},{"id":"read-only-1","type":"READ_ONLY","username":"readonly-1"},{"id":"read-only-2","type":"READ_ONLY","username":"readonly-2"}]}'
			),
			(
				'instance-with-single-read-only',
				'{"dataSources":[{"id":"admin","type":"ADMIN","username":"admin"},{"id":"read-only-1","type":"READ_ONLY","username":"readonly-1"}]}'
			),
			(
				'instance-with-non-array-data-sources',
				'{"dataSources":{"id":"read-only-1","type":"READ_ONLY","username":"readonly-1"}}'
			),
			(
				'instance-without-data-sources',
				'{"engine":"POSTGRES"}'
			);
	`
	_, err := db.ExecContext(ctx, setup)
	require.NoError(t, err)

	statement, err := migrationFS.ReadFile("migration/3.17/0015##dedupe_read_only_data_sources.sql")
	require.NoError(t, err)

	_, err = db.ExecContext(ctx, string(statement))
	require.NoError(t, err)

	getMetadata := func(resourceID string) string {
		t.Helper()
		var metadata string
		err := db.QueryRowContext(ctx, `SELECT metadata::text FROM instance WHERE resource_id = $1`, resourceID).Scan(&metadata)
		require.NoError(t, err)
		return metadata
	}

	require.JSONEq(t, `{"dataSources":[{"id":"admin","type":"ADMIN","username":"admin"},{"id":"read-only-1","type":"READ_ONLY","username":"readonly-1"}]}`, getMetadata("instance-with-duplicate-read-only"))
	require.JSONEq(t, `{"dataSources":[{"id":"admin","type":"ADMIN","username":"admin"},{"id":"read-only-1","type":"READ_ONLY","username":"readonly-1"}]}`, getMetadata("instance-with-single-read-only"))
	require.JSONEq(t, `{"dataSources":{"id":"read-only-1","type":"READ_ONLY","username":"readonly-1"}}`, getMetadata("instance-with-non-array-data-sources"))
	require.JSONEq(t, `{"engine":"POSTGRES"}`, getMetadata("instance-without-data-sources"))
}

// TestMigration3_7_20_ScalarTaskUpdateTasks verifies that the migration 3.7.20
// UPDATE on issue_comment handles scalar (non-array) taskUpdate.tasks values
// without error. Regression test for "cannot get array length of a scalar".
func TestMigration3_7_20_ScalarTaskUpdateTasks(t *testing.T) {
	ctx := context.Background()
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(ctx) })

	db := container.GetDB()

	// Create minimal schema.
	setup := `
		CREATE TABLE stage (
			id INT PRIMARY KEY,
			environment TEXT NOT NULL
		);
		INSERT INTO stage (id, environment) VALUES (101, 'environments/prod');

		CREATE OR REPLACE FUNCTION update_stage_reference(resource_path text) RETURNS text AS $$
		DECLARE
			stage_match text;
			stage_id int;
			environment_id text;
		BEGIN
			IF resource_path !~ '/stages/[0-9]+' THEN
				RETURN resource_path;
			END IF;
			stage_match := substring(resource_path from '/stages/([0-9]+)');
			IF stage_match IS NULL THEN
				RETURN resource_path;
			END IF;
			stage_id := stage_match::int;
			SELECT s.environment INTO environment_id FROM stage s WHERE s.id = stage_id;
			IF environment_id IS NULL THEN
				RETURN resource_path;
			END IF;
			RETURN regexp_replace(resource_path, '/stages/' || stage_id, '/stages/' || environment_id);
		END;
		$$ LANGUAGE plpgsql;

		CREATE TABLE issue_comment (
			id SERIAL PRIMARY KEY,
			payload JSONB NOT NULL
		);

		INSERT INTO issue_comment (payload) VALUES
			('{"taskUpdate":{"tasks":["projects/p1/rollouts/1/stages/101/tasks/1"]}}'),
			('{"taskUpdate":{"tasks":"projects/p1/rollouts/1/stages/101/tasks/1"}}'),
			('{"taskUpdate":{"tasks":null}}');
	`
	_, err := db.ExecContext(ctx, setup)
	require.NoError(t, err)

	// Run the exact UPDATE from migration 3.7.20 with the fixed WHERE clause.
	migrate := `
		UPDATE issue_comment
		SET payload = jsonb_set(
			payload,
			'{taskUpdate,tasks}',
			(
				SELECT jsonb_agg(update_stage_reference(task_ref))
				FROM jsonb_array_elements_text(payload->'taskUpdate'->'tasks') AS task_ref
			)
		)
		WHERE payload->'taskUpdate' IS NOT NULL
		  AND jsonb_typeof(payload->'taskUpdate'->'tasks') = 'array'
		  AND CASE WHEN jsonb_typeof(payload->'taskUpdate'->'tasks') = 'array'
		           THEN jsonb_array_length(payload->'taskUpdate'->'tasks') > 0
		           ELSE false END;
	`
	_, err = db.ExecContext(ctx, migrate)
	require.NoError(t, err, "migration UPDATE must not fail on scalar tasks values")

	// Verify: valid array row was rewritten with environment ID.
	var arrayPayload string
	err = db.QueryRowContext(ctx, `SELECT payload::text FROM issue_comment WHERE id = 1`).Scan(&arrayPayload)
	require.NoError(t, err)
	assert.Contains(t, arrayPayload, "environments/prod", "array row should have rewritten stage reference")
	assert.NotContains(t, arrayPayload, "stages/101", "array row should no longer have numeric stage ID")

	// Verify: scalar row was NOT modified.
	var scalarPayload string
	err = db.QueryRowContext(ctx, `SELECT payload::text FROM issue_comment WHERE id = 2`).Scan(&scalarPayload)
	require.NoError(t, err)
	assert.Contains(t, scalarPayload, "stages/101", "scalar row should be unchanged")

	// Verify: null row was NOT modified.
	var nullPayload string
	err = db.QueryRowContext(ctx, `SELECT payload::text FROM issue_comment WHERE id = 3`).Scan(&nullPayload)
	require.NoError(t, err)
	assert.Contains(t, nullPayload, "null", "null row should be unchanged")
}

// TestMigration3_22_5_ScopeSheetBlob verifies the sheet_blob_ref backfill.
//
// The four project-scoped sources (plan, task, release, plan_check_run) each
// yield one (project, hash) ref; references naming a hash with no blob are
// skipped; unreferenced blobs end with zero refs; and a hash claimed by more
// than one project is recorded per project so the multi-project audit reports
// it — detected, not silently merged.
//
// The revision branch derives the authoring project from corroborated
// provenance (never db.project). A corroborating row is almost always itself
// a source granting the same (project, hash), so the observable probe for
// each corroboration rule is the one additive shape: task-run provenance
// whose task routes its hash through a release in a DIFFERENT project. There
// the revision branch grants (task-run project, hash), which no source scan
// produces, so a rule failure is visible as that ref's absence.
func TestMigration3_22_5_ScopeSheetBlob(t *testing.T) {
	ctx := context.Background()
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(ctx) })
	db := container.GetDB()

	_, err := db.ExecContext(ctx, `
		CREATE TABLE project (resource_id TEXT PRIMARY KEY);
		CREATE TABLE sheet_blob (sha256 BYTEA NOT NULL PRIMARY KEY, content TEXT NOT NULL);
		CREATE TABLE plan (project TEXT NOT NULL, config JSONB NOT NULL DEFAULT '{}');
		CREATE TABLE task (project TEXT NOT NULL, id BIGINT NOT NULL, payload JSONB NOT NULL DEFAULT '{}');
		CREATE TABLE task_run (project TEXT NOT NULL, id BIGINT NOT NULL, task_id INTEGER NOT NULL, created_at TIMESTAMPTZ NOT NULL DEFAULT now());
		CREATE TABLE release (project TEXT NOT NULL, release_id TEXT NOT NULL DEFAULT '', created_at TIMESTAMPTZ NOT NULL DEFAULT now(), payload JSONB NOT NULL DEFAULT '{}');
		CREATE TABLE plan_check_run (project TEXT NOT NULL, result JSONB NOT NULL DEFAULT '{}');
		CREATE TABLE revision (resource_id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text, instance TEXT NOT NULL, db_name TEXT NOT NULL, created_at TIMESTAMPTZ NOT NULL DEFAULT now(), payload JSONB NOT NULL DEFAULT '{}');
		INSERT INTO project VALUES
			('p1'), ('p2'), ('p3'), ('p4'),
			('rel-ok'), ('rel-upper'), ('tr-ok'), ('rel-old'), ('tr-late'), ('rel-mismatch'), ('tr-mismatch'),
			('rel-dup'), ('tr-dup'), ('rel-task'), ('rel-taskid'), ('tr-taskid'), ('tr-direct'),
			('rel-prefer'), ('rel-nest-prefer'), ('tr-prefer'),
			('rel-newer'), ('rel-nest-fb'), ('tr-fb');
	`)
	require.NoError(t, err)

	sheetHex := func(content string) string {
		h := sha256.Sum256([]byte(content))
		return hex.EncodeToString(h[:])
	}
	blobs := map[string]string{}
	for _, name := range []string{
		"plan", "task", "release", "pcr", "shared", "orphan",
		"rev-rel", "rev-upper", "rev-nest", "rev-late", "rev-mismatch", "rev-other",
		"rev-dup", "rev-taskid", "rev-direct", "rev-prefer", "rev-fb", "rev-ghost", "rev-bare",
		"rev-overflow", "rev-stale-stamp", "rev-empty-stamp",
	} {
		content := "SELECT '" + name + "';"
		blobs[name] = sheetHex(content)
		_, err := db.ExecContext(ctx,
			"INSERT INTO sheet_blob VALUES (decode($1, 'hex'), $2)", blobs[name], content)
		require.NoError(t, err)
	}
	missingBlob := strings.Repeat("f", 64)

	const (
		older = "2026-01-01 00:00:00+00" // predates every revision
		rev   = "2026-02-01 00:00:00+00" // every revision's created_at
		newer = "2026-03-01 00:00:00+00" // postdates every revision
	)
	taskRunName := func(project string, taskID, taskRunID int) string {
		return fmt.Sprintf("projects/%s/plans/1/rollout/stages/prod/tasks/%d/taskRuns/%d", project, taskID, taskRunID)
	}

	type seed struct {
		query string
		args  []any
	}
	seeds := []seed{
		// The four project-scoped sources.
		{"INSERT INTO plan (project, config) VALUES ('p1', $1::jsonb)",
			[]any{fmt.Sprintf(`{"specs":[{"changeDatabaseConfig":{"sheetSha256":%q}},{"createDatabaseConfig":{}}]}`, blobs["plan"])}},
		// The same hash referenced from two projects: honest dedup or a
		// laundered reference — indistinguishable, so both refs are
		// backfilled and the multi-project audit reports the hash.
		{"INSERT INTO plan (project, config) VALUES ('p1', $1::jsonb)",
			[]any{fmt.Sprintf(`{"specs":[{"changeDatabaseConfig":{"sheetSha256":%q}}]}`, blobs["shared"])}},
		{"INSERT INTO task (project, id, payload) VALUES ('p2', 1, $1::jsonb)",
			[]any{fmt.Sprintf(`{"sheetSha256":%q}`, blobs["task"])}},
		{"INSERT INTO task (project, id, payload) VALUES ('p2', 2, $1::jsonb)",
			[]any{fmt.Sprintf(`{"sheetSha256":%q}`, blobs["shared"])}},
		// A reference naming a hash with no blob is skipped by the EXISTS
		// guard, not a foreign-key abort.
		{"INSERT INTO task (project, id, payload) VALUES ('p2', 3, $1::jsonb)",
			[]any{fmt.Sprintf(`{"sheetSha256":%q}`, missingBlob)}},
		{"INSERT INTO task (project, id, payload) VALUES ('p2', 4, '{}')", nil},
		{"INSERT INTO release (project, release_id, payload) VALUES ('p3', 'r1', $1::jsonb)",
			[]any{fmt.Sprintf(`{"files":[{"path":"a.sql","sheetSha256":%q},{"path":"no-sheet.sql"}]}`, blobs["release"])}},
		{"INSERT INTO plan_check_run (project, result) VALUES ('p4', $1::jsonb)",
			[]any{fmt.Sprintf(`{"results":[{"sheetSha256":%q},{"title":"no sheet"}]}`, blobs["pcr"])}},

		// Null and malformed shapes: every one must expand to nothing, not
		// abort. JSON null in place of an array is the case COALESCE alone
		// would pass through to jsonb_array_elements.
		{"INSERT INTO plan (project, config) VALUES ('p1', '{}')", nil},
		{`INSERT INTO plan (project, config) VALUES ('p1', '{"specs": null}')`, nil},
		{`INSERT INTO plan (project, config) VALUES ('p1', '{"specs": []}')`, nil},
		{`INSERT INTO plan (project, config) VALUES ('p1', '{"specs": ["garbage", null]}')`, nil},
		{"INSERT INTO task (project, id, payload) VALUES ('p2', 5, 'null'::jsonb)", nil},
		{`INSERT INTO release (project, release_id, payload) VALUES ('p3', 'r-null', '{"files": null}')`, nil},
		{`INSERT INTO release (project, release_id, payload) VALUES ('p3', 'r-nullelem', '{"files": [null]}')`, nil},
		{`INSERT INTO plan_check_run (project, result) VALUES ('p4', '{"results": null}')`, nil},

		// Revision: release provenance corroborates (exactly one row, older,
		// carries the hash) → (rel-ok, rev-rel). Redundant with the release
		// source, so presence-only.
		{"INSERT INTO release (project, release_id, created_at, payload) VALUES ('rel-ok', 'r', $1, $2::jsonb)",
			[]any{older, fmt.Sprintf(`{"files":[{"sheetSha256":%q}]}`, blobs["rev-rel"])}},
		{"INSERT INTO revision (instance, db_name, created_at, payload) VALUES ('i', 'd', $1, $2::jsonb)",
			[]any{rev, fmt.Sprintf(`{"sheetSha256":%q,"release":"projects/rel-ok/releases/r"}`, blobs["rev-rel"])}},

		// Legacy uppercase hex: the revision stores the hash in uppercase
		// while the release file carries lowercase. Text comparisons are
		// lowered, so this corroborates → (rel-upper, rev-upper).
		{"INSERT INTO release (project, release_id, created_at, payload) VALUES ('rel-upper', 'r', $1, $2::jsonb)",
			[]any{older, fmt.Sprintf(`{"files":[{"sheetSha256":%q}]}`, blobs["rev-upper"])}},
		{"INSERT INTO revision (instance, db_name, created_at, payload) VALUES ('i', 'd', $1, $2::jsonb)",
			[]any{rev, fmt.Sprintf(`{"sheetSha256":%q,"release":"projects/rel-upper/releases/r"}`, strings.ToUpper(blobs["rev-upper"]))}},

		// Observable probe, all constraints satisfied: task-run provenance in
		// tr-ok whose task routes the hash through a release in rel-task.
		// Expect the additive (tr-ok, rev-nest) alongside the source-derived
		// (rel-task, rev-nest).
		{"INSERT INTO release (project, release_id, created_at, payload) VALUES ('rel-task', 'nest', $1, $2::jsonb)",
			[]any{older, fmt.Sprintf(`{"files":[{"sheetSha256":%q}]}`, blobs["rev-nest"])}},
		{"INSERT INTO task (project, id, payload) VALUES ('tr-ok', 10, $1::jsonb)",
			[]any{`{"release":"projects/rel-task/releases/nest"}`}},
		{"INSERT INTO task_run (project, id, task_id, created_at) VALUES ('tr-ok', 100, 10, $1)", []any{older}},
		{"INSERT INTO revision (instance, db_name, created_at, payload) VALUES ('i', 'd', $1, $2::jsonb)",
			[]any{rev, fmt.Sprintf(`{"sheetSha256":%q,"taskRun":%q}`, blobs["rev-nest"], taskRunName("tr-ok", 10, 100))}},

		// Temporal violation: the task run postdates the revision (the ID-reuse
		// shape after a purge). No (tr-late, rev-late).
		{"INSERT INTO release (project, release_id, created_at, payload) VALUES ('rel-old', 'nest', $1, $2::jsonb)",
			[]any{older, fmt.Sprintf(`{"files":[{"sheetSha256":%q}]}`, blobs["rev-late"])}},
		{"INSERT INTO task (project, id, payload) VALUES ('tr-late', 10, $1::jsonb)",
			[]any{`{"release":"projects/rel-old/releases/nest"}`}},
		{"INSERT INTO task_run (project, id, task_id, created_at) VALUES ('tr-late', 100, 10, $1)", []any{newer}},
		{"INSERT INTO revision (instance, db_name, created_at, payload) VALUES ('i', 'd', $1, $2::jsonb)",
			[]any{rev, fmt.Sprintf(`{"sheetSha256":%q,"taskRun":%q}`, blobs["rev-late"], taskRunName("tr-late", 10, 100))}},

		// Hash mismatch: the nested release carries a different hash than the
		// revision claims (laundered provenance). No (tr-mismatch, rev-mismatch).
		{"INSERT INTO release (project, release_id, created_at, payload) VALUES ('rel-mismatch', 'nest', $1, $2::jsonb)",
			[]any{older, fmt.Sprintf(`{"files":[{"sheetSha256":%q}]}`, blobs["rev-other"])}},
		{"INSERT INTO task (project, id, payload) VALUES ('tr-mismatch', 10, $1::jsonb)",
			[]any{`{"release":"projects/rel-mismatch/releases/nest"}`}},
		{"INSERT INTO task_run (project, id, task_id, created_at) VALUES ('tr-mismatch', 100, 10, $1)", []any{older}},
		{"INSERT INTO revision (instance, db_name, created_at, payload) VALUES ('i', 'd', $1, $2::jsonb)",
			[]any{rev, fmt.Sprintf(`{"sheetSha256":%q,"taskRun":%q}`, blobs["rev-mismatch"], taskRunName("tr-mismatch", 10, 100))}},

		// (project, release_id) matching two rows: not a unique key, so
		// exactly-one fails even though both rows carry the hash and predate
		// the revision. No (tr-dup, rev-dup).
		{"INSERT INTO release (project, release_id, created_at, payload) VALUES ('rel-dup', 'nest', $1, $2::jsonb)",
			[]any{older, fmt.Sprintf(`{"files":[{"sheetSha256":%q}]}`, blobs["rev-dup"])}},
		{"INSERT INTO release (project, release_id, created_at, payload) VALUES ('rel-dup', 'nest', $1, $2::jsonb)",
			[]any{older, fmt.Sprintf(`{"files":[{"sheetSha256":%q}]}`, blobs["rev-dup"])}},
		{"INSERT INTO task (project, id, payload) VALUES ('tr-dup', 10, $1::jsonb)",
			[]any{`{"release":"projects/rel-dup/releases/nest"}`}},
		{"INSERT INTO task_run (project, id, task_id, created_at) VALUES ('tr-dup', 100, 10, $1)", []any{older}},
		{"INSERT INTO revision (instance, db_name, created_at, payload) VALUES ('i', 'd', $1, $2::jsonb)",
			[]any{rev, fmt.Sprintf(`{"sheetSha256":%q,"taskRun":%q}`, blobs["rev-dup"], taskRunName("tr-dup", 10, 100))}},

		// Task-ID mismatch: the task run exists on (project, id) but belongs
		// to a different task than the name claims. No (tr-taskid, rev-taskid).
		{"INSERT INTO release (project, release_id, created_at, payload) VALUES ('rel-taskid', 'nest', $1, $2::jsonb)",
			[]any{older, fmt.Sprintf(`{"files":[{"sheetSha256":%q}]}`, blobs["rev-taskid"])}},
		{"INSERT INTO task (project, id, payload) VALUES ('tr-taskid', 11, $1::jsonb)",
			[]any{`{"release":"projects/rel-taskid/releases/nest"}`}},
		{"INSERT INTO task_run (project, id, task_id, created_at) VALUES ('tr-taskid', 100, 11, $1)", []any{older}},
		{"INSERT INTO revision (instance, db_name, created_at, payload) VALUES ('i', 'd', $1, $2::jsonb)",
			[]any{rev, fmt.Sprintf(`{"sheetSha256":%q,"taskRun":%q}`, blobs["rev-taskid"], taskRunName("tr-taskid", 10, 100))}},

		// Direct task hash: corroborates through task.payload.sheetSha256.
		// Redundant with the task source, so presence-only.
		{"INSERT INTO task (project, id, payload) VALUES ('tr-direct', 10, $1::jsonb)",
			[]any{fmt.Sprintf(`{"sheetSha256":%q}`, blobs["rev-direct"])}},
		{"INSERT INTO task_run (project, id, task_id, created_at) VALUES ('tr-direct', 100, 10, $1)", []any{older}},
		{"INSERT INTO revision (instance, db_name, created_at, payload) VALUES ('i', 'd', $1, $2::jsonb)",
			[]any{rev, fmt.Sprintf(`{"sheetSha256":%q,"taskRun":%q}`, blobs["rev-direct"], taskRunName("tr-direct", 10, 100))}},

		// Preference: both release and task-run provenance corroborate. The
		// release wins, so the task-run fallback's additive grant
		// (tr-prefer, rev-prefer) must NOT appear.
		{"INSERT INTO release (project, release_id, created_at, payload) VALUES ('rel-prefer', 'r', $1, $2::jsonb)",
			[]any{older, fmt.Sprintf(`{"files":[{"sheetSha256":%q}]}`, blobs["rev-prefer"])}},
		{"INSERT INTO release (project, release_id, created_at, payload) VALUES ('rel-nest-prefer', 'nest', $1, $2::jsonb)",
			[]any{older, fmt.Sprintf(`{"files":[{"sheetSha256":%q}]}`, blobs["rev-prefer"])}},
		{"INSERT INTO task (project, id, payload) VALUES ('tr-prefer', 10, $1::jsonb)",
			[]any{`{"release":"projects/rel-nest-prefer/releases/nest"}`}},
		{"INSERT INTO task_run (project, id, task_id, created_at) VALUES ('tr-prefer', 100, 10, $1)", []any{older}},
		{"INSERT INTO revision (instance, db_name, created_at, payload) VALUES ('i', 'd', $1, $2::jsonb)",
			[]any{rev, fmt.Sprintf(`{"sheetSha256":%q,"release":"projects/rel-prefer/releases/r","taskRun":%q}`,
				blobs["rev-prefer"], taskRunName("tr-prefer", 10, 100))}},

		// Fallback: the release provenance fails on age (the release postdates
		// the revision), so the corroborated task run supplies the grant —
		// (tr-fb, rev-fb) must appear.
		{"INSERT INTO release (project, release_id, created_at, payload) VALUES ('rel-newer', 'r', $1, $2::jsonb)",
			[]any{newer, fmt.Sprintf(`{"files":[{"sheetSha256":%q}]}`, blobs["rev-fb"])}},
		{"INSERT INTO release (project, release_id, created_at, payload) VALUES ('rel-nest-fb', 'nest', $1, $2::jsonb)",
			[]any{older, fmt.Sprintf(`{"files":[{"sheetSha256":%q}]}`, blobs["rev-fb"])}},
		{"INSERT INTO task (project, id, payload) VALUES ('tr-fb', 10, $1::jsonb)",
			[]any{`{"release":"projects/rel-nest-fb/releases/nest"}`}},
		{"INSERT INTO task_run (project, id, task_id, created_at) VALUES ('tr-fb', 100, 10, $1)", []any{older}},
		{"INSERT INTO revision (instance, db_name, created_at, payload) VALUES ('i', 'd', $1, $2::jsonb)",
			[]any{rev, fmt.Sprintf(`{"sheetSha256":%q,"release":"projects/rel-newer/releases/r","taskRun":%q}`,
				blobs["rev-fb"], taskRunName("tr-fb", 10, 100))}},

		// Provenance naming rows that no longer exist (purged authoring
		// project): no ref at all, the hash stays zero-ref.
		{"INSERT INTO revision (instance, db_name, created_at, payload) VALUES ('i', 'd', $1, $2::jsonb)",
			[]any{rev, fmt.Sprintf(`{"sheetSha256":%q,"release":"projects/ghost/releases/r","taskRun":%q}`,
				blobs["rev-ghost"], taskRunName("ghost", 10, 100))}},

		// No provenance at all: no ref, the hash stays zero-ref.
		{"INSERT INTO revision (instance, db_name, created_at, payload) VALUES ('i', 'd', $1, $2::jsonb)",
			[]any{rev, fmt.Sprintf(`{"sheetSha256":%q}`, blobs["rev-bare"])}},

		// Abort hazards: neither row may fail the migration.
		// A taskRun name whose IDs overflow bigint fails the bounded digit
		// match and stays uncorroborated instead of erroring the cast.
		{"INSERT INTO revision (instance, db_name, created_at, payload) VALUES ('i', 'd', $1, $2::jsonb)",
			[]any{rev, fmt.Sprintf(`{"sheetSha256":%q,"taskRun":"projects/tr-ok/plans/1/rollout/stages/prod/tasks/99999999999999999999999/taskRuns/99999999999999999999999"}`,
				blobs["rev-overflow"])}},
		// A pre-existing payload.project naming no live project row is kept
		// as-is but granted no ref: the project EXISTS guard skips it instead
		// of aborting on the foreign key.
		{"INSERT INTO revision (instance, db_name, created_at, payload) VALUES ('i', 'd', $1, $2::jsonb)",
			[]any{rev, fmt.Sprintf(`{"sheetSha256":%q,"project":"stale-ghost"}`, blobs["rev-stale-stamp"])}},
		// Same for the empty string, which survives the IS NOT NULL filter.
		{"INSERT INTO revision (instance, db_name, created_at, payload) VALUES ('i', 'd', $1, $2::jsonb)",
			[]any{rev, fmt.Sprintf(`{"sheetSha256":%q,"project":""}`, blobs["rev-empty-stamp"])}},
	}
	for _, s := range seeds {
		_, err := db.ExecContext(ctx, s.query, s.args...)
		require.NoError(t, err, s.query)
	}

	statement, err := migrationFS.ReadFile("migration/3.22/0005##scope_sheet_blob.sql")
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, string(statement))
	require.NoError(t, err)

	rows, err := db.QueryContext(ctx,
		"SELECT project, encode(sha256, 'hex') FROM sheet_blob_ref ORDER BY project, sha256")
	require.NoError(t, err)
	defer rows.Close()
	var got []string
	for rows.Next() {
		var project, sha string
		require.NoError(t, rows.Scan(&project, &sha))
		got = append(got, project+"|"+sha)
	}
	require.NoError(t, rows.Err())
	require.ElementsMatch(t, []string{
		// The four project-scoped sources.
		"p1|" + blobs["plan"],
		"p1|" + blobs["shared"],
		"p2|" + blobs["task"],
		"p2|" + blobs["shared"],
		"p3|" + blobs["release"],
		"p4|" + blobs["pcr"],
		// Release rows seeded as corroboration targets are sources themselves.
		"rel-ok|" + blobs["rev-rel"],
		"rel-upper|" + blobs["rev-upper"],
		"rel-task|" + blobs["rev-nest"],
		"rel-old|" + blobs["rev-late"],
		"rel-mismatch|" + blobs["rev-other"],
		"rel-dup|" + blobs["rev-dup"],
		"rel-taskid|" + blobs["rev-taskid"],
		"tr-direct|" + blobs["rev-direct"],
		"rel-prefer|" + blobs["rev-prefer"],
		"rel-nest-prefer|" + blobs["rev-prefer"],
		"rel-newer|" + blobs["rev-fb"],
		"rel-nest-fb|" + blobs["rev-fb"],
		// The revision branch's observable grants: corroborated task-run
		// provenance routing through another project's release, and the
		// age-based fallback. Every negative scenario is covered by this
		// list's exactness: tr-late, tr-mismatch, tr-dup, tr-taskid and
		// tr-prefer hold no refs.
		"tr-ok|" + blobs["rev-nest"],
		"tr-fb|" + blobs["rev-fb"],
	}, got)

	// Corroboration also stamps payload.project — the stored fact new
	// revision writers set at creation. Each scenario's revision is
	// identified by its unique hash; an empty stamp means the provenance did
	// not corroborate. rev-prefer pins the release-over-taskRun preference
	// directly: both branches corroborate and the release's project wins.
	for sha, want := range map[string]string{
		blobs["rev-rel"]:         "rel-ok",
		blobs["rev-upper"]:       "rel-upper",
		blobs["rev-nest"]:        "tr-ok",
		blobs["rev-late"]:        "",
		blobs["rev-mismatch"]:    "",
		blobs["rev-dup"]:         "",
		blobs["rev-taskid"]:      "",
		blobs["rev-direct"]:      "tr-direct",
		blobs["rev-prefer"]:      "rel-prefer",
		blobs["rev-fb"]:          "tr-fb",
		blobs["rev-ghost"]:       "",
		blobs["rev-bare"]:        "",
		blobs["rev-overflow"]:    "",
		blobs["rev-stale-stamp"]: "stale-ghost",
		blobs["rev-empty-stamp"]: "",
	} {
		var got string
		require.NoError(t, db.QueryRowContext(ctx,
			"SELECT COALESCE(payload->>'project', '') FROM revision WHERE lower(payload->>'sheetSha256') = $1", sha).Scan(&got))
		require.Equal(t, want, got, "stamped project for hash %s", sha)
	}

	// The zero-ref audit counts the never-referenced blob plus the revisions
	// whose provenance was absent or uncorroborated with no other source.
	var zeroRefBlobs int
	require.NoError(t, db.QueryRowContext(ctx, `
		SELECT count(*) FROM sheet_blob b
		WHERE NOT EXISTS (SELECT 1 FROM sheet_blob_ref r WHERE r.sha256 = b.sha256)
	`).Scan(&zeroRefBlobs))
	require.Equal(t, 7, zeroRefBlobs, "orphan, rev-mismatch, rev-ghost, rev-bare, rev-overflow, rev-stale-stamp, and rev-empty-stamp have zero refs")

	// The multi-project audit reports every hash claimed by more than one
	// project, including the deliberately shared one.
	auditRows, err := db.QueryContext(ctx, `
		SELECT encode(sha256, 'hex')
		FROM sheet_blob_ref
		GROUP BY sha256
		HAVING count(DISTINCT project) > 1
	`)
	require.NoError(t, err)
	defer auditRows.Close()
	multiProject := map[string]bool{}
	for auditRows.Next() {
		var sha string
		require.NoError(t, auditRows.Scan(&sha))
		multiProject[sha] = true
	}
	require.NoError(t, auditRows.Err())
	require.True(t, multiProject[blobs["shared"]], "the cross-project reference must be reported, not blessed")

	// The source-level missing-ref audit from the design doc: a plan that
	// references a pre-existing foreign hash AFTER the backfill invokes no
	// CreateSheets and writes no ref, so only this query can see it.
	_, err = db.ExecContext(ctx, "INSERT INTO plan (project, config) VALUES ('p1', $1::jsonb)",
		fmt.Sprintf(`{"specs":[{"changeDatabaseConfig":{"sheetSha256":%q}}]}`, blobs["pcr"]))
	require.NoError(t, err)
	var missingProject, missingSha string
	require.NoError(t, db.QueryRowContext(ctx, `
		SELECT DISTINCT pl.project, spec->'changeDatabaseConfig'->>'sheetSha256' AS sha
		FROM plan pl
		CROSS JOIN LATERAL jsonb_array_elements(
			CASE WHEN jsonb_typeof(pl.config->'specs') = 'array' THEN pl.config->'specs' ELSE '[]'::jsonb END) AS spec
		WHERE spec->'changeDatabaseConfig'->>'sheetSha256' ~ '^[0-9a-fA-F]{64}$'
			AND NOT EXISTS (
				SELECT 1 FROM sheet_blob_ref r
				WHERE r.project = pl.project
					AND r.sha256 = decode(spec->'changeDatabaseConfig'->>'sheetSha256', 'hex')
			)
	`).Scan(&missingProject, &missingSha))
	require.Equal(t, "p1", missingProject)
	require.Equal(t, blobs["pcr"], missingSha)
}
