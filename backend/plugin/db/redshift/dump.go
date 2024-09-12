// Package redshift is the plugin for RedShift driver.
package redshift

import (
	"context"
	"io"
)

// Dump dumps the database to the writer. But not implemented yet.
func (*Driver) Dump(context.Context, io.Writer) error {
	return nil
}
