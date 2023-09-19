package risingwave

import (
	"context"
	"io"
)

// Dump dumps the database.
// TODO: RisingWave doesn't support pg_dump yet.
func (*Driver) Dump(_ context.Context, _ io.Writer, _ bool) (string, error) {
	return "", nil
}

// Restore restores a database.
// TODO: RisingWave doesn't support pg_dump yet.
func (*Driver) Restore(_ context.Context, _ io.Reader) error {
	return nil
}
