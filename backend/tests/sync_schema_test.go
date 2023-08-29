package tests

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	parser "github.com/bytebase/bytebase/backend/plugin/parser/sql"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"

	"github.com/bytebase/bytebase/backend/resources/postgres"
	"github.com/bytebase/bytebase/backend/tests/fake"
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
		dataDir:                   dataDir,
		vcsProviderCreator:        fake.NewGitLab,
		developmentUseV2Scheduler: true,
	})
	a.NoError(err)
	defer ctl.Close(ctx)

	// Create a PostgreSQL instance.
	pgPort := getTestPort()
	stopInstance := postgres.SetupTestInstance(t, pgPort, resourceDir)
	defer stopInstance()

	pgDB, err := sql.Open("pgx", fmt.Sprintf("host=/tmp port=%d user=root database=postgres", pgPort))
	a.NoError(err)
	defer pgDB.Close()

	err = pgDB.Ping()
	a.NoError(err)

	_, err = pgDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %v", databaseName))
	a.NoError(err)

	_, err = pgDB.Exec("CREATE USER bytebase WITH ENCRYPTED PASSWORD 'bytebase'")
	a.NoError(err)

	_, err = pgDB.Exec("ALTER USER bytebase WITH SUPERUSER")
	a.NoError(err)

	// Create a project.
	project, err := ctl.createProject(ctx)
	a.NoError(err)
	projectUID, err := strconv.Atoi(project.Uid)
	a.NoError(err)

	prodEnvironment, err := ctl.getEnvironment(ctx, "prod")
	a.NoError(err)

	instance, err := ctl.instanceServiceClient.CreateInstance(ctx, &v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance", 10),
		Instance: &v1pb.Instance{
			Title:       "pgTestSyncSchema",
			Engine:      v1pb.Engine_POSTGRES,
			Environment: prodEnvironment.Name,
			Activation:  true,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: "/tmp", Port: strconv.Itoa(pgPort), Username: "bytebase", Password: "bytebase"}},
		},
	})
	a.NoError(err)

	err = ctl.createDatabaseV2(ctx, project, instance, databaseName, "bytebase", nil)
	a.NoError(err)

	database, err := ctl.databaseServiceClient.GetDatabase(ctx, &v1pb.GetDatabaseRequest{
		Name: fmt.Sprintf("%s/databases/%s", instance.Name, databaseName),
	})
	a.NoError(err)

	sheet, err := ctl.sheetServiceClient.CreateSheet(ctx, &v1pb.CreateSheetRequest{
		Parent: project.Name,
		Sheet: &v1pb.Sheet{
			Title:      "create schema",
			Content:    []byte(createSchema),
			Visibility: v1pb.Sheet_VISIBILITY_PROJECT,
			Source:     v1pb.Sheet_SOURCE_BYTEBASE_ARTIFACT,
			Type:       v1pb.Sheet_TYPE_SQL,
		},
	})
	a.NoError(err)

	// Create an issue that updates database schema.
	err = ctl.changeDatabase(ctx, project, database, sheet, v1pb.Plan_ChangeDatabaseConfig_MIGRATE)
	a.NoError(err)

	resp, err := ctl.databaseServiceClient.ListChangeHistories(ctx, &v1pb.ListChangeHistoriesRequest{
		Parent: database.Name,
	})
	a.NoError(err)
	histories := resp.ChangeHistories
	// history[0] is SchemaUpdate.
	a.Equal(1, len(histories))
	latest := histories[0]

	err = ctl.createDatabase(ctx, projectUID, instance, newDatabaseName, "bytebase", nil)
	a.NoError(err)

	newDatabase, err := ctl.databaseServiceClient.GetDatabase(ctx, &v1pb.GetDatabaseRequest{
		Name: fmt.Sprintf("%s/databases/%s", instance.Name, newDatabaseName),
	})
	a.NoError(err)

	newDatabaseSchema, err := ctl.databaseServiceClient.GetDatabaseSchema(ctx, &v1pb.GetDatabaseSchemaRequest{
		Name: fmt.Sprintf("%s/schema", newDatabase.Name),
	})
	a.NoError(err)

	diff, err := ctl.getSchemaDiff(schemaDiffRequest{
		EngineType:   parser.Postgres,
		SourceSchema: latest.Schema,
		TargetSchema: newDatabaseSchema.Schema,
	})

	a.NoError(err)
	a.Equal(expectedDiff, diff)
}
