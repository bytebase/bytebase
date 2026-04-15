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
