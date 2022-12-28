// Package pg provides the Postgres schema edit plugin.
package pg

import (
	bbparser "github.com/bytebase/bytebase/plugin/parser"

	"github.com/bytebase/bytebase/plugin/parser/edit"
)

var (
	_ edit.SchemaEditor = (*SchemaEditor)(nil)
)

func init() {
	edit.Register(bbparser.Postgres, &SchemaEditor{})
}

// SchemaEditor it the editor for Postgres dialect.
type SchemaEditor struct{}
