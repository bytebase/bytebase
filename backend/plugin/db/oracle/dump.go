package oracle

import (
	"context"
	"io"

	metadatapb "github.com/bytebase/omni/metadata"
)

// Dump renders the synced schema as DDL. The renderer reads no server, so an
// edit or an omission in the metadata reaches the dump.
func (d *Driver) Dump(ctx context.Context, out io.Writer, metadata *metadatapb.DatabaseSchemaMetadata) error {
	return d.impl.Dump(ctx, out, metadata)
}
