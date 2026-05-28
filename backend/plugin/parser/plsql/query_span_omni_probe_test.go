package plsql

import (
	"io"
	"os"
	"reflect"
	"testing"

	"github.com/bytebase/omni/oracle/ast"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestOracleOmniParsesQuerySpanFixtureCorpus(t *testing.T) {
	type testCase struct {
		Description string `yaml:"description,omitempty"`
		Statement   string `yaml:"statement,omitempty"`
	}

	yamlFile, err := os.Open("test-data/query_span.yaml")
	require.NoError(t, err)
	defer yamlFile.Close()

	byteValue, err := io.ReadAll(yamlFile)
	require.NoError(t, err)

	var testCases []testCase
	require.NoError(t, yaml.Unmarshal(byteValue, &testCases))

	for _, tc := range testCases {
		t.Run(tc.Description, func(t *testing.T) {
			list, err := ParsePLSQLOmni(tc.Statement)
			require.NoError(t, err)
			require.NotNil(t, list)
			require.Len(t, list.Items, 1)
			require.IsType(t, &ast.RawStmt{}, list.Items[0])
		})
	}
}

func TestOracleOmniQuerySpanMigrationProbe(t *testing.T) {
	tests := []struct {
		name          string
		statement     string
		expectedNodes []string
		check         func(*testing.T, *ast.RawStmt)
	}{
		{
			name:          "select columns and table ref",
			statement:     "SELECT A, T.B, PUBLIC.T.C FROM T",
			expectedNodes: []string{"SelectStmt", "ResTarget", "ColumnRef", "TableRef"},
		},
		{
			name:          "join using",
			statement:     "SELECT * FROM T T1 JOIN T T2 USING(A)",
			expectedNodes: []string{"SelectStmt", "JoinClause", "TableRef"},
		},
		{
			name:          "derived table",
			statement:     "SELECT * FROM (SELECT A FROM T) DT",
			expectedNodes: []string{"SelectStmt", "SubqueryRef", "TableRef"},
		},
		{
			name:          "scalar and in subqueries",
			statement:     "SELECT (SELECT MAX(A) FROM T) AS M FROM T WHERE B IN (SELECT C FROM T)",
			expectedNodes: []string{"SelectStmt", "SubqueryExpr", "FuncCallExpr", "InExpr"},
		},
		{
			name:          "cte and set operation",
			statement:     "WITH T1(D, C) AS (SELECT A, B FROM T UNION ALL SELECT C, D FROM T) SELECT * FROM T1",
			expectedNodes: []string{"WithClause", "CTE", "SelectStmt", "TableRef"},
			check: func(t *testing.T, raw *ast.RawStmt) {
				var foundSetOp bool
				ast.Inspect(raw, func(node ast.Node) bool {
					if sel, ok := node.(*ast.SelectStmt); ok && sel.Op != ast.SETOP_NONE {
						foundSetOp = true
					}
					return true
				})
				require.True(t, foundSetOp)
			},
		},
		{
			name:          "database link",
			statement:     "SELECT * FROM SCHEMA1.LT1@REMOTE",
			expectedNodes: []string{"SelectStmt", "TableRef", "ObjectName"},
			check: func(t *testing.T, raw *ast.RawStmt) {
				var foundDBLink bool
				ast.Inspect(raw, func(node ast.Node) bool {
					if name, ok := node.(*ast.ObjectName); ok && name.Schema == "SCHEMA1" && name.Name == "LT1" && name.DBLink == "REMOTE" {
						foundDBLink = true
					}
					return true
				})
				require.True(t, foundDBLink)
			},
		},
		{
			name:          "json table",
			statement:     "SELECT JT.ID FROM T, JSON_TABLE(T.J, '$' COLUMNS (ID NUMBER PATH '$.id')) JT",
			expectedNodes: []string{"SelectStmt", "JsonTableRef", "JsonTableColumn", "TableRef"},
		},
		{
			name:          "xml table",
			statement:     "SELECT XT.ID FROM T, XMLTABLE('/root' PASSING T.X COLUMNS ID NUMBER PATH 'id') XT",
			expectedNodes: []string{"SelectStmt", "XmlTableRef", "XmlTableColumn", "TableRef"},
		},
		{
			name:          "containers",
			statement:     "SELECT A FROM CONTAINERS(T)",
			expectedNodes: []string{"SelectStmt", "ContainersExpr", "ObjectName"},
		},
		{
			name:          "inline external table",
			statement:     "SELECT EXT.A FROM EXTERNAL ((A NUMBER, B VARCHAR2(20)) TYPE ORACLE_LOADER DEFAULT DIRECTORY D ACCESS PARAMETERS (FIELDS TERMINATED BY ',') LOCATION ('x.csv')) EXT",
			expectedNodes: []string{"SelectStmt", "InlineExternalTable", "ColumnDef"},
		},
		{
			name:          "pivot",
			statement:     "SELECT * FROM (SELECT A, B FROM T) PIVOT (COUNT(*) FOR B IN (1 AS ONE))",
			expectedNodes: []string{"SelectStmt", "PivotClause", "SubqueryRef"},
		},
		{
			name:          "unpivot",
			statement:     "SELECT * FROM (SELECT A, B FROM T) UNPIVOT (VAL FOR COL IN (A AS 'A', B AS 'B'))",
			expectedNodes: []string{"SelectStmt", "UnpivotClause", "SubqueryRef"},
		},
		{
			name:          "model",
			statement:     "SELECT * FROM T MODEL DIMENSION BY (A) MEASURES (B) RULES (B[1] = 1)",
			expectedNodes: []string{"SelectStmt", "ModelClause", "ModelRule"},
		},
		{
			name:          "lateral inline view",
			statement:     "SELECT * FROM T, LATERAL (SELECT A FROM DUAL) L",
			expectedNodes: []string{"SelectStmt", "LateralRef", "TableRef"},
		},
		{
			name:          "hierarchical query",
			statement:     "SELECT A FROM T START WITH A = 1 CONNECT BY PRIOR A = B",
			expectedNodes: []string{"SelectStmt", "HierarchicalClause", "UnaryExpr", "BinaryExpr"},
		},
		{
			name:          "analytic function",
			statement:     "SELECT SUM(A) OVER (PARTITION BY B ORDER BY C) FROM T",
			expectedNodes: []string{"SelectStmt", "FuncCallExpr", "WindowSpec", "SortBy"},
		},
		{
			name:          "explain root",
			statement:     "EXPLAIN PLAN FOR SELECT * FROM T",
			expectedNodes: []string{"ExplainPlanStmt", "SelectStmt"},
		},
		{
			name:          "dml roots",
			statement:     "INSERT INTO T(A) SELECT A FROM T2",
			expectedNodes: []string{"InsertStmt", "SelectStmt", "TableRef"},
		},
		{
			name:          "ddl root",
			statement:     "CREATE TABLE T(A NUMBER)",
			expectedNodes: []string{"CreateTableStmt", "ColumnDef"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			list, err := ParsePLSQLOmni(test.statement)
			require.NoError(t, err)
			require.Len(t, list.Items, 1)
			raw, ok := list.Items[0].(*ast.RawStmt)
			require.True(t, ok)

			nodeTypes := collectOracleOmniNodeTypes(raw)
			for _, expectedNode := range test.expectedNodes {
				require.Contains(t, nodeTypes, expectedNode)
			}
			if test.check != nil {
				test.check(t, raw)
			}
		})
	}
}

func collectOracleOmniNodeTypes(node ast.Node) map[string]bool {
	result := make(map[string]bool)
	ast.Inspect(node, func(node ast.Node) bool {
		if node == nil {
			return false
		}
		typ := reflect.TypeOf(node)
		if typ.Kind() == reflect.Pointer {
			typ = typ.Elem()
		}
		result[typ.Name()] = true
		return true
	})
	return result
}
