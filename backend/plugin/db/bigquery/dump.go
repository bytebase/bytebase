package bigquery

import (
	"context"
	"io"

	metadatapb "github.com/bytebase/omni/metadata"
)

// Dump dumps the database.
func (*Driver) Dump(_ context.Context, _ io.Writer, _ *metadatapb.DatabaseSchemaMetadata) error {
	return nil
}
