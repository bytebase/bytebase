package starrocks

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestGetQueryType(t *testing.T) {
	tests := []struct {
		statement string
		want      base.QueryType
	}{
		{
			statement: "SELECT * FROM users",
			want:      base.Select,
		},
		{
			statement: "INSERT INTO users (id, name) VALUES (1, 'test')",
			want:      base.DML,
		},
		{
			statement: "UPDATE users SET name = 'test' WHERE id = 1",
			want:      base.DML,
		},
		{
			statement: "DELETE FROM users WHERE id = 1",
			want:      base.DML,
		},
		{
			statement: "CREATE TABLE test (id INT)",
			want:      base.DDL,
		},
		{
			statement: "SHOW DATABASES",
			want:      base.SelectInfoSchema,
		},
		{
			statement: "SHOW TABLES",
			want:      base.SelectInfoSchema,
		},
		{
			statement: "SHOW DATA",
			want:      base.SelectInfoSchema,
		},
		{
			statement: "SHOW DATA FROM db1",
			want:      base.SelectInfoSchema,
		},
		{
			statement: "SHOW TABLETS FROM table1",
			want:      base.SelectInfoSchema,
		},
		{
			statement: "SHOW VARIABLES",
			want:      base.SelectInfoSchema,
		},
		{
			statement: "SHOW CREATE TABLE users",
			want:      base.SelectInfoSchema,
		},
		{
			// EXPLAIN over a query is read-only.
			statement: "EXPLAIN SELECT * FROM users",
			want:      base.Select,
		},
		{
			// EXPLAIN defers to the inner statement's type — EXPLAIN over
			// DML is classified as DML for ACL purposes, not Select.
			statement: "EXPLAIN INSERT INTO users (id) VALUES (1)",
			want:      base.DML,
		},
		{
			// EXPLAIN over DDL is DDL, not a read-only downgrade.
			statement: "EXPLAIN DROP TABLE users",
			want:      base.DDL,
		},
		{
			// USE is intentionally Unknown so ACL rejects it as a hard
			// deny rather than authorising it under bb.sql.ddl.
			statement: "USE db1",
			want:      base.QueryTypeUnknown,
		},
	}

	for _, tc := range tests {
		t.Run(tc.statement, func(t *testing.T) {
			a := require.New(t)
			extractor := newQuerySpanExtractor("testdb", base.GetQuerySpanContext{}, false)
			result, err := extractor.getQuerySpan(context.Background(), tc.statement)
			a.NoError(err)
			a.Equal(tc.want, result.Type, "statement: %s", tc.statement)
		})
	}
}
