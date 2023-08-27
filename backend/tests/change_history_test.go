package tests

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/resources/postgres"
	"github.com/bytebase/bytebase/backend/tests/fake"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

var (
	databaseName = "testFilterChangeHistoryDatabase1"
	statements   = []string{
		`CREATE TABLE t1(a int);`,
		`CREATE TABLE t2(a int); CREATE TABLE t3(a int);`,
		`DROP TABLE t2;`,
		`ALTER TABLE t3 ADD COLUMN b int;`,
	}

	tests = []struct {
		filter         string
		wantStatements []string
	}{
		{
			filter: fmt.Sprintf(`tableExists("%s", "public", "t2")`, databaseName),
			wantStatements: []string{
				statements[1],
				statements[2],
			},
		},
		{
			filter: fmt.Sprintf(`tableExists("%s", "public", "t2") && tableExists("%s", "public", "t3")`, databaseName, databaseName),
			wantStatements: []string{
				statements[1],
			},
		},
		{
			filter: fmt.Sprintf(`
				(tableExists("%s", "public", "t2") && tableExists("%s", "public", "t3"))
				||
				tableExists("%s", "public", "t1")
			`, databaseName, databaseName, databaseName),
			wantStatements: []string{
				statements[0],
				statements[1],
			},
		},
	}
)

func TestFilterChangeHistoryByResources(t *testing.T) {
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

	err = ctl.setLicense()
	a.NoError(err)
	// Create a project.
	project, err := ctl.createProject(ctx)
	a.NoError(err)

	prodEnvironment, err := ctl.getEnvironment(ctx, "prod")
	a.NoError(err)

	instance, err := ctl.instanceServiceClient.CreateInstance(ctx, &v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance", 10),
		Instance: &v1pb.Instance{
			Title:       "testFilterChangeHistoryInstance1",
			Engine:      v1pb.Engine_POSTGRES,
			Environment: prodEnvironment.Name,
			Activation:  true,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: "/tmp", Port: strconv.Itoa(pgPort), Username: "bytebase", Password: "bytebase"}},
		},
	})
	a.NoError(err)

	// Create an issue that creates a database.
	err = ctl.createDatabaseV2(ctx, project, instance, databaseName, "bytebase", nil)
	a.NoError(err)

	database, err := ctl.databaseServiceClient.GetDatabase(ctx, &v1pb.GetDatabaseRequest{
		Name: fmt.Sprintf("%s/databases/%s", instance.Name, databaseName),
	})
	a.NoError(err)

	for i, stmt := range statements {
		sheet, err := ctl.sheetServiceClient.CreateSheet(ctx, &v1pb.CreateSheetRequest{
			Parent: project.Name,
			Sheet: &v1pb.Sheet{
				Title:      fmt.Sprintf("migration statement sheet %d", i+1),
				Content:    []byte(stmt),
				Visibility: v1pb.Sheet_VISIBILITY_PROJECT,
				Source:     v1pb.Sheet_SOURCE_BYTEBASE_ARTIFACT,
				Type:       v1pb.Sheet_TYPE_SQL,
			},
		})
		a.NoError(err)

		// Create an issue that updates database schema.
		err = ctl.changeDatabase(ctx, project, database, sheet, v1pb.Plan_ChangeDatabaseConfig_MIGRATE)
		a.NoError(err)
	}

	// Get migration history by filter.
	for _, tt := range tests {
		resp, err := ctl.databaseServiceClient.ListChangeHistories(ctx, &v1pb.ListChangeHistoriesRequest{
			Parent: database.Name,
			Filter: tt.filter,
		})
		a.NoError(err)
		a.Equal(len(tt.wantStatements), len(resp.ChangeHistories), tt.filter)
		sort.Slice(resp.ChangeHistories, func(i, j int) bool {
			return resp.ChangeHistories[i].Uid < resp.ChangeHistories[j].Uid
		})
		for i, wantStatement := range tt.wantStatements {
			a.Equal(wantStatement, string(resp.ChangeHistories[i].Statement), tt.filter)
		}
	}
}
