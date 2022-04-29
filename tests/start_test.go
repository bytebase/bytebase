//go:build omni
// +build omni

package tests

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServiceRestart(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	ctl := &controller{}
	dataDir := t.TempDir()
	err := ctl.StartMain(ctx, dataDir, getTestPort(t.Name()))
	require.NoError(t, err)

	err = ctl.Login()
	require.NoError(t, err)

	projects, err := ctl.getProjects()
	require.NoError(t, err)

	// Test seed should have more than one project.
	assert.Greater(t, len(projects), 1)

	// Restart the server.
	err = ctl.Close()
	require.NoError(t, err)

	err = ctl.StartMain(ctx, dataDir, getTestPort(t.Name()))
	require.NoError(t, err)
	defer ctl.Close()

	err = ctl.Login()
	require.NoError(t, err)
}
