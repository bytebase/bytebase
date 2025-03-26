package mssql

import (
	"context"
	"io"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/schema"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// Dump dumps the database.
func (*Driver) Dump(_ context.Context, out io.Writer, metadata *storepb.DatabaseSchemaMetadata) error {
	text, err := schema.GetDatabaseDefinition(storepb.Engine_MSSQL, schema.GetDefinitionContext{}, metadata)
	if err != nil {
		return errors.Wrap(err, "failed to get database definition")
	}
	_, err = out.Write([]byte(text))
	return err
}
