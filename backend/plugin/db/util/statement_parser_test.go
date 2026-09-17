//nolint:revive
package util

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRemoveCommentsAndTrim(t *testing.T) {
	tests := []struct {
		input   string
		want    string
		wantErr bool
	}{
		{
			input: ``,
			want:  ``,
		},
		{
			input: `SELECT 1;`,
			want:  `SELECT 1`,
		},
		{
			input: `-- This is a single line comment
SELECT 1;`,
			want: `SELECT 1`,
		},
		{
			input: `# This is a single line comment
SELECT 1;`,
			want: `SELECT 1`,
		},
		{
			input: `/* This is a multi line comment on one line */
SELECT 1;`,
			want: `SELECT 1`,
		},
		{
			input: `/* This
is
a
multiline
comment
*/
SELECT 1;`,
			want: `SELECT 1`,
		},
		{
			input: `/* This
* is
* a
* multiline
* comment
*/
SELECT 1;`,
			want: `SELECT 1`,
		},
		{
			input: `/** This is a javadoc style comment on one line */
SELECT 1;`,
			want: `SELECT 1`,
		},
		{
			input: `/** This
is
a
javadoc
style
comment
on
multiple
lines
*/
SELECT 1;`,
			want: `SELECT 1`,
		},
		{
			input: `/** This
* is
* a
* javadoc
* style
* comment
* on
* multiple
* lines
*/
SELECT 1;`,
			want: `SELECT 1`,
		},
		{
			input: `-- First comment
SELECT--second comment
1`,
			want: `SELECT
1`,
		},
		{
			input: `# First comment
SELECT#second comment
1`,
			want: `SELECT
1`,
		},
		{
			input: `-- First comment
SELECT--second comment
1--third comment`,
			want: `SELECT
1`,
		},
		{
			input: `# First comment
SELECT#second comment
1#third comment`,
			want: `SELECT
1`,
		},
		{
			input: `/* First comment */
SELECT/* second comment */
1`,
			want: `SELECT
1`,
		},
		{
			input: `/* First comment */
SELECT/* second comment */
1/* third comment */`,
			want: `SELECT
1`,
		},
		{
			input: `SELECT "TEST -- This is not a comment"`,
			want:  `SELECT "TEST -- This is not a comment"`,
		},
		{
			input: `-- This is a comment
SELECT "TEST -- This is not a comment"`,
			want: `SELECT "TEST -- This is not a comment"`,
		},
		{
			input: `-- This is a comment
SELECT "TEST -- This is not a comment" -- This is a comment`,
			want: `SELECT "TEST -- This is not a comment"`,
		},
		{
			input: `SELECT "TEST # This is not a comment"`,
			want:  `SELECT "TEST # This is not a comment"`,
		},
		{
			input: `# This is a comment
SELECT "TEST # This is not a comment"`,
			want: `SELECT "TEST # This is not a comment"`,
		},
		{
			input: `# This is a comment
SELECT "TEST # This is not a comment" # This is a comment`,
			want: `SELECT "TEST # This is not a comment"`,
		},
		{
			input: `SELECT "TEST /* This is not a comment */"`,
			want:  `SELECT "TEST /* This is not a comment */"`,
		},
		{
			input: `/* This is a comment */
SELECT "TEST /* This is not a comment */"`,
			want: `SELECT "TEST /* This is not a comment */"`,
		},
		{
			input: `/* This is a comment */
SELECT "TEST /* This is not a comment */" /* This is a comment */`,
			want: `SELECT "TEST /* This is not a comment */"`,
		},
		{
			input: `SELECT 'TEST -- This is not a comment'`,
			want:  `SELECT 'TEST -- This is not a comment'`,
		},
		{
			input: `-- This is a comment
SELECT 'TEST -- This is not a comment'`,
			want: `SELECT 'TEST -- This is not a comment'`,
		},
		{
			input: `-- This is a comment
SELECT 'TEST -- This is not a comment' -- This is a comment`,
			want: `SELECT 'TEST -- This is not a comment'`,
		},
		{
			input: `SELECT 'TEST # This is not a comment'`,
			want:  `SELECT 'TEST # This is not a comment'`,
		},
		{
			input: `# This is a comment
SELECT 'TEST # This is not a comment'`,
			want: `SELECT 'TEST # This is not a comment'`,
		},
		{
			input: `# This is a comment
SELECT 'TEST # This is not a comment' # This is a comment`,
			want: `SELECT 'TEST # This is not a comment'`,
		},
		{
			input: `SELECT 'TEST /* This is not a comment */'`,
			want:  `SELECT 'TEST /* This is not a comment */'`,
		},
		{
			input: `/* This is a comment */
SELECT 'TEST /* This is not a comment */'`,
			want: `SELECT 'TEST /* This is not a comment */'`,
		},
		{
			input: `/* This is a comment */
SELECT 'TEST /* This is not a comment */' /* This is a comment */`,
			want: `SELECT 'TEST /* This is not a comment */'`,
		},
		{
			input: `SELECT '''TEST
-- This is not a comment
'''`,
			want: `SELECT '''TEST
-- This is not a comment
'''`,
		},
		{
			input: ` -- This is a comment
SELECT '''TEST
-- This is not a comment
''' -- This is a comment`,
			want: `SELECT '''TEST
-- This is not a comment
'''`,
		},
		{
			input: `SELECT '''TEST
# This is not a comment
'''`,
			want: `SELECT '''TEST
# This is not a comment
'''`,
		},
		{
			input: ` # This is a comment
SELECT '''TEST
# This is not a comment
''' # This is a comment`,
			want: `SELECT '''TEST
# This is not a comment
'''`,
		},
		{
			input: `SELECT '''TEST
/* This is not a comment */
'''`,
			want: `SELECT '''TEST
/* This is not a comment */
'''`,
		},
		{
			input: ` /* This is a comment */
SELECT '''TEST
/* This is not a comment */
''' /* This is a comment */`,
			want: `SELECT '''TEST
/* This is not a comment */
'''`,
		},
		{
			input: `SELECT """TEST
-- This is not a comment
"""`,
			want: `SELECT """TEST
-- This is not a comment
"""`,
		},
		{
			input: ` -- This is a comment
SELECT """TEST
-- This is not a comment
""" -- This is a comment`,
			want: `SELECT """TEST
-- This is not a comment
"""`,
		},
		{
			input: `SELECT """TEST
# This is not a comment
"""`,
			want: `SELECT """TEST
# This is not a comment
"""`,
		},
		{
			input: ` # This is a comment
SELECT """TEST
# This is not a comment
""" # This is a comment`,
			want: `SELECT """TEST
# This is not a comment
"""`,
		},
		{
			input: `SELECT """TEST
/* This is not a comment */
"""`,
			want: `SELECT """TEST
/* This is not a comment */
"""`,
		},
		{
			input: ` /* This is a comment */
SELECT """TEST
/* This is not a comment */
""" /* This is a comment */`,
			want: `SELECT """TEST
/* This is not a comment */
"""`,
		},
		{
			input: `/* This is a comment /* this is still a comment */
SELECT 1`,
			want: `SELECT 1`,
		},
		{
			input: `/** This is a javadoc style comment /* this is still a comment */
SELECT 1`,
			want: `SELECT 1`,
		},
		{
			input: `/** This is a javadoc style comment /** this is still a comment */
SELECT 1`,
			want: `SELECT 1`,
		},
		{
			input: `/** This is a javadoc style comment /** this is still a comment **/
SELECT 1`,
			want: `SELECT 1`,
		},
	}
	a := require.New(t)
	for _, tc := range tests {
		got, err := removeCommentsAndTrim(tc.input)
		if tc.wantErr {
			a.Error(err)
		} else {
			a.NoError(err)
		}
		a.Equal(tc.want, got)
	}
}

