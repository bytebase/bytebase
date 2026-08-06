package server

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/testcontainer"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/migrator"
	"github.com/bytebase/bytebase/backend/store"
)

// devEnterpriseLicense is the pre-signed dev-mode Enterprise license used by the
// test suite (see backend/tests/subscription.go). It has no seat claim, so it
// means unlimited seats.
const devEnterpriseLicense = "eyJhbGciOiJSUzI1NiIsImtpZCI6InYxIiwidHlwIjoiSldUIn0.eyJpbnN0YW5jZUNvdW50Ijo5OTksInRyaWFsaW5nIjpmYWxzZSwicGxhbiI6IkVOVEVSUFJJU0UiLCJvcmdOYW1lIjoiYmIiLCJhdWQiOiJiYi5saWNlbnNlIiwiZXhwIjo3OTc0OTc5MjAwLCJpYXQiOjE2NjM2Njc1NjEsImlzcyI6ImJ5dGViYXNlIiwic3ViIjoiMDAwMDEwMDAuIn0.JjYCMeAAMB9FlVeDFLdN3jvFcqtPsbEzaIm1YEDhUrfekthCbIOeX_DB2Bg2OUji3HSX5uDvG9AkK4Gtrc4gLMPI3D5mk3L-6wUKZ0L4REztS47LT4oxVhpqPQayYa9lKJB1YoHaqeMV4Z5FXeOXwuACoELznlwpT6pXo9xXm_I6QwQiO7-zD83XOTO4PRjByc-q3GKQu_64zJMIKiCW0I8a3GvrdSnO7jUuYU1KPmCuk0ZRq3I91m29LTo478BMST59HqCLj1GGuCKtR3SL_376XsZfUUM0iSAur5scg99zNGWRj-sUo05wbAadYx6V6TKaWrBUi_8_0RnJyP5gbA"

// newMetricsTestEcho returns an echo server with the self-host /metrics route
// wired to a real store and license service backed by a fresh PostgreSQL.
func newMetricsTestEcho(t *testing.T) (*echo.Echo, *store.Store, *enterprise.LicenseService) {
	t.Helper()
	ctx := context.Background()
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(ctx) })

	db := container.GetDB()
	require.NoError(t, migrator.MigrateSchema(ctx, db))

	pgURL := fmt.Sprintf("host=%s port=%s user=postgres password=root-password database=postgres",
		container.GetHost(), container.GetPort())
	st, err := store.New(ctx, pgURL, false)
	require.NoError(t, err)
	t.Cleanup(func() { st.Close() })

	licenseService, err := enterprise.NewLicenseService(common.ReleaseModeDev, st, false, "")
	require.NoError(t, err)

	e := echo.New()
	registerMetricsRoute(e, &config.Profile{SaaS: false}, st, licenseService)
	return e, st, licenseService
}

func scrapeMetrics(t *testing.T, e *echo.Echo, wantStatus int) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	require.Equal(t, wantStatus, rec.Code, rec.Body.String())
	return rec.Body.String()
}

