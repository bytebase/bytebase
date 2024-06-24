package risingwave

import (
	"context"
	"io"
)

// Dump dumps the database.
// TODO: RisingWave doesn't support pg_dump yet.
func (*Driver) Dump(_ context.Context, _ io.Writer) (string, error) {
	return "", nil
}
