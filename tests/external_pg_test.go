package tests

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

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
	})
	a.NoError(err)
	defer ctl.Close(ctx)
}
