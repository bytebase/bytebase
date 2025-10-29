package audit

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// mockLogger creates a logger for testing without full database setup
func mockLogger(config Config, bytebaseID string) (*Logger, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	shutdownCtx, shutdownCancel := context.WithCancel(context.Background())

	l := &Logger{
		service: &Service{
			store: nil, // Tests don't require real database
		},
		bytebaseID:     bytebaseID,
		currentSeq:     0,
		config:         config,
		inputChan:      make(chan *Input, config.InputQueueSize),
		stdoutQueue:    make(chan *storepb.AuditLog, config.StdoutQueueSize),
		dbQueue:        make(chan *storepb.AuditLog, config.DBBatchQueueSize),
		shutdownCtx:    shutdownCtx,
		shutdownCancel: shutdownCancel,
	}

	l.accepting.Store(true)
	l.sequencerWg.Add(1)
	go l.runSequencer()

	return l, nil
}

// Test: Config validation
func TestConfigValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		config  Config
		wantErr string
	}{
		{
			name:   "default config is valid",
			config: DefaultConfig(),
		},
		{
			name: "zero input queue size",
			config: Config{
				InputQueueSize:         0,
				StdoutQueueSize:        100,
				DBBatchQueueSize:       100,
				DBBatchSize:            10,
				DBBatchInterval:        100 * time.Millisecond,
				DBWriteTimeout:         5 * time.Second,
				DBCircuitTimeoutThresh: 3,
				StdoutCircuitErrThresh: 10,
				RecoveryBaseInterval:   30 * time.Second,
				RecoveryMaxInterval:    30 * time.Minute,
			},
			wantErr: "InputQueueSize must be > 0",
		},
		{
			name: "batch size exceeds queue size",
			config: Config{
				InputQueueSize:         100,
				StdoutQueueSize:        100,
				DBBatchQueueSize:       50,
				DBBatchSize:            100, // Exceeds queue size
				DBBatchInterval:        100 * time.Millisecond,
				DBWriteTimeout:         5 * time.Second,
				DBCircuitTimeoutThresh: 3,
				StdoutCircuitErrThresh: 10,
				RecoveryBaseInterval:   30 * time.Second,
				RecoveryMaxInterval:    30 * time.Minute,
			},
			wantErr: "DBBatchSize (100) cannot exceed DBBatchQueueSize (50)",
		},
		{
			name: "negative timeout",
			config: Config{
				InputQueueSize:         100,
				StdoutQueueSize:        100,
				DBBatchQueueSize:       100,
				DBBatchSize:            10,
				DBBatchInterval:        -1 * time.Millisecond,
				DBWriteTimeout:         5 * time.Second,
				DBCircuitTimeoutThresh: 3,
				StdoutCircuitErrThresh: 10,
				RecoveryBaseInterval:   30 * time.Second,
				RecoveryMaxInterval:    30 * time.Minute,
			},
			wantErr: "DBBatchInterval must be > 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.config.Validate()
			if tt.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErr)
			}
		})
	}
}

// Test: Payload validation
func TestValidatePayload(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		payload *storepb.AuditLog
		wantErr string
	}{
		{
			name: "valid small payload",
			payload: &storepb.AuditLog{
				Method:   "GET",
				Resource: "/api/v1/test",
				User:     "users/test@example.com",
				Request:  "{}",
				Response: "{}",
			},
		},
		{
			name:    "nil payload",
			payload: nil,
			wantErr: "payload cannot be nil",
		},
		{
			name: "oversized request field",
			payload: &storepb.AuditLog{
				Method:   "POST",
				Resource: "/api/v1/test",
				User:     "users/test@example.com",
				Request:  strings.Repeat("a", MaxPayloadSize+1),
				Response: "{}",
			},
			wantErr: "request field too large",
		},
		{
			name: "oversized response field",
			payload: &storepb.AuditLog{
				Method:   "GET",
				Resource: "/api/v1/test",
				User:     "users/test@example.com",
				Request:  "{}",
				Response: strings.Repeat("b", MaxPayloadSize+1),
			},
			wantErr: "response field too large",
		},
		{
			name: "total size exceeds max",
			payload: &storepb.AuditLog{
				Method:   strings.Repeat("M", 10000),
				Resource: strings.Repeat("R", 5000),
				User:     strings.Repeat("U", 5000),
				Request:  strings.Repeat("x", MaxPayloadSize),
				Response: strings.Repeat("y", MaxPayloadSize),
			},
			wantErr: "total event size too large",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidatePayload(tt.payload)
			if tt.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErr)
			}
		})
	}
}

