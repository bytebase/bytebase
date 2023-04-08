// Package redshift is the plugin for RedShift driver.
package redshift

import (
	"context"
	"io"

	"github.com/pkg/errors"
)

// Dump dumps the database to the writer. But not implemented yet.
func (*Driver) Dump(context.Context, io.Writer, bool) (string, error) {
	return "", nil
}

// Restore the database from src, which is a full backup.
func (*Driver) Restore(context.Context, io.Reader) error {
	return errors.Errorf("not implemented")
}
