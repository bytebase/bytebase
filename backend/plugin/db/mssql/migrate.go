package mssql

import (
	"context"
	"errors"

	"github.com/bytebase/bytebase/backend/plugin/db"
)

// FindMigrationHistoryList finds the migration history list and return most recent item first.
func (*Driver) FindMigrationHistoryList(context.Context, *db.MigrationHistoryFind) ([]*db.MigrationHistory, error) {
	return nil, errors.New("unsupported")
}
