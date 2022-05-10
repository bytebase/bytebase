package tests

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestServiceRestart(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	ctl := &controller{}
	dataDir := t.TempDir()
	err := ctl.StartServer(ctx, dataDir, getTestPort(t.Name()))
	a.NoError(err)

	err = ctl.Login()
	a.NoError(err)

	projects, err := ctl.getProjects()
	a.NoError(err)

	// Test seed should have more than one project.
	a.Greater(len(projects), 1)

	// Restart the server.
	err = ctl.Close(ctx)
	a.NoError(err)

	err = ctl.StartServer(ctx, dataDir, getTestPort(t.Name()))
	a.NoError(err)
	defer ctl.Close(ctx)

	err = ctl.Login()
	a.NoError(err)
}
