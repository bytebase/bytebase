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
	require.Equal(t, semver.MustParse("3.18.2"), *files[len(files)-1].version)
	require.Equal(t, "migration/3.18/0002##oauth2_workspace_on_token.sql", files[len(files)-1].path)
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
