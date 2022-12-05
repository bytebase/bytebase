// Package mysql provides the MySQL schema edit plugin.
package mysql

import (
	bbparser "github.com/bytebase/bytebase/plugin/parser"

	"github.com/bytebase/bytebase/plugin/parser/edit"
	// Register pingcap parser driver.
	_ "github.com/pingcap/tidb/types/parser_driver"
)

var (
	_ edit.SchemaEditor = (*SchemaEditor)(nil)
)

func init() {
	edit.Register(bbparser.MySQL, &SchemaEditor{})
}

// SchemaEditor it the editor for MySQL dialect.
type SchemaEditor struct{}
