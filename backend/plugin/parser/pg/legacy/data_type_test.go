package pg

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/sql/ast"
)

func TestEquivalentType(t *testing.T) {
	type testData struct {
		originType string
		matchType  string
	}
	tests := []testData{
		{
			originType: "int",
			matchType:  "INTEGER",
		},
		{
			originType: "int",
			matchType:  "int4",
		},
		{
			originType: "decimal(10, 2)",
			matchType:  "decimal",
		},
		{
			originType: "float4",
			matchType:  "real",
		},
		{
			originType: "float",
			matchType:  "double precision",
		},
	}

	for _, test := range tests {
		stmt := fmt.Sprintf("ALTER TABLE t ADD COLUMN a %s", test.originType)
		nodeList, err := Parse(ParseContext{}, stmt)
		require.NoError(t, err)
		require.Len(t, nodeList, 1)
		node := nodeList[0]
		alterTable, ok := node.(*ast.AlterTableStmt)
		require.True(t, ok)
		require.Len(t, alterTable.AlterItemList, 1)
		addColumn, ok := alterTable.AlterItemList[0].(*ast.AddColumnListStmt)
		require.True(t, ok)
		require.Len(t, addColumn.ColumnList, 1)
		column := addColumn.ColumnList[0]
		require.True(t, column.Type.EquivalentType(test.matchType))
	}
}
