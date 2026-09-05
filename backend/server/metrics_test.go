package server

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bytebase/bytebase/backend/common/testpg"

	"github.com/labstack/echo/v5"
	metricpb "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/productmetrics"
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

// devEnterpriseLicense is the pre-signed dev-mode Enterprise license used by the
// test suite (see backend/tests/subscription.go). It has no seat claim, so it
// means unlimited seats.
const devEnterpriseLicense = "eyJhbGciOiJSUzI1NiIsImtpZCI6InYxIiwidHlwIjoiSldUIn0.eyJpbnN0YW5jZUNvdW50Ijo5OTksInRyaWFsaW5nIjpmYWxzZSwicGxhbiI6IkVOVEVSUFJJU0UiLCJvcmdOYW1lIjoiYmIiLCJhdWQiOiJiYi5saWNlbnNlIiwiZXhwIjo3OTc0OTc5MjAwLCJpYXQiOjE2NjM2Njc1NjEsImlzcyI6ImJ5dGViYXNlIiwic3ViIjoiMDAwMDEwMDAuIn0.JjYCMeAAMB9FlVeDFLdN3jvFcqtPsbEzaIm1YEDhUrfekthCbIOeX_DB2Bg2OUji3HSX5uDvG9AkK4Gtrc4gLMPI3D5mk3L-6wUKZ0L4REztS47LT4oxVhpqPQayYa9lKJB1YoHaqeMV4Z5FXeOXwuACoELznlwpT6pXo9xXm_I6QwQiO7-zD83XOTO4PRjByc-q3GKQu_64zJMIKiCW0I8a3GvrdSnO7jUuYU1KPmCuk0ZRq3I91m29LTo478BMST59HqCLj1GGuCKtR3SL_376XsZfUUM0iSAur5scg99zNGWRj-sUo05wbAadYx6V6TKaWrBUi_8_0RnJyP5gbA"

// newMetricsTestEcho returns an echo server with the self-host /metrics route
// wired to a real store and license service backed by a fresh PostgreSQL.
func newMetricsTestEcho(t *testing.T) (*echo.Echo, *store.Store, *enterprise.LicenseService) {
	e, st, licenseService, _ := newMetricsTestEchoWithCollector(t)
	return e, st, licenseService
}

func newMetricsTestEchoWithCollector(t *testing.T) (*echo.Echo, *store.Store, *enterprise.LicenseService, *productmetrics.ProductMetrics) {
	t.Helper()
	_, st, _ := testpg.New(t)

	licenseService, err := enterprise.NewLicenseService(common.ReleaseModeDev, st, false, "")
	require.NoError(t, err)

	e := echo.New()
	metrics := productmetrics.New(st, licenseService)
	registerMetricsRoute(e, &config.Profile{SaaS: false}, metrics)
	return e, st, licenseService, metrics
}

