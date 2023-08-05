package risingwave

import (
	"context"
	"io"
)

// Dump dumps the database.
// TODO: RisingWave doesn't support pg_dump yet
func (driver *Driver) Dump(ctx context.Context, out io.Writer, schemaOnly bool) (string, error) {
	return "", nil
}

// Restore restores a database.
// TODO: RisingWave doesn't support pg_dump yet
func (driver *Driver) Restore(ctx context.Context, sc io.Reader) error {
	return nil
}
