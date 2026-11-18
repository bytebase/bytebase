package mssql

import (
	"context"
	"io"

	metadatapb "github.com/bytebase/omni/metadata"
)

// Dump writes the database's DDL. The rendering is omni's catalog's; the adapter
// passes the synced metadata through.
func (d *Driver) Dump(ctx context.Context, out io.Writer, dbMetadata *metadatapb.DatabaseSchemaMetadata) error {
	return d.impl.Dump(ctx, out, dbMetadata)
}
