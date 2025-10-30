package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"sync"
	"sync/atomic"
	"time"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// Logger is the interface for audit logging that includes both logging and running
type Logger interface {
	Log(ctx context.Context, log *storepb.AuditLog) error
	Run(ctx context.Context, wg *sync.WaitGroup)
}

// LogOutput represents the JSON structure written to stdout
type LogOutput struct {
	Timestamp string       `json:"@timestamp"`
	Level     string       `json:"level"`
	Message   string       `json:"message"`
	Audit     FieldsOutput `json:"audit"`
	Stats     *StatsOutput `json:"stats,omitempty"`
}

// FieldsOutput contains audit-specific fields
type FieldsOutput struct {
	Parent        string `json:"parent,omitempty"`
	Method        string `json:"method"`
	Resource      string `json:"resource,omitempty"`
	User          string `json:"user,omitempty"`
	Type          string `json:"type,omitempty"`
	StatusCode    int32  `json:"status_code,omitempty"`
	StatusMessage string `json:"status_message,omitempty"`
	LatencyMs     int64  `json:"latency_ms,omitempty"`
	ClientIP      string `json:"client_ip,omitempty"`
	UserAgent     string `json:"user_agent,omitempty"`
}

// StatsOutput contains logger statistics
type StatsOutput struct {
	Written int64 `json:"written"`
	Dropped int64 `json:"dropped"`
}

// StdoutLogger writes audit logs to stdout as structured JSON
type StdoutLogger struct {
	writer io.Writer
	mu     sync.Mutex

	eventChan chan *storepb.AuditLog

	heartbeatInterval time.Duration

	written atomic.Int64
	dropped atomic.Int64
}

// StdoutLoggerConfig defines configuration
type StdoutLoggerConfig struct {
	Writer            io.Writer
	BufferSize        int
	HeartbeatInterval time.Duration
}

// NewStdoutLogger creates a stdout logger (does not start it)
func NewStdoutLogger(config StdoutLoggerConfig) *StdoutLogger {
	if config.Writer == nil {
		config.Writer = os.Stdout
	}
	if config.BufferSize <= 0 {
		config.BufferSize = 1000
	}
	if config.HeartbeatInterval <= 0 {
		config.HeartbeatInterval = 5 * time.Minute
	}

	return &StdoutLogger{
		writer:            config.Writer,
		eventChan:         make(chan *storepb.AuditLog, config.BufferSize),
		heartbeatInterval: config.HeartbeatInterval,
	}
}

// Log queues an audit event. It blocks if the queue is full,
// applying backpressure to the API handler.
// It returns an error only if the request's context times out or is cancelled.
func (l *StdoutLogger) Log(ctx context.Context, log *storepb.AuditLog) error {
	select {
	case l.eventChan <- log:
		return nil

	case <-ctx.Done():
		l.dropped.Add(1)
		slog.Warn("stdout audit channel full, request timed out, dropping event",
			slog.String("method", log.Method),
			slog.String("user", log.User))
		return ctx.Err()
	}
}

// Run starts the stdout logger goroutine following the standard runner pattern
func (l *StdoutLogger) Run(ctx context.Context, wg *sync.WaitGroup) {
	heartbeatTicker := time.NewTicker(l.heartbeatInterval)
	defer heartbeatTicker.Stop()
	defer l.drainEvents()
	defer wg.Done()

	slog.Info("stdout audit logger started",
		slog.Int("buffer_size", cap(l.eventChan)),
		slog.Duration("heartbeat_interval", l.heartbeatInterval))

	for {
		select {
		case event := <-l.eventChan:
			l.writeEvent(event)

		case <-heartbeatTicker.C:
			l.writeHeartbeat()

		case <-ctx.Done():
			slog.Info("stdout audit logger shutdown initiated",
				slog.Int("queued", len(l.eventChan)))
			return
		}
	}
}

