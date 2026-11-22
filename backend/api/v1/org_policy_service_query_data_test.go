package v1

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
)

func TestQueryDataPolicyWriteRejectsDisableExport(t *testing.T) {
	t.Parallel()
	// The rejection runs before the license check, so a zero service is enough.
	s := &OrgPolicyService{}
	_, err := s.convertPolicyPayloadToString(context.Background(), &v1pb.Policy{
		Type: v1pb.PolicyType_DATA_QUERY,
		Policy: &v1pb.Policy_QueryDataPolicy{QueryDataPolicy: &v1pb.QueryDataPolicy{
			DisableExport:     true, //nolint:staticcheck
			MaximumResultRows: 10,
		}},
	})
	require.Error(t, err)
	connectErr := new(connect.Error)
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodeInvalidArgument, connectErr.Code())
	require.Equal(t, "disable_export is removed. Export requires the bb.sql.export permission; to require approval for export, assign roles without it.", connectErr.Message())
}

func TestConvertToQueryDataPolicyPayloadDropsDisableExportFalse(t *testing.T) {
	t.Parallel()
	payload, err := convertToQueryDataPolicyPayload(&v1pb.QueryDataPolicy{
		DisableExport:        false, //nolint:staticcheck
		MaximumResultRows:    10,
		DisableCopyData:      true,
		AllowAdminDataSource: true,
	})
	require.NoError(t, err)
	require.Equal(t, int32(10), payload.GetMaximumResultRows())
	require.True(t, payload.GetDisableCopyData())
	require.True(t, payload.GetAllowAdminDataSource())
}

func TestConvertToPolicyQueryDataIgnoresStoredDisableExport(t *testing.T) {
	t.Parallel()
	policy, err := convertToPolicy(&store.PolicyMessage{
		Resource:     "workspaces/example",
		ResourceType: storepb.Policy_WORKSPACE,
		Type:         storepb.Policy_QUERY_DATA,
		Payload:      `{"disableExport":true,"maximumResultRows":10,"disableCopyData":true}`,
		Enforce:      true,
	})
	require.NoError(t, err)
	queryData := policy.GetQueryDataPolicy()
	require.NotNil(t, queryData)
	require.False(t, queryData.GetDisableExport()) //nolint:staticcheck
	require.Equal(t, int32(10), queryData.GetMaximumResultRows())
	require.True(t, queryData.GetDisableCopyData())
}

func TestQueryDataPolicyDisableDownloadRoundTrip(t *testing.T) {
	t.Parallel()
	payload, err := convertToQueryDataPolicyPayload(&v1pb.QueryDataPolicy{DisableDownload: true})
	require.NoError(t, err)
	data, err := protojson.Marshal(payload)
	require.NoError(t, err)
	require.JSONEq(t, `{"disableDownload":true}`, string(data))

	policy, err := convertToPolicy(&store.PolicyMessage{
		Resource:     "workspaces/example",
		ResourceType: storepb.Policy_WORKSPACE,
		Type:         storepb.Policy_QUERY_DATA,
		Payload:      string(data),
		Enforce:      true,
	})
	require.NoError(t, err)
	require.True(t, policy.GetQueryDataPolicy().GetDisableDownload())
}
