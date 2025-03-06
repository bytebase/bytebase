package tests

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
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
		expectedDiff = `DROP INDEX "schema_a"."idx_table_t1_c1_c2_c3";

ALTER SEQUENCE "schema_a"."sequence_s1"
    OWNED BY NONE;

DROP TABLE "schema_a"."table_t1";

DROP SEQUENCE "schema_a"."sequence_s1";

DROP SCHEMA "schema_a";

`
	)

	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	dataDir := t.TempDir()
	ctx, err := ctl.StartServerWithExternalPg(ctx, &config{
		dataDir: dataDir,
	})
	a.NoError(err)
	defer ctl.Close(ctx)

	pgContainer, err := getPgContainer(ctx)
	a.NoError(err)

	defer func() {
		pgContainer.db.Close()
		err := pgContainer.container.Terminate(ctx)
		a.NoError(err)
	}()

	pgDB := pgContainer.db
	err = pgDB.Ping()
	a.NoError(err)

	_, err = pgDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %v", databaseName))
	a.NoError(err)
	_, err = pgDB.Exec("CREATE USER bytebase WITH ENCRYPTED PASSWORD 'bytebase'")
	a.NoError(err)
	_, err = pgDB.Exec("ALTER USER bytebase WITH SUPERUSER")
	a.NoError(err)

	instance, err := ctl.instanceServiceClient.CreateInstance(ctx, &v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance", 10),
		Instance: &v1pb.Instance{
			Title:       "pgTestSyncSchema",
			Engine:      v1pb.Engine_POSTGRES,
			Environment: "environments/prod",
			Activation:  true,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: pgContainer.host, Port: pgContainer.port, Username: "bytebase", Password: "bytebase", Id: "admin"}},
		},
	})
	a.NoError(err)

	err = ctl.createDatabaseV2(ctx, ctl.project, instance, nil /* environment */, databaseName, "bytebase")
	a.NoError(err)

	database, err := ctl.databaseServiceClient.GetDatabase(ctx, &v1pb.GetDatabaseRequest{
		Name: fmt.Sprintf("%s/databases/%s", instance.Name, databaseName),
	})
	a.NoError(err)

	sheet, err := ctl.sheetServiceClient.CreateSheet(ctx, &v1pb.CreateSheetRequest{
		Parent: ctl.project.Name,
		Sheet: &v1pb.Sheet{
			Title:   "create schema",
			Content: []byte(createSchema),
		},
	})
	a.NoError(err)

	// Create an issue that updates database schema.
	err = ctl.changeDatabase(ctx, ctl.project, database, sheet, v1pb.Plan_ChangeDatabaseConfig_MIGRATE)
	a.NoError(err)

	resp, err := ctl.databaseServiceClient.ListChangelogs(ctx, &v1pb.ListChangelogsRequest{
		Parent: database.Name,
		View:   v1pb.ChangelogView_CHANGELOG_VIEW_FULL,
	})
	a.NoError(err)
	changelogs := resp.Changelogs
	a.Equal(1, len(changelogs))

	err = ctl.createDatabaseV2(ctx, ctl.project, instance, nil /* environment */, newDatabaseName, "bytebase")
	a.NoError(err)

	newDatabase, err := ctl.databaseServiceClient.GetDatabase(ctx, &v1pb.GetDatabaseRequest{
		Name: fmt.Sprintf("%s/databases/%s", instance.Name, newDatabaseName),
	})
	a.NoError(err)

	newDatabaseSchema, err := ctl.databaseServiceClient.GetDatabaseSchema(ctx, &v1pb.GetDatabaseSchemaRequest{
		Name: fmt.Sprintf("%s/schema", newDatabase.Name),
	})
	a.NoError(err)

	diff, err := ctl.getSchemaDiff(ctx, &v1pb.DiffSchemaRequest{
		Name: database.Name,
		Target: &v1pb.DiffSchemaRequest_Schema{
			Schema: newDatabaseSchema.Schema,
		},
	})

	a.NoError(err)
	a.Equal(expectedDiff, diff)
}
