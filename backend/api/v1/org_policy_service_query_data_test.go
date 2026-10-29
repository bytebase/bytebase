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

func TestConvertToQueryDataPolicyPayloadExportDestinationPolicy(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		allowed []v1pb.ExportDestination
		want    []storepb.ExportDestination
		wantErr string
	}{
		{name: "download and sharepoint", allowed: []v1pb.ExportDestination{v1pb.ExportDestination_SHAREPOINT, v1pb.ExportDestination_DOWNLOAD}, want: []storepb.ExportDestination{storepb.ExportDestination_SHAREPOINT, storepb.ExportDestination_DOWNLOAD}},
		{name: "empty", wantErr: "must allow at least one destination"},
		{name: "unspecified", allowed: []v1pb.ExportDestination{v1pb.ExportDestination_EXPORT_DESTINATION_UNSPECIFIED}, wantErr: "unsupported export destination"},
		{name: "duplicate", allowed: []v1pb.ExportDestination{v1pb.ExportDestination_DOWNLOAD, v1pb.ExportDestination_DOWNLOAD}, wantErr: "more than once"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			payload, err := convertToQueryDataPolicyPayload(&v1pb.QueryDataPolicy{
				ExportDestinationPolicy: &v1pb.QueryDataPolicy_ExportDestinationPolicy{AllowedDestinations: tc.allowed},
			})
			if tc.wantErr != "" {
				require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
				require.ErrorContains(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, payload.GetExportDestinationPolicy().GetAllowedDestinations())
		})
	}

	payload, err := convertToQueryDataPolicyPayload(&v1pb.QueryDataPolicy{MaximumResultRows: 10})
	require.NoError(t, err)
	require.Nil(t, payload.GetExportDestinationPolicy(), "an omitted export destination policy stays unset")
}

func TestConvertToPolicyQueryDataExportDestinationDefault(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name         string
		resourceType storepb.Policy_Resource
		payload      string
		want         *v1pb.QueryDataPolicy_ExportDestinationPolicy
	}{
		{
			name:         "workspace without the field reads the default",
			resourceType: storepb.Policy_WORKSPACE,
			payload:      `{"maximumResultRows":10}`,
			want:         &v1pb.QueryDataPolicy_ExportDestinationPolicy{AllowedDestinations: []v1pb.ExportDestination{v1pb.ExportDestination_DOWNLOAD}},
		},
		{
			name:         "workspace with the field reads it",
			resourceType: storepb.Policy_WORKSPACE,
			payload:      `{"exportDestinationPolicy":{"allowedDestinations":["SHAREPOINT"]}}`,
			want:         &v1pb.QueryDataPolicy_ExportDestinationPolicy{AllowedDestinations: []v1pb.ExportDestination{v1pb.ExportDestination_SHAREPOINT}},
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
			require.Empty(t, cmp.Diff(tc.want, policy.GetQueryDataPolicy().GetExportDestinationPolicy(), protocmp.Transform()))
		})
	}
}

func TestKeepStoredExportDestinationPolicy(t *testing.T) {
	t.Parallel()
	const stored = `{"maximumResultRows":5,"exportDestinationPolicy":{"allowedDestinations":["DOWNLOAD","SHAREPOINT"]}}`

	got, err := keepStoredExportDestinationPolicy(`{"maximumResultRows":10}`, stored)
	require.NoError(t, err)
	require.JSONEq(t, `{"maximumResultRows":10,"exportDestinationPolicy":{"allowedDestinations":["DOWNLOAD","SHAREPOINT"]}}`, got, "an update that omits it keeps the stored value")

	got, err = keepStoredExportDestinationPolicy(`{"exportDestinationPolicy":{"allowedDestinations":["SHAREPOINT"]}}`, stored)
	require.NoError(t, err)
	require.JSONEq(t, `{"exportDestinationPolicy":{"allowedDestinations":["SHAREPOINT"]}}`, got, "an update that sets it replaces the stored value")
}

func TestValidateQueryDataPolicyResource(t *testing.T) {
	t.Parallel()
	policy := &v1pb.Policy{Policy: &v1pb.Policy_QueryDataPolicy{QueryDataPolicy: &v1pb.QueryDataPolicy{
		ExportDestinationPolicy: &v1pb.QueryDataPolicy_ExportDestinationPolicy{AllowedDestinations: []v1pb.ExportDestination{v1pb.ExportDestination_DOWNLOAD}},
	}}}
	require.NoError(t, validateQueryDataPolicyResource(storepb.Policy_WORKSPACE, policy))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(validateQueryDataPolicyResource(storepb.Policy_PROJECT, policy)))
	require.NoError(t, validateQueryDataPolicyResource(storepb.Policy_PROJECT, &v1pb.Policy{Policy: &v1pb.Policy_QueryDataPolicy{QueryDataPolicy: &v1pb.QueryDataPolicy{MaximumResultRows: 10}}}))
}