func TestSanitizeQuery(t *testing.T) {
	tests := []struct {
		input   string
		want    []string
		wantErr bool
	}{
		{
			input: `
			--- 123456
			SELECT 1;
			--- 6312
			SELECT 2;
			---123
			---123
			/* 213123 */
			SELECT 3;
			`,
			want: []string{
				"SELECT 1", "SELECT 2", "SELECT 3",
			},
		},
		{
			input: `
-- This is the bytebase schema to track migration info for Spanner
-- Create a database called bytebase in the driver.
-- CREATE DATABASE bytebase;

-- Create migration_history table
CREATE TABLE migration_history (
    -- id is UUIDv4.
    id STRING(MAX) NOT NULL,
    created_by STRING(MAX) NOT NULL,
    created_ts INT64 NOT NULL,
    updated_by STRING(MAX) NOT NULL,
    updated_ts INT64 NOT NULL,
    -- Record the client version creating this migration history. For Bytebase, we use its binary release version. Different Bytebase release might
    -- record different history info and this field helps to handle such situation properly. Moreover, it helps debugging.
    release_version STRING(MAX) NOT NULL,
    -- Allows granular tracking of migration history (e.g If an application manages schemas for a multi-tenant service and each tenant has its own schema, that application can use namespace to record the tenant name to track the per-tenant schema migration)
    -- Since bytebase also manages different application databases from an instance, it leverages this field to track each database migration history.
    namespace STRING(MAX) NOT NULL,
    -- Used to detect out of order migration together with 'namespace' and 'version' column.
    sequence INT64 NOT NULL,
    CONSTRAINT sequence_is_non_negative CHECK (sequence >= 0),
    -- Current allowed values are BASELINE, MIGRATE, MIGRATE_SDL, BRANCH, DATA.
    type STRING(MAX) NOT NULL,
    -- Current allowed values are PENDING, DONE, FAILED.
    -- We create a "PENDING" record before applying the DDL and update that record to "DONE" after applying the DDL.
    status STRING(MAX) NOT NULL,
    -- Record the migration version.
    version STRING(MAX) NOT NULL,
    description STRING(MAX) NOT NULL,
    -- Record the migration statement
    statement STRING(MAX) NOT NULL,
    -- Record the schema after migration
    schema STRING(MAX) NOT NULL,
    -- Record the schema before migration. Though we could also fetch it from the previous migration history, it would complicate fetching logic.
    -- Besides, by storing the schema_prev, we can perform consistency check to see if the migration history has any gaps.
    schema_prev STRING(MAX) NOT NULL,
    execution_duration_ns INT64 NOT NULL,
    issue_id STRING(MAX) NOT NULL,
    payload STRING(MAX) NOT NULL
) PRIMARY KEY(id);

CREATE UNIQUE INDEX bytebase_idx_unique_migration_history_namespace_sequence ON migration_history (namespace, sequence);

CREATE UNIQUE INDEX bytebase_idx_unique_migration_history_namespace_version ON migration_history (namespace, version);

CREATE INDEX bytebase_idx_migration_history_namespace_source_type ON migration_history(namespace, source, type);

CREATE INDEX bytebase_idx_migration_history_namespace_created ON migration_history(namespace, created_ts);
      `,
			want: []string{
				`CREATE TABLE migration_history (
    
    id STRING(MAX) NOT NULL,
    created_by STRING(MAX) NOT NULL,
    created_ts INT64 NOT NULL,
    updated_by STRING(MAX) NOT NULL,
    updated_ts INT64 NOT NULL,
    
    
    release_version STRING(MAX) NOT NULL,
    
    
    namespace STRING(MAX) NOT NULL,
    
    sequence INT64 NOT NULL,
    CONSTRAINT sequence_is_non_negative CHECK (sequence >= 0),
    
    type STRING(MAX) NOT NULL,
    
    
    status STRING(MAX) NOT NULL,
    
    version STRING(MAX) NOT NULL,
    description STRING(MAX) NOT NULL,
    
    statement STRING(MAX) NOT NULL,
    
    schema STRING(MAX) NOT NULL,
    
    
    schema_prev STRING(MAX) NOT NULL,
    execution_duration_ns INT64 NOT NULL,
    issue_id STRING(MAX) NOT NULL,
    payload STRING(MAX) NOT NULL
) PRIMARY KEY(id)`,
				"CREATE UNIQUE INDEX bytebase_idx_unique_migration_history_namespace_sequence ON migration_history (namespace, sequence)",
				"CREATE UNIQUE INDEX bytebase_idx_unique_migration_history_namespace_version ON migration_history (namespace, version)",
				"CREATE INDEX bytebase_idx_migration_history_namespace_source_type ON migration_history(namespace, source, type)",
				"CREATE INDEX bytebase_idx_migration_history_namespace_created ON migration_history(namespace, created_ts)",
			},
		},
		{
			input: `
				SELECT 1;
				SELECT '2;3;4;';
				SELECT 2;
			`,
			want: []string{"SELECT 1", "SELECT '2;3;4;'", "SELECT 2"},
		},
		{
			input: `
				SELECT 1;;;;
				SELECT '2;3;4;';;
				SELECT 2;;
			`,
			want: []string{"SELECT 1", "SELECT '2;3;4;'", "SELECT 2"},
		},
	}
	a := require.New(t)
	for _, tc := range tests {
		got, err := SanitizeSQL(tc.input)
		if tc.wantErr {
			a.Error(err)
		} else {
			a.NoError(err)
		}
		a.Equal(tc.want, got)
	}
}

