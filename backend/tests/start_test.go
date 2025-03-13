package tests

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/resources/mongoutil"

	"github.com/bytebase/bytebase/backend/resources/postgres"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func startStopServer(ctx context.Context, a *require.Assertions, ctl *controller, dataDir string) {
	ctx, err := ctl.StartServerWithExternalPg(ctx)
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

	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	err = ctl.Close(ctx)
	a.NoError(err)

	ctx, err = ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	err = ctl.Close(ctx)
	a.NoError(err)
}

func TestMain(m *testing.M) {
	resourceDir = os.TempDir()
	if _, err := postgres.Install(resourceDir); err != nil {
		log.Fatal(err)
	}
	if _, err := mongoutil.Install(resourceDir); err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	pgContainer, err := getPgContainer(ctx)
	defer func() {
		pgContainer.Close(ctx)
	}()
	if err != nil {
		log.Fatal(err)
	}
	externalPgHost = pgContainer.host
	externalPgPort = pgContainer.port

	code := m.Run()

	os.Exit(code)
}