// writeEvent writes a single audit event as JSON
func (l *StdoutLogger) writeEvent(event *storepb.AuditLog) {
	auditFields := FieldsOutput{
		Parent:   event.Parent,
		Method:   event.Method,
		Resource: event.Resource,
		User:     event.User,
	}

	if event.Status != nil {
		auditFields.StatusCode = event.Status.Code
		auditFields.StatusMessage = event.Status.Message
	}

	if event.Latency != nil {
		auditFields.LatencyMs = event.Latency.AsDuration().Milliseconds()
	}

	if event.RequestMetadata != nil {
		auditFields.ClientIP = event.RequestMetadata.CallerIp
		auditFields.UserAgent = event.RequestMetadata.CallerSuppliedUserAgent
	}

	logEntry := LogOutput{
		Timestamp: timestamp(),
		Level:     severityToLevel(event.Severity),
		Message:   event.Method,
		Audit:     auditFields,
	}

	// Encode to buffer first (no lock needed for CPU-intensive encoding)
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	if err := encoder.Encode(logEntry); err != nil {
		slog.Error("failed to encode audit log to JSON",
			slog.String("method", event.Method),
			slog.String("error", err.Error()))
		return
	}

	// Lock only for the I/O write
	l.mu.Lock()
	defer l.mu.Unlock()

	if _, err := l.writer.Write(buf.Bytes()); err != nil {
		slog.Error("failed to write audit log to stdout",
			slog.String("method", event.Method),
			slog.String("error", err.Error()))
		return
	}

	l.written.Add(1)
}

// writeHeartbeat emits a heartbeat log (proves system is alive)
func (l *StdoutLogger) writeHeartbeat() {
	heartbeat := LogOutput{
		Timestamp: timestamp(),
		Level:     "INFO",
		Message:   "audit.heartbeat",
		Audit: FieldsOutput{
			Method: "audit.heartbeat",
			Type:   "heartbeat",
		},
		Stats: &StatsOutput{
			Written: l.written.Load(),
			Dropped: l.dropped.Load(),
		},
	}

	// Encode to buffer first (no lock needed for CPU-intensive encoding)
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	if err := encoder.Encode(heartbeat); err != nil {
		slog.Error("failed to encode heartbeat to JSON",
			slog.String("error", err.Error()))
		return
	}

	// Lock only for the I/O write
	l.mu.Lock()
	defer l.mu.Unlock()

	if _, err := l.writer.Write(buf.Bytes()); err != nil {
		slog.Error("failed to write heartbeat to stdout",
			slog.String("error", err.Error()))
	}
}

// drainEvents attempts to write remaining events during shutdown
func (l *StdoutLogger) drainEvents() {
	timeout := time.After(1 * time.Second)

	for {
		select {
		case event := <-l.eventChan:
			l.writeEvent(event)
		case <-timeout:
			remaining := len(l.eventChan)
			if remaining > 0 {
				slog.Warn("stdout audit logger shutdown with events remaining",
					slog.Int("remaining", remaining),
					slog.Int64("total_written", l.written.Load()),
					slog.Int64("total_dropped", l.dropped.Load()))
			} else {
				slog.Info("stdout audit logger shutdown completed",
					slog.Int64("total_written", l.written.Load()),
					slog.Int64("total_dropped", l.dropped.Load()))
			}
			return
		}
	}
}

// Stats returns current statistics
func (l *StdoutLogger) Stats() (written, dropped int64) {
	return l.written.Load(), l.dropped.Load()
}

// NoopAuditLogger is a no-op implementation of audit logging
type NoopAuditLogger struct{}

// Log does nothing for the no-op logger
func (*NoopAuditLogger) Log(context.Context, *storepb.AuditLog) error {
	return nil
}

// Run does nothing for the no-op logger
func (*NoopAuditLogger) Run(_ context.Context, wg *sync.WaitGroup) {
	wg.Done()
}

// timestamp returns the current UTC timestamp in RFC3339Nano format
func timestamp() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

// severityToLevel converts protobuf severity enum to log level string
func severityToLevel(severity storepb.AuditLog_Severity) string {
	switch severity {
	case storepb.AuditLog_INFO:
		return "INFO"
	case storepb.AuditLog_WARNING:
		return "WARNING"
	case storepb.AuditLog_ERROR:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}
