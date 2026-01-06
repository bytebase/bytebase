package tests

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

func TestSyncSchema(t *testing.T) {
	databaseName := "sync_schema"
	newDatabaseName := "sync_schema_new"
	const (
		createSchema = `
			create schema schema_a;
			create table schema_a.table_t1(c1 int, c2 int, c3 int);
			create index idx_table_t1_c1_c2_c3 on schema_a.table_t1(c1, c2, c3);
			create sequence schema_a.sequence_s1;
			alter sequence schema_a.sequence_s1 owned by schema_a.table_t1.c1;
			alter table schema_a.table_t1 alter column c1 set default nextval('schema_a.sequence_s1'::regclass);
		`
		expectedDiff = `DROP TABLE IF EXISTS "schema_a"."table_t1";
DROP SEQUENCE IF EXISTS "schema_a"."sequence_s1";
DROP SCHEMA IF EXISTS "schema_a";
`
	)

	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	pgContainer, err := getPgContainer(ctx)
	defer func() {
		pgContainer.Close(ctx)
	}()
	a.NoError(err)

	pgDB := pgContainer.db
	err = pgDB.Ping()
	a.NoError(err)

	_, err = pgDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %v", databaseName))
	a.NoError(err)
	_, err = pgDB.Exec("CREATE USER bytebase WITH ENCRYPTED PASSWORD 'bytebase'")
	a.NoError(err)
	_, err = pgDB.Exec("ALTER USER bytebase WITH SUPERUSER")
	a.NoError(err)

	instanceResp, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance"),
		Instance: &v1pb.Instance{
			Title:       "pgTestSyncSchema",
			Engine:      v1pb.Engine_POSTGRES,
			Environment: stringPtr("environments/prod"),
			Activation:  true,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: pgContainer.host, Port: pgContainer.port, Username: "bytebase", Password: "bytebase", Id: "admin"}},
		},
	}))
	a.NoError(err)
	instance := instanceResp.Msg

	err = ctl.createDatabase(ctx, ctl.project, instance, nil /* environment */, databaseName, "bytebase")
	a.NoError(err)

	databaseResp, err := ctl.databaseServiceClient.GetDatabase(ctx, connect.NewRequest(&v1pb.GetDatabaseRequest{
		Name: fmt.Sprintf("%s/databases/%s", instance.Name, databaseName),
	}))
	a.NoError(err)
	database := databaseResp.Msg

	sheetResp, err := ctl.sheetServiceClient.CreateSheet(ctx, connect.NewRequest(&v1pb.CreateSheetRequest{
		Parent: ctl.project.Name,
		Sheet: &v1pb.Sheet{
			Content: []byte(createSchema),
		},
	}))
	a.NoError(err)
	sheet := sheetResp.Msg

	// Create an issue that updates database schema.
	err = ctl.changeDatabase(ctx, ctl.project, database, sheet, false)
	a.NoError(err)

	resp, err := ctl.databaseServiceClient.ListChangelogs(ctx, connect.NewRequest(&v1pb.ListChangelogsRequest{
		Parent: database.Name,
		View:   v1pb.ChangelogView_CHANGELOG_VIEW_FULL,
	}))
	a.NoError(err)
	changelogs := resp.Msg.Changelogs
	// Expect 2 changelogs: baseline (auto-created on first migration) + migration
	a.Equal(2, len(changelogs))
	// First changelog should be the migration (most recent)
	a.Equal(v1pb.Changelog_MIGRATE, changelogs[0].Type)
	// Second changelog should be the baseline
	a.Equal(v1pb.Changelog_BASELINE, changelogs[1].Type)

	err = ctl.createDatabase(ctx, ctl.project, instance, nil /* environment */, newDatabaseName, "bytebase")
	a.NoError(err)

	newDatabaseResp, err := ctl.databaseServiceClient.GetDatabase(ctx, connect.NewRequest(&v1pb.GetDatabaseRequest{
		Name: fmt.Sprintf("%s/databases/%s", instance.Name, newDatabaseName),
	}))
	a.NoError(err)
	newDatabase := newDatabaseResp.Msg

	newDatabaseSchemaResp, err := ctl.databaseServiceClient.GetDatabaseSchema(ctx, connect.NewRequest(&v1pb.GetDatabaseSchemaRequest{
		Name: fmt.Sprintf("%s/schema", newDatabase.Name),
	}))
	a.NoError(err)
	newDatabaseSchema := newDatabaseSchemaResp.Msg

	diff, err := ctl.getSchemaDiff(ctx, &v1pb.DiffSchemaRequest{
		Name: database.Name,
		Target: &v1pb.DiffSchemaRequest_Schema{
			Schema: newDatabaseSchema.Schema,
		},
	})

	a.NoError(err)
	a.Equal(expectedDiff, diff)
}

