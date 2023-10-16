package tests

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/resources/mongoutil"
	resourcemysql "github.com/bytebase/bytebase/backend/resources/mysql"
	"github.com/bytebase/bytebase/backend/resources/mysqlutil"

	"github.com/bytebase/bytebase/backend/resources/postgres"
	"github.com/bytebase/bytebase/backend/tests/fake"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func startStopServer(ctx context.Context, a *require.Assertions, ctl *controller, dataDir string) {
	ctx, err := ctl.StartServer(ctx, &config{
		dataDir:            dataDir,
		vcsProviderCreator: fake.NewGitLab,
	})
	a.NoError(err)

	resp, err := ctl.projectServiceClient.ListProjects(ctx, &v1pb.ListProjectsRequest{})
	a.NoError(err)
	projects := resp.Projects

	// Default.
	a.Equal(1, len(projects))
	a.Equal("Default", projects[0].Title)

	err = ctl.Close(ctx)
	a.NoError(err)
}

func TestServerRestart(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	dataDir := t.TempDir()
	// Start server in non-readonly mode to init schema and register user.
	ctx, err := ctl.StartServer(ctx, &config{
		dataDir:            dataDir,
		vcsProviderCreator: fake.NewGitLab,
	})
	a.NoError(err)
	err = ctl.Close(ctx)
	a.NoError(err)

	// Start server in readonly mode
	startStopServer(ctx, a, ctl, dataDir)

	// Start server in non-readonly mode
	startStopServer(ctx, a, ctl, dataDir)
}

func TestMain(m *testing.M) {
	resourceDir = os.TempDir()
	dir, err := postgres.Install(resourceDir)
	if err != nil {
		log.Fatal(err)
	}
	pgBinDir = dir
	if _, err := mysqlutil.Install(resourceDir); err != nil {
		log.Fatal(err)
	}
	if _, err := mongoutil.Install(resourceDir); err != nil {
		log.Fatal(err)
	}
	dir, err = resourcemysql.Install(resourceDir)
	if err != nil {
		log.Fatal(err)
	}
	mysqlBinDir = dir

	dir, err = os.MkdirTemp("", "bbtest-pgdata")
	if err != nil {
		log.Fatal(err)
	}
	stopInstance := postgres.SetupTestInstance(pgBinDir, dir, externalPgPort)

	code := m.Run()

	stopInstance()

	os.Exit(code)
}
