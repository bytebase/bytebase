package oceanbase

import (
	"github.com/bytebase/omni/mysql/ast"

	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

func omniLine(baseLine int, text string, loc ast.Loc) int {
	if loc.Start < 0 {
		return baseLine + 1
	}
	return baseLine + int(mysqlparser.ByteOffsetToRunePosition(text, loc.Start).Line)
}

func omniTableName(table *ast.TableRef) (string, string) {
	if table == nil {
		return "", ""
	}
	return table.Schema, table.Name
}

func omniColumnHasPrimaryKey(column *ast.ColumnDef) bool {
	if column == nil {
		return false
	}
	for _, constraint := range column.Constraints {
		if constraint.Type == ast.ColConstrPrimaryKey {
			return true
		}
	}
	return false
}
