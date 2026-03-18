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
	require.Equal(t, semver.MustParse("3.17.9"), *files[len(files)-1].version)
	require.Equal(t, "migration/3.17/0009##add_workspace_table.sql", files[len(files)-1].path)
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

// TestMigration3_16_0_SplitPrincipalTableDropsCreatorDeleterFKsBeforeDeletingNonEndUsers
// verifies migration 3.16.0 preserves ON UPDATE CASCADE for service-account email fixes
// and only deletes non-END_USER principals after creator/deleter FKs are dropped.
func TestMigration3_16_0_SplitPrincipalTableDropsCreatorDeleterFKsBeforeDeletingNonEndUsers(t *testing.T) {
	ctx := context.Background()
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(ctx) })

	db := container.GetDB()

	setup := `
		CREATE TABLE project (
			resource_id TEXT PRIMARY KEY
		);
		INSERT INTO project(resource_id) VALUES ('projects/demo');

		CREATE TABLE principal (
			id INT PRIMARY KEY,
			deleted BOOLEAN NOT NULL DEFAULT FALSE,
			created_at timestamptz NOT NULL DEFAULT now(),
			name TEXT NOT NULL,
			email TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			project TEXT,
			profile JSONB NOT NULL DEFAULT '{}'::jsonb,
			type TEXT NOT NULL
		);

		CREATE TABLE policy (
			payload JSONB NOT NULL DEFAULT '{}'::jsonb
		);
		CREATE TABLE user_group (
			payload JSONB NOT NULL DEFAULT '{}'::jsonb
		);

		CREATE TABLE plan (
			creator TEXT NOT NULL,
			CONSTRAINT plan_creator_fkey FOREIGN KEY (creator) REFERENCES principal(email) ON UPDATE CASCADE
		);
		CREATE TABLE revision (
			deleter TEXT,
			CONSTRAINT revision_deleter_fkey FOREIGN KEY (deleter) REFERENCES principal(email) ON UPDATE CASCADE
		);

		INSERT INTO principal (id, name, email, password_hash, project, type) VALUES
			(1, 'End User', 'user@example.com', 'hash', NULL, 'END_USER'),
			(2, 'Service Account', 'svc@legacy.example.com', 'service-hash', NULL, 'SERVICE_ACCOUNT');
		INSERT INTO plan(creator) VALUES ('svc@legacy.example.com');
		INSERT INTO revision(deleter) VALUES ('svc@legacy.example.com');
		INSERT INTO policy(payload) VALUES ('{"member":"serviceAccounts/svc@legacy.example.com"}');
		INSERT INTO user_group(payload) VALUES ('{"member":"serviceAccounts/svc@legacy.example.com"}');
	`
	_, err := db.ExecContext(ctx, setup)
	require.NoError(t, err)

	// Run the exact 3.16.0 steps relevant to email cascading and FK-drop ordering.
	migrate := `
		CREATE TABLE IF NOT EXISTS service_account (
			deleted boolean NOT NULL DEFAULT FALSE,
			created_at timestamptz NOT NULL DEFAULT now(),
			name text NOT NULL,
			email text NOT NULL PRIMARY KEY,
			service_key_hash text NOT NULL,
			project text REFERENCES project(resource_id)
		);

		CREATE INDEX IF NOT EXISTS idx_service_account_project ON service_account(project) WHERE project IS NOT NULL;

		CREATE TABLE IF NOT EXISTS workload_identity (
			deleted boolean NOT NULL DEFAULT FALSE,
			created_at timestamptz NOT NULL DEFAULT now(),
			name text NOT NULL,
			email text NOT NULL PRIMARY KEY,
			project text REFERENCES project(resource_id),
			config jsonb NOT NULL DEFAULT '{}'
		);

		CREATE INDEX IF NOT EXISTS idx_workload_identity_project ON workload_identity(project) WHERE project IS NOT NULL;

		DO $$
		DECLARE
			rec RECORD;
			new_email TEXT;
			base_local TEXT;
		BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_name = 'principal' AND column_name = 'type'
			) THEN
				RETURN;
			END IF;

			FOR rec IN
				SELECT id, email, project
				FROM principal
				WHERE type = 'SERVICE_ACCOUNT'
				  AND email NOT LIKE '%@service.bytebase.com'
				  AND email NOT LIKE '%@%.service.bytebase.com'
			LOOP
				base_local := split_part(rec.email, '@', 1);
				IF rec.project IS NOT NULL THEN
					new_email := base_local || '@' || rec.project || '.service.bytebase.com';
				ELSE
					new_email := base_local || '@service.bytebase.com';
				END IF;
				IF EXISTS (SELECT 1 FROM principal WHERE email = new_email AND id != rec.id) THEN
					IF rec.project IS NOT NULL THEN
						new_email := base_local || '-' || rec.id || '@' || rec.project || '.service.bytebase.com';
					ELSE
						new_email := base_local || '-' || rec.id || '@service.bytebase.com';
					END IF;
				END IF;
				UPDATE principal SET email = new_email WHERE id = rec.id;
				UPDATE policy
				SET payload = replace(payload::text, 'serviceAccounts/' || rec.email, 'serviceAccounts/' || new_email)::jsonb
				WHERE payload::text LIKE '%serviceAccounts/' || rec.email || '%';
				UPDATE user_group
				SET payload = replace(payload::text, 'serviceAccounts/' || rec.email, 'serviceAccounts/' || new_email)::jsonb
				WHERE payload::text LIKE '%serviceAccounts/' || rec.email || '%';
			END LOOP;

			INSERT INTO service_account (deleted, created_at, name, email, service_key_hash, project)
			SELECT deleted, created_at, name, email, password_hash, project
			FROM principal WHERE type = 'SERVICE_ACCOUNT'
			ON CONFLICT (email) DO NOTHING;

			INSERT INTO workload_identity (deleted, created_at, name, email, project, config)
			SELECT deleted, created_at, name, email, project,
			       COALESCE(profile->'workloadIdentityConfig', '{}')
			FROM principal WHERE type = 'WORKLOAD_IDENTITY'
			ON CONFLICT (email) DO NOTHING;
		END $$;

		ALTER TABLE plan DROP CONSTRAINT IF EXISTS plan_creator_fkey;
		ALTER TABLE revision DROP CONSTRAINT IF EXISTS revision_deleter_fkey;

		DO $$
		BEGIN
			IF EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_name = 'principal' AND column_name = 'type'
			) THEN
				DELETE FROM principal WHERE type != 'END_USER';
			END IF;
		END $$;
	`
	_, err = db.ExecContext(ctx, migrate)
	require.NoError(t, err, "migration 3.16.0 ordering must tolerate creator/deleter FK dependencies")

	var creator string
	err = db.QueryRowContext(ctx, `SELECT creator FROM plan`).Scan(&creator)
	require.NoError(t, err)
	assert.Equal(t, "svc@service.bytebase.com", creator)

	var deleter string
	err = db.QueryRowContext(ctx, `SELECT deleter FROM revision`).Scan(&deleter)
	require.NoError(t, err)
	assert.Equal(t, "svc@service.bytebase.com", deleter)

	var serviceAccountCount int
	err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM service_account WHERE email = 'svc@service.bytebase.com'`).Scan(&serviceAccountCount)
	require.NoError(t, err)
	assert.Equal(t, 1, serviceAccountCount)

	var nonEndUserCount int
	err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM principal WHERE type != 'END_USER'`).Scan(&nonEndUserCount)
	require.NoError(t, err)
	assert.Zero(t, nonEndUserCount)
}

