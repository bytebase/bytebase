package tests

import (
	"context"
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/bytebase/bytebase/common"
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
		pgURL:  fmt.Sprintf("postgresql://%s@:%d/%s?host=%s", pgUser, port, "postgres", common.GetPostgresSocketDir()),
		pgUser: pgUser,
	}, nil
}

func (f *fakeExternalPg) Destroy() error {
	return f.pgIns.Stop(os.Stderr, os.Stderr)
}
func TestBootWithExternalPg(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()

	pgTmpDir := t.TempDir()

	port := getTestPort(t.Name())
	serverPort := port + 1

	externalPg, err := newFakeExternalPg(pgTmpDir, port)
	a.NoError(err)
	defer func() {
		if err = externalPg.Destroy(); err != nil {
			fmt.Printf("cannot destroy postgres instance, error: %s", err.Error())
			t.FailNow()
		}
	}()

	ctl := &controller{}
	dataTmpDir := t.TempDir()
	err = ctl.StartServerWithExternalPg(ctx, dataTmpDir, serverPort, externalPg.pgUser, externalPg.pgURL)
	a.NoError(err)
	defer ctl.Close(ctx)
}
