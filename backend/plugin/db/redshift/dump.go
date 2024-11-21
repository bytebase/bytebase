// Package redshift is the plugin for RedShift driver.
package redshift

import (
	"context"
	"io"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// Dump dumps the database to the writer. But not implemented yet.
func (*Driver) Dump(context.Context, io.Writer, *storepb.DatabaseSchemaMetadata) error {
	return nil
}
