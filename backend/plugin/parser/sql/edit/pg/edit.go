// Package pg provides the Postgres schema edit plugin.
package pg

import (
	"github.com/bytebase/bytebase/backend/plugin/parser/sql/edit"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ edit.SchemaEditor = (*SchemaEditor)(nil)
)

func init() {
	edit.Register(storepb.Engine_POSTGRES, &SchemaEditor{})
}

// SchemaEditor it the editor for Postgres dialect.
type SchemaEditor struct{}
