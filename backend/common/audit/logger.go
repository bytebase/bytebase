package audit

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

// Logger manages audit log sequencing and dispatch
// Core guarantee: gap-free sequence numbers (1, 2, 3, ...)
type Logger struct {
	service    *Service
	bytebaseID string // Unique deployment identifier

	// Sequencer state (ONLY accessed by sequencer goroutine - no mutex needed)
	currentSeq int64 // NOT atomic - single writer guarantee

	config Config

	// Channels
	inputChan   chan *Input            // API handlers → Sequencer
	stdoutQueue chan *storepb.AuditLog // Sequencer → Stdout writer (PR #3)
	dbQueue     chan *storepb.AuditLog // Sequencer → DB batcher (PR #3)

	// Shutdown coordination
	accepting      atomic.Bool
	shutdownCtx    context.Context
	shutdownCancel context.CancelFunc
	sequencerWg    sync.WaitGroup
	shutdownOnce   sync.Once
}

// Input represents an audit event submitted by API handlers
type Input struct {
	Ctx     context.Context // Request context for cancellation propagation
	Payload *storepb.AuditLog
	ResChan chan error // Response channel for synchronous error handling
}

// NewLogger creates and starts the audit logger with gap-free sequencing
func NewLogger(s *store.Store, config Config) (*Logger, error) {
	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, common.Wrap(err, common.Invalid)
	}

	// Create service layer for orchestration
	service := NewService(s)

	// Generate unique Bytebase ID with retry (via service layer)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	bytebaseID, err := service.GenerateUniqueBytebaseIDWithRetry(ctx, 5)
	if err != nil {
		return nil, common.Wrapf(err, common.Internal, "failed to generate bytebase_id")
	}

	// Load max sequence from database with retry (via service layer)
	maxSeq, err := service.LoadMaxSequenceWithRetry(ctx, bytebaseID)
	if err != nil {
		return nil, common.Wrapf(err, common.Internal, "failed to load max sequence")
	}

	// Create shutdown context
	shutdownCtx, shutdownCancel := context.WithCancel(context.Background())

	// Create logger
	l := &Logger{
		service:    service,
		bytebaseID: bytebaseID,
		currentSeq: maxSeq,
		config:     config,

		// Create channels
		inputChan:   make(chan *Input, config.InputQueueSize),
		stdoutQueue: make(chan *storepb.AuditLog, config.StdoutQueueSize),
		dbQueue:     make(chan *storepb.AuditLog, config.DBBatchQueueSize),

		// Shutdown coordination
		shutdownCtx:    shutdownCtx,
		shutdownCancel: shutdownCancel,
	}

	l.accepting.Store(true)

	// Start sequencer goroutine
	l.sequencerWg.Add(1)
	go l.runSequencer()

	slog.Info("audit logger started",
		slog.String("bytebase_id", bytebaseID),
		slog.Int64("starting_sequence", maxSeq))

	return l, nil
}

// Log submits an audit event to the sequencer (called by API handlers)
func (l *Logger) Log(ctx context.Context, payload *storepb.AuditLog) error {
	// Check if shutting down
	if !l.accepting.Load() {
		return common.Errorf(common.Internal, "audit logger shutting down")
	}

	// Validate payload (REJECT if oversized)
	if err := ValidatePayload(payload); err != nil {
		return common.Wrapf(err, common.Invalid, "payload validation failed")
	}

	// Create response channel
	resChan := make(chan error, 1)

	// Submit to sequencer (non-blocking with backpressure)
	select {
	case l.inputChan <- &Input{Ctx: ctx, Payload: payload, ResChan: resChan}:
		// Queued successfully
	case <-l.shutdownCtx.Done():
		return common.Errorf(common.Internal, "shutdown in progress")
	default:
		// Input queue full - backpressure
		return common.Errorf(common.SizeExceeded, "audit system overloaded - input queue full")
	}

	// Wait for sequencer response (with timeout)
	select {
	case err := <-resChan:
		return err
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(500 * time.Millisecond):
		return common.Errorf(common.Internal, "audit processing timeout")
	}
}

// runSequencer is the sequencer goroutine (ONLY goroutine that modifies currentSeq)
func (l *Logger) runSequencer() {
	defer l.sequencerWg.Done()

	slog.Info("sequencer started",
		slog.Int64("starting_sequence", l.currentSeq))

	for input := range l.inputChan {
		// Check if caller's context is cancelled before processing
		select {
		case <-input.Ctx.Done():
			// Request cancelled - skip processing, notify caller
			select {
			case input.ResChan <- input.Ctx.Err():
			default:
			}
			continue
		default:
		}

		// Process one audit event
		err := l.processAuditEvent(input.Payload)

		// Send result back to caller (non-blocking in case caller timed out)
		select {
		case input.ResChan <- err:
		default:
			// Caller timed out or cancelled - continue anyway
		}
	}

	slog.Info("sequencer stopped",
		slog.Int64("final_sequence", l.currentSeq))
}