func scrapeMetrics(t *testing.T, e *echo.Echo, wantStatus int) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.RemoteAddr = "127.0.0.1:12345"
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
	require.Contains(t, body, "bytebase_license_instances_used 0")
	require.Contains(t, body, "bytebase_license_instances_limit 10")
	require.Contains(t, body, "bytebase_license_expiry_timestamp_seconds +Inf")

	_, err := st.GetDB().ExecContext(ctx, `
		INSERT INTO workspace (resource_id) VALUES ('ws-test');
		INSERT INTO setting (name, workspace, value) VALUES ('SYSTEM', 'ws-test', '{}'::jsonb);
		INSERT INTO project (resource_id, workspace, name) VALUES ('project-a', 'ws-test', 'Project A');
		INSERT INTO instance (resource_id, workspace, project, deleted) VALUES
			('workspace-instance', 'ws-test', NULL, FALSE),
			('project-instance', 'ws-test', 'project-a', FALSE),
			('deleted-instance', 'ws-test', NULL, TRUE);
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
	require.Contains(t, body, "bytebase_license_instances_used 2")

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
	require.Contains(t, body, "bytebase_license_instances_limit +Inf")

	// A configured but malformed license invalidates the complete scrape.
	require.NoError(t, st.UpdateLicense(ctx, "ws-test", "not-a-license"))
	scrapeMetrics(t, e, http.StatusInternalServerError)
	require.NoError(t, licenseService.StoreLicense(ctx, "ws-test", devEnterpriseLicense))

	// A failing metadata read must fail the scrape instead of emitting
	// zero, stale, or missing gauges.
	require.NoError(t, st.Close())
	scrapeMetrics(t, e, http.StatusInternalServerError)
}

func TestMetricsEventsAreServerLocal(t *testing.T) {
	e1, _, _, metrics1 := newMetricsTestEchoWithCollector(t)
	e2, _, _, _ := newMetricsTestEchoWithCollector(t)
	metrics1.RecordRunnerRun(productmetrics.RunnerPlanCheck, productmetrics.ResultSuccess, time.Second)

	body1 := scrapeMetrics(t, e1, http.StatusOK)
	require.Contains(t, body1, `bytebase_runner_run_duration_seconds_count{result="success",runner="plan_check"} 1`)
	body2 := scrapeMetrics(t, e2, http.StatusOK)
	require.NotContains(t, body2, "bytebase_runner_run_duration_seconds")
	for _, name := range []string{
		"bytebase_license_expiry_timestamp_seconds",
		"bytebase_license_instances_used",
		"bytebase_license_instances_limit",
	} {
		require.Equal(t, metricTextLine(body1, name), metricTextLine(body2, name))
	}

	var wg sync.WaitGroup
	scrapeErr := make(chan string, 8)
	for range 8 {
		wg.Go(func() {
			metrics1.RecordRunnerRun(productmetrics.RunnerPlanCheck, productmetrics.ResultSuccess, time.Millisecond)
			req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
			req.RemoteAddr = "127.0.0.1:12345"
			rec := httptest.NewRecorder()
			e1.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				scrapeErr <- rec.Body.String()
			}
		})
	}
	wg.Wait()
	close(scrapeErr)
	for err := range scrapeErr {
		t.Errorf("concurrent scrape failed: %s", err)
	}

	format := expfmt.NewFormat(expfmt.TypeProtoDelim)
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set("Accept", string(format))
	rec := httptest.NewRecorder()
	e1.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	decoder := expfmt.NewDecoder(rec.Body, format)
	foundNative := false
	for {
		family := &metricpb.MetricFamily{}
		if err := decoder.Decode(family); err != nil {
			require.ErrorIs(t, err, io.EOF)
			break
		}
		if family.GetName() == "bytebase_runner_run_duration_seconds" {
			require.Len(t, family.Metric, 1)
			require.NotNil(t, family.Metric[0].GetHistogram().Schema)
			foundNative = true
		}
	}
	require.True(t, foundNative)
}

func TestMetricsAccessHandler(t *testing.T) {
	tests := []struct {
		name         string
		remoteAccess bool
		remoteAddr   string
		forwardedFor string
		forwarded    string
		wantStatus   int
		wantCalls    int
	}{
		{
			name:       "IPv4 loopback",
			remoteAddr: "127.0.0.1:12345",
			wantStatus: http.StatusNoContent,
			wantCalls:  1,
		},
		{
			name:       "IPv6 loopback",
			remoteAddr: "[::1]:12345",
			wantStatus: http.StatusNoContent,
			wantCalls:  1,
		},
		{
			name:       "IPv4-mapped IPv6 loopback",
			remoteAddr: "[::ffff:127.0.0.1]:12345",
			wantStatus: http.StatusNoContent,
			wantCalls:  1,
		},
		{
			name:       "remote IPv4",
			remoteAddr: "192.0.2.1:12345",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "remote IPv6",
			remoteAddr: "[2001:db8::1]:12345",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "empty remote address",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "malformed remote address",
			remoteAddr: "not-an-address",
			wantStatus: http.StatusNotFound,
		},
		{
			name:         "spoofed forwarding headers",
			remoteAddr:   "192.0.2.1:12345",
			forwardedFor: "127.0.0.1",
			forwarded:    "for=127.0.0.1",
			wantStatus:   http.StatusNotFound,
		},
		{
			name:         "remote access bypasses peer parsing",
			remoteAccess: true,
			remoteAddr:   "not-an-address",
			wantStatus:   http.StatusNoContent,
			wantCalls:    1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			handler := metricsAccessHandler(tt.remoteAccess, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				calls++
				w.WriteHeader(http.StatusNoContent)
			}))
			req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
			req.RemoteAddr = tt.remoteAddr
			req.Header.Set("X-Forwarded-For", tt.forwardedFor)
			req.Header.Set("Forwarded", tt.forwarded)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			require.Equal(t, tt.wantStatus, rec.Code)
			require.Equal(t, tt.wantCalls, calls, "collection handler calls")
		})
	}
}

func metricTextLine(body, name string) string {
	for line := range strings.SplitSeq(body, "\n") {
		if strings.HasPrefix(line, name+" ") {
			return line
		}
	}
	return ""
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
