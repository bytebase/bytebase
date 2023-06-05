// Package oracle is the advisor for oracle database.
package oracle

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
)

func TestOracleSyntax(t *testing.T) {
	tests := []advisor.TestCase{
		{
			Statement: "CREATE TABLE book(id int);",
			Want: []advisor.Advice{
				{
					Status:  advisor.Success,
					Code:    advisor.Ok,
					Title:   "Syntax OK",
					Content: "OK",
				},
			},
		},
		{
			Statement: "DROP TABLE book CASCADE CONSTRAINTS;",
			Want: []advisor.Advice{
				{
					Status:  advisor.Success,
					Code:    advisor.Ok,
					Title:   "Syntax OK",
					Content: "OK",
				},
			},
		},
		{
			Statement: "CREATE TABLE book(id int) ENGINE=INNODB;",
			Want: []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.StatementSyntaxError,
					Title:   "Syntax error",
					Content: "line 1:26 no viable alternative at input 'ENGINE' \nline 1:32 extraneous input '=' expecting {<EOF>, 'ABORT', 'ABS', 'ABSENT', 'ACCESS', 'ACCESSED', 'ACCOUNT', 'ACL', 'ACOS', 'ACROSS', 'ACTION', 'ACTIONS', 'ACTIVATE', 'ACTIVE', 'ACTIVE_COMPONENT', 'ACTIVE_DATA', 'ACTIVE_FUNCTION', 'ACTIVE_TAG', 'ACTIVITY', 'ADAPTIVE_PLAN', 'ADD', 'ADD_COLUMN', 'ADD_GROUP', 'ADD_MONTHS', 'ADJ_DATE', 'ADMIN', 'ADMINISTER', 'ADMINISTRATOR', 'ADVANCED', 'ADVISE', 'ADVISOR', 'AFD_DISKSTRING', 'AFTER', 'AGENT', 'AGGREGATE', 'A', 'ALIAS', 'ALLOCATE', 'ALLOW', 'ALL_ROWS', 'ALTER', 'ALTERNATE', 'ALWAYS', 'ANALYTIC', 'ANALYZE', 'ANCESTOR', 'ANCILLARY', 'AND_EQUAL', 'ANOMALY', 'ANSI_REARCH', 'ANTIJOIN', 'ANYSCHEMA', 'APPEND', 'APPENDCHILDXML', 'APPEND_VALUES', 'APPLICATION', 'APPLY', 'APPROX_COUNT_DISTINCT', 'ARCHIVAL', 'ARCHIVE', 'ARCHIVED', 'ARCHIVELOG', 'ARRAY', 'ASCII', 'ASCIISTR', 'ASIN', 'ASIS', 'ASSEMBLY', 'ASSIGN', 'ASSOCIATE', 'ASYNC', 'ASYNCHRONOUS', 'ATAN2', 'ATAN', 'AT', 'ATTRIBUTE', 'ATTRIBUTES', 'AUDIT', 'AUTHENTICATED', 'AUTHENTICATION', 'AUTHID', 'AUTHORIZATION', 'AUTOALLOCATE', 'A \nline 1:40 extraneous input ';' expecting {<EOF>, 'ABORT', 'ABS', 'ABSENT', 'ACCESS', 'ACCESSED', 'ACCOUNT', 'ACL', 'ACOS', 'ACROSS', 'ACTION', 'ACTIONS', 'ACTIVATE', 'ACTIVE', 'ACTIVE_COMPONENT', 'ACTIVE_DATA', 'ACTIVE_FUNCTION', 'ACTIVE_TAG', 'ACTIVITY', 'ADAPTIVE_PLAN', 'ADD', 'ADD_COLUMN', 'ADD_GROUP', 'ADD_MONTHS', 'ADJ_DATE', 'ADMIN', 'ADMINISTER', 'ADMINISTRATOR', 'ADVANCED', 'ADVISE', 'ADVISOR', 'AFD_DISKSTRING', 'AFTER', 'AGENT', 'AGGREGATE', 'A', 'ALIAS', 'ALLOCATE', 'ALLOW', 'ALL_ROWS', 'ALTER', 'ALTERNATE', 'ALWAYS', 'ANALYTIC', 'ANALYZE', 'ANCESTOR', 'ANCILLARY', 'AND_EQUAL', 'ANOMALY', 'ANSI_REARCH', 'ANTIJOIN', 'ANYSCHEMA', 'APPEND', 'APPENDCHILDXML', 'APPEND_VALUES', 'APPLICATION', 'APPLY', 'APPROX_COUNT_DISTINCT', 'ARCHIVAL', 'ARCHIVE', 'ARCHIVED', 'ARCHIVELOG', 'ARRAY', 'ASCII', 'ASCIISTR', 'ASIN', 'ASIS', 'ASSEMBLY', 'ASSIGN', 'ASSOCIATE', 'ASYNC', 'ASYNCHRONOUS', 'ATAN2', 'ATAN', 'AT', 'ATTRIBUTE', 'ATTRIBUTES', 'AUDIT', 'AUTHENTICATED', 'AUTHENTICATION', 'AUTHID', 'AUTHORIZATION', 'AUTOALLOCATE', 'A",
					Line:    1,
				},
			},
		},
	}

	adv := &SyntaxAdvisor{}

	for _, tc := range tests {
		adviceList, err := adv.Check(advisor.Context{}, tc.Statement)
		require.NoError(t, err)
		assert.Equal(t, tc.Want, adviceList)
	}
}
