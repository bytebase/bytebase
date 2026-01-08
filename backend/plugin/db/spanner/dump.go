package spanner

import (
	"context"
	"fmt"
	"io"

	"cloud.google.com/go/spanner/admin/database/apiv1/databasepb"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// Dump dumps database.
func (d *Driver) Dump(ctx context.Context, out io.Writer, _ *storepb.DatabaseSchemaMetadata) error {
	resp, err := d.dbClient.GetDatabaseDdl(ctx, &databasepb.GetDatabaseDdlRequest{
		Database: getDSN(d.config.DataSource.Host, d.databaseName),
	})
	if err != nil {
		return err
	}

	for _, stmt := range resp.Statements {
		if _, err := io.WriteString(out, fmt.Sprintf("%s;\n", stmt)); err != nil {
			return err
		}
	}

	return nil
}
