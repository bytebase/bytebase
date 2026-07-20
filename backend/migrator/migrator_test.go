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
	require.Equal(t, semver.MustParse("3.21.1"), *files[len(files)-1].version)
	require.Equal(t, "migration/3.21/0001##backfill_ui_plan_draft_issues.sql", files[len(files)-1].path)
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
