package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
	spb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/protobuf/types/known/durationpb"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// TestLogAuditToStdoutFormat is a contract test for the structured JSON fields
// emitted to stdout when audit log stdout is enabled. Operators run commands
// like `docker logs <container> | grep '"log_type":"audit"'` — changes to
// these keys or their values are breaking for downstream log pipelines.
func TestLogAuditToStdoutFormat(t *testing.T) {
	a := require.New(t)

	// Redirect slog default to a JSON handler writing into a buffer so the
	// test can inspect the exact structured payload.
	prev := slog.Default()
	t.Cleanup(func() { slog.SetDefault(prev) })

	var buf bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	p := &storepb.AuditLog{
		Parent:   "workspaces/ws-abc",
		Method:   "/bytebase.v1.AuthService/Login",
		Resource: "alice@example.com",
		Severity: storepb.AuditLog_INFO,
		User:     "users/alice@example.com",
		Request:  `{"email":"alice@example.com","web":true}`,
		Response: `{"user":{"name":"users/alice@example.com","email":"alice@example.com"}}`,
		Status:   &spb.Status{Code: 0, Message: ""},
		Latency:  durationpb.New(123_000_000), // 123ms
		RequestMetadata: &storepb.RequestMetadata{
			CallerIp:                "10.0.1.50",
			CallerSuppliedUserAgent: "TestAgent/1.0",
		},
	}

	logAuditToStdout(context.Background(), p)

	// Every line the handler wrote is an independent JSON object.
	lines := bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte{'\n'})
	a.Len(lines, 1, "logAuditToStdout must emit exactly one JSON record")

	var got map[string]any
	a.NoError(json.Unmarshal(lines[0], &got),
		"stdout record must be valid JSON: %s", string(lines[0]))

	// Keys the customer explicitly greps for — these form our public contract.
	// If any of these assertions need to change, it's a breaking change for
	// everyone consuming audit stdout logs.
	a.Equal("audit", got["log_type"], `log_type=="audit" is how operators filter audit lines from application logs`)
	a.Equal("workspaces/ws-abc", got["parent"])
	a.Equal("/bytebase.v1.AuthService/Login", got["method"])
	a.Equal("alice@example.com", got["resource"])
	a.Equal("users/alice@example.com", got["user"])
	a.Equal("INFO", got["severity"])
	a.Equal("10.0.1.50", got["client_ip"])
	a.Equal("TestAgent/1.0", got["user_agent"])
	a.Equal(float64(123), got["latency_ms"], "latency must be exposed in milliseconds")
	a.Equal(`{"email":"alice@example.com","web":true}`, got["request"],
		"request payload must be emitted verbatim (already redacted upstream)")
	a.Equal(`{"user":{"name":"users/alice@example.com","email":"alice@example.com"}}`, got["response"])

	// Status code 0 (OK) should be reported explicitly so downstream pipelines
	// can distinguish successful from failed calls.
	a.Equal(float64(0), got["status_code"])

	// slog standard fields — keep them stable too.
	a.Equal("INFO", got["level"])
	a.Equal("/bytebase.v1.AuthService/Login", got["msg"],
		"slog msg is set to the method so operators can read the log without parsing")

	// Error status round-trip: a failed call must emit status_code and status_message.
	buf.Reset()
	p2 := &storepb.AuditLog{
		Parent:   "workspaces/ws-abc",
		Method:   "/bytebase.v1.AuthService/Login",
		Severity: storepb.AuditLog_INFO,
		Status:   &spb.Status{Code: 16, Message: "invalid email or password"}, // UNAUTHENTICATED
		Latency:  durationpb.New(5_000_000),
	}
	logAuditToStdout(context.Background(), p2)

	var failed map[string]any
	a.NoError(json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &failed))
	a.Equal(float64(16), failed["status_code"], "failed call must carry status_code")
	a.Equal("invalid email or password", failed["status_message"],
		"failed call must carry status_message")
}

// TestAuditRedactsCredentials covers the request/response redactors that
// strip secrets before the audit pipeline serializes payloads. Specifically
// guards against regressions like the Signup password leak and the
// ExchangeToken OIDC/access-token leak that Codex flagged on #20024 — both
// only surfaced once the corresponding audit path was re-enabled by the
// SetAuditWorkspaceID callback.
func TestAuditRedactsCredentials(t *testing.T) {
	a := require.New(t)

	t.Run("LoginRequest redacts password and MFA secrets", func(_ *testing.T) {
		otp := "123456"
		mfa := "mfa-temp-jwt"
		reqStr, err := getRequestString(&v1pb.LoginRequest{
			Email:        "alice@example.com",
			Password:     "hunter2",
			OtpCode:      &otp,
			MfaTempToken: &mfa,
		})
		a.NoError(err)
		a.Contains(reqStr, "alice@example.com", "non-sensitive email stays")
		a.NotContains(reqStr, "hunter2", "plaintext password must not appear")
		a.NotContains(reqStr, "123456", "OTP must not appear")
		a.NotContains(reqStr, "mfa-temp-jwt", "MFA temp token must not appear")
	})

	t.Run("LoginResponse drops token", func(_ *testing.T) {
		respStr, err := getResponseString(&v1pb.LoginResponse{
			Token: "secret-access-token",
			User:  &v1pb.User{Name: "users/alice@example.com"},
		})
		a.NoError(err)
		a.Contains(respStr, "users/alice@example.com", "user info is retained")
		a.NotContains(respStr, "secret-access-token", "access token must not appear")
	})

	t.Run("SignupRequest redacts password", func(_ *testing.T) {
		reqStr, err := getRequestString(&v1pb.SignupRequest{
			Email:    "bob@example.com",
			Password: "signup-password",
			Title:    "bob",
		})
		a.NoError(err)
		a.Contains(reqStr, "bob@example.com")
		a.NotContains(reqStr, "signup-password",
			"plaintext password must not appear in Signup audit")
	})

	t.Run("ExchangeTokenRequest redacts OIDC token", func(_ *testing.T) {
		reqStr, err := getRequestString(&v1pb.ExchangeTokenRequest{
			Token: "oidc.jwt.payload",
			Email: "ci-bot@workload.bytebase.com",
		})
		a.NoError(err)
		a.Contains(reqStr, "ci-bot@workload.bytebase.com",
			"workload email retained for audit correlation")
		a.NotContains(reqStr, "oidc.jwt.payload",
			"external OIDC token must not appear in audit — it can be replayed "+
				"against the original IdP or reveal workload claims")
	})

	t.Run("ExchangeTokenResponse drops issued access token", func(_ *testing.T) {
		respStr, err := getResponseString(&v1pb.ExchangeTokenResponse{
			AccessToken: "issued-bytebase-api-token",
		})
		a.NoError(err)
		a.NotContains(respStr, "issued-bytebase-api-token",
			"issued Bytebase access token must never be logged — anyone with "+
				"audit read access could use it as a valid API token")
	})
}
