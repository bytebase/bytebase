package obo

import (
	"context"
	"io"

	"github.com/pkg/errors"
)

// Dump the database.
// The returned string is the JSON encoded metadata for the logical dump.
// For MySQL, the payload contains the binlog filename and position when the dump is generated.
func (*Driver) Dump(context.Context, io.Writer, bool) (string, error) {
	return "", errors.New("not implemented")
}

// Restore the database from src, which is a full backup.
func (*Driver) Restore(context.Context, io.Reader) error {
	return errors.New("not implemented")
}