// TestSyncSchemaWithTempSchema tests that schema sync works correctly even when pg_temp schemas exist
// This reproduces the issue from SUP-3 where users get permission errors on pg_temp schemas
func TestSyncSchemaWithTempSchema(t *testing.T) {
	databaseName := "sync_schema_temp"

	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	pgContainer, err := getPgContainer(ctx)
	defer func() {
		pgContainer.Close(ctx)
	}()
	a.NoError(err)

	pgDB := pgContainer.db
	err = pgDB.Ping()
	a.NoError(err)

	// Create database and user
	_, err = pgDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %v", databaseName))
	a.NoError(err)
	_, err = pgDB.Exec(fmt.Sprintf("CREATE DATABASE %v", databaseName))
	a.NoError(err)

	// Create a limited user (not superuser) to simulate real-world scenario
	_, err = pgDB.Exec("DROP USER IF EXISTS testuser")
	a.NoError(err)
	_, err = pgDB.Exec("CREATE USER testuser WITH ENCRYPTED PASSWORD 'testpass'")
	a.NoError(err)
	_, err = pgDB.Exec(fmt.Sprintf("GRANT ALL PRIVILEGES ON DATABASE %v TO testuser", databaseName))
	a.NoError(err)

	// Connect to the test database to create temp tables and schemas
	testDB, err := sql.Open("pgx", fmt.Sprintf("host=%s port=%s user=postgres password=root-password dbname=%s sslmode=disable", pgContainer.host, pgContainer.port, databaseName))
	a.NoError(err)
	defer testDB.Close()

	// Create some regular schemas and tables
	_, err = testDB.Exec(`
		CREATE SCHEMA app;
		CREATE TABLE app.users (id INT PRIMARY KEY, name TEXT);
		GRANT USAGE ON SCHEMA app TO testuser;
		GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA app TO testuser;
	`)
	a.NoError(err)

	// Create temporary tables which will create pg_temp schemas
	// These temp tables will be in pg_temp_N schemas that testuser doesn't have permission to access
	_, err = testDB.Exec(`
		CREATE TEMP TABLE temp_table1 (id INT);
		CREATE TEMP TABLE temp_table2 (data TEXT);
	`)
	a.NoError(err)

	// Verify that pg_temp schemas exist
	var tempSchemaCount int
	err = testDB.QueryRow(`
		SELECT COUNT(*)
		FROM pg_namespace
		WHERE nspname LIKE 'pg_temp%'
	`).Scan(&tempSchemaCount)
	a.NoError(err)
	a.Greater(tempSchemaCount, 0, "pg_temp schemas should exist")

	// Now try to sync schema using the limited testuser account
	// This should NOT fail even though testuser doesn't have permissions on pg_temp schemas
	instanceResp, err := ctl.instanceServiceClient.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance"),
		Instance: &v1pb.Instance{
			Title:       "pgTestSyncSchemaWithTemp",
			Engine:      v1pb.Engine_POSTGRES,
			Environment: stringPtr("environments/prod"),
			Activation:  true,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: pgContainer.host, Port: pgContainer.port, Username: "testuser", Password: "testpass", Id: "admin"}},
		},
	}))
	a.NoError(err)
	instance := instanceResp.Msg

	// Sync the database schema - this should succeed and ignore pg_temp schemas
	_, err = ctl.databaseServiceClient.SyncDatabase(ctx, connect.NewRequest(&v1pb.SyncDatabaseRequest{
		Name: fmt.Sprintf("%s/databases/%s", instance.Name, databaseName),
	}))
	a.NoError(err, "Schema sync should succeed even with pg_temp schemas present")

	// Verify that we can get the database schema metadata without errors
	schemaResp, err := ctl.databaseServiceClient.GetDatabaseSchema(ctx, connect.NewRequest(&v1pb.GetDatabaseSchemaRequest{
		Name: fmt.Sprintf("%s/databases/%s/schema", instance.Name, databaseName),
	}))
	a.NoError(err)

	// Verify that the schema string doesn't contain pg_temp or pg_toast references
	a.NotContains(schemaResp.Msg.Schema, "pg_temp", "pg_temp schemas should be filtered out")
	a.NotContains(schemaResp.Msg.Schema, "pg_toast", "pg_toast schemas should be filtered out")

	// Verify that our regular schema is present in the result
	a.Contains(schemaResp.Msg.Schema, "app", "app schema should be present in synced metadata")
	a.Contains(schemaResp.Msg.Schema, "users", "users table should be present in synced metadata")
}