// TestMigration3_16_2_DropUnusedIDColumnsReusesNaturalKeyIndexes verifies migration 3.16.2
// reuses existing natural-key indexes so dependent FKs survive the PK swap.
func TestMigration3_16_2_DropUnusedIDColumnsReusesNaturalKeyIndexes(t *testing.T) {
	ctx := context.Background()
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(ctx) })

	db := container.GetDB()

	setup := `
			CREATE TABLE project (
				id INT PRIMARY KEY,
				resource_id TEXT NOT NULL
			);
			CREATE UNIQUE INDEX idx_project_unique_resource_id ON project(resource_id);
			CREATE TABLE project_child (
				project TEXT NOT NULL REFERENCES project(resource_id)
			);

			CREATE TABLE instance (
				id INT PRIMARY KEY,
				resource_id TEXT NOT NULL
			);
			CREATE UNIQUE INDEX idx_instance_unique_resource_id ON instance(resource_id);
			CREATE TABLE instance_child (
				instance TEXT NOT NULL REFERENCES instance(resource_id)
			);

			CREATE TABLE db (
				id INT PRIMARY KEY,
				instance TEXT NOT NULL REFERENCES instance(resource_id),
				name TEXT NOT NULL
			);
			CREATE UNIQUE INDEX idx_db_unique_instance_name ON db(instance, name);
			CREATE TABLE db_child (
				instance TEXT NOT NULL,
				db_name TEXT NOT NULL,
				CONSTRAINT db_child_fk FOREIGN KEY (instance, db_name) REFERENCES db(instance, name)
			);

			CREATE TABLE setting (
				id INT PRIMARY KEY,
				name TEXT NOT NULL
			);
			CREATE UNIQUE INDEX idx_setting_unique_name ON setting(name);
			CREATE TABLE setting_child (
				setting_name TEXT NOT NULL REFERENCES setting(name)
			);

			CREATE TABLE policy (
				id INT PRIMARY KEY,
				resource_type TEXT NOT NULL,
				resource TEXT NOT NULL,
				type TEXT NOT NULL
			);
			CREATE UNIQUE INDEX idx_policy_unique_resource_type_resource_type ON policy(resource_type, resource, type);
			CREATE TABLE policy_child (
				resource_type TEXT NOT NULL,
				resource TEXT NOT NULL,
				policy_type TEXT NOT NULL,
				CONSTRAINT policy_child_fk FOREIGN KEY (resource_type, resource, policy_type) REFERENCES policy(resource_type, resource, type)
			);

			CREATE TABLE idp (
				id INT PRIMARY KEY,
				resource_id TEXT NOT NULL
			);
			CREATE UNIQUE INDEX idx_idp_unique_resource_id ON idp(resource_id);
			CREATE TABLE idp_child (
				idp TEXT NOT NULL REFERENCES idp(resource_id)
			);

			CREATE TABLE role (
				id INT PRIMARY KEY,
				resource_id TEXT NOT NULL
			);
			CREATE UNIQUE INDEX idx_role_unique_resource_id ON role(resource_id);
			CREATE TABLE role_child (
				role TEXT NOT NULL REFERENCES role(resource_id)
			);

			CREATE TABLE db_schema (
				id INT PRIMARY KEY,
				instance TEXT NOT NULL,
				db_name TEXT NOT NULL
			);
			CREATE UNIQUE INDEX idx_db_schema_unique_instance_db_name ON db_schema(instance, db_name);
			CREATE TABLE db_schema_child (
				instance TEXT NOT NULL,
				db_name TEXT NOT NULL,
				CONSTRAINT db_schema_child_fk FOREIGN KEY (instance, db_name) REFERENCES db_schema(instance, db_name)
			);

			CREATE TABLE db_group (
				id INT PRIMARY KEY,
				project TEXT NOT NULL,
				resource_id TEXT NOT NULL
			);
			CREATE UNIQUE INDEX idx_db_group_unique_project_resource_id ON db_group(project, resource_id);
			CREATE TABLE db_group_child (
				project TEXT NOT NULL,
				resource_id TEXT NOT NULL,
				CONSTRAINT db_group_child_fk FOREIGN KEY (project, resource_id) REFERENCES db_group(project, resource_id)
			);

			CREATE TABLE release (
				id INT PRIMARY KEY,
				project TEXT NOT NULL,
				train TEXT NOT NULL,
				iteration INT NOT NULL
			);
			CREATE UNIQUE INDEX idx_release_project_train_iteration ON release(project, train, iteration);
			CREATE TABLE release_child (
				project TEXT NOT NULL,
				train TEXT NOT NULL,
				iteration INT NOT NULL,
				CONSTRAINT release_child_fk FOREIGN KEY (project, train, iteration) REFERENCES release(project, train, iteration)
			);

			INSERT INTO project(id, resource_id) VALUES (1, 'projects/demo');
			INSERT INTO project_child(project) VALUES ('projects/demo');
			INSERT INTO instance(id, resource_id) VALUES (1, 'instances/demo');
			INSERT INTO instance_child(instance) VALUES ('instances/demo');
			INSERT INTO db(id, instance, name) VALUES (1, 'instances/demo', 'db1');
			INSERT INTO db_child(instance, db_name) VALUES ('instances/demo', 'db1');
			INSERT INTO setting(id, name) VALUES (1, 'bb.workspace');
			INSERT INTO setting_child(setting_name) VALUES ('bb.workspace');
			INSERT INTO policy(id, resource_type, resource, type) VALUES (1, 'PROJECT', 'projects/demo', 'IAM');
			INSERT INTO policy_child(resource_type, resource, policy_type) VALUES ('PROJECT', 'projects/demo', 'IAM');
			INSERT INTO idp(id, resource_id) VALUES (1, 'idps/demo');
			INSERT INTO idp_child(idp) VALUES ('idps/demo');
			INSERT INTO role(id, resource_id) VALUES (1, 'roles/demo');
			INSERT INTO role_child(role) VALUES ('roles/demo');
			INSERT INTO db_schema(id, instance, db_name) VALUES (1, 'instances/demo', 'db1');
			INSERT INTO db_schema_child(instance, db_name) VALUES ('instances/demo', 'db1');
			INSERT INTO db_group(id, project, resource_id) VALUES (1, 'projects/demo', 'groups/demo');
			INSERT INTO db_group_child(project, resource_id) VALUES ('projects/demo', 'groups/demo');
			INSERT INTO release(id, project, train, iteration) VALUES (1, 'projects/demo', 'trains/demo', 1);
			INSERT INTO release_child(project, train, iteration) VALUES ('projects/demo', 'trains/demo', 1);
		`
	_, err := db.ExecContext(ctx, setup)
	require.NoError(t, err)

	// Run the exact 3.16.2 statements for the affected tables.
	migrate := `
			ALTER TABLE project DROP CONSTRAINT IF EXISTS project_pkey;
			ALTER TABLE project DROP COLUMN IF EXISTS id;
			DO $$
			BEGIN
				IF to_regclass('idx_project_unique_resource_id') IS NOT NULL THEN
					ALTER TABLE project ADD CONSTRAINT project_pkey PRIMARY KEY USING INDEX idx_project_unique_resource_id;
				ELSE
					ALTER TABLE project ADD PRIMARY KEY (resource_id);
				END IF;
			END $$;

			ALTER TABLE instance DROP CONSTRAINT IF EXISTS instance_pkey;
			ALTER TABLE instance DROP COLUMN IF EXISTS id;
			DO $$
			BEGIN
				IF to_regclass('idx_instance_unique_resource_id') IS NOT NULL THEN
					ALTER TABLE instance ADD CONSTRAINT instance_pkey PRIMARY KEY USING INDEX idx_instance_unique_resource_id;
				ELSE
					ALTER TABLE instance ADD PRIMARY KEY (resource_id);
				END IF;
			END $$;

			ALTER TABLE db DROP CONSTRAINT IF EXISTS db_pkey;
			ALTER TABLE db DROP COLUMN IF EXISTS id;
			DO $$
			BEGIN
				IF to_regclass('idx_db_unique_instance_name') IS NOT NULL THEN
					ALTER TABLE db ADD CONSTRAINT db_pkey PRIMARY KEY USING INDEX idx_db_unique_instance_name;
				ELSE
					ALTER TABLE db ADD PRIMARY KEY (instance, name);
				END IF;
			END $$;

			ALTER TABLE setting DROP CONSTRAINT IF EXISTS setting_pkey;
			ALTER TABLE setting DROP COLUMN IF EXISTS id;
			DO $$
			BEGIN
				IF to_regclass('idx_setting_unique_name') IS NOT NULL THEN
					ALTER TABLE setting ADD CONSTRAINT setting_pkey PRIMARY KEY USING INDEX idx_setting_unique_name;
				ELSE
					ALTER TABLE setting ADD PRIMARY KEY (name);
				END IF;
			END $$;

			ALTER TABLE policy DROP CONSTRAINT IF EXISTS policy_pkey;
			ALTER TABLE policy DROP COLUMN IF EXISTS id;
			DO $$
			BEGIN
				IF to_regclass('idx_policy_unique_resource_type_resource_type') IS NOT NULL THEN
					ALTER TABLE policy ADD CONSTRAINT policy_pkey PRIMARY KEY USING INDEX idx_policy_unique_resource_type_resource_type;
				ELSE
					ALTER TABLE policy ADD PRIMARY KEY (resource_type, resource, type);
				END IF;
			END $$;

			ALTER TABLE idp DROP CONSTRAINT IF EXISTS idp_pkey;
			ALTER TABLE idp DROP COLUMN IF EXISTS id;
			DO $$
			BEGIN
				IF to_regclass('idx_idp_unique_resource_id') IS NOT NULL THEN
					ALTER TABLE idp ADD CONSTRAINT idp_pkey PRIMARY KEY USING INDEX idx_idp_unique_resource_id;
				ELSE
					ALTER TABLE idp ADD PRIMARY KEY (resource_id);
				END IF;
			END $$;

			ALTER TABLE role DROP CONSTRAINT IF EXISTS role_pkey;
			ALTER TABLE role DROP COLUMN IF EXISTS id;
			DO $$
			BEGIN
				IF to_regclass('idx_role_unique_resource_id') IS NOT NULL THEN
					ALTER TABLE role ADD CONSTRAINT role_pkey PRIMARY KEY USING INDEX idx_role_unique_resource_id;
				ELSE
					ALTER TABLE role ADD PRIMARY KEY (resource_id);
				END IF;
			END $$;

			ALTER TABLE db_schema DROP CONSTRAINT IF EXISTS db_schema_pkey;
			ALTER TABLE db_schema DROP COLUMN IF EXISTS id;
			DO $$
			BEGIN
				IF to_regclass('idx_db_schema_unique_instance_db_name') IS NOT NULL THEN
					ALTER TABLE db_schema ADD CONSTRAINT db_schema_pkey PRIMARY KEY USING INDEX idx_db_schema_unique_instance_db_name;
				ELSE
					ALTER TABLE db_schema ADD PRIMARY KEY (instance, db_name);
				END IF;
			END $$;

			ALTER TABLE db_group DROP CONSTRAINT IF EXISTS db_group_pkey;
			ALTER TABLE db_group DROP COLUMN IF EXISTS id;
			DO $$
			BEGIN
				IF to_regclass('idx_db_group_unique_project_resource_id') IS NOT NULL THEN
					ALTER TABLE db_group ADD CONSTRAINT db_group_pkey PRIMARY KEY USING INDEX idx_db_group_unique_project_resource_id;
				ELSE
					ALTER TABLE db_group ADD PRIMARY KEY (project, resource_id);
				END IF;
			END $$;

			ALTER TABLE release DROP CONSTRAINT IF EXISTS release_pkey;
			ALTER TABLE release DROP COLUMN IF EXISTS id;
			DO $$
			BEGIN
				IF to_regclass('idx_release_project_train_iteration') IS NOT NULL THEN
					ALTER TABLE release ADD CONSTRAINT release_pkey PRIMARY KEY USING INDEX idx_release_project_train_iteration;
				ELSE
					ALTER TABLE release ADD PRIMARY KEY (project, train, iteration);
				END IF;
			END $$;
		`
	_, err = db.ExecContext(ctx, migrate)
	require.NoError(t, err, "migration 3.16.2 must reuse dependent natural-key indexes")

	assertPromotedIndex := func(indexName, constraintName string) {
		t.Helper()

		var exists bool
		err := db.QueryRowContext(ctx, `SELECT to_regclass($1) IS NOT NULL`, constraintName).Scan(&exists)
		require.NoError(t, err)
		assert.True(t, exists)

		err = db.QueryRowContext(ctx, `SELECT to_regclass($1) IS NOT NULL`, indexName).Scan(&exists)
		require.NoError(t, err)
		assert.False(t, exists)
	}

	assertForeignKeyStillEnforced := func(insertSQL string, args ...any) {
		t.Helper()

		_, err := db.ExecContext(ctx, insertSQL, args...)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "foreign key")
	}

	var exists bool
	err = db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'project' AND column_name = 'id')`).Scan(&exists)
	require.NoError(t, err)
	assert.False(t, exists)

	assertPromotedIndex("idx_project_unique_resource_id", "project_pkey")
	assertForeignKeyStillEnforced(`INSERT INTO project_child(project) VALUES ('projects/missing')`)

	assertPromotedIndex("idx_instance_unique_resource_id", "instance_pkey")
	assertForeignKeyStillEnforced(`INSERT INTO instance_child(instance) VALUES ('instances/missing')`)

	assertPromotedIndex("idx_db_unique_instance_name", "db_pkey")
	assertForeignKeyStillEnforced(`INSERT INTO db_child(instance, db_name) VALUES ('instances/demo', 'missing')`)

	assertPromotedIndex("idx_setting_unique_name", "setting_pkey")
	assertForeignKeyStillEnforced(`INSERT INTO setting_child(setting_name) VALUES ('bb.missing')`)

	assertPromotedIndex("idx_policy_unique_resource_type_resource_type", "policy_pkey")
	assertForeignKeyStillEnforced(`INSERT INTO policy_child(resource_type, resource, policy_type) VALUES ('PROJECT', 'projects/demo', 'MISSING')`)

	assertPromotedIndex("idx_idp_unique_resource_id", "idp_pkey")
	assertForeignKeyStillEnforced(`INSERT INTO idp_child(idp) VALUES ('idps/missing')`)

	assertPromotedIndex("idx_role_unique_resource_id", "role_pkey")
	assertForeignKeyStillEnforced(`INSERT INTO role_child(role) VALUES ('roles/missing')`)

	assertPromotedIndex("idx_db_schema_unique_instance_db_name", "db_schema_pkey")
	assertForeignKeyStillEnforced(`INSERT INTO db_schema_child(instance, db_name) VALUES ('instances/demo', 'missing')`)

	assertPromotedIndex("idx_db_group_unique_project_resource_id", "db_group_pkey")
	assertForeignKeyStillEnforced(`INSERT INTO db_group_child(project, resource_id) VALUES ('projects/demo', 'groups/missing')`)

	assertPromotedIndex("idx_release_project_train_iteration", "release_pkey")
	assertForeignKeyStillEnforced(`INSERT INTO release_child(project, train, iteration) VALUES ('projects/demo', 'trains/demo', 2)`)
}
