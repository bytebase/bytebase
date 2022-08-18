package parser_test

import (
	"testing"

	"github.com/bytebase/bytebase/plugin/parser"
	"github.com/bytebase/bytebase/plugin/parser/ast"
	"github.com/stretchr/testify/require"

	// Register postgres parser driver.
	_ "github.com/bytebase/bytebase/plugin/parser/engine/pg"
)

type setLineTestData struct {
	statement          string
	columnLineList     []int
	constraintLineList []int
}

func TestPGCreateTableSetLine(t *testing.T) {
	tests := []setLineTestData{
		{
			statement: `
			CREATE TABLE t(
				a int, B int,
				C int,
				"D" int NOT NULL,
				CONSTRAINT unique_a UNIQUE (a),
				UNIQUE (b, c),
				PRIMARY KEY (d),CHECK (a > 0),

				FOREIGN KEY (a, b, c) REFERENCES t1(a, b, c)
			)
			`,
			columnLineList:     []int{3, 3, 4, 5},
			constraintLineList: []int{6, 7, 8, 8, 10},
		},
		{
			// test for Windows.
			statement: "\r\n" +
				"CREATE TABLE t(" + "\r\n" +
				"a int, B int," + "\r\n" +
				"C int," + "\r\n" +
				`"D" int NOT NULL,` + "\r\n" +
				"CONSTRAINT unique_a UNIQUE (a)," + "\r\n" +
				"UNIQUE (b, c)," + "\r\n" +
				"PRIMARY KEY (d),CHECK (a > 0)," + "\r\n" +
				"\r\n" +
				"FOREIGN KEY (a, b, c) REFERENCES t1(a, b, c)" + "\r\n" +
				")",
			columnLineList:     []int{3, 3, 4, 5},
			constraintLineList: []int{6, 7, 8, 8, 10},
		},
		{
			statement: `
			CREATE TABLE t(
				a int PRIMARY KEY,
				b int CHECK(b>1), c int UNIQUE
			)
			`,
			columnLineList:     []int{3, 4, 4},
			constraintLineList: []int{},
		},
	}

	for _, test := range tests {
		nodeList, err := parser.Parse(parser.Postgres, parser.Context{}, test.statement)
		require.NoError(t, err)
		require.Len(t, nodeList, 1)
		node, ok := nodeList[0].(*ast.CreateTableStmt)
		require.True(t, ok)
		require.Equal(t, len(test.columnLineList), len(node.ColumnList))
		require.Equal(t, len(test.constraintLineList), len(node.ConstraintList))
		for i, col := range node.ColumnList {
			require.Equal(t, col.Line(), test.columnLineList[i], i)
		}
		for i, cons := range node.ConstraintList {
			require.Equal(t, cons.Line(), test.constraintLineList[i], i)
		}
	}
}
