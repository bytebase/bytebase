package oracle

import (
	"context"

	"github.com/bytebase/bytebase/backend/plugin/db"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// SyncInstance syncs the instance.
func (driver *Driver) SyncInstance(ctx context.Context) (*db.InstanceMetadata, error) {
	var version string
	if err := driver.db.QueryRowContext(ctx, "SELECT BANNER FROM v$version WHERE banner LIKE 'Oracle%'").Scan(&version); err != nil {
		return nil, err
	}

	var databases []*storepb.DatabaseMetadata
	rows, err := driver.db.QueryContext(ctx, "SELECT name FROM v$database")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		database := &storepb.DatabaseMetadata{}
		if err := rows.Scan(&database.Name); err != nil {
			return nil, err
		}
		databases = append(databases, database)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &db.InstanceMetadata{
		Version:   version,
		Databases: databases,
	}, nil
}

// SyncDBSchema syncs a single database schema.
func (*Driver) SyncDBSchema(_ context.Context, database string) (*storepb.DatabaseMetadata, error) {
	// TODO(d): implement it.
	return &storepb.DatabaseMetadata{
		Name: database,
	}, nil
}
