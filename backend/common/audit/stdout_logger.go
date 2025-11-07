package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// Logger is the interface for audit logging
type Logger interface {
	Log(ctx context.Context, log *storepb.AuditLog) error
}

// LogOutput represents the JSON structure written to stdout
type LogOutput struct {
	Timestamp string       `json:"@timestamp"`
	Level     string       `json:"level"`
	Message   string       `json:"message"`
	Audit     FieldsOutput `json:"audit"`
}

// FieldsOutput contains audit-specific fields
type FieldsOutput struct {
	Parent        string `json:"parent,omitempty"`
	Method        string `json:"method"`
	Resource      string `json:"resource,omitempty"`
	User          string `json:"user,omitempty"`
	StatusCode    int32  `json:"status_code,omitempty"`
	StatusMessage string `json:"status_message,omitempty"`
	LatencyMs     int64  `json:"latency_ms,omitempty"`
	ClientIP      string `json:"client_ip,omitempty"`
	UserAgent     string `json:"user_agent,omitempty"`
}

// StdoutLogger writes audit logs to stdout as structured JSON synchronously.
// Stdout writes are fast (OS-buffered) and don't require async handling.
type StdoutLogger struct {
	writer  io.Writer
	mu      sync.Mutex
	enabled *atomic.Bool
}

// NewStdoutLogger creates a stdout logger
func NewStdoutLogger(writer io.Writer, enabled *atomic.Bool) *StdoutLogger {
	if writer == nil {
		writer = os.Stdout
	}
	return &StdoutLogger{
		writer:  writer,
		enabled: enabled,
	}
}

// Log writes an audit event to stdout as JSON.
// This is a synchronous operation - stdout is OS-buffered and fast.
// Returns error for logging/observability but callers should not fail operations on error.
func (l *StdoutLogger) Log(_ context.Context, log *storepb.AuditLog) error {
	if l.enabled != nil && !l.enabled.Load() {
		return nil
	}

	auditFields := FieldsOutput{
		Parent:   log.Parent,
		Method:   log.Method,
		Resource: log.Resource,
		User:     log.User,
	}

	if log.Status != nil {
		auditFields.StatusCode = log.Status.Code
		auditFields.StatusMessage = log.Status.Message
	}

	if log.Latency != nil {
		auditFields.LatencyMs = log.Latency.AsDuration().Milliseconds()
	}

	if log.RequestMetadata != nil {
		auditFields.ClientIP = log.RequestMetadata.CallerIp
		auditFields.UserAgent = log.RequestMetadata.CallerSuppliedUserAgent
	}

	logEntry := LogOutput{
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Level:     severityToLevel(log.Severity),
		Message:   log.Method,
		Audit:     auditFields,
	}

	// Encode outside lock to minimize mutex contention
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	if err := encoder.Encode(logEntry); err != nil {
		return errors.Wrapf(err, "failed to encode audit log")
	}

	// Lock only for I/O write
	l.mu.Lock()
	defer l.mu.Unlock()

	if _, err := l.writer.Write(buf.Bytes()); err != nil {
		return errors.Wrapf(err, "failed to write audit log")
	}

	return nil
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

// NoopAuditLogger is a no-op implementation of audit logging
type NoopAuditLogger struct{}

// Log does nothing for the no-op logger
func (*NoopAuditLogger) Log(context.Context, *storepb.AuditLog) error {
	return nil
}
