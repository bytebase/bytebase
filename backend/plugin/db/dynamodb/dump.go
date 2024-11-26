package dynamodb

import (
	"context"
	"io"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// Dump dumps the database.
func (*Driver) Dump(_ context.Context, _ io.Writer, _ *storepb.DatabaseSchemaMetadata) error {
	return nil
}
