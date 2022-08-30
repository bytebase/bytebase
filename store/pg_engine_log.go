//go:build store.db
// +build store.db

// Log SQL query hitting our metadata db. Useful to spot unnecessary SQLs.
package store

import (
	"context"
	"database/sql"
	"regexp"

	"go.uber.org/zap"

	"github.com/bytebase/bytebase/common/log"
)

// Replace mutiple whitespace characters including /t/n with a single space.
var pattern = regexp.MustCompile(`\s+`)

func cleanQuery(query string) string {
	return pattern.ReplaceAllString(query, " ")
}

// PrepareContext overrides sql.Tx PrepareContext.
func (tx *Tx) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	log.Debug("PrepareContext", zap.String("query", cleanQuery(query)))
	return tx.Tx.PrepareContext(ctx, query)
}

// ExecContext overrides sql.Tx ExecContext.
func (tx *Tx) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	log.Debug("ExecContext", zap.String("query", cleanQuery(query)))
	return tx.Tx.ExecContext(ctx, query, args...)
}

// QueryContext overrides sql.Tx QueryContext.
func (tx *Tx) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	log.Debug("QueryContext", zap.String("query", cleanQuery(query)))
	return tx.Tx.QueryContext(ctx, query, args...)
}

// QueryRowContext overrides sql.Tx QueryRowContext.
func (tx *Tx) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	log.Debug("QueryRowContext", zap.String("query", cleanQuery(query)))
	return tx.Tx.QueryRowContext(ctx, query, args...)
}
