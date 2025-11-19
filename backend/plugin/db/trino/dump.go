package trino

import (
	"context"
	"io"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

// Dump dumps the database schema as SQL statements.
func (*Driver) Dump(_ context.Context, out io.Writer, dbMetadata *storepb.DatabaseSchemaMetadata) error {
	text, err := schema.GetDatabaseDefinition(storepb.Engine_TRINO, schema.GetDefinitionContext{}, dbMetadata)
	if err != nil {
		return errors.Wrapf(err, "failed to get database definition")
	}

	_, err = out.Write([]byte(text))
	return err
}
