package trino

import (
	"context"
	"io"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// Dump dumps the database.
func (*Driver) Dump(_ context.Context, out io.Writer, dbSchema *storepb.DatabaseSchemaMetadata) error {
	return nil
}
