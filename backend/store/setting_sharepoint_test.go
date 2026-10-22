package store_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/testing/protocmp"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

func TestGetSharePointSetting(t *testing.T) {
	t.Parallel()
	fixture := newStorePostgresFixture(t, "")
	ctx, s := fixture.ctx, fixture.store

	setting, err := s.GetSharePointSetting(ctx, "default")
	require.NoError(t, err)
	require.Empty(t, cmp.Diff(&storepb.SharePointSetting{}, setting, protocmp.Transform()), "a workspace has no row until SharePoint is configured")

	want := &storepb.SharePointSetting{TenantId: "tenant", ClientId: "client", ClientSecret: "secret"}
	_, err = s.UpsertSetting(ctx, &store.SettingMessage{
		Name:      storepb.SettingName_SHAREPOINT,
		Workspace: "default",
		Value:     want,
	})
	require.NoError(t, err)
	setting, err = s.GetSharePointSetting(ctx, "default")
	require.NoError(t, err)
	require.Empty(t, cmp.Diff(want, setting, protocmp.Transform()))
}
