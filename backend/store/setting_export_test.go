package store_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

func TestGetExportSetting(t *testing.T) {
	t.Parallel()
	fixture := newStorePostgresFixture(t, "")
	ctx, s := fixture.ctx, fixture.store

	setting, err := s.GetExportSetting(ctx, "default")
	require.NoError(t, err)
	require.Empty(t, cmp.Diff(&storepb.ExportSetting{}, setting, protocmp.Transform()), "a workspace has no row until a destination is configured")

	want := &storepb.ExportSetting{Sharepoint: &storepb.SharePointSetting{TenantId: "tenant", ClientId: "client", ClientSecret: "secret"}}
	_, err = s.UpsertSetting(ctx, &store.SettingMessage{
		Name:      storepb.SettingName_EXPORT,
		Workspace: "default",
		Value:     want,
	})
	require.NoError(t, err)
	setting, err = s.GetExportSetting(ctx, "default")
	require.NoError(t, err)
	require.Empty(t, cmp.Diff(want, setting, protocmp.Transform()))
}

func TestGetWorkspaceQueryDataPolicy(t *testing.T) {
	t.Parallel()
	fixture := newStorePostgresFixture(t, "")
	ctx, s := fixture.ctx, fixture.store
	local := []storepb.ExportDelivery{storepb.ExportDelivery_LOCAL}
	both := []storepb.ExportDelivery{storepb.ExportDelivery_LOCAL, storepb.ExportDelivery_SHAREPOINT}

	policy, err := s.GetWorkspaceQueryDataPolicy(ctx, "default")
	require.NoError(t, err)
	require.Nil(t, policy.GetExportDeliveryPolicy())
	require.Equal(t, local, store.GetAllowedExportDeliveries(policy), "a workspace without a policy allows the default")

	_, err = s.CreatePolicy(ctx, &store.PolicyMessage{
		Workspace:    "default",
		ResourceType: storepb.Policy_PROJECT,
		Resource:     common.FormatProject("default"),
		Type:         storepb.Policy_QUERY_DATA,
		Payload:      `{"exportDeliveryPolicy":{"allowedDeliveries":["SHAREPOINT"]}}`,
		Enforce:      true,
	})
	require.NoError(t, err)
	policy, err = s.GetWorkspaceQueryDataPolicy(ctx, "default")
	require.NoError(t, err)
	require.Equal(t, local, store.GetAllowedExportDeliveries(policy), "a project policy does not set the workspace policy")

	_, err = s.CreatePolicy(ctx, &store.PolicyMessage{
		Workspace:    "default",
		ResourceType: storepb.Policy_WORKSPACE,
		Resource:     common.FormatWorkspace("default"),
		Type:         storepb.Policy_QUERY_DATA,
		Payload:      `{"exportDeliveryPolicy":{"allowedDeliveries":["LOCAL","SHAREPOINT"]}}`,
		Enforce:      true,
	})
	require.NoError(t, err)
	policy, err = s.GetWorkspaceQueryDataPolicy(ctx, "default")
	require.NoError(t, err)
	require.Equal(t, both, store.GetAllowedExportDeliveries(policy))
}
