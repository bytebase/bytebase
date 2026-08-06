package v1

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"reflect"
	"testing"
	"time"
	"unsafe"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/testcontainer"
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/migrator"
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
// The license service is constructed normally (NewLicenseService with the
// embedded dev public key) and then given a test-only RSA keypair so the test
// can mint a license for each plan. The license is stored in the SYSTEM
// setting and loaded through the real subscription path, so the feature gate,
// retention cutoff, filter application, and SQL execution are all exercised.
// The keypair swap uses the same reflect/unsafe access to the unexported
// config field as backend/tests/sql_query_data_source_test.go.
func TestAuditLogRetentionFilteringEndToEnd(t *testing.T) {
	ctx, stores, licenseService := setupAuditLogRetentionTest(t)

	now := time.Now().UTC()
	oldLogTime := now.AddDate(0, 0, -8).Truncate(time.Microsecond)
	newLogTime := now.AddDate(0, 0, -1).Truncate(time.Microsecond)
	seedAuditLog(ctx, t, stores, "old-log", oldLogTime)
	seedAuditLog(ctx, t, stores, "new-log", newLogTime)

	service := NewAuditLogService(stores, licenseService)

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
// ApplyRetentionFilter against real PostgreSQL rows: a row created exactly at
// the cutoff is retained and a row one microsecond earlier is dropped. This
// exercises the same filter wiring SearchAuditLogs uses (ApplyRetentionFilter
// feeding store.SearchAuditLogs) with a fixed, deterministic cutoff.
func TestAuditLogRetentionFilterIncludesExactCutoffRow(t *testing.T) {
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

	var names []string
	for _, l := range convertToAuditLogs(logs) {
		names = append(names, l.Name)
	}
	require.Equal(t, []string{
		fmt.Sprintf("%s/%s%s", testAuditLogParent, common.AuditLogPrefix, "after-cutoff"),
		fmt.Sprintf("%s/%s%s", testAuditLogParent, common.AuditLogPrefix, "at-cutoff"),
	}, names)
}

func setupAuditLogRetentionTest(t *testing.T) (context.Context, *store.Store, *enterprise.LicenseService) {
	t.Helper()
	ctx := context.WithValue(context.Background(), common.WorkspaceIDContextKey, testAuditLogWorkspace)
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(ctx) })

	db := container.GetDB()
	require.NoError(t, migrator.MigrateSchema(ctx, db))
	_, err := db.ExecContext(ctx, `INSERT INTO workspace (resource_id) VALUES ('default')`)
	require.NoError(t, err)

	pgURL := fmt.Sprintf("host=%s port=%s user=postgres password=root-password database=postgres",
		container.GetHost(), container.GetPort())
	stores, err := store.New(ctx, pgURL, false)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, stores.Close()) })

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

// installTestLicenseKeypair replaces the license service's embedded dev public
// key with a test-only RSA keypair so the test can mint a license for any
// plan. The private key matching the embedded dev public key is not checked
// in, and the LicenseService config field is unexported, so this uses the same
// reflect/unsafe access as backend/tests/sql_query_data_source_test.go.
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

// setAuditLogLicense signs a license for the given plan with the test keypair,
// stores it through the real subscription setting, and invalidates the
// subscription cache so the next SearchAuditLogs call reloads it from the
// database.
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

// seedAuditLog inserts an audit_log row directly with a controlled created_at
// and a canonical project-scoped payload.
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
