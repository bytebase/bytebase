package parser

import (
	"fmt"
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
			errorMessage: "line 1:26 extraneous input '=' expecting {<EOF>, 'ABORT', 'ABS', 'ABSENT', 'ACCESS', 'ACCESSED', 'ACCOUNT', 'ACL', 'ACOS', 'ACROSS', 'ACTION', 'ACTIONS', 'ACTIVATE', 'ACTIVE', 'ACTIVE_COMPONENT', 'ACTIVE_DATA', 'ACTIVE_FUNCTION', 'ACTIVE_TAG', 'ACTIVITY', 'ADAPTIVE_PLAN', 'ADD', 'ADD_COLUMN', 'ADD_GROUP', 'ADD_MONTHS', 'ADJ_DATE', 'ADMIN', 'ADMINISTER', 'ADMINISTRATOR', 'ADVANCED', 'ADVISE', 'ADVISOR', 'AFD_DISKSTRING', 'AFTER', 'AGENT', 'AGGREGATE', 'A', 'ALIAS', 'ALLOCATE', 'ALLOW', 'ALL_ROWS', 'ALTER', 'ALTERNATE', 'ALWAYS', 'ANALYTIC', 'ANALYZE', 'ANCESTOR', 'ANCILLARY', 'AND_EQUAL', 'ANOMALY', 'ANSI_REARCH', 'ANTIJOIN', 'ANYSCHEMA', 'APPEND', 'APPENDCHILDXML', 'APPEND_VALUES', 'APPLICATION', 'APPLY', 'APPROX_COUNT_DISTINCT', 'ARCHIVAL', 'ARCHIVE', 'ARCHIVED', 'ARCHIVELOG', 'ARRAY', 'ASCII', 'ASCIISTR', 'ASIN', 'ASIS', 'ASSEMBLY', 'ASSIGN', 'ASSOCIATE', 'ASYNC', 'ASYNCHRONOUS', 'ATAN2', 'ATAN', 'AT', 'ATTRIBUTE', 'ATTRIBUTES', 'AUDIT', 'AUTHENTICATED', 'AUTHENTICATION', 'AUTHID', 'AUTHORIZATION', 'AUTOALLOCATE', 'A",
		},
	}

	for _, test := range tests {
		tree, _, err := ParsePLSQL(test.statement)
		if sql, ok := tree.(*plsqlparser.Sql_scriptContext); ok {
			fmt.Println(sql.GetText())
		}
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
