package hive

import (
	"context"
	"time"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/db"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func (d *Driver) SyncInstance(ctx context.Context) (*db.InstanceMetadata, error) {
	var instanceMetadata db.InstanceMetadata
	// version.
	results, err := d.QueryConn(ctx, nil, "SELECT VERSION()", nil)
	if err != nil || len(results) == 0 {
		return nil, errors.Wrap(err, "failed to get version from instance")
	}
	version := results[0].Rows[0].Values[0].GetStringValue()
	// databases.
	results, err = d.QueryConn(ctx, nil, "SHOW DATABASES", nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get databases from instance")
	}
	for _, row := range results[0].Rows {
		instanceMetadata.Databases = append(instanceMetadata.Databases, &storepb.DatabaseSchemaMetadata{
			Name: row.Values[0].GetStringValue(),
		})
	}
	// TODO(tommy): roles
	instanceMetadata.Version = version
	return &instanceMetadata, nil
}

// SyncDBSchema syncs a single database schema.
func (*Driver) SyncDBSchema(_ context.Context) (*storepb.DatabaseSchemaMetadata, error) {
	return nil, errors.Errorf("Not implemeted")
}

func (*Driver) SyncSlowQuery(_ context.Context, _ time.Time) (map[string]*storepb.SlowQueryStatistics, error) {
	return nil, errors.Errorf("Not implemeted")
}

func (*Driver) CheckSlowQueryLogEnabled(_ context.Context) error {
	return errors.Errorf("Not implemeted")
}