func TestIsSelect(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		expected bool
	}{
		{
			name:     "simple SELECT",
			query:    "SELECT * FROM users",
			expected: true,
		},
		{
			name:     "SELECT with WHERE clause",
			query:    "SELECT id, name FROM users WHERE age > 18",
			expected: true,
		},
		{
			name:     "SELECT lowercase",
			query:    "select * from users",
			expected: true,
		},
		{
			name:     "CTE with SELECT (starts with WITH)",
			query:    "WITH cte AS (SELECT * FROM users) SELECT * FROM cte",
			expected: true,
		},
		{
			name:     "INSERT statement",
			query:    "INSERT INTO test_table (id, name) VALUES (1, 'test')",
			expected: false,
		},
		{
			name:     "UPDATE statement",
			query:    "UPDATE test_table SET name = 'updated' WHERE id = 1",
			expected: false,
		},
		{
			name:     "DELETE statement",
			query:    "DELETE FROM test_table WHERE id = 1",
			expected: false,
		},
		{
			name:     "CREATE TABLE",
			query:    "CREATE TABLE test (id INT64, name STRING(100))",
			expected: false,
		},
		{
			name:     "DROP TABLE",
			query:    "DROP TABLE test",
			expected: false,
		},
		{
			name:     "ALTER TABLE",
			query:    "ALTER TABLE test ADD COLUMN age INT64",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := require.New(t)
			result := IsSelect(tt.query)
			a.Equal(tt.expected, result)
		})
	}
}

