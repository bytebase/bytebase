package pg

import (
	"context"
	"io"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/schema"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// Dump dumps the database.
func (*Driver) Dump(_ context.Context, out io.Writer, metadata *storepb.DatabaseSchemaMetadata) error {
	text, err := schema.GetDatabaseDefinition(storepb.Engine_POSTGRES, schema.GetDefinitionContext{
		SkipBackupSchema: true,
		PrintHeader:      true,
	}, metadata)
	if err != nil {
		return errors.Wrapf(err, "failed to get database definition")
	}

	_, err = out.Write([]byte(text))
	return err
}
