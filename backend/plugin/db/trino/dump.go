package trino

import (
	"context"
	"io"

	metadatapb "github.com/bytebase/omni/metadata"
)

// Dump renders the synced metadata as the DDL that recreates it, through the
// omni driver's deparser.
func (d *Driver) Dump(ctx context.Context, out io.Writer, dbMetadata *metadatapb.DatabaseSchemaMetadata) error {
	return d.driver.Dump(ctx, out, dbMetadata)
}
