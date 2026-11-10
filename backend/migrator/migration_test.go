package migrator_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	"github.com/bytebase/bytebase/backend/migrator"
)

func TestMain(m *testing.M) { testcontainer.Main(m) }

func TestMigration3_24_1_MSSQLVerifyTLSOff(t *testing.T) {
	ctx := context.Background()
	_, db := testcontainer.NewPgDatabase(t)
	_, err := db.ExecContext(ctx, `
		CREATE TABLE instance (
			resource_id TEXT PRIMARY KEY,
			metadata JSONB NOT NULL DEFAULT '{}'
		);
		INSERT INTO instance (resource_id, metadata) VALUES
			('mssql-verify', '{"engine":"MSSQL","dataSources":[{"id":"admin","useSsl":true,"verifyTlsCertificate":true,"sslCa":"ca"},{"id":"ro","type":"READ_ONLY","useSsl":true,"verifyTlsCertificate":true}]}'),
			('mssql-verify-without-tls', '{"engine":"MSSQL","dataSources":[{"id":"admin","useSsl":true,"verifyTlsCertificate":true},{"id":"ro","type":"READ_ONLY","verifyTlsCertificate":true}]}'),
			('mssql-tls', '{"engine":"MSSQL","dataSources":[{"id":"admin","useSsl":true}]}'),
			('mssql-no-data-sources', '{"engine":"MSSQL","dataSources":[]}'),
			('postgres-verify', '{"engine":"POSTGRES","dataSources":[{"id":"admin","useSsl":true,"verifyTlsCertificate":true}]}');
	`)
	require.NoError(t, err)

	statement, err := migrator.MigrationFS.ReadFile("migration/3.24/0001##mssql_verify_tls_off.sql")
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, string(statement))
	require.NoError(t, err)

	for resourceID, want := range map[string]string{
		"mssql-verify":             `{"engine":"MSSQL","dataSources":[{"id":"admin","useSsl":true,"sslCa":"ca"},{"id":"ro","type":"READ_ONLY","useSsl":true}]}`,
		"mssql-verify-without-tls": `{"engine":"MSSQL","dataSources":[{"id":"admin","useSsl":true},{"id":"ro","type":"READ_ONLY","verifyTlsCertificate":true}]}`,
		"mssql-tls":                `{"engine":"MSSQL","dataSources":[{"id":"admin","useSsl":true}]}`,
		"mssql-no-data-sources":    `{"engine":"MSSQL","dataSources":[]}`,
		"postgres-verify":          `{"engine":"POSTGRES","dataSources":[{"id":"admin","useSsl":true,"verifyTlsCertificate":true}]}`,
	} {
		var got string
		require.NoError(t, db.QueryRowContext(ctx, `SELECT metadata::text FROM instance WHERE resource_id = $1`, resourceID).Scan(&got))
		require.JSONEq(t, want, got, resourceID)
	}
}

