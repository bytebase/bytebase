package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/durationpb"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func TestStdoutLogger_BasicWrite(t *testing.T) {
	var buf bytes.Buffer
	var enabled atomic.Bool
	enabled.Store(true)

	logger := NewStdoutLogger(&buf, &enabled)

	err := logger.Log(context.Background(), &storepb.AuditLog{
		Method:   "/bytebase.v1.SQLService/Query",
		Resource: "instances/prod",
		User:     "users/alice@example.com",
		Severity: storepb.AuditLog_INFO,
	})
	require.NoError(t, err)

	output := buf.String()
	require.NotEmpty(t, output)

	var logEntry map[string]any
	err = json.Unmarshal([]byte(strings.TrimSpace(output)), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "INFO", logEntry["level"])
	assert.Equal(t, "/bytebase.v1.SQLService/Query", logEntry["message"])

	audit, ok := logEntry["audit"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "/bytebase.v1.SQLService/Query", audit["method"])
	assert.Equal(t, "instances/prod", audit["resource"])
	assert.Equal(t, "users/alice@example.com", audit["user"])
}

func TestStdoutLogger_MultipleEvents(t *testing.T) {
	var buf bytes.Buffer
	var enabled atomic.Bool
	enabled.Store(true)

	logger := NewStdoutLogger(&buf, &enabled)

	for i := 0; i < 5; i++ {
		err := logger.Log(context.Background(), &storepb.AuditLog{
			Method:   "/test",
			Severity: storepb.AuditLog_INFO,
		})
		require.NoError(t, err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	assert.Len(t, lines, 5)

	for _, line := range lines {
		var logEntry map[string]any
		err := json.Unmarshal([]byte(line), &logEntry)
		require.NoError(t, err)
	}
}

func TestStdoutLogger_OptionalFields(t *testing.T) {
	var buf bytes.Buffer
	var enabled atomic.Bool
	enabled.Store(true)

	logger := NewStdoutLogger(&buf, &enabled)

	err := logger.Log(context.Background(), &storepb.AuditLog{
		Method:   "/test",
		Severity: storepb.AuditLog_INFO,
		Latency:  durationpb.New(250 * time.Millisecond),
		RequestMetadata: &storepb.RequestMetadata{
			CallerIp:                "192.168.1.100",
			CallerSuppliedUserAgent: "Mozilla/5.0",
		},
	})
	require.NoError(t, err)

	var logEntry map[string]any
	err = json.Unmarshal([]byte(strings.TrimSpace(buf.String())), &logEntry)
	require.NoError(t, err)

	audit, ok := logEntry["audit"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, float64(250), audit["latency_ms"])
	assert.Equal(t, "192.168.1.100", audit["client_ip"])
	assert.Equal(t, "Mozilla/5.0", audit["user_agent"])
}

func TestStdoutLogger_DisabledDoesNotLog(t *testing.T) {
	var buf bytes.Buffer
	var enabled atomic.Bool
	enabled.Store(false) // disabled

	logger := NewStdoutLogger(&buf, &enabled)

	err := logger.Log(context.Background(), &storepb.AuditLog{
		Method: "/test",
		User:   "user@example.com",
	})
	require.NoError(t, err)

	assert.Empty(t, buf.String(), "should not log when disabled")
}

func TestStdoutLogger_RuntimeToggle(t *testing.T) {
	var buf bytes.Buffer
	var enabled atomic.Bool
	enabled.Store(false) // start disabled

	logger := NewStdoutLogger(&buf, &enabled)

	event := &storepb.AuditLog{
		Method: "/test",
		User:   "user@example.com",
	}

	// Log when disabled - should not write
	err := logger.Log(context.Background(), event)
	require.NoError(t, err)

	// Enable and log - should write
	enabled.Store(true)
	err = logger.Log(context.Background(), event)
	require.NoError(t, err)

	// Disable and log - should not write
	enabled.Store(false)
	err = logger.Log(context.Background(), event)
	require.NoError(t, err)

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	// Should have exactly 1 log entry (when enabled)
	assert.Len(t, lines, 1, "should have exactly one log entry")

	var logEntry LogOutput
	err = json.Unmarshal([]byte(lines[0]), &logEntry)
	require.NoError(t, err)
	assert.Equal(t, "/test", logEntry.Audit.Method)
	assert.Equal(t, "user@example.com", logEntry.Audit.User)
}

func TestNoopAuditLogger_DoesNothing(t *testing.T) {
	logger := &NoopAuditLogger{}

	// Should not panic or error
	err := logger.Log(context.Background(), &storepb.AuditLog{Method: "/test"})
	require.NoError(t, err)
}
