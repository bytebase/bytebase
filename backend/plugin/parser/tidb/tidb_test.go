package tidb

import (
	"testing"

	tidbparser "github.com/pingcap/tidb/pkg/parser"
	tidbast "github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

type setLineTestData struct {
	statement string
	// firstLine is the 1-based line number where the CREATE TABLE statement starts.
	// This is used to set OriginTextPosition, which the tokenizer uses to calculate
	// absolute line numbers for columns and constraints.
	firstLine          int
	columnLineList     []int
	constraintLineList []int
}

func TestMySQLCreateTableSetLine(t *testing.T) {
	tests := []setLineTestData{
		{
			statement:          "CREATE TABLE t as select * from t1",
			firstLine:          1,
			columnLineList:     []int{},
			constraintLineList: []int{},
		},
		{
			statement:          "CREATE TABLE t like t1",
			firstLine:          1,
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
			firstLine:          1,
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
			firstLine:          1,
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
			firstLine:          1,
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
			firstLine:          1,
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
		node.SetOriginTextPosition(test.firstLine)
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
	tests := []struct {
		statement    string
		expectedLine int32
		expectedCol  int32
		expectedMsg  string
	}{
		{
			statement:    "SELECT 𝄞𝄞hello TO world;",
			expectedLine: 1,
			expectedCol:  23,
			expectedMsg:  `line 1 column 23 near "TO world;" `,
		},
		{
			statement:    "SELECT 1;\nSELEC 5;\nSELECT 6;",
			expectedLine: 2,
			expectedCol:  6,
			expectedMsg:  "line 2 column 6 near \"SELEC 5;\nSELECT 6;\" ",
		},
		{
			statement:    "SELECT 1;\n   SELEC 5;\nSELECT 6;",
			expectedLine: 2,
			expectedCol:  9,
			expectedMsg:  "line 2 column 9 near \"SELEC 5;\nSELECT 6;\" ",
		},
	}

	for _, test := range tests {
		_, err := ParseTiDB(test.statement, "", "")
		require.Error(t, err)
		syntaxErr, ok := err.(*base.SyntaxError)
		require.True(t, ok)
		require.Equal(t, test.expectedLine, syntaxErr.Position.GetLine())
		require.Equal(t, test.expectedCol, syntaxErr.Position.GetColumn())
		require.Equal(t, test.expectedMsg, syntaxErr.Message)
	}
}

func TestParseTiDBForSyntaxCheckError(t *testing.T) {
	tests := []struct {
		name         string
		statement    string
		expectedLine int32 // Expected to be the END line of the statement with error
	}{
		{
			name:         "single statement with syntax error",
			statement:    "SELECT * FRAM table;",
			expectedLine: 1,
		},
		{
			name: "multi-line single statement with syntax error",
			statement: `SELECT
	1,
	2
FRAM table;`,
			expectedLine: 4, // Last line of the statement
		},
		{
			name: "multiple statements - error in first",
			statement: `SELECT * FRAM t1;
SELECT 1;
SELECT 2;`,
			expectedLine: 1, // End of first statement
		},
		{
			name: "multiple statements - error in second",
			statement: `SELECT 1;
SELECT * FRAM t2;
SELECT 2;`,
			expectedLine: 2, // End of second statement
		},
		{
			name: "multiple statements - error in third",
			statement: `SELECT 1;
SELECT 2;
SELECT * FRAM t3;`,
			expectedLine: 3, // End of third statement
		},
		{
			name: "multi-line statements - error in second statement",
			statement: `SELECT
	1,
	2
FROM table1;
SELECT
	*
FRAM table2;`,
			expectedLine: 7, // End of second statement
		},
		{
			name: "complex multi-line - error on specific line",
			statement: `-- Comment line 1
SELECT 1;
-- Comment line 3
SELECT
	a,
	b,
	c,
	d
FRAM table;
SELECT 2;`,
			expectedLine: 9, // End of the SELECT statement with error
		},
		{
			name: "error with WHERE typo",
			statement: `SELECT 1;
SELECT 2;
SELECT * FROM t1 WHER id = 1;`,
			expectedLine: 3, // End of third statement
		},
		{
			name: "error in second statement with indentation",
			statement: `select 1;
   selec 2;
select 3;`,
			expectedLine: 2, // End of second statement
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := ParseTiDBForSyntaxCheck(test.statement)
			require.Error(t, err, "expected syntax error for statement: %s", test.statement)
			syntaxErr, ok := err.(*base.SyntaxError)
			require.True(t, ok, "expected error to be *base.SyntaxError, got %T", err)
			require.NotNil(t, syntaxErr.Position, "expected position to be set")
			require.Equal(t, test.expectedLine, syntaxErr.Position.GetLine(),
				"incorrect line number for statement:\n%s\nError: %s", test.statement, syntaxErr.Message)
		})
	}
}
