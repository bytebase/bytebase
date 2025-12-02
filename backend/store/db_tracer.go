package store

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// metadataDBTracer implements pgx.QueryTracer to record query metrics.
type metadataDBTracer struct{}

// queryTracerData stores data passed from TraceQueryStart to TraceQueryEnd.
type queryTracerData struct {
	startTime time.Time
	sql       string
}

// queryTracerCtxKey is the context key for storing query trace data.
type queryTracerCtxKey struct{}

// TraceQueryStart is called at the beginning of Query, QueryRow, and Exec calls.
func (*metadataDBTracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	traceData := queryTracerData{
		startTime: time.Now(),
		sql:       data.SQL,
	}
	return context.WithValue(ctx, queryTracerCtxKey{}, traceData)
}

// TraceQueryEnd is called at the end of Query, QueryRow, and Exec calls.
func (*metadataDBTracer) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryEndData) {
	traceData, ok := ctx.Value(queryTracerCtxKey{}).(queryTracerData)
	if !ok {
		return
	}

	duration := time.Since(traceData.startTime).Seconds()
	operation := extractQueryOperation(traceData.sql)

	status := "success"
	if data.Err != nil {
		if errors.Is(data.Err, context.Canceled) ||
			errors.Is(data.Err, context.DeadlineExceeded) {
			status = "cancelled"
		} else {
			status = "error"
		}
	}

	metadataDBQueryDuration.WithLabelValues(operation, status).Observe(duration)
}

// extractQueryOperation extracts the SQL operation type from a query string.
// This is a lightweight operation extraction for metrics purposes only.
func extractQueryOperation(sql string) string {
	// Normalize: trim and lowercase
	normalized := strings.TrimSpace(strings.ToLower(sql))
	if normalized == "" {
		return "other"
	}

	// Find first word
	firstWord := normalized
	if idx := strings.IndexAny(normalized, " \t\n\r"); idx > 0 {
		firstWord = normalized[:idx]
	}

	switch firstWord {
	case "select", "show":
		return "select"
	case "insert":
		return "insert"
	case "update":
		return "update"
	case "delete":
		return "delete"
	case "with":
		// CTE - look for actual operation
		if strings.Contains(normalized, " insert ") {
			return "insert"
		} else if strings.Contains(normalized, " update ") {
			return "update"
		} else if strings.Contains(normalized, " delete ") {
			return "delete"
		}
		return "select"
	default:
		return "other"
	}
}
