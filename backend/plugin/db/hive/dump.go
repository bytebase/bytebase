package hive

import (
	"context"
	"io"

	"github.com/pkg/errors"
)

// Dump and restore
// Dump the database.
// The returned string is the JSON encoded metadata for the logical dump.
// For MySQL, the payload contains the binlog filename and position when the dump is generated.
func (*Driver) Dump(_ context.Context, _ io.Writer, _ bool) (string, error) {
	return "", errors.Errorf("Not implemeted")
}

// Restore the database from src, which is a full backup.
func (*Driver) Restore(_ context.Context, _ io.Reader) error {
	return errors.Errorf("Not implemeted")
}
