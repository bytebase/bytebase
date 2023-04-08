package spanner

import (
	"context"
	"fmt"
	"io"

	"cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
	"github.com/pkg/errors"
)

// Dump dumps the database.
func (d *Driver) Dump(ctx context.Context, out io.Writer, schemaOnly bool) (string, error) {
	if !schemaOnly {
		return "", errors.New("Dump can only dump schemas")
	}
	instance, err := d.SyncInstance(ctx)
	if err != nil {
		return "", err
	}
	var dumpableDbNames []string
	if d.databaseName != "" {
		exist := false
		for _, db := range instance.Databases {
			if db.Name == d.databaseName {
				exist = true
				break
			}
		}
		if !exist {
			return "", errors.Errorf("database %q not found", d.databaseName)
		}
		dumpableDbNames = []string{d.databaseName}
	} else {
		for _, db := range instance.Databases {
			dumpableDbNames = append(dumpableDbNames, db.Name)
		}
	}
	for _, db := range dumpableDbNames {
		resp, err := d.dbClient.GetDatabaseDdl(ctx, &databasepb.GetDatabaseDdlRequest{
			Database: getDSN(d.config.Host, db),
		})
		if err != nil {
			return "", err
		}
		for _, stmt := range resp.Statements {
			if _, err := io.WriteString(out, fmt.Sprintf("%s;\n", stmt)); err != nil {
				return "", err
			}
		}
	}

	return "", nil
}

// Restore restores a database.
func (*Driver) Restore(_ context.Context, _ io.Reader) error {
	panic("not implemented")
}
