package cassandra

import (
	"context"
	"io"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func (*Driver) Dump(context.Context, io.Writer, *storepb.DatabaseSchemaMetadata) error {
	return nil
}
