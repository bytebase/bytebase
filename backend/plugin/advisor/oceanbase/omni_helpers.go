package oceanbase

import (
	"github.com/bytebase/omni/oceanbase/ast"

	oceanbaseparser "github.com/bytebase/bytebase/backend/plugin/parser/oceanbase"
)

func omniLine(baseLine int, text string, loc ast.Loc) int {
	if loc.Start < 0 {
		return baseLine + 1
	}
	return baseLine + int(oceanbaseparser.ByteOffsetToRunePosition(text, loc.Start).Line)
}

func omniTableSchemaAndName(table *ast.TableRef) (string, string) {
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
