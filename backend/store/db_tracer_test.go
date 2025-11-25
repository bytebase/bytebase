package store

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
)

func TestExtractQueryOperation(t *testing.T) {
	tests := []struct {
		name     string
		sql      string
		expected string
	}{
		{"SELECT query", "SELECT * FROM users", "select"},
		{"INSERT query", "INSERT INTO users VALUES (1)", "insert"},
		{"UPDATE query", "UPDATE users SET name = 'foo'", "update"},
		{"DELETE query", "DELETE FROM users WHERE id = 1", "delete"},
		{"SHOW query", "SHOW max_connections", "select"},
		{"CTE with SELECT", "WITH cte AS (SELECT 1) SELECT * FROM cte", "select"},
		{"CTE with INSERT", "WITH cte AS (SELECT 1) INSERT INTO users VALUES (1)", "insert"},
		{"CTE with UPDATE", "WITH cte AS (SELECT 1) UPDATE users SET name = 'foo'", "update"},
		{"CTE with DELETE", "WITH cte AS (SELECT 1) DELETE FROM users WHERE id = 1", "delete"},
		{"Other query", "CREATE TABLE users (id INT)", "other"},
		{"Empty query", "", "other"},
		{"Whitespace", "   \n\t  ", "other"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractQueryOperation(tt.sql)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMetadataDBTracer_TraceQueryEnd(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		expectedStatus string
	}{
		{"Success", nil, "success"},
		{"Error", errors.New("some error"), "error"},
		{"Context Canceled", context.Canceled, "cancelled"},
		{"Context Deadline Exceeded", context.DeadlineExceeded, "cancelled"},
		{"Wrapped Context Canceled", errors.Join(errors.New("wrapper"), context.Canceled), "cancelled"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracer := &metadataDBTracer{}

			// Start trace
			ctx := context.Background()
			startData := pgx.TraceQueryStartData{
				SQL: "SELECT * FROM users",
			}
			ctx = tracer.TraceQueryStart(ctx, nil, startData)

			// Small delay to ensure measurable duration
			time.Sleep(1 * time.Millisecond)

			// End trace
			endData := pgx.TraceQueryEndData{
				Err: tt.err,
			}

			// We can't directly verify the metric was recorded, but we can verify
			// the function doesn't panic and handles different error types
			assert.NotPanics(t, func() {
				tracer.TraceQueryEnd(ctx, nil, endData)
			})
		})
	}
}

func TestMetadataDBTracer_TraceQueryEnd_MissingContext(t *testing.T) {
	tracer := &metadataDBTracer{}

	// End trace without start - should not panic
	ctx := context.Background()
	endData := pgx.TraceQueryEndData{
		Err: nil,
	}

	assert.NotPanics(t, func() {
		tracer.TraceQueryEnd(ctx, nil, endData)
	})
}