func TestMigration3_24_2_OracleTLSOff(t *testing.T) {
	ctx := context.Background()
	_, db := testcontainer.NewPgDatabase(t)
	_, err := db.ExecContext(ctx, `
		CREATE TABLE instance (
			resource_id TEXT PRIMARY KEY,
			metadata JSONB NOT NULL DEFAULT '{}'
		);
		INSERT INTO instance (resource_id, metadata) VALUES
			('oracle-tls', '{"engine":"ORACLE","dataSources":[{"id":"admin","useSsl":true,"verifyTlsCertificate":true,"obfuscatedSslCa":"ca","obfuscatedSslCert":"cert","obfuscatedSslKey":"key"},{"id":"ro","type":"READ_ONLY","useSsl":true,"obfuscatedSslCaPath":"ca-path","obfuscatedSslCertPath":"cert-path","obfuscatedSslKeyPath":"key-path","extraConnectionParameters":{"SSL":"TRUE","WALLET":"/opt/oracle/wallet"}}]}'),
			('oracle-tls-on-one', '{"engine":"ORACLE","dataSources":[{"id":"admin","useSsl":true},{"id":"ro","type":"READ_ONLY","verifyTlsCertificate":true,"obfuscatedSslCa":"ca"}]}'),
			('oracle-tls-off-with-ca', '{"engine":"ORACLE","dataSources":[{"id":"admin","verifyTlsCertificate":true,"obfuscatedSslCa":"ca"}]}'),
			('oracle-no-tls', '{"engine":"ORACLE","dataSources":[{"id":"admin","verifyTlsCertificate":true}]}'),
			('oracle-no-data-sources', '{"engine":"ORACLE","dataSources":[]}'),
			('oracle-without-data-sources', '{"engine":"ORACLE"}'),
			('oracle-null-data-sources', '{"engine":"ORACLE","dataSources":null}'),
			('postgres-tls', '{"engine":"POSTGRES","dataSources":[{"id":"admin","useSsl":true,"obfuscatedSslCa":"ca"}]}'),
			('mssql-tls', '{"engine":"MSSQL","dataSources":[{"id":"admin","useSsl":true}]}');
	`)
	require.NoError(t, err)

	statement, err := migrator.MigrationFS.ReadFile("migration/3.24/0002##oracle_tls_off.sql")
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, string(statement))
	require.NoError(t, err)

	for resourceID, want := range map[string]string{
		"oracle-tls":                  `{"engine":"ORACLE","dataSources":[{"id":"admin","verifyTlsCertificate":true},{"id":"ro","type":"READ_ONLY","extraConnectionParameters":{"SSL":"TRUE","WALLET":"/opt/oracle/wallet"}}]}`,
		"oracle-tls-on-one":           `{"engine":"ORACLE","dataSources":[{"id":"admin"},{"id":"ro","type":"READ_ONLY","verifyTlsCertificate":true}]}`,
		"oracle-tls-off-with-ca":      `{"engine":"ORACLE","dataSources":[{"id":"admin","verifyTlsCertificate":true}]}`,
		"oracle-no-tls":               `{"engine":"ORACLE","dataSources":[{"id":"admin","verifyTlsCertificate":true}]}`,
		"oracle-no-data-sources":      `{"engine":"ORACLE","dataSources":[]}`,
		"oracle-without-data-sources": `{"engine":"ORACLE"}`,
		"oracle-null-data-sources":    `{"engine":"ORACLE","dataSources":null}`,
		"postgres-tls":                `{"engine":"POSTGRES","dataSources":[{"id":"admin","useSsl":true,"obfuscatedSslCa":"ca"}]}`,
		"mssql-tls":                   `{"engine":"MSSQL","dataSources":[{"id":"admin","useSsl":true}]}`,
	} {
		var got string
		require.NoError(t, db.QueryRowContext(ctx, `SELECT metadata::text FROM instance WHERE resource_id = $1`, resourceID).Scan(&got))
		require.JSONEq(t, want, got, resourceID)
	}
}

