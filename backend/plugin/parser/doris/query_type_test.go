package doris

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
