package parser

import (
	"testing"

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
			statement:    "SELECT * FROM t1 WHERE c1 = ",
			errorMessage: "line 1:26 extraneous input '=' expecting {<EOF>, 'ABORT', 'ABS', 'ABSENT', 'ACCESS', 'ACCESSED', 'ACCOUNT', 'ACL', 'ACOS', 'ACROSS', 'ACTION', 'ACTIONS', 'ACTIVATE', 'ACTIVE', 'ACTIVE_COMPONENT', 'ACTIVE_DATA', 'ACTIVE_FUNCTION', 'ACTIVE_TAG', 'ACTIVITY', 'ADAPTIVE_PLAN', 'ADD', 'ADD_COLUMN', 'ADD_GROUP', 'ADD_MONTHS', 'ADJ_DATE', 'ADMIN', 'ADMINISTER', 'ADMINISTRATOR', 'ADVANCED', 'ADVISE', 'ADVISOR', 'AFD_DISKSTRING', 'AFTER', 'AGENT', 'AGGREGATE', 'A', 'ALIAS', 'ALLOCATE', 'ALLOW', 'ALL_ROWS', 'ALTER', 'ALTERNATE', 'ALWAYS', 'ANALYTIC', 'ANALYZE', 'ANCESTOR', 'ANCILLARY', 'AND_EQUAL', 'ANOMALY', 'ANSI_REARCH', 'ANTIJOIN', 'ANYSCHEMA', 'APPEND', 'APPENDCHILDXML', 'APPEND_VALUES', 'APPLICATION', 'APPLY', 'APPROX_COUNT_DISTINCT', 'ARCHIVAL', 'ARCHIVE', 'ARCHIVED', 'ARCHIVELOG', 'ARRAY', 'ASCII', 'ASCIISTR', 'ASIN', 'ASIS', 'ASSEMBLY', 'ASSIGN', 'ASSOCIATE', 'ASYNC', 'ASYNCHRONOUS', 'ATAN2', 'ATAN', 'AT', 'ATTRIBUTE', 'ATTRIBUTES', 'AUDIT', 'AUTHENTICATED', 'AUTHENTICATION', 'AUTHID', 'AUTHORIZATION', 'AUTOALLOCATE', 'A",
		},
	}

	for _, test := range tests {
		_, err := ParsePLSQL(test.statement)
		if test.errorMessage == "" {
			require.NoError(t, err)
		} else {
			require.EqualError(t, err, test.errorMessage)
		}
	}
}
