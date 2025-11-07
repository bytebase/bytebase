package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"sync"
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

	logger := NewStdoutLogger(StdoutLoggerConfig{
		Writer:            &buf,
		BufferSize:        10,
		HeartbeatInterval: 1 * time.Hour,
		Enabled:           &enabled,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)
	go logger.Run(ctx, &wg)

	err := logger.Log(context.Background(), &storepb.AuditLog{
		Method:   "/bytebase.v1.SQLService/Query",
		Resource: "instances/prod",
		User:     "users/alice@example.com",
		Severity: storepb.AuditLog_INFO,
	})
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)
	cancel()
	wg.Wait()

	output := buf.String()
	require.NotEmpty(t, output)

	var logEntry map[string]any
	err = json.Unmarshal([]byte(output), &logEntry)
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

	logger := NewStdoutLogger(StdoutLoggerConfig{
		Writer:            &buf,
		BufferSize:        10,
		HeartbeatInterval: 1 * time.Hour,
		Enabled:           &enabled,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)
	go logger.Run(ctx, &wg)

	for i := 0; i < 5; i++ {
		err := logger.Log(context.Background(), &storepb.AuditLog{
			Method:   "/test",
			Severity: storepb.AuditLog_INFO,
		})
		require.NoError(t, err)
	}

	time.Sleep(200 * time.Millisecond)
	cancel()
	wg.Wait()

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	assert.Len(t, lines, 5)

	for _, line := range lines {
		var logEntry map[string]any
		err := json.Unmarshal([]byte(line), &logEntry)
		require.NoError(t, err)
	}
}

