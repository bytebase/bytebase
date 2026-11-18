package mssql

import (
	"context"
	"io"

	metadatapb "github.com/bytebase/omni/metadata"
)

// Dump writes the database's DDL. omni's catalog renders the synced metadata,
// where the driver this replaced called Bytebase's schema renderer: a filtered
// index keeps its WHERE predicate and a view definition ends with one `;`.
func (d *Driver) Dump(ctx context.Context, out io.Writer, dbMetadata *metadatapb.DatabaseSchemaMetadata) error {
	return d.impl.Dump(ctx, out, dbMetadata)
}
