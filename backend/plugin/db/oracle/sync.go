package snowflake

import (
	"context"

	"github.com/bytebase/bytebase/backend/plugin/db"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// SyncInstance syncs the instance.
func (*Driver) SyncInstance(_ context.Context) (*db.InstanceMetadata, error) {
	// TODO(d): implement it.
	return nil, nil
}

// SyncDBSchema syncs a single database schema.
func (*Driver) SyncDBSchema(_ context.Context, _ string) (*storepb.DatabaseMetadata, error) {
	// TODO(d): implement it.
	return nil, nil
}