func TestMigration3_25_1_DBSchemaSyncedAtBackfill(t *testing.T) {
	ctx := context.Background()
	_, db := testcontainer.NewPgDatabase(t)
	_, err := db.ExecContext(ctx, `
		CREATE TABLE db (
			instance TEXT NOT NULL,
			name TEXT NOT NULL,
			metadata JSONB NOT NULL DEFAULT '{}',
			PRIMARY KEY (instance, name)
		);
		CREATE TABLE db_schema (
			instance TEXT NOT NULL,
			db_name TEXT NOT NULL,
			raw_dump TEXT NOT NULL DEFAULT '',
			PRIMARY KEY (instance, db_name)
		);
		INSERT INTO db (instance, name, metadata) VALUES
			-- protojson writes a Timestamp with nanosecond precision as nine
			-- fractional digits, more than timestamptz keeps.
			('prod', 'synced', '{"lastSyncTime":"2026-10-28T04:05:06.123456789Z","syncStatus":"SYNC_STATUS_OK"}'),
			('prod', 'whole-second', '{"lastSyncTime":"2026-10-28T04:05:06Z"}'),
			('prod', 'never-synced', '{"syncStatus":"SYNC_STATUS_OK"}'),
			-- A clock that ran ahead, so the recorded time is in the future.
			('prod', 'clock-ahead', '{"lastSyncTime":"2099-01-01T00:00:00Z"}'),
			-- Same database name in another instance: db_schema is keyed by
			-- (instance, db_name) and the backfill joins on the whole key.
			('staging', 'synced', '{"lastSyncTime":"2026-10-20T01:02:03Z"}');
		INSERT INTO db_schema (instance, db_name, raw_dump) VALUES
			('prod', 'synced', 'prod dump'),
			('prod', 'whole-second', 'whole second dump'),
			('prod', 'never-synced', 'unstamped dump'),
			('prod', 'clock-ahead', 'clock ahead dump'),
			('staging', 'synced', 'staging dump');
	`)
	require.NoError(t, err)

	statement, err := migrator.MigrationFS.ReadFile("migration/3.25/0001##db_schema_synced_at.sql")
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, string(statement))
	require.NoError(t, err)

	for _, tc := range []struct {
		instance     string
		databaseName string
		want         string
		wantClamped  bool
	}{
		{instance: "prod", databaseName: "synced", want: "2026-10-28T04:05:06.123457Z"},
		{instance: "prod", databaseName: "whole-second", want: "2026-10-28T04:05:06Z"},
		{instance: "prod", databaseName: "never-synced"},
		{instance: "prod", databaseName: "clock-ahead", wantClamped: true},
		{instance: "staging", databaseName: "synced", want: "2026-10-20T01:02:03Z"},
	} {
		var syncedAt sql.NullTime
		require.NoError(t, db.QueryRowContext(ctx,
			`SELECT synced_at FROM db_schema WHERE instance = $1 AND db_name = $2`,
			tc.instance, tc.databaseName).Scan(&syncedAt))
		switch {
		case tc.wantClamped:
			// A stamp in the future would refuse every sync until wall clock passed it.
			require.True(t, syncedAt.Valid, "%s/%s", tc.instance, tc.databaseName)
			require.WithinDuration(t, time.Now(), syncedAt.Time, time.Hour, "%s/%s", tc.instance, tc.databaseName)
		case tc.want == "":
			require.False(t, syncedAt.Valid, "%s/%s", tc.instance, tc.databaseName)
		default:
			require.True(t, syncedAt.Valid, "%s/%s", tc.instance, tc.databaseName)
			require.Equal(t, tc.want, syncedAt.Time.UTC().Format(time.RFC3339Nano), "%s/%s", tc.instance, tc.databaseName)
		}
	}
}

