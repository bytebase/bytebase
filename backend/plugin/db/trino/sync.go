package trino

import (
	"context"

	metadatapb "github.com/bytebase/omni/metadata"

	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

// SyncInstance reads the server version and its catalogs.
func (d *Driver) SyncInstance(ctx context.Context) (*db.InstanceMetadata, error) {
	instance, err := d.driver.SyncInstance(ctx)
	if err != nil {
		return nil, err
	}
	return util.OmniInstanceToMetadata(instance, true), nil
}

// SyncDBSchema reads the catalog the driver connected to, as the omni Syncer
// reads Config.Database.
func (d *Driver) SyncDBSchema(ctx context.Context) (*metadatapb.DatabaseSchemaMetadata, error) {
	return d.driver.SyncDatabase(ctx)
}
