package store_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"reflect"
	"testing"
	"time"
	"unsafe"

	"github.com/bytebase/bytebase/backend/common/testcontainer"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"

	apiv1 "github.com/bytebase/bytebase/backend/api/v1"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
)

const (
	testAuditLogWorkspace = "default"
	testAuditLogParent    = "projects/project-a"
)

// TestAuditLogRetentionFilteringEndToEnd drives the real SearchAuditLogs
// handler against a PostgreSQL-backed store under every plan:
//   - FREE: audit log access is gated off (PermissionDenied).
//   - TEAM: rows older than the 7-day retention cutoff are excluded, rows
//     newer than the cutoff are included with canonical resource names.
//   - ENTERPRISE: unlimited retention, so rows older than the cutoff remain
//     visible.
//
// The license is minted with a test keypair and stored through the real
// subscription path, so the feature gate, retention cutoff, filter application
// and SQL execution are all exercised.
func TestAuditLogRetentionFilteringEndToEnd(t *testing.T) {
	// Not parallel: the subtests set the workspace license in turn and read it back,
	// so they share one mutable row and have to run in order.
	ctx, stores, licenseService := setupAuditLogRetentionTest(t)

	now := time.Now().UTC()
	oldLogTime := now.AddDate(0, 0, -8).Truncate(time.Microsecond)
	newLogTime := now.AddDate(0, 0, -1).Truncate(time.Microsecond)
	seedAuditLog(ctx, t, stores, "old-log", oldLogTime)
	seedAuditLog(ctx, t, stores, "new-log", newLogTime)

	service := apiv1.NewAuditLogService(stores, licenseService)

	t.Run("FREE plan has no audit log access", func(t *testing.T) {
		setAuditLogLicense(ctx, t, stores, licenseService, v1pb.PlanType_FREE)

		_, err := service.SearchAuditLogs(ctx, connect.NewRequest(&v1pb.SearchAuditLogsRequest{
			Parent:   "projects/-",
			PageSize: 100,
		}))
		require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
	})

	t.Run("TEAM plan excludes rows older than the retention cutoff", func(t *testing.T) {
		setAuditLogLicense(ctx, t, stores, licenseService, v1pb.PlanType_TEAM)

		resp, err := service.SearchAuditLogs(ctx, connect.NewRequest(&v1pb.SearchAuditLogsRequest{
			Parent:   "projects/-",
			PageSize: 100,
		}))
		require.NoError(t, err)
		require.Empty(t, resp.Msg.NextPageToken)
		require.Len(t, resp.Msg.AuditLogs, 1, "the row older than the 7-day cutoff must be filtered out")

		log := resp.Msg.AuditLogs[0]
		require.Equal(t, fmt.Sprintf("%s/%s%s", testAuditLogParent, common.AuditLogPrefix, "new-log"), log.Name)
		require.Equal(t, testAuditLogParent, log.Resource)
		require.Equal(t, "/bytebase.v1.ProjectService/GetProject", log.Method)
		require.Equal(t, "users/alice@example.com", log.User)
		require.Equal(t, v1pb.AuditLog_INFO, log.Severity)
		require.Equal(t, newLogTime.UnixNano(), log.CreateTime.AsTime().UnixNano())
	})

	t.Run("ENTERPRISE plan keeps rows older than the retention cutoff", func(t *testing.T) {
		setAuditLogLicense(ctx, t, stores, licenseService, v1pb.PlanType_ENTERPRISE)

		resp, err := service.SearchAuditLogs(ctx, connect.NewRequest(&v1pb.SearchAuditLogsRequest{
			Parent:   "projects/-",
			PageSize: 100,
		}))
		require.NoError(t, err)
		require.Len(t, resp.Msg.AuditLogs, 2, "ENTERPRISE plan has unlimited retention")
	})
}

