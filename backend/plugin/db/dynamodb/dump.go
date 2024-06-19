package dynamodb

import (
	"context"
	"io"
)

// Dump dumps the database.
func (*Driver) Dump(_ context.Context, _ io.Writer) (string, error) {
	return "", nil
}
