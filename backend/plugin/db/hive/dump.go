package hive

import (
	"context"
	"io"

	"github.com/pkg/errors"
)

func (*Driver) Dump(_ context.Context, _ io.Writer, _ bool) (string, error) {
	return "", errors.Errorf("Not implemeted")
}

// Restore the database from src, which is a full backup.
func (*Driver) Restore(_ context.Context, _ io.Reader) error {
	return errors.Errorf("Not implemeted")
}