func TestMigration3_25_2_AIReviewResultTitle(t *testing.T) {
	ctx := context.Background()
	_, db := testcontainer.NewPgDatabase(t)
	_, err := db.ExecContext(ctx, `
		CREATE TABLE issue_comment (
			resource_id TEXT PRIMARY KEY,
			payload JSONB NOT NULL DEFAULT '{}'
		);
		INSERT INTO issue_comment (resource_id, payload) VALUES
			('one-database', '{"comment":"Do not drop a table that holds data\n\nThe rental table holds 10 rows.\n\nRule: Data risk.\n\nSuggested fix: Keep the table.","reviewMetadata":{"runType":"AI","priority":"P0"}}'),
			('databases', '{"comment":"Create the index concurrently\n\nThe table is large.\n- **P1** on a, b\n\nRule: Locks.\n\nSuggested fix: Add CONCURRENTLY.","reviewMetadata":{"runType":"AI"}}'),
			('title-only', '{"comment":"  Only a title  ","reviewMetadata":{"runType":"AI"}}'),
			('migrated', '{"comment":"Evidence\n\nRule: R.","reviewMetadata":{"runType":"AI","title":"Kept title"}}'),
			('rule', '{"comment":"Drops table rental.\nMore.","reviewMetadata":{"runType":"RULE","ruleType":"TABLE_DROP"}}'),
			('person', '{"comment":"Looks good\n\nShip it."}');
	`)
	require.NoError(t, err)

	statement, err := migrator.MigrationFS.ReadFile("migration/3.25/0002##ai_review_result_title.sql")
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, string(statement))
	require.NoError(t, err)

	for _, tc := range []struct {
		id      string
		title   sql.NullString
		comment string
	}{
		{id: "one-database", title: sql.NullString{String: "Do not drop a table that holds data", Valid: true}, comment: "The rental table holds 10 rows.\n\nRule: Data risk.\n\nSuggested fix: Keep the table."},
		{id: "databases", title: sql.NullString{String: "Create the index concurrently", Valid: true}, comment: "The table is large.\n- **P1** on a, b\n\nRule: Locks.\n\nSuggested fix: Add CONCURRENTLY."},
		{id: "title-only", title: sql.NullString{String: "Only a title", Valid: true}, comment: ""},
		{id: "migrated", title: sql.NullString{String: "Kept title", Valid: true}, comment: "Evidence\n\nRule: R."},
		{id: "rule", comment: "Drops table rental.\nMore."},
		{id: "person", comment: "Looks good\n\nShip it."},
	} {
		var title sql.NullString
		var comment string
		require.NoError(t, db.QueryRowContext(ctx,
			`SELECT payload->'reviewMetadata'->>'title', payload->>'comment' FROM issue_comment WHERE resource_id = $1`,
			tc.id).Scan(&title, &comment))
		require.Equal(t, tc.title, title, tc.id)
		require.Equal(t, tc.comment, comment, tc.id)
	}
}

