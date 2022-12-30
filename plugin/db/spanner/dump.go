package spanner

import (
	"context"
	"io"
)

// Dump dumps the database.
func (*Driver) Dump(_ context.Context, _ string, _ io.Writer, _ bool) (string, error) {
	panic("not implemented")
}

// Restore restores a database.
func (*Driver) Restore(_ context.Context, _ io.Reader) error {
	panic("not implemented")
}
