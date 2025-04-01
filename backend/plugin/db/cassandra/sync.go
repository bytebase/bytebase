package cassandra

import (
	"context"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/db"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// https://docs.datastax.com/en/cql/hcd/reference/system-virtual-tables/system-keyspace-tables.html
func isSystemDatabase(database string) bool {
	switch database {
	case
		"dse_security",
		"system_traces",
		"system_auth",
		"system_distributed",
		"system_schema",
		"system":
		return true
	default:
		return false
	}
}

func (d *Driver) SyncInstance(ctx context.Context) (*db.InstanceMetadata, error) {
	version, err := d.getVersion(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get version")
	}
	databases, err := d.getDatabases(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get databases")
	}

	var filteredDatabases []*storepb.DatabaseSchemaMetadata
	for _, database := range databases {
		if isSystemDatabase(database.Name) {
			continue
		}
		filteredDatabases = append(filteredDatabases, database)
	}

	return &db.InstanceMetadata{
		Version:   version,
		Databases: filteredDatabases,
	}, nil
}

func (d *Driver) SyncDBSchema(ctx context.Context) (*storepb.DatabaseSchemaMetadata, error) {
	return nil, nil
}

func (d *Driver) getVersion(ctx context.Context) (string, error) {
	var version string
	if err := d.session.Query("SELECT release_version FROM system.local").WithContext(ctx).Scan(&version); err != nil {
		return "", errors.Wrapf(err, "failed to query")
	}
	return version, nil
}

func (d *Driver) getDatabases(ctx context.Context) ([]*storepb.DatabaseSchemaMetadata, error) {
	scanner := d.session.Query("SELECT keyspace_name FROM system_schema.keyspaces").Iter().Scanner()

	var databases []*storepb.DatabaseSchemaMetadata
	for scanner.Next() {
		var database storepb.DatabaseSchemaMetadata
		if err := scanner.Scan(&database.Name); err != nil {
			return nil, errors.Wrapf(err, "failed to scan")
		}
		databases = append(databases, &database)
	}

	if err := scanner.Err(); err != nil {
		return nil, errors.Wrapf(err, "scan error")
	}

	return databases, nil
}