func TestMigration3_25_0_SQLExportPermission(t *testing.T) {
	ctx := context.Background()
	_, db := testcontainer.NewPgDatabase(t)
	createMigration3_25_0Tables(ctx, t, db)
	_, err := db.ExecContext(ctx, `
		INSERT INTO role (resource_id, workspace, permissions) VALUES
			('export-select', 'ws-export', '{"permissions":["bb.databases.get","bb.sql.select"]}'),
			('export-no-select', 'ws-export', '{"permissions":["bb.databases.get","bb.sql.info"]}'),
			('export-already', 'ws-export', '{"permissions":["bb.sql.select","bb.sql.export"]}'),
			('approval-select', 'ws-approval', '{"permissions":["bb.databases.get","bb.sql.select"]}'),
			('approval-no-select', 'ws-approval', '{"permissions":["bb.databases.get"]}'),
			('unenforced-select', 'ws-unenforced', '{"permissions":["bb.sql.select"]}'),
			('unchanged-no-select', 'ws-unchanged', '{"permissions":["bb.databases.get"]}');
	`)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `
		INSERT INTO policy (workspace, resource_type, resource, type, enforce, payload) VALUES
			('ws-export', 'WORKSPACE', 'workspaces/ws-export', 'IAM', true, $1),
			('ws-export', 'PROJECT', 'projects/export-project', 'IAM', true, $2),
			('ws-export', 'WORKSPACE', 'workspaces/ws-export', 'QUERY_DATA', true, '{"maximumResultRows":100,"disableCopyData":true}'),
			('ws-export', 'PROJECT', 'projects/export-project', 'QUERY_DATA', true, '{"maximumResultRows":50,"disableExport":true}'),
			('ws-approval', 'WORKSPACE', 'workspaces/ws-approval', 'IAM', true, $3),
			('ws-approval', 'PROJECT', 'projects/approval-project', 'IAM', true, $4),
			('ws-approval', 'WORKSPACE', 'workspaces/ws-approval', 'QUERY_DATA', true, '{"disableExport":true,"maximumResultRows":10,"allowAdminDataSource":true}'),
			('ws-unenforced', 'PROJECT', 'projects/unenforced-project', 'IAM', true, $5),
			('ws-unenforced', 'WORKSPACE', 'workspaces/ws-unenforced', 'QUERY_DATA', false, '{"disableExport":true,"disableCopyData":true}'),
			('ws-unchanged', 'WORKSPACE', 'workspaces/ws-unchanged', 'IAM', true, $6),
			('ws-unchanged', 'WORKSPACE', 'workspaces/ws-unchanged', 'QUERY_DATA', true, '{"maximumResultRows":5}');
	`,
		migration3_25_0WorkspaceIAM,
		migration3_25_0ProjectIAM,
		migration3_25_0WorkspaceIAM,
		migration3_25_0ProjectIAM,
		`{"bindings":[{"role":"roles/sqlEditorReadUser","members":["groups/analysts@example.com"]}]}`,
		`{"bindings":[{"role":"roles/workspaceMember","members":["allUsers"]}]}`,
	)
	require.NoError(t, err)

	rolePermissions := func() map[string]string {
		rows, err := db.QueryContext(ctx, `SELECT resource_id, permissions::text FROM role`)
		require.NoError(t, err)
		defer rows.Close()
		permissions := map[string]string{}
		for rows.Next() {
			var role, text string
			require.NoError(t, rows.Scan(&role, &text))
			permissions[role] = text
		}
		require.NoError(t, rows.Err())
		return permissions
	}
	rolesBefore := rolePermissions()
	require.Len(t, rolesBefore, 7)

	require.NoError(t, runMigration3_25_0SQL(ctx, db))

	require.Equal(t, rolesBefore, rolePermissions(), "the migration does not change custom roles")

	type policyKey struct{ workspace, resource, policyType string }
	type policyWant struct {
		enforce bool
		payload string
	}
	for key, want := range map[policyKey]policyWant{
		{"ws-export", "workspaces/ws-export", "IAM"}: {true, `{"bindings":[
			{"role":"roles/workspaceMember","members":["allUsers"]},
			{"role":"roles/sqlEditorUser","members":["users/a@example.com"],"condition":{"title":"prod app","expression":"resource.database in [\"instances/prod/databases/app\"] && request.time < timestamp(\"2027-01-01T00:00:00Z\")"}},
			{"role":"roles/sqlEditorReadUser","members":["groups/analysts@example.com"]},
			{"role":"roles/sqlEditorExporter","members":["users/a@example.com"],"condition":{"title":"prod app","expression":"resource.database in [\"instances/prod/databases/app\"] && request.time < timestamp(\"2027-01-01T00:00:00Z\")"}},
			{"role":"roles/sqlEditorExporter","members":["groups/analysts@example.com"]}]}`},
		{"ws-export", "projects/export-project", "IAM"}: {true, `{"bindings":[
			{"role":"roles/projectDeveloper","members":["users/b@example.com"]},
			{"role":"roles/sqlEditorReadUser","members":["groups/analysts@example.com"]},
			{"role":"roles/sqlEditorUser","members":["users/b@example.com","allUsers"],"condition":{"expression":"resource.database == \"instances/prod/databases/app\" && resource.schema_name == \"public\" && resource.table_name in [\"orders\"]"}},
			{"role":"roles/sqlEditorExporter","members":["groups/analysts@example.com"]},
			{"role":"roles/sqlEditorExporter","members":["users/b@example.com","allUsers"],"condition":{"expression":"resource.database == \"instances/prod/databases/app\" && resource.schema_name == \"public\" && resource.table_name in [\"orders\"]"}}]}`},
		{"ws-export", "workspaces/ws-export", "QUERY_DATA"}:         {true, `{"maximumResultRows":100,"disableCopyData":true}`},
		{"ws-export", "projects/export-project", "QUERY_DATA"}:      {true, `{"maximumResultRows":50}`},
		{"ws-approval", "workspaces/ws-approval", "IAM"}:            {true, migration3_25_0WorkspaceIAM},
		{"ws-approval", "projects/approval-project", "IAM"}:         {true, migration3_25_0ProjectIAM},
		{"ws-approval", "workspaces/ws-approval", "QUERY_DATA"}:     {true, `{"maximumResultRows":10,"allowAdminDataSource":true,"disableDownload":true}`},
		{"ws-unenforced", "projects/unenforced-project", "IAM"}:     {true, `{"bindings":[{"role":"roles/sqlEditorReadUser","members":["groups/analysts@example.com"]}]}`},
		{"ws-unenforced", "workspaces/ws-unenforced", "QUERY_DATA"}: {false, `{"disableCopyData":true,"disableDownload":true}`},
		{"ws-unchanged", "workspaces/ws-unchanged", "IAM"}:          {true, `{"bindings":[{"role":"roles/workspaceMember","members":["allUsers"]}]}`},
		{"ws-unchanged", "workspaces/ws-unchanged", "QUERY_DATA"}:   {true, `{"maximumResultRows":5}`},
	} {
		var enforce bool
		var payload string
		require.NoError(t, db.QueryRowContext(ctx,
			`SELECT enforce, payload::text FROM policy WHERE workspace = $1 AND resource = $2 AND type = $3`,
			key.workspace, key.resource, key.policyType,
		).Scan(&enforce, &payload))
		require.Equal(t, want.enforce, enforce, key)
		require.JSONEq(t, want.payload, payload, key)
	}
}

