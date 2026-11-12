package v1

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/testcontainer"
	"github.com/bytebase/bytebase/backend/component/config"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/neon"
	"github.com/bytebase/bytebase/backend/store"
)

func TestNeonVendorConversion(t *testing.T) {
	instance := &v1pb.Instance{Engine: v1pb.Engine_POSTGRES, Vendor: v1pb.Vendor_NEON, DataSources: []*v1pb.DataSource{{Id: "admin", Type: v1pb.DataSourceType_ADMIN, Host: "ep.neon.tech", UseSsl: true, VerifyTlsCertificate: true}}}
	stored, err := convertToStoreInstance("neon", instance)
	require.NoError(t, err)
	require.Equal(t, storepb.Vendor_NEON, stored.Metadata.Vendor)
	require.Equal(t, v1pb.Vendor_NEON, convertToV1Instance(stored, false).Vendor)
	instance.DataSources[0].VerifyTlsCertificate = false
	_, err = convertToStoreInstance("neon", instance)
	require.Error(t, err)
	instance.DataSources = nil
	instance.Engine = v1pb.Engine_MYSQL
	_, err = convertToStoreInstance("neon", instance)
	require.Error(t, err)
}

func TestNeonRegistrationPersistsWithoutSync(t *testing.T) {
	ctx := context.WithValue(context.Background(), common.WorkspaceIDContextKey, "default")
	db, stores, _ := testcontainer.NewMetadataDB(t)
	_, err := db.ExecContext(ctx, `INSERT INTO workspace(resource_id) VALUES ('default')`)
	require.NoError(t, err)
	service := &InstanceService{store: stores, profile: &config.Profile{}, licenseService: newInstanceServiceTestLicenseService(t, stores)}
	request := connect.NewRequest(&v1pb.CreateInstanceRequest{InstanceId: "neon-instance", Instance: &v1pb.Instance{Title: "Neon", Engine: v1pb.Engine_POSTGRES, Vendor: v1pb.Vendor_NEON, DataSources: []*v1pb.DataSource{{Id: "admin", Type: v1pb.DataSourceType_ADMIN, Host: "ep.neon.tech", UseSsl: true, VerifyTlsCertificate: true}}}})
	response, err := service.createInstanceWithSync(ctx, request, false)
	require.NoError(t, err)
	require.Equal(t, v1pb.Vendor_NEON, response.Msg.Vendor)
	instance, err := stores.GetInstance(ctx, &store.FindInstanceMessage{ResourceID: new("neon-instance"), Workspace: "default"})
	require.NoError(t, err)
	require.Equal(t, storepb.Vendor_NEON, instance.Metadata.Vendor)
	require.Nil(t, instance.Metadata.LastSyncTime)
	updated, err := service.UpdateInstance(ctx, connect.NewRequest(&v1pb.UpdateInstanceRequest{Instance: &v1pb.Instance{Name: response.Msg.Name, Vendor: v1pb.Vendor_VENDOR_UNSPECIFIED}, UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"vendor"}}}))
	require.NoError(t, err)
	require.Equal(t, v1pb.Vendor_VENDOR_UNSPECIFIED, updated.Msg.Vendor)
	require.Equal(t, "ep.neon.tech", updated.Msg.DataSources[0].Host)
	require.Equal(t, v1pb.Engine_POSTGRES, updated.Msg.Engine)
}

func TestNeonVendorErrorMapping(t *testing.T) {
	for _, tc := range []struct {
		name   string
		err    error
		code   connect.Code
		reason string
		// Whether the reason is a vendor condition, which names Neon in metadata.
		vendor bool
	}{
		{name: "rejected key", err: &neon.APIError{StatusCode: 401}, code: connect.CodeFailedPrecondition, reason: reasonVendorCredentialInvalid, vendor: true},
		{name: "access denied", err: &neon.APIError{StatusCode: 403}, code: connect.CodeFailedPrecondition, reason: reasonVendorAccessDenied, vendor: true},
		{name: "resource missing", err: &neon.APIError{StatusCode: 404}, code: connect.CodeFailedPrecondition, reason: reasonVendorResourceNotFound, vendor: true},
		{name: "password not revealed", err: &neon.APIError{StatusCode: 412}, code: connect.CodeFailedPrecondition, reason: reasonVendorPasswordRequired, vendor: true},
		{name: "rate limit", err: &neon.APIError{StatusCode: 429}, code: connect.CodeResourceExhausted, reason: reasonVendorRateLimited, vendor: true},
		{name: "outage", err: &neon.APIError{StatusCode: 503}, code: connect.CodeUnavailable, reason: reasonVendorUnavailable, vendor: true},
		{name: "unreachable", err: neon.ErrUnavailable, code: connect.CodeUnavailable, reason: reasonVendorUnavailable, vendor: true},
		{name: "deadline", err: context.DeadlineExceeded, code: connect.CodeDeadlineExceeded, reason: reasonVendorUnavailable, vendor: true},
		{name: "unexpected status", err: &neon.APIError{StatusCode: 422}, code: connect.CodeInternal, reason: reasonVendorResponseInvalid, vendor: true},
		{name: "malformed response", err: errors.Wrap(neon.ErrInvalidResponse, "invalid Neon roles response"), code: connect.CodeInternal, reason: reasonVendorResponseInvalid, vendor: true},
		{name: "password required", err: neon.ErrPasswordRequired, code: connect.CodeFailedPrecondition, reason: reasonVendorPasswordRequired, vendor: true},
		{name: "database gone", err: neon.ErrDatabaseNotFound, code: connect.CodeFailedPrecondition, reason: reasonVendorDatabaseNotFound, vendor: true},
		{name: "role gone", err: neon.ErrRoleNotFound, code: connect.CodeFailedPrecondition, reason: reasonVendorUserNotFound, vendor: true},
		{name: "no endpoint", err: neon.ErrEndpointUnavailable, code: connect.CodeFailedPrecondition, reason: reasonVendorEndpointUnavailable, vendor: true},
		{name: "page token", err: neon.ErrInvalidPageToken, code: connect.CodeInvalidArgument, reason: reasonInvalidPageToken},
		{name: "malformed identifier", err: neon.ErrInvalidArgument, code: connect.CodeInvalidArgument},
		{name: "canceled", err: context.Canceled, code: connect.CodeCanceled, reason: reasonRequestCanceled},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := neonVendorError(tc.err)
			require.Equal(t, tc.code, err.Code())
			require.NotEqual(t, connect.CodePermissionDenied, err.Code())
			require.NotEqual(t, connect.CodeNotFound, err.Code())
			info := statusErrorInfo(t, importStatus(err))
			require.Equal(t, tc.reason, info.Reason)
			if tc.vendor {
				require.Equal(t, "NEON", info.Metadata["vendor"])
			} else {
				require.Empty(t, info.Metadata)
			}
		})
	}
}

func TestNeonRateLimitCarriesRetryDelay(t *testing.T) {
	status := importStatus(neonVendorError(&neon.APIError{StatusCode: 429, RetryAfter: 12 * time.Second}))
	var delay *errdetails.RetryInfo
	for _, detail := range status.Details {
		retry := &errdetails.RetryInfo{}
		if detail.UnmarshalTo(retry) == nil {
			delay = retry
		}
	}
	require.NotNil(t, delay)
	require.Equal(t, int64(12), delay.RetryDelay.Seconds)
}
