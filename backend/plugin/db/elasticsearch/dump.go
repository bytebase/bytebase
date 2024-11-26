package elasticsearch

import (
	"context"
	"io"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// Dump() is not applicable to Elasticsearch.
func (*Driver) Dump(_ context.Context, _ io.Writer, _ *storepb.DatabaseSchemaMetadata) error {
	return nil
}
