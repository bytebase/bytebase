package dbauth

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

// Configure applies metadata database authentication settings to pgxConfig.
func Configure(_ context.Context, _ *pgx.ConnConfig) ([]stdlib.OptionOpenDB, error) {
	return nil, nil
}

// IsKeywordValueRuntimeParam reports whether key is a Bytebase metadata DB auth runtime parameter.
func IsKeywordValueRuntimeParam(_ string) bool {
	return false
}
