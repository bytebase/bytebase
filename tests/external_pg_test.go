package tests

import (
	"context"
	"fmt"
	"os"
	"path"
	"testing"
	"time"

	"github.com/bytebase/bytebase/resources/postgres"
	"github.com/stretchr/testify/require"
)

type fakeExternalPostgres struct {
	instance *postgres.Instance
	pgUser   string
	pgURL    string
}

// newFakeExternalPostgres will install postgres in tmpDir and lis
func newFakeExternalPostgres(tmpDir string, port int) (*fakeExternalPostgres, error) {
	resourceDir := path.Join(tmpDir, "resources")
	dataDir := path.Join(tmpDir, "pgdata")
	pgUser := "bbexternal"
	pgIns, err := postgres.Install(resourceDir, dataDir, pgUser)
	if err != nil {
		return nil, err
	}

	err = pgIns.Start(port, os.Stderr, os.Stderr)
	if err != nil {
		return nil, err
	}
	return &fakeExternalPostgres{
		instance: pgIns,
		pgUser:   pgUser,
		// The host component is interpreted as described for the parameter host.
		// In particular, a Unix-domain socket connection is chosen if the host part is either empty or starts with a slash,
		// otherwise a TCP/IP connection is initiated.
		pgURL: fmt.Sprintf("postgresql://%s@:%d/%s", pgUser, port, "postgres"),
	}, nil
}

func (f *fakeExternalPostgres) destroy() error {
	return f.instance.Stop(os.Stderr, os.Stderr)
}

func TestBootWithExternalPostgres(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	pgTmpDir := t.TempDir()
	dataDir := t.TempDir()
	port := getTestPort(t.Name())
	serverPort := port + 1

	externalPg, err := newFakeExternalPostgres(pgTmpDir, port)
	require.NoError(t, err)
	defer func() {
		_ = externalPg.destroy()
	}()

	ctl := &controller{}
	err = ctl.StartServerWithExternalPg(ctx, dataDir, serverPort, externalPg.pgURL)
	require.NoError(t, err)
	defer ctl.Close(ctx)

	timer := time.NewTimer(5 * time.Second)
	for {
		select {
		case <-timer.C:
			return
		default:
			err := ctl.reachHealthz()
			require.NoError(t, err)
		}
	}
}
