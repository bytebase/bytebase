package v1

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
)

func TestConvertAISetting(t *testing.T) {
	t.Parallel()

	stored := convertAISetting(&v1pb.AISetting{
		Enabled:          true,
		Provider:         v1pb.AISetting_CLAUDE,
		Endpoint:         "https://api.anthropic.com/v1/messages",
		ApiKey:           "key",
		Model:            "claude-sonnet-5-5",
		Version:          "2023-06-01",
		SqlReviewEnabled: true,
	})
	require.Equal(t, &storepb.AISetting{
		Enabled:          true,
		Provider:         storepb.AISetting_CLAUDE,
		Endpoint:         "https://api.anthropic.com/v1/messages",
		ApiKey:           "key",
		Model:            "claude-sonnet-5-5",
		Version:          "2023-06-01",
		SqlReviewEnabled: true,
	}, stored)

	returned := convertToAISetting(stored)
	require.Empty(t, returned.ApiKey, "the key is never returned")
	require.True(t, returned.SqlReviewEnabled)
	require.True(t, returned.Enabled)

	require.False(t, convertToAISetting(&storepb.AISetting{Enabled: true}).SqlReviewEnabled, "the review is off until it is switched on")
}

func TestConvertExportSettingNeverReturnsSecret(t *testing.T) {
	t.Parallel()

	stored := convertExportSetting(&v1pb.ExportSetting{Sharepoint: &v1pb.SharePointSetting{
		TenantId:        "tenant",
		ClientId:        "client",
		ClientSecret:    "secret",
		ClientSecretSet: false,
	}})
	require.True(t, proto.Equal(&storepb.ExportSetting{Sharepoint: &storepb.SharePointSetting{TenantId: "tenant", ClientId: "client", ClientSecret: "secret"}}, stored))

	setting, err := convertToSettingMessage(&store.SettingMessage{Name: storepb.SettingName_EXPORT, Value: stored})
	require.NoError(t, err)
	require.Equal(t, "settings/EXPORT", setting.Name)
	require.True(t, proto.Equal(&v1pb.ExportSetting{Sharepoint: &v1pb.SharePointSetting{TenantId: "tenant", ClientId: "client", ClientSecretSet: true}}, setting.Value.GetExport()))

	require.False(t, convertToSharePointSetting(&storepb.SharePointSetting{TenantId: "tenant"}).ClientSecretSet)
	require.Nil(t, convertExportSetting(nil).Sharepoint)
	require.Nil(t, convertToExportSetting(&storepb.ExportSetting{}).Sharepoint, "unset when SharePoint is not configured")
}

func TestSettingNameMappingRoundTrips(t *testing.T) {
	t.Parallel()

	for value, name := range v1pb.Setting_SettingName_name {
		v1Name := v1pb.Setting_SettingName(value)
		require.Equal(t, v1Name, convertStoreSettingNameToV1(convertV1SettingNameToStore(v1Name)), name)
	}
}
