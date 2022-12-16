package tests

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/plugin/parser"
	"github.com/bytebase/bytebase/resources/postgres"
	"github.com/bytebase/bytebase/tests/fake"
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
	err := ctl.StartServerWithExternalPg(ctx, &config{
		dataDir:            dataDir,
		vcsProviderCreator: fake.NewGitLab,
	})
	a.NoError(err)
	defer ctl.Close(ctx)

	// Create a PostgreSQL instance.
	pgPort := getTestPort()
	stopInstance := postgres.SetupTestInstance(t, pgPort, resourceDirOverride)
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
	project, err := ctl.createProject(api.ProjectCreate{
		Name: "Test sync schema Project",
		Key:  "TestSyncSchemaProject",
	})
	a.NoError(err)

	environments, err := ctl.getEnvironments()
	a.NoError(err)
	prodEnvironment, err := findEnvironment(environments, "Prod")
	a.NoError(err)

	err = ctl.setLicense()
	a.NoError(err)

	instance, err := ctl.addInstance(api.InstanceCreate{
		EnvironmentID: prodEnvironment.ID,
		Name:          "pgTestSyncSchema",
		Engine:        db.Postgres,
		Host:          "/tmp",
		Port:          strconv.Itoa(pgPort),
		Username:      "bytebase",
		Password:      "bytebase",
	})
	a.NoError(err)

	databases, err := ctl.getDatabases(api.DatabaseFind{
		ProjectID: &project.ID,
	})
	a.NoError(err)
	a.Nil(databases)

	err = ctl.createDatabase(project, instance, databaseName, "bytebase", nil)
	a.NoError(err)

	databases, err = ctl.getDatabases(api.DatabaseFind{
		ProjectID: &project.ID,
	})
	a.NoError(err)
	a.Equal(1, len(databases))

	database := databases[0]
	a.Equal(instance.ID, database.Instance.ID)

	// Create an issue that updates database schema.
	createContext, err := json.Marshal(&api.MigrationContext{
		DetailList: []*api.MigrationDetail{
			{
				MigrationType: db.Migrate,
				DatabaseID:    database.ID,
				Statement:     createSchema,
			},
		},
	})
	a.NoError(err)
	issue, err := ctl.createIssue(api.IssueCreate{
		ProjectID:   project.ID,
		Name:        fmt.Sprintf("Create sequence for database %q", databaseName),
		Type:        api.IssueDatabaseSchemaUpdate,
		Description: fmt.Sprintf("Create sequence of database %q.", databaseName),
		// Assign to self.
		AssigneeID:    project.Creator.ID,
		CreateContext: string(createContext),
	})
	a.NoError(err)
	status, err := ctl.waitIssuePipeline(issue.ID)
	a.NoError(err)
	a.Equal(api.TaskDone, status)

	history, err := ctl.getInstanceMigrationHistory(db.MigrationHistoryFind{
		ID:       &instance.ID,
		Database: &database.Name,
	})
	a.NoError(err)

	// history[0] is SchemaUpdate
	// history[1] is CreateDatabase
	a.Equal(2, len(history))
	latest := history[0]

	err = ctl.createDatabase(project, instance, newDatabaseName, "bytebase", nil)
	a.NoError(err)

	dbName := newDatabaseName
	databases, err = ctl.getDatabases(api.DatabaseFind{
		ProjectID: &project.ID,
		Name:      &dbName,
	})
	a.NoError(err)
	a.Equal(1, len(databases))

	newDatabase := databases[0]
	a.Equal(instance.ID, database.Instance.ID)

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