// Test: UTF-8 safe truncation
func TestTruncateUTF8(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		maxBytes int
		wantLen  int // Approximate expected length (varies due to UTF-8)
	}{
		{
			name:     "ASCII string under limit",
			input:    "hello world",
			maxBytes: 100,
			wantLen:  11, // No truncation
		},
		{
			name:     "ASCII string over limit",
			input:    strings.Repeat("a", 200),
			maxBytes: 100,
			wantLen:  100, // Truncated + marker
		},
		{
			name:     "UTF-8 multibyte characters",
			input:    strings.Repeat("日本語", 100), // 3 bytes per character
			maxBytes: 100,
			wantLen:  100, // Should preserve character boundaries
		},
		{
			name:     "emoji characters",
			input:    strings.Repeat("🚀", 100), // 4 bytes per emoji
			maxBytes: 100,
			wantLen:  100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := truncateUTF8(tt.input, tt.maxBytes)

			// Verify result is valid UTF-8
			require.True(t, strings.ToValidUTF8(result, "") == result, "result contains invalid UTF-8")

			// Verify length is at most maxBytes
			require.LessOrEqual(t, len(result), tt.maxBytes)

			// If truncated, should have marker
			if len(tt.input) > tt.maxBytes {
				require.Contains(t, result, "... [truncated]")
			}
		})
	}
}

// Test: TruncatePayloadFields
func TestTruncatePayloadFields(t *testing.T) {
	t.Parallel()
	a := require.New(t)

	payload := &storepb.AuditLog{
		Method:   "POST",
		Resource: "/api/v1/test",
		User:     "users/test@example.com",
		Request:  strings.Repeat("x", MaxPayloadSize+100),
		Response: strings.Repeat("y", MaxPayloadSize+200),
	}

	TruncatePayloadFields(payload)

	// Verify fields are truncated
	a.LessOrEqual(len(payload.Request), MaxPayloadSize)
	a.LessOrEqual(len(payload.Response), MaxPayloadSize)

	// Verify truncation markers
	a.Contains(payload.Request, "... [truncated]")
	a.Contains(payload.Response, "... [truncated]")
}

// Test: Invalid config validation
func TestNewLoggerInvalidConfig(t *testing.T) {
	t.Parallel()
	a := require.New(t)

	invalidConfig := Config{
		InputQueueSize: -1, // Invalid
	}

	logger, err := mockLogger(invalidConfig, "test-id")
	a.Error(err)
	a.Nil(logger)
	a.Contains(err.Error(), "InputQueueSize must be > 0")
}

// Note: Full integration test for NewLogger with real store.Store
// will be in backend/tests/audit_integration_test.go

// Test: Sequencer assigns gap-free sequences
func TestSequencerAssignsGapFreeSequences(t *testing.T) {
	t.Parallel()
	a := require.New(t)

	config := DefaultConfig()
	config.InputQueueSize = 200
	config.StdoutQueueSize = 200
	config.DBBatchQueueSize = 200

	logger, err := mockLogger(config, "test-bytebase-id")
	a.NoError(err)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = logger.Shutdown(ctx)
	}()

	// Submit 100 events
	ctx := context.Background()
	for i := 0; i < 100; i++ {
		payload := &storepb.AuditLog{
			Method:   "GET",
			Resource: fmt.Sprintf("/api/v1/test/%d", i),
			User:     "users/test@example.com",
			Severity: storepb.AuditLog_INFO,
		}
		err := logger.Log(ctx, payload)
		a.NoError(err)
	}

	// Drain stdout queue and verify sequences
	time.Sleep(100 * time.Millisecond) // Give sequencer time to process

	sequences := make([]int64, 0, 100)
	for i := 0; i < 100; i++ {
		select {
		case event := <-logger.stdoutQueue:
			sequences = append(sequences, event.SequenceNumber)
		case <-time.After(1 * time.Second):
			t.Fatalf("timeout waiting for event %d, got %d events", i, len(sequences))
		}
	}

	// Verify gap-free: 1, 2, 3, ..., 100
	for i, seq := range sequences {
		a.Equal(int64(i+1), seq, "gap detected at index %d", i)
	}
}

// Test: Sequencer clones payloads
func TestSequencerClonesPayload(t *testing.T) {
	t.Parallel()
	a := require.New(t)

	config := DefaultConfig()
	config.InputQueueSize = 10
	config.StdoutQueueSize = 10
	config.DBBatchQueueSize = 10
	config.DBBatchSize = 5 // Adjust to be less than queue size

	logger, err := mockLogger(config, "test-bytebase-id")
	a.NoError(err)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = logger.Shutdown(ctx)
	}()

	originalPayload := &storepb.AuditLog{
		Method:   "GET",
		Resource: "/api/v1/test",
		User:     "users/test@example.com",
	}

	ctx := context.Background()
	err = logger.Log(ctx, originalPayload)
	a.NoError(err)

	// Wait for processing
	time.Sleep(50 * time.Millisecond)

	// Drain queue
	select {
	case queuedEvent := <-logger.stdoutQueue:
		// Verify sequence was assigned (proves it was cloned and modified)
		a.Equal(int64(1), queuedEvent.SequenceNumber)
		a.NotEmpty(queuedEvent.BytebaseId)

		// Original should be unchanged
		a.Equal(int64(0), originalPayload.SequenceNumber)
		a.Empty(originalPayload.BytebaseId)
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for event")
	}
}

