package oracle

import (
	"context"
	"io"

	"github.com/bytebase/bytebase/backend/plugin/schema"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// Dump dumps the database.
func (*Driver) Dump(_ context.Context, out io.Writer, metadata *storepb.DatabaseSchemaMetadata) error {
	text, err := schema.GetDatabaseDefinition(storepb.Engine_ORACLE, schema.GetDefinitionContext{}, metadata)
	if err != nil {
		return err
	}
	_, _ = out.Write([]byte(text))
	return nil
}
