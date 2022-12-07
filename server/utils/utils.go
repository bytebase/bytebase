// Package utils is a utility library for server.
package utils

import (
	"context"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/plugin/db"
)

// GetLatestSchemaVersion gets the latest schema version for a database.
func GetLatestSchemaVersion(ctx context.Context, driver db.Driver, databaseName string) (string, error) {
	// TODO(d): support semantic versioning.
	limit := 1
	history, err := driver.FindMigrationHistoryList(ctx, &db.MigrationHistoryFind{
		Database: &databaseName,
		Limit:    &limit,
	})
	if err != nil {
		return "", errors.Wrapf(err, "failed to get migration history for database %q", databaseName)
	}
	var schemaVersion string
	if len(history) == 1 {
		schemaVersion = history[0].Version
	}
	return schemaVersion, nil
}