// Test: Backpressure when input queue full
func TestSequencerBackpressure(t *testing.T) {
	t.Parallel()
	a := require.New(t)

	config := DefaultConfig()
	config.InputQueueSize = 5 // Small queue
	config.StdoutQueueSize = 5
	config.DBBatchQueueSize = 5
	config.DBBatchSize = 2 // Adjust to be less than queue size

	logger, err := mockLogger(config, "test-bytebase-id")
	a.NoError(err)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = logger.Shutdown(ctx)
	}()

	ctx := context.Background()
	payload := &storepb.AuditLog{
		Method:   "GET",
		Resource: "/api/v1/test",
		User:     "users/test@example.com",
	}

	// Submit many events rapidly to trigger backpressure
	var (
		mu     sync.Mutex
		errors []error
	)

	g := new(errgroup.Group)
	for i := 0; i < 20; i++ {
		g.Go(func() error {
			if err := logger.Log(ctx, payload); err != nil {
				mu.Lock()
				errors = append(errors, err)
				mu.Unlock()
			}
			return nil // Don't stop on first error - collect all
		})
	}

	_ = g.Wait()

	// At least one should fail with backpressure (either input queue or output queues full)
	hasBackpressureError := false
	for _, err := range errors {
		if strings.Contains(err.Error(), "input queue full") ||
			strings.Contains(err.Error(), "failed to queue event") ||
			strings.Contains(err.Error(), "overloaded") {
			hasBackpressureError = true
			break
		}
	}
	a.True(hasBackpressureError, "expected at least one backpressure error")
}

// Test: Shutdown drains input queue
func TestShutdownDrainsInputQueue(t *testing.T) {
	t.Parallel()
	a := require.New(t)

	logger, err := mockLogger(DefaultConfig(), "test-bytebase-id")
	a.NoError(err)

	// Submit 50 events
	ctx := context.Background()
	for i := 0; i < 50; i++ {
		payload := &storepb.AuditLog{
			Method:   "GET",
			Resource: "/api/v1/test",
			User:     "users/test@example.com",
			Severity: storepb.AuditLog_INFO,
		}
		err := logger.Log(ctx, payload)
		a.NoError(err)
	}

	// Shutdown with generous timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = logger.Shutdown(shutdownCtx)
	a.NoError(err)

	// Verify all 50 events processed
	processedCount := 0
	for {
		select {
		case <-logger.stdoutQueue:
			processedCount++
		default:
			goto done
		}
	}
done:
	a.Equal(50, processedCount, "not all events processed before shutdown")
}

// Test: Concurrent load with gap-free guarantee
func TestSequencerGapFreeUnderConcurrentLoad(t *testing.T) {
	t.Parallel()
	a := require.New(t)

	config := DefaultConfig()
	config.InputQueueSize = 10000
	config.StdoutQueueSize = 10000
	config.DBBatchQueueSize = 10000

	logger, err := mockLogger(config, "test-bytebase-id")
	a.NoError(err)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_ = logger.Shutdown(ctx)
	}()

	// Launch 100 concurrent clients, each sending 100 events
	const clients = 100
	const eventsPerClient = 100
	const totalEvents = clients * eventsPerClient // 10,000

	g := new(errgroup.Group)
	for clientID := 0; clientID < clients; clientID++ {
		clientID := clientID // Capture loop variable
		g.Go(func() error {
			ctx := context.Background()
			for i := 0; i < eventsPerClient; i++ {
				payload := &storepb.AuditLog{
					Method:   "GET",
					Resource: fmt.Sprintf("/api/v1/client/%d/event/%d", clientID, i),
					User:     fmt.Sprintf("users/client%d@example.com", clientID),
					Severity: storepb.AuditLog_INFO,
				}

				if err := logger.Log(ctx, payload); err != nil {
					return err
				}
			}
			return nil
		})
	}

	// Wait for all clients to finish
	err = g.Wait()
	a.NoError(err, "client encountered error")

	// Drain stdout queue and collect sequences
	time.Sleep(500 * time.Millisecond) // Give sequencer time

	sequences := make([]int64, 0, totalEvents)
	for i := 0; i < totalEvents; i++ {
		select {
		case event := <-logger.stdoutQueue:
			sequences = append(sequences, event.SequenceNumber)
		case <-time.After(5 * time.Second):
			t.Fatalf("timeout: only got %d/%d events", len(sequences), totalEvents)
		}
	}

	// Sort sequences (clients submit concurrently, order non-deterministic)
	slices.SortFunc(sequences, func(a, b int64) int {
		if a < b {
			return -1
		}
		if a > b {
			return 1
		}
		return 0
	})

	// Verify gap-free: 1, 2, 3, ..., 10000
	for i, seq := range sequences {
		expected := int64(i + 1)
		a.Equal(expected, seq, "gap detected: expected %d, got %d", expected, seq)
	}

	t.Logf("✅ Gap-free guarantee verified: %d events from %d concurrent clients", totalEvents, clients)
}
