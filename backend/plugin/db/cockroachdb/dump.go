package cockroachdb

import (
	"context"
	"io"
)

// Dump dumps the database.
func (driver *Driver) Dump(ctx context.Context, out io.Writer) (string, error) {
	return "", nil
}
