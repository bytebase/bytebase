package risingwave

import (
	"context"
	"io"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// Dump dumps the database.
// TODO: RisingWave doesn't support pg_dump yet.
func (*Driver) Dump(_ context.Context, _ io.Writer, _ *storepb.DatabaseSchemaMetadata) error {
	return nil
}