func TestIsDDL(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		expected bool
	}{
		{
			name:     "CREATE TABLE",
			query:    "CREATE TABLE test (id INT64, name STRING(100))",
			expected: true,
		},
		{
			name:     "DROP TABLE",
			query:    "DROP TABLE test",
			expected: true,
		},
		{
			name:     "ALTER TABLE",
			query:    "ALTER TABLE test ADD COLUMN age INT64",
			expected: true,
		},
		{
			name:     "CREATE INDEX",
			query:    "CREATE INDEX idx_name ON test(name)",
			expected: true,
		},
		{
			name:     "DROP INDEX",
			query:    "DROP INDEX idx_name",
			expected: true,
		},
		{
			name:     "ALTER lowercase",
			query:    "alter table test drop column age",
			expected: true,
		},
		{
			name:     "SELECT statement",
			query:    "SELECT * FROM users",
			expected: false,
		},
		{
			name:     "INSERT statement",
			query:    "INSERT INTO test_table (id, name) VALUES (1, 'test')",
			expected: false,
		},
		{
			name:     "UPDATE statement",
			query:    "UPDATE test_table SET name = 'updated' WHERE id = 1",
			expected: false,
		},
		{
			name:     "DELETE statement",
			query:    "DELETE FROM test_table WHERE id = 1",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := require.New(t)
			result := IsDDL(tt.query)
			a.Equal(tt.expected, result)
		})
	}
}
