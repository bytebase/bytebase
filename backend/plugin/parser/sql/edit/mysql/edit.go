// Package mysql provides the MySQL schema edit plugin.
package mysql

import (
	"github.com/bytebase/bytebase/backend/plugin/parser/sql/edit"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ edit.SchemaEditor = (*SchemaEditor)(nil)
)

func init() {
	edit.Register(storepb.Engine_MYSQL, &SchemaEditor{})
}

// SchemaEditor it the editor for MySQL dialect.
type SchemaEditor struct{}
