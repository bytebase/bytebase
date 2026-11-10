package store_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

// TestMCPSettingsWithNoRowAreReadWrite pins the default every new WRITE method
// relies on: a workspace from before the MCP setting has no row and keeps
// Read-write, so the method reaches it with no migration.
func TestMCPSettingsWithNoRowAreReadWrite(t *testing.T) {
	t.Parallel()
	fixture := newStorePostgresFixture(t, "")
	ctx, s := fixture.ctx, fixture.store

	settings, err := s.GetMCPSettingsUncached(ctx, "default")
	require.NoError(t, err)
	require.Equal(t, storepb.MCPSetting_READ_WRITE, settings.GetCapability())

	_, err = s.UpsertSetting(ctx, &store.SettingMessage{
		Name:      storepb.SettingName_MCP,
		Workspace: "default",
		Value:     &storepb.MCPSetting{Capability: storepb.MCPSetting_READ_ONLY},
	})
	require.NoError(t, err)
	settings, err = s.GetMCPSettingsUncached(ctx, "default")
	require.NoError(t, err)
	require.Equal(t, storepb.MCPSetting_READ_ONLY, settings.GetCapability(), "a saved row is read as saved")
}
