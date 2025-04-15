package trino

import (
	"context"
	"io"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/schema"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// Dump dumps the database schema as SQL statements.
func (*Driver) Dump(_ context.Context, out io.Writer, dbSchema *storepb.DatabaseSchemaMetadata) error {
	text, err := schema.GetDatabaseDefinition(storepb.Engine_TRINO, schema.GetDefinitionContext{}, dbSchema)
	if err != nil {
		return errors.Wrapf(err, "failed to get database definition")
	}

	_, err = out.Write([]byte(text))
	return err
}
