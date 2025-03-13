package tests

import (
	"context"
	"fmt"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
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
			filter: fmt.Sprintf(`table = "tableExists('%s', 'public', 't2')"`, databaseName),
			wantStatements: []string{
				statements[1],
				statements[2],
			},
		},
		{
			filter: fmt.Sprintf(`table = "tableExists('%s', 'public', 't2') && tableExists('%s', 'public', 't3')"`, databaseName, databaseName),
			wantStatements: []string{
				statements[1],
			},
		},
		{
			filter: fmt.Sprintf(`table = "(tableExists('%s', 'public', 't2') && tableExists('%s', 'public', 't3')) || tableExists('%s', 'public', 't1')"`, databaseName, databaseName, databaseName),
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

	instance, err := ctl.instanceServiceClient.CreateInstance(ctx, &v1pb.CreateInstanceRequest{
		InstanceId: generateRandomString("instance", 10),
		Instance: &v1pb.Instance{
			Title:       "testFilterChangeHistoryInstance1",
			Engine:      v1pb.Engine_POSTGRES,
			Environment: "environments/prod",
			Activation:  true,
			DataSources: []*v1pb.DataSource{{Type: v1pb.DataSourceType_ADMIN, Host: pgContainer.host, Port: pgContainer.port, Username: "bytebase", Password: "bytebase", Id: "admin"}},
		},
	})
	a.NoError(err)

	// Create an issue that creates a database.
	err = ctl.createDatabaseV2(ctx, ctl.project, instance, nil /* environment */, databaseName, "bytebase")
	a.NoError(err)

	database, err := ctl.databaseServiceClient.GetDatabase(ctx, &v1pb.GetDatabaseRequest{
		Name: fmt.Sprintf("%s/databases/%s", instance.Name, databaseName),
	})
	a.NoError(err)

	for i, stmt := range statements {
		sheet, err := ctl.sheetServiceClient.CreateSheet(ctx, &v1pb.CreateSheetRequest{
			Parent: ctl.project.Name,
			Sheet: &v1pb.Sheet{
				Title:   fmt.Sprintf("migration statement sheet %d", i+1),
				Content: []byte(stmt),
			},
		})
		a.NoError(err)

		// Create an issue that updates database schema.
		err = ctl.changeDatabase(ctx, ctl.project, database, sheet, v1pb.Plan_ChangeDatabaseConfig_MIGRATE)
		a.NoError(err)
	}

	// Get migration history by filter.
	for _, tt := range tests {
		resp, err := ctl.databaseServiceClient.ListChangelogs(ctx, &v1pb.ListChangelogsRequest{
			Parent: database.Name,
			View:   v1pb.ChangelogView_CHANGELOG_VIEW_FULL,
			Filter: tt.filter,
		})
		a.NoError(err)
		a.Equal(len(tt.wantStatements), len(resp.Changelogs), tt.filter)
		for i, wantStatement := range tt.wantStatements {
			// Sort by changelog UID.
			sort.Slice(resp.Changelogs, func(i, j int) bool {
				_, _, id1, err := common.GetInstanceDatabaseChangelogUID(resp.Changelogs[i].Name)
				a.NoError(err)
				_, _, id2, err := common.GetInstanceDatabaseChangelogUID(resp.Changelogs[j].Name)
				a.NoError(err)
				return id1 < id2
			})
			a.Equal(wantStatement, string(resp.Changelogs[i].Statement), tt.filter)
		}
	}
}
