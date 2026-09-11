package elasticsearch

import (
	"context"
	"io"

	metadatapb "github.com/bytebase/omni/metadata"
)

// Dump() is not applicable to Elasticsearch.
func (*Driver) Dump(_ context.Context, _ io.Writer, _ *metadatapb.DatabaseSchemaMetadata) error {
	return nil
}