func TestMigration3_25_0_SQLEditorExporterIDCollision(t *testing.T) {
	ctx := context.Background()
	_, db := testcontainer.NewPgDatabase(t)
	createMigration3_25_0Tables(ctx, t, db)
	_, err := db.ExecContext(ctx, `
		INSERT INTO role (resource_id, workspace, permissions) VALUES
			('sqlEditorExporter', 'ws-export', '{"permissions":["bb.sql.select"]}');
	`)
	require.NoError(t, err)

	err = runMigration3_25_0SQL(ctx, db)
	require.ErrorContains(t, err, `custom role "sqlEditorExporter" in workspace "ws-export" conflicts with a new built-in role`)

	var got string
	require.NoError(t, db.QueryRowContext(ctx, `SELECT permissions::text FROM role WHERE resource_id = 'sqlEditorExporter'`).Scan(&got))
	require.JSONEq(t, `{"permissions":["bb.sql.select"]}`, got)
}

const (
	migration3_25_0WorkspaceIAM = `{"bindings":[
		{"role":"roles/workspaceMember","members":["allUsers"]},
		{"role":"roles/sqlEditorUser","members":["users/a@example.com"],"condition":{"title":"prod app","expression":"resource.database in [\"instances/prod/databases/app\"] && request.time < timestamp(\"2027-01-01T00:00:00Z\")"}},
		{"role":"roles/sqlEditorReadUser","members":["groups/analysts@example.com"]}]}`
	migration3_25_0ProjectIAM = `{"bindings":[
		{"role":"roles/projectDeveloper","members":["users/b@example.com"]},
		{"role":"roles/sqlEditorReadUser","members":["groups/analysts@example.com"]},
		{"role":"roles/sqlEditorUser","members":["users/b@example.com","allUsers"],"condition":{"expression":"resource.database == \"instances/prod/databases/app\" && resource.schema_name == \"public\" && resource.table_name in [\"orders\"]"}}]}`
)

func createMigration3_25_0Tables(ctx context.Context, t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.ExecContext(ctx, `
		CREATE TABLE role (
			resource_id text NOT NULL PRIMARY KEY,
			workspace text NOT NULL,
			permissions jsonb NOT NULL DEFAULT '{}'
		);
		CREATE TABLE policy (
			enforce boolean NOT NULL DEFAULT TRUE,
			workspace text NOT NULL,
			resource_type text NOT NULL,
			resource text NOT NULL,
			type text NOT NULL,
			payload jsonb NOT NULL DEFAULT '{}',
			PRIMARY KEY (workspace, resource_type, resource, type)
		);
	`)
	require.NoError(t, err)
}

// runMigration3_25_0SQL runs the file in one transaction, as the migrator does.
func runMigration3_25_0SQL(ctx context.Context, db *sql.DB) error {
	statement, err := migrator.MigrationFS.ReadFile("migration/3.25/0000##sql_export_permission.sql")
	if err != nil {
		return err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, string(statement)); err != nil {
		return err
	}
	return tx.Commit()
}
