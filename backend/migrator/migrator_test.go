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
	require.Equal(t, semver.MustParse("3.17.7"), *files[len(files)-1].version)
	require.Equal(t, "migration/3.17/0007##fix_project_fk_references.sql", files[len(files)-1].path)
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

func TestMigration3_16_0_SplitPrincipalTableOrdering(t *testing.T) {
	ctx := context.Background()
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(ctx) })

	db := container.GetDB()

	setup := `
		CREATE TABLE instance_change_history (
			id SERIAL PRIMARY KEY,
			version TEXT NOT NULL
		);

		CREATE TABLE project (
			resource_id TEXT PRIMARY KEY
		);
		INSERT INTO project(resource_id) VALUES ('proj');

		CREATE TABLE principal (
			id INT PRIMARY KEY,
			deleted BOOLEAN NOT NULL DEFAULT FALSE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			name TEXT NOT NULL,
			email TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL DEFAULT '',
			type TEXT NOT NULL,
			project TEXT,
			profile JSONB NOT NULL DEFAULT '{}'
		);

		INSERT INTO principal (id, name, email, password_hash, type, project, profile) VALUES
			(1, 'End User', 'user@bytebase.com', 'end-user-secret', 'END_USER', NULL, '{}'),
			(2, 'Service Account', 'legacy-sa', 'service-account-secret', 'SERVICE_ACCOUNT', 'proj', '{}'),
			(3, 'Workload Identity', 'wi@proj.service.bytebase.com', 'workload-identity-secret', 'WORKLOAD_IDENTITY', 'proj', '{"workloadIdentityConfig":{"issuer":"issuer"}}');

		CREATE TABLE policy (payload JSONB NOT NULL);
		INSERT INTO policy(payload) VALUES ('{"member":"serviceAccounts/legacy-sa"}');

		CREATE TABLE user_group (payload JSONB NOT NULL);
		INSERT INTO user_group(payload) VALUES ('{"member":"serviceAccounts/legacy-sa"}');

		CREATE TABLE plan (creator TEXT NOT NULL);
		CREATE TABLE task_run (creator TEXT NOT NULL);
		CREATE TABLE issue (creator TEXT NOT NULL);
		CREATE TABLE issue_comment (creator TEXT NOT NULL);
		CREATE TABLE query_history (creator TEXT NOT NULL);
		CREATE TABLE worksheet (creator TEXT NOT NULL);
		CREATE TABLE worksheet_organizer (principal TEXT NOT NULL);
		CREATE TABLE revision (deleter TEXT NOT NULL);
		CREATE TABLE release (creator TEXT NOT NULL);
		CREATE TABLE access_grant (creator TEXT NOT NULL);
	`
	_, err := db.ExecContext(ctx, setup)
	require.NoError(t, err)

	statement, err := migrationFS.ReadFile("migration/3.16/0000##split_principal_table.sql")
	require.NoError(t, err)

	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer conn.Close()

	err = executeMigration(ctx, conn, string(statement), "3.16.0")
	require.NoError(t, err, "migration should create split principal tables before the DO block uses them")

	var serviceAccountEmail string
	err = db.QueryRowContext(ctx, `SELECT email FROM service_account`).Scan(&serviceAccountEmail)
	require.NoError(t, err)
	assert.Equal(t, "legacy-sa@proj.service.bytebase.com", serviceAccountEmail)

	var workloadIdentityConfig string
	err = db.QueryRowContext(ctx, `SELECT config::text FROM workload_identity WHERE email = 'wi@proj.service.bytebase.com'`).Scan(&workloadIdentityConfig)
	require.NoError(t, err)
	assert.Contains(t, workloadIdentityConfig, "issuer")

	var principalCount int
	err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM principal`).Scan(&principalCount)
	require.NoError(t, err)
	assert.Equal(t, 1, principalCount, "only END_USER principals should remain")

	var hasTypeColumn bool
	err = db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_name = 'principal' AND column_name = 'type'
		)
	`).Scan(&hasTypeColumn)
	require.NoError(t, err)
	assert.False(t, hasTypeColumn, "principal.type should be removed after migration")

	var policyPayload string
	err = db.QueryRowContext(ctx, `SELECT payload::text FROM policy`).Scan(&policyPayload)
	require.NoError(t, err)
	assert.Contains(t, policyPayload, "serviceAccounts/legacy-sa@proj.service.bytebase.com")

	var userGroupPayload string
	err = db.QueryRowContext(ctx, `SELECT payload::text FROM user_group`).Scan(&userGroupPayload)
	require.NoError(t, err)
	assert.Contains(t, userGroupPayload, "serviceAccounts/legacy-sa@proj.service.bytebase.com")

	var recordedVersion string
	err = db.QueryRowContext(ctx, `SELECT version FROM instance_change_history ORDER BY id DESC LIMIT 1`).Scan(&recordedVersion)
	require.NoError(t, err)
	assert.Equal(t, "3.16.0", recordedVersion)
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
