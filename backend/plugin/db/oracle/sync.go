package oracle

import (
	"context"

	metadatapb "github.com/bytebase/omni/metadata"

	"github.com/bytebase/bytebase/backend/plugin/db"
)

// SyncInstance reads the server version and its schemas. Oracle's instance
// metadata carries no roles.
func (d *Driver) SyncInstance(ctx context.Context) (*db.InstanceMetadata, error) {
	instance, err := d.impl.SyncInstance(ctx)
	if err != nil {
		return nil, err
	}
	return &db.InstanceMetadata{
		Version:   instance.Version,
		Databases: instance.Databases,
	}, nil
}

// SyncDBSchema reads the schema the driver was opened on.
func (d *Driver) SyncDBSchema(ctx context.Context) (*metadatapb.DatabaseSchemaMetadata, error) {
	return d.impl.SyncDatabase(ctx)
}
