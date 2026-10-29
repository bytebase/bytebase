package v1

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/testing/protocmp"

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

func TestConvertToQueryDataPolicyPayloadExportDeliveryPolicy(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		allowed []v1pb.ExportDelivery
		want    []storepb.ExportDelivery
		wantErr string
	}{
		{name: "local and sharepoint", allowed: []v1pb.ExportDelivery{v1pb.ExportDelivery_SHAREPOINT, v1pb.ExportDelivery_LOCAL}, want: []storepb.ExportDelivery{storepb.ExportDelivery_SHAREPOINT, storepb.ExportDelivery_LOCAL}},
		{name: "empty", wantErr: "must allow at least one delivery"},
		{name: "unspecified", allowed: []v1pb.ExportDelivery{v1pb.ExportDelivery_EXPORT_DELIVERY_UNSPECIFIED}, wantErr: "unsupported export delivery"},
		{name: "duplicate", allowed: []v1pb.ExportDelivery{v1pb.ExportDelivery_LOCAL, v1pb.ExportDelivery_LOCAL}, wantErr: "more than once"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			payload, err := convertToQueryDataPolicyPayload(&v1pb.QueryDataPolicy{
				ExportDeliveryPolicy: &v1pb.QueryDataPolicy_ExportDeliveryPolicy{AllowedDeliveries: tc.allowed},
			})
			if tc.wantErr != "" {
				require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
				require.ErrorContains(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, payload.GetExportDeliveryPolicy().GetAllowedDeliveries())
		})
	}

	payload, err := convertToQueryDataPolicyPayload(&v1pb.QueryDataPolicy{MaximumResultRows: 10})
	require.NoError(t, err)
	require.Nil(t, payload.GetExportDeliveryPolicy(), "an omitted export delivery policy stays unset")
}

func TestConvertToPolicyQueryDataExportDeliveryDefault(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name         string
		resourceType storepb.Policy_Resource
		payload      string
		want         *v1pb.QueryDataPolicy_ExportDeliveryPolicy
	}{
		{
			name:         "workspace without the field reads the default",
			resourceType: storepb.Policy_WORKSPACE,
			payload:      `{"maximumResultRows":10}`,
			want:         &v1pb.QueryDataPolicy_ExportDeliveryPolicy{AllowedDeliveries: []v1pb.ExportDelivery{v1pb.ExportDelivery_LOCAL}},
		},
		{
			name:         "workspace with the field reads it",
			resourceType: storepb.Policy_WORKSPACE,
			payload:      `{"exportDeliveryPolicy":{"allowedDeliveries":["SHAREPOINT"]}}`,
			want:         &v1pb.QueryDataPolicy_ExportDeliveryPolicy{AllowedDeliveries: []v1pb.ExportDelivery{v1pb.ExportDelivery_SHAREPOINT}},
		},
		{
			name:         "project has none",
			resourceType: storepb.Policy_PROJECT,
			payload:      `{"maximumResultRows":10}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			policy, err := convertToPolicy(&store.PolicyMessage{
				Resource:     "workspaces/example",
				ResourceType: tc.resourceType,
				Type:         storepb.Policy_QUERY_DATA,
				Payload:      tc.payload,
				Enforce:      true,
			})
			require.NoError(t, err)
			require.Empty(t, cmp.Diff(tc.want, policy.GetQueryDataPolicy().GetExportDeliveryPolicy(), protocmp.Transform()))
		})
	}
}

func TestKeepStoredExportDeliveryPolicy(t *testing.T) {
	t.Parallel()
	const stored = `{"maximumResultRows":5,"exportDeliveryPolicy":{"allowedDeliveries":["LOCAL","SHAREPOINT"]}}`

	got, err := keepStoredExportDeliveryPolicy(`{"maximumResultRows":10}`, stored)
	require.NoError(t, err)
	require.JSONEq(t, `{"maximumResultRows":10,"exportDeliveryPolicy":{"allowedDeliveries":["LOCAL","SHAREPOINT"]}}`, got, "an update that omits it keeps the stored value")

	got, err = keepStoredExportDeliveryPolicy(`{"exportDeliveryPolicy":{"allowedDeliveries":["SHAREPOINT"]}}`, stored)
	require.NoError(t, err)
	require.JSONEq(t, `{"exportDeliveryPolicy":{"allowedDeliveries":["SHAREPOINT"]}}`, got, "an update that sets it replaces the stored value")
}

func TestValidateQueryDataPolicyResource(t *testing.T) {
	t.Parallel()
	policy := &v1pb.Policy{Policy: &v1pb.Policy_QueryDataPolicy{QueryDataPolicy: &v1pb.QueryDataPolicy{
		ExportDeliveryPolicy: &v1pb.QueryDataPolicy_ExportDeliveryPolicy{AllowedDeliveries: []v1pb.ExportDelivery{v1pb.ExportDelivery_LOCAL}},
	}}}
	require.NoError(t, validateQueryDataPolicyResource(storepb.Policy_WORKSPACE, policy))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(validateQueryDataPolicyResource(storepb.Policy_PROJECT, policy)))
	require.NoError(t, validateQueryDataPolicyResource(storepb.Policy_PROJECT, &v1pb.Policy{Policy: &v1pb.Policy_QueryDataPolicy{QueryDataPolicy: &v1pb.QueryDataPolicy{MaximumResultRows: 10}}}))
}