func TestCollisionMetricsLicenseSeats(t *testing.T) {
	ctx := context.Background()
	e, st, licenseService := newMetricsTestEcho(t)

	// No workspace yet: no seats, Free-plan limit.
	body := scrapeMetrics(t, e, http.StatusOK)
	require.Contains(t, body, "bytebase_license_seats_used 0")
	require.Contains(t, body, "bytebase_license_seats_limit 20")

	_, err := st.GetDB().ExecContext(ctx, `
		INSERT INTO workspace (resource_id) VALUES ('ws-test');
		INSERT INTO setting (name, workspace, value) VALUES ('SYSTEM', 'ws-test', '{}'::jsonb);
		INSERT INTO principal (name, email, password_hash, deleted) VALUES
			('alice', 'alice@example.com', 'x', FALSE),
			('bob', 'bob@example.com', 'x', FALSE),
			('carol', 'carol@example.com', 'x', TRUE),
			('eve', 'eve@example.com', 'x', FALSE);
		INSERT INTO user_group (id, workspace, email, name, description, payload) VALUES
			('group-1', 'ws-test', 'eng@example.com', 'Eng', '', '{"members":[{"member":"users/bob@example.com","role":"MEMBER"},{"member":"users/dave@example.com","role":"MEMBER"}]}'::jsonb);
	`)
	require.NoError(t, err)

	policyPayload, err := protojson.Marshal(&storepb.IamPolicy{
		Bindings: []*storepb.Binding{
			{
				Role: "roles/workspaceAdmin",
				Members: []string{
					// Direct end user.
					"users/alice@example.com",
					// Deleted principal: does not occupy a seat.
					"users/carol@example.com",
					// Pending invite: no principal yet, still occupies a seat.
					"users/pending@example.com",
					// Non-user identities: never occupy seats.
					"serviceAccounts/ci@service.bytebase.com",
					"workloadIdentities/bot@workload.bytebase.com",
					// Group expansion: bob (principal) and dave (pending invite).
					"groups/eng@example.com",
				},
			},
		},
	})
	require.NoError(t, err)
	_, err = st.GetDB().ExecContext(ctx, `
		INSERT INTO policy (workspace, resource_type, resource, type, payload, inherit_from_parent, enforce)
		VALUES ('ws-test', 'WORKSPACE', 'workspaces/ws-test', 'IAM', $1, FALSE, TRUE)
	`, string(policyPayload))
	require.NoError(t, err)

	// Alice (direct), pending (direct), bob (group), dave (group) = 4 seats.
	body = scrapeMetrics(t, e, http.StatusOK)
	require.Contains(t, body, "bytebase_license_seats_used 4")
	require.Contains(t, body, "bytebase_license_seats_limit 20")

	// A colliding group email in another workspace must not leak into this
	// workspace's count. Keep the second workspace deleted so GetWorkspaceID
	// continues to select ws-test.
	otherPolicyPayload, err := protojson.Marshal(&storepb.IamPolicy{
		Bindings: []*storepb.Binding{{Role: "roles/workspaceAdmin", Members: []string{"groups/eng@example.com"}}},
	})
	require.NoError(t, err)
	_, err = st.GetDB().ExecContext(ctx, `
		INSERT INTO workspace (resource_id, deleted) VALUES ('ws-other', TRUE);
		INSERT INTO user_group (id, workspace, email, name, description, payload) VALUES
			('group-other', 'ws-other', 'eng@example.com', 'Other Eng', '', '{"members":[{"member":"users/other@example.com","role":"MEMBER"}]}'::jsonb);
	`)
	require.NoError(t, err)
	_, err = st.GetDB().ExecContext(ctx, `
		INSERT INTO policy (workspace, resource_type, resource, type, payload, inherit_from_parent, enforce)
		VALUES ('ws-other', 'WORKSPACE', 'workspaces/ws-other', 'IAM', $1, FALSE, TRUE)
	`, string(otherPolicyPayload))
	require.NoError(t, err)
	body = scrapeMetrics(t, e, http.StatusOK)
	require.Contains(t, body, "bytebase_license_seats_used 4")

	// An Enterprise license without a seat claim means unlimited seats (+Inf).
	require.NoError(t, licenseService.StoreLicense(ctx, "ws-test", devEnterpriseLicense))
	body = scrapeMetrics(t, e, http.StatusOK)
	require.Contains(t, body, "bytebase_license_seats_used 4")
	require.Contains(t, body, "bytebase_license_seats_limit +Inf")

	// A failing metadata read must fail the scrape instead of emitting
	// zero, stale, or missing gauges.
	require.NoError(t, st.Close())
	scrapeMetrics(t, e, http.StatusInternalServerError)
}

func TestMetricsLicenseSeatsAllUsers(t *testing.T) {
	ctx := context.Background()
	e, st, _ := newMetricsTestEcho(t)

	_, err := st.GetDB().ExecContext(ctx, `
		INSERT INTO workspace (resource_id) VALUES ('ws-allusers');
		INSERT INTO principal (name, email, password_hash, deleted) VALUES
			('p1', 'p1@example.com', 'x', FALSE),
			('p2', 'p2@example.com', 'x', FALSE),
			('p3', 'p3@example.com', 'x', TRUE);
	`)
	require.NoError(t, err)
	policyPayload, err := protojson.Marshal(&storepb.IamPolicy{
		Bindings: []*storepb.Binding{{Role: "roles/workspaceMember", Members: []string{"allUsers"}}},
	})
	require.NoError(t, err)
	_, err = st.GetDB().ExecContext(ctx, `
		INSERT INTO policy (workspace, resource_type, resource, type, payload, inherit_from_parent, enforce)
		VALUES ('ws-allusers', 'WORKSPACE', 'workspaces/ws-allusers', 'IAM', $1, FALSE, TRUE)
	`, string(policyPayload))
	require.NoError(t, err)

	// allUsers means every active principal occupies a seat.
	body := scrapeMetrics(t, e, http.StatusOK)
	require.Contains(t, body, "bytebase_license_seats_used 2")
	require.Contains(t, body, "bytebase_license_seats_limit 20")
}