func TestStdoutLogger_Heartbeat(t *testing.T) {
	var buf bytes.Buffer
	var enabled atomic.Bool
	enabled.Store(true)

	logger := NewStdoutLogger(StdoutLoggerConfig{
		Writer:            &buf,
		BufferSize:        10,
		HeartbeatInterval: 100 * time.Millisecond,
		Enabled:           &enabled,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)
	go logger.Run(ctx, &wg)

	time.Sleep(150 * time.Millisecond)
	cancel()
	wg.Wait()

	output := buf.String()
	assert.Contains(t, output, "audit.heartbeat")
	assert.Contains(t, output, "heartbeat")
}

type blockingWriter struct {
	mu      sync.Mutex
	blocked bool
}

func (w *blockingWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.blocked {
		time.Sleep(1 * time.Second)
	}
	return len(p), nil
}

func TestStdoutLogger_Backpressure_BlocksAndTimesOut(t *testing.T) {
	blocker := &blockingWriter{blocked: true}
	var enabled atomic.Bool
	enabled.Store(true)

	logger := NewStdoutLogger(StdoutLoggerConfig{
		Writer:            blocker,
		BufferSize:        1,
		HeartbeatInterval: 1 * time.Hour,
		Enabled:           &enabled,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)
	go logger.Run(ctx, &wg)

	err := logger.Log(context.Background(), &storepb.AuditLog{Method: "/log1"})
	require.NoError(t, err)

	err = logger.Log(context.Background(), &storepb.AuditLog{Method: "/log2"})
	require.NoError(t, err)

	timeoutCtx, timeoutCancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer timeoutCancel()

	startTime := time.Now()
	err = logger.Log(timeoutCtx, &storepb.AuditLog{Method: "/log3"})
	duration := time.Since(startTime)

	require.Error(t, err, "should have failed due to context timeout")
	assert.ErrorIs(t, err, context.DeadlineExceeded, "error should be context.DeadlineExceeded")
	assert.GreaterOrEqual(t, duration, 50*time.Millisecond, "call should have blocked for at least 50ms")

	_, dropped, _ := logger.Stats()
	assert.Equal(t, int64(1), dropped, "should have 1 dropped event")

	cancel()
	wg.Wait()
}

func TestStdoutLogger_GracefulShutdown(t *testing.T) {
	var buf bytes.Buffer
	var enabled atomic.Bool
	enabled.Store(true)

	logger := NewStdoutLogger(StdoutLoggerConfig{
		Writer:            &buf,
		BufferSize:        100,
		HeartbeatInterval: 1 * time.Hour,
		Enabled:           &enabled,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)
	go logger.Run(ctx, &wg)

	for i := 0; i < 10; i++ {
		err := logger.Log(context.Background(), &storepb.AuditLog{
			Method:   "/test",
			Severity: storepb.AuditLog_INFO,
		})
		require.NoError(t, err)
	}

	cancel()
	wg.Wait()

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	assert.Equal(t, 10, len(lines))

	written, dropped, writeErrors := logger.Stats()
	assert.Equal(t, int64(10), written)
	assert.Equal(t, int64(0), dropped)
	assert.Equal(t, int64(0), writeErrors)
}

func TestStdoutLogger_OptionalFields(t *testing.T) {
	var buf bytes.Buffer
	var enabled atomic.Bool
	enabled.Store(true)

	logger := NewStdoutLogger(StdoutLoggerConfig{
		Writer:            &buf,
		BufferSize:        10,
		HeartbeatInterval: 1 * time.Hour,
		Enabled:           &enabled,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)
	go logger.Run(ctx, &wg)

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

	time.Sleep(100 * time.Millisecond)
	cancel()
	wg.Wait()

	var logEntry map[string]any
	err = json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)

	audit, ok := logEntry["audit"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, float64(250), audit["latency_ms"])
	assert.Equal(t, "192.168.1.100", audit["client_ip"])
	assert.Equal(t, "Mozilla/5.0", audit["user_agent"])
}

func TestNoopAuditLogger_DoesNotHang(t *testing.T) {
	logger := &NoopAuditLogger{}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)
	go logger.Run(ctx, &wg)

	// Log should do nothing
	err := logger.Log(context.Background(), &storepb.AuditLog{Method: "/test"})
	require.NoError(t, err)

	// Cancel and wait - should complete without hanging
	cancel()

	// Use a timeout to detect if it hangs
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success - didn't hang
	case <-time.After(1 * time.Second):
		t.Fatal("NoopAuditLogger.Run() did not call wg.Done() - server would hang on shutdown")
	}
}

func TestConditionalLogger_TogglesLogging(t *testing.T) {
	var buf bytes.Buffer
	var enabled atomic.Bool

	logger := NewConditionalLogger(&enabled, StdoutLoggerConfig{
		Writer:            &buf,
		BufferSize:        10,
		HeartbeatInterval: 1 * time.Hour,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)
	go logger.Run(ctx, &wg)

	event := &storepb.AuditLog{
		Method: "/test",
		User:   "user@example.com",
	}

	// Initially disabled - should not log
	err := logger.Log(context.Background(), event)
	require.NoError(t, err)
	time.Sleep(100 * time.Millisecond)
	assert.Empty(t, buf.String(), "should not log when disabled")

	// Enable - should log
	enabled.Store(true)
	err = logger.Log(context.Background(), event)
	require.NoError(t, err)
	time.Sleep(100 * time.Millisecond)
	assert.NotEmpty(t, buf.String(), "should log when enabled")

	// Verify the log entry
	var logEntry LogOutput
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	require.Len(t, lines, 1, "should have exactly one log entry")
	err = json.Unmarshal([]byte(lines[0]), &logEntry)
	require.NoError(t, err)
	assert.Equal(t, "/test", logEntry.Audit.Method)
	assert.Equal(t, "user@example.com", logEntry.Audit.User)

	// Disable again - should not log new events
	buf.Reset()
	enabled.Store(false)
	err = logger.Log(context.Background(), event)
	require.NoError(t, err)
	time.Sleep(100 * time.Millisecond)
	assert.Empty(t, buf.String(), "should not log after disabled")

	cancel()
	wg.Wait()
}
