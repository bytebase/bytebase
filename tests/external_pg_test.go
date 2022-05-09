package tests

import (
	"context"
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/bytebase/bytebase/resources/postgres"
	"github.com/stretchr/testify/require"
)

type fakeExternalPg struct {
	pgIns  *postgres.Instance
	pgURL  string
	pgUser string
}

// newFakeExternalPg will install postgres in tmpDir and listen on Unix domain socket with port
func newFakeExternalPg(tmpDir string, port int) (*fakeExternalPg, error) {
	resourceDir := path.Join(tmpDir, "resources")
	dataDir := path.Join(tmpDir, "pgdata")
	pgUser := "bbexternal"
	socket := "/tmp"
	pgIns, err := postgres.Install(resourceDir, dataDir, pgUser)
	if err != nil {
		return nil, fmt.Errorf("cannot install postgres, error: %w", err)
	}

	err = pgIns.Start(port, os.Stderr, os.Stderr)
	if err != nil {
		return nil, fmt.Errorf("cannot start postgres server, error: %w", err)
	}

	return &fakeExternalPg{
		pgIns:  pgIns,
		pgURL:  fmt.Sprintf("postgresql://%s@:%d/%s?host=%s", pgUser, port, "postgres", socket),
		pgUser: pgUser,
	}, nil
}

func (f *fakeExternalPg) Destroy() error {
	return f.pgIns.Stop(os.Stderr, os.Stderr)
}
func TestBootWithExternalPg(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	pgTmpDir := t.TempDir()

	port := getTestPort(t.Name())
	serverPort := port + 1

	externalPg, err := newFakeExternalPg(pgTmpDir, port)
	require.NoError(t, err)
	defer func() {
		if err = externalPg.Destroy(); err != nil {
			fmt.Printf("cannot destroy pginstance, error: %s", err.Error())
			t.FailNow()
		}
	}()

	ctl := &controller{}
	err = ctl.StartServerWithExternalPg(ctx, serverPort, externalPg.pgUser, externalPg.pgURL)
	require.NoError(t, err)
	defer ctl.Close(ctx)
}
