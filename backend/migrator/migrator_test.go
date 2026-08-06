package migrator

import (
	"context"
	"testing"

	"github.com/blang/semver/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	_ "github.com/bytebase/bytebase/backend/plugin/db/pg"
)

func TestLatestVersion(t *testing.T) {
	files, err := getSortedVersionedFiles()
	require.NoError(t, err)
	require.Equal(t, semver.MustParse("3.20.5"), *files[len(files)-1].version)
	require.Equal(t, "migration/3.20/0005##drop_task_run_log_pkey.sql", files[len(files)-1].path)
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

// TestMigration3_20_5_DropTaskRunLogPkey verifies that the pkey drop converges
// every historical shape — the 2-column PK left by upgrading through the
// original 3.16.2, the 3-column PK of fresh installs, and the no-PK state left
// by a backport of the 3.16.2 fix — is idempotent, and that same-microsecond
// entries insert afterwards instead of colliding.
func TestMigration3_20_5_DropTaskRunLogPkey(t *testing.T) {
	ctx := context.Background()
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(ctx) })

	db := container.GetDB()

	statement, err := migrationFS.ReadFile("migration/3.20/0005##drop_task_run_log_pkey.sql")
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
			"no pkey from a backport of the 3.16.2 fix",
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