// TestAuditLogRetentionFilterIncludesExactCutoffRow pins the >= boundary of
// ApplyRetentionFilter against real rows: a row created exactly at the cutoff
// is retained and a row one microsecond earlier is dropped.
func TestAuditLogRetentionFilterIncludesExactCutoffRow(t *testing.T) {
	t.Parallel()
	ctx, stores, _ := setupAuditLogRetentionTest(t)

	cutoff := time.Now().UTC().Truncate(time.Microsecond)
	seedAuditLog(ctx, t, stores, "before-cutoff", cutoff.Add(-time.Microsecond))
	seedAuditLog(ctx, t, stores, "at-cutoff", cutoff)
	seedAuditLog(ctx, t, stores, "after-cutoff", cutoff.Add(time.Microsecond))

	logs, err := stores.SearchAuditLogs(ctx, &store.AuditLogFind{
		Workspace: testAuditLogWorkspace,
		FilterQ:   store.ApplyRetentionFilter(nil, &cutoff),
	})
	require.NoError(t, err)

	var resourceIDs []string
	for _, l := range logs {
		resourceIDs = append(resourceIDs, l.ResourceID)
	}
	require.Equal(t, []string{"after-cutoff", "at-cutoff"}, resourceIDs)
}

func setupAuditLogRetentionTest(t *testing.T) (context.Context, *store.Store, *enterprise.LicenseService) {
	t.Helper()
	ctx := context.WithValue(context.Background(), common.WorkspaceIDContextKey, testAuditLogWorkspace)
	_, stores, _ := testcontainer.NewMetadataDB(t)

	_, err := stores.GetDB().ExecContext(ctx, `INSERT INTO workspace (resource_id) VALUES ('default')`)
	require.NoError(t, err)

	// The SYSTEM setting must exist before UpdateLicense can write the license.
	_, err = stores.UpsertSetting(ctx, &store.SettingMessage{
		Name:      storepb.SettingName_SYSTEM,
		Workspace: testAuditLogWorkspace,
		Value:     &storepb.SystemSetting{},
	})
	require.NoError(t, err)

	licenseService, err := enterprise.NewLicenseService(common.ReleaseModeDev, stores, false, "")
	require.NoError(t, err)
	installTestLicenseKeypair(t, licenseService)
	return ctx, stores, licenseService
}

// installTestLicenseKeypair replaces the embedded dev public key with a
// test-only keypair so the test can mint a license for any plan. The matching
// private key is not checked in and the config field is unexported, hence the
// reflect/unsafe access.
func installTestLicenseKeypair(t *testing.T, licenseService *enterprise.LicenseService) {
	t.Helper()
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	field := reflect.ValueOf(licenseService).Elem().FieldByName("config")
	config, ok := reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Interface().(*enterprise.Config)
	require.True(t, ok)
	config.PrivateKey = privateKey
	config.PublicKey = &privateKey.PublicKey
}

// setAuditLogLicense signs a license for the plan, stores it through the real
// subscription setting, and drops the cache so the next call reloads it.
func setAuditLogLicense(ctx context.Context, t *testing.T, stores *store.Store, licenseService *enterprise.LicenseService, plan v1pb.PlanType) {
	t.Helper()
	token, err := licenseService.CreateLicense(&enterprise.LicenseParams{
		Plan:        plan.String(),
		Seats:       10,
		Instances:   10,
		WorkspaceID: testAuditLogWorkspace,
	})
	require.NoError(t, err)
	require.NoError(t, stores.UpdateLicense(ctx, testAuditLogWorkspace, token))
	licenseService.InvalidateCache(testAuditLogWorkspace)
}

// seedAuditLog inserts a row directly with a controlled created_at.
func seedAuditLog(ctx context.Context, t *testing.T, stores *store.Store, resourceID string, createdAt time.Time) {
	t.Helper()
	payload, err := protojson.Marshal(&storepb.AuditLog{
		Parent:   testAuditLogParent,
		Method:   "/bytebase.v1.ProjectService/GetProject",
		Resource: testAuditLogParent,
		User:     "users/alice@example.com",
		Severity: storepb.AuditLog_INFO,
		Request:  `{"name":"projects/project-a"}`,
		Response: `{"name":"projects/project-a"}`,
	})
	require.NoError(t, err)
	_, err = stores.GetDB().ExecContext(ctx, `
		INSERT INTO audit_log (resource_id, workspace, created_at, payload)
		VALUES ($1, $2, $3, $4::jsonb)
	`, resourceID, testAuditLogWorkspace, createdAt, string(payload))
	require.NoError(t, err)
}
