package tests

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/tests/fake"
)

func TestBootWithExternalPg(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()

	ctl := &controller{}
	err := ctl.StartServerWithExternalPg(ctx, &config{
		dataDir:            externalPgDataDir,
		vcsProviderCreator: fake.NewGitLab,
		pgUser:             externalPgUser,
		pgURL:              fmt.Sprintf("postgresql://%s@:%d/%s?host=%s", externalPgUser, externalPgPort, "postgres", common.GetPostgresSocketDir()),
	})
	a.NoError(err)
	defer ctl.Close(ctx)
}
