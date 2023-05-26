package tests

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/db"
	parser "github.com/bytebase/bytebase/backend/plugin/parser/sql"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"

	"github.com/bytebase/bytebase/backend/resources/postgres"
	"github.com/bytebase/bytebase/backend/tests/fake"
)

func TestSyncSchema(t *testing.T) {
	const (
		databaseName    = "sync_schema"
		newDatabaseName = "sync_schema_new"
		createSchema    = `
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
		dataDir:            dataDir,
		vcsProviderCreator: fake.NewGitLab,
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

	prodEnvironment, _, err := ctl.getEnvironment(ctx, "prod")
	a.NoError(err)

	instance, err := ctl.instanceServiceClient.CreateInstance(ctx, &v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance", 10),
		Instance: &v1pb.Instance{
			Title:       "pgTestSyncSchema",
			Engine:      v1pb.Engine_POSTGRES,
			Environment: prodEnvironment.Name,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: "/tmp", Port: strconv.Itoa(pgPort), Username: "bytebase", Password: "bytebase"}},
		},
	})
	a.NoError(err)
	instanceUID, err := strconv.Atoi(instance.Uid)
	a.NoError(err)

	databases, err := ctl.getDatabases(api.DatabaseFind{
		ProjectID: &projectUID,
	})
	a.NoError(err)
	a.Nil(databases)

	err = ctl.createDatabase(ctx, projectUID, instance, databaseName, "bytebase", nil)
	a.NoError(err)

	databases, err = ctl.getDatabases(api.DatabaseFind{
		ProjectID: &projectUID,
	})
	a.NoError(err)
	a.Equal(1, len(databases))

	database := databases[0]
	a.Equal(instanceUID, database.Instance.ID)

	sheet, err := ctl.createSheet(api.SheetCreate{
		ProjectID:  projectUID,
		Name:       "create schema",
		Statement:  createSchema,
		Visibility: api.ProjectSheet,
		Source:     api.SheetFromBytebaseArtifact,
		Type:       api.SheetForSQL,
	})
	a.NoError(err)

	// Create an issue that updates database schema.
	createContext, err := json.Marshal(&api.MigrationContext{
		DetailList: []*api.MigrationDetail{
			{
				MigrationType: db.Migrate,
				DatabaseID:    database.ID,
				SheetID:       sheet.ID,
			},
		},
	})
	a.NoError(err)
	issue, err := ctl.createIssue(api.IssueCreate{
		ProjectID:     projectUID,
		Name:          fmt.Sprintf("Create sequence for database %q", databaseName),
		Type:          api.IssueDatabaseSchemaUpdate,
		Description:   fmt.Sprintf("Create sequence of database %q.", databaseName),
		AssigneeID:    api.SystemBotID,
		CreateContext: string(createContext),
	})
	a.NoError(err)
	status, err := ctl.waitIssuePipeline(ctx, issue.ID)
	a.NoError(err)
	a.Equal(api.TaskDone, status)

	history, err := ctl.getInstanceMigrationHistory(instanceUID, db.MigrationHistoryFind{
		Database: &database.Name,
	})
	a.NoError(err)

	// history[0] is SchemaUpdate
	a.Equal(1, len(history))
	latest := history[0]

	err = ctl.createDatabase(ctx, projectUID, instance, newDatabaseName, "bytebase", nil)
	a.NoError(err)

	dbName := newDatabaseName
	databases, err = ctl.getDatabases(api.DatabaseFind{
		ProjectID: &projectUID,
		Name:      &dbName,
	})
	a.NoError(err)
	a.Equal(1, len(databases))

	newDatabase := databases[0]
	a.Equal(instanceUID, database.Instance.ID)

	newDatabaseSchema, err := ctl.getLatestSchemaDump(newDatabase.ID)
	a.NoError(err)

	diff, err := ctl.getSchemaDiff(schemaDiffRequest{
		EngineType:   parser.Postgres,
		SourceSchema: latest.Schema,
		TargetSchema: newDatabaseSchema,
	})

	a.NoError(err)
	a.Equal(expectedDiff, diff)
}
