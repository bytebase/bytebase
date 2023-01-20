package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/api"
	"github.com/bytebase/bytebase/backend/plugin/db"
	resourcemysql "github.com/bytebase/bytebase/backend/resources/mysql"
	"github.com/bytebase/bytebase/backend/resources/postgres"
	"github.com/bytebase/bytebase/backend/tests/fake"
)

func TestAdminQueryAffectedRows(t *testing.T) {
	tests := []struct {
		databaseName      string
		dbType            db.Type
		prepareStatements string
		query             string
		want              bool
		affectedRows      string
	}{
		{
			databaseName:      "Test1",
			dbType:            db.MySQL,
			prepareStatements: "CREATE TABLE tbl(id INT PRIMARY KEY);",
			query:             "INSERT INTO tbl VALUES(1);",
			affectedRows:      `[["Affected Rows"],["INT"],[[1]]]`,
		},
		{
			databaseName:      "Test2",
			dbType:            db.MySQL,
			prepareStatements: "CREATE TABLE tbl(id INT PRIMARY KEY);",
			query:             "INSERT INTO tbl VALUES(1); DELETE FROM tbl WHERE id = 1;",
			affectedRows:      `[[],null,[]]`,
		},
		{
			databaseName:      "Test3",
			dbType:            db.Postgres,
			prepareStatements: "CREATE TABLE public.tbl(id INT PRIMARY KEY);",
			query:             "INSERT INTO tbl VALUES(1),(2);",
			affectedRows:      `[["Affected Rows"],["INT"],[[2]]]`,
		},
		{
			databaseName:      "Test4",
			dbType:            db.Postgres,
			prepareStatements: "CREATE TABLE tbl(id INT PRIMARY KEY);",
			query:             "ALTER TABLE tbl ADD COLUMN name VARCHAR(255);",
			affectedRows:      `[[],null,[]]`,
		},
	}

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

	mysqlPort := getTestPort()
	mysqlStopInstance := resourcemysql.SetupTestInstance(t, mysqlPort, mysqlBinDir)
	defer mysqlStopInstance()

	// Create a PostgreSQL instance.
	pgPort := getTestPort()
	pgStopInstance := postgres.SetupTestInstance(t, pgPort, resourceDirOverride)
	defer pgStopInstance()

	// Create a project.
	project, err := ctl.createProject(api.ProjectCreate{
		Name: "Test Project",
		Key:  t.Name(),
	})
	a.NoError(err)
	environments, err := ctl.getEnvironments()
	a.NoError(err)
	prodEnvironment, err := findEnvironment(environments, "Prod")
	a.NoError(err)

	mysqlInstance, err := ctl.addInstance(api.InstanceCreate{
		EnvironmentID: prodEnvironment.ID,
		Name:          "mysqlInstance",
		Engine:        db.MySQL,
		Host:          "127.0.0.1",
		Port:          strconv.Itoa(mysqlPort),
		Username:      "root",
		Password:      "",
	})
	a.NoError(err)

	pgInstance, err := ctl.addInstance(api.InstanceCreate{
		EnvironmentID: prodEnvironment.ID,
		Name:          "pgInstance",
		Engine:        db.Postgres,
		Host:          "/tmp",
		Port:          strconv.Itoa(pgPort),
		Username:      "root",
	})
	a.NoError(err)

	for idx, tt := range tests {
		var instance *api.Instance
		databaseOwner := ""
		switch tt.dbType {
		case db.MySQL:
			instance = mysqlInstance
		case db.Postgres:
			instance = pgInstance
			databaseOwner = "root"
		default:
			a.FailNow("unsupported db type")
		}
		err = ctl.createDatabase(project, instance, tt.databaseName, databaseOwner, nil)
		a.NoError(err)

		databases, err := ctl.getDatabases(api.DatabaseFind{
			ProjectID: &project.ID,
		})
		a.NoError(err)
		a.Equal(idx+1, len(databases))

		var database *api.Database
		for _, d := range databases {
			if d.Name == tt.databaseName {
				database = d
				break
			}
		}
		a.NotNil(database)

		a.Equal(instance.ID, database.Instance.ID)

		// Create an issue that updates database schema.
		createContext, err := json.Marshal(&api.MigrationContext{
			DetailList: []*api.MigrationDetail{
				{
					MigrationType: db.Migrate,
					DatabaseID:    database.ID,
					Statement:     tt.prepareStatements,
				},
			},
		})
		a.NoError(err)
		issue, err := ctl.createIssue(api.IssueCreate{
			ProjectID:     project.ID,
			Name:          fmt.Sprintf("Prepare statements of database %q", tt.databaseName),
			Type:          api.IssueDatabaseSchemaUpdate,
			Description:   fmt.Sprintf("Prepare statements of database %q.", tt.databaseName),
			AssigneeID:    api.SystemBotID,
			CreateContext: string(createContext),
		})
		a.NoError(err)
		status, err := ctl.waitIssuePipeline(issue.ID)
		a.NoError(err)
		a.Equal(api.TaskDone, status)

		affectedRows, err := ctl.adminQuery(instance, tt.databaseName, tt.query)
		a.NoError(err)
		a.Equal(tt.affectedRows, affectedRows)
	}
}
