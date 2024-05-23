package bigquery

import (
	"context"
	"time"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/db"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// SyncInstance syncs the instance.
func (*Driver) SyncInstance(_ context.Context) (*db.InstanceMetadata, error) {
	return nil, nil
}

// SyncDBSchema syncs a single database schema.
func (*Driver) SyncDBSchema(_ context.Context) (*storepb.DatabaseSchemaMetadata, error) {
	return nil, nil
}

// SyncSlowQuery syncs the slow query.
func (*Driver) SyncSlowQuery(_ context.Context, _ time.Time) (map[string]*storepb.SlowQueryStatistics, error) {
	return nil, errors.Errorf("not implemented")
}

// CheckSlowQueryLogEnabled checks if slow query log is enabled.
func (*Driver) CheckSlowQueryLogEnabled(_ context.Context) error {
	return errors.Errorf("not implemented")
}
