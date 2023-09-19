package parser

import (
	"testing"

	plsqlparser "github.com/bytebase/plsql-parser"
	"github.com/stretchr/testify/require"
)

func TestPLSQLParser(t *testing.T) {
	tests := []struct {
		statement    string
		errorMessage string
	}{
		{
			statement: "SELECT * FROM t1 WHERE c1 = 1; SELECT * FROM t2;",
		},
		{
			statement: "CREATE TABLE t1 (c1 NUMBER(10,2), c2 VARCHAR2(10));",
		},
		{
			statement: "SELECT * FROM t1;",
		},
		{
			statement: "SELECT * FROM t1 WHERE c1 = 1",
		},
		{
			statement:    "SELECT * FROM t1 WHERE c1 = ",
			errorMessage: "Syntax error at line 1:26 \nrelated text: SELECT * FROM t1 WHERE c1 =",
		},
	}

	for _, test := range tests {
		tree, _, err := ParsePLSQL(test.statement)
		_, _ = tree.(*plsqlparser.Sql_scriptContext)
		if test.errorMessage == "" {
			require.NoError(t, err)
		} else {
			require.EqualError(t, err, test.errorMessage)
		}
	}
}

func TestExtractOracleResourceList(t *testing.T) {
	tests := []struct {
		statement string
		expected  []SchemaResource
	}{
		{
			statement: "SELECT * FROM t1 WHERE c1 = 1; SELECT * FROM t2;",
			expected: []SchemaResource{
				{
					Database: "DB",
					Schema:   "ROOT",
					Table:    "T1",
				},
				{
					Database: "DB",
					Schema:   "ROOT",
					Table:    "T2",
				},
			},
		},
		{
			statement: "SELECT * FROM schema1.t1 JOIN schema2.t2 ON t1.c1 = t2.c1;",
			expected: []SchemaResource{
				{
					Database: "DB",
					Schema:   "SCHEMA1",
					Table:    "T1",
				},
				{
					Database: "DB",
					Schema:   "SCHEMA2",
					Table:    "T2",
				},
			},
		},
		{
			statement: "SELECT a > (select max(a) from t1) FROM t2;",
			expected: []SchemaResource{
				{
					Database: "DB",
					Schema:   "ROOT",
					Table:    "T1",
				},
				{
					Database: "DB",
					Schema:   "ROOT",
					Table:    "T2",
				},
			},
		},
	}

	for _, test := range tests {
		resources, err := extractOracleResourceList("DB", "ROOT", test.statement)
		require.NoError(t, err)
		require.Equal(t, test.expected, resources, test.statement)
	}
}
