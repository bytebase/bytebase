package tests

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/resources/mongoutil"
	resourcemysql "github.com/bytebase/bytebase/resources/mysql"
	"github.com/bytebase/bytebase/resources/mysqlutil"

	"github.com/bytebase/bytebase/resources/postgres"
	"github.com/bytebase/bytebase/tests/fake"
)

func TestServiceRestart(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	dataDir := t.TempDir()
	err := ctl.StartServer(ctx, &config{
		dataDir:            dataDir,
		vcsProviderCreator: fake.NewGitLab,
	})
	a.NoError(err)
	err = ctl.Signup()
	a.NoError(err)
	err = ctl.Login()
	a.NoError(err)

	projects, err := ctl.getProjects()
	a.NoError(err)

	// Default project.
	a.Equal(1, len(projects))
	a.Equal("Default", projects[0].Name)

	// Restart the server.
	err = ctl.Close(ctx)
	a.NoError(err)

	err = ctl.StartServer(ctx, &config{
		dataDir:            dataDir,
		vcsProviderCreator: fake.NewGitLab,
	})
	a.NoError(err)
	defer ctl.Close(ctx)
}

var (
	mysqlBinDir string
)

func TestMain(m *testing.M) {
	resourceDirOverride = os.TempDir()
	dir, err := postgres.Install(resourceDirOverride)
	if err != nil {
		log.Fatal(err)
	}
	externalPgBinDir = dir
	if _, err := mysqlutil.Install(resourceDirOverride); err != nil {
		log.Fatal(err)
	}
	if _, err := mongoutil.Install(resourceDirOverride); err != nil {
		log.Fatal(err)
	}
	dir, err = resourcemysql.Install(resourceDirOverride)
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
