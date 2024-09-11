package cockroachdb

import (
	"context"
	"io"
)

// Dump dumps the database.
func (*Driver) Dump(context.Context, io.Writer) (string, error) {
	return "", nil
}
