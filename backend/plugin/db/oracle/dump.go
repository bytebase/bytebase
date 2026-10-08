package oracle

import (
	"context"
	"io"

	metadatapb "github.com/bytebase/omni/metadata"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

// Dump dumps the database.
func (*Driver) Dump(_ context.Context, out io.Writer, metadata *metadatapb.DatabaseSchemaMetadata) error {
	text, err := schema.GetDatabaseDefinition(storepb.Engine_ORACLE, schema.GetDefinitionContext{}, metadata)
	if err != nil {
		return err
	}
	_, _ = out.Write([]byte(text))
	return nil
}
