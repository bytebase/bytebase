package plsql

import (
	"testing"

	oracleast "github.com/bytebase/omni/oracle/ast"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestOracleOmniQueryTypeDDLClassification(t *testing.T) {
	tests := []struct {
		name      string
		statement string
		want      base.QueryType
	}{
		{
			name:      "create table",
			statement: "CREATE TABLE T(A NUMBER)",
			want:      base.DDL,
		},
		{
			name:      "create database admin ddl",
			statement: "CREATE DATABASE",
			want:      base.DDL,
		},
		{
			name:      "alter database admin ddl",
			statement: "ALTER DATABASE OPEN",
			want:      base.DDL,
		},
		{
			name:      "drop database admin ddl",
			statement: "DROP DATABASE",
			want:      base.DDL,
		},
		{
			name:      "lock table is dml",
			statement: "LOCK TABLE T IN EXCLUSIVE MODE",
			want:      base.DML,
		},
		{
			name:      "transaction remains unknown",
			statement: "COMMIT",
			want:      base.QueryTypeUnknown,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			list, err := ParsePLSQLOmni(test.statement)
			require.NoError(t, err)
			require.Len(t, list.Items, 1)
			raw, ok := list.Items[0].(*oracleast.RawStmt)
			require.True(t, ok)
			require.NotNil(t, raw.Stmt)

			require.Equal(t, test.want, omniQueryType(raw.Stmt, false))
		})
	}
}
