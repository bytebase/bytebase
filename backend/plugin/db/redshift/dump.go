// Package redshift is the plugin for RedShift driver.
package redshift

import (
	"context"
	"io"

	metadatapb "github.com/bytebase/omni/metadata"
)

// Dump dumps the database to the writer. But not implemented yet.
func (*Driver) Dump(context.Context, io.Writer, *metadatapb.DatabaseSchemaMetadata) error {
	return nil
}
