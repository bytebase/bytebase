package cassandra

import (
	"context"
	"io"

	metadatapb "github.com/bytebase/omni/metadata"
)

func (*Driver) Dump(context.Context, io.Writer, *metadatapb.DatabaseSchemaMetadata) error {
	return nil
}
