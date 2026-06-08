package redshift

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestGetStatementTypes(t *testing.T) {
	tests := []struct {
		name      string
		statement string
		want      []storepb.StatementType
	}{
		{
			name:      "create table",
			statement: "CREATE TABLE t1 (id INT);",
			want: []storepb.StatementType{
				storepb.StatementType_CREATE_TABLE,
			},
		},
		{
			name:      "create table as select",
			statement: "CREATE TABLE t1 AS SELECT 1;",
			want: []storepb.StatementType{
				storepb.StatementType_CREATE_TABLE,
			},
		},
		{
			name:      "create view",
			statement: "CREATE VIEW v1 AS SELECT 1;",
			want: []storepb.StatementType{
				storepb.StatementType_CREATE_VIEW,
			},
		},
		{
			name:      "create materialized view",
			statement: "CREATE MATERIALIZED VIEW mv1 AS SELECT 1;",
			want: []storepb.StatementType{
				storepb.StatementType_CREATE_VIEW,
			},
		},
		{
			name:      "create index",
			statement: "CREATE INDEX idx_t1_id ON t1(id);",
			want: []storepb.StatementType{
				storepb.StatementType_CREATE_INDEX,
			},
		},
		{
			name:      "create sequence",
			statement: "CREATE SEQUENCE seq1;",
			want: []storepb.StatementType{
				storepb.StatementType_CREATE_SEQUENCE,
			},
		},
		{
			name:      "create schema",
			statement: "CREATE SCHEMA s1;",
			want: []storepb.StatementType{
				storepb.StatementType_CREATE_SCHEMA,
			},
		},
		{
			name:      "create database",
			statement: "CREATE DATABASE db1;",
			want: []storepb.StatementType{
				storepb.StatementType_CREATE_DATABASE,
			},
		},
		{
			name:      "drop table",
			statement: "DROP TABLE t1;",
			want: []storepb.StatementType{
				storepb.StatementType_DROP_TABLE,
			},
		},
		{
			name:      "drop view",
			statement: "DROP VIEW v1;",
			want: []storepb.StatementType{
				storepb.StatementType_DROP_VIEW,
			},
		},
		{
			name:      "drop materialized view",
			statement: "DROP MATERIALIZED VIEW mv1;",
			want: []storepb.StatementType{
				storepb.StatementType_DROP_TABLE,
			},
		},
		{
			name:      "drop schema",
			statement: "DROP SCHEMA s1;",
			want: []storepb.StatementType{
				storepb.StatementType_DROP_SCHEMA,
			},
		},
		{
			name:      "alter table",
			statement: "ALTER TABLE t1 ADD COLUMN name TEXT;",
			want: []storepb.StatementType{
				storepb.StatementType_ALTER_TABLE,
			},
		},
		{
			name:      "alter datashare",
			statement: "ALTER DATASHARE salesshare ADD SCHEMA public;",
			want:      []storepb.StatementType{},
		},
		{
			name:      "insert update delete",
			statement: "INSERT INTO t1 VALUES (1); UPDATE t1 SET id = 2; DELETE FROM t1;",
			want: []storepb.StatementType{
				storepb.StatementType_INSERT,
				storepb.StatementType_UPDATE,
				storepb.StatementType_DELETE,
			},
		},
		{
			name:      "truncate table",
			statement: "TRUNCATE TABLE t1;",
			want: []storepb.StatementType{
				storepb.StatementType_TRUNCATE,
			},
		},
		{
			name:      "comment",
			statement: "COMMENT ON TABLE t1 IS 'hello';",
			want: []storepb.StatementType{
				storepb.StatementType_COMMENT,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stmts, err := base.ParseStatements(storepb.Engine_REDSHIFT, tt.statement)
			require.NoError(t, err)

			got, err := GetStatementTypes(base.ExtractASTs(stmts))
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestGetStatementTypesWithPosition(t *testing.T) {
	statement := `CREATE TABLE t1 (id INT);
DROP TABLE t2;
INSERT INTO t1 VALUES (1);`
	stmts, err := base.ParseStatements(storepb.Engine_REDSHIFT, statement)
	require.NoError(t, err)

	got, err := GetStatementTypesWithPosition(base.ExtractASTs(stmts))
	require.NoError(t, err)
	require.Len(t, got, 3)
	require.Equal(t, storepb.StatementType_CREATE_TABLE, got[0].Type)
	require.Equal(t, 1, got[0].Line)
	require.Equal(t, "CREATE TABLE t1 (id INT);", got[0].Text)
	require.Equal(t, storepb.StatementType_DROP_TABLE, got[1].Type)
	require.Equal(t, 2, got[1].Line)
	require.Equal(t, storepb.StatementType_INSERT, got[2].Type)
	require.Equal(t, 3, got[2].Line)
}
