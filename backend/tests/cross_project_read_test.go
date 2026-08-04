package tests

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

func TestCollisionListDatabasesProjectIsolation(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	fixture := setupCollidingProjects(ctx, t, ctl)

	list := func(project *v1pb.Project) []*v1pb.Database {
		t.Helper()
		response, err := ctl.databaseServiceClient.ListDatabases(ctx, connect.NewRequest(&v1pb.ListDatabasesRequest{
			Parent:   project.Name,
			PageSize: 1000,
		}))
		a.NoError(err)
		return response.Msg.Databases
	}

	databasesA := list(fixture.ProjectA)
	a.NotEmpty(databasesA)
	a.Contains(databaseNames(databasesA), fixture.DatabaseA.Name)
	a.NotContains(databaseNames(databasesA), fixture.DatabaseB.Name)
	for _, database := range databasesA {
		a.Equal(fixture.ProjectA.Name, database.Project)
	}

	databasesB := list(fixture.ProjectB)
	a.NotEmpty(databasesB)
	a.Contains(databaseNames(databasesB), fixture.DatabaseB.Name)
	a.NotContains(databaseNames(databasesB), fixture.DatabaseA.Name)
	for _, database := range databasesB {
		a.Equal(fixture.ProjectB.Name, database.Project)
	}
}

func databaseNames(databases []*v1pb.Database) []string {
	names := make([]string, 0, len(databases))
	for _, database := range databases {
		names = append(names, database.Name)
	}
	return names
}
