package cassandra

import (
	"context"
	"io"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func (*Driver) Dump(context.Context, io.Writer, *storepb.DatabaseSchemaMetadata) error {
	return nil
}
