package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

func newTrialLicenseFixture(t *testing.T) (context.Context, *store.Store) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)
	_, stores, _ := testcontainer.NewMetadataDBWithCache(t, true)

	_, err := stores.GetDB().ExecContext(ctx, `INSERT INTO workspace (resource_id) VALUES ('default')`)
	require.NoError(t, err)
	_, err = stores.UpsertSetting(ctx, &store.SettingMessage{
		Name:      storepb.SettingName_SYSTEM,
		Workspace: "default",
		Value:     &storepb.SystemSetting{},
	})
	require.NoError(t, err)
	return ctx, stores
}

func TestUpdateTrialLicense(t *testing.T) {
	t.Parallel()

	t.Run("stores license", func(t *testing.T) {
		t.Parallel()
		ctx, stores := newTrialLicenseFixture(t)

		err := stores.UpdateTrialLicense(ctx, "default", "trial-license")
		require.NoError(t, err)

		setting, err := stores.GetSystemSettingUncached(ctx, "default")
		require.NoError(t, err)
		require.Equal(t, "trial-license", setting.License)
		cached, err := stores.GetSystemSetting(ctx, "default")
		require.NoError(t, err)
		require.Equal(t, "trial-license", cached.License)
	})

	t.Run("replaces license", func(t *testing.T) {
		t.Parallel()
		ctx, stores := newTrialLicenseFixture(t)
		require.NoError(t, stores.UpdateLicense(ctx, "default", "team-trial"))

		err := stores.UpdateTrialLicense(ctx, "default", "enterprise-trial")
		require.NoError(t, err)

		setting, err := stores.GetSystemSettingUncached(ctx, "default")
		require.NoError(t, err)
		require.Equal(t, "enterprise-trial", setting.License)
	})
}
