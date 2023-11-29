package tidb

import (
	"strings"
	"testing"

	tidbparser "github.com/pingcap/tidb/pkg/parser"
	tidbast "github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

type setLineTestData struct {
	statement          string
	columnLineList     []int
	constraintLineList []int
}

func TestMySQLCreateTableSetLine(t *testing.T) {
	tests := []setLineTestData{
		{
			statement:          "CREATE TABLE t as select * from t1",
			columnLineList:     []int{},
			constraintLineList: []int{},
		},
		{
			statement:          "CREATE TABLE t like t1",
			columnLineList:     []int{},
			constraintLineList: []int{},
		},
		{
			statement: `-- this is the first line.
			CREATE TABLE t(
				a int, B int,
				C int,
				D int NOT NULL,
				INDEX (a),
				KEY (a),
				CONSTRAINT unique_a UNIQUE (a),
				UNIQUE (b, c),
				PRIMARY KEY (d),CHECK (a > 0),

				-- it's a comment.
				FOREIGN KEY (a, b, c) REFERENCES t1(a, b, c)
				



			)
			`,
			columnLineList:     []int{3, 3, 4, 5},
			constraintLineList: []int{6, 7, 8, 9, 10, 10, 13},
		},
		{
			// test for Windows.
			statement: "\r\n" +
				"CREATE TABLE t(" + "\r\n" +
				"a int, B int," + "\r\n" +
				"C int," + "\r\n" +
				`D int NOT NULL,` + "\r\n" +
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
			statement: `-- this is the first line.
			CREATE TABLE t(
				a int PRIMARY KEY,
				b int CHECK(b>1), c int UNIQUE
			)
			`,
			columnLineList:     []int{3, 4, 4},
			constraintLineList: []int{},
		},
		{
			statement: `-- complex example
			CREATE TABLE t(
				a int PRIMARY KEY,
				name varchar(255) DEFAULT 'UNIQUE on (a, b, c)(',
				UNIQUE(a, name),
				UNIQUE(name)
			)
			`,
			columnLineList:     []int{3, 4},
			constraintLineList: []int{5, 6},
		},
	}

	for _, test := range tests {
		p := tidbparser.New()
		p.EnableWindowFunc(true)
		nodeList, _, err := p.Parse(test.statement, "", "")
		require.NoError(t, err)
		require.Len(t, nodeList, 1)
		node, ok := nodeList[0].(*tidbast.CreateTableStmt)
		require.True(t, ok)
		require.Equal(t, len(test.columnLineList), len(node.Cols))
		require.Equal(t, len(test.constraintLineList), len(node.Constraints))
		node.SetOriginTextPosition(strings.Count(test.statement, "\n") + 1)
		err = SetLineForMySQLCreateTableStmt(node)
		require.NoError(t, err)
		for i, col := range node.Cols {
			require.Equal(t, col.OriginTextPosition(), test.columnLineList[i], i)
		}
		for i, cons := range node.Constraints {
			require.Equal(t, cons.OriginTextPosition(), test.constraintLineList[i], i)
		}
	}
}

func TestTiDBParserError(t *testing.T) {
	_, err := ParseTiDB("SELECT hello TO world;", "", "")
	require.Error(t, err)
	syntaxErr, ok := err.(*base.SyntaxError)
	require.True(t, ok)
	require.Equal(t, 1, syntaxErr.Line)
	require.Equal(t, 15, syntaxErr.Column)
	require.Equal(t, `line 1 column 15 near "TO world;" `, syntaxErr.Message)
}