// processAuditEvent implements the gap-free sequence assignment algorithm
func (l *Logger) processAuditEvent(payload *storepb.AuditLog) error {
	// Clone payload to avoid mutating caller's data
	payloadCopy, ok := proto.Clone(payload).(*storepb.AuditLog)
	if !ok {
		return common.Errorf(common.Internal, "failed to clone audit payload")
	}

	// Defensive truncation (should rarely execute since validation rejects oversized)
	TruncatePayloadFields(payloadCopy)

	// Assign next sequence (CRITICAL: Only this goroutine can do this)
	nextSeq := l.currentSeq + 1
	payloadCopy.SequenceNumber = nextSeq
	payloadCopy.BytebaseId = l.bytebaseID

	// Try to dispatch to both destinations (non-blocking)
	stdoutQueued := l.tryQueueStdout(payloadCopy)
	dbQueued := l.tryQueueDB(payloadCopy)

	// CRITICAL: Only increment sequence if at least one succeeded
	if stdoutQueued || dbQueued {
		l.currentSeq = nextSeq // ✅ Commit sequence
		return nil
	}

	// Both queues full - FAIL FAST (no retry at sequencer layer)
	//
	// ARCHITECTURAL DECISION: Retry belongs in PR #3's consumer goroutines, not here.
	//
	// Why fail-fast when internal queues are full?
	// 1. Industry best practice (GitLab, Slack, Datadog): Internal queue overflow = backpressure signal
	// 2. Retry is for EXTERNAL failures (network, 5xx errors) in consumer goroutines
	// 3. Internal queue full indicates system overload - retrying here creates head-of-line blocking
	// 4. Sequencer retry can't help: If queues are full now, they'll stay full (no consumers in PR #2)
	//
	// Where retry SHOULD happen (PR #3):
	// - DB batcher: Retry database writes (network issues, 5xx errors)
	// - Stdout forwarder: Retry GitLab webhook delivery (network issues, 5xx errors)
	// - Both: Circuit breakers after consecutive failures (aligned with GitLab's approach)
	//
	// This aligns with research findings (audit-log-external-delivery-research.md):
	// - "Choose FAIL-FAST When: System Protection - Queue at capacity (backpressure)"
	// - "Retry belongs in PR #3's consumer goroutines when: Network errors, 5xx responses"
	//
	// Gap-free guarantee preserved: sequence was never incremented, so no gap created.
	slog.Warn("both queues full, failing fast",
		slog.Int64("sequence", nextSeq),
		slog.Int("stdout_queue_len", len(l.stdoutQueue)),
		slog.Int("db_queue_len", len(l.dbQueue)))
	return common.Errorf(common.Internal, "both output queues full (backpressure) - seq=%d not assigned", nextSeq)
}

// tryQueueStdout attempts non-blocking queue to stdout writer
func (l *Logger) tryQueueStdout(payload *storepb.AuditLog) bool {
	select {
	case l.stdoutQueue <- payload:
		return true // Success
	case <-l.shutdownCtx.Done():
		return false // Shutdown in progress
	default:
		return false // Queue full
	}
}

// tryQueueDB attempts non-blocking queue to DB batcher
func (l *Logger) tryQueueDB(payload *storepb.AuditLog) bool {
	select {
	case l.dbQueue <- payload:
		return true // Success
	case <-l.shutdownCtx.Done():
		return false // Shutdown in progress
	default:
		return false // Queue full
	}
}

// Shutdown gracefully stops the audit logger
// Follows server.Server.Shutdown pattern: takes context for timeout coordination
func (l *Logger) Shutdown(ctx context.Context) error {
	var shutdownErr error

	l.shutdownOnce.Do(func() {
		slog.Info("audit logger shutdown initiated",
			slog.Int("input_queue", len(l.inputChan)))

		start := time.Now()

		// Phase 1: Stop accepting new events
		l.accepting.Store(false)
		l.shutdownCancel() // Signal all goroutines

		// Phase 2: Close input channel, wait for sequencer to drain
		close(l.inputChan)
		done := make(chan struct{})
		go func() {
			l.sequencerWg.Wait()
			close(done)
		}()

		select {
		case <-done:
			slog.Info("sequencer drained successfully")
		case <-ctx.Done():
			shutdownErr = common.Errorf(common.Internal, "sequencer timeout: %d events not processed", len(l.inputChan))
			slog.Error("sequencer timeout", slog.Int("remaining", len(l.inputChan)))
		}

		elapsed := time.Since(start)
		if shutdownErr != nil {
			slog.Error("shutdown completed with errors",
				slog.Duration("elapsed", elapsed),
				slog.String("error", shutdownErr.Error()))
		} else {
			slog.Info("shutdown completed successfully",
				slog.Duration("elapsed", elapsed),
				slog.Int64("final_sequence", l.currentSeq))
		}
	})

	return shutdownErr
}
