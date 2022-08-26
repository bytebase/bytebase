package pg

import (
	"fmt"

	"github.com/bytebase/bytebase/plugin/parser"
	"github.com/bytebase/bytebase/plugin/schemadiff"
)

func init() {
	schemadiff.Register(parser.Postgres, &postgreSQLDiffer{})
}

// todo docstring
type postgreSQLDiffer struct{}

// todo docstring
func (d *postgreSQLDiffer) AddColumn(tableName, columnName, columnType string) string {
	return fmt.Sprintf(`ALTER TABLE %q ADD COLUMN %q %s;`, tableName, columnName, columnType)
}

// todo docstring
func (d *postgreSQLDiffer) DropColumn(tableName, columnName string) string {
	return fmt.Sprintf(`ALTER TABLE %q DROP COLUMN %q;`, tableName, columnName)
}
