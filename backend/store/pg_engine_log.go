//go:build store.db

// Log SQL query hitting our metadata db. Useful to spot unnecessary SQLs.
package store

import (
	"context"
	"database/sql"
	"log/slog"
	"regexp"
)

// Replace mutiple whitespace characters including /t/n with a single space.
var pattern = regexp.MustCompile(`\s+`)

func cleanQuery(query string) string {
	return pattern.ReplaceAllString(query, " ")
}

// PrepareContext overrides sql.Tx PrepareContext.
func (tx *Tx) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	slog.Debug("PrepareContext", slog.String("query", cleanQuery(query)))
	return tx.Tx.PrepareContext(ctx, query)
}

// ExecContext overrides sql.Tx ExecContext.
func (tx *Tx) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	slog.Debug("ExecContext", slog.String("query", cleanQuery(query)))
	return tx.Tx.ExecContext(ctx, query, args...)
}

// QueryContext overrides sql.Tx QueryContext.
func (tx *Tx) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	slog.Debug("QueryContext", slog.String("query", cleanQuery(query)))
	return tx.Tx.QueryContext(ctx, query, args...)
}

// QueryRowContext overrides sql.Tx QueryRowContext.
func (tx *Tx) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	slog.Debug("QueryRowContext", slog.String("query", cleanQuery(query)))
	return tx.Tx.QueryRowContext(ctx, query, args...)
}
