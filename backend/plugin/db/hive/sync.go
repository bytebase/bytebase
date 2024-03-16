package hive

import (
	"context"
	"time"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/db"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// TODO(tommy): consider a more elegant way to pass HMS's client.
func (d *Driver) SyncInstance(ctx context.Context) (*db.InstanceMetadata, error) {

	var instanceMetadata db.InstanceMetadata
	results, err := d.QueryConn(ctx, nil, "SELECT VERSION()", nil)
	if err != nil || len(results) == 0 {
		return nil, errors.Wrap(err, "failed to get version from instance")
	}
	version := results[0].Rows[0].Values[0].GetStringValue()

	instanceMetadata.Version = version
	return &instanceMetadata, nil
}

// SyncDBSchema syncs a single database schema.
func (*Driver) SyncDBSchema(_ context.Context) (*storepb.DatabaseSchemaMetadata, error) {
	return nil, errors.Errorf("Not implemeted")
}

// Sync slow query logs
// SyncSlowQuery syncs the slow query logs.
// The returned map is keyed by database name, and the value is list of slow query statistics grouped by query fingerprint.
func (*Driver) SyncSlowQuery(_ context.Context, _ time.Time) (map[string]*storepb.SlowQueryStatistics, error) {
	return nil, errors.Errorf("Not implemeted")
}

// CheckSlowQueryLogEnabled checks if the slow query log is enabled.
func (*Driver) CheckSlowQueryLogEnabled(_ context.Context) error {
	return errors.Errorf("Not implemeted")
}
