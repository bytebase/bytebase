package tests

import (
	"context"
	"fmt"
	"path"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/common/log"
	"github.com/bytebase/bytebase/resources/postgres"
	"github.com/bytebase/bytebase/tests/fake"
)

type fakeExternalPg struct {
	pgBinDir  string
	pgDataDir string
	pgURL     string
	pgUser    string
}

// newFakeExternalPg will install postgres in tmpDir and listen on Unix domain socket with port.
func newFakeExternalPg(tmpDir string, port int) (*fakeExternalPg, error) {
	resourceDir := path.Join(tmpDir, "resources")
	dataDir := path.Join(tmpDir, "pgdata")
	pgUser := "bbexternal"
	pgBinDir, err := postgres.Install(resourceDir)
	if err != nil {
		return nil, errors.Wrap(err, "cannot install postgres")
	}

	if err := postgres.InitDB(pgBinDir, dataDir, pgUser); err != nil {
		return nil, errors.Wrap(err, "cannot initdb")
	}

	err = postgres.Start(port, pgBinDir, dataDir)
	if err != nil {
		return nil, errors.Wrap(err, "cannot start postgres server")
	}

	return &fakeExternalPg{
		pgBinDir:  pgBinDir,
		pgDataDir: dataDir,
		pgURL:     fmt.Sprintf("postgresql://%s@:%d/%s?host=%s", pgUser, port, "postgres", common.GetPostgresSocketDir()),
		pgUser:    pgUser,
	}, nil
}

func (f *fakeExternalPg) Destroy() error {
	return postgres.Stop(f.pgBinDir, f.pgDataDir)
}

func TestBootWithExternalPg(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()

	pgTmpDir := t.TempDir()

	pgPort := getTestPort()
	externalPg, err := newFakeExternalPg(pgTmpDir, pgPort)
	a.NoError(err)
	defer func() {
		if err = externalPg.Destroy(); err != nil {
			log.Error("cannot destroy postgres instance", zap.Error(err))
			t.FailNow()
		}
	}()

	ctl := &controller{}
	dataTmpDir := t.TempDir()
	err = ctl.StartServerWithExternalPg(ctx, &config{
		dataDir:            dataTmpDir,
		vcsProviderCreator: fake.NewGitLab,
		pgUser:             externalPg.pgUser,
		pgURL:              externalPg.pgURL,
	})
	a.NoError(err)
	defer ctl.Close(ctx)
}
