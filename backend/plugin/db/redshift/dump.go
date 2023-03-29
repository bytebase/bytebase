package redshift

import (
	"context"
	"io"
)

// Dump and restore
// Dump the database, if dbName is empty, then dump all databases.
// The returned string is the JSON encoded metadata for the logical dump.
// For MySQL, the payload contains the binlog filename and position when the dump is generated.
func (_ *Driver) Dump(ctx context.Context, database string, out io.Writer, schemaOnly bool) (string, error) {
	return "", nil
}

// Restore the database from src, which is a full backup.
func (_ *Driver) Restore(ctx context.Context, src io.Reader) error {
	return nil
}
