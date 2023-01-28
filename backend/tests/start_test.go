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
)

func startStopServer(ctx context.Context, a *require.Assertions, ctl *controller, dataDir string, readOnly bool) {
	err := ctl.StartServer(ctx, &config{
		dataDir:            dataDir,
		vcsProviderCreator: fake.NewGitLab,
		readOnly:           readOnly,
	})
	a.NoError(err)

	projects, err := ctl.getProjects()
	a.NoError(err)

	// Default + Sample project.
	a.Equal(2, len(projects))
	a.Equal("Default", projects[0].Name)
	a.Equal("Sample Project", projects[1].Name)

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
	err := ctl.StartServer(ctx, &config{
		dataDir:            dataDir,
		vcsProviderCreator: fake.NewGitLab,
	})
	a.NoError(err)
	err = ctl.Signup()
	a.NoError(err)
	err = ctl.Login()
	a.NoError(err)

	err = ctl.Close(ctx)
	a.NoError(err)

	// Start server in readonly mode
	startStopServer(ctx, a, ctl, dataDir, true /*readOnly*/)

	// Start server in non-readonly mode
	startStopServer(ctx, a, ctl, dataDir, false /*readOnly*/)
}

var (
	mysqlBinDir string
)

func TestMain(m *testing.M) {
	resourceDir = os.TempDir()
	dir, err := postgres.Install(resourceDir)
	if err != nil {
		log.Fatal(err)
	}
	externalPgBinDir = dir
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
	externalPgDataDir = dir
	if err := postgres.InitDB(externalPgBinDir, externalPgDataDir, externalPgUser); err != nil {
		log.Fatal(err)
	}
	if err = postgres.Start(externalPgPort, externalPgBinDir, externalPgDataDir, false /* serverLog */); err != nil {
		log.Fatal(err)
	}

	code := m.Run()

	// Graceful shutdown.
	if err := postgres.Stop(externalPgBinDir, externalPgDataDir); err != nil {
		log.Fatal(err)
	}
	if err := os.RemoveAll(externalPgDataDir); err != nil {
		log.Fatal(err)
	}

	os.Exit(code)
}
