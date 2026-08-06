package migrator

import (
	"context"
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
	require.Equal(t, semver.MustParse("3.21.5"), *files[len(files)-1].version)
	require.Equal(t, "migration/3.21/0005##drop_task_run_log_pkey.sql", files[len(files)-1].path)
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

// TestMigration3_21_5_DropTaskRunLogPkey verifies that the pkey drop converges
// every historical shape — the 2-column PK left by upgrading through the
// original 3.16.2, the 3-column PK of fresh installs, and the no-PK state left
// by a backport of the 3.16.2 fix — is idempotent, and that same-microsecond
// entries insert afterwards instead of colliding.
func TestMigration3_21_5_DropTaskRunLogPkey(t *testing.T) {
	ctx := context.Background()
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(ctx) })

	db := container.GetDB()

	statement, err := migrationFS.ReadFile("migration/3.21/0005##drop_task_run_log_pkey.sql")
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
